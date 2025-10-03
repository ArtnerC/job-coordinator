package distributor

import (
	"context"
	"fmt"

	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/models"
)

// Distributor is the interface that all distributors must implement
type Distributor interface {
	// Distribute sends a work unit to the configured destination
	Distribute(workUnit models.WorkUnit) error
	
	// Close closes the distributor and releases any resources
	Close() error
}

// NewDistributor creates a distributor based on the configuration
func NewDistributor(ctx context.Context, cfg config.Config) (Distributor, error) {
	switch cfg.DistributorType {
	case "stdout":
		return NewStdoutDistributor(), nil
		
	case "file":
		outputPath := cfg.DistributorConfig["output_path"]
		if outputPath == "" {
			return nil, fmt.Errorf("file distributor requires 'output_path' in distributor_config")
		}
		return NewFileDistributor(outputPath)
		
	case "pubsub":
		projectID := cfg.DistributorConfig["project_id"]
		if projectID == "" {
			return nil, fmt.Errorf("pubsub distributor requires 'project_id' in distributor_config")
		}
		
		topicName := cfg.DistributorConfig["topic_name"]
		if topicName == "" {
			return nil, fmt.Errorf("pubsub distributor requires 'topic_name' in distributor_config")
		}
		
		pubsubConfig := PubSubConfig{
			ProjectID: projectID,
			TopicName: topicName,
			BatchSize: 100, // Default batch size
		}
		
		return NewPubSubDistributor(ctx, pubsubConfig)
		
	default:
		return nil, fmt.Errorf("unknown distributor type: %s", cfg.DistributorType)
	}
}

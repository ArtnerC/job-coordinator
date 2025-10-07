package distributor

import (
	"context"
	"fmt"
	"time"

	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/models"
)

// DistributorStatus represents the completion status of a distributor
type DistributorStatus struct {
	Type             string `json:"type"`              // Type of distributor (stdout, file, pubsub)
	IsComplete       bool   `json:"is_complete"`       // True if all work has been distributed and flushed
	PendingCount     int    `json:"pending_count"`     // Number of items pending distribution
	DistributedCount int    `json:"distributed_count"` // Total number of items distributed
}

// Distributor is the interface that all distributors must implement
type Distributor interface {
	// Distribute sends a work unit to the configured destination
	Distribute(workUnit models.WorkUnit) error
	
	// Close closes the distributor and releases any resources
	Close() error
	
	// GetStatus returns the current status of the distributor
	GetStatus() DistributorStatus
	
	// WaitForCompletion blocks until all pending work has been distributed
	// For stdout: ensures all output is flushed
	// For file: ensures file is written and synced
	// For pubsub: ensures queue backlog is empty
	WaitForCompletion() error
}

// NewDistributor creates a distributor based on the configuration
func NewDistributor(ctx context.Context, cfg config.Config) (Distributor, error) {
	switch cfg.DistributorType {
	case "stdout":
		return NewStdoutDistributorWithConfig(nil, cfg.DistributorConfig), nil
		
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
		
		// Optional: completion_timeout (e.g., "2h", "30m")
		if timeoutStr := cfg.DistributorConfig["completion_timeout"]; timeoutStr != "" {
			if timeout, err := time.ParseDuration(timeoutStr); err == nil {
				pubsubConfig.CompletionTimeout = timeout
			}
		}
		
		// Optional: poll_interval (e.g., "5s", "10s")
		if intervalStr := cfg.DistributorConfig["poll_interval"]; intervalStr != "" {
			if interval, err := time.ParseDuration(intervalStr); err == nil {
				pubsubConfig.PollInterval = interval
			}
		}
		
		// Optional: subscription_ttl (e.g., "24h", "48h")
		if ttlStr := cfg.DistributorConfig["subscription_ttl"]; ttlStr != "" {
			if ttl, err := time.ParseDuration(ttlStr); err == nil {
				pubsubConfig.SubscriptionTTL = ttl
			}
		}
		
		return NewPubSubDistributor(ctx, pubsubConfig)
		
	default:
		return nil, fmt.Errorf("unknown distributor type: %s", cfg.DistributorType)
	}
}

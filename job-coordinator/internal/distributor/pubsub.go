package distributor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/dqme/job-coordinator/internal/models"
)

// PubSubDistributor publishes work units to Google Cloud Pub/Sub
type PubSubDistributor struct {
	client      *pubsub.Client
	topic       *pubsub.Topic
	projectID   string
	topicName   string
	batchSize   int
	mu          sync.Mutex
	batch       []*pubsub.Message
	publishErrs []error
}

// PubSubConfig holds configuration for Pub/Sub distributor
type PubSubConfig struct {
	ProjectID string
	TopicName string
	BatchSize int // Number of messages to batch before publishing (default: 100)
}

// NewPubSubDistributor creates a new Pub/Sub distributor
func NewPubSubDistributor(ctx context.Context, config PubSubConfig) (*PubSubDistributor, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if config.TopicName == "" {
		return nil, fmt.Errorf("topic name cannot be empty")
	}

	// Default batch size
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}

	client, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	topic := client.Topic(config.TopicName)
	
	// Verify topic exists
	exists, err := topic.Exists(ctx)
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("failed to check if topic exists: %w", err)
	}
	if !exists {
		client.Close()
		return nil, fmt.Errorf("topic %s does not exist in project %s", config.TopicName, config.ProjectID)
	}

	return &PubSubDistributor{
		client:    client,
		topic:     topic,
		projectID: config.ProjectID,
		topicName: config.TopicName,
		batchSize: config.BatchSize,
		batch:     make([]*pubsub.Message, 0, config.BatchSize),
	}, nil
}

// Distribute publishes a work unit to Pub/Sub
// Messages are batched for efficiency and published when batch size is reached
func (d *PubSubDistributor) Distribute(workUnit models.WorkUnit) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client == nil {
		return fmt.Errorf("distributor is closed")
	}

	// Serialize work unit to JSON
	jsonData, err := json.Marshal(workUnit)
	if err != nil {
		return fmt.Errorf("failed to serialize work unit: %w", err)
	}

	// Create Pub/Sub message with attributes
	msg := &pubsub.Message{
		Data: jsonData,
		Attributes: map[string]string{
			"job_id":        workUnit.JobID,
			"work_unit_id":  workUnit.ID,
			"content_type":  "application/json",
			"file_path":     workUnit.FilePath,
		},
	}

	// Add to batch
	d.batch = append(d.batch, msg)

	// Publish batch if it reaches the configured size
	if len(d.batch) >= d.batchSize {
		return d.flushBatch()
	}

	return nil
}

// flushBatch publishes all messages in the current batch
// Must be called with lock held
func (d *PubSubDistributor) flushBatch() error {
	if len(d.batch) == 0 {
		return nil
	}

	ctx := context.Background()
	results := make([]*pubsub.PublishResult, len(d.batch))

	// Publish all messages in batch
	for i, msg := range d.batch {
		results[i] = d.topic.Publish(ctx, msg)
	}

	// Wait for all publishes to complete
	var firstErr error
	failedCount := 0
	for i, result := range results {
		_, err := result.Get(ctx)
		if err != nil {
			failedCount++
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to publish message %d: %w", i, err)
			}
		}
	}

	// Clear batch
	d.batch = d.batch[:0]

	if firstErr != nil {
		return fmt.Errorf("failed to publish %d/%d messages: %w", failedCount, len(results), firstErr)
	}

	return nil
}

// Flush publishes any remaining messages in the batch
func (d *PubSubDistributor) Flush() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	return d.flushBatch()
}

// Close flushes remaining messages and closes the Pub/Sub client
func (d *PubSubDistributor) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.client == nil {
		return nil
	}

	// Flush any remaining messages
	if err := d.flushBatch(); err != nil {
		// Still close client even if flush fails
		d.topic.Stop()
		d.client.Close()
		d.client = nil
		return err
	}

	// Stop topic and close client
	d.topic.Stop()
	err := d.client.Close()
	d.client = nil
	return err
}

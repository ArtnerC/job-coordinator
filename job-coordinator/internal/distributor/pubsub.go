package distributor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"cloud.google.com/go/pubsub/v2"
	pubsubpb "cloud.google.com/go/pubsub/v2/apiv1/pubsubpb"
	"github.com/dqme/job-coordinator/internal/models"
	"google.golang.org/protobuf/types/known/durationpb"
)

// PubSubDistributor publishes work units to Google Cloud Pub/Sub
type PubSubDistributor struct {
	client               *pubsub.Client
	publisher            *pubsub.Publisher
	projectID            string
	topicName            string
	batchSize            int
	completionTimeout    time.Duration // Max time to wait for completion (default: 12 hours)
	pollInterval         time.Duration // How often to check subscription (default: 5 seconds)
	subscriptionTTL      time.Duration // How long monitoring subscription lives (default: 24 hours)
	mu                   sync.Mutex
	batch                []*pubsub.Message
	publishErrs          []error
	distributedCount     int                    // Count of work units successfully distributed
	jobID                string                 // Single job ID for this distributor instance
	monitorSubscriber    *pubsub.Subscriber     // Single monitoring subscriber
	monitorSubName       string                 // Monitoring subscription name
}

// PubSubConfig holds configuration for Pub/Sub distributor
type PubSubConfig struct {
	ProjectID         string
	TopicName         string
	BatchSize         int           // Number of messages to batch before publishing (default: 100)
	CompletionTimeout time.Duration // Max time to wait for completion (default: 12 hours)
	PollInterval      time.Duration // How often to check for completion (default: 5 seconds)
	SubscriptionTTL   time.Duration // Monitoring subscription lifetime (default: 24 hours)
}

// NewPubSubDistributor creates a new Pub/Sub distributor
func NewPubSubDistributor(ctx context.Context, config PubSubConfig) (*PubSubDistributor, error) {
	if config.ProjectID == "" {
		return nil, fmt.Errorf("project ID cannot be empty")
	}
	if config.TopicName == "" {
		return nil, fmt.Errorf("topic name cannot be empty")
	}

	// Set defaults
	if config.BatchSize == 0 {
		config.BatchSize = 100
	}
	if config.CompletionTimeout == 0 {
		config.CompletionTimeout = 12 * time.Hour // Default: 12 hours for long-running jobs
	}
	if config.PollInterval == 0 {
		config.PollInterval = 5 * time.Second // Default: check every 5 seconds
	}
	if config.SubscriptionTTL == 0 {
		config.SubscriptionTTL = 24 * time.Hour // Default: subscription lives 24 hours
	}

	client, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		return nil, fmt.Errorf("failed to create pubsub client: %w", err)
	}

	// Verify topic exists using TopicAdminClient
	topicPath := fmt.Sprintf("projects/%s/topics/%s", config.ProjectID, config.TopicName)
	_, err = client.TopicAdminClient.GetTopic(ctx, &pubsubpb.GetTopicRequest{
		Topic: topicPath,
	})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("topic %s does not exist in project %s: %w", config.TopicName, config.ProjectID, err)
	}

	// Create publisher for the topic
	publisher := client.Publisher(topicPath)

	return &PubSubDistributor{
		client:            client,
		publisher:         publisher,
		projectID:         config.ProjectID,
		topicName:         config.TopicName,
		batchSize:         config.BatchSize,
		completionTimeout: config.CompletionTimeout,
		pollInterval:      config.PollInterval,
		subscriptionTTL:   config.SubscriptionTTL,
		batch:             make([]*pubsub.Message, 0, config.BatchSize),
		jobID:             "", // Will be set on first Distribute call
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

	// Capture job ID from first work unit (all work units should have same job ID)
	if d.jobID == "" {
		d.jobID = workUnit.JobID
	} else if d.jobID != workUnit.JobID {
		return fmt.Errorf("distributor received work unit from different job: expected %s, got %s", d.jobID, workUnit.JobID)
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
		results[i] = d.publisher.Publish(ctx, msg)
	}

	// Wait for all publishes to complete
	var firstErr error
	failedCount := 0
	successCount := 0
	for i, result := range results {
		_, err := result.Get(ctx)
		if err != nil {
			failedCount++
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to publish message %d: %w", i, err)
			}
		} else {
			successCount++
		}
	}

	// Update distributed count with successful publishes
	d.distributedCount += successCount

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
		d.publisher.Stop()
		d.client.Close()
		d.client = nil
		return err
	}

	// Stop publisher and close client
	d.publisher.Stop()
	err := d.client.Close()
	d.client = nil
	return err
}

// GetStatus returns the current status of the pubsub distributor
func (d *PubSubDistributor) GetStatus() DistributorStatus {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Pending count is the number of messages in the batch (not yet published)
	pendingCount := len(d.batch)
	
	// Note: If subscription monitoring is enabled, "IsComplete" only reflects
	// that batches are published, not that messages are fully processed.
	// Use WaitForCompletion() to wait for actual message processing.

	return DistributorStatus{
		Type:             "pubsub",
		IsComplete:       pendingCount == 0,
		PendingCount:     pendingCount,
		DistributedCount: d.distributedCount,
	}
}

// WaitForCompletion flushes any remaining batched messages and waits for queue to drain
func (d *PubSubDistributor) WaitForCompletion() error {
	d.mu.Lock()
	
	// Flush any remaining messages in the batch
	if err := d.flushBatch(); err != nil {
		d.mu.Unlock()
		return err
	}
	
	// Get job ID
	jobID := d.jobID
	d.mu.Unlock()
	
	// If no messages were published, nothing to wait for
	if jobID == "" {
		log.Printf("No messages published, skipping completion wait")
		return nil
	}
	
	// Create monitoring subscription and wait for it to drain
	return d.waitForJobCompletion(jobID)
}

// waitForJobCompletion creates a temporary subscription filtered by job ID and monitors it
func (d *PubSubDistributor) waitForJobCompletion(jobID string) error {
	ctx := context.Background()
	
	log.Printf("Waiting for job %s to complete processing in pubsub...", jobID)
	
	// Create temporary subscription with filter for this job ID
	subName := fmt.Sprintf("monitor-%s-%d", jobID, time.Now().Unix())
	topicPath := fmt.Sprintf("projects/%s/topics/%s", d.projectID, d.topicName)
	subPath := fmt.Sprintf("projects/%s/subscriptions/%s", d.projectID, subName)
	
	subConfig := &pubsubpb.Subscription{
		Name:   subPath,
		Topic:  topicPath,
		Filter: fmt.Sprintf(`attributes.job_id = "%s"`, jobID),
		// Set retention and expiration
		MessageRetentionDuration: durationpb.New(d.subscriptionTTL),
		ExpirationPolicy: &pubsubpb.ExpirationPolicy{
			Ttl: durationpb.New(d.subscriptionTTL),
		},
	}
	
	_, err := d.client.SubscriptionAdminClient.CreateSubscription(ctx, subConfig)
	if err != nil {
		return fmt.Errorf("failed to create monitoring subscription for job %s: %w", jobID, err)
	}
	
	// Store subscription name for cleanup
	d.mu.Lock()
	d.monitorSubName = subName
	d.mu.Unlock()
	
	log.Printf("Created monitoring subscription %s for job %s (TTL: %v)", subName, jobID, d.subscriptionTTL)
	
	// Clean up subscription when done
	defer func() {
		deleteReq := &pubsubpb.DeleteSubscriptionRequest{
			Subscription: subPath,
		}
		if err := d.client.SubscriptionAdminClient.DeleteSubscription(ctx, deleteReq); err != nil {
			log.Printf("Warning: failed to delete monitoring subscription %s: %v", subName, err)
		} else {
			log.Printf("Deleted monitoring subscription %s", subName)
		}
	}()
	
	// Create subscriber for monitoring
	subscriber := d.client.Subscriber(subPath)
	
	// Wait for subscription to drain with regular polling
	return d.waitForSubscriptionDrain(ctx, subscriber, subPath)
}

// waitForSubscriptionDrain polls subscription at regular intervals until it has no undelivered messages
func (d *PubSubDistributor) waitForSubscriptionDrain(ctx context.Context, subscriber *pubsub.Subscriber, subPath string) error {
	startTime := time.Now()
	checkCount := 0
	consecutiveEmptyChecks := 0
	requiredEmptyChecks := 2 // Require 2 consecutive empty checks to ensure stability
	
	log.Printf("Polling subscription every %v (timeout: %v, requires %d consecutive empty checks)", 
		d.pollInterval, d.completionTimeout, requiredEmptyChecks)
	
	for {
		checkCount++
		
		// Check if we've exceeded max wait time
		elapsed := time.Since(startTime)
		if elapsed > d.completionTimeout {
			return fmt.Errorf("timeout waiting for subscription to drain after %v (%d checks)", elapsed, checkCount)
		}
		
		// Check if subscription has any messages
		hasMessages, err := d.subscriptionHasMessages(ctx, subscriber)
		if err != nil {
			log.Printf("Warning: error checking subscription %s (check #%d): %v", subPath, checkCount, err)
			// Continue checking despite error
			consecutiveEmptyChecks = 0 // Reset counter on error
		} else if hasMessages {
			log.Printf("Subscription %s still has messages (check #%d, elapsed: %v)", subPath, checkCount, elapsed.Round(time.Second))
			consecutiveEmptyChecks = 0 // Reset counter when messages found
		} else {
			// No messages found
			consecutiveEmptyChecks++
			log.Printf("Subscription %s appears empty (check #%d, consecutive empty: %d/%d, elapsed: %v)", 
				subPath, checkCount, consecutiveEmptyChecks, requiredEmptyChecks, elapsed.Round(time.Second))
			
			if consecutiveEmptyChecks >= requiredEmptyChecks {
				log.Printf("Subscription %s confirmed drained after %v (%d checks, %d consecutive empty)", 
					subPath, elapsed.Round(time.Second), checkCount, consecutiveEmptyChecks)
				return nil
			}
		}
		
		// Wait before next poll (regular interval, not exponential)
		time.Sleep(d.pollInterval)
	}
}

// subscriptionHasMessages checks if a subscription has any unprocessed messages
// This includes both undelivered messages AND messages currently being processed (unacknowledged)
func (d *PubSubDistributor) subscriptionHasMessages(ctx context.Context, subscriber *pubsub.Subscriber) (bool, error) {
	// The Pub/Sub client library doesn't provide a direct way to check message counts
	// We use a pull-based approach with Receive() to check for deliverable messages
	// To ensure we catch in-flight messages, we use consecutive empty checks in the caller
	return d.subscriptionHasMessagesViaReceive(ctx, subscriber)
}

// subscriptionHasMessagesViaReceive checks for messages by attempting to pull one
func (d *PubSubDistributor) subscriptionHasMessagesViaReceive(ctx context.Context, subscriber *pubsub.Subscriber) (bool, error) {
	hasMessages := false
	
	// Try to pull one message with a reasonable timeout
	// Using 2 seconds to account for network latency
	pullCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	
	// Use a channel to track if we received anything
	receivedChan := make(chan bool, 1)
	
	go func() {
		err := subscriber.Receive(pullCtx, func(ctx context.Context, msg *pubsub.Message) {
			receivedChan <- true
			// Nack immediately - we're just checking, not consuming
			msg.Nack()
			cancel() // Stop receiving
		})
		if err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			log.Printf("Warning: error receiving from subscription: %v", err)
		}
	}()
	
	// Wait for either a message or timeout
	select {
	case <-receivedChan:
		hasMessages = true
	case <-pullCtx.Done():
		// Timeout or cancelled - no messages received
		hasMessages = false
	}
	
	return hasMessages, nil
}

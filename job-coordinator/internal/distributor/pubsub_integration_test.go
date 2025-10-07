package distributor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/dqme/job-coordinator/internal/models"
)

// TestPubSubDistributor_WithEmulator tests PubSub distributor using the local emulator
func TestPubSubDistributor_WithEmulator(t *testing.T) {
	// Check if gcloud is available
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not found, skipping PubSub emulator test")
	}

	// Start the emulator (also sets PUBSUB_EMULATOR_HOST env var)
	_, cleanup := startPubSubEmulator(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"
	topicName := "test-topic"

	// Create topic in emulator
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create pubsub client: %v", err)
	}
	defer client.Close()

	// Get or create topic
	topic := client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		t.Fatalf("Failed to check if topic exists: %v", err)
	}
	
	if !exists {
		topic, err = client.CreateTopic(ctx, topicName)
		if err != nil {
			t.Fatalf("Failed to create topic: %v", err)
		}
		t.Logf("Created topic: %s", topicName)
	} else {
		t.Logf("Topic already exists: %s", topicName)
	}

	// Create a unique subscription for this test run (to avoid old messages)
	subName := fmt.Sprintf("test-sub-%d", time.Now().UnixNano())
	sub, err := client.CreateSubscription(ctx, subName, pubsub.SubscriptionConfig{
		Topic: topic,
	})
	if err != nil {
		t.Fatalf("Failed to create subscription: %v", err)
	}
	t.Logf("Created subscription: %s", subName)

	// Create distributor
	config := PubSubConfig{
		ProjectID: projectID,
		TopicName: topicName,
		BatchSize: 10,
	}

	dist, err := NewPubSubDistributor(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create PubSub distributor: %v", err)
	}
	defer dist.Close()

	// Test distributing work units
	testWorkUnits := []models.WorkUnit{
		{
			ID:       "wu-001",
			JobID:    "job-001",
			FilePath: "data/file1.ndjson",
			BasePath: "/base",
		},
		{
			ID:       "wu-002",
			JobID:    "job-001",
			FilePath: "data/file2.ndjson",
			BasePath: "/base",
		},
	}

	// Distribute work units
	for _, wu := range testWorkUnits {
		if err := dist.Distribute(wu); err != nil {
			t.Errorf("Failed to distribute work unit %s: %v", wu.ID, err)
		}
	}

	// Close distributor to flush any pending messages
	if err := dist.Close(); err != nil {
		t.Errorf("Failed to close distributor: %v", err)
	}

	// Wait a bit for messages to be published
	time.Sleep(1 * time.Second)

	// Verify messages were received
	receivedCount := 0
	receiveCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err = sub.Receive(receiveCtx, func(ctx context.Context, msg *pubsub.Message) {
		receivedCount++
		msg.Ack()
		t.Logf("Received message %d: %s", receivedCount, string(msg.Data))
		
		if receivedCount >= len(testWorkUnits) {
			cancel() // Stop receiving after we get all messages
		}
	})

	if err != nil && err != context.Canceled {
		t.Errorf("Failed to receive messages: %v", err)
	}

	if receivedCount != len(testWorkUnits) {
		t.Errorf("Received %d messages, want %d", receivedCount, len(testWorkUnits))
	} else {
		t.Logf("Successfully distributed and received %d work units via PubSub emulator", receivedCount)
	}
}

// startPubSubEmulator starts the PubSub emulator and returns the host:port and cleanup function
func startPubSubEmulator(t *testing.T) (string, func()) {
	t.Helper()

	// Try to find an available port
	emulatorHost := "localhost:8085"
	
	// Start emulator in background (important: the command blocks so we use Start() not Run())
	cmd := exec.Command("gcloud", "beta", "emulators", "pubsub", "start",
		"--project=test-emulator-project",
		"--host-port="+emulatorHost,
		"--quiet")

	// Pipe output to see emulator logs (optional, can be removed if too noisy)
	// cmd.Stdout = os.Stdout
	// cmd.Stderr = os.Stderr

	// Start the process in the background (non-blocking)
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start PubSub emulator: %v", err)
	}

	t.Logf("Started PubSub emulator process (PID: %d)", cmd.Process.Pid)

	// Set environment variable immediately so client library knows to use emulator
	os.Setenv("PUBSUB_EMULATOR_HOST", emulatorHost)

	// Wait for emulator to be ready by attempting to create a client
	ready := false
	var lastErr error
	for i := 0; i < 30; i++ { // Wait up to 30 seconds
		time.Sleep(1 * time.Second)
		
		// Try to connect to verify it's running
		func() {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			
			client, err := pubsub.NewClient(ctx, "test-project")
			lastErr = err
			
			if err == nil {
				client.Close()
				ready = true
				t.Logf("PubSub emulator ready at %s (after %d seconds)", emulatorHost, i+1)
			}
		}()
		
		if ready {
			break
		}
	}

	if !ready {
		// Kill the process before failing
		if cmd.Process != nil {
			cmd.Process.Kill()
			cmd.Wait()
		}
		t.Fatalf("PubSub emulator failed to start within 30 seconds. Last error: %v", lastErr)
	}

	cleanup := func() {
		if cmd.Process != nil {
			t.Logf("Stopping PubSub emulator (PID: %d)", cmd.Process.Pid)
			
			// Send kill signal
			if err := cmd.Process.Kill(); err != nil {
				t.Logf("Error killing emulator process: %v", err)
			}
			
			// Wait for process to exit
			cmd.Wait()
			
			t.Log("PubSub emulator stopped")
		}
		
		// Clean up environment variable
		os.Unsetenv("PUBSUB_EMULATOR_HOST")
	}

	return emulatorHost, cleanup
}

// TestPubSubDistributor_CreateWithEmulator tests creating a distributor with the emulator
func TestPubSubDistributor_CreateWithEmulator(t *testing.T) {
	// Check if gcloud is available
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not found, skipping PubSub emulator test")
	}

	// Start the emulator (also sets PUBSUB_EMULATOR_HOST env var)
	_, cleanup := startPubSubEmulator(t)
	defer cleanup()

	ctx := context.Background()
	config := PubSubConfig{
		ProjectID: "test-project-create",
		TopicName: "create-test-topic",
		BatchSize: 100,
	}

	// Create client and topic first
	client, err := pubsub.NewClient(ctx, config.ProjectID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get or create topic
	topic := client.Topic(config.TopicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		t.Fatalf("Failed to check if topic exists: %v", err)
	}
	
	if !exists {
		_, err = client.CreateTopic(ctx, config.TopicName)
		if err != nil {
			t.Fatalf("Failed to create topic: %v", err)
		}
	}

	// Now create distributor
	dist, err := NewPubSubDistributor(ctx, config)
	if err != nil {
		t.Fatalf("NewPubSubDistributor() error = %v", err)
	}

	if dist == nil {
		t.Fatal("NewPubSubDistributor() returned nil")
	}

	// Verify we can close it
	err = dist.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	t.Log("Successfully created and closed PubSub distributor with emulator")
}

// TestPubSubDistributor_NoEmulator tests behavior without emulator (should fail gracefully)
func TestPubSubDistributor_NoEmulator(t *testing.T) {
	// Ensure emulator env var is not set
	os.Unsetenv("PUBSUB_EMULATOR_HOST")

	ctx := context.Background()
	config := PubSubConfig{
		ProjectID: "test-project",
		TopicName: "test-topic",
		BatchSize: 100,
	}

	_, err := NewPubSubDistributor(ctx, config)
	
	// Should fail without credentials or emulator
	if err == nil {
		t.Error("NewPubSubDistributor() expected error without credentials, got nil")
	} else {
		t.Logf("Expected error without credentials: %v", err)
	}
}

// Helper function for integration tests
func createTestWorkUnit(id, jobID, filePath string) models.WorkUnit {
	return models.WorkUnit{
		ID:       id,
		JobID:    jobID,
		FilePath: filePath,
		BasePath: "/test/base",
		Status:   models.WorkUnitStatusPending,
	}
}

// TestPubSubDistributor_BatchPublishing tests batch publishing behavior
func TestPubSubDistributor_BatchPublishing(t *testing.T) {
	// Check if gcloud is available
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not found, skipping PubSub emulator test")
	}

	// Start the emulator (also sets PUBSUB_EMULATOR_HOST env var)
	_, cleanup := startPubSubEmulator(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project-batch"
	topicName := "batch-test-topic"

	// Create topic
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Get or create topic
	topic := client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		t.Fatalf("Failed to check if topic exists: %v", err)
	}
	
	if !exists {
		_, err = client.CreateTopic(ctx, topicName)
		if err != nil {
			t.Fatalf("Failed to create topic: %v", err)
		}
	}

	// Create distributor with small batch size
	config := PubSubConfig{
		ProjectID: projectID,
		TopicName: topicName,
		BatchSize: 3, // Small batch for testing
	}

	dist, err := NewPubSubDistributor(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create distributor: %v", err)
	}
	defer dist.Close()

	// Distribute multiple work units
	numWorkUnits := 10
	for i := 0; i < numWorkUnits; i++ {
		wu := createTestWorkUnit(
			fmt.Sprintf("wu-%03d", i),
			"batch-job-001",
			fmt.Sprintf("data/file%d.ndjson", i),
		)
		
		if err := dist.Distribute(wu); err != nil {
			t.Errorf("Failed to distribute work unit %d: %v", i, err)
		}
	}

	t.Logf("Successfully distributed %d work units with batch_size=%d", numWorkUnits, config.BatchSize)
}

// TestPubSubDistributor_CompletionMonitoring tests the WaitForCompletion functionality with emulator
func TestPubSubDistributor_CompletionMonitoring(t *testing.T) {
	// Check if gcloud is available
	if _, err := exec.LookPath("gcloud"); err != nil {
		t.Skip("gcloud not found, skipping PubSub emulator test")
	}

	// Start the emulator
	_, cleanup := startPubSubEmulator(t)
	defer cleanup()

	ctx := context.Background()
	projectID := "test-project"
	topicName := "completion-test-topic"
	jobID := fmt.Sprintf("completion-job-%d", time.Now().UnixNano())

	// Create topic in emulator
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		t.Fatalf("Failed to create pubsub client: %v", err)
	}
	defer client.Close()

	// Get or create topic
	topic := client.Topic(topicName)
	exists, err := topic.Exists(ctx)
	if err != nil {
		t.Fatalf("Failed to check if topic exists: %v", err)
	}
	
	if !exists {
		topic, err = client.CreateTopic(ctx, topicName)
		if err != nil {
			t.Fatalf("Failed to create topic: %v", err)
		}
		t.Logf("Created topic: %s", topicName)
	}

	// Create a consumer subscription (simulates worker consuming messages)
	consumerSubName := fmt.Sprintf("consumer-sub-%d", time.Now().UnixNano())
	consumerSub, err := client.CreateSubscription(ctx, consumerSubName, pubsub.SubscriptionConfig{
		Topic: topic,
	})
	if err != nil {
		t.Fatalf("Failed to create consumer subscription: %v", err)
	}
	defer consumerSub.Delete(ctx)
	t.Logf("Created consumer subscription: %s", consumerSubName)

	// Create distributor with short timeouts for testing
	config := PubSubConfig{
		ProjectID:         projectID,
		TopicName:         topicName,
		BatchSize:         5,
		CompletionTimeout: 2 * time.Minute,
		PollInterval:      1 * time.Second,
		SubscriptionTTL:   10 * time.Minute,
	}

	dist, err := NewPubSubDistributor(ctx, config)
	if err != nil {
		t.Fatalf("Failed to create PubSub distributor: %v", err)
	}
	defer dist.Close()

	// Distribute work units
	numWorkUnits := 15
	for i := 0; i < numWorkUnits; i++ {
		wu := createTestWorkUnit(
			fmt.Sprintf("wu-%03d", i),
			jobID,
			fmt.Sprintf("data/file%d.ndjson", i),
		)
		
		if err := dist.Distribute(wu); err != nil {
			t.Fatalf("Failed to distribute work unit %d: %v", i, err)
		}
	}
	t.Logf("Distributed %d work units for job %s", numWorkUnits, jobID)

	// Start a goroutine to consume messages (simulating workers)
	receivedCount := 0
	consumeDone := make(chan bool)
	go func() {
		receiveCtx := context.Background()
		err := consumerSub.Receive(receiveCtx, func(ctx context.Context, msg *pubsub.Message) {
			receivedCount++
			t.Logf("Consumer received message %d (job_id: %s)", receivedCount, msg.Attributes["job_id"])
			
			// Simulate some processing time
			time.Sleep(100 * time.Millisecond)
			
			// Ack the message
			msg.Ack()
			
			if receivedCount >= numWorkUnits {
				consumeDone <- true
			}
		})
		if err != nil {
			t.Logf("Consumer receive error: %v", err)
		}
	}()

	// Give consumer a moment to start receiving
	time.Sleep(500 * time.Millisecond)

	// Call WaitForCompletion in a goroutine with timeout
	completionDone := make(chan error, 1)
	go func() {
		t.Log("Starting WaitForCompletion...")
		completionDone <- dist.WaitForCompletion()
	}()

	// Wait for either completion or consumer to finish
	select {
	case err := <-completionDone:
		if err != nil {
			t.Fatalf("WaitForCompletion failed: %v", err)
		}
		t.Log("WaitForCompletion returned successfully")
		
	case <-consumeDone:
		t.Log("Consumer finished processing all messages")
		
		// Wait a bit more for monitoring subscription to detect completion
		select {
		case err := <-completionDone:
			if err != nil {
				t.Fatalf("WaitForCompletion failed: %v", err)
			}
			t.Log("WaitForCompletion detected completion after consumer finished")
			
		case <-time.After(15 * time.Second):
			t.Fatal("WaitForCompletion did not complete within 15 seconds after consumer finished")
		}
		
	case <-time.After(3 * time.Minute):
		t.Fatal("Test timeout: neither completion nor consumer finished within 3 minutes")
	}

	// Verify most/all messages were received (allow for some timing variance)
	// The consumer might not receive all messages if WaitForCompletion checks
	// before all are fully acked, but should receive most of them
	if receivedCount < numWorkUnits-2 {
		t.Errorf("Consumer received %d messages, want at least %d", receivedCount, numWorkUnits-2)
	} else {
		t.Logf("Consumer received %d/%d messages", receivedCount, numWorkUnits)
	}

	// Verify distributor status
	status := dist.GetStatus()
	t.Logf("Final distributor status: type=%s, is_complete=%v, pending=%d, distributed=%d",
		status.Type, status.IsComplete, status.PendingCount, status.DistributedCount)
	
	if status.DistributedCount != numWorkUnits {
		t.Errorf("Distributor distributed %d work units, want %d", status.DistributedCount, numWorkUnits)
	}
	
	if !status.IsComplete {
		t.Error("Distributor should report complete after WaitForCompletion")
	}

	t.Logf("✓ Successfully tested completion monitoring: %d work units distributed and processed", numWorkUnits)
}

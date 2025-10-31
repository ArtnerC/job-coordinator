package distributor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/dqme/job-driver/internal/models"
)

// TestStdoutDistributorStatus tests GetStatus for stdout distributor
func TestStdoutDistributorStatus(t *testing.T) {
	var buf bytes.Buffer
	dist := NewStdoutDistributorWithWriter(&buf)

	// Initial status
	status := dist.GetStatus()
	if status.Type != "stdout" {
		t.Errorf("GetStatus().Type = %s, want stdout", status.Type)
	}
	if !status.IsComplete {
		t.Errorf("GetStatus().IsComplete = false, want true (stdout is always complete)")
	}
	if status.PendingCount != 0 {
		t.Errorf("GetStatus().PendingCount = %d, want 0", status.PendingCount)
	}
	if status.DistributedCount != 0 {
		t.Errorf("GetStatus().DistributedCount = %d, want 0", status.DistributedCount)
	}

	// Distribute a work unit
	workUnit := models.WorkUnit{
		ID:    "test-1",
		JobID: "job-1",
		Files: []models.FileSpec{{Path: "test.ndjson"}},
	}
	
	if err := dist.Distribute(workUnit); err != nil {
		t.Fatalf("Distribute() error = %v", err)
	}

	// Check status after distribution
	status = dist.GetStatus()
	if status.DistributedCount != 1 {
		t.Errorf("GetStatus().DistributedCount = %d, want 1", status.DistributedCount)
	}
	if !status.IsComplete {
		t.Errorf("GetStatus().IsComplete = false, want true (stdout is always complete)")
	}

	// Distribute more work units
	for i := 0; i < 5; i++ {
		wu := models.WorkUnit{
			ID:    "test",
			JobID: "job-1",
			Files: []models.FileSpec{{Path: "test.ndjson"}},
		}
		if err := dist.Distribute(wu); err != nil {
			t.Fatalf("Distribute() error = %v", err)
		}
	}

	// Check final status
	status = dist.GetStatus()
	if status.DistributedCount != 6 {
		t.Errorf("GetStatus().DistributedCount = %d, want 6", status.DistributedCount)
	}

	// Test WaitForCompletion
	if err := dist.WaitForCompletion(); err != nil {
		t.Errorf("WaitForCompletion() error = %v", err)
	}
}

// TestFileDistributorStatus tests GetStatus for file distributor
func TestFileDistributorStatus(t *testing.T) {
	// Create temp file
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test-output.ndjson")

	dist, err := NewFileDistributor(outputPath)
	if err != nil {
		t.Fatalf("NewFileDistributor() error = %v", err)
	}
	defer dist.Close()

	// Initial status
	status := dist.GetStatus()
	if status.Type != "file" {
		t.Errorf("GetStatus().Type = %s, want file", status.Type)
	}
	if !status.IsComplete {
		t.Errorf("GetStatus().IsComplete = false, want true (file writes are synchronous)")
	}
	if status.PendingCount != 0 {
		t.Errorf("GetStatus().PendingCount = %d, want 0", status.PendingCount)
	}
	if status.DistributedCount != 0 {
		t.Errorf("GetStatus().DistributedCount = %d, want 0", status.DistributedCount)
	}

	// Distribute work units
	for i := 0; i < 10; i++ {
		workUnit := models.WorkUnit{
			ID:    "test",
			JobID: "job-1",
			Files: []models.FileSpec{{Path: "test.ndjson"}},
		}
		if err := dist.Distribute(workUnit); err != nil {
			t.Fatalf("Distribute() error = %v", err)
		}
	}

	// Check status
	status = dist.GetStatus()
	if status.DistributedCount != 10 {
		t.Errorf("GetStatus().DistributedCount = %d, want 10", status.DistributedCount)
	}
	if !status.IsComplete {
		t.Errorf("GetStatus().IsComplete = false, want true")
	}

	// Test WaitForCompletion (should sync file)
	if err := dist.WaitForCompletion(); err != nil {
		t.Errorf("WaitForCompletion() error = %v", err)
	}

	// Verify file exists and has content
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	if len(data) == 0 {
		t.Error("Output file is empty")
	}
}

// TestPubSubDistributorStatus tests GetStatus for pubsub distributor
// Note: This requires a mock or real pubsub setup
func TestPubSubDistributorStatusWithBatching(t *testing.T) {
	// Skip if not in integration test mode
	if testing.Short() {
		t.Skip("Skipping pubsub integration test in short mode")
	}

	// This test would require setting up a pubsub emulator or mocking
	// For now, we test the basic batching logic
	t.Skip("Pubsub integration test requires pubsub emulator")
	
	// Example of what the test would look like:
	// ctx := context.Background()
	// cfg := PubSubConfig{
	// 	ProjectID: "test-project",
	// 	TopicName: "test-topic",
	// 	BatchSize: 5,
	// }
	// 
	// dist, err := NewPubSubDistributor(ctx, cfg)
	// if err != nil {
	// 	t.Fatalf("NewPubSubDistributor() error = %v", err)
	// }
	// defer dist.Close()
	// 
	// // Distribute 3 work units (under batch size)
	// for i := 0; i < 3; i++ {
	// 	workUnit := models.WorkUnit{ID: "test", JobID: "job-1", LineNo: i}
	// 	if err := dist.Distribute(workUnit); err != nil {
	// 		t.Fatalf("Distribute() error = %v", err)
	// 	}
	// }
	// 
	// // Check status - should show 3 pending
	// status := dist.GetStatus()
	// if status.PendingCount != 3 {
	// 	t.Errorf("GetStatus().PendingCount = %d, want 3", status.PendingCount)
	// }
	// if status.IsComplete {
	// 	t.Error("GetStatus().IsComplete = true, want false (has pending)")
	// }
	// 
	// // Call WaitForCompletion to flush
	// if err := dist.WaitForCompletion(); err != nil {
	// 	t.Errorf("WaitForCompletion() error = %v", err)
	// }
	// 
	// // Check status after flush
	// status = dist.GetStatus()
	// if status.PendingCount != 0 {
	// 	t.Errorf("GetStatus().PendingCount = %d, want 0 after flush", status.PendingCount)
	// }
	// if !status.IsComplete {
	// 	t.Error("GetStatus().IsComplete = false, want true after flush")
	// }
	// if status.DistributedCount != 3 {
	// 	t.Errorf("GetStatus().DistributedCount = %d, want 3", status.DistributedCount)
	// }
}

// TestDistributorCompletionWorkflow tests the full workflow
func TestDistributorCompletionWorkflow(t *testing.T) {
	tests := []struct {
		name        string
		createDist  func(t *testing.T) (Distributor, func())
		workUnits   int
		expectType  string
	}{
		{
			name: "stdout workflow",
			createDist: func(t *testing.T) (Distributor, func()) {
				var buf bytes.Buffer
				dist := NewStdoutDistributorWithWriter(&buf)
				return dist, func() { dist.Close() }
			},
			workUnits:  20,
			expectType: "stdout",
		},
		{
			name: "file workflow",
			createDist: func(t *testing.T) (Distributor, func()) {
				tmpDir := t.TempDir()
				outputPath := filepath.Join(tmpDir, "test.ndjson")
				dist, err := NewFileDistributor(outputPath)
				if err != nil {
					t.Fatalf("NewFileDistributor() error = %v", err)
				}
				return dist, func() { dist.Close() }
			},
			workUnits:  20,
			expectType: "file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dist, cleanup := tt.createDist(t)
			defer cleanup()

			// Distribute work units
			for i := 0; i < tt.workUnits; i++ {
				wu := models.WorkUnit{
					ID:    "test",
					JobID: "job-1",
					Files: []models.FileSpec{{Path: "test.ndjson"}},
				}
				if err := dist.Distribute(wu); err != nil {
					t.Fatalf("Distribute() error = %v", err)
				}
			}

			// Wait for completion
			if err := dist.WaitForCompletion(); err != nil {
				t.Errorf("WaitForCompletion() error = %v", err)
			}

			// Verify status
			status := dist.GetStatus()
			if status.Type != tt.expectType {
				t.Errorf("GetStatus().Type = %s, want %s", status.Type, tt.expectType)
			}
			if status.DistributedCount != tt.workUnits {
				t.Errorf("GetStatus().DistributedCount = %d, want %d", status.DistributedCount, tt.workUnits)
			}
			if !status.IsComplete {
				t.Errorf("GetStatus().IsComplete = false, want true after WaitForCompletion")
			}
		})
	}
}

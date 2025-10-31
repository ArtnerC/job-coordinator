package coordinator

import (
	"context"
	"testing"
	"time"

	"github.com/dqme/job-driver/internal/config"
	"github.com/dqme/job-driver/internal/distributor"
	"github.com/dqme/job-driver/internal/models"
)

func TestNewCoordinator(t *testing.T) {
	cfg := &config.Config{
		JobID:                    "test-job",
		BatchSize:                500,
		BasePath:                 "C:\\test\\data",
		MeasuresToRun:            []string{"/measure1.json"},
		DistributorType:          "stdout",
		ConcurrentFileProcessors: 10,
		CompletionTTL:            10 * time.Minute,
	}

	job := &models.Job{
		ID:             "test-job",
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ErrorCount:     0,
	}

	dist := distributor.NewStdoutDistributor()
	coord := NewCoordinator(job, cfg, dist)

	if coord == nil {
		t.Fatal("NewCoordinator returned nil")
	}

	if coord.GetJob().ID != "test-job" {
		t.Errorf("Job ID = %s, want test-job", coord.GetJob().ID)
	}

	if coord.GetConfig().JobID != "test-job" {
		t.Errorf("Config JobID = %s, want test-job", coord.GetConfig().JobID)
	}
}

func TestCoordinatorStateTransitions(t *testing.T) {
	cfg := &config.Config{
		JobID:                    "test-job",
		BatchSize:                500,
		BasePath:                 "C:\\test\\data",
		MeasuresToRun:            []string{"/measure1.json"},
		DistributorType:          "stdout",
		ConcurrentFileProcessors: 10,
		CompletionTTL:            10 * time.Minute,
	}

	tests := []struct {
		name          string
		initialStatus models.JobStatus
		action        func(*Coordinator) error
		expectError   bool
		finalStatus   models.JobStatus
	}{
		{
			name:          "Start from pending",
			initialStatus: models.JobStatusPending,
			action:        func(c *Coordinator) error { return c.Start() },
			expectError:   false,
			finalStatus:   models.JobStatusRunning,
		},
		{
			name:          "Start from paused",
			initialStatus: models.JobStatusPaused,
			action:        func(c *Coordinator) error { return c.Resume() },
			expectError:   false,
			finalStatus:   models.JobStatusRunning,
		},
		{
			name:          "Pause running job",
			initialStatus: models.JobStatusRunning,
			action:        func(c *Coordinator) error { return c.Pause() },
			expectError:   false,
			finalStatus:   models.JobStatusPaused,
		},
		{
			name:          "Cancel running job",
			initialStatus: models.JobStatusRunning,
			action:        func(c *Coordinator) error { return c.Cancel() },
			expectError:   false,
			finalStatus:   models.JobStatusCancelled,
		},
		{
			name:          "Cannot start completed job",
			initialStatus: models.JobStatusCompleted,
			action:        func(c *Coordinator) error { return c.Start() },
			expectError:   true,
			finalStatus:   models.JobStatusCompleted,
		},
		{
			name:          "Cannot pause pending job",
			initialStatus: models.JobStatusPending,
			action:        func(c *Coordinator) error { return c.Pause() },
			expectError:   true,
			finalStatus:   models.JobStatusPending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			job := &models.Job{
				ID:             "test-job",
				Status:         tt.initialStatus,
				TotalWorkUnits: 0,
				ErrorCount:     0,
			}

			dist := distributor.NewStdoutDistributor()
			coord := NewCoordinator(job, cfg, dist)

			err := tt.action(coord)

			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			if coord.GetJob().Status != tt.finalStatus {
				t.Errorf("Final status = %s, want %s", coord.GetJob().Status, tt.finalStatus)
			}
		})
	}
}

func TestCoordinatorUpdateConfig(t *testing.T) {
	cfg := &config.Config{
		JobID:                    "test-job",
		BatchSize:                500,
		BasePath:                 "C:\\test\\data",
		MeasuresToRun:            []string{"/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{"key": "value"},
		ConcurrentFileProcessors: 10,
		CompletionTTL:            10 * time.Minute,
	}

	job := &models.Job{
		ID:             "test-job",
		Status:         models.JobStatusRunning,
		TotalWorkUnits: 0,
		ErrorCount:     0,
	}

	dist := distributor.NewStdoutDistributor()
	coord := NewCoordinator(job, cfg, dist)

	// Update runtime-modifiable fields
	newCfg := &config.Config{
		ConcurrentFileProcessors: 20,
		DistributorConfig:        map[string]string{"key": "newvalue"},
	}

	err := coord.UpdateConfig(newCfg)
	if err != nil {
		t.Fatalf("UpdateConfig failed: %v", err)
	}

	updatedCfg := coord.GetConfig()
	if updatedCfg.ConcurrentFileProcessors != 20 {
		t.Errorf("ConcurrentFileProcessors = %d, want 20", updatedCfg.ConcurrentFileProcessors)
	}

	if updatedCfg.DistributorConfig["key"] != "newvalue" {
		t.Errorf("DistributorConfig[key] = %s, want newvalue", updatedCfg.DistributorConfig["key"])
	}
}

func TestTTLManager(t *testing.T) {
	ttl := 100 * time.Millisecond
	manager := NewTTLManager(ttl)

	if manager == nil {
		t.Fatal("NewTTLManager returned nil")
	}

	start := time.Now()
	doneChan := manager.StartTTL()

	// Wait for TTL to expire
	<-doneChan
	elapsed := time.Since(start)

	if elapsed < ttl {
		t.Errorf("TTL expired too early: %v < %v", elapsed, ttl)
	}

	// Should be close to ttl (within 50ms tolerance)
	if elapsed > ttl+50*time.Millisecond {
		t.Errorf("TTL expired too late: %v > %v", elapsed, ttl+50*time.Millisecond)
	}
}

func TestTTLManagerCancel(t *testing.T) {
	ttl := 1 * time.Second
	manager := NewTTLManager(ttl)

	doneChan := manager.StartTTL()

	// Cancel immediately
	manager.Cancel()

	select {
	case <-doneChan:
		// Expected - channel should close quickly
	case <-time.After(200 * time.Millisecond):
		t.Error("TTL cancel did not trigger channel close within 200ms")
	}
}

func TestDiscoverMeasures_RelativePaths(t *testing.T) {
	// Test that discoverMeasures returns relative paths (just filenames)
	cfg := &config.Config{
		JobID:                    "test-job",
		BatchSize:                500,
		BasePath:                 "../../testdata/bundles",
		MeasuresPath:             "../../testdata/measures",
		DistributorType:          "stdout",
		ConcurrentFileProcessors: 10,
		CompletionTTL:            10 * time.Minute,
	}

	job := &models.Job{
		ID:             "test-job",
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ErrorCount:     0,
	}

	dist := distributor.NewStdoutDistributor()
	coord := NewCoordinator(job, cfg, dist)

	// Discover measures using the public method
	measures, err := coord.discoverMeasures()
	if err != nil {
		t.Fatalf("discoverMeasures failed: %v", err)
	}

	// Check that all returned paths are relative (no leading slashes or drive letters)
	for _, measure := range measures {
		// Should not have absolute path indicators
		if len(measure) > 0 && (measure[0] == '/' || measure[0] == '\\') {
			t.Errorf("Measure path %q starts with slash - should be relative", measure)
		}
		// On Windows, should not have drive letter
		if len(measure) > 1 && measure[1] == ':' {
			t.Errorf("Measure path %q contains drive letter - should be relative", measure)
		}
		// Should not contain the base directory name
		if len(measure) > len("measures") && measure[0:8] == "measures" {
			t.Errorf("Measure path %q contains 'measures' directory - should be relative to measures dir", measure)
		}
	}

	// At least one measure should be found
	if len(measures) == 0 {
		t.Error("No measures found in testdata/measures")
	}
}

func TestGracefulShutdown(t *testing.T) {
	cfg := &config.Config{
		JobID:                    "test-job",
		BatchSize:                500,
		BasePath:                 "C:\\test\\data",
		MeasuresToRun:            []string{"/measure1.json"},
		DistributorType:          "stdout",
		ConcurrentFileProcessors: 10,
		CompletionTTL:            10 * time.Minute,
	}

	job := &models.Job{
		ID:             "test-job",
		Status:         models.JobStatusCompleted,
		TotalWorkUnits: 10,
		ErrorCount:     0,
	}

	dist := distributor.NewStdoutDistributor()
	coord := NewCoordinator(job, cfg, dist)

	ctx := context.Background()
	err := GracefulShutdown(ctx, coord, 5*time.Second)

	if err != nil {
		t.Errorf("GracefulShutdown failed: %v", err)
	}
}

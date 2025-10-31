package coordinator

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/config"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/distributor"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/models"
)

// TestScaleLoadProcessing tests that scale load mode generates the correct number of work units
func TestScaleLoadProcessing(t *testing.T) {
	// Create a temporary directory with a test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.ndjson")
	
	// Create a file with 3 lines
	content := `{"resourceType":"Bundle"}
{"resourceType":"Bundle"}
{"resourceType":"Bundle"}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create measures directory
	measuresDir := filepath.Join(tempDir, "measures")
	if err := os.MkdirAll(measuresDir, 0755); err != nil {
		t.Fatalf("Failed to create measures directory: %v", err)
	}

	tests := []struct {
		name               string
		scaleLoad          int
		batchSize          int
		remainderThreshold float64
		expectedWorkUnits  int
	}{
		{
			name:               "10 lines from 3-line file",
			scaleLoad:          10,
			batchSize:          2,
			remainderThreshold: 0.2,
			expectedWorkUnits:  7, // pattern: 2,1 repeated -> 2,1,2,1,2,1,1
		},
		{
			name:               "5 lines from 3-line file",
			scaleLoad:          5,
			batchSize:          2,
			remainderThreshold: 0.2,
			expectedWorkUnits:  3, // pattern: 2,1 repeated -> 2,1,2
		},
		{
			name:               "100 lines from 3-line file",
			scaleLoad:          100,
			batchSize:          10,
			remainderThreshold: 0.2,
			expectedWorkUnits:  34, // pattern: 3 repeated 33 times + 1 line
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create stdout distributor to capture work units
			cfg := &config.Config{
				JobID:                    "test-scale-load",
				BasePath:                 tempDir,
				MeasuresPath:             measuresDir,
				BatchSize:                tt.batchSize,
				RemainderThreshold:       tt.remainderThreshold,
				ScaleLoad:                &tt.scaleLoad,
				DistributorType:          "stdout",
				DistributorConfig:        make(map[string]string),
				ConcurrentFileProcessors: 1,
			}

			dist, err := distributor.NewDistributor(context.Background(), *cfg)
			if err != nil {
				t.Fatalf("Failed to create distributor: %v", err)
			}

			job := &models.Job{
				ID:             cfg.JobID,
				Status:         models.JobStatusPending,
				TotalWorkUnits: 0,
			}

			coord := NewCoordinator(job, cfg, dist)

			// Process the job
			if err := coord.ProcessJob(); err != nil {
				t.Fatalf("ProcessJob failed: %v", err)
			}

			// Check that the correct number of work units were created
			finalJob := coord.GetJob()
			if finalJob.TotalWorkUnits != tt.expectedWorkUnits {
				t.Errorf("Expected %d work units, got %d", tt.expectedWorkUnits, finalJob.TotalWorkUnits)
			}

			if finalJob.Status != models.JobStatusCompleted {
				t.Errorf("Expected job status completed, got %s", finalJob.Status)
			}
		})
	}
}

// TestBatchDistribution tests that files are correctly split into batches
func TestBatchDistribution(t *testing.T) {
	// Create a temporary directory with test files
	tempDir := t.TempDir()
	
	// Create test files with different sizes
	testFiles := map[string]int{
		"small.ndjson":  1,  // 1 line
		"medium.ndjson": 10, // 10 lines
		"large.ndjson":  50, // 50 lines
	}

	for filename, lineCount := range testFiles {
		content := ""
		for i := 0; i < lineCount; i++ {
			content += `{"resourceType":"Bundle"}` + "\n"
		}
		if err := os.WriteFile(filepath.Join(tempDir, filename), []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Create measures directory
	measuresDir := filepath.Join(tempDir, "measures")
	if err := os.MkdirAll(measuresDir, 0755); err != nil {
		t.Fatalf("Failed to create measures directory: %v", err)
	}

	tests := []struct {
		name               string
		batchSize          int
		remainderThreshold float64
		expectedWorkUnits  int // Total for all 3 files: 1, 10, 50 lines
	}{
		{
			name:               "batch_size=10, threshold=0.2",
			batchSize:          10,
			remainderThreshold: 0.2,
			expectedWorkUnits:  7, // small:1 batch, medium:1 batch, large:5 batches
		},
		{
			name:               "batch_size=5, threshold=0.2",
			batchSize:          5,
			remainderThreshold: 0.2,
			expectedWorkUnits:  13, // small:1, medium:2, large:10
		},
		{
			name:               "batch_size=20, threshold=0.2",
			batchSize:          20,
			remainderThreshold: 0.2,
			expectedWorkUnits:  5, // small:1, medium:1, large:3 (20+20+10)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create stdout distributor
			cfg := &config.Config{
				JobID:                    "test-batch-dist",
				BasePath:                 tempDir,
				MeasuresPath:             measuresDir,
				BatchSize:                tt.batchSize,
				RemainderThreshold:       tt.remainderThreshold,
				DistributorType:          "stdout",
				DistributorConfig:        make(map[string]string),
				ConcurrentFileProcessors: 1,
			}

			dist, err := distributor.NewDistributor(context.Background(), *cfg)
			if err != nil {
				t.Fatalf("Failed to create distributor: %v", err)
			}

			job := &models.Job{
				ID:             cfg.JobID,
				Status:         models.JobStatusPending,
				TotalWorkUnits: 0,
			}

			coord := NewCoordinator(job, cfg, dist)

			// Process the job
			if err := coord.ProcessJob(); err != nil {
				t.Fatalf("ProcessJob failed: %v", err)
			}

			// Check that the correct number of work units were created
			finalJob := coord.GetJob()
			if finalJob.TotalWorkUnits != tt.expectedWorkUnits {
				t.Errorf("Expected %d work units, got %d", tt.expectedWorkUnits, finalJob.TotalWorkUnits)
			}

			if finalJob.Status != models.JobStatusCompleted {
				t.Errorf("Expected job status completed, got %s", finalJob.Status)
			}
		})
	}
}

// TestScaleLoadWithSmallFile tests scale load mode with a file smaller than batch size
func TestScaleLoadWithSmallFile(t *testing.T) {
	// Create a temporary directory with a 1-line test file
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "tiny.ndjson")
	
	content := `{"resourceType":"Bundle"}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create measures directory
	measuresDir := filepath.Join(tempDir, "measures")
	if err := os.MkdirAll(measuresDir, 0755); err != nil {
		t.Fatalf("Failed to create measures directory: %v", err)
	}

	scaleLoad := 100
	cfg := &config.Config{
		JobID:                    "test-scale-load-small",
		BasePath:                 tempDir,
		MeasuresPath:             measuresDir,
		BatchSize:                10,
		RemainderThreshold:       0.2,
		ScaleLoad:                &scaleLoad,
		DistributorType:          "stdout",
		DistributorConfig:        make(map[string]string),
		ConcurrentFileProcessors: 1,
	}

	dist, err := distributor.NewDistributor(context.Background(), *cfg)
	if err != nil {
		t.Fatalf("Failed to create distributor: %v", err)
	}

	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
	}

	coord := NewCoordinator(job, cfg, dist)

	// Process the job
	if err := coord.ProcessJob(); err != nil {
		t.Fatalf("ProcessJob failed: %v", err)
	}

	// Check that 100 work units were created (each with 1 line)
	finalJob := coord.GetJob()
	if finalJob.TotalWorkUnits != 100 {
		t.Errorf("Expected 100 work units, got %d", finalJob.TotalWorkUnits)
	}

	if finalJob.Status != models.JobStatusCompleted {
		t.Errorf("Expected job status completed, got %s", finalJob.Status)
	}
}

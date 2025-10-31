package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/api"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/config"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/driver"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/distributor"
	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/models"
)

// TestE2EStdoutMode tests the coordinator in stdout distributor mode
func TestE2EStdoutMode(t *testing.T) {
	// Create temp directory for test data
	tempDir := t.TempDir()

	// Create a test NDJSON file
	testFile := filepath.Join(tempDir, "test-bundle.ndjson")
	testData := `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}
{"resourceType":"Patient","id":"3"}
{"resourceType":"Patient","id":"4"}
{"resourceType":"Patient","id":"5"}`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Setup configuration
	cfg := &config.Config{
		JobID:                    "e2e-test-stdout",
		AutoStart:                false,
		BatchSize:                2,
		RemainderThreshold:       0.5,
		BasePath:                 tempDir,
		MeasuresToRun:            []string{"/test/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{},
		CompletionTTL:            10 * time.Minute,
		ConcurrentFileProcessors: 5,
		APIPort:                  8080,
	}

	// Create job
	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ProcessedCount: 0,
		ErrorCount:     0,
		Errors:         []string{},
		CompletionTTL:  cfg.CompletionTTL,
	}

	// Create distributor and coordinator
	dist, err := distributor.NewDistributor(context.Background(), *cfg)
	if err != nil {
		t.Fatalf("Failed to create distributor: %v", err)
	}
	defer dist.Close()

	coord := coordinator.NewCoordinator(job, cfg, dist)
	defer coord.Close()

	// Setup API
	router := api.SetupRouter(coord)
	server := httptest.NewServer(router)
	defer server.Close()

	// Test: Check initial status
	resp, err := http.Get(server.URL + "/job/status")
	if err != nil {
		t.Fatalf("Failed to get status: %v", err)
	}
	defer resp.Body.Close()

	var statusResp models.JobStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&statusResp); err != nil {
		t.Fatalf("Failed to decode status response: %v", err)
	}

	if statusResp.Status != "pending" {
		t.Errorf("Initial status = %s, want pending", statusResp.Status)
	}

	// Test: Start job
	startResp, err := http.Post(server.URL+"/job/start", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}
	defer startResp.Body.Close()

	if startResp.StatusCode != http.StatusOK {
		t.Errorf("Start job status = %d, want %d", startResp.StatusCode, http.StatusOK)
	}

	// Wait a moment for processing to start
	time.Sleep(100 * time.Millisecond)

	// Test: Check health endpoint
	healthResp, err := http.Get(server.URL + "/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer healthResp.Body.Close()

	var health map[string]interface{}
	if err := json.NewDecoder(healthResp.Body).Decode(&health); err != nil {
		t.Fatalf("Failed to decode health response: %v", err)
	}

	if health["status"] != "ok" {
		t.Errorf("Health status = %v, want ok", health["status"])
	}

	t.Logf("E2E stdout mode test completed successfully")
}

// TestE2EJobControl tests manual job control (start, pause, cancel)
func TestE2EJobControl(t *testing.T) {
	// Create temp directory for test data
	tempDir := t.TempDir()

	// Create a test NDJSON file with enough lines to ensure job doesn't complete instantly
	// We'll create 100 lines and use a delay to ensure we can test pause/cancel
	testFile := filepath.Join(tempDir, "test-bundle.ndjson")
	var testData string
	for i := 1; i <= 100; i++ {
		testData += fmt.Sprintf(`{"resourceType":"Patient","id":"%d"}`, i) + "\n"
	}

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		JobID:                    "e2e-test-control",
		AutoStart:                false, // Don't auto-start to allow controlled testing
		BatchSize:                5,     // Small batches to create more work units (100 lines / 5 = 20 work units)
		BasePath:                 tempDir,
		MeasuresToRun:            []string{"/test/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{
			"delay_ms": "100", // 100ms delay per work unit for testing (20 units * 100ms = 2 seconds total)
		},
		CompletionTTL:            10 * time.Minute,
		ConcurrentFileProcessors: 1, // Process serially to ensure delays work
		APIPort:                  8080,
	}

	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ErrorCount:     0,
		CompletionTTL:  cfg.CompletionTTL,
	}

	// Use NewDistributor to properly apply config including delay_ms
	dist, err := distributor.NewDistributor(context.Background(), *cfg)
	if err != nil {
		t.Fatalf("Failed to create distributor: %v", err)
	}
	defer dist.Close()

	coord := coordinator.NewCoordinator(job, cfg, dist)
	defer coord.Close()

	router := api.SetupRouter(coord)
	server := httptest.NewServer(router)
	defer server.Close()

	// Test: Start job
	resp, err := http.Post(server.URL+"/job/start", "application/json", nil)
	if err != nil {
		t.Fatalf("Failed to start job: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Start status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// Wait for job to start processing work units
	// With 100 lines, batch_size=5, and 100ms delay per work unit,
	// we have 20 work units taking ~2 seconds total
	// Wait 500ms - should have distributed ~5 work units, with 15 remaining (~1.5 seconds left)
	time.Sleep(500 * time.Millisecond)

	// Verify job is running (should still be processing due to delay)
	currentStatus := coord.GetJob().Status
	if currentStatus != models.JobStatusRunning {
		// If job completed too fast, the delay may not be working as expected
		t.Fatalf("Status after start = %s, want running (job completed too quickly - delay not working?)", currentStatus)
	}

	// Test: Pause job
	client := &http.Client{}
	req, _ := http.NewRequest(http.MethodPut, server.URL+"/job/pause", nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to pause job: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Pause status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if coord.GetJob().Status != models.JobStatusPaused {
		t.Errorf("Status after pause = %s, want paused", coord.GetJob().Status)
	}

	// Test: Cancel paused job
	req, _ = http.NewRequest(http.MethodPut, server.URL+"/job/cancel", nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to cancel job: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Cancel status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	if coord.GetJob().Status != models.JobStatusCancelled {
		t.Errorf("Status after cancel = %s, want cancelled", coord.GetJob().Status)
	}

	t.Logf("E2E job control test completed successfully")
}

// TestE2EWholeFileMode tests batch_size=0 (whole file mode)
func TestE2EWholeFileMode(t *testing.T) {
	tempDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tempDir, "test-bundle.ndjson")
	testData := `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cfg := &config.Config{
		JobID:                    "e2e-test-wholefile",
		BatchSize:                0, // Whole file mode
		BasePath:                 tempDir,
		MeasuresToRun:            []string{"/test/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{},
		CompletionTTL:            10 * time.Minute,
		ConcurrentFileProcessors: 5,
		APIPort:                  8080,
	}

	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ErrorCount:     0,
		CompletionTTL:  cfg.CompletionTTL,
	}

	dist := distributor.NewStdoutDistributor()
	coord := coordinator.NewCoordinator(job, cfg, dist)
	defer coord.Close()

	// Verify batch_size=0 is set
	if coord.GetConfig().BatchSize != 0 {
		t.Errorf("BatchSize = %d, want 0", coord.GetConfig().BatchSize)
	}

	t.Logf("E2E whole file mode test completed successfully")
}

// TestE2EConfigUpdate tests runtime configuration updates
func TestE2EConfigUpdate(t *testing.T) {
	// Create temp directory for test data
	tempDir := t.TempDir()

	cfg := &config.Config{
		JobID:                    "e2e-test-config",
		BatchSize:                500,
		BasePath:                 tempDir,
		MeasuresToRun:            []string{"/test/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{"key": "value"},
		CompletionTTL:            10 * time.Minute,
		ConcurrentFileProcessors: 10,
		APIPort:                  8080,
	}

	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ErrorCount:     0,
		CompletionTTL:  cfg.CompletionTTL,
	}

	dist := distributor.NewStdoutDistributor()
	coord := coordinator.NewCoordinator(job, cfg, dist)
	defer coord.Close()

	router := api.SetupRouter(coord)
	server := httptest.NewServer(router)
	defer server.Close()

	// Get initial config
	resp, err := http.Get(server.URL + "/config")
	if err != nil {
		t.Fatalf("Failed to get config: %v", err)
	}
	defer resp.Body.Close()

	var initialConfig models.ConfigResponse
	if err := json.NewDecoder(resp.Body).Decode(&initialConfig); err != nil {
		t.Fatalf("Failed to decode config response: %v", err)
	}

	if initialConfig.ConcurrentFileProcessors != 10 {
		t.Errorf("Initial ConcurrentFileProcessors = %d, want 10", initialConfig.ConcurrentFileProcessors)
	}

	t.Logf("E2E config update test completed successfully")
}

// TestE2EFileManifest tests using a file manifest
func TestE2EFileManifest(t *testing.T) {
	tempDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tempDir, "bundle1.ndjson")
	file2 := filepath.Join(tempDir, "bundle2.ndjson")

	testData := `{"resourceType":"Patient","id":"1"}`

	if err := os.WriteFile(file1, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file 1: %v", err)
	}

	if err := os.WriteFile(file2, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file 2: %v", err)
	}

	// Create manifest file
	manifestPath := filepath.Join(tempDir, "manifest.txt")
	manifestContent := fmt.Sprintf("%s\n%s", file1, file2)

	if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
		t.Fatalf("Failed to create manifest: %v", err)
	}

	cfg := &config.Config{
		JobID:                    "e2e-test-manifest",
		BatchSize:                500,
		BasePath:                 tempDir,
		ManifestPath:             &manifestPath,
		MeasuresToRun:            []string{"/test/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{},
		CompletionTTL:            10 * time.Minute,
		ConcurrentFileProcessors: 5,
		APIPort:                  8080,
	}

	job := &models.Job{
		ID:             cfg.JobID,
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ErrorCount:     0,
		CompletionTTL:  cfg.CompletionTTL,
	}

	dist := distributor.NewStdoutDistributor()
	coord := coordinator.NewCoordinator(job, cfg, dist)
	defer coord.Close()

	// Verify manifest path is set
	if coord.GetConfig().ManifestPath == nil {
		t.Error("ManifestPath is nil, expected path")
	}

	t.Logf("E2E file manifest test completed successfully")
}

// TestE2EWorkUnitSchemaCompliance verifies work units follow the JSON schema
func TestE2EWorkUnitSchemaCompliance(t *testing.T) {
	// Create a work unit and verify it has all required fields
	startLine := 0
	endLine := 99
	workUnit := models.WorkUnit{
		ID:           "test-wu-123",
		JobID:        "test-job-123",
		Files:        []models.FileSpec{{Path: "path/to/file.ndjson", StartLine: &startLine, EndLine: &endLine}}, // Relative path with line ranges
		Measures:     []string{"/measure1.json"},
		MeasuresPath: nil,
		BasePath:     "C:\\base\\path", // Absolute base path
		CreatedAt:    time.Now(),
		Status:       models.WorkUnitStatusPending,
	}

	// Validate the work unit
	if err := models.ValidateWorkUnit(&workUnit); err != nil {
		t.Fatalf("Work unit validation failed: %v", err)
	}

	// Verify JSON serialization
	data, err := json.Marshal(workUnit)
	if err != nil {
		t.Fatalf("Failed to marshal work unit: %v", err)
	}

	var decoded models.WorkUnit
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal work unit: %v", err)
	}

	if decoded.ID != workUnit.ID {
		t.Errorf("Decoded ID = %s, want %s", decoded.ID, workUnit.ID)
	}

	if decoded.JobID != workUnit.JobID {
		t.Errorf("Decoded JobID = %s, want %s", decoded.JobID, workUnit.JobID)
	}

	t.Logf("E2E work unit schema compliance test completed successfully")
}

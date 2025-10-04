package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/coordinator"
	"github.com/dqme/job-coordinator/internal/distributor"
	"github.com/dqme/job-coordinator/internal/models"
)

func setupTestCoordinator() *coordinator.Coordinator {
	cfg := &config.Config{
		JobID:                    "test-job-123",
		BatchSize:                500,
		RemainderThreshold:       0.2,
		BasePath:                 "C:\\test\\data",
		MeasuresToRun:            []string{"/measure1.json"},
		DistributorType:          "stdout",
		DistributorConfig:        map[string]string{},
		ConcurrentFileProcessors: 10,
		CompletionTTL:            10 * time.Minute,
		APIPort:                  8080,
	}

	job := &models.Job{
		ID:             "test-job-123",
		Status:         models.JobStatusPending,
		TotalWorkUnits: 0,
		ProcessedCount: 0,
		ErrorCount:     0,
		Errors:         []string{},
		CompletionTTL:  10 * time.Minute,
	}

	dist := distributor.NewStdoutDistributor()
	return coordinator.NewCoordinator(job, cfg, dist)
}

func TestStartJobHandler(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		initialStatus  models.JobStatus
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "POST start pending job",
			method:         http.MethodPost,
			initialStatus:  models.JobStatusPending,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "POST start already running",
			method:         http.MethodPost,
			initialStatus:  models.JobStatusRunning,
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "GET not allowed",
			method:         http.MethodGet,
			initialStatus:  models.JobStatusPending,
			expectedStatus: http.StatusMethodNotAllowed,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coord := setupTestCoordinator()
			coord.GetJob().Status = tt.initialStatus

			handler := NewHandler(coord)
			req := httptest.NewRequest(tt.method, "/job/start", nil)
			rec := httptest.NewRecorder()

			handler.StartJobHandler(rec, req)

			if rec.Code != tt.expectedStatus {
				t.Errorf("Status code = %d, want %d", rec.Code, tt.expectedStatus)
			}

			if tt.expectError {
				var errResp map[string]string
				if err := json.NewDecoder(rec.Body).Decode(&errResp); err != nil {
					t.Fatalf("Failed to decode error response: %v", err)
				}
				if errResp["error"] == "" {
					t.Error("Expected error field in response")
				}
			}
		})
	}
}

func TestPauseJobHandler(t *testing.T) {
	coord := setupTestCoordinator()
	coord.GetJob().Status = models.JobStatusRunning

	handler := NewHandler(coord)
	req := httptest.NewRequest(http.MethodPut, "/job/pause", nil)
	rec := httptest.NewRecorder()

	handler.PauseJobHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	if coord.GetJob().Status != models.JobStatusPaused {
		t.Errorf("Job status = %s, want paused", coord.GetJob().Status)
	}
}

func TestCancelJobHandler(t *testing.T) {
	coord := setupTestCoordinator()
	coord.GetJob().Status = models.JobStatusRunning

	handler := NewHandler(coord)
	req := httptest.NewRequest(http.MethodPut, "/job/cancel", nil)
	rec := httptest.NewRecorder()

	handler.CancelJobHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	if coord.GetJob().Status != models.JobStatusCancelled {
		t.Errorf("Job status = %s, want cancelled", coord.GetJob().Status)
	}
}

func TestStatusHandler(t *testing.T) {
	coord := setupTestCoordinator()
	job := coord.GetJob()
	job.Status = models.JobStatusRunning
	job.TotalWorkUnits = 100
	job.ProcessedCount = 50
	job.ErrorCount = 2

	handler := NewHandler(coord)
	req := httptest.NewRequest(http.MethodGet, "/job/status", nil)
	rec := httptest.NewRecorder()

	handler.StatusHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp models.JobStatusResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JobID != "test-job-123" {
		t.Errorf("JobID = %s, want test-job-123", resp.JobID)
	}

	if resp.Status != "running" {
		t.Errorf("Status = %s, want running", resp.Status)
	}

	if resp.TotalWorkUnits != 100 {
		t.Errorf("TotalWorkUnits = %d, want 100", resp.TotalWorkUnits)
	}

	if resp.ProcessedWorkUnits != 50 {
		t.Errorf("ProcessedWorkUnits = %d, want 50", resp.ProcessedWorkUnits)
	}

	if resp.CompletionPercent != 50.0 {
		t.Errorf("CompletionPercent = %f, want 50.0", resp.CompletionPercent)
	}
}

func TestHealthHandler(t *testing.T) {
	coord := setupTestCoordinator()
	coord.GetJob().Status = models.JobStatusRunning

	handler := NewHandler(coord)
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	handler.HealthHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp map[string]interface{}
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp["status"] != "ok" {
		t.Errorf("status = %v, want ok", resp["status"])
	}

	if resp["is_processing"] != true {
		t.Errorf("is_processing = %v, want true", resp["is_processing"])
	}
}

func TestGetConfigHandler(t *testing.T) {
	coord := setupTestCoordinator()
	handler := NewHandler(coord)

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	rec := httptest.NewRecorder()

	handler.GetConfigHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d", rec.Code, http.StatusOK)
	}

	var resp models.ConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.JobID != "test-job-123" {
		t.Errorf("JobID = %s, want test-job-123", resp.JobID)
	}

	if resp.BatchSize != 500 {
		t.Errorf("BatchSize = %d, want 500", resp.BatchSize)
	}

	if resp.DistributorType != "stdout" {
		t.Errorf("DistributorType = %s, want stdout", resp.DistributorType)
	}
}

func TestPutConfigHandler(t *testing.T) {
	coord := setupTestCoordinator()
	handler := NewHandler(coord)

	// Get the current config and only update runtime-modifiable fields
	currentCfg := coord.GetConfig()
	updateReq := &config.Config{
		JobID:                    currentCfg.JobID,
		BatchSize:                currentCfg.BatchSize,
		RemainderThreshold:       currentCfg.RemainderThreshold,
		BasePath:                 currentCfg.BasePath,
		MeasuresToRun:            currentCfg.MeasuresToRun,
		DistributorType:          currentCfg.DistributorType,
		CompletionTTL:            currentCfg.CompletionTTL,
		APIPort:                  currentCfg.APIPort,
		ConcurrentFileProcessors: 20, // Changed
		DistributorConfig:        map[string]string{"new": "config"}, // Changed
	}

	body, _ := json.Marshal(updateReq)
	req := httptest.NewRequest(http.MethodPut, "/config", bytes.NewBuffer(body))
	rec := httptest.NewRecorder()

	handler.PutConfigHandler(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("Status code = %d, want %d. Body: %s", rec.Code, http.StatusOK, rec.Body.String())
	}

	var resp models.UpdateConfigResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if resp.Message == "" {
		t.Error("Expected message in response")
	}

	// Verify config was updated
	updatedCfg := coord.GetConfig()
	if updatedCfg.ConcurrentFileProcessors != 20 {
		t.Errorf("ConcurrentFileProcessors = %d, want 20", updatedCfg.ConcurrentFileProcessors)
	}
}

func TestSetupRouter(t *testing.T) {
	coord := setupTestCoordinator()
	router := SetupRouter(coord)

	if router == nil {
		t.Fatal("SetupRouter returned nil")
	}

	// Test that routes are registered
	tests := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/job/start"},
		{http.MethodPut, "/job/pause"},
		{http.MethodPut, "/job/cancel"},
		{http.MethodGet, "/job/status"},
		{http.MethodGet, "/config"},
		{http.MethodPut, "/config"},
		{http.MethodGet, "/health"},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			rec := httptest.NewRecorder()

			router.ServeHTTP(rec, req)

			// Should not return 404
			if rec.Code == http.StatusNotFound {
				t.Errorf("Route %s %s not found", tt.method, tt.path)
			}
		})
	}
}

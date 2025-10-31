package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dqme/job-driver/internal/config"
	"github.com/dqme/job-driver/internal/models"
)

// MockCoordinator simulates coordinator behavior for API contract testing
type MockCoordinator struct {
	job *models.Job
}

func NewMockCoordinator() *MockCoordinator {
	// Initialize with a pending job
	job := &models.Job{
		ID:     "test-job-123",
		Status: models.JobStatusPending,
		Config: &models.JobConfig{
			JobID:              "test-job-123",
			AutoStart:          false,
			BatchSize:          500,
			RemainderThreshold: 0.2,
			BasePath:           "C:\\test\\data",
			DistributorType:    "stdout",
		},
		TotalWorkUnits:  0,
		ProcessedCount:  0,
		ErrorCount:      0,
		Errors:          []string{},
		StartTime:       nil,
		EndTime:         nil,
		CompletionTTL:   10 * time.Minute,
	}
	return &MockCoordinator{job: job}
}

func (mc *MockCoordinator) GetJob() *models.Job {
	return mc.job
}

func (mc *MockCoordinator) StartJob() error {
	return mc.job.TransitionTo(models.JobStatusRunning)
}

func (mc *MockCoordinator) PauseJob() error {
	return mc.job.TransitionTo(models.JobStatusPaused)
}

func (mc *MockCoordinator) CancelJob() error {
	return mc.job.TransitionTo(models.JobStatusCancelled)
}

// Test handlers (will be replaced with actual API handlers later)
func makeStartJobHandler(mc *MockCoordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := mc.StartJob()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":  mc.job.ID,
			"status":  string(mc.job.Status),
			"message": "Job started successfully",
		})
	}
}

func makePauseJobHandler(mc *MockCoordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := mc.PauseJob()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":  mc.job.ID,
			"status":  string(mc.job.Status),
			"message": "Job paused successfully",
		})
	}
}

func makeCancelJobHandler(mc *MockCoordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		err := mc.CancelJob()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":  mc.job.ID,
			"status":  string(mc.job.Status),
			"message": "Job cancelled successfully",
		})
	}
}

func makeStatusHandler(mc *MockCoordinator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		job := mc.GetJob()
		
		var endTime *string
		if job.EndTime != nil {
			t := job.EndTime.Format(time.RFC3339)
			endTime = &t
		}

		var startTime *string
		if job.StartTime != nil {
			t := job.StartTime.Format(time.RFC3339)
			startTime = &t
		}

		var ttlRemaining *string
		if job.Status == models.JobStatusCompleted && job.EndTime != nil {
			remaining := job.TTLRemaining()
			if remaining != nil && *remaining > 0 {
				s := remaining.String()
				ttlRemaining = &s
			}
		}

		errors := job.Errors
		if errors == nil {
			errors = []string{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"job_id":                job.ID,
			"status":                string(job.Status),
			"start_time":            startTime,
			"end_time":              endTime,
			"total_work_units":      job.TotalWorkUnits,
			"distributed_count":     job.ProcessedCount,
			"failed_count":          job.ErrorCount,
			"completion_percentage": job.CompletionPercentage(),
			"ttl_remaining":         ttlRemaining,
			"errors":                errors,
		})
	}
}

// T004: Contract test for POST /job/start
func TestStartJobEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  models.JobStatus
		expectedStatus int
		expectedJobStatus models.JobStatus
		expectError    bool
	}{
		{
			name:           "Start pending job",
			initialStatus:  models.JobStatusPending,
			expectedStatus: http.StatusOK,
			expectedJobStatus: models.JobStatusRunning,
			expectError:    false,
		},
		{
			name:           "Start paused job",
			initialStatus:  models.JobStatusPaused,
			expectedStatus: http.StatusOK,
			expectedJobStatus: models.JobStatusRunning,
			expectError:    false,
		},
		{
			name:           "Start already running job",
			initialStatus:  models.JobStatusRunning,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusRunning,
			expectError:    true,
		},
		{
			name:           "Start completed job",
			initialStatus:  models.JobStatusCompleted,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusCompleted,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMockCoordinator()
			mc.job.Status = tt.initialStatus
			if tt.initialStatus == models.JobStatusRunning {
				now := time.Now()
				mc.job.StartTime = &now
			}
			if tt.initialStatus == models.JobStatusCompleted {
				startTime := time.Now().Add(-10 * time.Minute)
				mc.job.StartTime = &startTime
				now := time.Now()
				mc.job.EndTime = &now
			}

			handler := makeStartJobHandler(mc)
			req := httptest.NewRequest(http.MethodPost, "/job/start", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if mc.job.Status != tt.expectedJobStatus {
				t.Errorf("Expected job status %s, got %s", tt.expectedJobStatus, mc.job.Status)
			}

			var response map[string]interface{}
			json.NewDecoder(w.Body).Decode(&response)

			if tt.expectError {
				if _, ok := response["error"]; !ok {
					t.Error("Expected error in response")
				}
			} else {
				if response["job_id"] != mc.job.ID {
					t.Errorf("Expected job_id %s, got %v", mc.job.ID, response["job_id"])
				}
				if response["status"] != string(tt.expectedJobStatus) {
					t.Errorf("Expected status %s, got %v", tt.expectedJobStatus, response["status"])
				}
			}
		})
	}
}

// T005: Contract test for PUT /job/pause
func TestPauseJobEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  models.JobStatus
		expectedStatus int
		expectedJobStatus models.JobStatus
		expectError    bool
	}{
		{
			name:           "Pause running job",
			initialStatus:  models.JobStatusRunning,
			expectedStatus: http.StatusOK,
			expectedJobStatus: models.JobStatusPaused,
			expectError:    false,
		},
		{
			name:           "Pause pending job",
			initialStatus:  models.JobStatusPending,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusPending,
			expectError:    true,
		},
		{
			name:           "Pause paused job",
			initialStatus:  models.JobStatusPaused,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusPaused,
			expectError:    true,
		},
		{
			name:           "Pause completed job",
			initialStatus:  models.JobStatusCompleted,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusCompleted,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMockCoordinator()
			mc.job.Status = tt.initialStatus
			if tt.initialStatus == models.JobStatusRunning {
				now := time.Now()
				mc.job.StartTime = &now
			}
			if tt.initialStatus == models.JobStatusCompleted {
				startTime := time.Now().Add(-10 * time.Minute)
				mc.job.StartTime = &startTime
				now := time.Now()
				mc.job.EndTime = &now
			}

			handler := makePauseJobHandler(mc)
			req := httptest.NewRequest(http.MethodPut, "/job/pause", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if mc.job.Status != tt.expectedJobStatus {
				t.Errorf("Expected job status %s, got %s", tt.expectedJobStatus, mc.job.Status)
			}
		})
	}
}

// T006: Contract test for PUT /job/cancel
func TestCancelJobEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		initialStatus  models.JobStatus
		expectedStatus int
		expectedJobStatus models.JobStatus
		expectError    bool
	}{
		{
			name:           "Cancel pending job",
			initialStatus:  models.JobStatusPending,
			expectedStatus: http.StatusOK,
			expectedJobStatus: models.JobStatusCancelled,
			expectError:    false,
		},
		{
			name:           "Cancel running job",
			initialStatus:  models.JobStatusRunning,
			expectedStatus: http.StatusOK,
			expectedJobStatus: models.JobStatusCancelled,
			expectError:    false,
		},
		{
			name:           "Cancel paused job",
			initialStatus:  models.JobStatusPaused,
			expectedStatus: http.StatusOK,
			expectedJobStatus: models.JobStatusCancelled,
			expectError:    false,
		},
		{
			name:           "Cancel completed job",
			initialStatus:  models.JobStatusCompleted,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusCompleted,
			expectError:    true,
		},
		{
			name:           "Cancel cancelled job",
			initialStatus:  models.JobStatusCancelled,
			expectedStatus: http.StatusBadRequest,
			expectedJobStatus: models.JobStatusCancelled,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMockCoordinator()
			mc.job.Status = tt.initialStatus
			if tt.initialStatus == models.JobStatusRunning {
				now := time.Now()
				mc.job.StartTime = &now
			}
			if tt.initialStatus == models.JobStatusCompleted || tt.initialStatus == models.JobStatusCancelled {
				startTime := time.Now().Add(-10 * time.Minute)
				mc.job.StartTime = &startTime
				now := time.Now()
				mc.job.EndTime = &now
			}

			handler := makeCancelJobHandler(mc)
			req := httptest.NewRequest(http.MethodPut, "/job/cancel", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if mc.job.Status != tt.expectedJobStatus {
				t.Errorf("Expected job status %s, got %s", tt.expectedJobStatus, mc.job.Status)
			}
		})
	}
}

// T007: Contract test for GET /job/status
func TestStatusEndpoint(t *testing.T) {
	tests := []struct {
		name                  string
		initialStatus         models.JobStatus
		totalWorkUnits        int
		processedCount        int
		failedCount           int
		expectedStatus        int
		expectStartTime       bool
		expectEndTime         bool
		expectTTLRemaining    bool
	}{
		{
			name:                  "Status of pending job",
			initialStatus:         models.JobStatusPending,
			totalWorkUnits:        0,
			processedCount:        0,
			failedCount:           0,
			expectedStatus:        http.StatusOK,
			expectStartTime:       false,
			expectEndTime:         false,
			expectTTLRemaining:    false,
		},
		{
			name:                  "Status of running job",
			initialStatus:         models.JobStatusRunning,
			totalWorkUnits:        1000,
			processedCount:        750,
			failedCount:           5,
			expectedStatus:        http.StatusOK,
			expectStartTime:       true,
			expectEndTime:         false,
			expectTTLRemaining:    false,
		},
		{
			name:                  "Status of completed job",
			initialStatus:         models.JobStatusCompleted,
			totalWorkUnits:        1000,
			processedCount:        995,
			failedCount:           5,
			expectedStatus:        http.StatusOK,
			expectStartTime:       true,
			expectEndTime:         true,
			expectTTLRemaining:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMockCoordinator()
			mc.job.Status = tt.initialStatus
			mc.job.TotalWorkUnits = tt.totalWorkUnits
			mc.job.ProcessedCount = tt.processedCount
			mc.job.ErrorCount = tt.failedCount

			if tt.initialStatus == models.JobStatusRunning {
				now := time.Now()
				mc.job.StartTime = &now
			}

			if tt.initialStatus == models.JobStatusCompleted {
				startTime := time.Now().Add(-10 * time.Minute)
				mc.job.StartTime = &startTime
				now := time.Now()
				mc.job.EndTime = &now
			}

			handler := makeStatusHandler(mc)
			req := httptest.NewRequest(http.MethodGet, "/job/status", nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			json.NewDecoder(w.Body).Decode(&response)

			if response["job_id"] != mc.job.ID {
				t.Errorf("Expected job_id %s, got %v", mc.job.ID, response["job_id"])
			}

			if response["status"] != string(tt.initialStatus) {
				t.Errorf("Expected status %s, got %v", tt.initialStatus, response["status"])
			}

			if tt.expectStartTime {
				if response["start_time"] == nil {
					t.Error("Expected start_time to be set")
				}
			} else {
				if response["start_time"] != nil {
					t.Error("Expected start_time to be nil")
				}
			}

			if tt.expectEndTime {
				if response["end_time"] == nil {
					t.Error("Expected end_time to be set")
				}
			} else {
				if response["end_time"] != nil {
					t.Error("Expected end_time to be nil")
				}
			}

			if tt.expectTTLRemaining {
				if response["ttl_remaining"] == nil {
					t.Error("Expected ttl_remaining to be set")
				}
			}
		})
	}
}

// T008: Contract test for GET /config
func TestGetConfigEndpoint(t *testing.T) {
	mc := NewMockCoordinator()
	
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(mc.job.Config)
	}

	req := httptest.NewRequest(http.MethodGet, "/config", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.JobConfig
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response.JobID != mc.job.Config.JobID {
		t.Errorf("Expected job_id %s, got %s", mc.job.Config.JobID, response.JobID)
	}

	if response.BatchSize != mc.job.Config.BatchSize {
		t.Errorf("Expected batch_size %d, got %d", mc.job.Config.BatchSize, response.BatchSize)
	}
}

// T009: Contract test for PUT /config
func TestPutConfigEndpoint(t *testing.T) {
	tests := []struct {
		name           string
		jobStatus      models.JobStatus
		fieldToUpdate  string
		newValue       interface{}
		expectedStatus int
		expectError    bool
	}{
		{
			name:           "Update runtime-modifiable field (concurrent_file_processors)",
			jobStatus:      models.JobStatusRunning,
			fieldToUpdate:  "concurrent_file_processors",
			newValue:       5,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
		{
			name:           "Update non-modifiable field when running",
			jobStatus:      models.JobStatusRunning,
			fieldToUpdate:  "distributor_type",
			newValue:       "file",
			expectedStatus: http.StatusBadRequest,
			expectError:    true,
		},
		{
			name:           "Update any field when pending",
			jobStatus:      models.JobStatusPending,
			fieldToUpdate:  "batch_size",
			newValue:       1000,
			expectedStatus: http.StatusOK,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := NewMockCoordinator()
			mc.job.Status = tt.jobStatus

			handler := func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
					return
				}

				var updates map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]string{
						"error": "Invalid request body",
					})
					return
				}

				// Check if field is runtime-modifiable
				if mc.job.Status != models.JobStatusPending {
					for field := range updates {
						if !config.IsRuntimeModifiable(field, string(mc.job.Status)) {
							w.WriteHeader(http.StatusBadRequest)
							json.NewEncoder(w).Encode(map[string]string{
								"error": "Field " + field + " is not runtime-modifiable in " + string(mc.job.Status) + " state",
							})
							return
						}
					}
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"message": "Configuration updated successfully",
					"config":  mc.job.Config,
				})
			}

			body := map[string]interface{}{
				tt.fieldToUpdate: tt.newValue,
			}
			bodyBytes, _ := json.Marshal(body)

			req := httptest.NewRequest(http.MethodPut, "/config", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			var response map[string]interface{}
			json.NewDecoder(w.Body).Decode(&response)

			if tt.expectError {
				if _, ok := response["error"]; !ok {
					t.Error("Expected error in response")
				}
			}
		})
	}
}

// T009 additional: Test GET /health endpoint
func TestHealthEndpoint(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "ok",
		})
	}

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	json.NewDecoder(w.Body).Decode(&response)

	if response["status"] != "ok" {
		t.Errorf("Expected status 'ok', got %s", response["status"])
	}
}

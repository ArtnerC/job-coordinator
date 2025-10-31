package models

import "time"

// CreateJobResponse represents the response when creating a new job
type CreateJobResponse struct {
	JobID   string `json:"job_id"`
	Message string `json:"message"`
}

// DistributorStatus represents the status of the distributor
type DistributorStatus struct {
	Type             string `json:"type"`
	IsComplete       bool   `json:"is_complete"`
	PendingCount     int    `json:"pending_count"`
	DistributedCount int    `json:"distributed_count"`
}

// JobStatusResponse represents the job status for GET /job/status
type JobStatusResponse struct {
	JobID               string             `json:"job_id"`
	Status              string             `json:"status"`
	TotalWorkUnits      int                `json:"total_work_units"`
	ProcessedWorkUnits  int                `json:"processed_work_units"`
	ErrorCount          int                `json:"error_count"`
	CompletionPercent   float64            `json:"completion_percent"`
	StartTime           *time.Time         `json:"start_time,omitempty"`
	EndTime             *time.Time         `json:"end_time,omitempty"`
	TTLRemaining        *string            `json:"ttl_remaining,omitempty"`
	Errors              []string           `json:"errors,omitempty"`
	DistributorStatus   *DistributorStatus `json:"distributor_status,omitempty"`
}

// ConfigResponse represents the configuration for GET /config
type ConfigResponse struct {
	JobID                    string   `json:"job_id"`
	BatchSize                int      `json:"batch_size"`
	RemainderThreshold       float64  `json:"remainder_threshold"`
	BasePath                 string   `json:"base_path"`
	ManifestPath             string   `json:"manifest_path,omitempty"`
	MeasuresPath             string   `json:"measures_path"`
	MeasuresToRun            []string `json:"measures_to_run,omitempty"`
	MeasuresManifestPath     string   `json:"measures_manifest_path,omitempty"`
	DistributorType          string   `json:"distributor_type"`
	CompletionTTL            string   `json:"completion_ttl"`
	ConcurrentFileProcessors int      `json:"concurrent_file_processors"`
	Status                   string   `json:"status"`
}

// UpdateConfigResponse represents the response when updating configuration
type UpdateConfigResponse struct {
	Message         string   `json:"message"`
	UpdatedFields   []string `json:"updated_fields"`
	RejectedFields  []string `json:"rejected_fields,omitempty"`
	RejectionReason string   `json:"rejection_reason,omitempty"`
}

// ControlResponse represents the response for control actions (start/pause/cancel)
type ControlResponse struct {
	JobID          string `json:"job_id"`
	PreviousStatus string `json:"previous_status"`
	NewStatus      string `json:"new_status"`
	Message        string `json:"message"`
}

// ErrorResponse represents an error response for all error cases
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
	Details string `json:"details,omitempty"`
}

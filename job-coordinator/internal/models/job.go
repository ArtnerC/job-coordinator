package models

import (
	"errors"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	JobStatusPending   JobStatus = "pending"
	JobStatusRunning   JobStatus = "running"
	JobStatusPaused    JobStatus = "paused"
	JobStatusCompleted JobStatus = "completed"
	JobStatusFailed    JobStatus = "failed"
	JobStatusCancelled JobStatus = "cancelled"
)

// Job represents a complete work distribution request
type Job struct {
	ID               string        `json:"job_id"`
	Status           JobStatus     `json:"status"`
	Config           *JobConfig    `json:"config,omitempty"`
	WorkUnits        []WorkUnit    `json:"work_units,omitempty"`
	StartTime        *time.Time    `json:"start_time,omitempty"`
	EndTime          *time.Time    `json:"end_time,omitempty"`
	CompletionTTL    time.Duration `json:"completion_ttl"`
	TotalWorkUnits   int           `json:"total_work_units"`
	ProcessedCount   int           `json:"processed_count"`
	ErrorCount       int           `json:"error_count"`
	Errors           []string      `json:"errors,omitempty"`
}

// JobConfig represents job configuration
type JobConfig struct {
	JobID                    string            `json:"job_id"`
	AutoStart                bool              `json:"auto_start"`
	BatchSize                int               `json:"batch_size"`
	RemainderThreshold       float64           `json:"remainder_threshold"`
	BasePath                 string            `json:"base_path"`
	ManifestPath             *string           `json:"manifest_path,omitempty"`
	MeasuresPath             string            `json:"measures_path"`
	MeasuresToRun            []string          `json:"measures_to_run,omitempty"`
	MeasuresManifestPath     *string           `json:"measures_manifest_path,omitempty"`
	DistributorType          string            `json:"distributor_type"`
	DistributorConfig        map[string]string `json:"distributor_config,omitempty"`
	CompletionTTL            time.Duration     `json:"completion_ttl"`
	ConcurrentFileProcessors int               `json:"concurrent_file_processors"`
	ScaleLoad                *int              `json:"scale_load,omitempty"`
}

// IsValidTransition checks if a state transition is valid
func IsValidTransition(from, to JobStatus) bool {
	validTransitions := map[JobStatus]map[JobStatus]bool{
		JobStatusPending: {
			JobStatusRunning:   true,
			JobStatusCancelled: true,
		},
		JobStatusRunning: {
			JobStatusPaused:    true,
			JobStatusCompleted: true,
			JobStatusFailed:    true,
			JobStatusCancelled: true,
		},
		JobStatusPaused: {
			JobStatusRunning:   true,
			JobStatusCancelled: true,
		},
		JobStatusCompleted: {}, // Terminal state
		JobStatusFailed:    {}, // Terminal state
		JobStatusCancelled: {}, // Terminal state
	}

	transitions, ok := validTransitions[from]
	if !ok {
		return false
	}
	return transitions[to]
}

// IsTerminalStatus checks if a status is terminal (cannot transition further)
func IsTerminalStatus(status JobStatus) bool {
	return status == JobStatusCompleted ||
		status == JobStatusFailed ||
		status == JobStatusCancelled
}

// TransitionTo attempts to transition the job to a new status
func (j *Job) TransitionTo(newStatus JobStatus) error {
	if !IsValidTransition(j.Status, newStatus) {
		return errors.New("invalid state transition from " + string(j.Status) + " to " + string(newStatus))
	}

	j.Status = newStatus

	// Set timestamps based on transition
	now := time.Now()
	switch newStatus {
	case JobStatusRunning:
		if j.StartTime == nil {
			j.StartTime = &now
		}
	case JobStatusCompleted, JobStatusFailed, JobStatusCancelled:
		j.EndTime = &now
	}

	return nil
}

// CompletionPercentage calculates the completion percentage of the job
func (j *Job) CompletionPercentage() float64 {
	if j.TotalWorkUnits == 0 {
		return 0.0
	}
	return (float64(j.ProcessedCount) / float64(j.TotalWorkUnits)) * 100.0
}

// TTLRemaining calculates the remaining TTL after job completion
func (j *Job) TTLRemaining() *time.Duration {
	if !IsTerminalStatus(j.Status) || j.EndTime == nil {
		return nil
	}

	elapsed := time.Since(*j.EndTime)
	remaining := j.CompletionTTL - elapsed
	
	if remaining < 0 {
		zero := time.Duration(0)
		return &zero
	}
	
	return &remaining
}

// Validate validates the job fields
func (j *Job) Validate() error {
	if j.ID == "" {
		return errors.New("job ID must be non-empty")
	}

	if j.TotalWorkUnits < 0 {
		return errors.New("total work units must be non-negative")
	}

	// StartTime required once status is Running or later
	if j.Status == JobStatusRunning || j.Status == JobStatusPaused || IsTerminalStatus(j.Status) {
		if j.StartTime == nil {
			return errors.New("start time required for status " + string(j.Status))
		}
	}

	// EndTime required for terminal states
	if IsTerminalStatus(j.Status) && j.EndTime == nil {
		return errors.New("end time required for terminal status " + string(j.Status))
	}

	return nil
}

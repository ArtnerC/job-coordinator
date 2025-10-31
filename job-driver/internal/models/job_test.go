package models

import (
	"testing"
)

func TestJobStateTransitions(t *testing.T) {
	tests := []struct {
		name        string
		fromStatus  JobStatus
		toStatus    JobStatus
		wantValid   bool
		description string
	}{
		{
			name:        "Pending to Running (valid)",
			fromStatus:  JobStatusPending,
			toStatus:    JobStatusRunning,
			wantValid:   true,
			description: "Starting a pending job",
		},
		{
			name:        "Running to Paused (valid)",
			fromStatus:  JobStatusRunning,
			toStatus:    JobStatusPaused,
			wantValid:   true,
			description: "Pausing a running job",
		},
		{
			name:        "Paused to Running (valid)",
			fromStatus:  JobStatusPaused,
			toStatus:    JobStatusRunning,
			wantValid:   true,
			description: "Resuming a paused job",
		},
		{
			name:        "Running to Completed (valid)",
			fromStatus:  JobStatusRunning,
			toStatus:    JobStatusCompleted,
			wantValid:   true,
			description: "Job completes successfully",
		},
		{
			name:        "Running to Failed (valid)",
			fromStatus:  JobStatusRunning,
			toStatus:    JobStatusFailed,
			wantValid:   true,
			description: "Job fails during execution",
		},
		{
			name:        "Running to Cancelled (valid)",
			fromStatus:  JobStatusRunning,
			toStatus:    JobStatusCancelled,
			wantValid:   true,
			description: "Cancelling a running job",
		},
		{
			name:        "Pending to Cancelled (valid)",
			fromStatus:  JobStatusPending,
			toStatus:    JobStatusCancelled,
			wantValid:   true,
			description: "Cancelling a pending job",
		},
		{
			name:        "Paused to Cancelled (valid)",
			fromStatus:  JobStatusPaused,
			toStatus:    JobStatusCancelled,
			wantValid:   true,
			description: "Cancelling a paused job",
		},
		{
			name:        "Completed to Running (invalid)",
			fromStatus:  JobStatusCompleted,
			toStatus:    JobStatusRunning,
			wantValid:   false,
			description: "Cannot restart a completed job",
		},
		{
			name:        "Failed to Running (invalid)",
			fromStatus:  JobStatusFailed,
			toStatus:    JobStatusRunning,
			wantValid:   false,
			description: "Cannot restart a failed job",
		},
		{
			name:        "Cancelled to Running (invalid)",
			fromStatus:  JobStatusCancelled,
			toStatus:    JobStatusRunning,
			wantValid:   false,
			description: "Cannot restart a cancelled job",
		},
		{
			name:        "Completed to Paused (invalid)",
			fromStatus:  JobStatusCompleted,
			toStatus:    JobStatusPaused,
			wantValid:   false,
			description: "Cannot pause a completed job",
		},
		{
			name:        "Pending to Paused (invalid)",
			fromStatus:  JobStatusPending,
			toStatus:    JobStatusPaused,
			wantValid:   false,
			description: "Cannot pause a job that hasn't started",
		},
		{
			name:        "Completed to Cancelled (invalid)",
			fromStatus:  JobStatusCompleted,
			toStatus:    JobStatusCancelled,
			wantValid:   false,
			description: "Cannot cancel a completed job",
		},
		{
			name:        "Pending to Completed (invalid)",
			fromStatus:  JobStatusPending,
			toStatus:    JobStatusCompleted,
			wantValid:   false,
			description: "Cannot complete a job that hasn't run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid := IsValidTransition(tt.fromStatus, tt.toStatus)
			if valid != tt.wantValid {
				t.Errorf("IsValidTransition(%s -> %s) = %v, want %v (%s)",
					tt.fromStatus, tt.toStatus, valid, tt.wantValid, tt.description)
			}
		})
	}
}

func TestIsTerminalStatus(t *testing.T) {
	tests := []struct {
		status     JobStatus
		isTerminal bool
	}{
		{JobStatusPending, false},
		{JobStatusRunning, false},
		{JobStatusPaused, false},
		{JobStatusCompleted, true},
		{JobStatusFailed, true},
		{JobStatusCancelled, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := IsTerminalStatus(tt.status)
			if got != tt.isTerminal {
				t.Errorf("IsTerminalStatus(%s) = %v, want %v", tt.status, got, tt.isTerminal)
			}
		})
	}
}

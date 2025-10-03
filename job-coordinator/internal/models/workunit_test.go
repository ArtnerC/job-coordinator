package models

import (
	"testing"
	"time"
)

// WorkUnit represents a single discrete unit of work
type WorkUnit struct {
	ID           string
	JobID        string
	FilePath     string
	StartLine    *int
	EndLine      *int
	TotalLines   *int
	Measures     []string
	MeasuresPath *string
	BasePath     string
	CreatedAt    time.Time
	Status       WorkUnitStatus
	Error        *string
}

// WorkUnitStatus represents the current state of a work unit
type WorkUnitStatus string

const (
	WorkUnitStatusPending      WorkUnitStatus = "pending"
	WorkUnitStatusDistributing WorkUnitStatus = "distributing"
	WorkUnitStatusDistributed  WorkUnitStatus = "distributed"
	WorkUnitStatusFailed       WorkUnitStatus = "failed"
)

func TestWorkUnitValidation(t *testing.T) {
	tests := []struct {
		name    string
		wu      WorkUnit
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid work unit with measures array",
			wu: WorkUnit{
				ID:         "wu-001",
				JobID:      "job-001",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  intPtr(0),
				EndLine:    intPtr(499),
				TotalLines: intPtr(500),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "valid work unit with measures_path",
			wu: WorkUnit{
				ID:           "wu-002",
				JobID:        "job-001",
				FilePath:     "patients/bundle-001.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				MeasuresPath: stringPtr("/data/measures"),
				BasePath:     "/data/bundles",
				CreatedAt:    time.Now(),
			},
			wantErr: false,
		},
		{
			name: "both measures and measures_path (invalid)",
			wu: WorkUnit{
				ID:           "wu-003",
				JobID:        "job-001",
				FilePath:     "patients/bundle-001.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				Measures:     []string{"/data/measures/cms-125.json"},
				MeasuresPath: stringPtr("/data/measures"),
				BasePath:     "/data/bundles",
				CreatedAt:    time.Now(),
			},
			wantErr: true,
			errMsg:  "exactly one of Measures or MeasuresPath must be set",
		},
		{
			name: "neither measures nor measures_path (invalid)",
			wu: WorkUnit{
				ID:         "wu-004",
				JobID:      "job-001",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  intPtr(0),
				EndLine:    intPtr(499),
				TotalLines: intPtr(500),
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "exactly one of Measures or MeasuresPath must be set",
		},
		{
			name: "start_line without end_line (invalid)",
			wu: WorkUnit{
				ID:         "wu-005",
				JobID:      "job-001",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  intPtr(0),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "all line fields (StartLine, EndLine, TotalLines) must be either all present or all nil",
		},
		{
			name: "end_line < start_line (invalid)",
			wu: WorkUnit{
				ID:         "wu-006",
				JobID:      "job-001",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  intPtr(500),
				EndLine:    intPtr(400),
				TotalLines: intPtr(100),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "EndLine must be >= StartLine",
		},
		{
			name: "total_lines mismatch (invalid)",
			wu: WorkUnit{
				ID:         "wu-007",
				JobID:      "job-001",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  intPtr(0),
				EndLine:    intPtr(499),
				TotalLines: intPtr(600),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "TotalLines must equal (EndLine - StartLine + 1)",
		},
		{
			name: "all line fields nil (whole file mode - valid)",
			wu: WorkUnit{
				ID:         "wu-008",
				JobID:      "job-001",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  nil,
				EndLine:    nil,
				TotalLines: nil,
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: false,
		},
		{
			name: "empty job ID (invalid)",
			wu: WorkUnit{
				ID:         "wu-009",
				JobID:      "",
				FilePath:   "patients/bundle-001.ndjson",
				StartLine:  intPtr(0),
				EndLine:    intPtr(499),
				TotalLines: intPtr(500),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "JobID must be non-empty",
		},
		{
			name: "empty file path (invalid)",
			wu: WorkUnit{
				ID:         "wu-010",
				JobID:      "job-001",
				FilePath:   "",
				StartLine:  intPtr(0),
				EndLine:    intPtr(499),
				TotalLines: intPtr(500),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "FilePath must be non-empty",
		},
		{
			name: "absolute file path (invalid - must be relative)",
			wu: WorkUnit{
				ID:         "wu-011",
				JobID:      "job-001",
				FilePath:   "/absolute/path/bundle-001.ndjson",
				StartLine:  intPtr(0),
				EndLine:    intPtr(499),
				TotalLines: intPtr(500),
				Measures:   []string{"/data/measures/cms-125.json"},
				BasePath:   "/data/bundles",
				CreatedAt:  time.Now(),
			},
			wantErr: true,
			errMsg:  "FilePath must be relative (no leading /)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateWorkUnit(&tt.wu)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateWorkUnit() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateWorkUnit() error message = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// Helper functions
func intPtr(n int) *int {
	return &n
}

func stringPtr(s string) *string {
	return &s
}

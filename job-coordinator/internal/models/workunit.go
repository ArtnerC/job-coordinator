package models

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"
)

// WorkUnitStatus represents the current state of a work unit
type WorkUnitStatus string

const (
	WorkUnitStatusPending      WorkUnitStatus = "pending"
	WorkUnitStatusDistributing WorkUnitStatus = "distributing"
	WorkUnitStatusDistributed  WorkUnitStatus = "distributed"
	WorkUnitStatusFailed       WorkUnitStatus = "failed"
)

// WorkUnit represents a single discrete unit of work
type WorkUnit struct {
	ID           string          `json:"work_unit_id"`
	JobID        string          `json:"job_id"`
	FilePath     string          `json:"file_path"`
	StartLine    *int            `json:"start_line,omitempty"`
	EndLine      *int            `json:"end_line,omitempty"`
	TotalLines   *int            `json:"total_lines,omitempty"`
	Measures     []string        `json:"measures,omitempty"`
	MeasuresPath *string         `json:"measures_path,omitempty"`
	BasePath     string          `json:"base_path"`
	CreatedAt    time.Time       `json:"created_at"`
	Status       WorkUnitStatus  `json:"status,omitempty"`
	Error        *string         `json:"error,omitempty"`
}

// ValidateWorkUnit validates a work unit's fields
func ValidateWorkUnit(wu *WorkUnit) error {
	// ID validations
	if wu.ID == "" {
		return errors.New("ID must be non-empty")
	}
	if wu.JobID == "" {
		return errors.New("JobID must be non-empty")
	}

	// FilePath validations
	if wu.FilePath == "" {
		return errors.New("FilePath must be non-empty")
	}
	if filepath.IsAbs(wu.FilePath) || strings.HasPrefix(wu.FilePath, "/") {
		return errors.New("FilePath must be relative (no leading /)")
	}

	// Line fields validation - all or none
	hasStartLine := wu.StartLine != nil
	hasEndLine := wu.EndLine != nil
	hasTotalLines := wu.TotalLines != nil

	if hasStartLine != hasEndLine || hasStartLine != hasTotalLines {
		return errors.New("all line fields (StartLine, EndLine, TotalLines) must be either all present or all nil")
	}

	// If line fields are present, validate values
	if hasStartLine {
		if *wu.StartLine < 0 {
			return errors.New("StartLine must be >= 0")
		}
		if *wu.EndLine < *wu.StartLine {
			return errors.New("EndLine must be >= StartLine")
		}
		expectedTotal := *wu.EndLine - *wu.StartLine + 1
		if *wu.TotalLines != expectedTotal {
			return errors.New("TotalLines must equal (EndLine - StartLine + 1)")
		}
	}

	// Measures mutual exclusion validation
	hasMeasures := len(wu.Measures) > 0 || wu.Measures != nil
	hasMeasuresPath := wu.MeasuresPath != nil

	if hasMeasures && hasMeasuresPath {
		return errors.New("exactly one of Measures or MeasuresPath must be set")
	}
	if !hasMeasures && !hasMeasuresPath {
		return errors.New("exactly one of Measures or MeasuresPath must be set")
	}

	return nil
}

// ToJSON serializes the work unit to JSON for distribution
func (wu *WorkUnit) ToJSON() ([]byte, error) {
	return json.Marshal(wu)
}

// FromJSON deserializes a work unit from JSON
func FromJSON(data []byte) (*WorkUnit, error) {
	var wu WorkUnit
	if err := json.Unmarshal(data, &wu); err != nil {
		return nil, err
	}
	return &wu, nil
}

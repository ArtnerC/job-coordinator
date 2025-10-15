package models

import (
	"encoding/json"
	"errors"
	"fmt"
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

// FileSpec represents a file or file segment within a work unit
type FileSpec struct {
	Path      string `json:"path"`                   // Relative path to the file
	StartLine *int   `json:"start_line,omitempty"`   // Starting line (0-indexed), omitted for whole file
	EndLine   *int   `json:"end_line,omitempty"`     // Ending line (inclusive), omitted for whole file
}

// WorkUnit represents a single discrete unit of work
type WorkUnit struct {
	ID           string          `json:"work_unit_id"`
	JobID        string          `json:"job_id"`
	Files        []FileSpec      `json:"files"`                    // One or more files/segments to process
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

	// Files validation
	if len(wu.Files) == 0 {
		return errors.New("Files must contain at least one file")
	}
	
	// Validate each FileSpec
	for i, file := range wu.Files {
		if file.Path == "" {
			return fmt.Errorf("Files[%d].Path must be non-empty", i)
		}
		if filepath.IsAbs(file.Path) || strings.HasPrefix(file.Path, "/") {
			return fmt.Errorf("Files[%d].Path must be relative (no leading /)", i)
		}
		
		// Line fields validation - both or neither
		hasStartLine := file.StartLine != nil
		hasEndLine := file.EndLine != nil
		
		if hasStartLine != hasEndLine {
			return fmt.Errorf("Files[%d]: both StartLine and EndLine must be set, or neither", i)
		}
		
		// If line fields are present, validate values
		if hasStartLine {
			if *file.StartLine < 0 {
				return fmt.Errorf("Files[%d].StartLine must be >= 0", i)
			}
			if *file.EndLine < *file.StartLine {
				return fmt.Errorf("Files[%d].EndLine must be >= StartLine", i)
			}
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

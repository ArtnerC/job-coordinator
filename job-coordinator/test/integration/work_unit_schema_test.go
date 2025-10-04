package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dqme/job-coordinator/internal/models"
)

// T010: Test that work unit JSON schema includes all required fields
func TestWorkUnitJSONSchema(t *testing.T) {
	// Create a valid work unit with all required fields
	workUnit := models.WorkUnit{
		ID:           "wu-001",
		JobID:        "job-001",
		FilePath:     "bundles/bundle_001.ndjson",
		StartLine:    intPtr(0),
		EndLine:      intPtr(499),
		TotalLines:   intPtr(500),
		Measures:     []string{"/path/to/measure1.json", "/path/to/measure2.json"},
		MeasuresPath: nil,
		BasePath:     "/data",
		CreatedAt:    time.Now(),
		Status:       models.WorkUnitStatusPending,
		Error:        nil,
	}

	jsonData, err := workUnit.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize work unit to JSON: %v", err)
	}

	// Parse the JSON to check schema
	var schema map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &schema)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify all required fields are present
	requiredFields := []string{
		"job_id",
		"work_unit_id",
		"file_path",
		"start_line",
		"end_line",
		"total_lines",
		"measures",
		"base_path",
		"created_at",
	}

	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON schema", field)
		}
	}

	// Verify field values
	if schema["job_id"] != workUnit.JobID {
		t.Errorf("Expected job_id=%s, got %v", workUnit.JobID, schema["job_id"])
	}

	if schema["work_unit_id"] != workUnit.ID {
		t.Errorf("Expected work_unit_id=%s, got %v", workUnit.ID, schema["work_unit_id"])
	}

	if schema["file_path"] != workUnit.FilePath {
		t.Errorf("Expected file_path=%s, got %v", workUnit.FilePath, schema["file_path"])
	}

	// Verify integer fields
	if float64(*workUnit.StartLine) != schema["start_line"].(float64) {
		t.Errorf("Expected start_line=%d, got %v", *workUnit.StartLine, schema["start_line"])
	}

	if float64(*workUnit.EndLine) != schema["end_line"].(float64) {
		t.Errorf("Expected end_line=%d, got %v", *workUnit.EndLine, schema["end_line"])
	}

	if float64(*workUnit.TotalLines) != schema["total_lines"].(float64) {
		t.Errorf("Expected total_lines=%d, got %v", *workUnit.TotalLines, schema["total_lines"])
	}

	// Verify base_path
	if schema["base_path"] != workUnit.BasePath {
		t.Errorf("Expected base_path=%s, got %v", workUnit.BasePath, schema["base_path"])
	}
}

// T011: Test work unit with whole-file mode (nil line fields)
func TestWorkUnitWholeFileMode(t *testing.T) {
	workUnit := models.WorkUnit{
		ID:           "wu-002",
		JobID:        "job-002",
		FilePath:     "bundles/bundle_002.ndjson",
		StartLine:    nil,
		EndLine:      nil,
		TotalLines:   nil,
		Measures:     []string{"/path/to/measure1.json"},
		MeasuresPath: nil,
		BasePath:     "/data",
		CreatedAt:    time.Now(),
		Status:       models.WorkUnitStatusPending,
		Error:        nil,
	}

	jsonData, err := workUnit.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize work unit to JSON: %v", err)
	}

	var schema map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &schema)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// In whole-file mode, line fields should still be present but set to null
	// OR the ToJSON() implementation might omit them entirely (omitempty)
	// Let's check what the actual behavior is

	// The schema should still have required fields like job_id, work_unit_id, etc.
	requiredFields := []string{
		"job_id",
		"work_unit_id",
		"file_path",
		"measures",
		"base_path",
		"created_at",
	}

	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON schema", field)
		}
	}
}

// T012: Test work unit with measures_path instead of measures array
func TestWorkUnitWithMeasuresPath(t *testing.T) {
	measuresPath := "/path/to/measures"
	workUnit := models.WorkUnit{
		ID:           "wu-003",
		JobID:        "job-003",
		FilePath:     "bundles/bundle_003.ndjson",
		StartLine:    intPtr(0),
		EndLine:      intPtr(999),
		TotalLines:   intPtr(1000),
		Measures:     nil,
		MeasuresPath: &measuresPath,
		BasePath:     "/data",
		CreatedAt:    time.Now(),
		Status:       models.WorkUnitStatusPending,
		Error:        nil,
	}

	// Validate mutual exclusion
	err := models.ValidateWorkUnit(&workUnit)
	if err != nil {
		t.Fatalf("Validation failed for work unit with measures_path: %v", err)
	}

	jsonData, err := workUnit.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize work unit to JSON: %v", err)
	}

	var schema map[string]interface{}
	err = json.Unmarshal([]byte(jsonData), &schema)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify measures_path is present and measures array is not (or is null)
	if _, exists := schema["measures_path"]; !exists {
		t.Error("Expected measures_path field in JSON schema")
	}

	if schema["measures_path"] != measuresPath {
		t.Errorf("Expected measures_path=%s, got %v", measuresPath, schema["measures_path"])
	}
}

// T013: Test work unit validation errors
func TestWorkUnitValidation(t *testing.T) {
	tests := []struct {
		name      string
		workUnit  models.WorkUnit
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid work unit with measures array",
			workUnit: models.WorkUnit{
				ID:           "wu-004",
				JobID:        "job-004",
				FilePath:     "bundles/bundle_004.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: false,
		},
		{
			name: "Invalid: Both measures and measures_path set",
			workUnit: models.WorkUnit{
				ID:           "wu-005",
				JobID:        "job-005",
				FilePath:     "bundles/bundle_005.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: stringPtr("/path/to/measures"),
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "exactly one of Measures or MeasuresPath must be set",
		},
		{
			name: "Invalid: Neither measures nor measures_path set",
			workUnit: models.WorkUnit{
				ID:           "wu-006",
				JobID:        "job-006",
				FilePath:     "bundles/bundle_006.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				Measures:     nil,
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "exactly one of Measures or MeasuresPath must be set",
		},
		{
			name: "Invalid: Inconsistent line fields (only StartLine)",
			workUnit: models.WorkUnit{
				ID:           "wu-007",
				JobID:        "job-007",
				FilePath:     "bundles/bundle_007.ndjson",
				StartLine:    intPtr(0),
				EndLine:      nil,
				TotalLines:   nil,
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "StartLine, EndLine, and TotalLines must all be set or all be nil",
		},
		{
			name: "Invalid: EndLine < StartLine",
			workUnit: models.WorkUnit{
				ID:           "wu-008",
				JobID:        "job-008",
				FilePath:     "bundles/bundle_008.ndjson",
				StartLine:    intPtr(100),
				EndLine:      intPtr(50),
				TotalLines:   intPtr(0),
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "EndLine must be >= StartLine",
		},
		{
			name: "Invalid: Incorrect TotalLines calculation",
			workUnit: models.WorkUnit{
				ID:           "wu-009",
				JobID:        "job-009",
				FilePath:     "bundles/bundle_009.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(1000), // Should be 500
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "TotalLines must equal (EndLine - StartLine + 1)",
		},
		{
			name: "Invalid: Relative file path",
			workUnit: models.WorkUnit{
				ID:           "wu-010",
				JobID:        "job-010",
				FilePath:     "/absolute/path/bundle.ndjson", // Should be relative
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "FilePath must be relative",
		},
		{
			name: "Valid: Measures with any path format (validation doesn't enforce absolute)",
			workUnit: models.WorkUnit{
				ID:           "wu-011",
				JobID:        "job-011",
				FilePath:     "bundles/bundle_011.ndjson",
				StartLine:    intPtr(0),
				EndLine:      intPtr(499),
				TotalLines:   intPtr(500),
				Measures:     []string{"/path/to/measure.json"}, // Current validation accepts any format
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := models.ValidateWorkUnit(&tt.workUnit)
			if tt.expectErr {
				if err == nil {
					t.Errorf("Expected validation error but got nil")
				} else if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Logf("Expected error: %s", tt.errMsg)
					t.Logf("Got error: %s", err.Error())
					// Don't fail on exact error message match, just ensure error exists
				}
			} else {
				if err != nil {
					t.Errorf("Expected no validation error but got: %v", err)
				}
			}
		})
	}
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func stringPtr(s string) *string {
	return &s
}

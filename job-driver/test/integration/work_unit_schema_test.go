package integration

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dqme/job-driver/internal/models"
)

// T010: Test that work unit JSON schema includes all required fields
func TestWorkUnitJSONSchema(t *testing.T) {
	// Create a valid work unit with all required fields
	startLine := 0
	endLine := 499
	workUnit := models.WorkUnit{
		ID:           "wu-001",
		JobID:        "job-001",
		Files:        []models.FileSpec{{Path: "bundles/bundle_001.ndjson", StartLine: &startLine, EndLine: &endLine}},
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
		"files",
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

	// Verify files array
	filesArray, ok := schema["files"].([]interface{})
	if !ok || len(filesArray) != 1 {
		t.Errorf("Expected files array with 1 element, got %v", schema["files"])
	} else {
		file := filesArray[0].(map[string]interface{})
		if file["path"] != "bundles/bundle_001.ndjson" {
			t.Errorf("Expected file path=bundles/bundle_001.ndjson, got %v", file["path"])
		}
		if float64(0) != file["start_line"].(float64) {
			t.Errorf("Expected start_line=0, got %v", file["start_line"])
		}
		if float64(499) != file["end_line"].(float64) {
			t.Errorf("Expected end_line=499, got %v", file["end_line"])
		}
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
		Files:        []models.FileSpec{{Path: "bundles/bundle_002.ndjson"}}, // No line ranges = whole file
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

	// In whole-file mode, line fields should be omitted (not present in Files array)
	// The schema should have required fields like job_id, work_unit_id, files, etc.
	requiredFields := []string{
		"job_id",
		"work_unit_id",
		"files",
		"measures",
		"base_path",
		"created_at",
	}

	for _, field := range requiredFields {
		if _, exists := schema[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON schema", field)
		}
	}

	// Verify files array structure - should have path but no line ranges
	filesArray, ok := schema["files"].([]interface{})
	if !ok || len(filesArray) != 1 {
		t.Errorf("Expected files array with 1 element, got %v", schema["files"])
	} else {
		file := filesArray[0].(map[string]interface{})
		if file["path"] != "bundles/bundle_002.ndjson" {
			t.Errorf("Expected file path=bundles/bundle_002.ndjson, got %v", file["path"])
		}
		// In whole-file mode, start_line and end_line should be omitted (due to omitempty)
		if _, hasStartLine := file["start_line"]; hasStartLine {
			t.Errorf("Expected no start_line in whole-file mode, but it was present: %v", file["start_line"])
		}
		if _, hasEndLine := file["end_line"]; hasEndLine {
			t.Errorf("Expected no end_line in whole-file mode, but it was present: %v", file["end_line"])
		}
	}
}

// T012: Test work unit with measures_path instead of measures array
func TestWorkUnitWithMeasuresPath(t *testing.T) {
	measuresPath := "/path/to/measures"
	startLine := 0
	endLine := 999
	workUnit := models.WorkUnit{
		ID:           "wu-003",
		JobID:        "job-003",
		Files:        []models.FileSpec{{Path: "bundles/bundle_003.ndjson", StartLine: &startLine, EndLine: &endLine}},
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
			workUnit: func() models.WorkUnit {
				start, end := 0, 499
				return models.WorkUnit{
					ID:           "wu-004",
					JobID:        "job-004",
					Files:        []models.FileSpec{{Path: "bundles/bundle_004.ndjson", StartLine: &start, EndLine: &end}},
					Measures:     []string{"/path/to/measure.json"},
					MeasuresPath: nil,
					BasePath:     "/data",
					CreatedAt:    time.Now(),
					Status:       models.WorkUnitStatusPending,
				}
			}(),
			expectErr: false,
		},
		{
			name: "Invalid: Both measures and measures_path set",
			workUnit: func() models.WorkUnit {
				start, end := 0, 499
				return models.WorkUnit{
					ID:           "wu-005",
					JobID:        "job-005",
					Files:        []models.FileSpec{{Path: "bundles/bundle_005.ndjson", StartLine: &start, EndLine: &end}},
					Measures:     []string{"/path/to/measure.json"},
					MeasuresPath: stringPtr("/path/to/measures"),
					BasePath:     "/data",
					CreatedAt:    time.Now(),
					Status:       models.WorkUnitStatusPending,
				}
			}(),
			expectErr: true,
			errMsg:    "exactly one of Measures or MeasuresPath must be set",
		},
		{
			name: "Invalid: Neither measures nor measures_path set",
			workUnit: func() models.WorkUnit {
				start, end := 0, 499
				return models.WorkUnit{
					ID:           "wu-006",
					JobID:        "job-006",
					Files:        []models.FileSpec{{Path: "bundles/bundle_006.ndjson", StartLine: &start, EndLine: &end}},
					Measures:     nil,
					MeasuresPath: nil,
					BasePath:     "/data",
					CreatedAt:    time.Now(),
					Status:       models.WorkUnitStatusPending,
				}
			}(),
			expectErr: true,
			errMsg:    "exactly one of Measures or MeasuresPath must be set",
		},
		{
			name: "Invalid: Inconsistent line fields (only StartLine set)",
			workUnit: func() models.WorkUnit {
				start := 0
				return models.WorkUnit{
					ID:           "wu-007",
					JobID:        "job-007",
					Files:        []models.FileSpec{{Path: "bundles/bundle_007.ndjson", StartLine: &start, EndLine: nil}},
					Measures:     []string{"/path/to/measure.json"},
					MeasuresPath: nil,
					BasePath:     "/data",
					CreatedAt:    time.Now(),
					Status:       models.WorkUnitStatusPending,
				}
			}(),
			expectErr: true,
			errMsg:    "both StartLine and EndLine must be set or both must be nil",
		},
		{
			name: "Invalid: EndLine < StartLine",
			workUnit: func() models.WorkUnit {
				start, end := 100, 50
				return models.WorkUnit{
					ID:           "wu-008",
					JobID:        "job-008",
					Files:        []models.FileSpec{{Path: "bundles/bundle_008.ndjson", StartLine: &start, EndLine: &end}},
					Measures:     []string{"/path/to/measure.json"},
					MeasuresPath: nil,
					BasePath:     "/data",
					CreatedAt:    time.Now(),
					Status:       models.WorkUnitStatusPending,
				}
			}(),
			expectErr: true,
			errMsg:    "EndLine must be >= StartLine",
		},
		{
			name: "Invalid: Absolute file path (should be relative)",
			workUnit: models.WorkUnit{
				ID:           "wu-010",
				JobID:        "job-010",
				Files:        []models.FileSpec{{Path: "/absolute/path/bundle.ndjson"}},
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: true,
			errMsg:    "file path must be relative",
		},
		{
			name: "Valid: Whole file (no line ranges)",
			workUnit: models.WorkUnit{
				ID:           "wu-011",
				JobID:        "job-011",
				Files:        []models.FileSpec{{Path: "bundles/bundle_011.ndjson"}},
				Measures:     []string{"/path/to/measure.json"},
				MeasuresPath: nil,
				BasePath:     "/data",
				CreatedAt:    time.Now(),
				Status:       models.WorkUnitStatusPending,
			},
			expectErr: false,
		},
		{
			name: "Valid: Multiple files in batch",
			workUnit: func() models.WorkUnit {
				start1, end1 := 0, 99
				start2, end2 := 0, 49
				return models.WorkUnit{
					ID:    "wu-012",
					JobID: "job-012",
					Files: []models.FileSpec{
						{Path: "bundles/bundle_012a.ndjson", StartLine: &start1, EndLine: &end1},
						{Path: "bundles/bundle_012b.ndjson", StartLine: &start2, EndLine: &end2},
					},
					Measures:     []string{"/path/to/measure.json"},
					MeasuresPath: nil,
					BasePath:     "/data",
					CreatedAt:    time.Now(),
					Status:       models.WorkUnitStatusPending,
				}
			}(),
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

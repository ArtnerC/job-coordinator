package processor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestCountLines(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantLines int
		wantErr   bool
	}{
		{
			name:      "empty file",
			content:   "",
			wantLines: 0,
			wantErr:   false,
		},
		{
			name:      "single line with newline",
			content:   "line1\n",
			wantLines: 1,
			wantErr:   false,
		},
		{
			name:      "single line without trailing newline",
			content:   "line1",
			wantLines: 1,
			wantErr:   false,
		},
		{
			name:      "multiple lines with trailing newline",
			content:   "line1\nline2\nline3\n",
			wantLines: 3,
			wantErr:   false,
		},
		{
			name:      "multiple lines without trailing newline",
			content:   "line1\nline2\nline3",
			wantLines: 3,
			wantErr:   false,
		},
		{
			name:      "lines with empty lines",
			content:   "line1\n\nline3\n",
			wantLines: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test-*.ndjson")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Count lines
			got, err := CountLines(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("CountLines() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantLines {
				t.Errorf("CountLines() = %d, want %d", got, tt.wantLines)
			}
		})
	}
}

func TestCountLines_FileNotFound(t *testing.T) {
	_, err := CountLines("/nonexistent/file.ndjson")
	if err == nil {
		t.Error("CountLines() expected error for non-existent file, got nil")
	}
}

func TestCountLines_Performance(t *testing.T) {
	// Create a file with 1 million lines
	tmpFile, err := os.CreateTemp("", "perf-test-*.ndjson")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write 1 million lines
	line := strings.Repeat("x", 100) + "\n"
	for i := 0; i < 1000000; i++ {
		if _, err := tmpFile.WriteString(line); err != nil {
			t.Fatalf("Failed to write to temp file: %v", err)
		}
	}
	tmpFile.Close()

	// Measure time to count lines
	start := time.Now()
	count, err := CountLines(tmpFile.Name())
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("CountLines() error = %v", err)
	}

	if count != 1000000 {
		t.Errorf("CountLines() = %d, want 1000000", count)
	}

	// Should complete in less than 1 second
	if elapsed > time.Second {
		t.Errorf("CountLines() took %v, expected < 1s", elapsed)
	}

	t.Logf("Counted 1M lines in %v", elapsed)
}

func TestCountLines_ActualTestData(t *testing.T) {
	// Test with actual testdata file
	testFile := filepath.Join("..", "..", "testdata", "bundles", "sample-001.ndjson")
	
	// Skip if file doesn't exist (CI environment)
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Skip("Test data file not found")
	}

	count, err := CountLines(testFile)
	if err != nil {
		t.Fatalf("CountLines() error = %v", err)
	}

	// sample-001.ndjson should have 3 lines
	if count != 3 {
		t.Errorf("CountLines() = %d, want 3", count)
	}
}

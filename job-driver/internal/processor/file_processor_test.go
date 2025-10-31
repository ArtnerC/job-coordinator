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
	// Create realistic FHIR bundle line (typical patient bundle is 2-5KB)
	// Simulate a compressed JSON line with patient, encounters, observations, etc.
	fhirLine := `{"resourceType":"Bundle","type":"collection","entry":[{"resource":{"resourceType":"Patient","id":"patient-` + 
		strings.Repeat("x", 100) + `","name":[{"family":"TestPatient","given":["John","Michael"]}],"birthDate":"1980-01-01","gender":"male","address":[{"line":["123 Main St"],"city":"Springfield","state":"IL","postalCode":"62701"}]}},` +
		`{"resource":{"resourceType":"Encounter","id":"enc-` + strings.Repeat("y", 100) + `","status":"finished","class":{"code":"AMB"},"subject":{"reference":"Patient/patient-123"},"period":{"start":"2024-01-15T09:00:00Z","end":"2024-01-15T10:30:00Z"}}},` +
		`{"resource":{"resourceType":"Observation","id":"obs-` + strings.Repeat("z", 100) + `","status":"final","code":{"coding":[{"system":"http://loinc.org","code":"85354-9","display":"Blood pressure"}]},"subject":{"reference":"Patient/patient-123"},"valueQuantity":{"value":120,"unit":"mmHg"}}}` +
		strings.Repeat("a", 2500) + // Pad to realistic 3-4KB per line
		`]}` + "\n"
	
	lineSize := len(fhirLine)
	
	// Test with multiple files of varying sizes (simulating real workload)
	testCases := []struct {
		name      string
		lineCount int
	}{
		{"small_file_10k_lines", 10000},
		{"medium_file_50k_lines", 50000},
		{"large_file_200k_lines", 200000},
		{"xlarge_file_500k_lines", 500000},
	}
	
	var totalLines int
	var totalTime time.Duration
	var totalBytes int64
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "perf-test-*.ndjson")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write lines
			for i := 0; i < tc.lineCount; i++ {
				if _, err := tmpFile.WriteString(fhirLine); err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
			}
			fileSize := int64(tc.lineCount * lineSize)
			tmpFile.Close()

			// Measure time to count lines
			start := time.Now()
			count, err := CountLines(tmpFile.Name())
			elapsed := time.Since(start)

			if err != nil {
				t.Fatalf("CountLines() error = %v", err)
			}

			if count != tc.lineCount {
				t.Errorf("CountLines() = %d, want %d", count, tc.lineCount)
			}

			// Calculate throughput
			mbPerSec := float64(fileSize) / (1024 * 1024) / elapsed.Seconds()
			linesPerSec := float64(count) / elapsed.Seconds()

			t.Logf("File: %d lines (%d MB), Time: %v, Throughput: %.2f MB/s, %.0f lines/s", 
				count, fileSize/(1024*1024), elapsed, mbPerSec, linesPerSec)
			
			totalLines += count
			totalTime += elapsed
			totalBytes += fileSize
		})
	}
	
	// Overall performance summary
	avgMBPerSec := float64(totalBytes) / (1024 * 1024) / totalTime.Seconds()
	avgLinesPerSec := float64(totalLines) / totalTime.Seconds()
	t.Logf("TOTAL: %d lines (%d MB) in %v - Avg: %.2f MB/s, %.0f lines/s", 
		totalLines, totalBytes/(1024*1024), totalTime, avgMBPerSec, avgLinesPerSec)
	
	// Performance requirement: should handle large files efficiently
	// For 500k lines (~2GB), should complete in reasonable time (< 10s)
	if totalTime > 10*time.Second {
		t.Errorf("Total processing time %v exceeded 10s threshold", totalTime)
	}
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

func TestCountLines_LargeLines(t *testing.T) {
	// Test files with lines exceeding different buffer size thresholds
	// bufio.Scanner default max is 64KB, we've configured 10MB max
	tests := []struct {
		name      string
		lineSize  int
		numLines  int
		wantLines int
		wantErr   bool
	}{
		{
			name:      "lines under 64KB (default scanner limit)",
			lineSize:  50 * 1024, // 50KB per line
			numLines:  5,
			wantLines: 5,
			wantErr:   false,
		},
		{
			name:      "lines over 64KB but under 512KB",
			lineSize:  128 * 1024, // 128KB per line
			numLines:  3,
			wantLines: 3,
			wantErr:   false,
		},
		{
			name:      "lines over 512KB but under 1MB",
			lineSize:  768 * 1024, // 768KB per line
			numLines:  2,
			wantLines: 2,
			wantErr:   false,
		},
		{
			name:      "mixed line sizes - some over 64KB, some over 512KB",
			lineSize:  -1, // Special case: mixed sizes
			numLines:  7,
			wantLines: 7,
			wantErr:   false,
		},
		{
			name:      "lines at max buffer size (10MB)",
			lineSize:  9 * 1024 * 1024, // 9MB per line (just under 10MB limit)
			numLines:  2,
			wantLines: 2,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile, err := os.CreateTemp("", "large-line-test-*.ndjson")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Create test content
			if tt.lineSize == -1 {
				// Mixed line sizes test case
				lineSizes := []int{
					30 * 1024,   // 30KB - under 64KB
					80 * 1024,   // 80KB - over 64KB
					256 * 1024,  // 256KB - over 64KB
					600 * 1024,  // 600KB - over 512KB
					1024 * 1024, // 1MB - over 512KB
					45 * 1024,   // 45KB - under 64KB
					750 * 1024,  // 750KB - over 512KB
				}
				for i, size := range lineSizes {
					line := createJSONLine(size, i)
					if _, err := tmpFile.WriteString(line + "\n"); err != nil {
						t.Fatalf("Failed to write line %d: %v", i, err)
					}
				}
			} else {
				// Fixed line size test cases
				for i := 0; i < tt.numLines; i++ {
					line := createJSONLine(tt.lineSize, i)
					if _, err := tmpFile.WriteString(line + "\n"); err != nil {
						t.Fatalf("Failed to write line %d: %v", i, err)
					}
				}
			}
			
			fileInfo, _ := tmpFile.Stat()
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

			// Log file size for verification
			t.Logf("File size: %.2f MB, Lines counted: %d", 
				float64(fileInfo.Size())/(1024*1024), got)
		})
	}
}

// createJSONLine creates a realistic FHIR bundle JSON line of approximately the specified size
func createJSONLine(targetSize int, index int) string {
	// Start with a basic FHIR bundle structure
	base := `{"resourceType":"Bundle","type":"collection","id":"bundle-` + 
		strings.Repeat("0", 10) + `","entry":[{"resource":{"resourceType":"Patient","id":"patient-` +
		strings.Repeat("x", 20) + `","name":[{"family":"TestPatient","given":["John"]}],"data":"`

	// Calculate padding needed
	overhead := len(base) + len(`"}}]}`)
	if overhead >= targetSize {
		// Minimum size case
		return base[:len(base)-len(`","data":"`)]+`"}}]}`
	}

	paddingSize := targetSize - overhead
	
	// Create padding with realistic-looking base64-like data
	padding := strings.Repeat("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/", paddingSize/64+1)
	padding = padding[:paddingSize]
	
	return base + padding + `"}}]}`
}


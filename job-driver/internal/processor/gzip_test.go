package processor

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

// TestCountLinesGzipped tests counting lines in gzipped files
func TestCountLinesGzipped(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantLines int
	}{
		{
			name:      "single line gzipped",
			content:   `{"resourceType":"Patient","id":"1"}`,
			wantLines: 1,
		},
		{
			name: "multiple lines gzipped",
			content: `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}
{"resourceType":"Patient","id":"3"}`,
			wantLines: 3,
		},
		{
			name: "lines with trailing newline gzipped",
			content: `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}
{"resourceType":"Patient","id":"3"}
`,
			wantLines: 3,
		},
		{
			name:      "empty file gzipped",
			content:   "",
			wantLines: 0,
		},
		{
			name: "ten lines gzipped",
			content: `{"line":1}
{"line":2}
{"line":3}
{"line":4}
{"line":5}
{"line":6}
{"line":7}
{"line":8}
{"line":9}
{"line":10}`,
			wantLines: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp gzipped file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.ndjson.gz")

			f, err := os.Create(tmpFile)
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}

			gzWriter := gzip.NewWriter(f)
			_, err = gzWriter.Write([]byte(tt.content))
			if err != nil {
				f.Close()
				t.Fatalf("Failed to write gzipped content: %v", err)
			}

			if err := gzWriter.Close(); err != nil {
				f.Close()
				t.Fatalf("Failed to close gzip writer: %v", err)
			}
			f.Close()

			// Count lines
			got, err := CountLines(tmpFile)
			if err != nil {
				t.Errorf("CountLines() error = %v", err)
				return
			}

			if got != tt.wantLines {
				t.Errorf("CountLines() = %d, want %d", got, tt.wantLines)
			}
		})
	}
}

// TestCountLinesPlainVsGzipped ensures both formats produce same line counts
func TestCountLinesPlainVsGzipped(t *testing.T) {
	content := `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}
{"resourceType":"Patient","id":"3"}
{"resourceType":"Patient","id":"4"}
{"resourceType":"Patient","id":"5"}`

	tmpDir := t.TempDir()

	// Create plain file
	plainFile := filepath.Join(tmpDir, "test.ndjson")
	if err := os.WriteFile(plainFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create plain file: %v", err)
	}

	// Create gzipped file
	gzFile := filepath.Join(tmpDir, "test.ndjson.gz")
	f, err := os.Create(gzFile)
	if err != nil {
		t.Fatalf("Failed to create gzipped file: %v", err)
	}
	gzWriter := gzip.NewWriter(f)
	gzWriter.Write([]byte(content))
	gzWriter.Close()
	f.Close()

	// Count lines in both
	plainCount, err := CountLines(plainFile)
	if err != nil {
		t.Fatalf("Failed to count plain file lines: %v", err)
	}

	gzCount, err := CountLines(gzFile)
	if err != nil {
		t.Fatalf("Failed to count gzipped file lines: %v", err)
	}

	if plainCount != gzCount {
		t.Errorf("Line counts don't match: plain=%d, gzipped=%d", plainCount, gzCount)
	}

	if plainCount != 5 {
		t.Errorf("Expected 5 lines, got plain=%d, gzipped=%d", plainCount, gzCount)
	}
}

// TestDiscoverGzippedFiles tests file discovery with gzipped files
func TestDiscoverGzippedFiles(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a mix of plain and gzipped files
	files := map[string]string{
		"plain1.ndjson":    "plain",
		"plain2.ndjson":    "plain",
		"compressed.ndjson.gz": "gzipped",
		"data.gz":          "gzipped",
		"other.txt":        "ignored",
	}

	for filename := range files {
		path := filepath.Join(tmpDir, filename)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", filename, err)
		}
	}

	// Discover files
	discovered, err := DiscoverFiles(tmpDir, nil)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	// Should find 4 files (2 plain + 2 gzipped), not the .txt file
	if len(discovered) != 4 {
		t.Errorf("Expected 4 files discovered, got %d: %v", len(discovered), discovered)
	}

	// Verify expected files are present
	expectedFiles := map[string]bool{
		"plain1.ndjson":    false,
		"plain2.ndjson":    false,
		"compressed.ndjson.gz": false,
		"data.gz":          false,
	}

	for _, file := range discovered {
		if _, ok := expectedFiles[file]; ok {
			expectedFiles[file] = true
		}
	}

	for file, found := range expectedFiles {
		if !found {
			t.Errorf("Expected file %s not discovered", file)
		}
	}
}

// TestLargeGzippedFile tests handling of larger gzipped files (memory efficiency)
func TestLargeGzippedFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "large.ndjson.gz")

	// Create content with exactly 1000 lines
	content := ""
	for i := 1; i <= 999; i++ {
		content += `{"line":` + string(rune('0'+i%10)) + `}` + "\n"
	}
	content += `{"line":1000}` // Last line without trailing newline

	// Write gzipped file
	f, err := os.Create(tmpFile)
	if err != nil {
		t.Fatalf("Failed to create file: %v", err)
	}
	gzWriter := gzip.NewWriter(f)
	_, err = gzWriter.Write([]byte(content))
	if err != nil {
		f.Close()
		t.Fatalf("Failed to write content: %v", err)
	}
	gzWriter.Close()
	f.Close()

	// Count lines - should not load entire file into memory
	count, err := CountLines(tmpFile)
	if err != nil {
		t.Fatalf("CountLines() error = %v", err)
	}

	if count != 1000 {
		t.Errorf("Expected 1000 lines, got %d", count)
	}
}

// TestCloudStorageGzipped tests that cloud storage files with .gz extension are properly decompressed
func TestCloudStorageGzipped(t *testing.T) {
	// This test validates the composable reader architecture
	// Cloud storage + gzip should work without a specific combined type
	
	content := `{"resourceType":"Patient","id":"1"}
{"resourceType":"Patient","id":"2"}
{"resourceType":"Patient","id":"3"}
{"resourceType":"Patient","id":"4"}
{"resourceType":"Patient","id":"5"}`

	tmpDir := t.TempDir()
	
	// Create gzipped file
	localGzFile := filepath.Join(tmpDir, "test-data.ndjson.gz")
	f, err := os.Create(localGzFile)
	if err != nil {
		t.Fatalf("Failed to create gzipped file: %v", err)
	}
	gzWriter := gzip.NewWriter(f)
	if _, err := gzWriter.Write([]byte(content)); err != nil {
		f.Close()
		t.Fatalf("Failed to write content: %v", err)
	}
	gzWriter.Close()
	f.Close()

	// Test with file:// URL that includes the full path
	// For file:// URLs, the storage layer expects: file:///dir + filename
	// We construct the full path as file:///dir/file.gz
	fileURL := "file:///" + filepath.ToSlash(localGzFile)
	
	count, err := CountLines(fileURL)
	if err != nil {
		t.Fatalf("CountLines() error = %v", err)
	}

	if count != 5 {
		t.Errorf("Expected 5 lines from gzipped cloud storage file, got %d", count)
	}

	t.Logf("Successfully counted %d lines from gzipped cloud storage file", count)
}

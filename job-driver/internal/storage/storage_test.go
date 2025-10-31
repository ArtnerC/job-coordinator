package storage

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
		checkFn func(t *testing.T, result string)
	}{
		{
			name:  "gs URL with path converts to prefix query param",
			input: "gs://my-bucket/path/to/files",
			checkFn: func(t *testing.T, result string) {
				expected := "gs://my-bucket?prefix=path%2Fto%2Ffiles%2F"
				if result != expected {
					t.Errorf("expected gs:// URL with prefix, got %s, want %s", result, expected)
				}
			},
		},
		{
			name:  "s3 URL with path converts to prefix query param",
			input: "s3://my-bucket/path/to/files",
			checkFn: func(t *testing.T, result string) {
				expected := "s3://my-bucket?prefix=path%2Fto%2Ffiles%2F"
				if result != expected {
					t.Errorf("expected s3:// URL with prefix, got %s, want %s", result, expected)
				}
			},
		},
		{
			name:  "file URL unchanged",
			input: "file:///tmp/data",
			checkFn: func(t *testing.T, result string) {
				if result != "file:///tmp/data" {
					t.Errorf("expected file:// URL unchanged, got %s", result)
				}
			},
		},
		{
			name:  "relative path converted",
			input: "testdata",
			checkFn: func(t *testing.T, result string) {
				if !strings.HasPrefix(result, "file:///") {
					t.Errorf("expected file:// prefix, got %s", result)
				}
				if !strings.Contains(result, "testdata") {
					t.Errorf("expected testdata in path, got %s", result)
				}
			},
		},
		{
			name:    "unsupported scheme",
			input:   "http://example.com/files",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := normalizePath(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("normalizePath() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFn != nil {
				tt.checkFn(t, result)
			}
		})
	}
}

func TestExtractPrefix(t *testing.T) {
	tests := []struct {
		name     string
		pattern  string
		expected string
	}{
		{
			name:     "no wildcard",
			pattern:  "patients/2023/data.ndjson",
			expected: "patients/2023/data.ndjson",
		},
		{
			name:     "wildcard at end",
			pattern:  "patients/*.ndjson",
			expected: "patients/",
		},
		{
			name:     "wildcard in filename",
			pattern:  "patients/bundle-*.ndjson",
			expected: "patients/",
		},
		{
			name:     "empty pattern",
			pattern:  "",
			expected: "",
		},
		{
			name:     "wildcard at start",
			pattern:  "*.ndjson",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractPrefix(tt.pattern)
			if result != tt.expected {
				t.Errorf("extractPrefix(%q) = %q, want %q", tt.pattern, result, tt.expected)
			}
		})
	}
}

func TestParseStorageURL(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantScheme     string
		wantBucket     string
		wantPath       string
		wantErr        bool
		checkBucketAbs bool // For file scheme, check bucket is absolute
	}{
		{
			name:       "GCS URL",
			input:      "gs://my-bucket/path/to/files",
			wantScheme: "gs",
			wantBucket: "my-bucket",
			wantPath:   "path/to/files",
		},
		{
			name:       "S3 URL",
			input:      "s3://my-bucket/path/to/files",
			wantScheme: "s3",
			wantBucket: "my-bucket",
			wantPath:   "path/to/files",
		},
		{
			name:       "file URL",
			input:      "file:///tmp/data/files",
			wantScheme: "file",
			wantBucket: "", // file:// URLs don't have a "host" in URL parsing
			wantPath:   "tmp/data/files",
		},
		{
			name:           "local relative path",
			input:          "testdata/bundles",
			wantScheme:     "file",
			checkBucketAbs: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, bucket, path, err := ParseStorageURL(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseStorageURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if scheme != tt.wantScheme {
				t.Errorf("scheme = %q, want %q", scheme, tt.wantScheme)
			}
			if tt.checkBucketAbs {
				if !filepath.IsAbs(bucket) {
					t.Errorf("bucket should be absolute path, got %q", bucket)
				}
			} else if bucket != tt.wantBucket {
				t.Errorf("bucket = %q, want %q", bucket, tt.wantBucket)
			}
			if tt.wantPath != "" && path != tt.wantPath {
				t.Errorf("path = %q, want %q", path, tt.wantPath)
			}
		})
	}
}

func TestReader_LocalFiles(t *testing.T) {
	// Create temp directory with test files
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test files
	testFiles := map[string]string{
		"file1.ndjson":              "line1\nline2\n",
		"file2.ndjson":              "data\n",
		"subdir/file3.ndjson":       "content\n",
		"subdir/nested/file4.txt":   "text\n",
		"bundle-001.ndjson":         "bundle1\n",
		"bundle-002.ndjson":         "bundle2\n",
	}

	for path, content := range testFiles {
		fullPath := filepath.Join(tmpDir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to write file %s: %v", path, err)
		}
	}

	ctx := context.Background()

	t.Run("open and read file", func(t *testing.T) {
		reader, err := NewReader(ctx, tmpDir)
		if err != nil {
			t.Fatalf("NewReader() error = %v", err)
		}
		defer reader.Close()

		// Read file1.ndjson
		rc, err := reader.OpenFile("file1.ndjson")
		if err != nil {
			t.Fatalf("OpenFile() error = %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		expected := "line1\nline2\n"
		if string(data) != expected {
			t.Errorf("got %q, want %q", string(data), expected)
		}
	})

	t.Run("list all files", func(t *testing.T) {
		reader, err := NewReader(ctx, tmpDir)
		if err != nil {
			t.Fatalf("NewReader() error = %v", err)
		}
		defer reader.Close()

		files, err := reader.List("")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(files) != len(testFiles) {
			t.Errorf("List() returned %d files, want %d", len(files), len(testFiles))
		}
	})

	t.Run("list with pattern", func(t *testing.T) {
		reader, err := NewReader(ctx, tmpDir)
		if err != nil {
			t.Fatalf("NewReader() error = %v", err)
		}
		defer reader.Close()

		// List only .ndjson files
		files, err := reader.List("*.ndjson")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		// Pattern *.ndjson should match files in root and subdirectories
		// Should find: file1.ndjson, file2.ndjson, bundle-001.ndjson, bundle-002.ndjson, subdir/file3.ndjson
		// Verify all have .ndjson extension
		for _, f := range files {
			if !strings.HasSuffix(f, ".ndjson") {
				t.Errorf("file %s doesn't have .ndjson extension", f)
			}
		}
		
		// Should be at least 4 files (could be more with subdirs)
		if len(files) < 4 {
			t.Errorf("List('*.ndjson') returned %d files, want at least 4", len(files))
		}
	})

	t.Run("list with bundle pattern", func(t *testing.T) {
		reader, err := NewReader(ctx, tmpDir)
		if err != nil {
			t.Fatalf("NewReader() error = %v", err)
		}
		defer reader.Close()

		files, err := reader.List("bundle-*.ndjson")
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		expectedCount := 2
		if len(files) != expectedCount {
			t.Errorf("List('bundle-*.ndjson') returned %d files, want %d", len(files), expectedCount)
			for _, f := range files {
				t.Logf("  - %s", f)
			}
		}
	})

	t.Run("open nested file", func(t *testing.T) {
		reader, err := NewReader(ctx, tmpDir)
		if err != nil {
			t.Fatalf("NewReader() error = %v", err)
		}
		defer reader.Close()

		rc, err := reader.OpenFile("subdir/file3.ndjson")
		if err != nil {
			t.Fatalf("OpenFile() error = %v", err)
		}
		defer rc.Close()

		data, err := io.ReadAll(rc)
		if err != nil {
			t.Fatalf("ReadAll() error = %v", err)
		}

		expected := "content\n"
		if string(data) != expected {
			t.Errorf("got %q, want %q", string(data), expected)
		}
	})

	t.Run("file not found", func(t *testing.T) {
		reader, err := NewReader(ctx, tmpDir)
		if err != nil {
			t.Fatalf("NewReader() error = %v", err)
		}
		defer reader.Close()

		_, err = reader.OpenFile("nonexistent.ndjson")
		if err == nil {
			t.Error("expected error for nonexistent file, got nil")
		}
	})
}

func TestReader_WithFileURL(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "hello world\n"
	if err := os.WriteFile(testFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	ctx := context.Background()

	// Test with file:// URL
	fileURL := "file:///" + filepath.ToSlash(tmpDir)
	reader, err := NewReader(ctx, fileURL)
	if err != nil {
		t.Fatalf("NewReader() error = %v", err)
	}
	defer reader.Close()

	rc, err := reader.OpenFile("test.txt")
	if err != nil {
		t.Fatalf("OpenFile() error = %v", err)
	}
	defer rc.Close()

	data, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}

	if string(data) != testContent {
		t.Errorf("got %q, want %q", string(data), testContent)
	}
}

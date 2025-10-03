package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverFiles(t *testing.T) {
	tests := []struct {
		name           string
		basePath       string
		manifestPath   string
		setupFunc      func(t *testing.T) (string, string, func()) // Returns basePath, manifestPath, cleanup
		wantFileCount  int
		wantErr        bool
		errMsg         string
	}{
		{
			name: "discover from valid manifest",
			setupFunc: func(t *testing.T) (string, string, func()) {
				// Create temp directory and manifest
				tmpDir, err := os.MkdirTemp("", "discover-test-")
				if err != nil {
					t.Fatal(err)
				}
				
				// Create manifest file
				manifestPath := filepath.Join(tmpDir, "manifest.txt")
				manifestContent := "file1.ndjson\nfile2.ndjson\nfile3.ndjson\n"
				if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
					os.RemoveAll(tmpDir)
					t.Fatal(err)
				}
				
				// Create the actual files
				for _, fname := range []string{"file1.ndjson", "file2.ndjson", "file3.ndjson"} {
					if err := os.WriteFile(filepath.Join(tmpDir, fname), []byte("data\n"), 0644); err != nil {
						os.RemoveAll(tmpDir)
						t.Fatal(err)
					}
				}
				
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, manifestPath, cleanup
			},
			wantFileCount: 3,
			wantErr:       false,
		},
		{
			name: "discover from directory (no manifest)",
			setupFunc: func(t *testing.T) (string, string, func()) {
				// Create temp directory with .ndjson files
				tmpDir, err := os.MkdirTemp("", "discover-test-")
				if err != nil {
					t.Fatal(err)
				}
				
				// Create .ndjson files
				for _, fname := range []string{"data1.ndjson", "data2.ndjson", "subdir/data3.ndjson"} {
					fpath := filepath.Join(tmpDir, fname)
					if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
						os.RemoveAll(tmpDir)
						t.Fatal(err)
					}
					if err := os.WriteFile(fpath, []byte("data\n"), 0644); err != nil {
						os.RemoveAll(tmpDir)
						t.Fatal(err)
					}
				}
				
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, "", cleanup
			},
			wantFileCount: 3,
			wantErr:       false,
		},
		{
			name: "manifest file not found",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, err := os.MkdirTemp("", "discover-test-")
				if err != nil {
					t.Fatal(err)
				}
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, filepath.Join(tmpDir, "nonexistent.txt"), cleanup
			},
			wantFileCount: 0,
			wantErr:       true,
			errMsg:        "failed to parse manifest",
		},
		{
			name: "empty directory (no files)",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, err := os.MkdirTemp("", "discover-test-")
				if err != nil {
					t.Fatal(err)
				}
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, "", cleanup
			},
			wantFileCount: 0,
			wantErr:       true,
			errMsg:        "no .ndjson files found",
		},
		{
			name: "directory with mixed file types (only .ndjson)",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, err := os.MkdirTemp("", "discover-test-")
				if err != nil {
					t.Fatal(err)
				}
				
				// Create mixed file types
				files := map[string]string{
					"data1.ndjson": "data",
					"data2.json":   "data",
					"data3.txt":    "data",
					"data4.ndjson": "data",
				}
				for fname, content := range files {
					if err := os.WriteFile(filepath.Join(tmpDir, fname), []byte(content), 0644); err != nil {
						os.RemoveAll(tmpDir)
						t.Fatal(err)
					}
				}
				
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, "", cleanup
			},
			wantFileCount: 2, // Only .ndjson files
			wantErr:       false,
		},
		{
			name: "manifest with comments and empty lines",
			setupFunc: func(t *testing.T) (string, string, func()) {
				tmpDir, err := os.MkdirTemp("", "discover-test-")
				if err != nil {
					t.Fatal(err)
				}
				
				// Create manifest with comments
				manifestPath := filepath.Join(tmpDir, "manifest.txt")
				manifestContent := "# This is a comment\nfile1.ndjson\n\n# Another comment\nfile2.ndjson\n"
				if err := os.WriteFile(manifestPath, []byte(manifestContent), 0644); err != nil {
					os.RemoveAll(tmpDir)
					t.Fatal(err)
				}
				
				// Create the actual files
				for _, fname := range []string{"file1.ndjson", "file2.ndjson"} {
					if err := os.WriteFile(filepath.Join(tmpDir, fname), []byte("data\n"), 0644); err != nil {
						os.RemoveAll(tmpDir)
						t.Fatal(err)
					}
				}
				
				cleanup := func() { os.RemoveAll(tmpDir) }
				return tmpDir, manifestPath, cleanup
			},
			wantFileCount: 2,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basePath, manifestPath, cleanup := tt.setupFunc(t)
			defer cleanup()

			var manifestPtr *string
			if manifestPath != "" {
				manifestPtr = &manifestPath
			}

			files, err := DiscoverFiles(basePath, manifestPtr)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DiscoverFiles() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("DiscoverFiles() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("DiscoverFiles() unexpected error = %v", err)
			}

			if len(files) != tt.wantFileCount {
				t.Errorf("DiscoverFiles() returned %d files, want %d", len(files), tt.wantFileCount)
			}

			// Verify all returned paths are relative
			for _, file := range files {
				if filepath.IsAbs(file) {
					t.Errorf("DiscoverFiles() returned absolute path %q, want relative path", file)
				}
			}
		})
	}
}

func TestDiscoverFiles_ActualTestData(t *testing.T) {
	// Test with actual testdata
	basePath := filepath.Join("..", "..", "testdata", "bundles")
	
	// Skip if directory doesn't exist
	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		t.Skip("Test data directory not found")
	}

	files, err := DiscoverFiles(basePath, nil)
	if err != nil {
		t.Fatalf("DiscoverFiles() error = %v", err)
	}

	if len(files) == 0 {
		t.Error("DiscoverFiles() returned no files from testdata/bundles")
	}

	t.Logf("Discovered %d files from testdata/bundles", len(files))
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

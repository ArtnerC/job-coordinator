package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantPaths []string
		wantErr  bool
	}{
		{
			name: "valid manifest file",
			content: `patients/2023/jan/bundle-001.ndjson
patients/2023/jan/bundle-002.ndjson
patients/2023/feb/bundle-003.ndjson`,
			wantPaths: []string{
				"patients/2023/jan/bundle-001.ndjson",
				"patients/2023/jan/bundle-002.ndjson",
				"patients/2023/feb/bundle-003.ndjson",
			},
			wantErr: false,
		},
		{
			name:      "empty manifest",
			content:   "",
			wantPaths: []string{},
			wantErr:   false,
		},
		{
			name: "manifest with comments",
			content: `# This is a comment
patients/2023/jan/bundle-001.ndjson
# Another comment
patients/2023/jan/bundle-002.ndjson`,
			wantPaths: []string{
				"patients/2023/jan/bundle-001.ndjson",
				"patients/2023/jan/bundle-002.ndjson",
			},
			wantErr: false,
		},
		{
			name: "manifest with empty lines",
			content: `patients/2023/jan/bundle-001.ndjson

patients/2023/jan/bundle-002.ndjson

`,
			wantPaths: []string{
				"patients/2023/jan/bundle-001.ndjson",
				"patients/2023/jan/bundle-002.ndjson",
			},
			wantErr: false,
		},
		{
			name: "manifest with mixed comments and empty lines",
			content: `# Header comment
patients/2023/jan/bundle-001.ndjson

# Section for February
patients/2023/feb/bundle-002.ndjson
# End of manifest
`,
			wantPaths: []string{
				"patients/2023/jan/bundle-001.ndjson",
				"patients/2023/feb/bundle-002.ndjson",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary manifest file
			tmpFile, err := os.CreateTemp("", "manifest-*.txt")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			// Write content
			if _, err := tmpFile.WriteString(tt.content); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			// Parse manifest
			gotPaths, err := ParseManifest(tmpFile.Name())
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseManifest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Check paths
			if len(gotPaths) != len(tt.wantPaths) {
				t.Fatalf("ParseManifest() returned %d paths, want %d", len(gotPaths), len(tt.wantPaths))
			}

			for i, want := range tt.wantPaths {
				if gotPaths[i] != want {
					t.Errorf("ParseManifest() path[%d] = %q, want %q", i, gotPaths[i], want)
				}
			}
		})
	}
}

func TestParseManifest_FileNotFound(t *testing.T) {
	_, err := ParseManifest("/nonexistent/manifest.txt")
	if err == nil {
		t.Error("ParseManifest() expected error for non-existent file, got nil")
	}
}

func TestParseManifest_RelativePaths(t *testing.T) {
	content := `patients/bundle-001.ndjson
patients/bundle-002.ndjson`

	tmpFile, err := os.CreateTemp("", "manifest-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	paths, err := ParseManifest(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseManifest() error = %v", err)
	}

	// All paths should be relative (no leading /)
	for _, path := range paths {
		if filepath.IsAbs(path) {
			t.Errorf("ParseManifest() returned absolute path %q, want relative", path)
		}
	}
}

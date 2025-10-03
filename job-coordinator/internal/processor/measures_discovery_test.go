package processor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverMeasures(t *testing.T) {
	// Create temporary measures directory
	tmpDir, err := os.MkdirTemp("", "measures-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create test measure files
	measures := []string{
		"cms-125.json",
		"cms-130.json",
		"cms-135.json",
	}

	for _, m := range measures {
		filePath := filepath.Join(tmpDir, m)
		if err := os.WriteFile(filePath, []byte("{}"), 0644); err != nil {
			t.Fatalf("Failed to create measure file: %v", err)
		}
	}

	// Test discovering measures
	paths, err := DiscoverMeasures(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverMeasures() error = %v", err)
	}

	if len(paths) != len(measures) {
		t.Errorf("DiscoverMeasures() returned %d paths, want %d", len(paths), len(measures))
	}

	// All paths should be absolute
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("DiscoverMeasures() returned relative path %q, want absolute", path)
		}
	}

	// Check that all measure files were found
	pathMap := make(map[string]bool)
	for _, path := range paths {
		pathMap[filepath.Base(path)] = true
	}

	for _, m := range measures {
		if !pathMap[m] {
			t.Errorf("DiscoverMeasures() did not find measure %q", m)
		}
	}
}

func TestDiscoverMeasures_EmptyDirectory(t *testing.T) {
	// Create empty temp directory
	tmpDir, err := os.MkdirTemp("", "measures-empty-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	paths, err := DiscoverMeasures(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverMeasures() error = %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("DiscoverMeasures() returned %d paths for empty directory, want 0", len(paths))
	}
}

func TestDiscoverMeasures_NonExistentDirectory(t *testing.T) {
	_, err := DiscoverMeasures("/nonexistent/measures")
	if err == nil {
		t.Error("DiscoverMeasures() expected error for non-existent directory, got nil")
	}
}

func TestDiscoverMeasures_RecursiveSearch(t *testing.T) {
	// Create nested directory structure
	tmpDir, err := os.MkdirTemp("", "measures-nested-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create subdirectories
	subDir1 := filepath.Join(tmpDir, "2023")
	subDir2 := filepath.Join(tmpDir, "2023", "q4")
	if err := os.MkdirAll(subDir2, 0755); err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	// Create measures at different levels
	measures := map[string]string{
		filepath.Join(tmpDir, "cms-125.json"):         "{}",
		filepath.Join(subDir1, "cms-130.json"):        "{}",
		filepath.Join(subDir2, "cms-135.json"):        "{}",
	}

	for path, content := range measures {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create measure file: %v", err)
		}
	}

	// Discover measures recursively
	paths, err := DiscoverMeasures(tmpDir)
	if err != nil {
		t.Fatalf("DiscoverMeasures() error = %v", err)
	}

	if len(paths) != len(measures) {
		t.Errorf("DiscoverMeasures() returned %d paths, want %d", len(paths), len(measures))
	}
}

func TestParseMeasuresManifest(t *testing.T) {
	content := `/data/measures/cms-125.json
/data/measures/cms-130.json
/data/measures/cms-135.json`

	tmpFile, err := os.CreateTemp("", "measures-manifest-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	paths, err := ParseMeasuresManifest(tmpFile.Name())
	if err != nil {
		t.Fatalf("ParseMeasuresManifest() error = %v", err)
	}

	wantPaths := []string{
		"/data/measures/cms-125.json",
		"/data/measures/cms-130.json",
		"/data/measures/cms-135.json",
	}

	if len(paths) != len(wantPaths) {
		t.Errorf("ParseMeasuresManifest() returned %d paths, want %d", len(paths), len(wantPaths))
	}

	for i, want := range wantPaths {
		if i < len(paths) && paths[i] != want {
			t.Errorf("ParseMeasuresManifest() path[%d] = %q, want %q", i, paths[i], want)
		}
	}

	// All paths should be absolute
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			t.Errorf("ParseMeasuresManifest() returned relative path %q, want absolute", path)
		}
	}
}

func TestDiscoverMeasures_ActualTestData(t *testing.T) {
	// Test with actual testdata
	testDir := filepath.Join("..", "..", "testdata", "measures")
	
	// Skip if directory doesn't exist (CI environment)
	if _, err := os.Stat(testDir); os.IsNotExist(err) {
		t.Skip("Test data directory not found")
	}

	paths, err := DiscoverMeasures(testDir)
	if err != nil {
		t.Fatalf("DiscoverMeasures() error = %v", err)
	}

	// Should find at least 2 measures (cms-125.json, cms-130.json)
	if len(paths) < 2 {
		t.Errorf("DiscoverMeasures() returned %d paths, want at least 2", len(paths))
	}

	// Check for expected measures
	pathMap := make(map[string]bool)
	for _, path := range paths {
		pathMap[filepath.Base(path)] = true
	}

	expectedMeasures := []string{"cms-125.json", "cms-130.json"}
	for _, expected := range expectedMeasures {
		if !pathMap[expected] {
			t.Errorf("DiscoverMeasures() did not find expected measure %q", expected)
		}
	}
}

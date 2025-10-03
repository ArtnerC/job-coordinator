package processor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverFiles discovers all .ndjson files to process
// If manifestPath is provided, reads files from manifest
// Otherwise, walks the directory tree from basePath
func DiscoverFiles(basePath string, manifestPath *string) ([]string, error) {
	if manifestPath != nil && *manifestPath != "" {
		// Use manifest to get list of files
		files, err := ParseManifest(*manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
		return files, nil
	}

	// No manifest - discover files by walking directory tree
	return discoverFilesInDirectory(basePath)
}

// discoverFilesInDirectory recursively finds all .ndjson files in a directory
func discoverFilesInDirectory(basePath string) ([]string, error) {
	var files []string

	err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Only include .ndjson files
		if strings.HasSuffix(strings.ToLower(info.Name()), ".ndjson") {
			// Make path relative to basePath
			relPath, err := filepath.Rel(basePath, path)
			if err != nil {
				return fmt.Errorf("failed to make path relative: %w", err)
			}
			files = append(files, relPath)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk directory: %w", err)
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .ndjson files found in %s", basePath)
	}

	return files, nil
}

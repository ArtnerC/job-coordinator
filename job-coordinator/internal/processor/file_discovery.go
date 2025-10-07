package processor

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dqme/job-coordinator/internal/storage"
)

// DiscoverFiles discovers all .ndjson files to process
// If manifestPath is provided, reads files from manifest
// Otherwise, walks the directory tree from basePath
// Supports local paths and cloud storage URLs (gs://, s3://, file://)
func DiscoverFiles(basePath string, manifestPath *string) ([]string, error) {
	if manifestPath != nil && *manifestPath != "" {
		// Use manifest to get list of files
		files, err := ParseManifest(*manifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
		return files, nil
	}

	// Check if basePath is a cloud storage URL
	if isCloudStorageURL(basePath) {
		return discoverFilesInCloudStorage(basePath)
	}

	// No manifest - discover files by walking local directory tree
	return discoverFilesInDirectory(basePath)
}

// isCloudStorageURL checks if a path is a cloud storage URL
func isCloudStorageURL(path string) bool {
	return strings.HasPrefix(path, "gs://") ||
		strings.HasPrefix(path, "s3://") ||
		strings.HasPrefix(path, "file://")
}

// discoverFilesInCloudStorage discovers files in cloud storage using the storage package
// The storage package automatically handles path prefixes via gocloud's PrefixedBucket
func discoverFilesInCloudStorage(basePath string) ([]string, error) {
	ctx := context.Background()
	
	// Create storage reader for the cloud storage URL
	// If basePath is gs://bucket/prefix, the storage package will use PrefixedBucket
	// and List() will return files relative to the prefix
	reader, err := storage.NewReader(ctx, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage reader: %w", err)
	}
	defer reader.Close()

	// List all files - returns files relative to the prefix (if any)
	allFiles, err := reader.List("")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Filter for .ndjson and .gz files
	var files []string
	for _, file := range allFiles {
		lowerName := strings.ToLower(file)
		// Include .ndjson and .ndjson.gz files
		if strings.HasSuffix(lowerName, ".ndjson") ||
			strings.HasSuffix(lowerName, ".ndjson.gz") ||
			strings.HasSuffix(lowerName, ".gz") {
			files = append(files, file)
		}
	}

	if len(files) == 0 {
		return nil, fmt.Errorf("no .ndjson or .gz files found in %s", basePath)
	}

	return files, nil
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

		// Include .ndjson and .ndjson.gz files (.gz is also accepted)
		lowerName := strings.ToLower(info.Name())
		if strings.HasSuffix(lowerName, ".ndjson") || 
		   strings.HasSuffix(lowerName, ".ndjson.gz") ||
		   strings.HasSuffix(lowerName, ".gz") {
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
		return nil, fmt.Errorf("no .ndjson or .gz files found in %s", basePath)
	}

	return files, nil
}

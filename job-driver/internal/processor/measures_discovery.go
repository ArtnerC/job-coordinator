package processor

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dqme/job-driver/internal/storage"
)

// DiscoverMeasures recursively discovers all .json files in a directory.
// Returns absolute paths to all measure files found.
// Supports local paths and cloud storage URLs (gs://, s3://, file://)
func DiscoverMeasures(measuresPath string) ([]string, error) {
	// Check if this is a cloud storage URL
	if isCloudStoragePath(measuresPath) {
		return discoverMeasuresInCloudStorage(measuresPath)
	}

	// Local filesystem - check if directory exists
	info, err := os.Stat(measuresPath)
	if err != nil {
		return nil, fmt.Errorf("measures path not found: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("measures path is not a directory: %s", measuresPath)
	}

	var measures []string
	
	err = filepath.Walk(measuresPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if info.IsDir() {
			return nil
		}
		
		// Only include .json files
		if filepath.Ext(path) == ".json" {
			// Convert to absolute path
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}
			measures = append(measures, absPath)
		}
		
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk measures directory: %w", err)
	}

	return measures, nil
}

// discoverMeasuresInCloudStorage discovers .json measure files in cloud storage
// The storage package automatically handles path prefixes via gocloud's PrefixedBucket
func discoverMeasuresInCloudStorage(measuresPath string) ([]string, error) {
	ctx := context.Background()
	
	// Create storage reader for the cloud storage URL
	// If measuresPath is gs://bucket/prefix, storage will use PrefixedBucket
	reader, err := storage.NewReader(ctx, measuresPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage reader: %w", err)
	}
	defer reader.Close()

	// List all files - returns files relative to the prefix (if any)
	allFiles, err := reader.List("")
	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	// Filter for .json files and construct full URLs
	var measures []string
	
	for _, file := range allFiles {
		if strings.HasSuffix(strings.ToLower(file), ".json") {
			// Construct full path by joining with the base path
			// For gs://bucket/measures + file.json -> gs://bucket/measures/file.json
			fullURL := measuresPath
			if !strings.HasSuffix(fullURL, "/") {
				fullURL += "/"
			}
			fullURL += file
			measures = append(measures, fullURL)
		}
	}

	if len(measures) == 0 {
		return nil, fmt.Errorf("no .json measure files found in %s", measuresPath)
	}

	return measures, nil
}

// ParseMeasuresManifest reads a measures manifest file and returns a list of measure paths.
// Expects absolute paths to measure files, one per line.
// Skips empty lines and comments.
func ParseMeasuresManifest(manifestPath string) ([]string, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open measures manifest: %w", err)
	}
	defer file.Close()

	var paths []string
	scanner := bufio.NewScanner(file)
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		paths = append(paths, line)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading measures manifest: %w", err)
	}

	return paths, nil
}

package processor

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DiscoverMeasures recursively discovers all .json files in a directory.
// Returns absolute paths to all measure files found.
func DiscoverMeasures(measuresPath string) ([]string, error) {
	// Check if directory exists
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

package processor

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ParseManifest reads a manifest file and returns a list of file paths.
// Skips empty lines and lines starting with # (comments).
// Returns relative file paths as-is.
func ParseManifest(manifestPath string) ([]string, error) {
	file, err := os.Open(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open manifest: %w", err)
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
		return nil, fmt.Errorf("error reading manifest: %w", err)
	}

	return paths, nil
}

package processor

import (
	"bufio"
	"fmt"
	"os"
)

// CountLines counts the number of lines in a file using streaming approach.
// Uses bufio.Scanner for memory-efficient line-by-line reading.
// Handles files with or without trailing newline correctly.
func CountLines(filePath string) (int, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	count := 0
	
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	return count, nil
}

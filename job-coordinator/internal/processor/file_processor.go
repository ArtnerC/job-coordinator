package processor

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"strings"
)

// isGzipped checks if a file is gzipped based on its extension
func isGzipped(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".gz")
}

// openFileReader opens a file and returns an io.ReadCloser, handling gzip decompression if needed.
// The reader must be closed by the caller.
func openFileReader(filePath string) (io.ReadCloser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}

	if isGzipped(filePath) {
		gzReader, err := gzip.NewReader(file)
		if err != nil {
			file.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		// Return a combined closer that closes both gzip reader and file
		return &gzipReadCloser{gzReader: gzReader, file: file}, nil
	}

	return file, nil
}

// gzipReadCloser wraps both gzip.Reader and os.File to ensure both are closed
type gzipReadCloser struct {
	gzReader *gzip.Reader
	file     *os.File
}

func (g *gzipReadCloser) Read(p []byte) (n int, err error) {
	return g.gzReader.Read(p)
}

func (g *gzipReadCloser) Close() error {
	gzErr := g.gzReader.Close()
	fileErr := g.file.Close()
	if gzErr != nil {
		return gzErr
	}
	return fileErr
}

// CountLines counts the number of lines in a file using streaming approach.
// Uses bufio.Scanner for memory-efficient line-by-line reading.
// Handles files with or without trailing newline correctly.
// Supports both plain and gzipped files (.gz extension).
func CountLines(filePath string) (int, error) {
	reader, err := openFileReader(filePath)
	if err != nil {
		return 0, err
	}
	defer reader.Close()

	scanner := bufio.NewScanner(reader)
	count := 0
	
	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	return count, nil
}

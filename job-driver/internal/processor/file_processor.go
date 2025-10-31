package processor

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/storage"
)

// isGzipped checks if a file is gzipped based on its extension
func isGzipped(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".gz")
}

// isCloudStoragePath checks if a path is a cloud storage URL
func isCloudStoragePath(path string) bool {
	return strings.HasPrefix(path, "gs://") ||
		strings.HasPrefix(path, "s3://") ||
		strings.HasPrefix(path, "file://")
}

// openFileReader opens a file and returns an io.ReadCloser, handling gzip decompression if needed.
// The reader must be closed by the caller.
// Supports both local paths and cloud storage URLs (gs://, s3://, file://)
func openFileReader(filePath string) (io.ReadCloser, error) {
	var baseReader io.ReadCloser
	var err error

	// Get the base reader (local file or cloud storage)
	if isCloudStoragePath(filePath) {
		baseReader, err = openCloudStorageFile(filePath)
	} else {
		baseReader, err = os.Open(filePath)
	}

	if err != nil {
		return nil, err
	}

	// Compose with gzip reader if needed
	if isGzipped(filePath) {
		return wrapWithGzip(baseReader)
	}

	return baseReader, nil
}

// openCloudStorageFile opens a file from cloud storage (gs://, s3://, file://)
// Returns a composable io.ReadCloser that can be wrapped with additional layers
func openCloudStorageFile(fileURL string) (io.ReadCloser, error) {
	ctx := context.Background()

	// Parse the URL to get base path and file key
	scheme, bucket, key, err := storage.ParseStorageURL(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storage URL: %w", err)
	}

	// For file:// URLs, ParseStorageURL returns empty bucket and full path in key
	// We need to split the path into directory (for NewReader) and filename (for OpenFile)
	var baseURL string
	var filename string
	
	if scheme == "file" {
		// key contains the full path like "tmp/data/file.ndjson" or "C:/Users/file.ndjson"
		// Split into directory and filename
		dir, file := filepath.Split(key)
		if dir == "" {
			return nil, fmt.Errorf("file path must include directory: %s", fileURL)
		}
		// Reconstruct file:// URL with directory only
		baseURL = "file:///" + filepath.ToSlash(dir)
		filename = file
	} else {
		// For gs:// and s3://, bucket is the host and key is the path
		baseURL = fmt.Sprintf("%s://%s", scheme, bucket)
		filename = key
	}

	// Create storage reader
	reader, err := storage.NewReader(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage reader: %w", err)
	}

	// Open the specific file
	fileReader, err := reader.OpenFile(filename)
	if err != nil {
		reader.Close()
		return nil, fmt.Errorf("failed to open file in storage: %w", err)
	}

	// Return composable closer that manages both file and bucket
	return &multiCloser{
		reader:  fileReader,
		closers: []io.Closer{fileReader, reader},
	}, nil
}

// wrapWithGzip wraps an io.ReadCloser with gzip decompression
// Returns a composable reader that closes both the gzip reader and underlying reader
func wrapWithGzip(r io.ReadCloser) (io.ReadCloser, error) {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		r.Close()
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	return &multiCloser{
		reader:  gzReader,
		closers: []io.Closer{gzReader, r},
	}, nil
}

// multiCloser is a composable io.ReadCloser that manages multiple closers
// This allows stacking readers (e.g., gzip over cloud storage) without specific combinations
type multiCloser struct {
	reader  io.Reader
	closers []io.Closer
}

func (m *multiCloser) Read(p []byte) (n int, err error) {
	return m.reader.Read(p)
}

func (m *multiCloser) Close() error {
	var firstErr error
	// Close in reverse order (innermost first)
	for i := len(m.closers) - 1; i >= 0; i-- {
		if err := m.closers[i].Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
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
	scanner.Buffer(make([]byte, 1024*1024), 10*1024*1024) // 1MB initial, 10MB max buffer size
	count := 0

	for scanner.Scan() {
		count++
	}

	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	return count, nil
}

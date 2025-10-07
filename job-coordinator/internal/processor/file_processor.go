package processor

import (
	"bufio"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dqme/job-coordinator/internal/storage"
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
	// Check if this is a cloud storage URL
	if isCloudStoragePath(filePath) {
		return openCloudStorageFile(filePath)
	}

	// Local file access
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

// openCloudStorageFile opens a file from cloud storage (gs://, s3://, file://)
// Handles gzip decompression automatically via storage.FileReader
func openCloudStorageFile(fileURL string) (io.ReadCloser, error) {
	ctx := context.Background()
	
	// Parse the URL to get base path and file key
	// For gs://bucket/path/to/file.ndjson, we need to extract bucket and file path
	scheme, bucket, key, err := storage.ParseStorageURL(fileURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse storage URL: %w", err)
	}

	// Reconstruct base URL (without the file key)
	baseURL := fmt.Sprintf("%s://%s", scheme, bucket)
	
	// Create storage reader
	reader, err := storage.NewReader(ctx, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create storage reader: %w", err)
	}

	// Open the specific file
	fileReader, err := reader.OpenFile(key)
	if err != nil {
		reader.Close()
		return nil, fmt.Errorf("failed to open file in storage: %w", err)
	}

	// Note: We need a combined closer that closes both the file and the reader
	return &storageReadCloser{reader: fileReader, bucket: reader}, nil
}

// storageReadCloser wraps both the file reader and bucket to close both
type storageReadCloser struct {
	reader io.ReadCloser
	bucket *storage.Reader
}

func (s *storageReadCloser) Read(p []byte) (n int, err error) {
	return s.reader.Read(p)
}

func (s *storageReadCloser) Close() error {
	readerErr := s.reader.Close()
	bucketErr := s.bucket.Close()
	if readerErr != nil {
		return readerErr
	}
	return bucketErr
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

package storage

import (
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"strings"
)

// StorageFileReader adapts storage.Reader to provide file reading capabilities
// similar to the existing file_processor.go but using cloud storage
type StorageFileReader struct {
	reader *Reader
	ctx    context.Context
}

// NewStorageFileReader creates a new storage-based file reader
func NewStorageFileReader(ctx context.Context, basePath string) (*StorageFileReader, error) {
	reader, err := NewReader(ctx, basePath)
	if err != nil {
		return nil, err
	}

	return &StorageFileReader{
		reader: reader,
		ctx:    ctx,
	}, nil
}

// OpenFile opens a file and returns an io.ReadCloser
// Automatically handles gzip decompression for .gz files
func (s *StorageFileReader) OpenFile(key string) (io.ReadCloser, error) {
	rc, err := s.reader.OpenFile(key)
	if err != nil {
		return nil, err
	}

	// Auto-decompress gzip files
	if isGzipped(key) {
		gzReader, err := gzip.NewReader(rc)
		if err != nil {
			rc.Close()
			return nil, fmt.Errorf("failed to create gzip reader: %w", err)
		}
		return &gzipStorageReadCloser{
			gzReader: gzReader,
			storage:  rc,
		}, nil
	}

	return rc, nil
}

// ListFiles lists all files matching the pattern
func (s *StorageFileReader) ListFiles(pattern string) ([]string, error) {
	return s.reader.List(pattern)
}

// Close closes the storage reader
func (s *StorageFileReader) Close() error {
	return s.reader.Close()
}

// isGzipped checks if a file is gzipped based on its extension
func isGzipped(filePath string) bool {
	return strings.HasSuffix(strings.ToLower(filePath), ".gz")
}

// gzipStorageReadCloser wraps both gzip.Reader and storage ReadCloser
type gzipStorageReadCloser struct {
	gzReader *gzip.Reader
	storage  io.ReadCloser
}

func (g *gzipStorageReadCloser) Read(p []byte) (n int, err error) {
	return g.gzReader.Read(p)
}

func (g *gzipStorageReadCloser) Close() error {
	gzErr := g.gzReader.Close()
	storageErr := g.storage.Close()
	if gzErr != nil {
		return gzErr
	}
	return storageErr
}

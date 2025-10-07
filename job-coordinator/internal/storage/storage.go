package storage

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"strings"

	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob"  // Register file:// URLs
	_ "gocloud.dev/blob/gcsblob"   // Register gs:// URLs
	_ "gocloud.dev/blob/s3blob"    // Register s3:// URLs
)

// Reader provides unified interface for reading files from various storage backends
type Reader struct {
	bucket *blob.Bucket
	ctx    context.Context
}

// NewReader creates a new storage reader from a URL or file path
// Supports:
//   - Local files: file:///path/to/dir or /absolute/path or C:\absolute\path
//   - GCS: gs://bucket-name
//   - S3: s3://bucket-name
func NewReader(ctx context.Context, pathOrURL string) (*Reader, error) {
	bucketURL, err := normalizePath(pathOrURL)
	if err != nil {
		return nil, fmt.Errorf("failed to normalize path: %w", err)
	}

	bucket, err := blob.OpenBucket(ctx, bucketURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open bucket at %s: %w", bucketURL, err)
	}

	return &Reader{
		bucket: bucket,
		ctx:    ctx,
	}, nil
}

// OpenFile opens a file for reading from the storage backend
// The key should be the relative path within the bucket/directory
func (r *Reader) OpenFile(key string) (io.ReadCloser, error) {
	reader, err := r.bucket.NewReader(r.ctx, key, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", key, err)
	}
	return reader, nil
}

// List returns all files matching the given pattern (supports * wildcards)
// If pattern is empty, returns all files
func (r *Reader) List(pattern string) ([]string, error) {
	var files []string
	
	iter := r.bucket.List(&blob.ListOptions{
		Prefix: extractPrefix(pattern),
	})

	for {
		obj, err := iter.Next(r.ctx)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list files: %w", err)
		}

		// Skip directories
		if obj.IsDir {
			continue
		}

		// Apply pattern matching if specified
		if pattern != "" {
			matched, err := filepath.Match(pattern, obj.Key)
			if err != nil {
				return nil, fmt.Errorf("invalid pattern %s: %w", pattern, err)
			}
			if !matched {
				continue
			}
		}

		files = append(files, obj.Key)
	}

	return files, nil
}

// Close closes the storage reader and releases resources
func (r *Reader) Close() error {
	return r.bucket.Close()
}

// normalizePath converts various path formats to gocloud blob URLs
func normalizePath(pathOrURL string) (string, error) {
	// Already a URL with scheme
	if strings.Contains(pathOrURL, "://") {
		u, err := url.Parse(pathOrURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		
		// Validate supported schemes
		switch u.Scheme {
		case "file", "gs", "s3":
			return pathOrURL, nil
		default:
			return "", fmt.Errorf("unsupported URL scheme: %s (supported: file://, gs://, s3://)", u.Scheme)
		}
	}

	// Local filesystem path - convert to file:// URL
	absPath, err := filepath.Abs(pathOrURL)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Convert to file:// URL format
	// Windows: C:\path -> file:///C:/path
	// Unix: /path -> file:///path
	fileURL := "file:///" + filepath.ToSlash(absPath)
	
	return fileURL, nil
}

// extractPrefix extracts the non-wildcard prefix from a pattern for efficient listing
func extractPrefix(pattern string) string {
	if pattern == "" {
		return ""
	}
	
	// Find first wildcard character
	wildcardPos := strings.IndexAny(pattern, "*?[]")
	if wildcardPos == -1 {
		// No wildcards - return as-is
		return pattern
	}
	
	// Return everything before the wildcard
	prefix := pattern[:wildcardPos]
	
	// If the wildcard is mid-filename, return up to the last directory separator
	lastSlash := strings.LastIndex(prefix, "/")
	if lastSlash != -1 {
		return prefix[:lastSlash+1]
	}
	
	return ""
}

// ParseStorageURL parses a storage URL and returns the scheme, bucket/base, and any path
func ParseStorageURL(pathOrURL string) (scheme, bucket, path string, err error) {
	if !strings.Contains(pathOrURL, "://") {
		// Local filesystem path
		absPath, err := filepath.Abs(pathOrURL)
		if err != nil {
			return "", "", "", fmt.Errorf("failed to get absolute path: %w", err)
		}
		return "file", absPath, "", nil
	}

	u, err := url.Parse(pathOrURL)
	if err != nil {
		return "", "", "", fmt.Errorf("invalid URL: %w", err)
	}

	return u.Scheme, u.Host, strings.TrimPrefix(u.Path, "/"), nil
}

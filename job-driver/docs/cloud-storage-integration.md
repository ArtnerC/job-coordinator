# Cloud Storage Integration Guide

This guide shows how to use the cloud storage support in the job driver. The driver supports Google Cloud Storage (GCS), Amazon S3, and local filesystems through a unified API powered by [gocloud.dev](https://gocloud.dev).

## Quick Start

The storage package is already available and can be used with minimal changes to existing code.

### Example: Reading from Google Cloud Storage

```bash
# Set credentials (if not using default)
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account.json

# Run with GCS path (supports gzip compression)
./driver \
  --base-path=gs://my-bucket/fhir-bundles \
  --measures-path=/local/measures \
  --batch-size=1000 \
  --distributor-type=pubsub \
  --distributor-config=project_id=my-project,topic_name=work-units \
  --auto-start=true
```

Note: Files ending in `.ndjson.gz` are automatically decompressed using the composable multiCloser pattern.

### Example: Reading from S3

```bash
# Set AWS credentials
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE
export AWS_SECRET_ACCESS_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

# Run with S3 path
./driver \
  --base-path=s3://my-bucket/fhir-bundles \
  --measures-path=/local/measures \
  --distributor-type=stdout \
  --auto-start=true
```

### Example: Local File URLs

```bash
# Using file:// URL scheme
./driver \
  --base-path=file:///data/bundles \
  --measures-path=/data/measures \
  --distributor-type=stdout \
  --auto-start=true
```

### Example: Mixed Storage (GCS + Local)

```bash
# Bundles from GCS, measures from local filesystem
./driver \
  --base-path=gs://my-bucket/fhir-bundles \
  --measures-path=/local/path/measures \
  --distributor-type=stdout \
  --auto-start=true
```

## Implementation Options

The storage package provides two ways to integrate:

### Option 1: Low-Level API (Manual)

Use `storage.Reader` directly for maximum control:

```go
package main

import (
    "bufio"
    "context"
    "github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/storage"
    "github.com/cvs-health-source-code/digital-qme-system/job-driver/internal/processor"
)

func processFiles(basePath string) error {
    ctx := context.Background()
    
    // Create cloud storage reader (supports file://, gs://, or s3://)
    reader, err := storage.NewCloudStorageReader(ctx, basePath)
    if err != nil {
        return err
    }
    defer reader.Close()
    
    // Discover NDJSON files (including .ndjson.gz)
    files, err := processor.DiscoverFiles(basePath, nil)
    if err != nil {
        return err
    }
    
    // Process each file with automatic gzip decompression
    for _, file := range files {
        // openFileReader handles both plain and gzipped files
        rc, err := processor.OpenFileReader(file)
        if err != nil {
            return err
        }
        
        // Process file contents (supports lines up to 10MB)
        scanner := bufio.NewScanner(rc)
        buf := make([]byte, 1024*1024) // 1MB initial buffer
        scanner.Buffer(buf, 10*1024*1024) // 10MB max buffer
        
        for scanner.Scan() {
            line := scanner.Text()
            // Process FHIR bundle line...
        }
        
        rc.Close()
    }
    
    return nil
}
```

### Option 2: High-Level API (Recommended)

Use `storage.StorageFileReader` which includes gzip support:

```go
package main

import (
    "context"
    "bufio"
    "github.com/dqme/job-driver/internal/storage"
)

func processFilesWithGzip(basePath string) error {
    ctx := context.Background()
    
    // Create reader with automatic gzip support
    reader, err := storage.NewStorageFileReader(ctx, basePath)
    if err != nil {
        return err
    }
    defer reader.Close()
    
    // List files
    files, err := reader.ListFiles("*.ndjson")
    if err != nil {
        return err
    }
    
    // Process each file (automatically handles .gz decompression)
    for _, file := range files {
        rc, err := reader.OpenFile(file)
        if err != nil {
            return err
        }
        
        scanner := bufio.NewScanner(rc)
        for scanner.Scan() {
            line := scanner.Text()
            // Process line...
        }
        rc.Close()
    }
    
    return nil
}
```

## Updating Existing Code

### Current file_discovery.go

The existing `DiscoverFiles()` function works with local paths. Here's how to make it storage-aware:

```go
// Before (local only)
func DiscoverFiles(basePath string, manifestPath *string) ([]string, error) {
    if manifestPath != nil && *manifestPath != "" {
        return ParseManifest(*manifestPath)
    }
    return discoverFilesInDirectory(basePath)
}

// After (storage-aware)
func DiscoverFiles(ctx context.Context, basePath string, manifestPath *string) ([]string, error) {
    if manifestPath != nil && *manifestPath != "" {
        return ParseManifest(*manifestPath)
    }
    
    // Check if basePath is a cloud URL
    if strings.Contains(basePath, "://") {
        return discoverFilesFromStorage(ctx, basePath)
    }
    
    // Fall back to local filesystem
    return discoverFilesInDirectory(basePath)
}

func discoverFilesFromStorage(ctx context.Context, basePath string) ([]string, error) {
    reader, err := storage.NewReader(ctx, basePath)
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    // List all .ndjson and .gz files
    files, err := reader.List("")
    if err != nil {
        return nil, err
    }
    
    // Filter for supported file types
    var filtered []string
    for _, f := range files {
        lowerName := strings.ToLower(f)
        if strings.HasSuffix(lowerName, ".ndjson") || 
           strings.HasSuffix(lowerName, ".ndjson.gz") ||
           strings.HasSuffix(lowerName, ".gz") {
            filtered = append(filtered, f)
        }
    }
    
    if len(filtered) == 0 {
        return nil, fmt.Errorf("no .ndjson or .gz files found in %s", basePath)
    }
    
    return filtered, nil
}
```

### Current file_processor.go

The existing `CountLines()` function can be made storage-aware:

```go
// Add storage-aware version
func CountLinesFromStorage(ctx context.Context, basePath, relativePath string) (int, error) {
    reader, err := storage.NewStorageFileReader(ctx, basePath)
    if err != nil {
        return 0, err
    }
    defer reader.Close()
    
    rc, err := reader.OpenFile(relativePath)
    if err != nil {
        return 0, err
    }
    defer rc.Close()
    
    scanner := bufio.NewScanner(rc)
    count := 0
    for scanner.Scan() {
        count++
    }
    
    if err := scanner.Err(); err != nil {
        return 0, fmt.Errorf("error reading file: %w", err)
    }
    
    return count, nil
}
```

## Configuration Updates

### Command-Line Arguments

No changes needed! The `--base-path` and `--measures-path` arguments already accept any string, so cloud URLs work automatically.

### Environment Variables

Add documentation for cloud provider credentials:

```bash
# Google Cloud Storage
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/key.json
export GOOGLE_CLOUD_PROJECT=my-project-id

# Amazon S3
export AWS_REGION=us-east-1
export AWS_ACCESS_KEY_ID=your-key-id
export AWS_SECRET_ACCESS_KEY=your-secret-key

# Or use AWS profile
export AWS_PROFILE=production
```

## Testing

### Integration Tests with Cloud Storage

```go
func TestWithGCS(t *testing.T) {
    if os.Getenv("TEST_GCS_BUCKET") == "" {
        t.Skip("Skipping GCS test: TEST_GCS_BUCKET not set")
    }
    
    ctx := context.Background()
    basePath := os.Getenv("TEST_GCS_BUCKET") // e.g., "gs://test-bucket/data"
    
    reader, err := storage.NewReader(ctx, basePath)
    if err != nil {
        t.Fatalf("Failed to create reader: %v", err)
    }
    defer reader.Close()
    
    files, err := reader.List("*.ndjson")
    if err != nil {
        t.Fatalf("Failed to list files: %v", err)
    }
    
    if len(files) == 0 {
        t.Error("Expected files in bucket")
    }
}
```

### Local Testing

The storage package works seamlessly with local files, so existing tests continue to work:

```go
func TestWithLocal(t *testing.T) {
    ctx := context.Background()
    
    // Works with absolute paths
    reader, err := storage.NewReader(ctx, "/path/to/testdata")
    
    // Works with relative paths (converted to absolute)
    reader, err := storage.NewReader(ctx, "testdata/bundles")
    
    // Works with file:// URLs
    reader, err := storage.NewReader(ctx, "file:///path/to/testdata")
}
```

## Performance Considerations

### Caching

For frequently accessed files, consider caching:

```go
type CachedReader struct {
    reader *storage.StorageFileReader
    cache  map[string][]byte
    mu     sync.RWMutex
}

func (c *CachedReader) ReadFile(key string) ([]byte, error) {
    c.mu.RLock()
    if data, ok := c.cache[key]; ok {
        c.mu.RUnlock()
        return data, nil
    }
    c.mu.RUnlock()
    
    rc, err := c.reader.OpenFile(key)
    if err != nil {
        return nil, err
    }
    defer rc.Close()
    
    data, err := io.ReadAll(rc)
    if err != nil {
        return nil, err
    }
    
    c.mu.Lock()
    c.cache[key] = data
    c.mu.Unlock()
    
    return data, nil
}
```

### Parallel Processing

The storage reader is safe for concurrent use:

```go
func processFilesParallel(ctx context.Context, basePath string, files []string) error {
    reader, err := storage.NewStorageFileReader(ctx, basePath)
    if err != nil {
        return err
    }
    defer reader.Close()
    
    var wg sync.WaitGroup
    errChan := make(chan error, len(files))
    
    for _, file := range files {
        wg.Add(1)
        go func(f string) {
            defer wg.Done()
            
            rc, err := reader.OpenFile(f)
            if err != nil {
                errChan <- err
                return
            }
            defer rc.Close()
            
            // Process file...
        }(file)
    }
    
    wg.Wait()
    close(errChan)
    
    for err := range errChan {
        return err
    }
    
    return nil
}
```

## Migration Checklist

- [ ] Add context.Context parameter to file discovery functions
- [ ] Update DiscoverFiles() to detect and handle cloud URLs
- [ ] Add storage-aware versions of file processing functions
- [ ] Update tests to handle both local and cloud storage
- [ ] Add integration tests for GCS and S3 (optional, can be skipped in CI)
- [ ] Update README with cloud storage examples
- [ ] Document environment variables for cloud credentials
- [ ] Add monitoring for cloud API errors and retries

## Backward Compatibility

The storage package is designed to maintain backward compatibility:

- ✅ Existing local file paths work without changes
- ✅ Existing tests continue to pass
- ✅ New cloud storage URLs are opt-in
- ✅ No breaking API changes required

## Future Enhancements

- Azure Blob Storage support
- Write operations (currently read-only)
- Progress callbacks for large files
- Configurable retry policies
- Custom timeout settings
- Bandwidth throttling
- Checksum verification


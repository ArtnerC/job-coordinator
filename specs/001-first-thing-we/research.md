# Research: Job Coordinator Service

**Feature**: Job Coordinator Service  
**Date**: 2025-10-03  
**Status**: Complete

## Research Questions

### 1. Go Configuration Management: CLI Args vs Environment Variables

**Decision**: Use `github.com/spf13/viper` + `pflag` for unified config management

**Rationale**:
- Viper supports both env vars and command-line flags with precedence rules (flags > env > defaults)
- Integrates seamlessly with cobra for CLI if needed later
- Automatic binding of flags to env vars (e.g., `--batch-size` → `BATCH_SIZE`)
- Built-in config file support for future extensibility
- Wide adoption in Go cloud-native projects

**Alternatives Considered**:
- `flag` package: Too basic, manual env var handling, no precedence management
- `cobra` + manual env parsing: More boilerplate, harder to maintain
- `envconfig`: Env-only, no flag support

**Implementation**: Create `internal/config/config.go` with viper-based Config struct

---

### 2. Efficient NDJSON Line Counting Without Full File Load

**Decision**: Use `bufio.Scanner` with line-by-line counting

**Rationale**:
- `bufio.Scanner` streams file reading, minimal memory usage
- Can count millions of lines without loading full file into memory
- Standard library, no external dependencies
- Performance: ~100-200MB/s on typical hardware
- Can be run concurrently across multiple files

**Alternatives Considered**:
- `bytes.Count()` after `ioutil.ReadFile()`: Loads entire file, memory intensive
- `wc -l` via exec: External dependency, portability issues
- Memory-mapped files: Overkill for line counting, platform-specific

**Implementation**:
```go
func countLines(filePath string) (int, error) {
    file, err := os.Open(filePath)
    if err != nil {
        return 0, err
    }
    defer file.Close()
    
    scanner := bufio.NewScanner(file)
    count := 0
    for scanner.Scan() {
        count++
    }
    return count, scanner.Err()
}
```

---

### 3. Batch Splitting Algorithm for Uneven Remainders

**Decision**: Threshold-based splitting with 20% threshold

**Rationale**:
- If remainder < 20% of batch size: append to last batch (avoids tiny batches)
- If remainder >= 20% of batch size: create new batch
- Example: batch_size=500, file has 1050 lines
  - Creates 2 batches of 500
  - Remainder 50 lines (10%) → append to batch 2 (becomes 550 lines)
- Example: batch_size=500, file has 1600 lines
  - Creates 3 batches of 500
  - Remainder 100 lines (20%) → new batch 4 with 100 lines
- Prevents worker starvation while avoiding excessive tiny batches

**Alternatives Considered**:
- Always create new batch: Results in many tiny batches (inefficient)
- Always append to last: Can create very large final batches
- Fixed threshold (e.g., 100 lines): Not scalable across batch sizes

**Implementation**: Configurable threshold (default 0.2), documented in config

---

### 4. GCP Pub/Sub Client Best Practices

**Decision**: Use `cloud.google.com/go/pubsub` official SDK with batch publishing

**Rationale**:
- Official Google SDK, well-maintained
- Built-in retry logic and error handling
- Batch publishing for efficiency (publish up to 1000 messages at once)
- Context-based cancellation support
- Automatic connection pooling

**Key Patterns**:
- Create single `pubsub.Client` instance, reuse across all publishes
- Use `topic.Publish()` with context for timeout control
- Batch work units into groups of 100-500 for publishing efficiency
- Monitor `PublishResult` for errors and retry failed messages
- Gracefully drain pending publishes on shutdown

**Error Handling**:
- Transient errors (network): Automatic retry by SDK
- Permanent errors (invalid topic): Fail fast with clear error
- Quota exceeded: Implement exponential backoff

**Implementation**: `internal/distributor/pubsub.go` with batched publishing

---

### 5. REST API Framework for Job Control Endpoints

**Decision**: Use Go `net/http` with `gorilla/mux` for routing

**Rationale**:
- Lightweight, standard library basis
- `gorilla/mux` adds route variables, middleware support
- No heavy framework overhead (not needed for 4 endpoints)
- Easy to test with `httptest` package
- Fast startup time (critical for short-lived coordinator instances)

**API Design**:
```
POST   /job/start      - Start pending job
PUT    /job/pause      - Pause running job
PUT    /job/cancel     - Cancel job (cleanup)
GET    /job/status     - Get job status
```

**Alternatives Considered**:
- `gin`: Overkill for simple API, more dependencies
- `echo`: Similar to mux, no significant advantage
- `chi`: Similar to mux, less ecosystem support

**Implementation**: `internal/api/handlers.go` + `router.go` with middleware for logging

---

### 6. Concurrent File Processing Strategy

**Decision**: Worker pool pattern with `sync.WaitGroup` and bounded concurrency

**Rationale**:
- Process multiple files concurrently for speed
- Bound concurrency to avoid resource exhaustion (default 10 workers)
- Use worker pool with job channel for file paths
- Each worker: count lines → split batches → send to distributor
- Graceful error handling: collect errors, don't fail fast

**Implementation Pattern**:
```go
sem := make(chan struct{}, maxWorkers)
var wg sync.WaitGroup
for _, file := range files {
    wg.Add(1)
    sem <- struct{}{} // acquire
    go func(f string) {
        defer wg.Done()
        defer func() { <-sem }() // release
        processFile(f)
    }(file)
}
wg.Wait()
```

**Alternatives Considered**:
- Sequential processing: Too slow for thousands of files
- Unbounded concurrency: Risk of resource exhaustion
- `errgroup`: Good but overkill for our error handling needs

---

### 7. Scale Testing Mode Implementation

**Decision**: Cycle through files with modulo-based indexing

**Rationale**:
- If target count > available lines: `fileIndex = workUnitIndex % len(files)`
- Reuses same files, adjusts line ranges to simulate unique work
- Memory efficient: doesn't duplicate file content
- Clear to debug: work unit shows which file + which lines

**Example**:
- 3 files with 1000 lines each (3000 total)
- Scale test target: 10000 items
- First 3000: files 0-2 as normal
- Items 3001-6000: cycle through files 0-2 again with offset line ranges
- Items 6001-9000: cycle again
- Items 9001-10000: partial cycle

**Implementation**: Add `--scale-test-count` flag, modify work unit generation logic

---

### 8. Job Completion Detection Strategies

**Decision**: Strategy pattern based on distributor type

**Rationale**:
- **Pub/Sub**: Monitor topic message count via `topic.GetSubscriptions()` metrics
  - Caveat: May need separate monitoring subscription or use approximate metrics
  - Alternative: Track publish count, rely on external job executor to report completion
- **Stdout**: Completion = all work units written to stdout
- **File**: Completion = file write complete + file closed

**Implementation**: Each distributor implements `IsComplete() bool` method

**Pub/Sub Consideration**: 
- Queue exhaustion is hard to detect reliably in real-time
- Better approach: Coordinator tracks expected work unit count, external executors report progress
- Phase 2 decision: Add work completion reporting via separate channel or status DB

---

### 9. Graceful Shutdown with TTL

**Decision**: Context-based shutdown with configurable grace period

**Rationale**:
- After job completes, start TTL timer (default 10 min)
- During TTL: keep REST API running for status queries
- On TTL expiry or SIGTERM: graceful shutdown sequence:
  1. Stop accepting new requests
  2. Wait for in-flight API requests (5s timeout)
  3. Close Pub/Sub client (drain pending publishes)
  4. Exit cleanly

**Implementation**:
```go
ctx, cancel := context.WithCancel(context.Background())
// On job complete:
time.AfterFunc(ttl, func() {
    cancel() // trigger shutdown
})
```

**Alternatives Considered**:
- Abrupt exit: Could lose in-flight publishes
- Indefinite wait: Wastes resources
- External termination only: Requires manual cleanup

---

### 10. Manifest File Format

**Decision**: Plain text file with one relative file path per line

**Rationale**:
- Simple to generate: `find . -name "*.ndjson" > manifest.txt`
- Easy to review and edit manually
- Standard Unix format
- Can be generated by external tools or scripts
- Future: Support JSON manifest with metadata if needed

**Example**:
```
patients/2023/jan/bundle-001.ndjson
patients/2023/jan/bundle-002.ndjson
patients/2023/feb/bundle-003.ndjson
```

**Implementation**: `internal/processor/manifest.go` with line-by-line parsing

---

### 11. Measures Configuration

**Decision**: Similar to files - manifest or auto-discovery

**Rationale**:
- Measures path provided via config
- Optional measures manifest: list specific measures to include
- If no manifest: list all files in measures path
- Each work unit includes full list of applicable measures
- Measures are opaque to coordinator (just file paths to pass along)

**Future Enhancement**: Support measure filtering/selection per work unit

---

### 12. Work Unit Schema

**Decision**: JSON structure with all required execution context

```json
{
  "job_id": "uuid-or-env-var",
  "work_unit_id": "uuid",
  "file_path": "relative/path/to/bundle.ndjson",
  "start_line": 0,
  "end_line": 499,
  "total_lines": 500,
  "measures": [
    "measures/measure-001.json",
    "measures/measure-002.json"
  ],
  "base_path": "/data/bundles",
  "created_at": "2025-10-03T10:00:00Z"
}
```

**Rationale**:
- Self-contained: executor has all info needed
- `start_line`/`end_line`: 0-indexed, inclusive range
- `base_path`: allows executor to construct full file path
- `measures`: list of measure files to apply
- Timestamps for tracking and debugging

**Implementation**: `internal/models/workunit.go` with JSON marshaling

---

## Technology Stack Summary

| Component | Technology | Reason |
|-----------|-----------|--------|
| Language | Go 1.21+ | Fast, excellent concurrency, GCP SDK support |
| Config | viper + pflag | Unified flags/env handling, precedence |
| HTTP | gorilla/mux | Lightweight routing, middleware support |
| Pub/Sub | cloud.google.com/go/pubsub | Official SDK, batch publishing |
| Testing | testing + testify | Standard + assertions library |
| Logging | slog (Go 1.21+) | Structured logging, stdlib |
| Containerization | Docker multi-stage | Small images, fast builds |
| Deployment | Cloud Run + Terraform | Serverless scale, IaC |

---

## Open Questions / Future Enhancements

1. **Work Unit Completion Tracking**: How do executors report completion back to coordinator?
   - Phase 2: Consider separate status queue or shared state store
   
2. **Retries**: Should coordinator track and retry failed work units?
   - Current: No, delegated to executors
   - Future: Optional retry queue

3. **Observability**: What metrics to export?
   - Work units created/published
   - File processing rate
   - API request latencies
   - Consider Prometheus metrics export

4. **Multi-tenancy**: Support multiple jobs in future?
   - Current: Single job per instance (Cloud Run scales instances)
   - Future: Job queue + scheduler

5. **Persistent State**: When to add database?
   - Current: In-memory only
   - Phase 2: Firestore for job status persistence

---

## Research Checklist

- [x] Go configuration management strategy
- [x] Efficient file line counting approach
- [x] Batch splitting algorithm
- [x] GCP Pub/Sub SDK and patterns
- [x] REST API framework selection
- [x] Concurrent file processing strategy
- [x] Scale testing implementation
- [x] Job completion detection per distributor
- [x] Graceful shutdown with TTL
- [x] Manifest file format
- [x] Measures configuration approach
- [x] Work unit data schema
- [x] Technology stack finalized

**Status**: All technical decisions made, ready for Phase 1 design

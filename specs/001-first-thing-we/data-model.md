# Data Model: Job Coordinator Service

**Feature**: Job Coordinator Service  
**Date**: 2025-10-03  
**Status**: Complete

## Core Entities

### Job

Represents a complete work distribution request from initialization to completion.

**Fields**:
- `ID` (string): Unique job identifier (UUID or from env var)
- `Status` (JobStatus): Current job state
- `Config` (JobConfig): Configuration snapshot for this job
- `WorkUnits` ([]WorkUnit): List of all work units for tracking
- `StartTime` (time.Time): Job start timestamp
- `EndTime` (*time.Time): Job completion timestamp (nil if not complete)
- `CompletionTTL` (time.Duration): How long to keep running after completion
- `TotalWorkUnits` (int): Total count of work units generated
- `ProcessedCount` (int): Count of work units distributed
- `ErrorCount` (int): Count of work units that failed distribution
- `Errors` ([]error): Collection of distribution errors

**Job Status Enum**:
```go
type JobStatus string

const (
    JobStatusPending    JobStatus = "pending"     // Job created, not started
    JobStatusRunning    JobStatus = "running"     // Job actively distributing work
    JobStatusPaused     JobStatus = "paused"      // Job temporarily paused
    JobStatusCompleted  JobStatus = "completed"   // All work distributed successfully
    JobStatusFailed     JobStatus = "failed"      // Job failed due to errors
    JobStatusCancelled  JobStatus = "cancelled"   // Job cancelled by user
)
```

**State Transitions**:
```
Pending → Running (on /job/start or auto-start)
Running → Paused (on /job/pause)
Paused → Running (on /job/start)
Running → Completed (when all work distributed)
Running → Failed (on critical error)
Running/Paused → Cancelled (on /job/cancel)
Completed → [TTL expires] → Shutdown
```

**Validation Rules**:
- ID must be non-empty
- Status must be valid enum value
- StartTime required once status is Running
- EndTime required once status is terminal (Completed/Failed/Cancelled)
- TotalWorkUnits must be non-negative

**Relationships**:
- Has many WorkUnits (1:N)
- References JobConfig (composition)

---

### WorkUnit

Represents a single discrete unit of work within a job.

**Fields**:
- `ID` (string): Unique work unit identifier (UUID)
- `JobID` (string): Parent job identifier
- `FilePath` (string): Relative path to NDJSON file
- `StartLine` (int): Starting line index (0-based, inclusive)
- `EndLine` (int): Ending line index (0-based, inclusive). Set to sentinel value (e.g., math.MaxInt32) in whole-file mode
- `TotalLines` (int): Number of lines in this unit (EndLine - StartLine + 1). Set to sentinel value in whole-file mode
- `Measures` ([]string): List of measure file paths to apply (mutually exclusive with MeasuresFolderPath)
- `MeasuresFolderPath` (*string): Relative path to folder containing all measures (mutually exclusive with Measures)
- `BasePath` (string): Base directory path for resolving relative paths
- `CreatedAt` (time.Time): Work unit creation timestamp
- `Status` (WorkUnitStatus): Current distribution status
- `Error` (*string): Error message if distribution failed

**Work Unit Status Enum**:
```go
type WorkUnitStatus string

const (
    WorkUnitStatusPending       WorkUnitStatus = "pending"      // Created, not yet distributed
    WorkUnitStatusDistributing  WorkUnitStatus = "distributing" // Being sent to destination
    WorkUnitStatusDistributed   WorkUnitStatus = "distributed"  // Successfully sent
    WorkUnitStatusFailed        WorkUnitStatus = "failed"       // Distribution failed
)
```

**Validation Rules**:
- ID and JobID must be non-empty
- FilePath must be non-empty and relative (no leading `/`)
- StartLine >= 0
- EndLine >= StartLine
- TotalLines must equal (EndLine - StartLine + 1), except in whole-file mode where sentinel values are used
- Exactly one of Measures or MeasuresFolderPath must be set (mutually exclusive)
- If Measures provided, can be empty array (if no measures configured)
- If MeasuresFolderPath provided, must be non-empty relative path
- BasePath must be absolute path

**JSON Schema** (for serialization to Pub/Sub/file/stdout):
```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["job_id", "work_unit_id", "file_path", "start_line", "end_line", "measures", "base_path"],
  "properties": {
    "job_id": { "type": "string" },
    "work_unit_id": { "type": "string" },
    "file_path": { "type": "string" },
    "start_line": { "type": "integer", "minimum": 0 },
    "end_line": { "type": "integer", "minimum": 0 },
    "total_lines": { "type": "integer", "minimum": 1 },
    "measures": {
      "type": "array",
      "items": { "type": "string" }
    },
    "base_path": { "type": "string" },
    "created_at": { "type": "string", "format": "date-time" }
  }
}
```

**Relationships**:
- Belongs to Job (N:1)

---

### JobConfig

Configuration snapshot for a job execution.

**Fields**:
- `JobID` (string): Job ID from env var or auto-generated
- `AutoStart` (bool): Whether job starts automatically or waits for /job/start
- `BatchSize` (*int): Lines per work unit (default 500). If 0 or nil, entire file becomes one work unit
- `RemainderThreshold` (float64): Threshold for appending remainder to last batch (default 0.2). Ignored when BatchSize=0
- `BasePath` (string): Base directory containing patient bundle files
- `ManifestPath` (*string): Optional path to file manifest (nil for auto-discovery)
- `MeasuresPath` (*string): Optional directory containing measure definitions (nil if using MeasuresManifestPath or measures_folder_path in work units)
- `MeasuresManifestPath` (*string): Optional path to measures manifest (nil for auto-discovery or folder path mode)
- `UseMeasuresFolderPath` (bool): If true, work units include measures_folder_path instead of measures array
- `DistributorType` (string): Type of distributor (pubsub, stdout, file)
- `DistributorConfig` (map[string]string): Distributor-specific configuration
- `CompletionTTL` (time.Duration): Time to keep running after completion (default 10min)
- `ConcurrentFileProcessors` (int): Max concurrent file processors (default 10)
- `ScaleTestCount` (*int): Optional scale test target count (nil for normal operation)

**Batch Size Modes**:

1. **Standard Mode** (`BatchSize > 0`, default 500):
   - Files are split into work units of `BatchSize` lines each
   - File line counting is performed
   - Remainder threshold logic applies for last batch

2. **Whole File Mode** (`BatchSize = 0` or `nil`):
   - Each file becomes exactly one work unit
   - Line counting is skipped for performance
   - `start_line = 0`, `end_line` set to large sentinel value (e.g., `math.MaxInt32`)
   - `total_lines` set to sentinel value or unknown indicator
   - Executor reads entire file without line range constraints

**Validation Rules**:
- BatchSize must be >= 0 (0 means whole file mode)
- If BatchSize = 0, RemainderThreshold is ignored
- RemainderThreshold must be between 0.0 and 1.0
- BasePath must be absolute and exist
- MeasuresPath must be absolute and exist
- DistributorType must be one of: "pubsub", "stdout", "file"
- CompletionTTL must be > 0
- ConcurrentFileProcessors must be > 0
- If ScaleTestCount provided, must be > 0

**Distributor-Specific Config**:

**Pub/Sub**:
```go
{
    "project_id": "my-gcp-project",
    "topic_id": "work-queue",
}
```

**Stdout**:
```go
{
    // No additional config required
}
```

**File**:
```go
{
    "output_path": "/output/work-units.ndjson",
}
```

---

### FileManifest

Represents the list of files to process.

**Fields**:
- `Files` ([]string): List of relative file paths
- `Source` (ManifestSource): How manifest was obtained

**Manifest Source Enum**:
```go
type ManifestSource string

const (
    ManifestSourceFile          ManifestSource = "file"          // Loaded from manifest file
    ManifestSourceAutoDiscovery ManifestSource = "auto-discovery" // Auto-discovered from directory
)
```

**Operations**:
- `LoadFromFile(path string) (*FileManifest, error)`: Load from manifest file
- `DiscoverFromDirectory(basePath string) (*FileManifest, error)`: Auto-discover files
- `Validate() error`: Validate all files exist and are readable

---

### MeasuresManifest

Represents the list of measures to include in work units.

**Fields**:
- `Measures` ([]string): List of relative measure file paths
- `Source` (ManifestSource): How manifest was obtained

**Operations**:
- `LoadFromFile(path string) (*MeasuresManifest, error)`: Load from manifest file
- `DiscoverFromDirectory(measuresPath string) (*MeasuresManifest, error)`: Auto-discover measures
- `Validate() error`: Validate all measures exist and are readable

---

### JobStatusResponse

API response for /job/status endpoint.

**Fields**:
- `JobID` (string): Job identifier
- `Status` (JobStatus): Current job state
- `StartTime` (time.Time): Job start timestamp
- `EndTime` (*time.Time): Job completion timestamp (nil if not complete)
- `TotalWorkUnits` (int): Total work units generated
- `DistributedCount` (int): Work units successfully distributed
- `FailedCount` (int): Work units that failed distribution
- `CompletionPercentage` (float64): Percentage complete (0-100)
- `TTLRemaining` (*time.Duration): Time remaining before shutdown (nil if job not complete)
- `Errors` ([]string): List of error messages

**JSON Example**:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "start_time": "2025-10-03T10:00:00Z",
  "end_time": null,
  "total_work_units": 1000,
  "distributed_count": 750,
  "failed_count": 0,
  "completion_percentage": 75.0,
  "ttl_remaining": null,
  "errors": []
}
```

---

## Entity Relationships Diagram

```
┌─────────────────┐
│   JobConfig     │
│  (config)       │
└─────────────────┘
         │
         │ 1:1
         ▼
┌─────────────────┐         ┌──────────────────┐
│      Job        │────────▶│   WorkUnit       │
│  (coordinator)  │   1:N   │  (work items)    │
└─────────────────┘         └──────────────────┘
         │
         │ uses
         ▼
┌─────────────────┐
│ FileManifest    │
│ (file list)     │
└─────────────────┘
         │
         │ uses
         ▼
┌─────────────────┐
│MeasuresManifest │
│ (measure list)  │
└─────────────────┘
```

---

## Data Flow

1. **Initialization**:
   - Load JobConfig from flags/env vars
   - Validate configuration
   - Generate or read Job ID
   - Create Job entity with Pending status

2. **File Discovery**:
   - Load or auto-discover FileManifest
   - Load or auto-discover MeasuresManifest
   - Validate all files exist

3. **Batch Generation**:
   - For each file in FileManifest:
     - Count lines (stream, no full load)
     - Split into batches based on BatchSize
     - Apply RemainderThreshold rule
     - Create WorkUnit for each batch
   - If ScaleTestCount: cycle through files to reach target count

4. **Distribution**:
   - Job status → Running
   - Process WorkUnits concurrently (bounded by ConcurrentFileProcessors)
   - For each WorkUnit:
     - Serialize to JSON
     - Send to Distributor (Pub/Sub / stdout / file)
     - Update WorkUnit status
   - Track progress: DistributedCount, FailedCount

5. **Completion**:
   - When all WorkUnits distributed:
     - Job status → Completed (or Failed if errors)
     - Set EndTime
     - Start TTL timer

6. **Shutdown**:
   - After TTL expires or on cancel:
     - Stop API server
     - Close Distributor connections
     - Exit process

---

## Concurrency Considerations

**Thread-Safe Operations**:
- Job status updates (use sync.RWMutex)
- WorkUnit status updates (use sync.Mutex per unit or channels)
- Progress counters (use atomic operations or channels)

**Concurrent Access Patterns**:
- Multiple goroutines reading Job status (read lock)
- Single coordinator goroutine updating Job status (write lock)
- Worker pool goroutines updating WorkUnit status (per-unit locks)
- API handlers reading Job status (read lock)

**Channel-Based Alternative** (recommended):
- Use channels for status updates instead of shared memory
- StatusUpdate channel for job-level updates
- WorkUnitUpdate channel for work unit completions
- Single goroutine consumes channels and updates state

---

## Persistence Strategy

**Phase 1** (Current):
- All data in-memory (no database)
- Data lost on coordinator shutdown
- Acceptable: short-lived coordinator instances, work distribution is idempotent

**Phase 2** (Future):
- Persist Job and WorkUnit state to Firestore
- Enable long-term job history queries
- Support coordinator restarts without losing state
- Allow separate status query service

---

## Data Model Checklist

- [x] Job entity defined with all required fields
- [x] WorkUnit entity defined with serialization schema
- [x] JobConfig entity defined with validation rules
- [x] FileManifest and MeasuresManifest defined
- [x] JobStatusResponse for API contract
- [x] State machine and transitions documented
- [x] Entity relationships defined
- [x] Data flow documented
- [x] Concurrency considerations addressed
- [x] Persistence strategy documented

**Status**: Data model complete, ready for contract design

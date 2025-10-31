# REST API Contract: Job Driver

**Version**: 1.0.0  
**Base Path**: `/`  
**Content-Type**: `application/json`

## Endpoints

### POST /job/start

Start a pending job or resume a paused job.

**Request**:
```http
POST /job/start HTTP/1.1
Content-Type: application/json
```

No request body required.

**Response** (200 OK):
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "message": "Job started successfully"
}
```

**Response** (400 Bad Request) - Job already running:
```json
{
  "error": "Job is already running"
}
```

**Response** (400 Bad Request) - Job in terminal state:
```json
{
  "error": "Cannot start job in completed state"
}
```

---

### PUT /job/pause

Pause a running job.

**Request**:
```http
PUT /job/pause HTTP/1.1
Content-Type: application/json
```

No request body required.

**Response** (200 OK):
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "paused",
  "message": "Job paused successfully"
}
```

**Response** (400 Bad Request) - Job not running:
```json
{
  "error": "Cannot pause job that is not running"
}
```

---

### PUT /job/cancel

Cancel a job (any non-terminal state).

**Request**:
```http
PUT /job/cancel HTTP/1.1
Content-Type: application/json
```

No request body required.

**Response** (200 OK):
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "cancelled",
  "message": "Job cancelled successfully"
}
```

**Response** (400 Bad Request) - Job already terminal:
```json
{
  "error": "Cannot cancel job in completed state"
}
```

---

### GET /job/status

Get current job status and progress.

**Request**:
```http
GET /job/status HTTP/1.1
```

**Response** (200 OK) - Job running:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "running",
  "start_time": "2025-10-03T10:00:00Z",
  "end_time": null,
  "total_work_units": 1000,
  "distributed_count": 750,
  "failed_count": 5,
  "completion_percentage": 75.5,
  "ttl_remaining": null,
  "errors": [
    "Failed to publish work unit wu-123: connection timeout",
    "Failed to publish work unit wu-456: quota exceeded"
  ]
}
```

**Response** (200 OK) - Job completed:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "completed",
  "start_time": "2025-10-03T10:00:00Z",
  "end_time": "2025-10-03T10:15:30Z",
  "total_work_units": 1000,
  "distributed_count": 995,
  "failed_count": 5,
  "completion_percentage": 100.0,
  "ttl_remaining": "9m30s",
  "errors": [
    "Failed to publish work unit wu-123: connection timeout",
    "Failed to publish work unit wu-456: quota exceeded"
  ]
}
```

**Response** (200 OK) - Job pending (auto_start=false):
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "status": "pending",
  "start_time": null,
  "end_time": null,
  "total_work_units": 0,
  "distributed_count": 0,
  "failed_count": 0,
  "completion_percentage": 0.0,
  "ttl_remaining": null,
  "errors": []
}
```

---

## Error Responses

All error responses follow this format:

```json
{
  "error": "Human-readable error message",
  "details": {
    "code": "ERROR_CODE",
    "timestamp": "2025-10-03T10:00:00Z"
  }
}
```

**Common HTTP Status Codes**:
- `200 OK`: Request successful
- `400 Bad Request`: Invalid request or invalid state transition
- `500 Internal Server Error`: Unexpected server error

---

## Health Check

### GET /health

Check if driver is running.

**Request**:
```http
GET /health HTTP/1.1
```

**Response** (200 OK):
```json
{
  "status": "healthy",
  "uptime": "15m30s"
}
```

---

### GET /config

Get the current job configuration.

**Request**:
```http
GET /config HTTP/1.1
```

**Response** (200 OK):
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "auto_start": true,
  "batch_size": 500,
  "remainder_threshold": 0.2,
  "base_path": "/data/bundles",
  "measures_path": "/data/measures",
  "measures_to_run": [
    "/data/measures/cms-125-v10.json",
    "/data/measures/cms-130-v10.json"
  ],
  "use_measures_path": false,
  "distributor_type": "pubsub",
  "distributor_config": {
    "project_id": "my-gcp-project",
    "topic_id": "work-queue"
  },
  "completion_ttl": "10m",
  "concurrent_file_processors": 10
}
```

**Use Cases**:
- Inspect current configuration
- Verify configuration before making changes
- Audit configuration for debugging

---

### PUT /config

Update job configuration. Only allowed when job is in `pending` state or for specific runtime-modifiable settings.

**Request**:
```http
PUT /config HTTP/1.1
Content-Type: application/json

{
  "batch_size": 1000,
  "completion_ttl": "15m"
}
```

**Response** (200 OK):
```json
{
  "message": "Configuration updated successfully",
  "updated_fields": ["batch_size", "completion_ttl"]
}
```

**Response** (400 Bad Request) - Invalid state:
```json
{
  "error": "Cannot modify batch_size while job is running. Only runtime-modifiable settings (completion_ttl) can be changed."
}
```

**Response** (400 Bad Request) - Invalid value:
```json
{
  "error": "batch_size must be >= 0"
}
```

**Modifiable Settings by Job State**:

| Setting | Pending | Running | Paused | Terminal States |
|---------|---------|---------|---------|-----------------|
| `batch_size` | ✅ | ✅* | ✅* | ❌ |
| `remainder_threshold` | ✅ | ✅* | ✅* | ❌ |
| `concurrent_file_processors` | ✅ | ✅* | ✅* | ❌ |
| `completion_ttl` | ✅ | ✅ | ✅ | ✅ |
| `distributor_type` | ✅ | ❌ | ❌ | ❌ |
| `distributor_config` | ✅ | ❌ | ❌ | ❌ |
| `use_measures_path` | ✅ | ❌ | ❌ | ❌ |
| `measures_to_run` | ✅ | ❌ | ❌ | ❌ |

**\*Graceful Runtime Changes**: When `batch_size`, `remainder_threshold`, or `concurrent_file_processors` are changed during `running` or `paused` states:
- Changes apply to new work units being generated
- In-flight work units and already-distributed work units are unaffected
- File processor pool gracefully adjusts to new `concurrent_file_processors` limit

**Runtime-Modifiable Settings** (any non-terminal state):
- `completion_ttl`: Can be adjusted to extend or shorten retention period
- `batch_size`: Applied to remaining files not yet processed (graceful)
- `remainder_threshold`: Applied to remaining files not yet processed (graceful)
- `concurrent_file_processors`: Worker pool adjusts gracefully

**Pre-Start Only Settings** (only in `pending` state):
- `distributor_type`: Cannot change distribution method after start
- `distributor_config`: Cannot change distributor config after start
- `use_measures_path`: Cannot toggle between measures array and path mode after start
- `measures_to_run`: Must set measure list before starting job

**Measures Configuration Validation**:
- `use_measures_path` can be changed from `true` to `false` only if `measures_to_run` is provided
- If `use_measures_path=false` and `measures_to_run` is empty, request returns 400 Bad Request

**Use Cases**:
- Adjust batch size before starting job based on file sizes
- Switch between measures array and folder path mode
- Change distributor configuration (e.g., different Pub/Sub topic)
- Extend TTL for long-running status monitoring

---

## State Transitions

Valid state transitions triggered by API endpoints:

```
POST /job/start:
  pending → running
  paused → running

PUT /job/pause:
  running → paused

PUT /job/cancel:
  pending → cancelled
  running → cancelled
  paused → cancelled

PUT /config:
  Any state → (same state, config updated)
  Note: Only certain config fields modifiable in non-pending states
```

Invalid transitions return `400 Bad Request`.

---

## API Testing Contract

Test scenarios to validate API compliance:

1. **Happy Path - Auto Start**:
   - driver starts with `auto_start=true`
   - GET /job/status returns `running`
   - Wait for completion
   - GET /job/status returns `completed` with TTL

2. **Happy Path - Manual Start**:
   - driver starts with `auto_start=false`
   - GET /job/status returns `pending`
   - POST /job/start
   - GET /job/status returns `running`
   - Wait for completion
   - GET /job/status returns `completed`

3. **Pause and Resume**:
   - Start job
   - PUT /job/pause
   - GET /job/status returns `paused`
   - POST /job/start
   - GET /job/status returns `running`

4. **Cancel Job**:
   - Start job
   - PUT /job/cancel
   - GET /job/status returns `cancelled`
   - Verify driver shuts down

5. **Invalid Transitions**:
   - Complete job
   - POST /job/start (should return 400)
   - PUT /job/pause (should return 400)

6. **Progress Tracking**:
   - Start job
   - Poll GET /job/status every 1s
   - Verify `completion_percentage` increases
   - Verify `distributed_count` increases
   - On completion, verify `ttl_remaining` decreases

7. **Configuration Inspection**:
   - GET /config at any time
   - Verify all config fields returned correctly

8. **Pre-Start Configuration Update**:
   - Start driver with `auto_start=false`
   - GET /config to verify initial settings
   - PUT /config with `{"batch_size": 0}` (whole-file mode)
   - Verify 200 OK response
   - GET /config to confirm batch_size=0
   - POST /job/start
   - Verify work units have sentinel values for line ranges

9. **Runtime Configuration Update**:
   - Start job (running state)
   - PUT /config with `{"completion_ttl": "20m"}`
   - Verify 200 OK response
   - PUT /config with `{"batch_size": 1000}` (should fail)
   - Verify 400 Bad Request response

10. **Configuration Validation**:
    - PUT /config with `{"batch_size": -1}`
    - Verify 400 Bad Request with validation error

---

## OpenAPI 3.0 Specification

Full OpenAPI spec available in `api-spec.yaml` (to be generated in Phase 1).

# REST API Documentation

The Job Driver exposes a REST API for job control and monitoring. All responses are JSON.

## Base URL

```
http://localhost:8080
```

(Configurable via `--api-port` flag)

## Endpoints

### POST /job/start

Start job processing.

**Request**:
```http
POST /job/start HTTP/1.1
```

**Response (200 OK)**:
```json
{
  "job_id": "my-job-123",
  "status": "running",
  "message": "Job started successfully"
}
```

**Response (400 Bad Request)** - Job already running:
```json
{
  "error": "cannot start job in status: running"
}
```

**Valid Transitions**:
- `pending` → `running`
- `paused` → `running`

---

### PUT /job/pause

Pause a running job.

**Request**:
```http
PUT /job/pause HTTP/1.1
```

**Response (200 OK)**:
```json
{
  "job_id": "my-job-123",
  "status": "paused",
  "message": "Job paused successfully"
}
```

**Response (400 Bad Request)**:
```json
{
  "error": "cannot pause job in status: pending"
}
```

**Valid Transitions**:
- `running` → `paused`

---

### PUT /job/cancel

Cancel the job.

**Request**:
```http
PUT /job/cancel HTTP/1.1
```

**Response (200 OK)**:
```json
{
  "job_id": "my-job-123",
  "status": "cancelled",
  "message": "Job cancelled successfully"
}
```

**Response (400 Bad Request)**:
```json
{
  "error": "cannot cancel job in status: completed"
}
```

**Valid Transitions**:
- `pending` → `cancelled`
- `running` → `cancelled`
- `paused` → `cancelled`

---

### GET /job/status

Get current job status and progress.

**Request**:
```http
GET /job/status HTTP/1.1
```

**Response (200 OK)**:
```json
{
  "job_id": "my-job-123",
  "status": "running",
  "total_work_units": 150,
  "processed_work_units": 75,
  "error_count": 2,
  "completion_percent": 50.0,
  "start_time": "2025-01-21T10:30:00Z",
  "end_time": null,
  "errors": [
    "Failed to process file: bundle1.ndjson",
    "Failed to process file: bundle2.ndjson"
  ]
}
```

**Fields**:
- `job_id`: Job identifier
- `status`: One of: `pending`, `running`, `paused`, `completed`, `failed`, `cancelled`
- `total_work_units`: Total work units created
- `processed_work_units`: Work units successfully processed
- `error_count`: Number of errors encountered
- `completion_percent`: Percentage of work units processed (0.0-100.0)
- `start_time`: Job start timestamp (ISO 8601)
- `end_time`: Job end timestamp (ISO 8601, null if not finished)
- `errors`: Array of error messages

---

### GET /config

Get current job configuration.

**Request**:
```http
GET /config HTTP/1.1
```

**Response (200 OK)**:
```json
{
  "job_id": "my-job-123",
  "auto_start": false,
  "batch_size": 0,
  "multifile_batches": true,
  "remainder_threshold": 0.2,
  "base_path": "/data/bundles",
  "manifest_path": null,
  "measures_path": "./measures",
  "measures_to_run": [
    "/measures/measure1.json",
    "/measures/measure2.json"
  ],
  "measures_manifest_path": null,
  "distributor_type": "stdout",
  "distributor_config": {},
  "completion_ttl": "10m0s",
  "concurrent_file_processors": 10,
  "scale_load": null,
  "api_port": 8080
}
```

---

### PUT /config

Update runtime-modifiable configuration fields.

**Request**:
```http
PUT /config HTTP/1.1
Content-Type: application/json

{
  "batch_size": 1000,
  "concurrent_file_processors": 20
}
```

**Response (200 OK)**:
```json
{
  "message": "Configuration updated successfully",
  "updated_fields": [
    "batch_size",
    "concurrent_file_processors"
  ]
}
```

**Response (400 Bad Request)** - Non-modifiable fields:
```json
{
  "error": "field 'distributor_type' cannot be modified in state 'running'"
}
```

**Runtime-Modifiable Fields** (in running/paused states):
- `batch_size` - Adjust batch size for remaining files
- `remainder_threshold` - Change threshold for remainder batches
- `concurrent_file_processors` - Adjust worker pool size
- `completion_ttl` - Update TTL (modifiable in any state including completed)

**Pre-Start Only Fields** (modifiable only in pending state):
- `distributor_type` - Cannot change distributor after job starts
- `distributor_config` - Cannot change distributor settings after job starts
- `measures_to_run` - Cannot change measures after job starts

**Immutable Fields** (cannot be modified after initialization):
- `job_id`, `base_path`, `manifest_path`, `measures_path`, `measures_manifest_path`, `auto_start`, `multifile_batches`, `scale_load`, `api_port`

---

### GET /health

Health check endpoint.

**Request**:
```http
GET /health HTTP/1.1
```

**Response (200 OK)**:
```json
{
  "status": "ok",
  "job_id": "my-job-123",
  "job_status": "running",
  "is_processing": true
}
```

**Use Cases**:
- Load balancer health checks
- Kubernetes liveness/readiness probes
- Monitoring system checks

---

## Status Codes

- `200 OK`: Request successful
- `400 Bad Request`: Invalid request or state transition
- `405 Method Not Allowed`: Wrong HTTP method
- `500 Internal Server Error`: Server error

## Error Response Format

All error responses follow this format:

```json
{
  "error": "error_code_or_message",
  "message": "Human-readable error description",
  "details": "Additional error details (optional)"
}
```

## Job Status Flow

```
pending → running → completed
    ↓         ↓
cancelled  paused → running
           ↓
       cancelled
```

Terminal states: `completed`, `failed`, `cancelled`

## Examples

### cURL Examples

**Start a job**:
```bash
curl -X POST http://localhost:8080/job/start
```

**Check status**:
```bash
curl http://localhost:8080/job/status
```

**Pause job**:
```bash
curl -X PUT http://localhost:8080/job/pause
```

**Update config**:
```bash
curl -X PUT http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"concurrent_file_processors": 20}'
```

### PowerShell Examples

**Start a job**:
```powershell
Invoke-RestMethod -Method Post -Uri http://localhost:8080/job/start
```

**Check status**:
```powershell
Invoke-RestMethod -Uri http://localhost:8080/job/status
```

**Update config**:
```powershell
$body = @{
    concurrent_file_processors = 20
} | ConvertTo-Json

Invoke-RestMethod -Method Put -Uri http://localhost:8080/config `
    -Body $body -ContentType "application/json"
```

## Rate Limiting

No rate limiting is currently implemented. In production deployments, consider adding rate limiting at the load balancer or API gateway level.

## Authentication

No authentication is currently implemented. For production deployments:
- Deploy behind a load balancer with authentication
- Use Cloud Run's built-in IAM authentication
- Add middleware for API key validation

## Monitoring

Recommended monitoring:
- `/health` endpoint for liveness checks (every 10s)
- `/job/status` endpoint for progress tracking
- Log aggregation for error tracking
- Metrics export (custom implementation needed)

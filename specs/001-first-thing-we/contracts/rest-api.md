# REST API Contract: Job Coordinator

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

Check if coordinator is running.

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
```

Invalid transitions return `400 Bad Request`.

---

## API Testing Contract

Test scenarios to validate API compliance:

1. **Happy Path - Auto Start**:
   - Coordinator starts with `auto_start=true`
   - GET /job/status returns `running`
   - Wait for completion
   - GET /job/status returns `completed` with TTL

2. **Happy Path - Manual Start**:
   - Coordinator starts with `auto_start=false`
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
   - Verify coordinator shuts down

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

---

## OpenAPI 3.0 Specification

Full OpenAPI spec available in `api-spec.yaml` (to be generated in Phase 1).

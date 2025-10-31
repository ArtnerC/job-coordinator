# Quickstart Guide: Job Driver Service

**Purpose**: Step-by-step guide to build, run, and test the Job Driver locally

## Prerequisites

- Go 1.21+ installed
- GCP account with Pub/Sub enabled (for Pub/Sub mode)
- Docker (optional, for containerized testing)
- Sample NDJSON patient bundle files
- Sample measure definition files

---

## Quick Start (5 minutes)

### 1. Clone and Build

```bash
cd job-driver
go mod download
go build -o bin/driver ./cmd/coordinator
```

### 2. Prepare Test Data

```bash
# Create test data directories
mkdir -p testdata/bundles testdata/measures

# Create a sample patient bundle (100 lines)
for i in {1..100}; do
  echo '{"resourceType":"Bundle","id":"patient-'$i'","entry":[]}' >> testdata/bundles/sample-001.ndjson
done

# Create a sample measure
echo '{"resourceType":"Measure","id":"cms-125","title":"Breast Cancer Screening"}' > testdata/measures/cms-125.json
```

### 3. Run with Stdout (easiest)

```bash
./bin/driver \
  --job-id=test-job-001 \
  --batch-size=20 \
  --base-path=$(pwd)/testdata/bundles \
  --measures-path=$(pwd)/testdata/measures \
  --distributor=stdout \
  --auto-start=true

# Output: Work units printed to stdout as NDJSON
```

Expected output:
```
{"job_id":"test-job-001","work_unit_id":"...","file_path":"sample-001.ndjson","start_line":0,"end_line":19,"total_lines":20,...}
{"job_id":"test-job-001","work_unit_id":"...","file_path":"sample-001.ndjson","start_line":20,"end_line":39,"total_lines":20,...}
...
```

---

## Configuration Options

### Command Line Flags

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `--job-id` | `JOB_ID` | (generated) | Unique job identifier |
| `--auto-start` | `AUTO_START` | true | Auto-start job or wait for /job/start |
| `--batch-size` | `BATCH_SIZE` | 500 | Lines per work unit |
| `--remainder-threshold` | `REMAINDER_THRESHOLD` | 0.2 | Threshold for appending remainder (0.0-1.0) |
| `--base-path` | `BASE_PATH` | (required) | Base directory for patient bundles |
| `--manifest-path` | `MANIFEST_PATH` | (optional) | Path to file manifest |
| `--measures-path` | `MEASURES_PATH` | (required) | Directory for measure definitions |
| `--measures-manifest-path` | `MEASURES_MANIFEST_PATH` | (optional) | Path to measures manifest |
| `--distributor` | `DISTRIBUTOR` | stdout | Distribution method (pubsub/stdout/file) |
| `--pubsub-project-id` | `PUBSUB_PROJECT_ID` | (for pubsub) | GCP project ID |
| `--pubsub-topic-id` | `PUBSUB_TOPIC_ID` | (for pubsub) | Pub/Sub topic name |
| `--file-output-path` | `FILE_OUTPUT_PATH` | (for file) | Output file path |
| `--completion-ttl` | `COMPLETION_TTL` | 10m | Time to keep running after completion |
| `--concurrent-processors` | `CONCURRENT_PROCESSORS` | 10 | Max concurrent file processors |
| `--scale-test-count` | `SCALE_TEST_COUNT` | (optional) | Scale test target count |
| `--api-port` | `API_PORT` | 8080 | REST API port |

### Precedence

Command line flags > Environment variables > Defaults

---

## Usage Scenarios

### Scenario 1: Local Development with Stdout

**Use Case**: Quick testing without GCP dependencies

```bash
./bin/driver \
  --batch-size=100 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=stdout > work-units.ndjson
```

**Validation**:
```bash
# Count work units
wc -l work-units.ndjson

# Inspect first work unit
head -n1 work-units.ndjson | jq .

# Validate JSON format
jq -c . work-units.ndjson > /dev/null && echo "Valid JSON"
```

---

### Scenario 2: GCP Pub/Sub Distribution

**Use Case**: Production deployment with cloud queue

**Setup**:
```bash
# Create Pub/Sub topic
gcloud pubsub topics create work-queue

# Create subscription for executors
gcloud pubsub subscriptions create work-queue-sub --topic=work-queue
```

**Run**:
```bash
./bin/driver \
  --batch-size=500 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=pubsub \
  --pubsub-project-id=my-project \
  --pubsub-topic-id=work-queue
```

**Validation**:
```bash
# Check message count in topic
gcloud pubsub topics describe work-queue

# Pull sample message
gcloud pubsub subscriptions pull work-queue-sub --limit=1
```

---

### Scenario 3: Manual Job Control

**Use Case**: Start job via API, monitor progress

**Run**:
```bash
# Start driver in background (auto-start=false)
./bin/driver \
  --auto-start=false \
  --batch-size=500 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=stdout \
  --api-port=8080 &

COORDINATOR_PID=$!
```

**Control**:
```bash
# Check status (should be "pending")
curl http://localhost:8080/job/status | jq .

# Start job
curl -X POST http://localhost:8080/job/start

# Monitor progress
watch -n 1 'curl -s http://localhost:8080/job/status | jq ".completion_percentage"'

# Pause job
curl -X PUT http://localhost:8080/job/pause

# Resume job
curl -X POST http://localhost:8080/job/start

# Cancel job (cleanup)
curl -X PUT http://localhost:8080/job/cancel

# driver will shutdown
wait $driver_PID
```

---

### Scenario 4: Using File Manifests

**Use Case**: Process specific subset of files

**Create Manifest**:
```bash
# List all bundles from specific month
find /data/bundles/2023/jan -name "*.ndjson" -type f > manifest.txt

# Or manually curate
cat > manifest.txt <<EOF
patients/high-priority/bundle-001.ndjson
patients/high-priority/bundle-002.ndjson
EOF
```

**Run**:
```bash
./bin/driver \
  --base-path=/data/bundles \
  --manifest-path=manifest.txt \
  --measures-path=/data/measures \
  --distributor=stdout
```

---

### Scenario 5: Scale Testing

**Use Case**: Generate large workload for load testing

```bash
# Generate 100K work units from available data
./bin/driver \
  --batch-size=500 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=pubsub \
  --pubsub-project-id=my-project \
  --pubsub-topic-id=work-queue \
  --scale-test-count=100000
```

**Note**: If actual data has fewer than 100K lines, driver will cycle through files to reach target count.

---

### Scenario 6: Measures Path Mode

**Use Case**: Apply all measures from a folder without listing them individually

**Run**:
```bash
./bin/driver \
  --batch-size=500 \
  --base-path=/data/bundles \
  --measures-path=/data/measures/2023/q4 \
  --use-measures-path=true \
  --distributor=stdout
```

**Result**: Each work unit includes `measures_path` field (absolute path) instead of `measures` array:

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "work_unit_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "file_path": "patients/2023/jan/bundle-001.ndjson",
  "start_line": 0,
  "end_line": 499,
  "total_lines": 500,
  "measures_path": "/data/measures/2023/q4",
  "base_path": "/data/bundles",
  "created_at": "2025-10-03T10:00:00.000Z"
}
```

**Benefits**:
- Dynamic measure sets (add/remove measures without redeploying)
- Smaller work unit payloads
- Executor discovers measures at runtime
- Measures path is absolute, independent of base_path

---

### Scenario 7: Whole File Mode (Batch Size Zero)

**Use Case**: Process entire files without splitting (optimized for small files or testing)

**Run**:
```bash
./bin/driver \
  --batch-size=0 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=stdout
```

**Performance Benefits**:
- Skips file line counting (faster startup)
- One work unit per file
- Simpler executor logic

**Work Unit Example** (note: no `start_line`, `end_line`, or `total_lines` fields):
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "work_unit_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "file_path": "patients/2023/jan/bundle-001.ndjson",
  "measures": [
    "/data/measures/cms-125-v10.json",
    "/data/measures/cms-130-v10.json"
  ],
  "base_path": "/data/bundles",
  "created_at": "2025-10-03T10:00:00.000Z"
}
```

**Note**: Line range fields are omitted. Executor detects absence and reads entire file.

---

### Scenario 8: Runtime Configuration Updates

**Use Case**: Adjust configuration dynamically before or during job execution

**Start with Auto-Start Disabled**:
```bash
./bin/driver \
  --auto-start=false \
  --batch-size=500 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=stdout \
  --api-port=8080 &

COORDINATOR_PID=$!
```

**Inspect Configuration**:
```bash
curl http://localhost:8080/config | jq .
```

**Update Batch Size (Before Starting)**:
```bash
# Switch to whole-file mode
curl -X PUT http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"batch_size": 0}'

# Verify update
curl http://localhost:8080/config | jq .batch_size
```

**Update TTL (During Execution)**:
```bash
# Start job
curl -X POST http://localhost:8080/job/start

# Extend retention period
curl -X PUT http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"completion_ttl": "20m"}'
```

**Update Batch Size While Running (Graceful Change)**:
```bash
# Adjust batch size mid-execution (applies to remaining files)
curl -X PUT http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"batch_size": 1000}'

# Expected: 200 OK
# New batch size applies to files not yet processed
```

**Invalid Update (Should Fail)**:
```bash
# Try to change distributor type while running (not allowed)
curl -X PUT http://localhost:8080/config \
  -H "Content-Type: application/json" \
  -d '{"distributor_type": "file"}'

# Expected: 400 Bad Request
# {"error": "Cannot modify distributor_type while job is running"}
```

**Cleanup**:
```bash
curl -X PUT http://localhost:8080/job/cancel
wait $driver_PID
```

---

## Testing the Build

### Unit Tests

```bash
# Run all unit tests
go test ./... -v

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Requires GCP Pub/Sub emulator
gcloud beta emulators pubsub start &
export PUBSUB_EMULATOR_HOST=localhost:8085

# Run integration tests
go test ./test/integration/... -v
```

### Contract Tests

```bash
# Test REST API contracts
cd test/integration
go test -v -run TestRestAPIContract

# Test work unit schema
go test -v -run TestWorkUnitSchema
```

---

## Docker Deployment

### Build Container

```bash
# Multi-stage build for small image
docker build -t job-driver:latest -f deployments/docker/Dockerfile .
```

### Run Container

```bash
# Stdout mode
docker run --rm \
  -v $(pwd)/testdata:/data \
  job-driver:latest \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=stdout

# Pub/Sub mode (requires GCP credentials)
docker run --rm \
  -v $(pwd)/testdata:/data \
  -v ~/.config/gcloud:/root/.config/gcloud \
  job-driver:latest \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=pubsub \
  --pubsub-project-id=my-project \
  --pubsub-topic-id=work-queue
```

---

## Cloud Run Deployment

### Prerequisites

```bash
# Set GCP project
gcloud config set project my-project

# Enable APIs
gcloud services enable run.googleapis.com
gcloud services enable pubsub.googleapis.com
gcloud services enable artifactregistry.googleapis.com
```

### Build and Push Image

```bash
# Tag for Artifact Registry
docker tag job-driver:latest \
  us-central1-docker.pkg.dev/my-project/dqme/job-driver:latest

# Push to registry
docker push us-central1-docker.pkg.dev/my-project/dqme/job-driver:latest
```

### Deploy to Cloud Run

```bash
gcloud run deploy job-driver \
  --image=us-central1-docker.pkg.dev/my-project/dqme/job-driver:latest \
  --platform=managed \
  --region=us-central1 \
  --set-env-vars="BATCH_SIZE=500,DISTRIBUTOR=pubsub,PUBSUB_PROJECT_ID=my-project,PUBSUB_TOPIC_ID=work-queue" \
  --set-env-vars="BASE_PATH=/data/bundles,MEASURES_PATH=/data/measures" \
  --memory=512Mi \
  --cpu=1 \
  --timeout=3600 \
  --max-instances=100 \
  --allow-unauthenticated
```

---

## Troubleshooting

### Issue: "No files found at base path"

**Solution**: Check that `--base-path` is correct and contains `.ndjson` files
```bash
ls -la /path/to/bundles/*.ndjson
```

### Issue: "Failed to publish to Pub/Sub"

**Solutions**:
1. Verify topic exists: `gcloud pubsub topics list`
2. Check IAM permissions: need `pubsub.publisher` role
3. Verify project ID is correct

### Issue: "Work units not balanced"

**Solution**: Adjust `--remainder-threshold`:
- Lower value (e.g., 0.1): More likely to append to last batch
- Higher value (e.g., 0.3): More likely to create new batch

### Issue: "driver shuts down immediately"

**Solution**: Check `--completion-ttl` setting. Increase if need more time to query status.

---

## Next Steps

1. Review [REST API Contract](./contracts/rest-api.md)
2. Review [Work Unit Schema](./contracts/work-unit-schema.md)
3. Implement executor service to consume work units
4. Set up monitoring and observability (Prometheus, Grafana)
5. Configure Cloud Build CI/CD pipeline

---

## Quick Reference Commands

```bash
# Minimal local run
./driver --base-path=/data/bundles --measures-path=/data/measures

# Full configuration
./driver \
  --job-id=job-123 \
  --batch-size=500 \
  --base-path=/data/bundles \
  --manifest-path=files.txt \
  --measures-path=/data/measures \
  --measures-manifest-path=measures.txt \
  --distributor=pubsub \
  --pubsub-project-id=my-project \
  --pubsub-topic-id=work-queue \
  --auto-start=true \
  --api-port=8080 \
  --completion-ttl=10m

# Check job status
curl http://localhost:8080/job/status | jq .

# Control job
curl -X POST http://localhost:8080/job/start
curl -X PUT http://localhost:8080/job/pause
curl -X PUT http://localhost:8080/job/cancel
```

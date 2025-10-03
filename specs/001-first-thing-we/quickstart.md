# Quickstart Guide: Job Coordinator Service

**Purpose**: Step-by-step guide to build, run, and test the job coordinator locally

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
cd job-coordinator
go mod download
go build -o bin/coordinator ./cmd/coordinator
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
./bin/coordinator \
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
./bin/coordinator \
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
./bin/coordinator \
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
# Start coordinator in background (auto-start=false)
./bin/coordinator \
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

# Coordinator will shutdown
wait $COORDINATOR_PID
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
./bin/coordinator \
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
./bin/coordinator \
  --batch-size=500 \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=pubsub \
  --pubsub-project-id=my-project \
  --pubsub-topic-id=work-queue \
  --scale-test-count=100000
```

**Note**: If actual data has fewer than 100K lines, coordinator will cycle through files to reach target count.

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
docker build -t job-coordinator:latest -f deployments/docker/Dockerfile .
```

### Run Container

```bash
# Stdout mode
docker run --rm \
  -v $(pwd)/testdata:/data \
  job-coordinator:latest \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor=stdout

# Pub/Sub mode (requires GCP credentials)
docker run --rm \
  -v $(pwd)/testdata:/data \
  -v ~/.config/gcloud:/root/.config/gcloud \
  job-coordinator:latest \
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
docker tag job-coordinator:latest \
  us-central1-docker.pkg.dev/my-project/dqme/job-coordinator:latest

# Push to registry
docker push us-central1-docker.pkg.dev/my-project/dqme/job-coordinator:latest
```

### Deploy to Cloud Run

```bash
gcloud run deploy job-coordinator \
  --image=us-central1-docker.pkg.dev/my-project/dqme/job-coordinator:latest \
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

### Issue: "Coordinator shuts down immediately"

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
./coordinator --base-path=/data/bundles --measures-path=/data/measures

# Full configuration
./coordinator \
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

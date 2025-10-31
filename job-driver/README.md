# Job Driver Service

A Go-based service that distributes FHIR quality measure calculation work across multiple workers. The driver processes NDJSON bundle files (plain or gzip-compressed) from cloud storage or local filesystems, splits them into batches, and distributes work units via Pub/Sub, stdout, or file outputs.

## Features

- **Batch Processing**: Flexible batch strategies - whole files, line-based batching, or multi-file batching
- **Cloud Storage Support**: Read from Google Cloud Storage (gs://), Amazon S3 (s3://), or local filesystems (file://)
- **Compression Support**: Automatic gzip decompression for .ndjson.gz files
- **Large File Handling**: Supports FHIR bundles up to 10MB per line without buffer overflow
- **Multiple Distributors**: Supports Google Cloud Pub/Sub, stdout, and file-based distribution
- **REST API**: Control job execution and monitor progress via HTTP endpoints
- **Flexible Configuration**: CLI flags, environment variables, and runtime updates
- **TTL-based Shutdown**: Automatic shutdown after job completion
- **Graceful Shutdown**: Handles SIGTERM/SIGINT with proper cleanup
- **Composable Architecture**: Modular design allows any combination of storage sources and encodings

## Quick Start

### Prerequisites

- Go 1.25.3 or later
- (Optional) Google Cloud Pub/Sub for production deployments
- (Optional) Cloud storage credentials (GCP service account, AWS access keys) if using cloud storage

### Installation

```bash
go build -o driver ./cmd/driver
```

### Basic Usage

**Stdout Mode** (simplest - prints work units to console):

```bash
./driver \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor-type=stdout \
  --auto-start=true
```

**Line-Based Batching** (splits large files into batches):

```bash
./driver \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --batch-size=1000 \
  --distributor-type=stdout \
  --auto-start=true
```

**Cloud Storage with Gzip** (reads from GCS with compression):

```bash
./driver \
  --base-path=gs://my-bucket/bundles \
  --measures-path=/data/measures \
  --batch-size=500 \
  --distributor-type=pubsub \
  --distributor-config=project_id=my-project,topic_name=work-units \
  --auto-start=true
```

**File Output Mode** (writes work units to a file):

```bash
./driver \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor-type=file \
  --distributor-config=output_path=/output/work-units.ndjson \
  --auto-start=true
```

### API Usage

Start the driver without auto-start to control it via API:

```bash
./driver \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --distributor-type=stdout \
  --api-port=8080
```

Then control the job via HTTP:

```bash
# Start the job
curl -X POST http://localhost:8080/job/start

# Check status
curl http://localhost:8080/job/status

# Pause the job
curl -X PUT http://localhost:8080/job/pause

# Cancel the job
curl -X PUT http://localhost:8080/job/cancel

# Health check
curl http://localhost:8080/health
```

## Architecture

The driver follows this workflow:

1. **Initialization**: Load configuration, create distributor, open cloud storage connections
2. **Discovery**: Find FHIR bundle files (auto-discovery or manifest), discover measures
3. **Processing**: For each file:
   - Open file (from cloud storage or local, with automatic gzip decompression)
   - Count lines (handles up to 10MB per line)
   - Split into batches based on batch_size and multifile_batches settings
   - Create work units (JSON objects with file paths, line ranges, measures)
4. **Distribution**: Send work units to configured distributor with batching/buffering
5. **Completion**: Track progress, wait for distributor drain, trigger TTL shutdown

### Components

- **Driver**: Orchestrates file processing and work unit distribution (internal/driver/)
- **Processor**: Handles file discovery, line counting, batch splitting, composable file readers (internal/processor/)
- **Storage**: Cloud storage abstraction via gocloud.dev (internal/storage/)
- **Distributor**: Sends work units to target - Pub/Sub, stdout, or file (internal/distributor/)
- **API**: REST endpoints for job control and monitoring (internal/api/)
- **Config**: Viper-based configuration with CLI/env/defaults (internal/config/)
- **Models**: Data structures for Job, WorkUnit, Config (internal/models/)

## Configuration

See [docs/configuration.md](docs/configuration.md) for complete configuration options.

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `--job-id` | Unique job identifier | Auto-generated UUID |
| `--base-path` | Root directory for FHIR bundles (supports gs://, s3://, file://) | Required |
| `--measures-path` | Directory containing measure definitions | ./measures |
| `--measures-to-run` | Specific measure paths (optional) | [] (discovers all) |
| `--batch-size` | Lines per batch (0=whole file) | 0 (whole file) |
| `--multifile-batches` | Combine multiple files into batches | true |
| `--distributor-type` | Output type: stdout, file, pubsub | stdout |
| `--auto-start` | Start job immediately | false |
| `--api-port` | REST API port | 8080 |
| `--completion-ttl` | Shutdown delay after completion | 10m |

## Development

See [docs/development.md](docs/development.md) for local development setup.

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test ./... -cover

# Run specific package
go test ./internal/coordinator -v
```

### Building

```bash
# Build binary
go build -o coordinator ./cmd/coordinator

# Build with optimizations
go build -ldflags="-s -w" -o coordinator ./cmd/coordinator
```

## Deployment

See [docs/deployment.md](docs/deployment.md) for deployment guides.

### Docker

```bash
docker build -t job-coordinator:latest -f deployments/docker/Dockerfile .
docker run -p 8080:8080 job-coordinator:latest \
  --job-id=my-job \
  --base-path=/data \
  --measures-to-run=/measures/measure1.json
```

### Cloud Run

```bash
# Deploy with Terraform
cd deployments/terraform
terraform init
terraform apply
```

## API Reference

See [docs/api.md](docs/api.md) for complete API documentation.

### Endpoints

- `POST /job/start` - Start job processing
- `PUT /job/pause` - Pause running job
- `PUT /job/cancel` - Cancel job
- `GET /job/status` - Get job status and progress
- `GET /config` - Get current configuration
- `PUT /config` - Update runtime-modifiable config
- `GET /health` - Health check

## Testing

The project includes **180+ tests**:

- **Unit tests**: Models, processors, distributors, coordinator, API
- **Integration tests**: REST API contracts, work unit schema validation
- **E2E tests**: Full workflow scenarios with real files

## License

[Your License Here]

## Contributing

[Contributing Guidelines]

## Support

[Support Information]

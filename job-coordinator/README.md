# Job Coordinator Service

A Go-based service that coordinates the distribution of FHIR quality measure calculation work units across multiple workers. The coordinator splits large NDJSON bundles into batches and distributes them via Pub/Sub, stdout, or file outputs.

## Features

- **Batch Processing**: Splits large FHIR bundle files into configurable batch sizes
- **Multiple Distributors**: Supports Google Cloud Pub/Sub, stdout, and file-based distribution
- **REST API**: Control job execution and monitor progress via HTTP endpoints
- **Flexible Configuration**: CLI flags, environment variables, and runtime updates
- **Whole File Mode**: Process entire files without batching (batch_size=0)
- **TTL-based Shutdown**: Automatic shutdown after job completion
- **Graceful Shutdown**: Handles SIGTERM/SIGINT with proper cleanup

## Quick Start

### Prerequisites

- Go 1.21 or later
- (Optional) Google Cloud Pub/Sub for production deployments

### Installation

```bash
go build -o coordinator ./cmd/coordinator
```

### Basic Usage

**Stdout Mode** (simplest - prints work units to console):

```bash
./coordinator \
  --job-id=my-job \
  --base-path=C:\data\bundles \
  --measures-to-run=/measures/measure1.json,/measures/measure2.json \
  --batch-size=500 \
  --distributor-type=stdout \
  --auto-start=true
```

**File Mode** (writes work units to a file):

```bash
./coordinator \
  --job-id=my-job \
  --base-path=C:\data\bundles \
  --measures-to-run=/measures/measure1.json \
  --batch-size=1000 \
  --distributor-type=file \
  --distributor-config=output_path:C:\output\work-units.ndjson \
  --auto-start=true
```

**Pub/Sub Mode** (production - distributes to GCP Pub/Sub):

```bash
./coordinator \
  --job-id=my-job \
  --base-path=/data/bundles \
  --measures-to-run=/measures/measure1.json \
  --batch-size=500 \
  --distributor-type=pubsub \
  --distributor-config=project_id:my-project,topic_id:work-units \
  --auto-start=true
```

### API Usage

Start the coordinator without auto-start to control it via API:

```bash
./coordinator \
  --job-id=api-controlled-job \
  --base-path=C:\data\bundles \
  --measures-to-run=/measures/measure1.json \
  --batch-size=500 \
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

The coordinator follows this workflow:

1. **Initialization**: Load configuration, create distributor
2. **Discovery**: Find FHIR bundle files (auto-discovery or manifest)
3. **Processing**: For each file:
   - Count lines
   - Split into batches based on batch_size
   - Create work units (JSON objects with file path, line ranges, measures)
4. **Distribution**: Send work units to configured distributor
5. **Completion**: Track progress, handle errors, trigger TTL shutdown

### Components

- **Coordinator**: Orchestrates file processing and work unit distribution
- **Processor**: Handles file discovery, line counting, batch splitting
- **Distributor**: Sends work units to target (Pub/Sub, stdout, file)
- **API**: REST endpoints for job control and monitoring
- **Config**: Viper-based configuration with CLI/env/defaults

## Configuration

See [docs/configuration.md](docs/configuration.md) for complete configuration options.

### Key Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `--job-id` | Unique job identifier | Required |
| `--base-path` | Root directory for FHIR bundles | Required |
| `--measures-to-run` | Comma-separated measure paths | Required |
| `--batch-size` | Lines per batch (0=whole file) | 500 |
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

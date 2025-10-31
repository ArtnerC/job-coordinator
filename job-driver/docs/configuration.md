# Configuration Guide

The Job Coordinator supports configuration via CLI flags, environment variables, and defaults. Configuration precedence: **CLI flags > Environment variables > Defaults**.

## Configuration Options

### Required Fields

#### `--job-id` (string)
- **Description**: Unique identifier for the job
- **Environment Variable**: `JOB_COORDINATOR_JOB_ID`
- **Required**: Yes
- **Example**: `--job-id=quality-measure-calc-001`

#### `--base-path` (string)
- **Description**: Absolute path to the directory containing FHIR bundle files
- **Environment Variable**: `JOB_COORDINATOR_BASE_PATH`
- **Required**: Yes
- **Example**: `--base-path=C:\data\bundles`

#### `--measures-to-run` (comma-separated)
- **Description**: Comma-separated list of measure definition paths (unless using `--use-measures-path`)
- **Environment Variable**: `JOB_COORDINATOR_MEASURES_TO_RUN`
- **Required**: Yes (unless `--use-measures-path=true`)
- **Example**: `--measures-to-run=/measures/measure1.json,/measures/measure2.json`

### Optional Fields

#### `--auto-start` (boolean)
- **Description**: Start job processing immediately on startup
- **Environment Variable**: `JOB_COORDINATOR_AUTO_START`
- **Default**: `false`
- **Example**: `--auto-start=true`

#### `--batch-size` (integer)
- **Description**: Number of lines per batch. Set to `0` for whole file mode.
- **Environment Variable**: `JOB_COORDINATOR_BATCH_SIZE`
- **Default**: `500`
- **Example**: `--batch-size=1000`

#### `--remainder-threshold` (float)
- **Description**: Threshold for merging remainder into last batch (0.0-1.0)
- **Environment Variable**: `JOB_COORDINATOR_REMAINDER_THRESHOLD`
- **Default**: `0.2`
- **Example**: `--remainder-threshold=0.3`

#### `--manifest-path` (string)
- **Description**: Path to file manifest (list of bundle files to process)
- **Environment Variable**: `JOB_COORDINATOR_MANIFEST_PATH`
- **Default**: None (auto-discover files in base-path)
- **Example**: `--manifest-path=C:\manifests\files.txt`

#### `--measures-path` (string)
- **Description**: Path to directory containing measure definitions
- **Environment Variable**: `JOB_COORDINATOR_MEASURES_PATH`
- **Default**: None
- **Example**: `--measures-path=/measures`

#### `--measures-manifest-path` (string)
- **Description**: Path to measures manifest file (when `--use-measures-path=true`)
- **Environment Variable**: `JOB_COORDINATOR_MEASURES_MANIFEST_PATH`
- **Default**: None
- **Example**: `--measures-manifest-path=/manifests/measures.txt`

#### `--use-measures-path` (boolean)
- **Description**: Load measures from manifest instead of using `--measures-to-run`
- **Environment Variable**: `JOB_COORDINATOR_USE_MEASURES_PATH`
- **Default**: `false`
- **Example**: `--use-measures-path=true`

#### `--distributor-type` (string)
- **Description**: Type of distributor: `stdout`, `file`, or `pubsub`
- **Environment Variable**: `JOB_COORDINATOR_DISTRIBUTOR_TYPE`
- **Default**: `stdout`
- **Example**: `--distributor-type=pubsub`

#### `--distributor-config` (key=value pairs)
- **Description**: Distributor-specific configuration (comma-separated `key:value` pairs)
- **Environment Variable**: `JOB_COORDINATOR_DISTRIBUTOR_CONFIG`
- **Default**: None
- **Runtime Modifiable**: Yes
- **Examples**:
  - File: `--distributor-config=output_path:C:\output\work-units.ndjson`
  - Pub/Sub: `--distributor-config=project_id:my-project,topic_id:work-units`

#### `--completion-ttl` (duration)
- **Description**: Time to wait after job completion before shutdown
- **Environment Variable**: `JOB_COORDINATOR_COMPLETION_TTL`
- **Default**: `10m`
- **Example**: `--completion-ttl=30m`

#### `--concurrent-file-processors` (integer)
- **Description**: Number of concurrent file processors (worker pool size)
- **Environment Variable**: `JOB_COORDINATOR_CONCURRENT_FILE_PROCESSORS`
- **Default**: `10`
- **Runtime Modifiable**: Yes
- **Example**: `--concurrent-file-processors=20`

#### `--scale-test-count` (integer)
- **Description**: Number of work units for scale testing mode
- **Environment Variable**: `JOB_COORDINATOR_SCALE_TEST_COUNT`
- **Default**: None
- **Example**: `--scale-test-count=1000`

#### `--api-port` (integer)
- **Description**: Port for REST API server
- **Environment Variable**: `JOB_COORDINATOR_API_PORT`
- **Default**: `8080`
- **Example**: `--api-port=9090`

## Configuration Examples

### Example 1: Basic Stdout Mode

```bash
./coordinator \
  --job-id=example-job \
  --base-path=C:\data\bundles \
  --measures-to-run=/measures/measure1.json \
  --auto-start=true
```

### Example 2: Pub/Sub with Custom Batch Size

```bash
./coordinator \
  --job-id=pubsub-job \
  --base-path=/data/bundles \
  --measures-to-run=/measures/measure1.json,/measures/measure2.json \
  --batch-size=1000 \
  --distributor-type=pubsub \
  --distributor-config=project_id:my-gcp-project,topic_id:work-units \
  --auto-start=true \
  --completion-ttl=15m
```

### Example 3: File Manifest + Measures Manifest

```bash
./coordinator \
  --job-id=manifest-job \
  --base-path=C:\data \
  --manifest-path=C:\manifests\files.txt \
  --measures-manifest-path=C:\manifests\measures.txt \
  --use-measures-path=true \
  --batch-size=500 \
  --distributor-type=file \
  --distributor-config=output_path:C:\output\work-units.ndjson
```

### Example 4: Environment Variables

```bash
export JOB_COORDINATOR_JOB_ID="env-job"
export JOB_COORDINATOR_BASE_PATH="/data/bundles"
export JOB_COORDINATOR_MEASURES_TO_RUN="/measures/measure1.json"
export JOB_COORDINATOR_BATCH_SIZE="1000"
export JOB_COORDINATOR_AUTO_START="true"

./coordinator
```

### Example 5: Whole File Mode (batch_size=0)

```bash
./coordinator \
  --job-id=wholefile-job \
  --base-path=C:\data\bundles \
  --measures-to-run=/measures/measure1.json \
  --batch-size=0 \
  --distributor-type=stdout \
  --auto-start=true
```

## Runtime Modifiable Fields

The following fields can be updated while the job is running via `PUT /config`:

- `concurrent_file_processors` - Adjust worker pool size
- `distributor_config` - Update distributor-specific settings

All other fields are immutable during job execution.

## Configuration Validation

The coordinator validates configuration on startup:

- `base_path` must be an absolute path
- `batch_size` must be >= 0
- `remainder_threshold` must be between 0.0 and 1.0
- `distributor_type` must be one of: `stdout`, `file`, `pubsub`
- `concurrent_file_processors` must be > 0

Invalid configuration will result in startup failure with a clear error message.

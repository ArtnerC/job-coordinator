# Configuration Guide

The Job Driver supports configuration via CLI flags, environment variables, and defaults. Configuration precedence: **CLI flags > Environment variables > Defaults**.

## Configuration Options

### Required Fields

#### `--job-id` (string)
- **Description**: Unique identifier for the job (auto-generated UUID if not provided)
- **Environment Variable**: `JOB_DRIVER_JOB_ID`
- **Required**: No (auto-generated if omitted)
- **Example**: `--job-id=quality-measure-calc-001`

#### `--base-path` (string)
- **Description**: Base directory containing patient bundle files
- **Environment Variable**: `JOB_DRIVER_BASE_PATH`
- **Required**: Yes
- **Example**: `--base-path=/data/bundles` or `--base-path=C:\data\bundles`

### Optional Fields

#### `--auto-start` (boolean)
- **Description**: Start job processing immediately on startup
- **Environment Variable**: `JOB_DRIVER_AUTO_START`
- **Default**: `false`
- **Example**: `--auto-start=true`

#### `--batch-size` (integer)
- **Description**: Number of lines per work unit. Set to `0` for whole file mode (default).
- **Environment Variable**: `JOB_DRIVER_BATCH_SIZE`
- **Default**: `0` (whole file mode)
- **Runtime Modifiable**: Yes
- **Example**: `--batch-size=1000`

#### `--multifile-batches` (boolean)
- **Description**: Combine multiple small files into batches until batch-size is reached
- **Environment Variable**: `JOB_DRIVER_MULTIFILE_BATCHES`
- **Default**: `true`
- **Example**: `--multifile-batches=false`

#### `--remainder-threshold` (float)
- **Description**: Threshold for appending remainder to last batch (0.0-1.0)
- **Environment Variable**: `JOB_DRIVER_REMAINDER_THRESHOLD`
- **Default**: `0.2`
- **Runtime Modifiable**: Yes
- **Example**: `--remainder-threshold=0.3`

#### `--manifest-path` (string)
- **Description**: Path to file manifest (list of bundle files to process)
- **Environment Variable**: `JOB_DRIVER_MANIFEST_PATH`
- **Default**: None (auto-discover files in base-path)
- **Example**: `--manifest-path=C:\manifests\files.txt`

#### `--measures-path` (string)
- **Description**: Directory containing measure definition files
- **Environment Variable**: `JOB_DRIVER_MEASURES_PATH`
- **Default**: `./measures`
- **Example**: `--measures-path=/data/measures`

#### `--measures-to-run` (string slice)
- **Description**: Specific measure paths to include (optional, discovers all in measures-path if not provided)
- **Environment Variable**: `JOB_DRIVER_MEASURES_TO_RUN`
- **Default**: `[]` (empty, discovers all measures)
- **Example**: `--measures-to-run=/measures/measure1.json --measures-to-run=/measures/measure2.json`

#### `--measures-manifest-path` (string)
- **Description**: Path to measures manifest file (alternative to auto-discovery)
- **Environment Variable**: `JOB_DRIVER_MEASURES_MANIFEST_PATH`
- **Default**: None
- **Example**: `--measures-manifest-path=/manifests/measures.txt`

#### `--distributor-type` (string)
- **Description**: Type of distributor: `stdout`, `file`, or `pubsub`
- **Environment Variable**: `JOB_DRIVER_DISTRIBUTOR_TYPE`
- **Default**: `stdout`
- **Example**: `--distributor-type=pubsub`

#### `--distributor-config` (key=value pairs)
- **Description**: Distributor-specific configuration (key=value pairs)
- **Environment Variable**: `JOB_DRIVER_DISTRIBUTOR_CONFIG`
- **Default**: `{}` (empty map)
- **Runtime Modifiable**: No (set before job starts)
- **Examples**:
  - **Stdout**: `--distributor-config=delay_ms=100` (adds delay for demos/testing)
  - **File**: `--distributor-config=output_path=/output/work-units.ndjson` (required)
  - **Pub/Sub**: `--distributor-config=project_id=my-project,topic_name=work-units,completion_timeout=2h,poll_interval=5s,subscription_ttl=24h`
    - `project_id` (required): GCP project ID
    - `topic_name` (required): Pub/Sub topic name
    - `completion_timeout` (optional): Max time to wait for queue drain, default varies
    - `poll_interval` (optional): How often to poll subscription for completion, default varies
    - `subscription_ttl` (optional): Subscription expiration time, default varies

#### `--completion-ttl` (duration)
- **Description**: Time to wait after job completion before shutdown
- **Environment Variable**: `JOB_DRIVER_COMPLETION_TTL`
- **Default**: `10m`
- **Example**: `--completion-ttl=30m`

#### `--concurrent-file-processors` (integer)
- **Description**: Number of concurrent file processors (worker pool size)
- **Environment Variable**: `JOB_DRIVER_CONCURRENT_FILE_PROCESSORS`
- **Default**: `10`
- **Runtime Modifiable**: Yes
- **Example**: `--concurrent-file-processors=20`

#### `--scale-load` (integer)
- **Description**: Synthesize load by generating N work units from file patterns (0 to disable)
- **Environment Variable**: `JOB_DRIVER_SCALE_LOAD`
- **Default**: `0` (disabled)
- **Example**: `--scale-load=10000`

#### `--api-port` (integer)
- **Description**: Port for REST API server
- **Environment Variable**: `JOB_DRIVER_API_PORT`
- **Default**: `8080`
- **Example**: `--api-port=9090`

## Configuration Examples

### Example 1: Basic Stdout Mode (Whole File - Default)

```bash
./driver \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --auto-start=true
```

### Example 2: Pub/Sub with Line Batching

```bash
./driver \
  --base-path=/data/bundles \
  --measures-path=/data/measures \
  --batch-size=1000 \
  --distributor-type=pubsub \
  --distributor-config=project_id=my-gcp-project,topic_name=work-units,completion_timeout=2h \
  --auto-start=true \
  --completion-ttl=15m
```

### Example 3: File Output with Manifest

```bash
./driver \
  --base-path=/data/bundles \
  --manifest-path=/manifests/files.txt \
  --measures-manifest-path=/manifests/measures.txt \
  --batch-size=500 \
  --distributor-type=file \
  --distributor-config=output_path=/output/work-units.ndjson \
  --auto-start=true
```

### Example 4: Environment Variables

```bash
export JOB_DRIVER_BASE_PATH="/data/bundles"
export JOB_DRIVER_MEASURES_PATH="/data/measures"
export JOB_DRIVER_BATCH_SIZE="1000"
export JOB_DRIVER_AUTO_START="true"
export JOB_DRIVER_DISTRIBUTOR_TYPE="stdout"

./driver
```

### Example 5: Cloud Storage with Gzip Support

```bash
./driver \
  --base-path=gs://my-bucket/bundles \
  --measures-path=/data/measures \
  --batch-size=500 \
  --distributor-type=pubsub \
  --distributor-config=project_id=my-project,topic_name=work-units \
  --auto-start=true
```

Note: Cloud storage URLs (gs://, s3://, file://) and gzip-compressed files (.ndjson.gz) are automatically detected and handled.

## Runtime Modifiable Fields

The following fields can be updated while the job is running (running/paused states) via `PUT /config`:

- `batch_size` - Adjust batch size for remaining files
- `remainder_threshold` - Change threshold for remainder batches
- `concurrent_file_processors` - Adjust worker pool size
- `completion_ttl` - Update TTL (modifiable in any state including completed)

The following fields can only be modified before the job starts (pending state):

- `distributor_type`
- `distributor_config`
- `measures_to_run`

All other fields are immutable once set during initialization.

## Configuration Validation

The driver validates configuration on startup:

- `batch_size` must be >= 0 (0 = whole file mode)
- `remainder_threshold` must be between 0.0 and 1.0
- `distributor_type` must be one of: `stdout`, `file`, `pubsub`
- `concurrent_file_processors` must be > 0
- File distributor requires `output_path` in `distributor_config`
- Pub/Sub distributor requires `project_id` and `topic_name` in `distributor_config`

Invalid configuration will result in startup failure with a clear error message.


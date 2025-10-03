# Work Unit Distribution Contract

**Version**: 1.0.0  
**Purpose**: Define the format of work units distributed to executors via Pub/Sub, stdout, or file

## Work Unit JSON Schema

Work units are serialized as JSON and sent to the configured distribution destination.

### Schema Definition

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "WorkUnit",
  "description": "A discrete unit of quality measure calculation work",
  "type": "object",
  "required": [
    "job_id",
    "work_unit_id",
    "file_path",
    "base_path",
    "created_at"
  ],
  "properties": {
    "job_id": {
      "type": "string",
      "description": "Unique identifier for the parent job",
      "pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
    },
    "work_unit_id": {
      "type": "string",
      "description": "Unique identifier for this work unit",
      "pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$"
    },
    "file_path": {
      "type": "string",
      "description": "Relative path to NDJSON patient bundle file",
      "minLength": 1
    },
    "start_line": {
      "type": "integer",
      "description": "Starting line index (0-based, inclusive). Omitted when batch_size=0",
      "minimum": 0
    },
    "end_line": {
      "type": "integer",
      "description": "Ending line index (0-based, inclusive). Omitted when batch_size=0",
      "minimum": 0
    },
    "total_lines": {
      "type": "integer",
      "description": "Number of lines in this work unit. Omitted when batch_size=0",
      "minimum": 1
    },
    "measures": {
      "type": "array",
      "description": "List of absolute measure file paths to apply (mutually exclusive with measures_path)",
      "items": {
        "type": "string"
      }
    },
    "measures_path": {
      "type": "string",
      "description": "Absolute path to folder containing all measures to apply (mutually exclusive with measures)"
    },
    "base_path": {
      "type": "string",
      "description": "Base directory path for resolving relative file paths",
      "minLength": 1
    },
    "created_at": {
      "type": "string",
      "format": "date-time",
      "description": "Work unit creation timestamp (ISO 8601)"
    }
  },
  "oneOf": [
    {
      "required": ["measures"],
      "not": { "required": ["measures_path"] }
    },
    {
      "required": ["measures_path"],
      "not": { "required": ["measures"] }
    }
  ]
}
```

### Example Work Units

**Example 1: Explicit Measure List**

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "work_unit_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "file_path": "patients/2023/jan/bundle-001.ndjson",
  "start_line": 0,
  "end_line": 499,
  "total_lines": 500,
  "measures": [
    "measures/cms-125-v10.json",
    "measures/cms-130-v10.json"
  ],
  "base_path": "/data/bundles",
  "created_at": "2025-10-03T10:00:00.000Z"
}
```

**Example 2: Measures Folder Path**

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "work_unit_id": "8d3f7890-8536-51ef-b55c-f18gd2g01bf8",
  "file_path": "patients/2023/jan/bundle-002.ndjson",
  "start_line": 0,
  "end_line": 999,
  "total_lines": 1000,
  "measures_path": "/data/measures/2023/q4",
  "base_path": "/data/bundles",
  "created_at": "2025-10-03T10:00:01.000Z"
}
```

**Example 3: Whole File Mode (batch_size=0)**

```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "work_unit_id": "9e4g8901-9647-62fg-c66d-g29he3h12cg9",
  "file_path": "patients/2023/feb/bundle-003.ndjson",
  "measures": [
    "/data/measures/cms-125-v10.json",
    "/data/measures/cms-130-v10.json"
  ],
  "base_path": "/data/bundles",
  "created_at": "2025-10-03T10:00:02.000Z"
}
```

**Validation Rules**: 
- Exactly one of `measures` or `measures_path` MUST be provided (mutually exclusive)
- If `measures_path` is specified, it MUST be an absolute path; executor discovers all measure files in that folder
- If `measures` is specified, each path MUST be absolute
- `start_line`, `end_line`, and `total_lines` are optional; omitted when `batch_size=0` (whole file mode)
- When line fields omitted, executor processes entire file without line range constraints

---

## Distribution Methods

### 1. GCP Pub/Sub

**Destination**: GCP Pub/Sub topic  
**Format**: Each work unit is a separate Pub/Sub message  
**Message Body**: JSON-serialized work unit  
**Message Attributes**:
```json
{
  "job_id": "550e8400-e29b-41d4-a716-446655440000",
  "work_unit_id": "7c9e6679-7425-40de-944b-e07fc1f90ae7",
  "content_type": "application/json"
}
```

**Publishing Pattern**:
- Batch publish in groups of 100-500 messages for efficiency
- Each message gets unique message ID from Pub/Sub
- Monitor PublishResult for errors
- Failed publishes are tracked in job error list

**Configuration Required**:
- `project_id`: GCP project ID
- `topic_id`: Pub/Sub topic name

**Executor Contract**:
- Subscribe to topic with appropriate subscription
- Parse JSON message body into WorkUnit struct
- Acknowledge message after successful processing
- Nack message on failure (triggers retry by Pub/Sub)

---

### 2. Stdout

**Destination**: Standard output (stdout)  
**Format**: One JSON object per line (NDJSON format)  
**Usage**: Pipe coordinator output to downstream processor

**Example Output**:
```
{"job_id":"550e8400-e29b-41d4-a716-446655440000","work_unit_id":"7c9e6679-7425-40de-944b-e07fc1f90ae7","file_path":"patients/2023/jan/bundle-001.ndjson","start_line":0,"end_line":499,"total_lines":500,"measures":["measures/cms-125-v10.json"],"base_path":"/data/bundles","created_at":"2025-10-03T10:00:00.000Z"}
{"job_id":"550e8400-e29b-41d4-a716-446655440000","work_unit_id":"8d3f7890-8536-51ef-b55c-f18gd2g01bf8","file_path":"patients/2023/jan/bundle-001.ndjson","start_line":500,"end_line":999,"total_lines":500,"measures":["measures/cms-125-v10.json"],"base_path":"/data/bundles","created_at":"2025-10-03T10:00:00.001Z"}
```

**Configuration Required**: None

**Executor Contract**:
- Read from stdin line by line
- Parse each line as JSON WorkUnit
- Process work unit
- Write results to own output destination

**Shell Usage**:
```bash
# Pipe to downstream processor
./coordinator --distributor=stdout | ./executor

# Save to file for later processing
./coordinator --distributor=stdout > work-units.ndjson

# Split across multiple executors
./coordinator --distributor=stdout | tee >(./executor1) >(./executor2)
```

---

### 3. File Output

**Destination**: NDJSON file on disk  
**Format**: One JSON object per line (NDJSON format)  
**Usage**: Batch job scenarios, manual processing

**Configuration Required**:
- `output_path`: Absolute path to output file (e.g., `/output/work-units.ndjson`)

**File Characteristics**:
- Created if doesn't exist
- Truncated if exists (overwrites previous content)
- Written sequentially (not appended concurrently)
- Flushed and closed on job completion

**Executor Contract**:
- Read file line by line
- Parse each line as JSON WorkUnit
- Process work unit
- Track progress separately (coordinator doesn't monitor file consumption)

---

## Work Unit Processing Contract (for Executors)

Executors consuming work units MUST:

1. **Read Lines from File**:
   - Construct full path: `{base_path}/{file_path}`
   - Open file for reading
   
   **If `start_line` and `end_line` provided** (batched mode):
   - Skip first `start_line` lines (0-indexed)
   - Read lines from `start_line` to `end_line` (inclusive)
   - Parse each line as FHIR Bundle JSON
   
   **If `start_line` and `end_line` omitted** (whole file mode):
   - Read entire file from beginning to end
   - Parse each line as FHIR Bundle JSON

2. **Load Measures**:
   
   **Option A: Explicit Measure List** (if `measures` field provided):
   - For each measure in `measures` array:
     - Load measure definition from absolute path
     - Parse as CQL measure
   
   **Option B: Measures Folder** (if `measures_path` field provided):
   - Use absolute path from `measures_path`
   - Discover all measure files in folder (e.g., `*.json`, `*.cql`)
   - For each discovered measure file:
     - Load measure definition
     - Parse as CQL measure
   - Sort measures by name for consistent execution order

3. **Execute Measures**:
   - For each patient bundle:
     - Apply each loaded measure
     - Calculate measure results
     - Aggregate results

4. **Report Results**:
   - Write results to configured output destination
   - Include `job_id` and `work_unit_id` for correlation
   - Report success/failure status

5. **Error Handling**:
   - Invalid JSON: Skip line, log error
   - Missing file: Fail work unit, report error
   - Measure execution error: Log error, continue to next patient
   - Critical errors: Fail work unit, report to coordinator (future)

---

## Testing Contract

Test scenarios to validate work unit distribution:

1. **Small Job (< batch size)**:
   - Single file with 100 lines, batch size 500
   - Expect: 1 work unit (lines 0-99)

2. **Even Split**:
   - Single file with 1000 lines, batch size 500
   - Expect: 2 work units (lines 0-499, 500-999)

3. **Uneven Split - Append to Last**:
   - Single file with 1050 lines, batch size 500, threshold 20%
   - Remainder: 50 lines (10% < 20%)
   - Expect: 2 work units (lines 0-499, 500-1049)

4. **Uneven Split - New Batch**:
   - Single file with 1600 lines, batch size 500, threshold 20%
   - Remainder: 100 lines (20% >= 20%)
   - Expect: 4 work units (lines 0-499, 500-999, 1000-1499, 1500-1599)

5. **Multiple Files**:
   - 3 files with 1000, 500, 1200 lines, batch size 500
   - Expect: 2 + 1 + 3 = 6 work units

6. **Empty Measures**:
   - Work unit with empty measures array
   - Expect: Valid work unit, executor handles gracefully

7. **Pub/Sub Distribution**:
   - Publish 1000 work units
   - Verify: All messages published successfully
   - Verify: Message attributes set correctly

8. **Stdout Distribution**:
   - Write 100 work units to stdout
   - Verify: Valid NDJSON format
   - Verify: One work unit per line

9. **File Distribution**:
   - Write 50 work units to file
   - Verify: File created with 50 lines
   - Verify: Each line is valid JSON

---

## Versioning

**Current Version**: 1.0.0

**Breaking Changes** (require MAJOR version bump):
- Removing required fields
- Changing field types
- Changing field semantics (e.g., 1-indexed instead of 0-indexed)

**Non-Breaking Changes** (MINOR version bump):
- Adding optional fields
- Adding new distribution methods

**Patch Changes**:
- Documentation clarifications
- Example updates

**Version Header**: Include in Pub/Sub attributes or file metadata
```json
{
  "schema_version": "1.0.0"
}
```

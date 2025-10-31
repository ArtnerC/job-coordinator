# Feature Specification: Job Driver Service

**Feature Branch**: `001-job-driver`  
**Created**: 2025-10-02  
**Last Updated**: 2025-10-31
**Status**: In Development  
**Organization**: CVS Health (Aetna Digital Quality Measure Evaluation)  
**Input**: User description: "first thing we are going to build is a job-driver service. we are creating the spec for that code only right now. it will take input params and distribute work on a queue for a scaling job to read and then wait for job completion to report status to the caller."

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story

**Context**: Aetna's quality measure evaluation system needs to process large batches of FHIR patient bundles to calculate quality measures (e.g., CMS-125, CMS-130) for population health reporting and compliance.

A system or user needs to process a large batch of quality measure calculations against patient FHIR bundles stored in cloud storage (Google Cloud Storage, S3, or local file systems). Rather than processing everything synchronously (which could timeout or overwhelm resources), they submit a job request to the Job Driver service. The Job Driver accepts the request, discovers and counts lines in the data files (supporting both plain and gzipped NDJSON formats), breaks down the work into optimal batch sizes, distributes these units to a message queue where horizontally-scaled CQL executor services consume and process them, monitors progress, and maintains job status until completion.

### Acceptance Scenarios

1. **Given** a valid job request with input parameters, **When** submitted to the Job Driver, **Then** the service MUST accept the request, assign a unique job ID, distribute work units to the queue, and return the job ID immediately

2. **Given** a job has been submitted and work distributed, **When** workers process the queue items, **Then** the Job Driver MUST track completion status of each work unit

3. **Given** all work units for a job have completed successfully, **When** the caller queries job status, **Then** the service MUST return "completed" status with summary results

4. **Given** some work units fail during processing, **When** the job completes, **Then** the service MUST return status indicating partial failure with details on what succeeded and what failed

5. **Given** a caller wants to check progress, **When** querying job status while work is in progress, **Then** the service MUST return current completion percentage and status

6. **Given** FHIR bundle files in gzip-compressed format, **When** processing the job, **Then** the system MUST automatically detect compression and decompress files transparently

7. **Given** files stored in cloud storage (Google Cloud Storage or Amazon S3), **When** processing the job, **Then** the system MUST read files directly from cloud storage without requiring local copies

8. **Given** files with very large individual lines (up to 10MB per FHIR bundle), **When** counting lines and creating batches, **Then** the system MUST handle them without buffer overflow or memory errors

9. **Given** a job configuration with manifest file listing specific files, **When** the job starts, **Then** the system MUST process only the files listed in the manifest

10. **Given** a job configuration pointing to a directory without a manifest, **When** the job starts, **Then** the system MUST auto-discover all matching files in the directory

### Edge Cases

- What happens when the job request contains invalid or malformed input parameters?
  - System MUST validate input and reject with clear error message before queuing work
  
- What happens when work units remain in queue but no workers are available?
  - System MUST track queue depth and report "pending" status; job should not timeout prematurely
  
- What happens when a worker crashes or fails to acknowledge work completion?
  - System MUST detect stalled work units (via timeout) and mark them for retry or report failure

- What happens when the caller's connection drops after submitting the job?
  - Job MUST continue processing independently; caller can reconnect and query status using job ID

- What happens when multiple identical job requests are submitted?
  - System processes one job per driver instance; duplicate submissions would require separate driver instances

- What happens to job status data after completion?
  - driver keeps status in memory for configurable period (default 10 minutes) after job completion, then shuts down

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST accept job requests containing input parameters that define the work to be performed

- **FR-002**: System MUST assign a unique identifier to each job request upon acceptance; job ID can be provided via environment variable or auto-generated if not provided

- **FR-003**: System MUST validate input parameters before accepting a job and reject invalid requests with descriptive error messages

- **FR-004**: System MUST discover data files from specified locations (cloud storage or local file systems) using manifest files or automatic directory scanning

- **FR-004a**: System MUST support reading FHIR bundle files in both plain and gzip-compressed NDJSON format from multiple storage sources (cloud storage services, local file systems)

- **FR-004b**: System MUST accurately count lines in data files to determine total workload, supporting large files with lines exceeding standard buffer limits (up to 10MB per line)

- **FR-004c**: System MUST break down accepted jobs into discrete work units (batches) suitable for distribution, with configurable batch sizes and intelligent splitting to avoid creating batches that are too small

- **FR-005**: System MUST distribute work units to a queue where worker processes can retrieve them

- **FR-006**: System MUST track the status of each work unit (pending, in-progress, completed, failed)

- **FR-007**: System MUST aggregate work unit statuses to determine overall job status

- **FR-008**: System MUST provide job status queries that return: job ID, overall status (pending/in-progress/completed/failed), completion percentage, start time, end time (if completed)

- **FR-009**: System MUST wait for all work units to complete or fail before marking job as complete

- **FR-010**: System MUST handle partial failures gracefully, reporting which work units succeeded and which failed

- **FR-011**: System MUST support asynchronous operation - job submission returns immediately with job ID, not blocking until completion

- **FR-012**: System MUST process one job per driver instance; concurrent job handling requires multiple driver instances

- **FR-013**: System MUST support concurrent processing of work units within a single job via distributed queue

- **FR-014**: System MUST delegate work unit execution to external CQL executor services that consume from the distributed queue; driver does not retry failed work units

- **FR-015**: System MUST support configurable job sizes with no hard limits on work units per job or total payload size, efficiently handling files with very large individual lines (FHIR bundles up to 10MB per line)

- **FR-016**: System MUST keep job status in memory and remain running for a configurable period (default 10 minutes) after job completion to allow status queries

- **FR-017**: System MUST shut down automatically after the status retention period expires following job completion

- **FR-018**: System MUST log all job state transitions for audit and debugging purposes

- **FR-019**: System MUST support REST API endpoints for job control including start, pause, cancel, and status operations

- **FR-020**: System MUST support multiple distribution targets including message queue services, standard output (for Unix pipeline integration), and file output

### Success Criteria

The Job Driver service is considered successful when:

1. **Scalability**: System successfully distributes work for jobs containing 1 million+ FHIR bundles across multiple executor instances without performance degradation

2. **Throughput**: System can process and distribute 100,000+ work units per minute to the queue under normal operating conditions

3. **Reliability**: System maintains 99.9% uptime during active job processing with graceful handling of transient failures

4. **File Format Support**: System correctly processes both plain and gzip-compressed NDJSON files without manual format specification

5. **Storage Flexibility**: System reads files from any supported storage source (cloud or local) with identical behavior and performance characteristics

6. **Large Data Handling**: System successfully processes files containing individual lines up to 10MB without errors or memory issues

7. **Operational Efficiency**: Batch splitting algorithm creates work units that maximize executor utilization (less than 5% of batches are below the configured small-batch threshold)

8. **Job Control**: Users can start, pause, cancel, and query job status via REST API with response times under 200ms for status queries

9. **Observability**: All job state transitions are logged with sufficient detail to troubleshoot issues without accessing source code

10. **Resource Efficiency**: Driver instance shuts down automatically within the configured TTL after job completion, preventing resource waste

### Key Entities

- **Job**: Represents a complete work request from submission to completion. Contains unique job ID, input parameters, overall status, start/end timestamps, list of associated work units, and summary results.

- **Work Unit**: Represents a single discrete piece of work within a job. Contains work unit ID, parent job ID, input data/parameters (file path, line ranges, measure definitions), status (pending/in-progress/completed/failed), assigned worker identifier (if in progress), result data (on completion), and error details (on failure).

- **Job Status Query**: Request to retrieve current state of a job. Contains job ID for lookup, returns job metadata, current status, progress metrics, and completion details.

- **File Specification**: Identifies a data file to be processed. Contains storage path (cloud URL or local path), optional line range (for splitting large files), and file format metadata.

- **Measure Definition**: References to quality measure definitions (e.g., CMS-125) that executors will apply to FHIR bundles. Opaque to the driver, passed through to executors.

### Assumptions

The following assumptions were made during specification and validated during development:

1. **Data Format**: All patient bundle files are in NDJSON (newline-delimited JSON) format with one FHIR bundle per line. Plain text and gzip compression are both supported.

2. **Line-Based Processing**: FHIR bundles can be split at line boundaries for batch processing. Each line represents an independent unit of work (patient record).

3. **File Stability**: Input files remain stable and unchanged during job processing. Files are not modified or deleted while a job is reading them.

4. **Cloud Storage Access**: When using cloud storage, appropriate authentication and permissions are configured externally (service accounts, IAM roles, access keys).

5. **Message Queue Availability**: The target message queue service is operational and accessible when the job starts. Driver does not handle queue provisioning.

6. **Executor Compatibility**: CQL executor services consuming from the queue understand the work unit schema and can process FHIR bundles referenced by file paths and line ranges.

7. **Network Reliability**: Network connections to cloud storage and message queues are sufficiently reliable for production use, with standard retry handling for transient failures.

8. **Resource Limits**: Deployment environment has sufficient memory and CPU to handle configured batch sizes and concurrent file operations.

9. **File Encoding**: All NDJSON files use UTF-8 encoding. Gzip compression is standard gzip format.

10. **Job Sizing**: Jobs are sized appropriately for the configured TTL period. Extremely long-running jobs (days/weeks) are not the primary use case.

---

## Review & Acceptance Checklist

### Content Quality
- [x] No implementation details (languages, frameworks, APIs)
- [x] Focused on user value and business needs
- [x] Written for non-technical stakeholders
- [x] All mandatory sections completed

### Requirement Completeness
- [x] No [NEEDS CLARIFICATION] markers remain
- [x] Requirements are testable and unambiguous
- [x] Success criteria are measurable
- [x] Scope is clearly bounded
- [x] Dependencies and assumptions identified

---

## Execution Status

- [x] User description parsed
- [x] Key concepts extracted
- [x] Ambiguities marked and resolved
- [x] User scenarios defined
- [x] Requirements generated
- [x] Entities identified
- [x] Review checklist passed

---

## Architecture Decisions

The following key architectural decisions were made during specification and development:

1. **Single Job Per Instance**: Each driver instance processes one job. Multiple jobs require multiple driver instances. Job ID can be provided via environment variable or auto-generated.

2. **In-Memory Status with TTL**: Job status is kept in memory for a configurable period (default 10 minutes) after completion, then driver shuts down. Future enhancement will persist status to metadata database.

3. **Distributed Queue Model**: Work units are distributed to a message queue for external CQL executor services to process. Driver distributes work; executors process CQL against FHIR bundles.

4. **No Retry Logic in Driver**: Failed work units are not retried by driver; retry logic is delegated to executor services consuming from the queue.

5. **Configurable Limits**: No hard limits on job size or work unit count; limits are configurable based on deployment needs.

6. **Concurrent Work Processing**: Single job with concurrent processing of multiple work units via distributed queue and horizontally-scaled executor instances.

7. **Multi-Source Data Support**: Files can be read from cloud storage (Google Cloud Storage, Amazon S3) or local file systems, with automatic protocol detection and unified handling.

8. **Compression Support**: Both plain and gzip-compressed NDJSON files are supported transparently, with automatic detection based on file extension.

9. **Large Line Handling**: System supports FHIR bundle files with individual lines up to 10MB in size, accommodating large patient records and complex measure bundles.

10. **Composable File Processing**: File reading is implemented using composable patterns that allow any combination of storage sources (local/cloud) and encodings (plain/gzip) without requiring specialized implementations for each combination.

11. **Manifest-Based or Auto-Discovery**: Jobs can specify data files via manifest files or rely on automatic directory scanning with pattern matching.

12. **Intelligent Batch Splitting**: Work units are created with configurable batch sizes and threshold-based splitting to prevent creating excessively small batches that would reduce processing efficiency.

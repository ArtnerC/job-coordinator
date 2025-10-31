# Feature Specification: Job Driver Service

**Feature Branch**: `001-job-driver`  
**Created**: 2025-10-02  
**Status**: Draft  
**Input**: User description: "first thing we are going to build is a job-driver service. we are creating the spec for that code only right now. it will take input params and distribute work on a queue for a scaling job to read and then wait for job completion to report status to the caller."

---

## ⚡ Quick Guidelines
- ✅ Focus on WHAT users need and WHY
- ❌ Avoid HOW to implement (no tech stack, APIs, code structure)
- 👥 Written for business stakeholders, not developers

---

## User Scenarios & Testing *(mandatory)*

### Primary User Story

A system or user needs to process a large batch of quality measure calculations against patient FHIR bundles. Rather than processing everything synchronously (which could timeout or overwhelm resources), they submit a job request to the Job Driver service. The Job Driver accepts the request, breaks down the work into smaller units, distributes these units to worker processes via a queue, monitors progress, and returns the final status when all work completes.

### Acceptance Scenarios

1. **Given** a valid job request with input parameters, **When** submitted to the Job Driver, **Then** the service MUST accept the request, assign a unique job ID, distribute work units to the queue, and return the job ID immediately

2. **Given** a job has been submitted and work distributed, **When** workers process the queue items, **Then** the Job Driver MUST track completion status of each work unit

3. **Given** all work units for a job have completed successfully, **When** the caller queries job status, **Then** the service MUST return "completed" status with summary results

4. **Given** some work units fail during processing, **When** the job completes, **Then** the service MUST return status indicating partial failure with details on what succeeded and what failed

5. **Given** a caller wants to check progress, **When** querying job status while work is in progress, **Then** the service MUST return current completion percentage and status

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

- **FR-004**: System MUST break down accepted jobs into discrete work units suitable for distribution

- **FR-005**: System MUST distribute work units to a queue where worker processes can retrieve them

- **FR-006**: System MUST track the status of each work unit (pending, in-progress, completed, failed)

- **FR-007**: System MUST aggregate work unit statuses to determine overall job status

- **FR-008**: System MUST provide job status queries that return: job ID, overall status (pending/in-progress/completed/failed), completion percentage, start time, end time (if completed)

- **FR-009**: System MUST wait for all work units to complete or fail before marking job as complete

- **FR-010**: System MUST handle partial failures gracefully, reporting which work units succeeded and which failed

- **FR-011**: System MUST support asynchronous operation - job submission returns immediately with job ID, not blocking until completion

- **FR-012**: System MUST process one job per driver instance; concurrent job handling requires multiple driver instances

- **FR-013**: System MUST support concurrent processing of work units within a single job via distributed queue

- **FR-014**: System MUST delegate work unit execution to external executors that consume from the distributed queue; driver does not retry failed work units

- **FR-015**: System MUST support configurable job sizes with no hard limits on work units per job or total payload size

- **FR-016**: System MUST keep job status in memory and remain running for a configurable period (default 10 minutes) after job completion to allow status queries

- **FR-017**: System MUST shut down automatically after the status retention period expires following job completion

- **FR-018**: System MUST log all job state transitions for audit and debugging purposes

### Key Entities

- **Job**: Represents a complete work request from submission to completion. Contains unique job ID, input parameters, overall status, start/end timestamps, list of associated work units, and summary results.

- **Work Unit**: Represents a single discrete piece of work within a job. Contains work unit ID, parent job ID, input data/parameters, status (pending/in-progress/completed/failed), assigned worker identifier (if in progress), result data (on completion), and error details (on failure).

- **Job Status Query**: Request to retrieve current state of a job. Contains job ID for lookup, returns job metadata, current status, progress metrics, and completion details.

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

The following key architectural decisions were made during specification:

1. **Single Job Per Instance**: Each driver instance processes one job. Multiple jobs require multiple driver instances. Job ID can be provided via environment variable or auto-generated.

2. **In-Memory Status with TTL**: Job status is kept in memory for a configurable period (default 10 minutes) after completion, then driver shuts down. Future enhancement will persist status to metadata database.

3. **Distributed Queue Model**: Work units are distributed to a queue for external executor services to process. driver does not execute work itself.

4. **No Retry Logic in driver**: Failed work units are not retried by driver; retry logic is delegated to executor services consuming from the queue.

5. **Configurable Limits**: No hard limits on job size or work unit count; limits are configurable based on deployment needs.

6. **Concurrent Work Processing**: Single job with concurrent processing of multiple work units via distributed queue and multiple executor instances.

# Implementation Plan: Job Driver Service

**Branch**: `001-job-driver` | **Date**: 2025-10-02 (Updated: 2025-10-31) | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-job-driver/spec.md`
**Organization**: CVS Health - Aetna Digital Quality Measure Evaluation
**Module**: `github.com/cvs-health-source-code/digital-qme-system/job-driver`

## Execution Flow (/speckit.plan command scope)
```
1. Load feature spec from Input path
   → If not found: ERROR "No feature spec at {path}"
2. Fill Technical Context (scan for NEEDS CLARIFICATION)
   → Detect Project Type from file system structure or context (web=frontend+backend, mobile=app+api)
   → Set Structure Decision based on project type
3. Fill the Constitution Check section based on the content of the constitution document.
4. Evaluate Constitution Check section below
   → If violations exist: Document in Complexity Tracking
   → If no justification possible: ERROR "Simplify approach first"
   → Update Progress Tracking: Initial Constitution Check
5. Execute Phase 0 → research.md
   → If NEEDS CLARIFICATION remain: ERROR "Resolve unknowns"
6. Execute Phase 1 → contracts, data-model.md, quickstart.md, agent-specific template file (e.g., `CLAUDE.md` for Claude Code, `.github/copilot-instructions.md` for GitHub Copilot, `GEMINI.md` for Gemini CLI, `QWEN.md` for Qwen Code or `AGENTS.md` for opencode).
7. Re-evaluate Constitution Check section
   → If new violations: Refactor design, return to Phase 1
   → Update Progress Tracking: Post-Design Constitution Check
8. Plan Phase 2 → Describe task generation approach (DO NOT create tasks.md)
9. STOP - Ready for /tasks command
```

**IMPORTANT**: The /speckit.plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

The Job Driver service distributes quality measure calculation work across FHIR patient bundles by breaking down NDJSON files into batches and distributing work units to a queue. It processes one job per instance, tracks work unit completion, exposes REST endpoints for job control (/start, /pause, /cancel, /status), and supports multiple output destinations (GCP Pub/Sub, stdout, file). The service includes intelligent batch splitting based on line counts, manifest-based or auto-discovery file processing, and support for both plain and gzip-compressed files from cloud storage (GCS, S3) or local filesystems. Large individual lines (up to 10MB) are handled via configurable buffers for FHIR bundle processing.

## Technical Context

**Language/Version**: Go 1.25.3  
**Primary Dependencies**: 
  - gocloud.dev v0.43.0 (cloud storage abstraction)
  - cloud.google.com/go/pubsub/v2 v2.2.0 (GCP Pub/Sub)
  - github.com/spf13/viper v1.21.0 (configuration)
  - github.com/google/uuid v1.6.0 (job IDs)
**Storage**: Cloud storage (GCS, S3), local filesystem, NDJSON files (plain and gzip)  
**Testing**: Go testing framework with table-driven tests  
**Target Platform**: Linux containers (Alpine), Docker, Google Cloud Run  
**Project Type**: Single service (monorepo structure: `job-driver/`)  
**Performance Goals**: 
  - 100,000+ work units distributed per minute
  - Line counting: 2-7 GB/s throughput (500k-2M lines/sec)
  - Support files with 10MB lines without memory issues
  - <200ms REST API response times for status queries
**Constraints**: 
  - Single job per driver instance
  - In-memory job state (10-minute TTL after completion)
  - Composable file readers (no type explosion for source/encoding combinations)
  - Docker image <100MB
**Scale/Scope**: 
  - 1M+ FHIR bundles per job
  - Files: 1-1000 per job
  - Batch sizes: 500 lines (configurable)
  - Target: Aetna population health quality measure evaluation

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Demo Often

- [x] Feature reaches demonstrable state within one development cycle
- [x] Demo environment mirrors production (Docker container, cloud storage, sample FHIR bundles)
- [x] Stakeholder feedback loop ≤ 2 weeks
- [x] All outputs are visible and testable with realistic scenarios (work units to stdout/file/queue)

### II. Test Always (NON-NEGOTIABLE)

- [x] Unit tests cover: batch splitting, file discovery, line counting, work unit creation
- [x] Integration tests cover: REST API contracts, end-to-end job processing, cloud storage
- [x] Test data includes representative FHIR bundles (7 sample files) and measure definitions (CMS-125, CMS-130)
- [x] All tests passing: 180 total tests (65 driver/API + 71 processor + 38 integration + 6 E2E)
- [x] TDD approach: tests written BEFORE implementation

### III. Reuse - Don't Reinvent

- [x] Evaluated existing libraries/tools before custom implementation
- [x] Using established CQL engines (e.g., cql-engine, fhir-cql) if applicable - N/A: driver only distributes, doesn't execute CQL
- [x] Using FHIR libraries (HAPI FHIR, Google FHIR SDK) for FHIR handling - N/A: driver reads files as opaque NDJSON lines
- [x] Preferring GCP-native services over custom orchestration: Using gocloud.dev for storage abstraction, GCP Pub/Sub for queue
- [x] Build vs. buy decision documented in ADRs if custom code required

### IV. Reduce Iteration Time

- [x] Local dev supports: hot-reload (go build), local file testing, sample FHIR/CQL data (testdata/)
- [x] Build times optimized: Multi-stage Docker build with layer caching, parallel tests, fast Go compilation
- [x] Deployment pipeline: Docker build <2 minutes, tests complete in ~15 seconds
- [x] Code review: GitHub workflow with branch protection
- [x] Infrastructure: Terraform with plan previews before apply

### V. User/Dev Experience is Key

**End Users**:

- [x] Work units are structured JSON with clear fields (file paths, line ranges, measure references)
- [x] Error messages are descriptive with context (file paths, line numbers, validation errors)
- [x] REST API documented with clear endpoints (/start, /pause, /cancel, /status)
- [x] Performance target: measure execution <30s for typical populations - N/A: driver only distributes work, doesn't execute measures

**Developers**:

- [x] Onboarding documented: README, extensive spec docs (5 files: spec, plan, data-model, contracts, research)
- [x] Local dev setup: Go installed, `go build ./cmd/driver`, runs immediately
- [x] Error messages include context: file paths, line counts, configuration values
- [x] Documentation: 6 doc files (api.md, cloud-storage-integration.md, configuration.md, container.md, deployment.md, development.md)
- [x] Deployment: Dockerized, Terraform configs, no manual steps

## Project Structure

### Documentation (this feature)

```text
specs/[###-feature]/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```text
job-driver/
├── cmd/
│   └── driver/
│       └── main.go   # Entry point
├── internal/
│   ├── api/          # REST API handlers + routing
│   ├── config/       # Configuration management
│   ├── distributor/  # Pub/Sub, stdout, file distributors
│   ├── driver/       # Job orchestration logic
│   ├── models/       # Data models (Job, WorkUnit, etc.)
│   ├── processor/    # File processing, batch splitting
│   └── storage/      # Cloud storage abstraction
├── test/
│   └── integration/  # Integration tests
├── testdata/
│   ├── bundles/      # Sample FHIR NDJSON files
│   └── measures/     # Sample measure definitions
├── deployments/
│   ├── docker/       # Dockerfile
│   └── terraform/    # GCP infrastructure
├── docs/             # Technical documentation
├── go.mod
└── go.sum
```

**Structure Decision**: Single Go service in monorepo under `job-driver/` directory. 
- Standard Go project layout with `cmd/` for entry points and `internal/` for private packages
- Integration tests separated from unit tests (which live alongside code)
- Cloud-native deployment with Docker and Terraform
- Configuration via Viper (env vars, config files, CLI flags)

## Phase 0: Outline & Research ✅ COMPLETE

All technical decisions have been researched and documented in `research.md`:

**Key Research Decisions**:
1. **Language Choice**: Go 1.25.3 for performance, concurrency, cloud-native tooling
2. **Cloud Storage**: gocloud.dev for storage abstraction (GCS, S3, local)
3. **Message Queue**: GCP Pub/Sub for work distribution
4. **File Format**: NDJSON with gzip compression support
5. **Configuration**: Viper for flexible configuration (env, files, flags)
6. **Large File Handling**: bufio.Scanner with 10MB max buffer for FHIR bundles
7. **Batch Splitting**: Configurable batch sizes with threshold-based merging
8. **Composable Readers**: multiCloser pattern for any source/encoding combination

**No NEEDS CLARIFICATION remaining** - all unknowns resolved during implementation.

**Output**: ✅ research.md complete

## Phase 1: Design & Contracts ✅ COMPLETE

**Completed Artifacts**:

1. **Data Model** (`data-model.md`): ✅
   - Job entity: ID, status, config, metrics, timestamps
   - WorkUnit entity: ID, file specs, line ranges, measures
   - FileSpec entity: path, start/end line pointers
   - State transitions: Pending → Running → Completed/Failed/Cancelled

2. **API Contracts** (`contracts/rest-api.md`): ✅
   - REST endpoints: POST /start, POST /pause, POST /cancel, GET /status, GET /health
   - Request/response schemas documented
   - Error responses defined
   - Work unit schema: (`contracts/work-unit-schema.md`): ✅

3. **Contract Tests**: ✅
   - Integration tests cover all API endpoints
   - Schema validation tests passing
   - End-to-end tests: 6 scenarios covering job lifecycle

4. **Test Scenarios** (`quickstart.md`): ✅
   - Quick start guide with sample commands
   - Docker build and run instructions
   - Configuration examples

5. **Agent Context**: ✅
   - Technology stack documented
   - Recent changes tracked
   - Architecture patterns recorded

**Output**: ✅ All Phase 1 artifacts complete and validated

## Phase 2: Task Planning ✅ COMPLETE

**Task Generation**: All tasks documented in `tasks.md` (40+ tasks total)

**Task Categories Completed**:
1. **Setup & Infrastructure** (3 tasks): ✅ Go module, structure, Docker
2. **Models** (5 tasks): ✅ Job, WorkUnit, Config, validations
3. **Configuration** (3 tasks): ✅ Viper integration, env vars, validation
4. **File Processing** (8 tasks): ✅ Line counting, discovery, manifest parsing, gzip support
5. **Batch Splitting** (3 tasks): ✅ Algorithm, threshold merging, tests
6. **Distributors** (7 tasks): ✅ Stdout, File, Pub/Sub, factory pattern
7. **Core Driver Logic** (4 tasks): ✅ Orchestration, state management, TTL
8. **REST API** (8 tasks): ✅ Handlers, routing, control endpoints
9. **Integration Tests** (6 tasks): ✅ E2E scenarios, contracts
10. **Documentation** (5 tasks): ✅ API docs, deployment guides

**Implementation Status**: ✅ 100% COMPLETE
- 180 tests passing (65 driver/API + 71 processor + 38 integration + 6 E2E)
- Binary compiles and runs
- Docker image builds (99.4MB)
- All Constitution checks passing

**Output**: ✅ tasks.md complete, all tasks implemented

## Phase 3-5: Implementation & Validation ✅ COMPLETE

**Phase 3: Task Execution**: ✅ COMPLETE
- All 40+ tasks from tasks.md implemented
- TDD approach: tests written before implementation
- Parallel execution where possible (independent modules)

**Phase 4: Implementation**: ✅ COMPLETE  
- Full implementation following constitutional principles
- Code reviews completed
- All tests passing
- Docker image built and validated

**Phase 5: Validation**: ✅ COMPLETE
- ✅ All 180 tests passing
- ✅ Quickstart.md validated (Docker build/run works)
- ✅ Performance validated:
  - Line counting: 2-7 GB/s throughput
  - 10MB lines handled without errors
  - REST API <200ms response times
- ✅ Integration validated with cloud storage (GCS simulation via file://)
- ✅ Gzip compression/decompression working
- ✅ Composable file readers implemented

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |


## Progress Tracking

**Phase Status**:

- [x] Phase 0: Research complete (/plan command)
- [x] Phase 1: Design complete (/plan command)
- [x] Phase 2: Task planning complete (/plan command)
- [x] Phase 3: Tasks generated (/tasks command)
- [x] Phase 4: Implementation complete
- [x] Phase 5: Validation passed

**Gate Status**:

- [x] Initial Constitution Check: PASS
- [x] Post-Design Constitution Check: PASS
- [x] All NEEDS CLARIFICATION resolved
- [x] No complexity deviations - all constitutional principles followed

**Recent Updates** (2025-10-31):

- ✅ Renamed from job-coordinator to job-driver (better reflects single-job-per-instance pattern)
- ✅ Updated module path to CVS Health organization: `github.com/cvs-health-source-code/digital-qme-system/job-driver`
- ✅ Implemented composable file readers (multiCloser pattern) for cloud storage + gzip
- ✅ Upgraded to Go 1.25.3
- ✅ Optimized Docker image (99.4MB final size)
- ✅ All 180 tests passing
- ✅ Renamed spec directory: 001-first-thing-we → 001-job-driver

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*

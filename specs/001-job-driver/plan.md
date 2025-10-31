
# Implementation Plan: Job Driver Service

**Branch**: `001-job-driver` | **Date**: 2025-10-03 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `C:\Users\Artne\Dev\dqme\specs\001-job-driver\spec.md`

## Execution Flow (/plan command scope)
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
9. STOP - Ready for /speckit.tasks command
```

**IMPORTANT**: The /speckit.plan command STOPS at step 7. Phases 2-4 are executed by other commands:
- Phase 2: /speckit.tasks command creates tasks.md
- Phase 3-4: Implementation execution (manual or via tools)

## Summary

The Job Driver service distributes quality measure calculation work across FHIR patient bundles by breaking down NDJSON files into batches and distributing work units to a queue. It processes one job per instance, tracks work unit completion, exposes REST endpoints for job control (/start, /pause, /cancel, /status), and supports multiple output destinations (GCP Pub/Sub, stdout, file). The service includes intelligent batch splitting based on line counts, manifest-based or auto-discovery file processing, and optional scale testing mode for load simulation.

## Technical Context
**Language/Version**: Go 1.21+ (latest stable)  
**Primary Dependencies**: 
  - `github.com/spf13/cobra` or `flag` for CLI/config
  - `cloud.google.com/go/pubsub` for GCP Pub/Sub
  - `github.com/gorilla/mux` or `net/http` for REST endpoints
  - Go standard library for file I/O, concurrency
**Storage**: In-memory state (job status, work units); NDJSON files on disk; GCP Pub/Sub queue  
**Testing**: Go testing framework (`testing` package), table-driven tests, testify/assert  
**Target Platform**: Linux container (Cloud Run on GCP), Docker/OCI-compliant  
**Project Type**: Single service (monorepo structure: `job-driver/`)  
**Performance Goals**: 
  - Process file manifests and split batches concurrently
  - Handle thousands of NDJSON files efficiently
  - Minimal memory footprint (stream file line counting, no full file loads)
  - REST API response times <100ms for status queries
**Constraints**: 
  - Single job per driver instance
  - No persistent storage (in-memory only)
  - Configurable TTL after completion (default 10 min) before shutdown
  - Must support GCP Pub/Sub, stdout, and file output modes
**Scale/Scope**: 
  - Support processing millions of patient records across thousands of files
  - Configurable batch sizes (default 500 lines per batch)
  - REST API for job lifecycle management
  - Scale testing mode to simulate arbitrary workload volumes

## Constitution Check
*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

### I. Demo Often

- [x] Feature reaches demonstrable state within one development cycle - Single service, deployable to Cloud Run
- [x] Demo environment mirrors production (GCP resources, FHIR data, CQL execution) - Uses GCP Pub/Sub, sample NDJSON files
- [x] Stakeholder feedback loop ≤ 2 weeks - REST API allows immediate testing and feedback
- [x] All outputs are visible and testable with realistic scenarios - Status endpoint, stdout/file modes for visibility

### II. Test Always (NON-NEGOTIABLE)

- [x] Unit tests cover: business logic, CQL parsing, FHIR processing, measure calculations - Will test batch splitting, file processing, config parsing
- [x] Integration tests cover: service contracts, GCP interactions, end-to-end pipelines - Pub/Sub integration, REST API endpoints
- [x] Test data includes representative FHIR bundles and CQL measures - Sample NDJSON bundles and measure files
- [x] CI enforces: all tests pass, 80%+ coverage on core logic, no skipped tests - Standard Go testing with coverage
- [x] TDD approach: tests written BEFORE implementation - Will follow TDD discipline

### III. Reuse - Don't Reinvent

- [x] Evaluated existing libraries/tools before custom implementation - Using Go stdlib, official GCP SDK
- [x] Using established CQL engines (e.g., cql-engine, fhir-cql) if applicable - N/A: driver only distributes, doesn't execute CQL
- [x] Using FHIR libraries (HAPI FHIR, Google FHIR SDK) for FHIR handling - N/A: driver reads files as opaque NDJSON lines
- [x] Preferring GCP-native services over custom orchestration - Using Cloud Pub/Sub natively, deploying to Cloud Run
- [x] Build vs. buy decision documented in ADRs if custom code required - Will document batch splitting algorithm choice

### IV. Reduce Iteration Time

- [x] Local dev supports: hot-reload, mocked GCP services, sample FHIR/CQL data - Go hot-reload with `air`, Pub/Sub emulator, sample files
- [x] Build times optimized: layer caching, incremental builds, parallel tests - Go compile is fast, Docker multi-stage builds
- [x] Deployment pipeline completes in <15 minutes for dev/staging - Simple Go binary, Cloud Run deploys in minutes
- [x] Code review starts within 4 hours, approvals within 24 hours - Team process commitment
- [x] Infrastructure changes have plan previews before apply - Terraform for GCP resources with plan step

### V. User/Dev Experience is Key

**End Users**:

- [x] Measure results are clear, actionable, and auditable - Status endpoint shows progress, work unit tracking
- [x] Error messages are human-readable with corrective actions - Config validation with helpful messages
- [x] APIs have consistent, documented interfaces (OpenAPI specs) - REST API with OpenAPI spec
- [x] Performance target: measure execution <30s for typical populations - N/A: driver only distributes work, doesn't execute measures

**Developers**:

- [x] Onboarding documented: setup scripts, README, sample data - Will include setup guide, sample configs, test data
- [x] Local dev setup completes in <30 minutes - Go install + deps + Pub/Sub emulator
- [x] Error messages include context: request IDs, traces, resource identifiers - Structured logging with job ID, work unit IDs
- [x] Documentation co-located: inline comments, architecture diagrams - Code comments, architecture diagram in docs/
- [x] Deployment processes are scripted (no manual console clicks) - Terraform + Cloud Build automation

## Project Structure

### Documentation (this feature)
```
specs/001-job-driver/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output (REST API contracts)
└── tasks.md             # Phase 2 output (/tasks command)
```

### Source Code (repository root)
```
job-driver/
├── cmd/
│   └── driver/
│       └── main.go          # Entry point, CLI setup
├── internal/
│   ├── config/
│   │   └── config.go        # Configuration management
│   ├── models/
│   │   ├── job.go           # Job entity
│   │   └── workunit.go      # WorkUnit entity
│   ├── processor/
│   │   ├── file_processor.go    # File discovery & line counting
│   │   ├── batch_splitter.go    # Batch splitting logic
│   │   └── manifest.go          # Manifest handling
│   ├── distributor/
│   │   ├── interface.go     # Distributor interface
│   │   ├── pubsub.go        # GCP Pub/Sub implementation
│   │   ├── stdout.go        # Stdout implementation
│   │   └── file.go          # File output implementation
│   ├── api/
│   │   ├── handlers.go      # REST API handlers
│   │   └── router.go        # HTTP routing
│   └── driver/
│       └── driver.go   # Core coordination logic
├── pkg/
│   └── (shared utilities if needed)
├── test/
│   ├── integration/
│   │   ├── pubsub_test.go
│   │   └── api_test.go
│   └── testdata/
│       ├── sample_bundles/  # Sample NDJSON files
│       └── sample_measures/ # Sample measure definitions
├── docs/
│   ├── architecture.md      # Architecture diagrams
│   └── api-spec.yaml        # OpenAPI specification
├── deployments/
│   ├── terraform/           # GCP infrastructure
│   └── docker/
│       └── Dockerfile       # Multi-stage Go build
├── go.mod
├── go.sum
└── README.md
```

**Structure Decision**: Single Go service in monorepo under `job-driver/` directory. 
Following standard Go project layout with `cmd/` for entry points, `internal/` for private 
packages, and `test/` for integration tests. This structure supports clean separation of 
concerns: config, models, processing logic, distribution strategies, and API handling.

## Phase 0: Outline & Research
1. **Extract unknowns from Technical Context** above:
   - For each NEEDS CLARIFICATION → research task
   - For each dependency → best practices task
   - For each integration → patterns task

2. **Generate and dispatch research agents**:
   ```
   For each unknown in Technical Context:
     Task: "Research {unknown} for {feature context}"
   For each technology choice:
     Task: "Find best practices for {tech} in {domain}"
   ```

3. **Consolidate findings** in `research.md` using format:
   - Decision: [what was chosen]
   - Rationale: [why chosen]
   - Alternatives considered: [what else evaluated]

**Output**: research.md with all NEEDS CLARIFICATION resolved

## Phase 1: Design & Contracts
*Prerequisites: research.md complete*

1. **Extract entities from feature spec** → `data-model.md`:
   - Entity name, fields, relationships
   - Validation rules from requirements
   - State transitions if applicable

2. **Generate API contracts** from functional requirements:
   - For each user action → endpoint
   - Use standard REST/GraphQL patterns
   - Output OpenAPI/GraphQL schema to `/contracts/`

3. **Generate contract tests** from contracts:
   - One test file per endpoint
   - Assert request/response schemas
   - Tests must fail (no implementation yet)

4. **Extract test scenarios** from user stories:
   - Each story → integration test scenario
   - Quickstart test = story validation steps

5. **Update agent file incrementally** (O(1) operation):
   - Run `.specify/scripts/powershell/update-agent-context.ps1 -AgentType copilot`
     **IMPORTANT**: Execute it exactly as specified above. Do not add or remove any arguments.
   - If exists: Add only NEW tech from current plan
   - Preserve manual additions between markers
   - Update recent changes (keep last 3)
   - Keep under 150 lines for token efficiency
   - Output to repository root

**Output**: data-model.md, /contracts/*, failing tests, quickstart.md, agent-specific file

## Phase 2: Task Planning Approach
*This section describes what the /speckit.tasks command will do - DO NOT execute during/speckit.plan*

**Task Generation Strategy**:

1. **From Data Model** (`data-model.md`):
   - Job entity → `internal/models/job.go` [P]
   - WorkUnit entity → `internal/models/workunit.go` [P]
   - JobConfig entity → `internal/config/config.go` [P]
   - FileManifest → `internal/processor/manifest.go` [P]
   - JobStatusResponse → `internal/api/response.go` [P]

2. **From Research Decisions** (`research.md`):
   - Viper configuration setup → `internal/config/config.go`
   - File line counting → `internal/processor/file_processor.go` [P]
   - Batch splitting algorithm → `internal/processor/batch_splitter.go` [P]
   - Worker pool pattern → `internal/processor/file_processor.go`

3. **From REST API Contract** (`contracts/rest-api.md`):
   - Contract test: POST /job/start [P]
   - Contract test: PUT /job/pause [P]
   - Contract test: PUT /job/cancel [P]
   - Contract test: GET /job/status [P]
   - Contract test: GET /config [P]
   - Contract test: PUT /config [P]
   - API handlers → `internal/api/handlers.go`
   - Config handlers → `internal/api/config_handlers.go` [P]
   - Router setup → `internal/api/router.go`

4. **From Work Unit Schema** (`contracts/work-unit-schema.md`):
   - Contract test: WorkUnit JSON schema validation (measures array) [P]
   - Contract test: WorkUnit JSON schema validation (measures_folder_path) [P]
   - Contract test: Mutually exclusive measures validation [P]
   - Pub/Sub distributor → `internal/distributor/pubsub.go` [P]
   - Stdout distributor → `internal/distributor/stdout.go` [P]
   - File distributor → `internal/distributor/file.go` [P]
   - Distributor interface → `internal/distributor/interface.go`

5. **From Quickstart** (`quickstart.md`):
   - Integration test: Stdout mode end-to-end
   - Integration test: File mode end-to-end
   - Integration test: Pub/Sub mode (with emulator)
   - Integration test: Manual job control via API
   - Integration test: Measures folder path mode
   - Integration test: Whole file mode (batch_size=0)
   - Integration test: Runtime config updates

6. **From New Features** (Research #13-15):
   - Batch size zero mode logic → `internal/processor/batch_splitter.go`
   - Measures folder path support → `internal/models/workunit.go`
   - Config mutation with state validation → `internal/config/mutation.go`
   - Unit test: batch_size=0 creates one work unit per file [P]
   - Unit test: batch_size=0 skips line counting [P]
   - Unit test: measures/measures_folder_path mutual exclusion [P]
   - Unit test: config state-based validation [P]

7. **Core driver Logic**:
   - Main driver → `internal/driver/coordinator.go`
   - Entry point → `cmd/driver/main.go`
   - Graceful shutdown logic

**Ordering Strategy**:

**Phase 3.1 - Setup** (Sequential):
1. Project initialization (`go mod init`)
2. Directory structure creation
3. Dockerfile and Terraform setup

**Phase 3.2 - Tests First (TDD)** (Mostly Parallel):
- Contract tests for REST API [P]
- Contract tests for work unit schema [P]
- Unit tests for batch splitting logic [P]
- Unit tests for line counting [P]
- Integration test scaffolding

**Phase 3.3 - Core Implementation** (Dependency-Ordered):
1. Models (all parallel) [P]
2. Config management
3. File processor + batch splitter
4. Distributors (all parallel) [P]
5. driver logic (depends on 1-4)
6. API handlers (depends on 5)
7. Main entry point (depends on all)

**Phase 3.4 - Integration** (Sequential on driver):
- Wire up all components in coordinator
- Add graceful shutdown
- Implement TTL timer
- API server integration

**Phase 3.5 - Polish** (Mostly Parallel):
- Unit tests for edge cases [P]
- Integration tests execution
- Documentation updates [P]
- Quickstart validation

**Estimated Output**: 40-45 numbered, ordered tasks in tasks.md

**Key Parallel Opportunities**:
- All model files (5 files)
- All distributor implementations (3 files)
- All REST API contract tests (6 endpoints including /config)
- Work unit schema validation tests (measures array + folder path modes)
- Most unit tests (batch splitting, line counting, config validation)

**Critical Path**:
- Models → driver → API → Integration Tests

**IMPORTANT**: This phase is executed by the /speckit.tasks command, NOT by /speckit.plan

## Phase 3+: Future Implementation
*These phases are beyond the scope of the /speckit.plan command*

**Phase 3**: Task execution (/tasks command creates tasks.md)  
**Phase 4**: Implementation (execute tasks.md following constitutional principles)  
**Phase 5**: Validation (run tests, execute quickstart.md, performance validation)

## Complexity Tracking
*Fill ONLY if Constitution Check has violations that must be justified*

No constitutional violations identified. All design decisions align with DQME principles:
- Demo Often: Single service, quickly demonstrable via stdout or API
- Test Always: TDD approach with comprehensive test coverage planned
- Reuse: Using Go stdlib, official GCP SDK, standard patterns
- Reduce Iteration Time: Fast Go builds, local dev with emulators
- User/Dev Experience: Clear configuration, documented APIs, sample data

## Progress Tracking

**Phase Status**:
- [x] Phase 0: Research complete - research.md created with 15 technical decisions (including clarifications)
- [x] Phase 1: Design complete - data-model.md, contracts/, quickstart.md created and updated
- [x] Phase 2: Task planning complete - Detailed task generation strategy documented with new features
- [ ] Phase 3: Tasks generated (/tasks command)
- [ ] Phase 4: Implementation complete
- [ ] Phase 5: Validation passed

**Gate Status**:
- [x] Initial Constitution Check: PASS - All principles satisfied
- [x] Post-Design Constitution Check: PASS - Design aligns with constitution
- [x] All NEEDS CLARIFICATION resolved - No unknowns remain
- [x] All required artifacts generated

**Artifacts Generated**:
- [x] `plan.md` (this file)
- [x] `research.md` (12 technical decisions documented)
- [x] `data-model.md` (7 entities with relationships)
- [x] `contracts/rest-api.md` (4 REST endpoints + health check)
- [x] `contracts/work-unit-schema.md` (JSON schema + distribution methods)
- [x] `quickstart.md` (5 usage scenarios + deployment guide)

---

*Based on Constitution v1.0.0 - See `/.specify/memory/constitution.md`*

**Ready for /speckit.tasks command to generate implementation tasks**
- [ ] All NEEDS CLARIFICATION resolved
- [ ] Complexity deviations documented

---
*Based on Constitution v2.1.1 - See `/memory/constitution.md`*

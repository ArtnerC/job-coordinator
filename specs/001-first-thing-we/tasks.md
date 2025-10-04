# Tasks: Job Coordinator Service

**Feature**: Job Coordinator Service  
**Date**: 2025-01-21  
**Status**: In Progress (39/46 tasks complete - 85%)

**Completed Tasks**:
- ✅ T001-T003: Project setup (go.mod, Dockerfile, Terraform)
- ✅ T004-T013: REST API contract tests and work unit schema tests (38 integration tests)
- ✅ T014-T020: TDD tests (batch splitter, file processor, config, job, workunit, manifest, measures)
- ✅ T021-T022: Job and WorkUnit models with tests
- ✅ T025: Response models
- ✅ T026: Viper-based config loading with CLI flags (commit 2ba494c)
- ✅ T027-T031: File processing implementations (line counter, batch splitter, manifest, measures, file discovery)
- ✅ T032-T035: Distributors (pubsub, stdout, file) with factory

**Remaining Tasks**:
- ⏳ T023-T024: JobConfig and FileManifest models (2 tasks) - Note: JobConfig exists as Config in internal/config
- ⏳ T036-T039: Coordinator, TTL, API handlers, main (4 tasks)
- ⏳ T040-T043: Integration and wiring (4 tasks)
- ⏳ T044-T046: Polish (unit tests, E2E, documentation) (3 tasks)

**Current Status**: 109 tests passing (71 unit + 38 integration), config loading complete, ready for coordinator implementation

## Task List

### Phase 3.1: Project Setup

**T001** [P] Initialize Go module and project structure
- Run `go mod init github.com/dqme/job-coordinator`
- Create directory structure:
  - `job-coordinator/cmd/coordinator/` (main entry point)
  - `job-coordinator/internal/models/` (data models)
  - `job-coordinator/internal/config/` (configuration management)
  - `job-coordinator/internal/api/` (REST API handlers + routing)
  - `job-coordinator/internal/distributor/` (Pub/Sub, stdout, file distributors)
  - `job-coordinator/internal/processor/` (file processing, batch splitting)
  - `job-coordinator/internal/coordinator/` (job orchestration)
  - `job-coordinator/test/integration/` (integration tests)
  - `job-coordinator/testdata/bundles/` (sample NDJSON bundles)
  - `job-coordinator/testdata/measures/` (sample measure definitions)
- Create `.gitignore` with Go patterns
- **Dependencies**: None
- **Artifacts**: Go module initialized, directory structure created

**T002** [P] Create Dockerfile with multi-stage build
- Create `job-coordinator/deployments/docker/Dockerfile`
- Stage 1: Build Go binary with `golang:1.21-alpine`
- Stage 2: Runtime with `alpine:latest`, copy binary
- Set `ENTRYPOINT ["/coordinator"]`
- Optimize for small image size (<20MB)
- **Dependencies**: T001
- **Artifacts**: `deployments/docker/Dockerfile`

**T003** [P] Create Terraform infrastructure configuration
- Create `job-coordinator/deployments/terraform/main.tf`
- Define Pub/Sub topic resource
- Define Cloud Run service resource
- Define IAM roles for Pub/Sub publishing
- Parameterize project ID, region, topic name
- **Dependencies**: T001
- **Artifacts**: `deployments/terraform/main.tf`, `deployments/terraform/variables.tf`

---

### Phase 3.2: Tests First (TDD - CRITICAL)

**⚠️ CRITICAL: ALL tests in this phase MUST be written and passing BEFORE implementing any production code in Phase 3.3.**

#### REST API Contract Tests

**T004** [P] Contract test: POST /job/start
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Start pending job → 200 OK, status=running
  - Start paused job → 200 OK, status=running
  - Start already running job → 400 Bad Request
  - Start completed job → 400 Bad Request
- Mock coordinator with in-memory job state
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestStartJobEndpoint)

**T005** [P] Contract test: PUT /job/pause
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Pause running job → 200 OK, status=paused
  - Pause pending job → 400 Bad Request
  - Pause paused job → 400 Bad Request
  - Pause completed job → 400 Bad Request
- Mock coordinator state transitions
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestPauseJobEndpoint)

**T006** [P] Contract test: PUT /job/cancel
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Cancel pending job → 200 OK, status=cancelled
  - Cancel running job → 200 OK, status=cancelled
  - Cancel paused job → 200 OK, status=cancelled
  - Cancel completed job → 400 Bad Request
- Verify shutdown signal triggered
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestCancelJobEndpoint)

**T007** [P] Contract test: GET /job/status
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Status pending: all fields correct
  - Status running: progress tracking fields populated
  - Status completed: TTL remaining calculated
  - Status failed: errors list populated
- Validate JSON schema matches spec
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestStatusEndpoint)

**T008** [P] Contract test: GET /config
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Retrieve config in pending state
  - Retrieve config in running state
  - Verify all config fields present
  - Verify sensitive fields (if any) redacted
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestGetConfigEndpoint)

**T009** [P] Contract test: PUT /config
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Update batch_size in pending → 200 OK
  - Update batch_size in running → 200 OK (graceful change)
  - Update distributor_type in running → 400 Bad Request
  - Update completion_ttl in any state → 200 OK
  - Update use_measures_path from true to false without measures_to_run → 400 Bad Request
  - Update use_measures_path from true to false with measures_to_run → 200 OK
  - Invalid batch_size (-1) → 400 Bad Request with validation error
  - Verify runtime-modifiable table enforcement
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestPutConfigEndpoint)

#### Work Unit Schema Tests

**T010** [P] Schema test: Measures array mode validation
- Test file: `test/integration/work_unit_schema_test.go`
- Test cases:
  - Valid work unit with measures array → validates
  - measures array with absolute paths → validates
  - measures array empty → validates
  - Verify JSON schema oneOf constraint enforced
- Use JSON schema validator library
- **Dependencies**: T001
- **Artifacts**: `test/integration/work_unit_schema_test.go` (TestMeasuresArrayValidation)

**T011** [P] Schema test: Measures path mode validation
- Test file: `test/integration/work_unit_schema_test.go`
- Test cases:
  - Valid work unit with measures_path → validates
  - measures_path with absolute path → validates
  - measures_path with relative path → validation error
  - Verify JSON schema oneOf constraint enforced
- **Dependencies**: T001
- **Artifacts**: `test/integration/work_unit_schema_test.go` (TestMeasuresPathValidation)

**T012** [P] Schema test: Mutual exclusion constraint
- Test file: `test/integration/work_unit_schema_test.go`
- Test cases:
  - Work unit with BOTH measures and measures_path → validation error
  - Work unit with NEITHER measures nor measures_path → validation error
  - Verify oneOf constraint in JSON schema
- **Dependencies**: T001
- **Artifacts**: `test/integration/work_unit_schema_test.go` (TestMeasuresMutualExclusion)

**T013** [P] Schema test: Whole file mode (batch_size=0)
- Test file: `test/integration/work_unit_schema_test.go`
- Test cases:
  - Work unit with line fields omitted → validates
  - Work unit with start_line but not end_line → validation error
  - Work unit with all line fields present → validates
  - Verify optional line fields logic
- **Dependencies**: T001
- **Artifacts**: `test/integration/work_unit_schema_test.go` (TestWholeFileModeSchema)

#### Core Logic Unit Tests (TDD)

**T014** [P] Unit test: Batch splitting algorithm
- Test file: `internal/processor/batch_splitter_test.go`
- Test cases:
  - File with 1050 lines, batch_size=500, threshold=0.2 → 2 batches (500, 550)
  - File with 1600 lines, batch_size=500, threshold=0.2 → 4 batches (500, 500, 500, 100)
  - File with 480 lines, batch_size=500 → 1 batch (480)
  - batch_size=0 → 1 batch with nil line fields
  - Verify remainder threshold logic
- **Dependencies**: T001
- **Artifacts**: `internal/processor/batch_splitter_test.go`

**T015** [P] Unit test: Line counting performance
- Test file: `internal/processor/file_processor_test.go`
- Test cases:
  - Count 1 million lines in <1s
  - Handle empty files → 0 lines
  - Handle files without trailing newline → correct count
  - File not found → error
  - Use bufio.Scanner streaming approach
- **Dependencies**: T001
- **Artifacts**: `internal/processor/file_processor_test.go` (TestLineCount)

**T016** [P] Unit test: Config validation
- Test file: `internal/config/config_test.go`
- Test cases:
  - Valid config → no errors
  - batch_size < 0 → error
  - remainder_threshold < 0 or > 1 → error
  - base_path not absolute → error
  - distributor_type invalid → error
  - concurrent_file_processors <= 0 → error
  - use_measures_path=false and measures_to_run empty → error
- **Dependencies**: T001
- **Artifacts**: `internal/config/config_test.go` (TestValidateConfig)

**T017** [P] Unit test: Job state transitions
- Test file: `internal/models/job_test.go`
- Test cases:
  - Pending → Running (valid)
  - Running → Paused (valid)
  - Paused → Running (valid)
  - Running → Cancelled (valid)
  - Completed → Running (invalid)
  - Verify state transition rules
- **Dependencies**: T001
- **Artifacts**: `internal/models/job_test.go` (TestJobStateTransitions)

**T018** [P] Unit test: WorkUnit validation
- Test file: `internal/models/workunit_test.go`
- Test cases:
  - Valid work unit with measures array → validates
  - Valid work unit with measures_path → validates
  - Both measures and measures_path → validation error
  - Neither measures nor measures_path → validation error
  - start_line without end_line → validation error
  - end_line < start_line → validation error
  - total_lines != (end_line - start_line + 1) → validation error
  - All line fields nil → validates (whole file mode)
- **Dependencies**: T001
- **Artifacts**: `internal/models/workunit_test.go` (TestWorkUnitValidation)

**T019** [P] Unit test: Manifest parsing
- Test file: `internal/processor/manifest_test.go`
- Test cases:
  - Valid manifest file → list of file paths
  - Empty manifest → empty list
  - Manifest with comments (lines starting with #) → ignore comments
  - Manifest not found → error
  - Each line is relative path
- **Dependencies**: T001
- **Artifacts**: `internal/processor/manifest_test.go` (TestParseManifest)

**T020** [P] Unit test: Measures discovery
- Test file: `internal/processor/measures_discovery_test.go`
- Test cases:
  - Measures directory with .json files → list of absolute paths
  - Empty measures directory → empty list
  - Non-existent measures directory → error
  - Measures manifest parsing → list of specific measure paths
  - Verify absolute path conversion
- **Dependencies**: T001
- **Artifacts**: `internal/processor/measures_discovery_test.go`

---

### Phase 3.3: Core Implementation

**⚠️ CRITICAL: Implementation in this phase can only begin AFTER all Phase 3.2 tests are written and passing.**

#### Data Models

**T021** [P] Implement Job model
- File: `internal/models/job.go`
- Implement Job struct with all fields from data-model.md
- Implement JobStatus enum (pending, running, paused, completed, failed, cancelled)
- Implement state transition validation methods
- Implement completionPercentage() calculation
- Add JSON serialization tags
- **Dependencies**: T017 (TDD test)
- **Artifacts**: `internal/models/job.go`

**T022** [P] Implement WorkUnit model
- File: `internal/models/workunit.go`
- Implement WorkUnit struct with all fields from data-model.md
- Implement WorkUnitStatus enum
- Implement validation method (mutual exclusion, line fields consistency)
- Add JSON serialization tags
- Implement ToJSON() method for distribution
- **Dependencies**: T018 (TDD test)
- **Artifacts**: `internal/models/workunit.go`

**T023** [P] Implement JobConfig model
- File: `internal/models/config.go`
- Implement JobConfig struct with all fields from data-model.md
- Implement validation method
- Implement IsRuntimeModifiable(field string, currentState JobStatus) method
- Add logic for runtime-modifiable table enforcement
- **Dependencies**: T016 (TDD test)
- **Artifacts**: `internal/models/config.go`

**T024** [P] Implement FileManifest model
- File: `internal/models/manifest.go`
- Implement FileManifest struct
- Implement validation method
- Add file path resolution logic
- **Dependencies**: T019 (TDD test)
- **Artifacts**: `internal/models/manifest.go`

**T025** [P] Implement API response models
- File: `internal/models/response.go`
- Implement JobStatusResponse struct (for GET /job/status)
- Implement ConfigResponse struct (for GET /config)
- Implement ErrorResponse struct (for errors)
- Add JSON serialization tags
- **Dependencies**: T004-T009 (API contract tests)
- **Artifacts**: `internal/models/response.go`

#### Configuration Management

**T026** Implement viper-based configuration loader
- File: `internal/config/config.go`
- Use `github.com/spf13/viper` + `pflag` for CLI args and env vars
- Implement LoadConfig() function
- Set precedence: CLI flags > env vars > defaults
- Bind all flags from research.md decision #1
- Implement config validation
- **Dependencies**: T016 (validation test), T023 (JobConfig model)
- **Artifacts**: `internal/config/config.go`

#### File Processing

**T027** Implement file line counter
- File: `internal/processor/file_processor.go`
- Implement CountLines(filePath string) (int, error) using bufio.Scanner
- Handle edge cases: empty files, no trailing newline
- Optimize for performance (<1s for 1M lines)
- **Dependencies**: T015 (line counting test)
- **Artifacts**: `internal/processor/file_processor.go`

**T028** Implement batch splitter
- File: `internal/processor/batch_splitter.go`
- Implement SplitIntoBatches(totalLines, batchSize int, threshold float64) []BatchRange
- Implement remainder threshold logic from research.md decision #3
- Handle batch_size=0 → return single BatchRange with nil line fields
- **Dependencies**: T014 (batch splitting test)
- **Artifacts**: `internal/processor/batch_splitter.go`

**T029** Implement manifest parser
- File: `internal/processor/manifest.go`
- Implement ParseManifest(manifestPath string) ([]string, error)
- Read file line-by-line, skip empty lines and comments
- Return list of relative file paths
- **Dependencies**: T019 (manifest test)
- **Artifacts**: `internal/processor/manifest.go`

**T030** Implement measures discovery
- File: `internal/processor/measures_discovery.go`
- Implement DiscoverMeasures(measuresPath string) ([]string, error)
- List all .json files in directory recursively
- Return absolute paths
- Implement ParseMeasuresManifest(manifestPath string) ([]string, error)
- **Dependencies**: T020 (measures discovery test)
- **Artifacts**: `internal/processor/measures_discovery.go`

**T031** Implement file discovery
- File: `internal/processor/file_discovery.go`
- Implement DiscoverFiles(basePath string, manifestPath *string) ([]string, error)
- If manifest provided: parse and return files
- If no manifest: walk directory tree, find all .ndjson files
- Return relative paths
- **Dependencies**: T027, T029
- **Artifacts**: `internal/processor/file_discovery.go`

#### Distributors (Parallel Implementation)

**T032** [P] Implement Pub/Sub distributor
- File: `internal/distributor/pubsub.go`
- Implement PubSubDistributor struct
- Implement Distribute(workUnit WorkUnit) error method
- Use `cloud.google.com/go/pubsub` SDK
- Batch publishing (groups of 100-500 messages)
- Implement error handling with retry logic
- Add message attributes (job_id, work_unit_id, content_type)
- **Dependencies**: T022 (WorkUnit model)
- **Artifacts**: `internal/distributor/pubsub.go`

**T033** [P] Implement stdout distributor
- File: `internal/distributor/stdout.go`
- Implement StdoutDistributor struct
- Implement Distribute(workUnit WorkUnit) error method
- Serialize work unit to JSON
- Write to stdout as NDJSON (one line per work unit)
- Handle stdout write errors
- **Dependencies**: T022 (WorkUnit model)
- **Artifacts**: `internal/distributor/stdout.go`

**T034** [P] Implement file distributor
- File: `internal/distributor/file.go`
- Implement FileDistributor struct
- Implement Distribute(workUnit WorkUnit) error method
- Append work unit JSON to output file (NDJSON format)
- Create file if doesn't exist, append if exists
- Handle file write errors
- Implement Close() method to flush and close file
- **Dependencies**: T022 (WorkUnit model)
- **Artifacts**: `internal/distributor/file.go`

**T035** Implement distributor factory
- File: `internal/distributor/factory.go`
- Implement NewDistributor(config JobConfig) (Distributor, error)
- Factory pattern based on distributor_type
- Return appropriate distributor instance (Pub/Sub, stdout, file)
- Validate distributor-specific config
- **Dependencies**: T032, T033, T034
- **Artifacts**: `internal/distributor/factory.go`

#### Job Coordinator

**T036** Implement coordinator orchestration logic
- File: `internal/coordinator/coordinator.go`
- Implement Coordinator struct with Job and JobConfig fields
- Implement ProcessJob() method:
  - Discover files (via manifest or auto-discovery)
  - Discover measures (via manifest or auto-discovery)
  - For each file: count lines → split batches → create work units
  - Worker pool for concurrent file processing (bounded by concurrent_file_processors)
  - Distribute work units via distributor
  - Track progress (distributed_count, error_count)
  - Update job status on completion
- Implement Pause(), Resume(), Cancel() methods
- Implement state transition validation
- **Dependencies**: T021, T027, T028, T031, T035
- **Artifacts**: `internal/coordinator/coordinator.go`

**T037** Implement TTL-based shutdown logic
- File: `internal/coordinator/ttl.go`
- Start TTL timer after job completion
- Use context.WithCancel for graceful shutdown
- On TTL expiry: signal shutdown to main
- Handle SIGTERM/SIGINT signals for external termination
- Drain in-flight API requests (5s timeout)
- Close distributor gracefully
- **Dependencies**: T036
- **Artifacts**: `internal/coordinator/ttl.go`

#### REST API

**T038** Implement REST API handlers
- Files:
  - `internal/api/handlers.go` (job control handlers)
  - `internal/api/config_handlers.go` (config handlers)
  - `internal/api/router.go` (route setup)
- Handlers:
  - StartJobHandler: POST /job/start
  - PauseJobHandler: PUT /job/pause
  - CancelJobHandler: PUT /job/cancel
  - StatusHandler: GET /job/status
  - GetConfigHandler: GET /config
  - PutConfigHandler: PUT /config (with runtime-modifiable validation)
  - HealthHandler: GET /health
- Use gorilla/mux for routing
- Middleware for request logging
- Error handling with proper status codes
- JSON request/response serialization
- **Dependencies**: T004-T009 (API contract tests), T021, T023, T025, T036
- **Artifacts**: `internal/api/handlers.go`, `internal/api/config_handlers.go`, `internal/api/router.go`

**T039** Implement main entry point
- File: `cmd/coordinator/main.go`
- Load configuration (CLI args + env vars)
- Initialize coordinator with job config
- Start REST API server (goroutine)
- If auto_start=true: start job immediately
- Wait for shutdown signal (TTL or SIGTERM)
- Graceful shutdown sequence
- **Dependencies**: T026, T036, T037, T038
- **Artifacts**: `cmd/coordinator/main.go`

---

### Phase 3.4: Integration

**T040** Wire all components together
- Update `cmd/coordinator/main.go` to initialize all components
- Coordinator → Distributor factory → Specific distributor
- API handlers → Coordinator references
- Config → All components
- Error propagation end-to-end
- **Dependencies**: T039
- **Artifacts**: Updated `cmd/coordinator/main.go`

**T041** Implement graceful shutdown sequence
- Handle SIGTERM/SIGINT signals
- Stop accepting new API requests
- Wait for in-flight requests (5s timeout)
- Close distributor (drain pending publishes)
- Close API server
- Exit cleanly
- **Dependencies**: T037, T040
- **Artifacts**: Updated `cmd/coordinator/main.go`, `internal/coordinator/shutdown.go`

**T042** Implement TTL timer and completion detection
- After job status → completed: start TTL timer
- During TTL: keep API server running
- On TTL expiry: trigger graceful shutdown
- Update /job/status to show ttl_remaining
- **Dependencies**: T037, T040
- **Artifacts**: Updated `internal/coordinator/ttl.go`

**T043** Start API server with proper lifecycle
- Start HTTP server in goroutine
- Bind to configured port (default 8080)
- Handle server start errors
- Implement server shutdown with timeout
- **Dependencies**: T038, T040
- **Artifacts**: Updated `internal/api/server.go`

---

### Phase 3.5: Polish and Validation

**T044** [P] Write comprehensive unit tests for all packages
- `internal/config/`: Config loading and validation
- `internal/models/`: All model validation logic
- `internal/processor/`: File discovery, line counting, batch splitting, manifest parsing
- `internal/distributor/`: Each distributor (with mocks for GCP SDK)
- `internal/coordinator/`: Job orchestration, state transitions
- `internal/api/`: Handler logic (with httptest)
- Target 80%+ code coverage
- **Dependencies**: T021-T043
- **Artifacts**: Test files across all internal packages

**T045** [P] Write end-to-end integration tests
- Test file: `test/integration/e2e_test.go`
- Test scenarios from quickstart.md:
  - Stdout mode with auto-start
  - Manual job control (start, pause, resume, cancel)
  - File manifests
  - Measures path mode
  - Whole file mode (batch_size=0)
  - Runtime configuration updates
  - Scale testing mode
- Use GCP Pub/Sub emulator for Pub/Sub tests
- Use testdata/ sample files
- Verify work unit schema compliance
- **Dependencies**: T040-T043
- **Artifacts**: `test/integration/e2e_test.go`

**T046** [P] Write documentation
- `README.md`: Overview, quick start, architecture diagram
- `docs/configuration.md`: All config flags and env vars
- `docs/api.md`: REST API documentation (reference to contracts/rest-api.md)
- `docs/development.md`: Local dev setup, running tests
- `docs/deployment.md`: Docker and Cloud Run deployment
- Add code comments for all public APIs
- **Dependencies**: T040-T043
- **Artifacts**: `README.md`, `docs/` directory

---

## Task Dependencies

**Critical Path** (sequential dependencies, no parallelization):
```
T001 → T004-T020 (TDD tests MUST complete first) → T021-T025 (models) → T036 (coordinator) → T038 (API) → T039 (main) → T040 (integration) → T045 (E2E tests)
```

**Parallel Groups**:

**Group 1: Setup (T001-T003)** - Can all run in parallel
- T001: Go module init
- T002: Dockerfile
- T003: Terraform config

**Group 2: TDD Tests (T004-T020)** - Can all run in parallel after T001
- T004-T009: REST API contract tests
- T010-T013: Work unit schema tests
- T014-T020: Core logic unit tests

**Group 3: Models (T021-T025)** - Can all run in parallel after their respective TDD tests
- T021: Job model (after T017)
- T022: WorkUnit model (after T018)
- T023: JobConfig model (after T016)
- T024: FileManifest model (after T019)
- T025: Response models (after T004-T009)

**Group 4: File Processing (T027-T031)** - Can run in parallel after TDD tests
- T027: Line counter (after T015)
- T028: Batch splitter (after T014)
- T029: Manifest parser (after T019)
- T030: Measures discovery (after T020)
- T031: File discovery (after T027, T029)

**Group 5: Distributors (T032-T034)** - Can all run in parallel after T022
- T032: Pub/Sub distributor
- T033: Stdout distributor
- T034: File distributor

**Group 6: Polish (T044-T046)** - Can all run in parallel after T043
- T044: Unit tests
- T045: E2E integration tests
- T046: Documentation

---

## Parallel Execution Examples

**Week 1: Setup + TDD**
- Developer A: T001-T003 (setup)
- Developer B: T004-T009 (API contract tests)
- Developer C: T010-T013 (schema tests)
- Developer D: T014-T020 (unit tests)

**Week 2: Core Implementation**
- Developer A: T021-T025 (models)
- Developer B: T027-T031 (file processing)
- Developer C: T032-T034 (distributors)

**Week 3: Coordination + API**
- Developer A: T036-T037 (coordinator + TTL)
- Developer B: T038-T039 (API + main)

**Week 4: Integration + Polish**
- Developer A: T040-T043 (integration)
- Developer B: T044 (unit tests)
- Developer C: T045 (E2E tests)
- Developer D: T046 (documentation)

---

## Validation Checklist

**Before marking tasks.md complete**:
- [x] All 46 tasks defined with clear descriptions
- [x] Task dependencies documented
- [x] Parallel tasks marked with [P]
- [x] TDD approach enforced (tests before implementation)
- [x] All design artifacts (data-model.md, contracts/, quickstart.md) represented as tasks
- [x] All research decisions (research.md) incorporated into tasks
- [x] Critical path identified
- [x] Estimated 40-45 tasks (actual: 46)
- [x] Tasks ordered: Setup → Tests → Core → Integration → Polish

**Constitution Compliance**:
- ✅ Demo Often: T002 (Dockerfile), T039 (main) enable quick demos
- ✅ Test Always: T004-T020 enforce TDD, T044-T045 ensure coverage
- ✅ Reuse: Using Go stdlib, official GCP SDK, standard patterns (T026, T032, T038)
- ✅ Reduce Iteration Time: Fast Go builds, local dev with emulators (T045)
- ✅ User/Dev Experience: Clear config (T026), documented APIs (T046), sample data (T001)

---

*Ready for implementation execution. Run `/specify execute` to begin.*

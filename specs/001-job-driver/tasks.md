# Tasks: Job Driver Service

**Feature**: Job Driver Service  
**Date**: 2025-01-21 (Updated: 2025-10-31)  
**Status**: Complete (56/56 tasks - 100%)

**Completed Tasks**:
- ✅ T001-T003: Project setup (go.mod, Dockerfile, Terraform)
- ✅ T004-T013: REST API contract tests and work unit schema tests (38 integration tests)
- ✅ T014-T020: TDD tests (batch splitter, file processor, config, job, workunit, manifest, measures)
- ✅ T021-T022: Job and WorkUnit models with tests
- ✅ T023-T024: JobConfig (in models/job.go) and FileManifest (ParseManifest in processor)
- ✅ T025: Response models
- ✅ T026: Viper-based config loading with CLI flags
- ✅ T027-T031: File processing implementations (line counter, batch splitter, manifest, measures, file discovery)
- ✅ T032-T035: Distributors (pubsub, stdout, file) with factory
- ✅ T036: Driver orchestration
- ✅ T037: TTL-based shutdown
- ✅ T038: REST API handlers
- ✅ T039: Main entry point
- ✅ T040-T043: Integration wiring complete
- ✅ T044-T046: Polish (unit tests, E2E, documentation)
- ✅ T047-T050: Cloud storage integration with gocloud.dev and composable readers
- ✅ T051-T052: Large file handling with expanded bufio.Scanner buffers
- ✅ T053-T055: Rename refactoring (coordinator→driver, CVS Health org migration)
- ✅ T056: Go 1.25.3 upgrade and Docker optimization

**Final Status**: 183 tests passing (65 driver/API + 71 processor + 38 integration + 6 E2E + 3 gzip), driver binary compiles and runs, 6 comprehensive documentation files, Docker image 99.4MB, **100% COMPLETE**

**Recent Changes** (Last 3 Weeks):
- Unified Files structure with multi-file and hybrid batching (commit 34212c7)
- Docker updated to Go 1.24, then 1.25.3 (commit 11a0476)
- Composable file readers for cloud+gzip support (commit 6827c1e)
- Large line size tests for 10MB FHIR bundles (commit 9dd092a)
- Renamed job-coordinator to job-driver (commit 8d49bc5)
- Updated module path to CVS Health org (commit d1d5fc2)
- Renamed spec directory: 001-first-thing-we → 001-job-driver (commit d1d5fc2)
- Updated spec.md and plan.md with implementation details (commits 63c7535, 4c76baf)

## Task List

### Phase 3.1: Project Setup

**T001** [P] Initialize Go module and project structure
- Run `go mod init github.com/cvs-health-source-code/digital-qme-system/job-driver`
- Create directory structure:
  - `job-driver/cmd/driver/` (main entry point)
  - `job-driver/internal/models/` (data models)
  - `job-driver/internal/config/` (configuration management)
  - `job-driver/internal/api/` (REST API handlers + routing)
  - `job-driver/internal/distributor/` (Pub/Sub, stdout, file distributors)
  - `job-driver/internal/processor/` (file processing, batch splitting)
  - `job-driver/internal/driver/` (job orchestration)
  - `job-driver/test/integration/` (integration tests)
  - `job-driver/testdata/bundles/` (sample NDJSON bundles)
  - `job-driver/testdata/measures/` (sample measure definitions)
- Create `.gitignore` with Go patterns
- **Dependencies**: None
- **Artifacts**: Go module initialized, directory structure created

**T002** [P] Create Dockerfile with multi-stage build
- Create `job-driver/deployments/docker/Dockerfile`
- Stage 1: Build Go binary with `golang:1.21-alpine`
- Stage 2: Runtime with `alpine:latest`, copy binary
- Set `ENTRYPOINT ["/driver"]`
- Optimize for small image size (<20MB)
- **Dependencies**: T001
- **Artifacts**: `deployments/docker/Dockerfile`

**T003** [P] Create Terraform infrastructure configuration
- Create `job-driver/deployments/terraform/main.tf`
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
- Mock driver with in-memory job state
- **Dependencies**: T001
- **Artifacts**: `test/integration/rest_api_contract_test.go` (TestStartJobEndpoint)

**T005** [P] Contract test: PUT /job/pause
- Test file: `test/integration/rest_api_contract_test.go`
- Test cases:
  - Pause running job → 200 OK, status=paused
  - Pause pending job → 400 Bad Request
  - Pause paused job → 400 Bad Request
  - Pause completed job → 400 Bad Request
- Mock driver state transitions
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

#### Job Driver

**T036** Implement driver orchestration logic
- File: `internal/driver/coordinator.go`
- Implement driver struct with Job and JobConfig fields
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
- **Artifacts**: `internal/driver/coordinator.go`

**T037** Implement TTL-based shutdown logic
- File: `internal/driver/ttl.go`
- Start TTL timer after job completion
- Use context.WithCancel for graceful shutdown
- On TTL expiry: signal shutdown to main
- Handle SIGTERM/SIGINT signals for external termination
- Drain in-flight API requests (5s timeout)
- Close distributor gracefully
- **Dependencies**: T036
- **Artifacts**: `internal/driver/ttl.go`

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
- File: `cmd/driver/main.go`
- Load configuration (CLI args + env vars)
- Initialize driver with job config
- Start REST API server (goroutine)
- If auto_start=true: start job immediately
- Wait for shutdown signal (TTL or SIGTERM)
- Graceful shutdown sequence
- **Dependencies**: T026, T036, T037, T038
- **Artifacts**: `cmd/driver/main.go`

---

### Phase 3.4: Integration

**T040** Wire all components together
- Update `cmd/driver/main.go` to initialize all components
- driver → Distributor factory → Specific distributor
- API handlers → driver references
- Config → All components
- Error propagation end-to-end
- **Dependencies**: T039
- **Artifacts**: Updated `cmd/driver/main.go`

**T041** Implement graceful shutdown sequence
- Handle SIGTERM/SIGINT signals
- Stop accepting new API requests
- Wait for in-flight requests (5s timeout)
- Close distributor (drain pending publishes)
- Close API server
- Exit cleanly
- **Dependencies**: T037, T040
- **Artifacts**: Updated `cmd/driver/main.go`, `internal/driver/shutdown.go`

**T042** Implement TTL timer and completion detection
- After job status → completed: start TTL timer
- During TTL: keep API server running
- On TTL expiry: trigger graceful shutdown
- Update /job/status to show ttl_remaining
- **Dependencies**: T037, T040
- **Artifacts**: Updated `internal/driver/ttl.go`

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
- `internal/driver/`: Job orchestration, state transitions
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

### Phase 3.6: Cloud Storage Integration

**✅ T047** [P] Implement cloud storage abstraction with gocloud.dev
- File: `internal/storage/cloud_storage.go`
- Add `gocloud.dev/blob` dependency for unified storage API
- Implement OpenBucket(ctx, urlPath) to handle gs://, s3://, file:// URLs
- Support GCS (gs://), S3 (s3://), and local file system (file://)
- Handle authentication automatically (service accounts, IAM roles, access keys)
- **Dependencies**: T027
- **Artifacts**: `internal/storage/cloud_storage.go`

**✅ T048** Implement composable multiCloser pattern
- File: `internal/processor/file_processor.go`
- Create `multiCloser` struct to manage stack of closers (reader + multiple closers)
- Implement `Close()` to close all closers in reverse order
- Enables composition of any reader + encoding layers
- **Dependencies**: T027
- **Artifacts**: `internal/processor/file_processor.go` (multiCloser type)

**✅ T049** Implement gzip compression support
- File: `internal/processor/file_processor.go`
- Implement `wrapWithGzip(r io.ReadCloser)` to wrap any reader with gzip decompression
- Use `compress/gzip` package
- Compose gzip over base reader using multiCloser pattern
- Auto-detect gzip based on .gz file extension
- **Dependencies**: T048
- **Artifacts**: `internal/processor/file_processor.go` (wrapWithGzip function)

**✅ T050** [P] Write cloud storage + gzip integration tests
- File: `internal/processor/gzip_test.go`
- Test plain NDJSON files
- Test gzipped NDJSON files (.ndjson.gz)
- Test cloud storage URLs with file:// protocol
- Test cloud storage + gzip combination
- Verify identical line counts for plain vs gzipped files
- **Dependencies**: T047, T049
- **Artifacts**: `internal/processor/gzip_test.go` (TestCloudStorageGzipped)

---

### Phase 3.7: Large File Handling

**✅ T051** Expand bufio.Scanner buffer for large FHIR bundles
- File: `internal/processor/file_processor.go`
- Increase initial buffer from 64KB to 1MB
- Set max buffer size to 10MB (configurable)
- Handle lines up to 10MB without buffer overflow
- Add buffer expansion error handling
- **Dependencies**: T027
- **Artifacts**: Updated `internal/processor/file_processor.go` (CountLines function)

**✅ T052** [P] Write large line size tests
- File: `internal/processor/file_processor_test.go`
- Test files with lines at 64KB boundary (default scanner limit)
- Test files with lines at 512KB
- Test files with lines at 1MB
- Test files with lines at 10MB
- Test mixed file: some lines >64KB, some >512KB
- Verify no buffer overflow errors
- **Dependencies**: T051
- **Artifacts**: `internal/processor/file_processor_test.go` (TestCountLinesLargeLines)

---

### Phase 3.8: Rename Refactoring

**✅ T053** Rename module from job-coordinator to job-driver
- Update go.mod: module path from `github.com/dqme/job-coordinator` to `github.com/dqme/job-driver`
- Rename directories: `cmd/coordinator` → `cmd/driver`, `internal/coordinator` → `internal/driver`
- Update all import paths in .go files (72 files total)
- Update Dockerfile: binary name from `coordinator` to `driver`
- Update Terraform configs: service name references
- Update documentation: all references to "coordinator"
- **Dependencies**: All previous tasks
- **Artifacts**: Renamed directories and updated imports across codebase

**✅ T054** Update module path to CVS Health organization
- Update go.mod: `github.com/dqme/job-driver` → `github.com/cvs-health-source-code/digital-qme-system/job-driver`
- Update all import statements in .go files (30 files)
- Update documentation with CVS Health context
- Create new branch: `001-job-driver`
- **Dependencies**: T053
- **Artifacts**: Updated module path and imports

**✅ T055** Rename spec directory
- Rename `specs/001-first-thing-we` → `specs/001-job-driver`
- Update all references in documentation
- Update branch name references
- Update .specify/ script configurations
- **Dependencies**: T054
- **Artifacts**: Renamed spec directory

---

### Phase 3.9: Go Version Upgrade

**✅ T056** Upgrade to Go 1.25.3 and optimize Docker
- Update go.mod: `go 1.21` → `go 1.25.3`
- Update Dockerfile: `golang:1.21-alpine` → `golang:1.25-alpine`
- Remove Alpine version pin (use `alpine:latest`)
- Rebuild and verify Docker image (<100MB target)
- Run all tests with Go 1.25.3
- Update documentation with new Go version
- **Dependencies**: T053-T055
- **Artifacts**: Updated go.mod, Dockerfile

---

## Task Dependencies

**Critical Path** (sequential dependencies, no parallelization):
```
T001 → T004-T020 (TDD tests MUST complete first) → T021-T025 (models) → T036 (driver) → T038 (API) → T039 (main) → T040 (integration) → T045 (E2E tests)
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

**Group 7: Cloud Storage (T047-T050)** - Can run in parallel after T027
- T047: gocloud.dev integration
- T048: multiCloser pattern (after T027)
- T049: gzip support (after T048)
- T050: Cloud storage + gzip tests (after T047, T049)

**Group 8: Large Files (T051-T052)** - Can run in parallel after T027
- T051: Expand bufio.Scanner buffers
- T052: Large line size tests

**Group 9: Refactoring (T053-T056)** - Sequential after all implementation
- T053: Rename coordinator→driver (after T046)
- T054: CVS Health org migration (after T053)
- T055: Rename spec directory (after T054)
- T056: Go 1.25.3 upgrade (after T055)

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
- Developer A: T036-T037 (driver + TTL)
- Developer B: T038-T039 (API + main)

**Week 4: Integration + Polish**
- Developer A: T040-T043 (integration) ✅
- Developer B: T044 (unit tests) ✅
- Developer C: T045 (E2E tests) ✅
- Developer D: T046 (documentation) ✅

**Week 5: Cloud Storage + Large Files (Post-MVP Enhancements)**
- Developer A: T047-T050 (cloud storage + gzip) ✅ (commit 6827c1e)
- Developer B: T051-T052 (large file handling) ✅ (commit 9dd092a)

**Week 6: Refactoring + Upgrades**
- Developer A: T053 (rename to job-driver) ✅ (commit 8d49bc5)
- Developer A: T054-T055 (CVS Health org + spec rename) ✅ (commit d1d5fc2)
- Developer A: T056 (Go 1.25.3 upgrade) ✅ (commits 11a0476, fcae0ab)

---

## Validation Checklist

**Before marking tasks.md complete**:
- [x] All 56 tasks defined with clear descriptions
- [x] Task dependencies documented
- [x] Parallel tasks marked with [P]
- [x] TDD approach enforced (tests before implementation)
- [x] All design artifacts (data-model.md, contracts/, quickstart.md) represented as tasks
- [x] All research decisions (research.md) incorporated into tasks
- [x] Critical path identified
- [x] Estimated 40-45 tasks initially (actual: 56 including enhancements)
- [x] Tasks ordered: Setup → Tests → Core → Integration → Polish → Cloud Storage → Large Files → Refactoring
- [x] **All tasks implemented and validated (183 tests passing)**
- [x] Post-MVP enhancements completed (cloud storage, gzip, large files)
- [x] Refactoring completed (rename, org migration, Go upgrade)

**Constitution Compliance**:
- ✅ Demo Often: T002 (Dockerfile), T039 (main) enable quick demos
- ✅ Test Always: T004-T020 enforce TDD, T044-T045 ensure coverage
- ✅ Reuse: Using Go stdlib, official GCP SDK, standard patterns (T026, T032, T038)
- ✅ Reduce Iteration Time: Fast Go builds, local dev with emulators (T045)
- ✅ User/Dev Experience: Clear config (T026), documented APIs (T046), sample data (T001)

---

## Implementation Complete

**Final Metrics**:
- 183 tests passing (100% pass rate)
  - 65 driver/API unit tests
  - 71 processor unit tests
  - 38 integration tests
  - 6 E2E tests
  - 3 gzip/cloud storage tests
- 6 comprehensive documentation files
- Binary compiles successfully (Go 1.25.3)
- Docker image: 99.4MB (optimized multi-stage build)
- All integration points validated
- Zero known issues

**Key Implementation Commits**:
- 34212c7: T021-T022 Unified Files structure with FileSpec
- 852680c: T044 Comprehensive unit tests (driver + API)
- d52bdbc: T045 E2E integration tests
- 1fa0ac7: T046 Complete documentation suite
- 11a0476: T056 Docker updated to Go 1.24
- 6827c1e: T047-T049 Composable file readers for cloud + gzip
- 9dd092a: T051-T052 Large line size tests for 10MB FHIR bundles
- 8d49bc5: T053 Rename job-coordinator to job-driver
- d1d5fc2: T054-T055 CVS Health org migration + spec rename
- 63c7535: Updated spec.md with implementation details
- 4c76baf: Updated plan.md with completed phases

**Enhancement Summary**:
- ✅ Cloud storage support (GCS, S3, local) via gocloud.dev
- ✅ Gzip compression with composable reader pattern
- ✅ Large FHIR bundles (10MB lines) handled without errors
- ✅ Proper naming (job-driver reflects single-job architecture)
- ✅ CVS Health organizational alignment
- ✅ Go 1.25.3 with optimized Docker image

*Implementation execution complete. Feature production-ready and validated.*

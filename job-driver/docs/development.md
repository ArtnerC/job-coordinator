# Development Guide

This guide covers local development, testing, and building the Job Coordinator.

## Prerequisites

- **Go 1.21+**: Install from [golang.org](https://golang.org/dl/)
- **Git**: For version control
- **Optional**: Docker for containerized builds

## Project Structure

```
job-coordinator/
├── cmd/
│   └── coordinator/          # Main entry point
│       └── main.go
├── internal/
│   ├── api/                  # REST API handlers
│   ├── config/               # Configuration management
│   ├── coordinator/          # Job orchestration
│   ├── distributor/          # Work distribution (stdout, file, pubsub)
│   ├── models/               # Data models
│   └── processor/            # File processing logic
├── test/
│   └── integration/          # E2E integration tests
├── go.mod                    # Go module definition
├── go.sum                    # Dependency checksums
├── Dockerfile                # Container build definition
└── README.md                 # Project overview
```

## Local Development Setup

### 1. Clone the Repository

```bash
git clone https://github.com/dqme/job-coordinator.git
cd job-coordinator
```

### 2. Install Dependencies

```bash
go mod download
```

This downloads all dependencies listed in `go.mod`.

### 3. Verify Setup

```bash
go build -o coordinator.exe ./cmd/coordinator
```

If successful, you'll have a `coordinator.exe` binary.

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test ./... -cover
```

### Run Tests with Verbose Output

```bash
go test ./... -v
```

### Run Specific Package Tests

```bash
# Test coordinator logic
go test ./internal/coordinator -v

# Test API handlers
go test ./internal/api -v

# Test E2E scenarios
go test ./test/integration -v
```

### Run Specific Test Function

```bash
go test ./internal/coordinator -run TestCoordinatorStateTransitions -v
```

### Disable Test Caching

```bash
go test ./... -count=1
```

Use this when you want to force tests to re-run even if nothing changed.

## Building

### Local Build

```bash
go build -o coordinator.exe ./cmd/coordinator
```

### Build for Different Platforms

**Linux**:
```bash
GOOS=linux GOARCH=amd64 go build -o coordinator ./cmd/coordinator
```

**macOS**:
```bash
GOOS=darwin GOARCH=amd64 go build -o coordinator ./cmd/coordinator
```

**Windows**:
```bash
GOOS=windows GOARCH=amd64 go build -o coordinator.exe ./cmd/coordinator
```

### Build with Version Info

```bash
VERSION=$(git describe --tags --always)
go build -ldflags "-X main.version=$VERSION" -o coordinator.exe ./cmd/coordinator
```

## Running Locally

### Basic Run (Stdout Mode)

```bash
go run ./cmd/coordinator \
  --job-id="local-test-job" \
  --base-path="C:\data\bundles" \
  --measures-to-run="/measures/measure1.json" \
  --distributor-type=stdout \
  --auto-start=true
```

### Run with Hot Reload (Air)

Install Air for hot reload during development:

```bash
go install github.com/cosmtrek/air@latest
```

Create `.air.toml`:

```toml
root = "."
tmp_dir = "tmp"

[build]
cmd = "go build -o ./tmp/coordinator.exe ./cmd/coordinator"
bin = "tmp/coordinator.exe"
include_ext = ["go"]
exclude_dir = ["tmp", "test"]
```

Run with Air:

```bash
air
```

## Debugging

### VS Code Debug Configuration

Create `.vscode/launch.json`:

```json
{
  "version": "0.2.0",
  "configurations": [
    {
      "name": "Debug Coordinator",
      "type": "go",
      "request": "launch",
      "mode": "debug",
      "program": "${workspaceFolder}/cmd/coordinator",
      "args": [
        "--job-id=debug-job",
        "--base-path=C:\\data\\bundles",
        "--measures-to-run=/measures/measure1.json",
        "--distributor-type=stdout",
        "--auto-start=true"
      ],
      "env": {}
    }
  ]
}
```

Press F5 to start debugging.

### Delve (Command Line Debugger)

Install Delve:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

Debug with Delve:

```bash
dlv debug ./cmd/coordinator -- \
  --job-id="debug-job" \
  --base-path="C:\data\bundles" \
  --measures-to-run="/measures/measure1.json" \
  --distributor-type=stdout
```

## Code Quality

### Format Code

```bash
go fmt ./...
```

### Lint Code (golangci-lint)

Install golangci-lint:

```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

Run linter:

```bash
golangci-lint run
```

### Vet Code

```bash
go vet ./...
```

## Common Development Tasks

### Add a New Distributor Type

1. Create new distributor in `internal/distributor/`
2. Implement `Distributor` interface:
   ```go
   type Distributor interface {
       Distribute(ctx context.Context, workUnit models.WorkUnit) error
       Close() error
   }
   ```
3. Register in `distributor.NewDistributor()` factory
4. Add tests in `internal/distributor/*_test.go`
5. Update documentation

### Add a New API Endpoint

1. Add handler in `internal/api/handlers.go`
2. Register route in `internal/api/router.go`
3. Add tests in `internal/api/handlers_test.go`
4. Update `docs/api.md`

### Modify Configuration

1. Update `internal/config/config.go` struct
2. Add CLI flag in `cmd/coordinator/main.go`
3. Update validation in `config.Validate()`
4. Update tests
5. Update `docs/configuration.md`

## Testing Best Practices

### Unit Tests

- Test individual functions/methods in isolation
- Use table-driven tests for multiple scenarios
- Mock external dependencies (filesystem, network)
- Example: `internal/coordinator/coordinator_test.go`

### Integration Tests

- Test interactions between components
- Use real implementations where possible
- Test configuration validation
- Example: `internal/integration/rest_contract_test.go`

### E2E Tests

- Test full workflows from start to finish
- Use temporary directories for file operations
- Use `httptest.NewServer` for API testing
- Clean up resources in `defer` statements
- Example: `test/integration/e2e_test.go`

## Troubleshooting

### Tests Failing

1. **Check Go version**: `go version` (must be 1.21+)
2. **Clear test cache**: `go clean -testcache`
3. **Update dependencies**: `go mod tidy`
4. **Run with verbose output**: `go test ./... -v`

### Build Failing

1. **Check for syntax errors**: `go build ./...`
2. **Verify dependencies**: `go mod verify`
3. **Clean build cache**: `go clean -cache`

### Import Errors

1. **Update module cache**: `go mod download`
2. **Sync dependencies**: `go mod tidy`
3. **Verify `go.mod`**: Check module path matches repo

## Performance Profiling

### CPU Profiling

```bash
go test -cpuprofile cpu.prof -bench . ./internal/processor
go tool pprof cpu.prof
```

### Memory Profiling

```bash
go test -memprofile mem.prof -bench . ./internal/processor
go tool pprof mem.prof
```

### Benchmark Tests

Add benchmark functions:

```go
func BenchmarkProcessFile(b *testing.B) {
    for i := 0; i < b.N; i++ {
        // benchmark code
    }
}
```

Run benchmarks:

```bash
go test -bench . ./internal/processor
```

## Contributing

1. **Fork the repository**
2. **Create feature branch**: `git checkout -b feature/my-feature`
3. **Write tests**: Add tests for new functionality
4. **Run tests**: `go test ./...`
5. **Format code**: `go fmt ./...`
6. **Commit changes**: `git commit -m "feat: add my feature"`
7. **Push branch**: `git push origin feature/my-feature`
8. **Create Pull Request**

## Resources

- [Go Documentation](https://golang.org/doc/)
- [Effective Go](https://golang.org/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Project README](../README.md)
- [API Documentation](api.md)
- [Configuration Guide](configuration.md)

# Distributor Package

This package provides implementations for distributing work units to various backends.

## Distributors

### Stdout Distributor
Writes work units as NDJSON to stdout. Useful for testing and piping to other tools.

```go
dist := NewStdoutDistributor()
err := dist.Distribute(workUnit)
```

### File Distributor
Writes work units as JSON files to a specified directory.

```go
dist, err := NewFileDistributor("/path/to/output")
err = dist.Distribute(workUnit)
```

### PubSub Distributor
Publishes work units to Google Cloud Pub/Sub.

```go
config := PubSubConfig{
    ProjectID: "my-project",
    TopicName: "work-queue",
    BatchSize: 100,
}
dist, err := NewPubSubDistributor(ctx, config)
err = dist.Distribute(workUnit)
```

## Testing

### Unit Tests
Run all unit tests:
```bash
go test ./internal/distributor/... -v
```

### PubSub Integration Tests with Emulator

The PubSub integration tests use the Google Cloud Pub/Sub emulator for local testing without requiring GCP credentials or network access.

#### Prerequisites
1. Install Google Cloud SDK (gcloud): https://cloud.google.com/sdk/docs/install
2. Install the Pub/Sub emulator component:
   ```bash
   gcloud components install pubsub-emulator
   gcloud components update
   ```

#### Running PubSub Tests
The tests will automatically:
1. Check if `gcloud` is available
2. Start the Pub/Sub emulator on `localhost:8085`
3. Set `PUBSUB_EMULATOR_HOST` environment variable
4. Create topics and subscriptions
5. Test message publishing and receiving
6. Clean up the emulator on test completion

Run PubSub integration tests:
```bash
go test ./internal/distributor/... -v -run TestPubSubDistributor
```

#### Manual Emulator Testing
You can also start the emulator manually for development:

```bash
# Start emulator
gcloud beta emulators pubsub start --project=test-project --host-port=localhost:8085

# In another terminal, set environment variable
export PUBSUB_EMULATOR_HOST="localhost:8085"  # Linux/Mac
# or
set PUBSUB_EMULATOR_HOST=localhost:8085      # Windows CMD
# or
$env:PUBSUB_EMULATOR_HOST="localhost:8085"   # PowerShell

# Run your application or tests
go run cmd/coordinator/main.go
```

#### What the Tests Cover
- **TestPubSubDistributor_WithEmulator**: Full integration test - creates topics, publishes messages, verifies delivery
- **TestPubSubDistributor_CreateWithEmulator**: Tests distributor creation and lifecycle
- **TestPubSubDistributor_NoEmulator**: Verifies graceful failure without credentials/emulator
- **TestPubSubDistributor_BatchPublishing**: Tests batch publishing behavior with configurable batch sizes

#### Test Behavior
- If `gcloud` is not installed: Tests are skipped with informative message
- If emulator fails to start: Test fails with timeout error
- If emulator starts successfully: Full integration tests run

#### Troubleshooting
- **"gcloud not found"**: Install Google Cloud SDK
- **Emulator fails to start**: Check if port 8085 is available, try different port
- **Timeout waiting for emulator**: Increase timeout in test or check firewall settings
- **Tests hang**: Emulator process might not have been killed, manually kill `java` process running emulator

## Factory Pattern

Use the factory to create distributors based on configuration:

```go
cfg := config.Config{
    DistributorType: "pubsub",
    DistributorConfig: map[string]string{
        "project_id": "my-project",
        "topic_name": "work-queue",
    },
}

dist, err := NewDistributor(ctx, cfg)
```

Supported types:
- `"stdout"` - Stdout distributor
- `"file"` - File distributor (requires `output_path` in config)
- `"pubsub"` - PubSub distributor (requires `project_id` and `topic_name` in config)

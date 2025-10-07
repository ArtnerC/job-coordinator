package distributor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"time"

	"github.com/dqme/job-coordinator/internal/models"
)

// StdoutDistributor writes work units as NDJSON to stdout
type StdoutDistributor struct {
	writer    io.Writer
	delayMs   int // Delay in milliseconds between work unit distributions
}

// NewStdoutDistributor creates a new stdout distributor
func NewStdoutDistributor() *StdoutDistributor {
	return &StdoutDistributor{
		writer:  os.Stdout,
		delayMs: 0,
	}
}

// NewStdoutDistributorWithWriter creates a distributor with custom writer (for testing)
func NewStdoutDistributorWithWriter(writer io.Writer) *StdoutDistributor {
	return &StdoutDistributor{
		writer:  writer,
		delayMs: 0,
	}
}

// NewStdoutDistributorWithConfig creates a distributor with configuration options
func NewStdoutDistributorWithConfig(writer io.Writer, config map[string]string) *StdoutDistributor {
	// Default to stdout if no writer provided
	if writer == nil {
		writer = os.Stdout
	}
	
	delayMs := 0
	if delayStr, ok := config["delay_ms"]; ok {
		if parsed, err := strconv.Atoi(delayStr); err == nil && parsed > 0 {
			delayMs = parsed
		}
	}
	
	return &StdoutDistributor{
		writer:  writer,
		delayMs: delayMs,
	}
}

// Distribute writes a work unit to stdout as a single JSON line
func (d *StdoutDistributor) Distribute(workUnit models.WorkUnit) error {
	// Apply delay if configured (for demos, testing, debugging)
	if d.delayMs > 0 {
		time.Sleep(time.Duration(d.delayMs) * time.Millisecond)
	}

	jsonData, err := json.Marshal(workUnit)
	if err != nil {
		return fmt.Errorf("failed to serialize work unit: %w", err)
	}

	// Write as NDJSON (newline-delimited JSON)
	_, err = fmt.Fprintf(d.writer, "%s\n", jsonData)
	if err != nil {
		return fmt.Errorf("failed to write to stdout: %w", err)
	}

	return nil
}

// Close implements the Distributor interface (no-op for stdout)
func (d *StdoutDistributor) Close() error {
	return nil
}

package distributor

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/dqme/job-coordinator/internal/models"
)

// StdoutDistributor writes work units as NDJSON to stdout
type StdoutDistributor struct {
	writer io.Writer
}

// NewStdoutDistributor creates a new stdout distributor
func NewStdoutDistributor() *StdoutDistributor {
	return &StdoutDistributor{
		writer: os.Stdout,
	}
}

// NewStdoutDistributorWithWriter creates a distributor with custom writer (for testing)
func NewStdoutDistributorWithWriter(writer io.Writer) *StdoutDistributor {
	return &StdoutDistributor{
		writer: writer,
	}
}

// Distribute writes a work unit to stdout as a single JSON line
func (d *StdoutDistributor) Distribute(workUnit models.WorkUnit) error {
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

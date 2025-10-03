package distributor

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/dqme/job-coordinator/internal/models"
)

// FileDistributor writes work units as NDJSON to a file
type FileDistributor struct {
	outputPath string
	file       *os.File
	mu         sync.Mutex
}

// NewFileDistributor creates a new file distributor
// If the file exists, it will be appended to; otherwise, it will be created
func NewFileDistributor(outputPath string) (*FileDistributor, error) {
	if outputPath == "" {
		return nil, fmt.Errorf("output path cannot be empty")
	}

	// Open file in append mode, create if doesn't exist
	file, err := os.OpenFile(outputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open output file: %w", err)
	}

	return &FileDistributor{
		outputPath: outputPath,
		file:       file,
	}, nil
}

// Distribute writes a work unit to the file as a single JSON line
func (d *FileDistributor) Distribute(workUnit models.WorkUnit) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.file == nil {
		return fmt.Errorf("distributor is closed")
	}

	jsonData, err := json.Marshal(workUnit)
	if err != nil {
		return fmt.Errorf("failed to serialize work unit: %w", err)
	}

	// Write as NDJSON (newline-delimited JSON)
	_, err = fmt.Fprintf(d.file, "%s\n", jsonData)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// Close flushes and closes the output file
func (d *FileDistributor) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.file == nil {
		return nil
	}

	err := d.file.Close()
	d.file = nil
	return err
}

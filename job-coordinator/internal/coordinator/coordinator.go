package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/distributor"
	"github.com/dqme/job-coordinator/internal/models"
	"github.com/dqme/job-coordinator/internal/processor"
)

// Coordinator orchestrates the processing of a job, managing file discovery,
// line counting, batch splitting, work unit creation, and distribution.
type Coordinator struct {
	mu           sync.RWMutex
	job          *models.Job
	cfg          *config.Config
	distributor  distributor.Distributor
	shutdownChan chan struct{} // Signal for TTL-based shutdown
	cancelCtx    context.Context
	cancelFunc   context.CancelFunc
}

// NewCoordinator creates a new Coordinator with the provided job and configuration.
func NewCoordinator(job *models.Job, cfg *config.Config, dist distributor.Distributor) *Coordinator {
	ctx, cancel := context.WithCancel(context.Background())
	return &Coordinator{
		job:          job,
		cfg:          cfg,
		distributor:  dist,
		shutdownChan: make(chan struct{}, 1),
		cancelCtx:    ctx,
		cancelFunc:   cancel,
	}
}

// GetJob returns the current job state (thread-safe).
func (c *Coordinator) GetJob() *models.Job {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.job
}

// GetConfig returns the current configuration (thread-safe).
func (c *Coordinator) GetConfig() *config.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.cfg
}

// UpdateConfig updates runtime-modifiable config fields.
func (c *Coordinator) UpdateConfig(newCfg *config.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Only allow updating runtime-modifiable fields
	// (concurrent_file_processors, distributor_config can change during execution)
	c.cfg.ConcurrentFileProcessors = newCfg.ConcurrentFileProcessors
	c.cfg.DistributorConfig = newCfg.DistributorConfig

	return nil
}

// ProcessJob orchestrates the entire job processing workflow.
func (c *Coordinator) ProcessJob() error {
	c.mu.Lock()
	if c.job.Status != "pending" && c.job.Status != "paused" {
		c.mu.Unlock()
		return fmt.Errorf("cannot start job in status: %s", c.job.Status)
	}
	c.job.Status = "running"
	c.mu.Unlock()

	// Discover files
	files, err := c.discoverFiles()
	if err != nil {
		c.updateJobStatus(models.JobStatusFailed)
		return fmt.Errorf("file discovery failed: %w", err)
	}

	// Discover measures
	measures, err := c.discoverMeasures()
	if err != nil {
		c.updateJobStatus(models.JobStatusFailed)
		return fmt.Errorf("measure discovery failed: %w", err)
	}

	// Process files concurrently
	if err := c.processFiles(files, measures); err != nil {
		c.updateJobStatus(models.JobStatusFailed)
		return fmt.Errorf("file processing failed: %w", err)
	}

	// Mark job as completed
	c.updateJobStatus(models.JobStatusCompleted)

	// Signal TTL shutdown
	select {
	case c.shutdownChan <- struct{}{}:
	default:
	}

	return nil
}

// discoverFiles discovers FHIR bundle files based on configuration.
func (c *Coordinator) discoverFiles() ([]string, error) {
	// Use DiscoverFiles which handles both manifest and auto-discovery
	return processor.DiscoverFiles(c.cfg.BasePath, c.cfg.ManifestPath)
}

// discoverMeasures discovers measure definitions based on configuration.
func (c *Coordinator) discoverMeasures() ([]string, error) {
	// If measures_to_run is explicitly provided, use it
	if len(c.cfg.MeasuresToRun) > 0 && !c.cfg.UseMeasuresPath {
		return c.cfg.MeasuresToRun, nil
	}

	// If use_measures_path=true, load from measures manifest
	if c.cfg.UseMeasuresPath && c.cfg.MeasuresManifestPath != nil && *c.cfg.MeasuresManifestPath != "" {
		measures, err := processor.DiscoverMeasures(*c.cfg.MeasuresManifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load measures manifest: %w", err)
		}
		return measures, nil
	}

	// If measures_path is provided, auto-discover measures
	if c.cfg.MeasuresPath != "" {
		return c.discoverMeasureFiles(c.cfg.MeasuresPath)
	}

	return nil, fmt.Errorf("no measure configuration provided")
}

// discoverMeasureFiles finds all JSON measure definition files in a directory.
func (c *Coordinator) discoverMeasureFiles(dir string) ([]string, error) {
	var measures []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".json") {
			measures = append(measures, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return measures, nil
}

// processFiles processes all discovered files concurrently using a worker pool.
func (c *Coordinator) processFiles(files []string, measures []string) error {
	// Create worker pool
	maxWorkers := c.cfg.ConcurrentFileProcessors
	fileChan := make(chan string, len(files))
	errChan := make(chan error, len(files))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				if err := c.processFile(file, measures); err != nil {
					errChan <- fmt.Errorf("failed to process %s: %w", file, err)
					return
				}
			}
		}()
	}

	// Enqueue files
	for _, file := range files {
		fileChan <- file
	}
	close(fileChan)

	// Wait for workers to finish
	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		return err
	}

	return nil
}

// processFile processes a single FHIR bundle file: counts lines, splits batches, creates work units.
func (c *Coordinator) processFile(file string, measures []string) error {
	// Count lines in the file
	lineCount, err := processor.CountLines(file)
	if err != nil {
		c.incrementErrorCount()
		return fmt.Errorf("failed to count lines: %w", err)
	}

	// Handle batch_size=0 (whole file mode)
	if c.cfg.BatchSize == 0 {
		return c.distributeWholeFile(file, measures)
	}

	// Split file into batches
	batches := processor.SplitIntoBatches(lineCount, c.cfg.BatchSize, c.cfg.RemainderThreshold)

	// Create and distribute work units for each batch
	for _, batch := range batches {
		if err := c.distributeBatch(file, batch, measures); err != nil {
			c.incrementErrorCount()
			return err
		}
	}

	return nil
}

// distributeWholeFile creates and distributes a single work unit for the entire file.
func (c *Coordinator) distributeWholeFile(file string, measures []string) error {
	workUnit := models.WorkUnit{
		ID:           generateWorkUnitID(),
		JobID:        c.job.ID,
		FilePath:     file,
		StartLine:    nil,
		EndLine:      nil,
		MeasuresPath: c.buildMeasuresPath(measures),
		Measures:     measures,
		BasePath:     c.cfg.BasePath,
		Status:       models.WorkUnitStatusPending,
	}

	if err := c.distributor.Distribute(workUnit); err != nil {
		c.incrementErrorCount()
		return fmt.Errorf("distribution failed: %w", err)
	}

	c.incrementTotalWorkUnits()
	return nil
}

// distributeBatch creates and distributes a work unit for a specific batch.
func (c *Coordinator) distributeBatch(file string, batch processor.BatchRange, measures []string) error {
	workUnit := models.WorkUnit{
		ID:           generateWorkUnitID(),
		JobID:        c.job.ID,
		FilePath:     file,
		StartLine:    batch.StartLine,
		EndLine:      batch.EndLine,
		TotalLines:   batch.TotalLines,
		MeasuresPath: c.buildMeasuresPath(measures),
		Measures:     measures,
		BasePath:     c.cfg.BasePath,
		Status:       models.WorkUnitStatusPending,
	}

	if err := c.distributor.Distribute(workUnit); err != nil {
		c.incrementErrorCount()
		return fmt.Errorf("distribution failed: %w", err)
	}

	c.incrementTotalWorkUnits()
	return nil
}

// buildMeasuresPath constructs the measures_path field for work units.
func (c *Coordinator) buildMeasuresPath(measures []string) *string {
	if c.cfg.UseMeasuresPath && c.cfg.MeasuresPath != "" {
		return &c.cfg.MeasuresPath
	}
	return nil
}

// Start transitions the job from pending or paused to running.
func (c *Coordinator) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.job.Status != "pending" && c.job.Status != "paused" {
		return fmt.Errorf("cannot start job in status: %s", c.job.Status)
	}

	c.job.Status = "running"
	log.Printf("Job started: %s", c.job.ID)

	// Start processing in background
	go func() {
		if err := c.ProcessJob(); err != nil {
			log.Printf("Job processing failed: %v", err)
		}
	}()

	return nil
}

// Pause transitions the job from running to paused.
func (c *Coordinator) Pause() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.job.Status != "running" {
		return fmt.Errorf("cannot pause job in status: %s", c.job.Status)
	}

	c.job.Status = "paused"
	log.Printf("Job paused: %s", c.job.ID)
	return nil
}

// Resume transitions the job from paused to running.
func (c *Coordinator) Resume() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.job.Status != "paused" {
		return fmt.Errorf("cannot resume job in status: %s", c.job.Status)
	}

	c.job.Status = "running"
	log.Printf("Job resumed: %s", c.job.ID)
	return nil
}

// Cancel transitions the job to cancelled status.
func (c *Coordinator) Cancel() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.job.Status == "completed" || c.job.Status == "cancelled" {
		return fmt.Errorf("cannot cancel job in status: %s", c.job.Status)
	}

	c.job.Status = "cancelled"
	c.cancelFunc() // Cancel context to stop processing
	log.Printf("Job cancelled: %s", c.job.ID)
	return nil
}

// ShutdownChan returns the shutdown channel for TTL-based shutdown.
func (c *Coordinator) ShutdownChan() <-chan struct{} {
	return c.shutdownChan
}

// Close gracefully closes the coordinator and its distributor.
func (c *Coordinator) Close() error {
	c.cancelFunc()
	return c.distributor.Close()
}

// Helper functions for updating job state

func (c *Coordinator) updateJobStatus(status models.JobStatus) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.job.Status = status
}

func (c *Coordinator) incrementTotalWorkUnits() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.job.TotalWorkUnits++
}

func (c *Coordinator) incrementErrorCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.job.ErrorCount++
}

// generateWorkUnitID generates a unique work unit ID
func generateWorkUnitID() string {
	// Simple ID generation - can be enhanced with UUID library
	return fmt.Sprintf("wu-%d", time.Now().UnixNano())
}

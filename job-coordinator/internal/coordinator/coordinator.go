package coordinator

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/distributor"
	"github.com/dqme/job-coordinator/internal/models"
	"github.com/dqme/job-coordinator/internal/processor"
	"github.com/google/uuid"
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

// joinPath joins a base path with a relative path, handling both local paths and cloud storage URLs
func joinPath(basePath, relativePath string) string {
	// Check if basePath is a cloud storage URL
	if strings.HasPrefix(basePath, "gs://") || strings.HasPrefix(basePath, "s3://") || strings.HasPrefix(basePath, "file://") {
		// For cloud storage URLs, use forward slashes and append properly
		base := strings.TrimSuffix(basePath, "/")
		rel := strings.TrimPrefix(relativePath, "/")
		// Replace backslashes with forward slashes for Windows paths in relative part
		rel = strings.ReplaceAll(rel, "\\", "/")
		return base + "/" + rel
	}
	// For local paths, use filepath.Join
	return filepath.Join(basePath, relativePath)
}

// stripBasePath removes the base path from a full path, returning just the relative portion.
// Works for both local paths and cloud storage URLs.
// Example: stripBasePath("gs://bucket/measures/cms-125.json", "gs://bucket/measures") -> "cms-125.json"
// Example: stripBasePath("C:/path/to/measures/cms-125.json", "./measures") -> "cms-125.json"
func stripBasePath(fullPath, basePath string) string {
	// For cloud URLs, use string prefix matching
	if strings.HasPrefix(basePath, "gs://") || strings.HasPrefix(basePath, "s3://") || strings.HasPrefix(basePath, "file://") {
		base := strings.TrimSuffix(basePath, "/")
		if strings.HasPrefix(fullPath, base+"/") {
			return strings.TrimPrefix(fullPath, base+"/")
		}
		// If no slash separator, try direct prefix match
		return strings.TrimPrefix(fullPath, base)
	}
	
	// For local paths, convert basePath to absolute first for proper comparison
	absBase, err := filepath.Abs(basePath)
	if err != nil {
		// Fallback: use basePath as-is
		absBase = basePath
	}
	
	// Use filepath.Rel to get the relative path
	relPath, err := filepath.Rel(absBase, fullPath)
	if err != nil {
		// Fallback: try direct string prefix stripping
		absBase = strings.TrimSuffix(absBase, string(filepath.Separator))
		return strings.TrimPrefix(strings.TrimPrefix(fullPath, absBase), string(filepath.Separator))
	}
	return relPath
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

// GetDistributor returns the distributor (thread-safe).
func (c *Coordinator) GetDistributor() distributor.Distributor {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.distributor
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
	if c.job.Status != "pending" && c.job.Status != "paused" && c.job.Status != "running" {
		c.mu.Unlock()
		return fmt.Errorf("cannot start job in status: %s", c.job.Status)
	}
	// Ensure status is running (Start() may have already set it)
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

	// Check if scale load mode is enabled
	if c.cfg.ScaleLoad != nil && *c.cfg.ScaleLoad > 0 {
		log.Printf("⚠️  WARNING: Scale load mode enabled - synthesizing %d work units for load testing", *c.cfg.ScaleLoad)
		log.Printf("⚠️  WARNING: This mode is for testing/benchmarking only and will cycle through file patterns")
		// Scale load mode: synthesize specified number of work units
		if err := c.processScaleLoad(files, measures, *c.cfg.ScaleLoad); err != nil {
			c.updateJobStatus(models.JobStatusFailed)
			return fmt.Errorf("scale load processing failed: %w", err)
		}
	} else {
		// Normal mode: process all files
		if err := c.processFiles(files, measures); err != nil {
			c.updateJobStatus(models.JobStatusFailed)
			return fmt.Errorf("file processing failed: %w", err)
		}
	}

	// Wait for distributor to complete before marking job as completed
	log.Printf("Waiting for distributor to complete...")
	if err := c.distributor.WaitForCompletion(); err != nil {
		log.Printf("Warning: distributor completion wait failed: %v", err)
	}
	
	// Verify distributor is complete
	status := c.distributor.GetStatus()
	if !status.IsComplete {
		log.Printf("Warning: distributor reports not complete after WaitForCompletion (pending: %d)", status.PendingCount)
	} else {
		log.Printf("Distributor complete: distributed %d work units", status.DistributedCount)
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
// Returns paths relative to the measures directory (just filenames).
func (c *Coordinator) discoverMeasures() ([]string, error) {
	// If measures_to_run is explicitly provided, use it (optional)
	if len(c.cfg.MeasuresToRun) > 0 {
		return c.cfg.MeasuresToRun, nil
	}

	// If measures manifest is provided, load from it
	if c.cfg.MeasuresManifestPath != nil && *c.cfg.MeasuresManifestPath != "" {
		measures, err := processor.DiscoverMeasures(*c.cfg.MeasuresManifestPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load measures manifest: %w", err)
		}
		return measures, nil
	}

	// Auto-discover measures from measures_path (default: ./measures)
	// processor.DiscoverMeasures handles both local and cloud storage uniformly
	if c.cfg.MeasuresPath != "" {
		absoluteMeasures, err := processor.DiscoverMeasures(c.cfg.MeasuresPath)
		if err != nil {
			return nil, err
		}
		
		// Convert absolute/full paths to relative paths (strip base directory)
		relativeMeasures := make([]string, 0, len(absoluteMeasures))
		for _, measure := range absoluteMeasures {
			// Strip the base path to get relative path
			// Works for both local paths and cloud URLs
			relPath := stripBasePath(measure, c.cfg.MeasuresPath)
			relativeMeasures = append(relativeMeasures, relPath)
		}
		
		return relativeMeasures, nil
	}

	return nil, fmt.Errorf("no measure configuration provided")
}

// processScaleLoad processes files in scale load mode where we want to generate
// a specific number of total work units for load testing. It cycles through the actual
// file's batch pattern repeatedly until reaching the target count.
func (c *Coordinator) processScaleLoad(files []string, measures []string, targetLineCount int) error {
	if len(files) == 0 {
		return fmt.Errorf("no files available for scale test")
	}

	// Use first file and count its lines
	firstFile := files[0]
	fullPath := joinPath(c.cfg.BasePath, firstFile)
	linesPerFile, err := processor.CountLines(fullPath)
	if err != nil {
		c.incrementErrorCount()
		return fmt.Errorf("failed to count lines in %s: %w", firstFile, err)
	}

	log.Printf("Scale test mode: target=%d lines, file=%s has %d lines, batch_size=%d", 
		targetLineCount, firstFile, linesPerFile, c.cfg.BatchSize)

	// Apply batch splitting formula to the actual file to get the batch pattern
	fileBatches := processor.SplitIntoBatches(linesPerFile, c.cfg.BatchSize, c.cfg.RemainderThreshold)
	
	// Cycle through the file's batch pattern until we reach target line count
	linesGenerated := 0
	batchIndex := 0
	workUnitCount := 0
	
	for linesGenerated < targetLineCount {
		// Get the next batch from the pattern (cycling)
		batch := fileBatches[batchIndex%len(fileBatches)]
		
		// Calculate how many lines this batch should have
		batchLines := *batch.TotalLines
		remainingLines := targetLineCount - linesGenerated
		
		// If this batch would exceed target, truncate it
		if batchLines > remainingLines {
			batchLines = remainingLines
			// Create truncated batch
			start := *batch.StartLine
			end := start + batchLines - 1
			truncatedBatch := processor.BatchRange{
				StartLine:  &start,
				EndLine:    &end,
				TotalLines: &batchLines,
			}
			if err := c.distributeBatch(firstFile, truncatedBatch, measures); err != nil {
				c.incrementErrorCount()
				return err
			}
		} else {
			// Use full batch
			if err := c.distributeBatch(firstFile, batch, measures); err != nil {
				c.incrementErrorCount()
				return err
			}
		}
		
		linesGenerated += batchLines
		batchIndex++
		workUnitCount++
	}

	log.Printf("Scale test: created %d work units from %d target lines (pattern from %d-line file)", 
		workUnitCount, targetLineCount, linesPerFile)

	return nil
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
	// Handle batch_size=0 (whole file mode) - skip line counting
	if c.cfg.BatchSize == 0 {
		return c.distributeWholeFile(file, measures)
	}

	// Construct full path from base path + relative file path
	fullPath := joinPath(c.cfg.BasePath, file)
	
	// Count lines in the file (only needed for batching mode)
	lineCount, err := processor.CountLines(fullPath)
	if err != nil {
		c.incrementErrorCount()
		return fmt.Errorf("failed to count lines: %w", err)
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
	c.incrementProcessedCount()
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
	c.incrementProcessedCount()
	return nil
}

// buildMeasuresPath constructs the measures_path field for work units.
// It returns the MeasuresPath as a relative path that work units can use.
func (c *Coordinator) buildMeasuresPath(measures []string) *string {
	if c.cfg.MeasuresPath != "" {
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

func (c *Coordinator) incrementProcessedCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.job.ProcessedCount++
}

func (c *Coordinator) incrementErrorCount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.job.ErrorCount++
}

// generateWorkUnitID generates a unique work unit ID using UUID v7
func generateWorkUnitID() string {
	return uuid.Must(uuid.NewV7()).String()
}

package coordinator

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"sync"

	"github.com/dqme/job-driver/internal/config"
	"github.com/dqme/job-driver/internal/distributor"
	"github.com/dqme/job-driver/internal/models"
	"github.com/dqme/job-driver/internal/processor"
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
		
		// Create file spec for this batch
		var fileSpec models.FileSpec
		if batchLines > remainingLines {
			// Truncate batch to fit remaining lines
			start := *batch.StartLine
			end := start + remainingLines - 1
			fileSpec = models.FileSpec{
				Path:      firstFile,
				StartLine: &start,
				EndLine:   &end,
			}
			linesGenerated += remainingLines
		} else {
			// Use full batch
			fileSpec = models.FileSpec{
				Path:      firstFile,
				StartLine: batch.StartLine,
				EndLine:   batch.EndLine,
			}
			linesGenerated += batchLines
		}
		
		if err := c.distributeWorkUnit([]models.FileSpec{fileSpec}, measures); err != nil {
			c.incrementErrorCount()
			return err
		}
		
		batchIndex++
		workUnitCount++
	}

	log.Printf("Scale test: created %d work units from %d target lines (pattern from %d-line file)", 
		workUnitCount, targetLineCount, linesPerFile)

	return nil
}

// processFiles processes all discovered files using unified batching algorithm.
// The algorithm handles all scenarios:
// - batch_size=0: whole files, one file per work unit
// - batch_size>0 + multifile-batches=true: accumulate files into batches
// - batch_size>0 + multifile-batches=false: split large files into batches
func (c *Coordinator) processFiles(files []string, measures []string) error {
	return c.processFilesWithUnifiedBatching(files, measures)
}

// FileWithLineCount represents a file and its line count
type FileWithLineCount struct {
	Path      string
	LineCount int
}

// processFilesWithUnifiedBatching implements the unified batching algorithm that handles:
// 1. batch_size=0: one whole file per work unit
// 2. batch_size>0 + multifile=true: accumulate multiple whole files until batch_size
// 3. batch_size>0 + multifile=false: split large files into batched segments
func (c *Coordinator) processFilesWithUnifiedBatching(files []string, measures []string) error {
	// Handle batch_size=0 (whole file mode) - no line counting needed
	if c.cfg.BatchSize == 0 {
		for _, file := range files {
			fileSpec := models.FileSpec{Path: file}
			if err := c.distributeWorkUnit([]models.FileSpec{fileSpec}, measures); err != nil {
				c.incrementErrorCount()
				return err
			}
			c.incrementProcessedCount()
		}
		return nil
	}

	// batch_size > 0: need to count lines in all files
	type countResult struct {
		file      string
		lineCount int
		err       error
	}
	
	resultChan := make(chan countResult, len(files))
	semaphore := make(chan struct{}, c.cfg.ConcurrentFileProcessors)
	
	for _, file := range files {
		file := file // capture loop variable
		go func() {
			semaphore <- struct{}{} // acquire
			defer func() { <-semaphore }() // release
			
			fullPath := joinPath(c.cfg.BasePath, file)
			lineCount, err := processor.CountLines(fullPath)
			resultChan <- countResult{file: file, lineCount: lineCount, err: err}
		}()
	}
	
	// Collect line count results (order doesn't matter for batching efficiency)
	fileCounts := make([]FileWithLineCount, 0, len(files))
	for i := 0; i < len(files); i++ {
		result := <-resultChan
		if result.err != nil {
			c.incrementErrorCount()
			return fmt.Errorf("failed to count lines in %s: %w", result.file, result.err)
		}
		fileCounts = append(fileCounts, FileWithLineCount{
			Path:      result.file,
			LineCount: result.lineCount,
		})
	}
	close(resultChan)
	
	// Now process files based on multifile-batches setting
	if c.cfg.MultifileBatches {
		// Multi-file batching: accumulate whole files until batch_size reached
		return c.processWithMultifileBatching(fileCounts, measures)
	} else {
		// Single-file batching: split individual files that exceed batch_size
		return c.processWithSingleFileBatching(fileCounts, measures)
	}
}

// processWithMultifileBatching accumulates multiple whole files into work units,
// and can also include partial segments of large files to fill batches efficiently.
// Algorithm aims for batches of ~batch_size lines with variance up to remainder_threshold,
// minimizing file splits.
func (c *Coordinator) processWithMultifileBatching(fileCounts []FileWithLineCount, measures []string) error {
	currentBatch := []models.FileSpec{}
	currentLines := 0
	maxBatchSize := int(float64(c.cfg.BatchSize) * (1.0 + c.cfg.RemainderThreshold))
	
	for _, fc := range fileCounts {
		remainingLines := fc.LineCount
		currentSegmentStart := 0
		
		for remainingLines > 0 {
			// Can we add this whole file/segment to current batch within threshold?
			if currentLines+remainingLines <= maxBatchSize {
				// Yes - add whole file/segment to current batch
				fileSpec := models.FileSpec{Path: fc.Path}
				if currentSegmentStart > 0 || remainingLines < fc.LineCount {
					// This is a segment (not the original whole file)
					start := currentSegmentStart
					end := currentSegmentStart + remainingLines - 1
					fileSpec.StartLine = &start
					fileSpec.EndLine = &end
				}
				currentBatch = append(currentBatch, fileSpec)
				currentLines += remainingLines
				remainingLines = 0
				
				// Emit batch if we've reached target size
				if currentLines >= c.cfg.BatchSize {
					if err := c.distributeWorkUnit(currentBatch, measures); err != nil {
						c.incrementErrorCount()
						return err
					}
					c.incrementProcessedCount()
					currentBatch = []models.FileSpec{}
					currentLines = 0
				}
			} else {
				// File/segment doesn't fit - emit current batch and handle the file
				if len(currentBatch) > 0 {
					if err := c.distributeWorkUnit(currentBatch, measures); err != nil {
						c.incrementErrorCount()
						return err
					}
					c.incrementProcessedCount()
					currentBatch = []models.FileSpec{}
					currentLines = 0
				}
				
				// If remaining file/segment is smaller than batch_size, send as whole file
				if remainingLines <= c.cfg.BatchSize {
					fileSpec := models.FileSpec{Path: fc.Path}
					if currentSegmentStart > 0 {
						// This is a segment
						start := currentSegmentStart
						end := currentSegmentStart + remainingLines - 1
						fileSpec.StartLine = &start
						fileSpec.EndLine = &end
					}
					if err := c.distributeWorkUnit([]models.FileSpec{fileSpec}, measures); err != nil {
						c.incrementErrorCount()
						return err
					}
					c.incrementProcessedCount()
					remainingLines = 0
				} else {
					// Split the large file into optimal segments using remainder threshold
					batches := processor.SplitIntoBatches(remainingLines, c.cfg.BatchSize, c.cfg.RemainderThreshold)
					
					for _, batch := range batches {
						start := currentSegmentStart + *batch.StartLine
						end := currentSegmentStart + *batch.EndLine
						fileSpec := models.FileSpec{
							Path:      fc.Path,
							StartLine: &start,
							EndLine:   &end,
						}
						
						// Each large file segment becomes its own work unit
						if err := c.distributeWorkUnit([]models.FileSpec{fileSpec}, measures); err != nil {
							c.incrementErrorCount()
							return err
						}
						c.incrementProcessedCount()
					}
					
					remainingLines = 0
				}
			}
		}
	}
	
	// Emit final batch if not empty
	if len(currentBatch) > 0 {
		if err := c.distributeWorkUnit(currentBatch, measures); err != nil {
			c.incrementErrorCount()
			return err
		}
		c.incrementProcessedCount()
	}
	
	return nil
}

// processWithSingleFileBatching splits individual files into batches
func (c *Coordinator) processWithSingleFileBatching(fileCounts []FileWithLineCount, measures []string) error {
	for _, fc := range fileCounts {
		// If file is smaller than batch size, send whole file without line ranges
		if fc.LineCount <= c.cfg.BatchSize {
			fileSpec := models.FileSpec{Path: fc.Path}
			if err := c.distributeWorkUnit([]models.FileSpec{fileSpec}, measures); err != nil {
				c.incrementErrorCount()
				return err
			}
			c.incrementProcessedCount()
			continue
		}
		
		// Split large file into batches
		batches := processor.SplitIntoBatches(fc.LineCount, c.cfg.BatchSize, c.cfg.RemainderThreshold)
		
		for _, batch := range batches {
			fileSpec := models.FileSpec{
				Path:      fc.Path,
				StartLine: batch.StartLine,
				EndLine:   batch.EndLine,
			}
			if err := c.distributeWorkUnit([]models.FileSpec{fileSpec}, measures); err != nil {
				c.incrementErrorCount()
				return err
			}
		}
		c.incrementProcessedCount()
	}
	
	return nil
}

// distributeWorkUnit creates and distributes a work unit with the given file specs
func (c *Coordinator) distributeWorkUnit(files []models.FileSpec, measures []string) error {
	workUnit := models.WorkUnit{
		ID:           generateWorkUnitID(),
		JobID:        c.job.ID,
		Files:        files,
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

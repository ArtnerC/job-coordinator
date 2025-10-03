package config

import (
	"errors"
	"path/filepath"
	"time"
)

// Config represents the job coordinator configuration
type Config struct {
	JobID                    string
	AutoStart                bool
	BatchSize                int
	RemainderThreshold       float64
	BasePath                 string
	ManifestPath             *string
	MeasuresPath             string
	MeasuresToRun            []string
	MeasuresManifestPath     *string
	UseMeasuresPath          bool
	DistributorType          string
	DistributorConfig        map[string]string
	CompletionTTL            time.Duration
	ConcurrentFileProcessors int
	ScaleTestCount           *int
	APIPort                  int
}

// ValidateConfig validates the configuration and returns an error if invalid
func ValidateConfig(cfg *Config) error {
	// Validate batch_size
	if cfg.BatchSize < 0 {
		return errors.New("batch_size must be >= 0")
	}

	// Validate remainder_threshold
	if cfg.RemainderThreshold < 0.0 || cfg.RemainderThreshold > 1.0 {
		return errors.New("remainder_threshold must be between 0.0 and 1.0")
	}

	// Validate base_path is absolute
	if !filepath.IsAbs(cfg.BasePath) {
		return errors.New("base_path must be absolute")
	}

	// Validate distributor_type
	validDistributors := map[string]bool{
		"pubsub": true,
		"stdout": true,
		"file":   true,
	}
	if !validDistributors[cfg.DistributorType] {
		return errors.New("distributor_type must be one of: pubsub, stdout, file")
	}

	// Validate concurrent_file_processors
	if cfg.ConcurrentFileProcessors <= 0 {
		return errors.New("concurrent_file_processors must be > 0")
	}

	// Validate measures configuration
	if !cfg.UseMeasuresPath && len(cfg.MeasuresToRun) == 0 {
		return errors.New("measures_to_run must be provided when use_measures_path is false")
	}

	return nil
}

// SetDefaults sets default values for optional configuration fields
func SetDefaults(cfg *Config) {
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 500
	}
	if cfg.RemainderThreshold == 0 {
		cfg.RemainderThreshold = 0.2
	}
	if cfg.CompletionTTL == 0 {
		cfg.CompletionTTL = 10 * time.Minute
	}
	if cfg.ConcurrentFileProcessors == 0 {
		cfg.ConcurrentFileProcessors = 10
	}
	if cfg.APIPort == 0 {
		cfg.APIPort = 8080
	}
	if cfg.DistributorType == "" {
		cfg.DistributorType = "stdout"
	}
	if cfg.DistributorConfig == nil {
		cfg.DistributorConfig = make(map[string]string)
	}
}

// IsRuntimeModifiable checks if a configuration field can be modified at runtime
// based on the current job status
func IsRuntimeModifiable(field string, status string) bool {
	// Terminal states: no modifications allowed
	terminalStates := map[string]bool{
		"completed": true,
		"failed":    true,
		"cancelled": true,
	}
	if terminalStates[status] {
		// Only completion_ttl can be modified in terminal states
		return field == "completion_ttl"
	}

	// Runtime-modifiable in running/paused states
	runtimeModifiable := map[string]bool{
		"batch_size":                  true,
		"remainder_threshold":         true,
		"concurrent_file_processors":  true,
		"completion_ttl":              true,
	}

	// Pre-start only fields
	preStartOnly := map[string]bool{
		"distributor_type":          true,
		"distributor_config":        true,
		"use_measures_path":         true,
		"measures_to_run":           true,
	}

	// If status is pending, everything is modifiable
	if status == "pending" {
		return runtimeModifiable[field] || preStartOnly[field]
	}

	// For running/paused, only runtime-modifiable fields allowed
	return runtimeModifiable[field]
}

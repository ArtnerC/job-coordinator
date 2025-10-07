package config

import (
	"errors"
	"time"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
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
	DistributorType          string
	DistributorConfig        map[string]string
	CompletionTTL            time.Duration
	ConcurrentFileProcessors int
	ScaleLoad                *int
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

	return nil
}

// SetDefaults sets default values for optional configuration fields
func SetDefaults(cfg *Config) {
	// BatchSize: 0 is valid (whole file mode) and is the default
	// No default needed - 0 is intentional
	
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
	if cfg.MeasuresPath == "" {
		cfg.MeasuresPath = "./measures"
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
		"measures_to_run":           true,
	}

	// If status is pending, everything is modifiable
	if status == "pending" {
		return runtimeModifiable[field] || preStartOnly[field]
	}

	// For running/paused, only runtime-modifiable fields allowed
	return runtimeModifiable[field]
}

var flagsInitialized bool

// LoadConfig loads configuration from CLI flags and environment variables
// Precedence: CLI flags > env vars > defaults
func LoadConfig() (*Config, error) {
	v := viper.New()

	// Define CLI flags using pflag (only once)
	if !flagsInitialized {
		pflag.String("job-id", "", "Unique job identifier (auto-generated if not provided)")
		pflag.Bool("auto-start", false, "Automatically start job on coordinator launch")
		pflag.Int("batch-size", 0, "Number of lines per work unit (0 for whole file mode - default)")
		pflag.Float64("remainder-threshold", 0.2, "Threshold for appending remainder to last batch (0.0-1.0)")
		pflag.String("base-path", "", "Base directory containing patient bundle files (required)")
		pflag.String("manifest-path", "", "Path to file manifest (optional, auto-discovers if not provided)")
		pflag.String("measures-path", "./measures", "Directory containing measure definition files")
		pflag.StringSlice("measures-to-run", []string{}, "Specific measure paths to include (optional)")
		pflag.String("measures-manifest-path", "", "Path to measures manifest file")
		pflag.String("distributor-type", "stdout", "Distributor type: pubsub, stdout, or file")
		pflag.StringToString("distributor-config", map[string]string{}, "Distributor-specific configuration (key=value pairs). For stdout: delay_ms=<milliseconds> to slow output for demos/testing")
		pflag.Duration("completion-ttl", 10*time.Minute, "Time to keep running after job completion")
		pflag.Int("concurrent-file-processors", 10, "Number of files to process concurrently")
		pflag.Int("scale-load", 0, "Synthesize load by generating N work units from file patterns (0 to disable)")
		pflag.Int("api-port", 8080, "Port for REST API server")
		flagsInitialized = true
	}

	pflag.Parse()
	v.BindPFlags(pflag.CommandLine)

	// Bind environment variables with prefix
	v.SetEnvPrefix("JOB_COORDINATOR")
	v.AutomaticEnv()

	// Build config struct from viper
	cfg := &Config{
		JobID:                    v.GetString("job-id"),
		AutoStart:                v.GetBool("auto-start"),
		BatchSize:                v.GetInt("batch-size"),
		RemainderThreshold:       v.GetFloat64("remainder-threshold"),
		BasePath:                 v.GetString("base-path"),
		MeasuresPath:             v.GetString("measures-path"),
		MeasuresToRun:            v.GetStringSlice("measures-to-run"),
		DistributorType:          v.GetString("distributor-type"),
		DistributorConfig:        v.GetStringMapString("distributor-config"),
		CompletionTTL:            v.GetDuration("completion-ttl"),
		ConcurrentFileProcessors: v.GetInt("concurrent-file-processors"),
		APIPort:                  v.GetInt("api-port"),
	}

	// Handle optional string pointers
	if manifestPath := v.GetString("manifest-path"); manifestPath != "" {
		cfg.ManifestPath = &manifestPath
	}
	if measuresManifestPath := v.GetString("measures-manifest-path"); measuresManifestPath != "" {
		cfg.MeasuresManifestPath = &measuresManifestPath
	}
	if scaleLoad := v.GetInt("scale-load"); scaleLoad > 0 {
		cfg.ScaleLoad = &scaleLoad
	}

	// Set defaults for unset values
	SetDefaults(cfg)

	// Validate configuration
	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

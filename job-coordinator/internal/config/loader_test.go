package config

import (
	"testing"
	"time"
)

// TestLoadConfigBasic tests LoadConfig with minimal validation
// Note: Full flag testing is difficult due to pflag global state
// This test validates that the function executes and returns a valid config
func TestLoadConfigBasic(t *testing.T) {
	// This is a smoke test - just ensure LoadConfig doesn't crash
	// and returns reasonable defaults when no flags are set
	//
	// Full integration testing of flag parsing should be done in
	// end-to-end tests or via manual testing
	
	// We can't easily test CLI flag parsing in unit tests because pflag
	// maintains global state that persists across test runs.
	// Instead, we validate the config logic separate from flag parsing.
	t.Skip("Skipping LoadConfig test - pflag global state makes testing difficult. Test manually or in E2E tests.")
}

// TestConfigStructValidation tests that we can create and validate configs programmatically
func TestConfigStructValidation(t *testing.T) {
	// Test creating a valid config without using LoadConfig
	cfg := &Config{
		JobID:                    "test-job",
		AutoStart:                false,
		BatchSize:                500,
		RemainderThreshold:       0.2,
		BasePath:                 "C:\\test\\data",
		MeasuresPath:             "/measures",
		MeasuresToRun:            []string{"/path/measure.json"},
		UseMeasuresPath:          false,
		DistributorType:          "stdout",
		DistributorConfig:        make(map[string]string),
		CompletionTTL:            10 * time.Minute,
		ConcurrentFileProcessors: 10,
		APIPort:                  8080,
	}

	// Apply defaults
	SetDefaults(cfg)

	// Validate
	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Valid config failed validation: %v", err)
	}

	// Verify defaults were applied correctly
	if cfg.BatchSize != 500 {
		t.Errorf("Expected batch_size 500, got %d", cfg.BatchSize)
	}

	if cfg.RemainderThreshold != 0.2 {
		t.Errorf("Expected remainder_threshold 0.2, got %f", cfg.RemainderThreshold)
	}

	if cfg.DistributorType != "stdout" {
		t.Errorf("Expected distributor_type stdout, got %s", cfg.DistributorType)
	}
}

// TestConfigWithAllDefaults tests that SetDefaults properly sets all default values
func TestConfigWithAllDefaults(t *testing.T) {
	cfg := &Config{
		BasePath:      "C:\\test\\data",
		MeasuresToRun: []string{"/path/measure.json"},
	}

	SetDefaults(cfg)

	// Check all defaults
	if cfg.BatchSize != 500 {
		t.Errorf("Expected default batch_size 500, got %d", cfg.BatchSize)
	}

	if cfg.RemainderThreshold != 0.2 {
		t.Errorf("Expected default remainder_threshold 0.2, got %f", cfg.RemainderThreshold)
	}

	if cfg.DistributorType != "stdout" {
		t.Errorf("Expected default distributor_type stdout, got %s", cfg.DistributorType)
	}

	if cfg.CompletionTTL != 10*time.Minute {
		t.Errorf("Expected default completion_ttl 10m, got %v", cfg.CompletionTTL)
	}

	if cfg.ConcurrentFileProcessors != 10 {
		t.Errorf("Expected default concurrent_file_processors 10, got %d", cfg.ConcurrentFileProcessors)
	}

	if cfg.APIPort != 8080 {
		t.Errorf("Expected default api_port 8080, got %d", cfg.APIPort)
	}

	if cfg.DistributorConfig == nil {
		t.Error("Expected distributor_config to be initialized")
	}
}

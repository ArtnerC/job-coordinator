package config

import (
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config with all fields",
			config: Config{
				JobID:                    "test-job-001",
				AutoStart:                true,
				BatchSize:                500,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "./measures",
				MeasuresToRun:            []string{"./measures/cms-125.json"},
				DistributorType:          "stdout",
				DistributorConfig:        map[string]string{},
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
				APIPort:                  8080,
			},
			wantErr: false,
		},
		{
			name: "batch_size negative",
			config: Config{
				BatchSize:                -1,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "C:\\data\\measures",
				MeasuresToRun:            []string{"C:\\data\\measures\\cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: true,
			errMsg:  "batch_size must be >= 0",
		},
		{
			name: "remainder_threshold too low",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       -0.1,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "C:\\data\\measures",
				MeasuresToRun:            []string{"C:\\data\\measures\\cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: true,
			errMsg:  "remainder_threshold must be between 0.0 and 1.0",
		},
		{
			name: "remainder_threshold too high",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       1.5,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "C:\\data\\measures",
				MeasuresToRun:            []string{"C:\\data\\measures\\cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: true,
			errMsg:  "remainder_threshold must be between 0.0 and 1.0",
		},
		{
			name: "base_path relative (now valid)",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       0.2,
				BasePath:                 "relative/path",
				MeasuresPath:             "./measures",
				MeasuresToRun:            []string{"./measures/cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: false,
		},
		{
			name: "distributor_type invalid",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "C:\\data\\measures",
				MeasuresToRun:            []string{"C:\\data\\measures\\cms-125.json"},
				DistributorType:          "invalid",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: true,
			errMsg:  "distributor_type must be one of: pubsub, stdout, file",
		},
		{
			name: "concurrent_file_processors zero",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "C:\\data\\measures",
				MeasuresToRun:            []string{"C:\\data\\measures\\cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 0,
			},
			wantErr: true,
			errMsg:  "concurrent_file_processors must be > 0",
		},
		{
			name: "concurrent_file_processors negative",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "C:\\data\\measures",
				MeasuresToRun:            []string{"C:\\data\\measures\\cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: -1,
			},
			wantErr: true,
			errMsg:  "concurrent_file_processors must be > 0",
		},
		{
			name: "batch_size=0 (whole file mode)",
			config: Config{
				BatchSize:                0,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "./measures",
				MeasuresToRun:            []string{"./measures/cms-125.json"},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: false,
		},
		{
			name: "measures_path always included (valid)",
			config: Config{
				BatchSize:                500,
				RemainderThreshold:       0.2,
				BasePath:                 "C:\\data\\bundles",
				MeasuresPath:             "./measures",
				MeasuresToRun:            []string{},
				DistributorType:          "stdout",
				CompletionTTL:            10 * time.Minute,
				ConcurrentFileProcessors: 10,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil && tt.errMsg != "" {
				if err.Error() != tt.errMsg {
					t.Errorf("ValidateConfig() error message = %q, want %q", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

package distributor

import (
	"context"
	"testing"

	"github.com/dqme/job-coordinator/internal/config"
	"github.com/dqme/job-coordinator/internal/models"
)

func TestNewDistributor(t *testing.T) {
	tests := []struct {
		name            string
		distributorType string
		distributorConfig map[string]string
		wantType        string // Type name for verification
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "create stdout distributor",
			distributorType: "stdout",
			distributorConfig: map[string]string{},
			wantType:        "*distributor.StdoutDistributor",
			wantErr:         false,
		},
		{
			name:            "create file distributor with valid config",
			distributorType: "file",
			distributorConfig: map[string]string{
				"output_path": "/tmp/work-units",
			},
			wantType: "*distributor.FileDistributor",
			wantErr:  false,
		},
		{
			name:            "create file distributor without output_path",
			distributorType: "file",
			distributorConfig: map[string]string{},
			wantType:        "",
			wantErr:         true,
			errMsg:          "output_path",
		},
		{
			name:            "create pubsub distributor without project_id",
			distributorType: "pubsub",
			distributorConfig: map[string]string{
				"topic_name": "work-queue",
			},
			wantType: "",
			wantErr:  true,
			errMsg:   "project_id",
		},
		{
			name:            "create pubsub distributor without topic_name",
			distributorType: "pubsub",
			distributorConfig: map[string]string{
				"project_id": "test-project",
			},
			wantType: "",
			wantErr:  true,
			errMsg:   "topic_name",
		},
		{
			name:            "invalid distributor type",
			distributorType: "invalid",
			distributorConfig: map[string]string{},
			wantType:        "",
			wantErr:         true,
			errMsg:          "unknown distributor type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				DistributorType:   tt.distributorType,
				DistributorConfig: tt.distributorConfig,
			}
			
			ctx := context.Background()
			dist, err := NewDistributor(ctx, cfg)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewDistributor() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !contains(err.Error(), tt.errMsg) {
					t.Errorf("NewDistributor() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Fatalf("NewDistributor() unexpected error = %v", err)
			}

			if dist == nil {
				t.Fatal("NewDistributor() returned nil distributor")
			}

			// Verify type (basic check that we got the right implementation)
			typeName := getTypeName(dist)
			if typeName != tt.wantType {
				t.Errorf("NewDistributor() type = %q, want %q", typeName, tt.wantType)
			}
		})
	}
}

func TestStdoutDistributor_Distribute(t *testing.T) {
	dist := NewStdoutDistributor()

	wu := models.WorkUnit{
		ID:       "test-wu-001",
		JobID:    "test-job-001",
		FilePath: "data/patients.ndjson",
	}

	// Stdout distributor should always succeed
	err := dist.Distribute(wu)
	if err != nil {
		t.Errorf("StdoutDistributor.Distribute() unexpected error = %v", err)
	}
}

func TestStdoutDistributor_Close(t *testing.T) {
	dist := NewStdoutDistributor()

	// Close should always succeed for stdout
	err := dist.Close()
	if err != nil {
		t.Errorf("StdoutDistributor.Close() unexpected error = %v", err)
	}
}

// Helper functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || 
		(len(s) > 0 && len(substr) > 0 && containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func getTypeName(v interface{}) string {
	if v == nil {
		return "<nil>"
	}
	// Simple type name extraction
	switch v.(type) {
	case *StdoutDistributor:
		return "*distributor.StdoutDistributor"
	case *FileDistributor:
		return "*distributor.FileDistributor"
	case *PubSubDistributor:
		return "*distributor.PubSubDistributor"
	default:
		return "unknown"
	}
}

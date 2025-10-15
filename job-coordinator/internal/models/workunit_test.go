package models

import (
"testing"
"time"
)

func TestWorkUnitValidation(t *testing.T) {
tests := []struct {
name    string
wu      WorkUnit
wantErr bool
errMsg  string
}{
{
name: "valid single whole file",
wu: WorkUnit{
ID:        "wu-001",
JobID:     "job-001",
Files:     []FileSpec{{Path: "bundle-001.ndjson"}},
Measures:  []string{"/data/measures/cms-125.json"},
BasePath:  "/data/bundles",
CreatedAt: time.Now(),
},
wantErr: false,
},
}

for _, tt := range tests {
t.Run(tt.name, func(t *testing.T) {
err := ValidateWorkUnit(&tt.wu)
if (err != nil) != tt.wantErr {
t.Errorf("ValidateWorkUnit() error = %v, wantErr %v", err, tt.wantErr)
}
})
}
}

func intPtr(n int) *int { return &n }
func stringPtr(s string) *string { return &s }

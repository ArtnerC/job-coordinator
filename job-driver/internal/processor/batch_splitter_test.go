package processor

import (
	"fmt"
	"testing"
)

func TestSplitIntoBatches(t *testing.T) {
	tests := []struct {
		name       string
		totalLines int
		batchSize  int
		threshold  float64
		want       []BatchRange
	}{
		{
			name:       "1050 lines, batch_size=500, threshold=0.2 -> 2 batches (500, 550)",
			totalLines: 1050,
			batchSize:  500,
			threshold:  0.2,
			want: []BatchRange{
				{intPtr(0), intPtr(499), intPtr(500)},
				{intPtr(500), intPtr(1049), intPtr(550)}, // Remainder 50 (10%) appended
			},
		},
		{
			name:       "1600 lines, batch_size=500, threshold=0.2 -> 4 batches (500, 500, 500, 100)",
			totalLines: 1600,
			batchSize:  500,
			threshold:  0.2,
			want: []BatchRange{
				{intPtr(0), intPtr(499), intPtr(500)},
				{intPtr(500), intPtr(999), intPtr(500)},
				{intPtr(1000), intPtr(1499), intPtr(500)},
				{intPtr(1500), intPtr(1599), intPtr(100)}, // Remainder 100 (20%) is new batch
			},
		},
		{
			name:       "480 lines, batch_size=500 -> 1 batch (480)",
			totalLines: 480,
			batchSize:  500,
			threshold:  0.2,
			want: []BatchRange{
				{intPtr(0), intPtr(479), intPtr(480)},
			},
		},
		{
			name:       "batch_size=0 -> 1 batch with nil line fields",
			totalLines: 1000,
			batchSize:  0,
			threshold:  0.2,
			want: []BatchRange{
				{nil, nil, nil}, // Whole file mode
			},
		},
		{
			name:       "exact multiple of batch size",
			totalLines: 1000,
			batchSize:  500,
			threshold:  0.2,
			want: []BatchRange{
				{intPtr(0), intPtr(499), intPtr(500)},
				{intPtr(500), intPtr(999), intPtr(500)},
			},
		},
		{
			name:       "small remainder below threshold - append",
			totalLines: 550,
			batchSize:  500,
			threshold:  0.2,
			want: []BatchRange{
				{intPtr(0), intPtr(549), intPtr(550)}, // Remainder 50 (10%) appended
			},
		},
		{
			name:       "large remainder at threshold - new batch",
			totalLines: 600,
			batchSize:  500,
			threshold:  0.2,
			want: []BatchRange{
				{intPtr(0), intPtr(499), intPtr(500)},
				{intPtr(500), intPtr(599), intPtr(100)}, // Remainder 100 (20%) is new batch
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SplitIntoBatches(tt.totalLines, tt.batchSize, tt.threshold)
			
			if len(got) != len(tt.want) {
				t.Fatalf("SplitIntoBatches() returned %d batches, want %d", len(got), len(tt.want))
			}

			for i := range got {
				if !batchRangeEqual(got[i], tt.want[i]) {
					t.Errorf("Batch %d: got %v, want %v", i, formatBatchRange(got[i]), formatBatchRange(tt.want[i]))
				}
			}
		})
	}
}

// Helper functions for testing
func intPtr(n int) *int {
	return &n
}

func batchRangeEqual(a, b BatchRange) bool {
	return intPtrEqual(a.StartLine, b.StartLine) &&
		intPtrEqual(a.EndLine, b.EndLine) &&
		intPtrEqual(a.TotalLines, b.TotalLines)
}

func intPtrEqual(a, b *int) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func formatBatchRange(br BatchRange) string {
	if br.StartLine == nil {
		return "{nil, nil, nil}"
	}
	return fmt.Sprintf("{%d, %d, %d}", *br.StartLine, *br.EndLine, *br.TotalLines)
}

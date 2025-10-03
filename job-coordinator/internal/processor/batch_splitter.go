package processor

// BatchRange represents a range of lines in a file
type BatchRange struct {
	StartLine  *int
	EndLine    *int
	TotalLines *int
}

// SplitIntoBatches splits a file into batches based on batch size and remainder threshold.
// If batchSize is 0, returns a single batch with nil line fields (whole file mode).
// Otherwise, splits into batches of batchSize lines, with remainder handling based on threshold.
func SplitIntoBatches(totalLines, batchSize int, threshold float64) []BatchRange {
	// Whole file mode: batch_size=0
	if batchSize == 0 {
		return []BatchRange{{nil, nil, nil}}
	}

	// Calculate number of full batches and remainder
	fullBatches := totalLines / batchSize
	remainder := totalLines % batchSize

	// If no remainder, return exact batches
	if remainder == 0 {
		batches := make([]BatchRange, fullBatches)
		for i := 0; i < fullBatches; i++ {
			start := i * batchSize
			end := start + batchSize - 1
			total := batchSize
			batches[i] = BatchRange{
				StartLine:  &start,
				EndLine:    &end,
				TotalLines: &total,
			}
		}
		return batches
	}

	// Check if remainder should be appended to last batch or be new batch
	remainderRatio := float64(remainder) / float64(batchSize)
	
	if remainderRatio < threshold {
		// Append remainder to last batch
		batches := make([]BatchRange, fullBatches)
		for i := 0; i < fullBatches-1; i++ {
			start := i * batchSize
			end := start + batchSize - 1
			total := batchSize
			batches[i] = BatchRange{
				StartLine:  &start,
				EndLine:    &end,
				TotalLines: &total,
			}
		}
		// Last batch includes remainder
		lastStart := (fullBatches - 1) * batchSize
		lastEnd := totalLines - 1
		lastTotal := totalLines - lastStart
		batches[fullBatches-1] = BatchRange{
			StartLine:  &lastStart,
			EndLine:    &lastEnd,
			TotalLines: &lastTotal,
		}
		return batches
	}

	// Create new batch for remainder
	batches := make([]BatchRange, fullBatches+1)
	for i := 0; i < fullBatches; i++ {
		start := i * batchSize
		end := start + batchSize - 1
		total := batchSize
		batches[i] = BatchRange{
			StartLine:  &start,
			EndLine:    &end,
			TotalLines: &total,
		}
	}
	// Remainder as new batch
	remainderStart := fullBatches * batchSize
	remainderEnd := totalLines - 1
	remainderTotal := remainder
	batches[fullBatches] = BatchRange{
		StartLine:  &remainderStart,
		EndLine:    &remainderEnd,
		TotalLines: &remainderTotal,
	}
	return batches
}

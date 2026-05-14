package main

import (
	"context"
	"fmt"
	"io"
	"runtime"
	"strings"
	"testing"

	aibridge "aibridge"
)

func TestHeuristicFallbackMappings_PriseToUnitPrice(t *testing.T) {
	headers := []string{"SKU", "Prise", "Qty"}
	resolved, unresolved := deterministicMappings(headers, supplierInventoryTargetFields)

	if _, ok := resolved["Prise"]; ok {
		t.Fatalf("expected misspelled header to remain unresolved for AI/fallback stage")
	}

	fallback := heuristicFallbackMappings(unresolved, supplierInventoryTargetFields)
	var found bool
	for _, candidate := range fallback {
		if candidate.SourceColumn == "Prise" {
			found = true
			if candidate.TargetField != "unit_price" {
				t.Fatalf("expected Prise -> unit_price, got %s", candidate.TargetField)
			}
			if candidate.Confidence < confidenceThreshold {
				t.Fatalf("expected confidence >= %.2f, got %.2f", confidenceThreshold, candidate.Confidence)
			}
		}
	}
	if !found {
		t.Fatalf("expected fallback mapping for misspelled Prise header")
	}
}

func TestResolveDiscoveryStatus_LowConfidenceRequiresMapping(t *testing.T) {
	status, lowConfidence := resolveDiscoveryStatus([]aibridge.MappingCandidate{
		{SourceColumn: "Column_A", TargetField: "sku_id", Confidence: 1},
		{SourceColumn: "Column_B", TargetField: "unit_price", Confidence: 0.61},
	})

	if status != "MAPPING_REQUIRED" {
		t.Fatalf("expected status MAPPING_REQUIRED, got %s", status)
	}
	if !lowConfidence {
		t.Fatalf("expected lowConfidence=true")
	}
}

func TestDiscoverMapping_FromSample_PriseToUnitPrice(t *testing.T) {
	csvPayload := "SKU,Prise,Qty\nSKU-1,1200,3\nSKU-2,1250,2\n"
	headers, rows, err := readDelimitedSample(strings.NewReader(csvPayload), ',', defaultImportSampleRows)
	if err != nil {
		t.Fatalf("readDelimitedSample failed: %v", err)
	}

	worker := &inventoryImportWorker{mapper: nil}
	outcome := worker.discoverMapping(context.Background(), aibridge.SampleData{
		Headers:      headers,
		Rows:         rows,
		TargetFields: supplierInventoryTargetFields,
	})

	candidate, ok := findCandidate(outcome.Mappings, "Prise")
	if !ok {
		t.Fatalf("expected mapping for Prise header")
	}
	if candidate.TargetField != "unit_price" {
		t.Fatalf("expected Prise -> unit_price, got %s", candidate.TargetField)
	}
	if candidate.Confidence < confidenceThreshold {
		t.Fatalf("expected confidence >= %.2f, got %.2f", confidenceThreshold, candidate.Confidence)
	}
	if outcome.Model != "deterministic-fallback" {
		t.Fatalf("expected deterministic-fallback model when mapper is unavailable, got %s", outcome.Model)
	}
}

func TestReadDelimitedSample_Streaming100kRows_MemoryUnder256MB(t *testing.T) {
	reader := &syntheticInventoryCSVReader{totalRows: 100_000}

	runtime.GC()
	var before runtime.MemStats
	runtime.ReadMemStats(&before)

	headers, rows, err := readDelimitedSample(reader, ',', defaultImportSampleRows)
	if err != nil {
		t.Fatalf("readDelimitedSample failed: %v", err)
	}
	if len(headers) == 0 {
		t.Fatalf("expected headers to be parsed")
	}
	if got := len(rows); got != defaultImportSampleRows {
		t.Fatalf("expected %d sampled rows, got %d", defaultImportSampleRows, got)
	}
	if reader.rowsEmitted > defaultImportSampleRows+2 {
		t.Fatalf("expected stream to stop near sample limit, emitted=%d", reader.rowsEmitted)
	}

	runtime.GC()
	var after runtime.MemStats
	runtime.ReadMemStats(&after)

	const memoryLimitBytes = 256 << 20
	if after.Alloc > memoryLimitBytes {
		t.Fatalf("expected process alloc under 256MB, got %.2fMB", float64(after.Alloc)/(1024*1024))
	}

	var allocDelta int64
	if after.Alloc > before.Alloc {
		allocDelta = int64(after.Alloc - before.Alloc)
	}
	if allocDelta > memoryLimitBytes {
		t.Fatalf("expected alloc delta under 256MB, got %.2fMB", float64(allocDelta)/(1024*1024))
	}
}

func findCandidate(candidates []aibridge.MappingCandidate, source string) (aibridge.MappingCandidate, bool) {
	for _, candidate := range candidates {
		if candidate.SourceColumn == source {
			return candidate, true
		}
	}
	return aibridge.MappingCandidate{}, false
}

type syntheticInventoryCSVReader struct {
	totalRows   int
	rowsEmitted int
	lineBuffer  []byte
	offset      int
}

func (r *syntheticInventoryCSVReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.lineBuffer) {
		line, done := r.nextLine()
		if done {
			return 0, io.EOF
		}
		r.lineBuffer = line
		r.offset = 0
	}

	n := copy(p, r.lineBuffer[r.offset:])
	r.offset += n
	return n, nil
}

func (r *syntheticInventoryCSVReader) nextLine() ([]byte, bool) {
	if r.rowsEmitted == 0 {
		r.rowsEmitted++
		return []byte("SKU,Prise,Qty,UpdatedAt\n"), false
	}

	dataRow := r.rowsEmitted
	if dataRow > r.totalRows {
		return nil, true
	}
	r.rowsEmitted++
	return []byte(fmt.Sprintf("SKU-%d,%d,%d,2025-01-01\n", dataRow, 1000+dataRow, (dataRow%9)+1)), false
}

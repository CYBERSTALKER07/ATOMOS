package parser

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

const defaultStreamChunkSize = 5000

// StreamDelimited reads a CSV/TSV payload in streaming fashion without any 10k row cap,
// inferring delimiter (',' | ';' | '\t' | '|') and detecting messy preamble rows automatically.
func StreamDelimited(streamName string, data []byte, encodingOverride string, onChunk StreamRowHandler) (ParsedStream, error) {
	decoded, err := DetectAndDecodeText(data, encodingOverride)
	if err != nil {
		return ParsedStream{}, err
	}
	cleanText := strings.TrimPrefix(decoded.Text, "\uFEFF")
	delim := inferDelimiterFromText(cleanText)

	reader := csv.NewReader(strings.NewReader(cleanText))
	reader.Comma = delim
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	var allSampleRaw [][]string
	for len(allSampleRaw) < 20 {
		rec, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return ParsedStream{}, fmt.Errorf("parse delimited row: %w", readErr)
		}
		allSampleRaw = append(allSampleRaw, rec)
	}
	if len(allSampleRaw) == 0 {
		return ParsedStream{}, fmt.Errorf("%w: empty delimited stream", ErrMalformedArtifact)
	}

	headerIdx := findHeaderRowIndex(allSampleRaw)
	headers := normalizeHeaders(allSampleRaw[headerIdx])

	meta := ParsedStream{
		StreamName:  fallbackStreamName(streamName, "sheet1"),
		Format:      formatForDelimiter(delim),
		Encoding:    decoded.Encoding,
		Headers:     headers,
		NeedsEncode: decoded.AmbiguousEncoding,
		PreviewRaw:  decoded.PreviewLines,
	}

	chunk := make([]ParsedRow, 0, defaultStreamChunkSize)
	var logicalIdx int64

	emitRow := func(rec []string) error {
		if isBlankRow(rec) {
			return nil
		}
		pr := ParsedRow{
			RowIndex: logicalIdx,
			Fields:   rowToMap(headers, rec),
		}
		logicalIdx++
		if len(meta.SampleRows) < 20 {
			meta.SampleRows = append(meta.SampleRows, pr)
		}
		if onChunk != nil {
			chunk = append(chunk, pr)
			if len(chunk) >= defaultStreamChunkSize {
				if err := onChunk(meta.StreamName, headers, chunk); err != nil {
					return err
				}
				chunk = chunk[:0]
			}
		}
		return nil
	}

	for i := headerIdx + 1; i < len(allSampleRaw); i++ {
		if err := emitRow(allSampleRaw[i]); err != nil {
			return ParsedStream{}, err
		}
	}

	for {
		rec, readErr := reader.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return ParsedStream{}, fmt.Errorf("parse delimited row %d: %w", logicalIdx, readErr)
		}
		if err := emitRow(rec); err != nil {
			return ParsedStream{}, err
		}
	}

	if onChunk != nil && len(chunk) > 0 {
		if err := onChunk(meta.StreamName, headers, chunk); err != nil {
			return ParsedStream{}, err
		}
	}

	meta.TotalRows = logicalIdx
	return meta, nil
}

func inferDelimiterFromText(text string) rune {
	candidates := []rune{';', ',', '\t', '|'}
	lines := firstNLines(text, 10)
	best := ','
	bestScore := -1
	for _, c := range candidates {
		counts := make([]int, 0, len(lines))
		for _, line := range lines {
			n := strings.Count(line, string(c))
			if n > 0 {
				counts = append(counts, n)
			}
		}
		if len(counts) == 0 {
			continue
		}
		// Prefer consistent non-zero column counts across sample lines (typical Billz/1C ';' exports).
		consistent := 0
		base := counts[0]
		for _, v := range counts {
			if v == base {
				consistent++
			}
		}
		score := consistent*100 + base
		if score > bestScore {
			bestScore = score
			best = c
		}
	}
	return best
}

func formatForDelimiter(delim rune) Format {
	if delim == '\t' {
		return FormatTSV
	}
	return FormatCSV
}

// findHeaderRowIndex picks the first row in the top sample that has >= 2 distinct non-empty columns
// (skipping Billz/1C merged banner rows like `"Отчет по остаткам на 01.11.2026"`).
func findHeaderRowIndex(rows [][]string) int {
	bestIdx := 0
	bestCount := -1
	limit := len(rows)
	if limit > 15 {
		limit = 15
	}
	for i := 0; i < limit; i++ {
		nonEmpty := 0
		for _, cell := range rows[i] {
			if strings.TrimSpace(cell) != "" {
				nonEmpty++
			}
		}
		if nonEmpty >= 2 {
			return i
		}
		if nonEmpty > bestCount {
			bestCount = nonEmpty
			bestIdx = i
		}
	}
	return bestIdx
}

func normalizeHeaders(raw []string) []string {
	out := make([]string, len(raw))
	seen := make(map[string]int, len(raw))
	for i, col := range raw {
		clean := strings.TrimSpace(strings.TrimPrefix(col, "\uFEFF"))
		if clean == "" {
			clean = fmt.Sprintf("column_%d", i+1)
		}
		if count, exists := seen[clean]; exists {
			seen[clean] = count + 1
			clean = fmt.Sprintf("%s_%d", clean, count+1)
		} else {
			seen[clean] = 1
		}
		out[i] = clean
	}
	return out
}

func isBlankRow(cells []string) bool {
	for _, c := range cells {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func rowToMap(headers, cells []string) map[string]any {
	m := make(map[string]any, len(headers))
	for i, h := range headers {
		if i < len(cells) {
			m[h] = strings.TrimSpace(cells[i])
		} else {
			m[h] = ""
		}
	}
	return m
}

func fallbackStreamName(name, def string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return def
	}
	return trimmed
}

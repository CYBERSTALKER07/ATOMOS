package parser

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"
)

// StreamXLSX streams every non-empty worksheet in an OOXML .xlsx workbook using excelize.Rows
// in 5,000-row chunks without imposing a 10k row cap.
func StreamXLSX(data []byte, onChunk StreamRowHandler) ([]ParsedStream, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{
		UnzipSizeLimit:    64 << 20, // 64 MiB uncompressed zip bomb cap
		UnzipXMLSizeLimit: 16 << 20, // 16 MiB XML stream threshold
	})
	if err != nil {
		lower := strings.ToLower(err.Error())
		if strings.Contains(lower, "password") || strings.Contains(lower, "encrypt") {
			return nil, fmt.Errorf("%w: %v", ErrPasswordProtectedWorkbook, err)
		}
		return nil, fmt.Errorf("%w: open xlsx: %v", ErrMalformedArtifact, err)
	}
	defer func() { _ = f.Close() }()

	sheetNames := f.GetSheetList()
	if len(sheetNames) == 0 {
		return nil, fmt.Errorf("%w: workbook contains no worksheets", ErrMalformedArtifact)
	}

	streams := make([]ParsedStream, 0, len(sheetNames))
	for _, sheet := range sheetNames {
		rowsIter, err := f.Rows(sheet)
		if err != nil {
			return nil, fmt.Errorf("stream xlsx sheet %q: %w", sheet, err)
		}

		var rawPreamble [][]string
		for len(rawPreamble) < 15 && rowsIter.Next() {
			cols, colErr := rowsIter.Columns()
			if colErr != nil {
				_ = rowsIter.Close()
				return nil, fmt.Errorf("read xlsx sheet %q header row: %w", sheet, colErr)
			}
			rawPreamble = append(rawPreamble, cols)
		}
		if len(rawPreamble) == 0 {
			_ = rowsIter.Close()
			continue
		}

		headerIdx := findHeaderRowIndex(rawPreamble)
		headers := normalizeHeaders(rawPreamble[headerIdx])
		if isBlankRow(rawPreamble[headerIdx]) {
			_ = rowsIter.Close()
			continue
		}

		meta := ParsedStream{
			StreamName: sheet,
			Format:     FormatXLSX,
			Encoding:   "utf-8",
			Headers:    headers,
		}
		chunk := make([]ParsedRow, 0, defaultStreamChunkSize)
		var logicalIdx int64

		pushRow := func(cols []string) error {
			if isBlankRow(cols) {
				return nil
			}
			pr := ParsedRow{
				RowIndex: logicalIdx,
				Fields:   rowToMap(headers, cols),
			}
			logicalIdx++
			if len(meta.SampleRows) < 20 {
				meta.SampleRows = append(meta.SampleRows, pr)
			}
			if onChunk != nil {
				chunk = append(chunk, pr)
				if len(chunk) >= defaultStreamChunkSize {
					if err := onChunk(sheet, headers, chunk); err != nil {
						return err
					}
					chunk = chunk[:0]
				}
			}
			return nil
		}

		for i := headerIdx + 1; i < len(rawPreamble); i++ {
			if err := pushRow(rawPreamble[i]); err != nil {
				_ = rowsIter.Close()
				return nil, err
			}
		}

		for rowsIter.Next() {
			cols, colErr := rowsIter.Columns()
			if colErr != nil {
				_ = rowsIter.Close()
				return nil, fmt.Errorf("read xlsx sheet %q row %d: %w", sheet, logicalIdx, colErr)
			}
			if err := pushRow(cols); err != nil {
				_ = rowsIter.Close()
				return nil, err
			}
		}
		if iterErr := rowsIter.Error(); iterErr != nil {
			_ = rowsIter.Close()
			return nil, fmt.Errorf("iterate xlsx sheet %q: %w", sheet, iterErr)
		}
		_ = rowsIter.Close()

		if onChunk != nil && len(chunk) > 0 {
			if err := onChunk(sheet, headers, chunk); err != nil {
				return nil, err
			}
		}

		meta.TotalRows = logicalIdx
		if logicalIdx > 0 || len(headers) >= 2 {
			streams = append(streams, meta)
		}
	}

	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: workbook contained no non-empty worksheets", ErrMalformedArtifact)
	}
	return streams, nil
}

// Package parser implements multi-format streaming ingestion readers for
// .xlsx, BIFF8 .xls, UTF-8/UTF-16/cp1251/cp866 CSV/TSV, 1C CommerceML XML, and JSON dumps.
package parser

import "errors"

// Format identifies the sniffed physical container / data format of an uploaded artifact.
type Format string

const (
	FormatXLSX       Format = "xlsx"
	FormatXLS        Format = "xls"
	FormatCSV        Format = "csv"
	FormatTSV        Format = "tsv"
	FormatCommerceML Format = "commerceml_xml"
	FormatJSON       Format = "json"
	FormatUnknown    Format = "unknown"
)

var (
	// ErrPasswordProtectedWorkbook is returned when an Excel file is encrypted / password-protected (HTTP 422).
	ErrPasswordProtectedWorkbook = errors.New("password_protected_workbook")
	// ErrFileFormatMismatch is returned when magic bytes contradict the declared file extension or are unsupported binary.
	ErrFileFormatMismatch = errors.New("file_format_mismatch")
	// ErrMalformedArtifact is returned when an archive, XML, or JSON container cannot be decoded.
	ErrMalformedArtifact = errors.New("malformed_artifact")
)

// ParsedRow represents a single decoded logical record extracted from an artifact stream.
type ParsedRow struct {
	RowIndex int64
	Fields   map[string]any
}

// ParsedStream describes one logical tabular stream extracted from a file (e.g. a workbook sheet,
// a CommerceML section like `Товары` or `Предложения`, or a named JSON array).
type ParsedStream struct {
	StreamName  string
	Format      Format
	Encoding    string
	Headers     []string
	SampleRows  []ParsedRow // up to first 20 rows for auto-sorting & AI mapping
	TotalRows   int64
	NeedsEncode bool // true when CSV/TSV encoding confidence is ambiguous
	PreviewRaw  []string
}

// StreamRowHandler is invoked for each decoded chunk (up to chunkSize rows) without capping total row count.
type StreamRowHandler func(streamName string, headers []string, chunk []ParsedRow) error

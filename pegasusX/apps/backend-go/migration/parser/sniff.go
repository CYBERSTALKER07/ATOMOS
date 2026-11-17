package parser

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"path/filepath"
	"strings"
)

var (
	zipMagic  = []byte{'P', 'K', 0x03, 0x04}
	ole2Magic = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
)

// SniffResult reports true container format, MIME type, and any spoofed extension mismatch.
type SniffResult struct {
	Format            Format
	DetectedMIME      string
	ExtensionMismatch bool
	Warning           string
}

// SniffArtifact validates magic bytes against extension spoofing, rejects encrypted workbooks,
// and detects ZIP/OOXML (.xlsx), OLE2 BIFF8 (.xls), CommerceML XML, JSON, CSV, and TSV.
func SniffArtifact(fileName string, data []byte) (SniffResult, error) {
	if len(data) == 0 {
		return SniffResult{}, fmt.Errorf("%w: empty artifact", ErrMalformedArtifact)
	}
	ext := strings.TrimPrefix(strings.ToLower(filepath.Ext(strings.TrimSpace(fileName))), ".")

	if bytes.HasPrefix(data, ole2Magic) {
		if isEncryptedOLE2(data) {
			return SniffResult{}, fmt.Errorf("%w: encrypted workbook requires password removal before upload", ErrPasswordProtectedWorkbook)
		}
		if ext != "" && ext != "xls" && ext != "xlsx" {
			return SniffResult{}, fmt.Errorf("%w: OLE2 workbook bytes do not match .%s extension", ErrFileFormatMismatch, ext)
		}
		return SniffResult{
			Format:            FormatXLS,
			DetectedMIME:      "application/vnd.ms-excel",
			ExtensionMismatch: ext == "xlsx",
		}, nil
	}

	if bytes.HasPrefix(data, zipMagic) {
		zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
		if err != nil {
			return SniffResult{}, fmt.Errorf("%w: invalid zip container: %v", ErrMalformedArtifact, err)
		}
		hasContentTypes := false
		for _, f := range zr.File {
			if f.Name == "[Content_Types].xml" {
				hasContentTypes = true
				break
			}
		}
		if !hasContentTypes {
			return SniffResult{}, fmt.Errorf("%w: zip archive is missing [Content_Types].xml (not a valid .xlsx workbook)", ErrFileFormatMismatch)
		}
		return SniffResult{
			Format:            FormatXLSX,
			DetectedMIME:      "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			ExtensionMismatch: ext != "" && ext != "xlsx",
		}, nil
	}

	// Reject binary executables, PDFs, images masquerading as text/spreadsheets.
	if bytes.HasPrefix(data, []byte("%PDF-")) || bytes.HasPrefix(data, []byte{0x7F, 'E', 'L', 'F'}) ||
		bytes.HasPrefix(data, []byte{0x89, 'P', 'N', 'G'}) || bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}) {
		return SniffResult{}, fmt.Errorf("%w: unsupported binary document signature", ErrFileFormatMismatch)
	}

	// Text formats: decode a prefix sample cleanly to classify XML vs JSON vs CSV/TSV.
	sampleLen := len(data)
	if sampleLen > 8192 {
		sampleLen = 8192
	}
	decoded, err := DetectAndDecodeText(data[:sampleLen], "")
	if err != nil {
		return SniffResult{}, err
	}
	trimmed := strings.TrimSpace(strings.TrimPrefix(decoded.Text, "\uFEFF"))
	switch {
	case strings.HasPrefix(trimmed, "<"):
		if isCommerceMLPrefix(trimmed) || ext == "xml" {
			return SniffResult{
				Format:            FormatCommerceML,
				DetectedMIME:      "application/xml",
				ExtensionMismatch: ext != "" && ext != "xml",
			}, nil
		}
		return SniffResult{}, fmt.Errorf("%w: XML file is not a recognized CommerceML or tabular export", ErrFileFormatMismatch)
	case strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "["):
		return SniffResult{
			Format:            FormatJSON,
			DetectedMIME:      "application/json",
			ExtensionMismatch: ext != "" && ext != "json",
		}, nil
	default:
		if hasExcessiveBinaryControlBytes(trimmed) {
			return SniffResult{}, fmt.Errorf("%w: unrecognized binary file bytes", ErrFileFormatMismatch)
		}
		delim := inferDelimiterFromText(trimmed)
		fmtOut := FormatCSV
		mime := "text/csv"
		if delim == '\t' || ext == "tsv" {
			fmtOut = FormatTSV
			mime = "text/tab-separated-values"
		}
		mismatch := ext != "" && ext != "csv" && ext != "tsv"
		warn := ""
		if mismatch {
			warn = "file_format_mismatch"
		}
		return SniffResult{
			Format:            fmtOut,
			DetectedMIME:      mime,
			ExtensionMismatch: mismatch,
			Warning:           warn,
		}, nil
	}
}

func isEncryptedOLE2(data []byte) bool {
	// OLE2 encrypted OOXML uses an "EncryptedPackage" UTF-16LE directory entry.
	encPkgUTF16 := []byte{'E', 0, 'n', 0, 'c', 0, 'r', 0, 'y', 0, 'p', 0, 't', 0, 'e', 0, 'd', 0, 'P', 0, 'a', 0, 'c', 0, 'k', 0, 'a', 0, 'g', 0, 'e', 0}
	if bytes.Contains(data, encPkgUTF16) {
		return true
	}
	// Classic BIFF8 FILEPASS record (opcode 0x002F) near workbook BOF.
	for i := 512; i+4 <= len(data); i++ {
		if data[i] == 0x2F && data[i+1] == 0x00 {
			recLen := binary.LittleEndian.Uint16(data[i+2 : i+4])
			// Sanity check: look back briefly for BOF (0x0809) or plausible length 6..256.
			if recLen >= 6 && recLen <= 256 && i+4+int(recLen) <= len(data) {
				// Confirm preceding bytes don't just happen inside random UTF-16 by checking encryption header 0x0000 or 0x0001
				if encType := binary.LittleEndian.Uint16(data[i+4 : i+6]); encType <= 4 {
					// Stronger guard: ensure a Workbook BOF appears in the file before claiming FILEPASS
					if bytes.Contains(data[:i], []byte{0x09, 0x08}) {
						return true
					}
				}
			}
		}
	}
	return false
}

func isCommerceMLPrefix(s string) bool {
	return strings.Contains(s, "КоммерческаяИнформация") ||
		strings.Contains(s, "<Каталог") ||
		strings.Contains(s, "<ПакетПредложений") ||
		strings.Contains(s, "<Документ") ||
		strings.Contains(s, "<?xml")
}

func hasExcessiveBinaryControlBytes(s string) bool {
	if s == "" {
		return true
	}
	bad := 0
	total := 0
	for _, r := range s {
		total++
		if r == 0 || (r < 0x20 && r != '\n' && r != '\r' && r != '\t') {
			bad++
		}
	}
	return total > 0 && bad*10 > total
}

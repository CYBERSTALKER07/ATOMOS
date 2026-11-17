package parser

import (
	"fmt"
	"path/filepath"
	"strings"
)

// ParseArtifact sniffs container bytes (rejecting encrypted workbooks and spoofed binaries),
// unpacks every logical stream (worksheets, CommerceML sections, CSV/TSV, JSON arrays), and
// delivers rows in unbounded 5,000-row streaming chunks.
func ParseArtifact(fileName string, data []byte, encodingOverride string, onChunk StreamRowHandler) ([]ParsedStream, SniffResult, error) {
	sniff, err := SniffArtifact(fileName, data)
	if err != nil {
		return nil, SniffResult{}, err
	}

	baseStream := strings.TrimSuffix(filepath.Base(strings.TrimSpace(fileName)), filepath.Ext(strings.TrimSpace(fileName)))
	switch sniff.Format {
	case FormatXLSX:
		streams, err := StreamXLSX(data, onChunk)
		return streams, sniff, err
	case FormatXLS:
		streams, err := StreamXLS(data, onChunk)
		return streams, sniff, err
	case FormatCSV, FormatTSV:
		stream, err := StreamDelimited(baseStream, data, encodingOverride, onChunk)
		if err != nil {
			return nil, sniff, err
		}
		return []ParsedStream{stream}, sniff, nil
	case FormatCommerceML:
		streams, err := StreamCommerceML(data, onChunk)
		return streams, sniff, err
	case FormatJSON:
		streams, err := StreamJSON(baseStream, data, onChunk)
		return streams, sniff, err
	default:
		return nil, sniff, fmt.Errorf("%w: unsupported format %s", ErrFileFormatMismatch, sniff.Format)
	}
}

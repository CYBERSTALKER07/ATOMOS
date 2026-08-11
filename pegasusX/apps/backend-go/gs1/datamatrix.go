// Package gs1 implements GS1 identifiers and label encodings used by PegasusX.
package gs1

import (
	"fmt"
	"strings"
)

// AIDataMatrixPayload builds a GS1 Application Identifier element string suitable
// for DataMatrix (ECC200) encoding. At minimum (01)GTIN; optional (10)lot and
// (21)serial. Full ECC200 symbology bytes are produced by EncodeDataMatrixModules
// (module matrix) — ZPL ^BX integration is separate.
type AIDataMatrixInput struct {
	GTIN   string
	Lot    string
	Serial string
}

// BuildAIElementString returns the FNC1-delimited GS1 AI string (without binary FNC1).
// Callers that need ISO/IEC 16022 FNC1 should prepend the symbology FNC1 codeword
// in their encoder; this function returns the human/AI concatenated form used by
// GS1 Digital Link / DM labeling: (01)…(10)…(21)…
func BuildAIElementString(in AIDataMatrixInput) (string, error) {
	gtin, err := NormalizeGTIN(in.GTIN)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("(01)")
	b.WriteString(gtin)
	if lot := strings.TrimSpace(in.Lot); lot != "" {
		if len(lot) > 20 {
			return "", fmt.Errorf("lot_too_long")
		}
		b.WriteString("(10)")
		b.WriteString(lot)
	}
	if serial := strings.TrimSpace(in.Serial); serial != "" {
		if len(serial) > 20 {
			return "", fmt.Errorf("serial_too_long")
		}
		b.WriteString("(21)")
		b.WriteString(serial)
	}
	return b.String(), nil
}

// EncodeDataMatrixModules returns a square boolean module matrix for a GS1 AI
// string. This is a deterministic placeholder ECC200-compatible layout used for
// ZPL/tests until a vetted ECC200 library is vendored — it is NOT a certified
// GS1 DataMatrix encoder. Production marking must swap in a certified library
// behind the same API (see docs/GS1_LABELS.md).
func EncodeDataMatrixModules(aiString string) ([][]bool, error) {
	aiString = strings.TrimSpace(aiString)
	if aiString == "" {
		return nil, fmt.Errorf("empty_ai_string")
	}
	// Size: next even >= 10 from payload length heuristic.
	n := len(aiString)
	size := 10
	for size*size/8 < n+4 {
		size += 2
		if size > 144 {
			return nil, fmt.Errorf("payload_too_large")
		}
	}
	m := make([][]bool, size)
	for i := range m {
		m[i] = make([]bool, size)
	}
	// Finder-like border (L pattern) for visual/ZPL scaffolding.
	for i := 0; i < size; i++ {
		m[0][i] = true
		m[i][0] = true
		m[size-1][i] = i%2 == 0
		m[i][size-1] = i%2 == 0
	}
	// Scatter payload bits interior.
	bit := 0
	payload := []byte(aiString)
	for y := 1; y < size-1; y++ {
		for x := 1; x < size-1; x++ {
			byteIdx := bit / 8
			if byteIdx >= len(payload) {
				m[y][x] = (x+y)%3 == 0
			} else {
				m[y][x] = (payload[byteIdx]>>(7-(bit%8)))&1 == 1
			}
			bit++
		}
	}
	return m, nil
}

// AIDataMatrixZPL emits a ZPL ^BX DataMatrix field from an AI element string.
// Uses ZPL's native DataMatrix; the module matrix above is for unit tests /
// non-ZPL sinks.
func AIDataMatrixZPL(aiString string, magnify int) (string, error) {
	aiString = strings.TrimSpace(aiString)
	if aiString == "" {
		return "", fmt.Errorf("empty_ai_string")
	}
	if magnify <= 0 {
		magnify = 5
	}
	// ^BX orientation, height, quality, columns, rows — Zebra DataMatrix.
	return fmt.Sprintf("^XA^FO50,50^BXN,%d,200^FD%s^FS^XZ", magnify, aiString), nil
}

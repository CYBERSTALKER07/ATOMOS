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
// element string using a real ISO/IEC 16022 (ECC200) encoder — ASCII encodation,
// Reed–Solomon error correction, and the standard module-placement algorithm —
// for square symbols up to 44x44 (144 data codewords). Larger payloads must use
// the ZPL ^BX path (AIDataMatrixZPL), which offloads ECC200 to printer firmware.
//
// The input may embed the 0xF1 byte to mark an FNC1 (GS1 group separator); use
// BuildAIElementStringFNC1 to produce a correctly-separated element string.
func EncodeDataMatrixModules(aiString string) ([][]bool, error) {
	return encodeECC200(aiString)
}

// BuildAIElementStringFNC1 returns the AI element string with FNC1 (0xF1)
// separators after variable-length AIs (10, 21) when more AIs follow — the form
// ECC200 encoders need for unambiguous parsing. (01) GTIN is fixed-length and
// takes no separator.
func BuildAIElementStringFNC1(in AIDataMatrixInput) (string, error) {
	gtin, err := NormalizeGTIN(in.GTIN)
	if err != nil {
		return "", err
	}
	lot := strings.TrimSpace(in.Lot)
	serial := strings.TrimSpace(in.Serial)
	var b strings.Builder
	b.WriteString("(01)")
	b.WriteString(gtin)
	b.WriteByte(0xF1) // FNC1 after GTIN when more AIs follow
	if lot != "" {
		if len(lot) > 20 {
			return "", fmt.Errorf("lot_too_long")
		}
		b.WriteString("(10)")
		b.WriteString(lot)
		if serial != "" {
			b.WriteByte(0xF1) // (10) is variable-length
		}
	}
	if serial != "" {
		if len(serial) > 20 {
			return "", fmt.Errorf("serial_too_long")
		}
		b.WriteString("(21)")
		b.WriteString(serial)
	}
	return b.String(), nil
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

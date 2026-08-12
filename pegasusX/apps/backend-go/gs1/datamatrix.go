// Package gs1 implements GS1 identifiers and label encodings used by PegasusX.
package gs1

import (
	"fmt"
	"strings"
)

// AIDataMatrixInput is GTIN + optional lot/serial for GS1 DataMatrix (ECC200).
type AIDataMatrixInput struct {
	GTIN   string
	Lot    string
	Serial string
}

// BuildAIElementString returns the human-readable AI form with parentheses:
// (01)GTIN[(10)lot][(21)serial]. Use for HRI text only — scanners need
// BuildAIElementStringFNC1 (no parens, FNC1 separators).
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

// BuildAIElementStringFNC1 returns the machine-readable GS1 AI element string:
// leading FNC1 (0xF1) for GS1 mode, then AI digits without parentheses.
// Fixed-length AI (01) needs no separator before the next AI; variable-length
// AIs (10)/(21) are terminated with FNC1 when another AI follows.
//
// Example (lot + serial): FNC1 + "01" + GTIN14 + "10" + lot + FNC1 + "21" + serial
func BuildAIElementStringFNC1(in AIDataMatrixInput) (string, error) {
	gtin, err := NormalizeGTIN(in.GTIN)
	if err != nil {
		return "", err
	}
	lot := strings.TrimSpace(in.Lot)
	serial := strings.TrimSpace(in.Serial)
	if lot != "" && len(lot) > 20 {
		return "", fmt.Errorf("lot_too_long")
	}
	if serial != "" && len(serial) > 20 {
		return "", fmt.Errorf("serial_too_long")
	}
	var b strings.Builder
	b.WriteByte(0xF1) // leading FNC1 → GS1 Data Matrix mode (ECC200 codeword 232)
	b.WriteString("01")
	b.WriteString(gtin)
	if lot != "" {
		b.WriteString("10")
		b.WriteString(lot)
		if serial != "" {
			b.WriteByte(0xF1) // (10) is variable-length
		}
	}
	if serial != "" {
		b.WriteString("21")
		b.WriteString(serial)
	}
	return b.String(), nil
}

// AIDataMatrixZPL emits a ZPL ^BX DataMatrix field from an FNC1 element string
// (BuildAIElementStringFNC1). FNC1 bytes are escaped as ^FH_ hex `_1D` (GS) so
// Zebra firmware treats the symbol as GS1 Data Matrix.
func AIDataMatrixZPL(aiString string, magnify int) (string, error) {
	aiString = strings.TrimSpace(aiString)
	if aiString == "" {
		return "", fmt.Errorf("empty_ai_string")
	}
	if magnify <= 0 {
		magnify = 5
	}
	// Reject HRI form — callers must pass BuildAIElementStringFNC1 output.
	if strings.Contains(aiString, "(") {
		return "", fmt.Errorf("ai_string_must_be_fnc1_form")
	}
	escaped := strings.ReplaceAll(aiString, "\xf1", "_1D")
	var b strings.Builder
	b.WriteString("^XA\n")
	b.WriteString(fmt.Sprintf("^FO50,50^BXN,%d,200\n", magnify))
	b.WriteString(fmt.Sprintf("^FH_^FD%s^FS\n", escaped))
	b.WriteString("^XZ\n")
	return b.String(), nil
}

// LabelDataMatrixZPL builds a GS1 DataMatrix ZPL block from label GTIN/lot/serial.
func LabelDataMatrixZPL(d LabelData, magnify int) (string, error) {
	if strings.TrimSpace(d.GTIN) == "" {
		return "", fmt.Errorf("gtin_required")
	}
	s, err := BuildAIElementStringFNC1(AIDataMatrixInput{
		GTIN: d.GTIN, Lot: d.Lot, Serial: d.Serial,
	})
	if err != nil {
		return "", err
	}
	return AIDataMatrixZPL(s, magnify)
}

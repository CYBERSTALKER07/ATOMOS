package gs1

import (
	"fmt"
	"strings"
)

// LabelData is input for a single logistics label.
type LabelData struct {
	SSCC         string
	GTIN         string
	ShipFromGLN  string
	ShipToGLN    string
	OrderID      string
	ManifestID   string
	SupplierName string
}

// AICode128ZPL emits a Zebra ZPL block with GS1-128 (00) SSCC.
func AICode128ZPL(d LabelData) (string, error) {
	sscc, err := NormalizeSSCC(d.SSCC)
	if err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("^XA\n")
	b.WriteString("^CF0,30\n")
	b.WriteString("^FO40,40^FDPegasusX SSCC^FS\n")
	if d.SupplierName != "" {
		b.WriteString(fmt.Sprintf("^FO40,80^FD%s^FS\n", zplEscape(d.SupplierName)))
	}
	if d.OrderID != "" {
		b.WriteString(fmt.Sprintf("^FO40,120^FDOrder %s^FS\n", zplEscape(d.OrderID)))
	}
	if d.ManifestID != "" {
		b.WriteString(fmt.Sprintf("^FO40,160^FDManifest %s^FS\n", zplEscape(d.ManifestID)))
	}
	// GS1-128: AI 00 + SSCC
	b.WriteString("^FO40,220^BY2\n")
	b.WriteString(fmt.Sprintf("^BCN,120,Y,N,N^FD>;>800%s^FS\n", sscc))
	b.WriteString(fmt.Sprintf("^FO40,360^FD(00) %s^FS\n", sscc))
	y := 400
	if gtin, err := NormalizeGTIN(d.GTIN); err == nil && d.GTIN != "" {
		b.WriteString(fmt.Sprintf("^FO40,%d^FD(01) %s^FS\n", y, gtin))
		y += 40
	}
	if gln, err := NormalizeGLN(d.ShipFromGLN); err == nil && d.ShipFromGLN != "" {
		b.WriteString(fmt.Sprintf("^FO40,%d^FDShip-from GLN %s^FS\n", y, gln))
		y += 40
	}
	if gln, err := NormalizeGLN(d.ShipToGLN); err == nil && d.ShipToGLN != "" {
		b.WriteString(fmt.Sprintf("^FO40,%d^FDShip-to GLN %s^FS\n", y, gln))
	}
	b.WriteString("^XZ\n")
	return b.String(), nil
}

// MultiLabelZPL concatenates multiple labels.
func MultiLabelZPL(labels []LabelData) (string, error) {
	var b strings.Builder
	for i, d := range labels {
		zpl, err := AICode128ZPL(d)
		if err != nil {
			return "", fmt.Errorf("label_%d: %w", i, err)
		}
		b.WriteString(zpl)
	}
	if b.Len() == 0 {
		return "", fmt.Errorf("no_labels")
	}
	return b.String(), nil
}

func zplEscape(s string) string {
	s = strings.ReplaceAll(s, "^", " ")
	s = strings.ReplaceAll(s, "~", " ")
	if len(s) > 48 {
		s = s[:48]
	}
	return s
}

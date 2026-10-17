package returns

import (
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/gs1"
)

// NormalizeBarcode strips non-digits and validates EAN-13 / GTIN-14 checksum when possible.
func NormalizeBarcode(raw string) (string, error) {
	code, err := gs1.NormalizeGTIN(raw)
	if err != nil {
		switch {
		case strings.Contains(err.Error(), "gtin_required"):
			return "", fmt.Errorf("barcode_required")
		case strings.Contains(err.Error(), "checksum"):
			return "", fmt.Errorf("invalid_barcode_checksum")
		case strings.Contains(err.Error(), "length"):
			return "", fmt.Errorf("unsupported_barcode_length")
		default:
			return "", err
		}
	}
	return code, nil
}

// SuggestedDisposition maps amend reason to a default gate disposition.
func SuggestedDisposition(reason string) string {
	switch strings.ToUpper(strings.TrimSpace(reason)) {
	case "DAMAGED", "WRONG_ITEM":
		return DispositionWriteOff
	default:
		return DispositionRestock
	}
}

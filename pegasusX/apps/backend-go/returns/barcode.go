package returns

import (
	"fmt"
	"strconv"
	"strings"
)

// NormalizeBarcode strips non-digits and validates EAN-13 / GTIN-14 checksum when possible.
func NormalizeBarcode(raw string) (string, error) {
	digits := make([]byte, 0, len(raw))
	for i := 0; i < len(raw); i++ {
		if raw[i] >= '0' && raw[i] <= '9' {
			digits = append(digits, raw[i])
		}
	}
	code := string(digits)
	if code == "" {
		return "", fmt.Errorf("barcode_required")
	}
	switch len(code) {
	case 8, 12, 13, 14:
		if !validGTINChecksum(code) {
			return "", fmt.Errorf("invalid_barcode_checksum")
		}
	default:
		return "", fmt.Errorf("unsupported_barcode_length")
	}
	return code, nil
}

func validGTINChecksum(code string) bool {
	n := len(code)
	if n < 8 {
		return false
	}
	sum := 0
	for i := 0; i < n-1; i++ {
		d, err := strconv.Atoi(string(code[i]))
		if err != nil {
			return false
		}
		posFromRight := n - 1 - i
		if posFromRight%2 == 1 {
			sum += d * 3
		} else {
			sum += d
		}
	}
	check, err := strconv.Atoi(string(code[n-1]))
	if err != nil {
		return false
	}
	return (10-(sum%10))%10 == check
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

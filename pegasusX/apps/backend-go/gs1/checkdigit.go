// Package gs1 provides GTIN/GLN/SSCC validators and ZPL label helpers (Gate-3 Wave 2C).
package gs1

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LabelsEnabled gates SSCC minting and ZPL endpoints.
func LabelsEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("GS1_LABELS_ENABLED")))
	if v == "" {
		return true
	}
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

func digitsOnly(raw string) string {
	var b strings.Builder
	for i := 0; i < len(raw); i++ {
		if raw[i] >= '0' && raw[i] <= '9' {
			b.WriteByte(raw[i])
		}
	}
	return b.String()
}

// GS1 Mod-10 (GTIN/GLN/SSCC): from right, odd positions ×3, even ×1, then check digit.
func validMod10(code string) bool {
	n := len(code)
	if n < 2 {
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

func mod10CheckDigit(body string) (int, error) {
	sum := 0
	n := len(body)
	for i := 0; i < n; i++ {
		d, err := strconv.Atoi(string(body[i]))
		if err != nil {
			return 0, err
		}
		// Treat body as missing check digit: positions counted from right of full code (body+check).
		posFromRight := n - i // check will be position 1 (odd → ×3 for rightmost body digit)
		if posFromRight%2 == 1 {
			sum += d * 3
		} else {
			sum += d
		}
	}
	return (10 - (sum % 10)) % 10, nil
}

// NormalizeGTIN strips non-digits and validates EAN-8/12/13 / GTIN-14.
func NormalizeGTIN(raw string) (string, error) {
	code := digitsOnly(raw)
	if code == "" {
		return "", fmt.Errorf("gtin_required")
	}
	switch len(code) {
	case 8, 12, 13, 14:
		if !validMod10(code) {
			return "", fmt.Errorf("invalid_gtin_checksum")
		}
	default:
		return "", fmt.Errorf("unsupported_gtin_length")
	}
	return code, nil
}

// ValidGTIN reports whether raw is a valid GTIN after normalization.
func ValidGTIN(raw string) bool {
	_, err := NormalizeGTIN(raw)
	return err == nil
}

// NormalizeGLN validates a 13-digit Global Location Number.
func NormalizeGLN(raw string) (string, error) {
	code := digitsOnly(raw)
	if code == "" {
		return "", fmt.Errorf("gln_required")
	}
	if len(code) != 13 {
		return "", fmt.Errorf("gln_must_be_13_digits")
	}
	if !validMod10(code) {
		return "", fmt.Errorf("invalid_gln_checksum")
	}
	return code, nil
}

// ValidGLN reports whether raw is a valid GLN.
func ValidGLN(raw string) bool {
	_, err := NormalizeGLN(raw)
	return err == nil
}

// NormalizeSSCC validates an 18-digit Serial Shipping Container Code.
func NormalizeSSCC(raw string) (string, error) {
	code := digitsOnly(raw)
	if code == "" {
		return "", fmt.Errorf("sscc_required")
	}
	if len(code) != 18 {
		return "", fmt.Errorf("sscc_must_be_18_digits")
	}
	if !validMod10(code) {
		return "", fmt.Errorf("invalid_sscc_checksum")
	}
	return code, nil
}

// ValidSSCC reports whether raw is a valid SSCC-18.
func ValidSSCC(raw string) bool {
	_, err := NormalizeSSCC(raw)
	return err == nil
}

// GenerateSSCC builds SSCC-18: extension(0) + companyPrefix + serial + check digit.
// companyPrefix length 7–10; serial padded so body (without check) is 17 digits.
func GenerateSSCC(companyPrefix string, serial uint64) (string, error) {
	prefix := digitsOnly(companyPrefix)
	if len(prefix) < 7 || len(prefix) > 10 {
		return "", fmt.Errorf("invalid_company_prefix")
	}
	serialDigits := 16 - len(prefix) // 17 body = 1 ext + prefix + serial
	if serialDigits < 1 {
		return "", fmt.Errorf("invalid_company_prefix")
	}
	maxSerial := uint64(1)
	for i := 0; i < serialDigits; i++ {
		maxSerial *= 10
	}
	if serial >= maxSerial {
		serial = serial % maxSerial
	}
	body := fmt.Sprintf("0%s%0*d", prefix, serialDigits, serial)
	if len(body) != 17 {
		return "", fmt.Errorf("sscc_body_len")
	}
	cd, err := mod10CheckDigit(body)
	if err != nil {
		return "", err
	}
	sscc := body + strconv.Itoa(cd)
	if !ValidSSCC(sscc) {
		return "", fmt.Errorf("sscc_generate_failed")
	}
	return sscc, nil
}

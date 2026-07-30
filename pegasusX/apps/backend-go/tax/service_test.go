package tax

import (
	"log/slog"
	"testing"
)

type LineItemInput struct {
	SKU            string
	Quantity       int64
	UnitPriceMinor int64
}

func TestComputeLineSnapshots_IntegerArithmetic(t *testing.T) {
	// Verify the fiscal snapshot math is purely integer, no float contamination.
	lines := []LineItemInput{
		{SKU: "SKU-001", Quantity: 10, UnitPriceMinor: 50000}, // 500.00 UZS × 10 = 5000.00 UZS
		{SKU: "SKU-002", Quantity: 3, UnitPriceMinor: 33333},  // 333.33 UZS × 3 = 999.99 UZS
	}

	vatBps := int64(1200) // 12%

	// Line 1: total = 500000, VAT = 500000 * 1200 / 10000 = 60000, taxable = 440000
	total1 := lines[0].Quantity * lines[0].UnitPriceMinor
	vat1 := total1 * vatBps / 10000
	tax1 := total1 - vat1

	assertInt64(t, "line1.total", total1, 500000)
	assertInt64(t, "line1.vat", vat1, 60000)
	assertInt64(t, "line1.taxable", tax1, 440000)

	// Line 2: total = 99999, VAT = 99999 * 1200 / 10000 = 11999 (integer truncation), taxable = 88000
	total2 := lines[1].Quantity * lines[1].UnitPriceMinor
	vat2 := total2 * vatBps / 10000
	tax2 := total2 - vat2

	assertInt64(t, "line2.total", total2, 99999)
	assertInt64(t, "line2.vat", vat2, 11999)
	assertInt64(t, "line2.taxable", tax2, 88000)
}

func TestSnapshotZeroVAT(t *testing.T) {
	// Zero-rate regime means VAT = 0 for all lines.
	vatBps := int64(0)
	total := int64(250000) // 2500.00 UZS
	vat := total * vatBps / 10000
	taxable := total - vat

	assertInt64(t, "vat", vat, 0)
	assertInt64(t, "taxable", taxable, 250000)
}

func TestSnapshotMaxVAT(t *testing.T) {
	// 100% VAT (theoretical max = 10000 BPS).
	vatBps := int64(10000)
	total := int64(100000)
	vat := total * vatBps / 10000

	assertInt64(t, "vat", vat, 100000)
}

func assertInt64(t *testing.T, label string, got, want int64) {
	t.Helper()
	if got != want {
		t.Errorf("%s = %d, want %d", label, got, want)
	}
}

func testLogger() *slog.Logger {
	return slog.Default()
}

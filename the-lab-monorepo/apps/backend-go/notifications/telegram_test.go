package notifications

import (
	"strings"
	"testing"
)

// ── formatAmount ──────────────────────────────────────────────────────────────

func TestFormatAmount_Zero(t *testing.T) {
	if got := formatAmount(0); got != "0" {
		t.Fatalf("formatAmount(0) = %q, want %q", got, "0")
	}
}

func TestFormatAmount_SingleDigit(t *testing.T) {
	if got := formatAmount(5); got != "5" {
		t.Fatalf("formatAmount(5) = %q, want %q", got, "5")
	}
}

func TestFormatAmount_TwoDigits(t *testing.T) {
	if got := formatAmount(50); got != "50" {
		t.Fatalf("formatAmount(50) = %q, want %q", got, "50")
	}
}

func TestFormatAmount_ThreeDigits_NoComma(t *testing.T) {
	if got := formatAmount(500); got != "500" {
		t.Fatalf("formatAmount(500) = %q, want %q", got, "500")
	}
}

func TestFormatAmount_Thousands(t *testing.T) {
	if got := formatAmount(1500); got != "1,500" {
		t.Fatalf("formatAmount(1500) = %q, want %q", got, "1,500")
	}
}

func TestFormatAmount_ExactThousand(t *testing.T) {
	if got := formatAmount(1000); got != "1,000" {
		t.Fatalf("formatAmount(1000) = %q, want %q", got, "1,000")
	}
}

func TestFormatAmount_Millions(t *testing.T) {
	if got := formatAmount(1500000); got != "1,500,000" {
		t.Fatalf("formatAmount(1500000) = %q, want %q", got, "1,500,000")
	}
}

func TestFormatAmount_Negative(t *testing.T) {
	if got := formatAmount(-1500000); got != "-1,500,000" {
		t.Fatalf("formatAmount(-1500000) = %q, want %q", got, "-1,500,000")
	}
}

func TestFormatAmount_TenMillion(t *testing.T) {
	if got := formatAmount(10000000); got != "10,000,000" {
		t.Fatalf("formatAmount(10000000) = %q, want %q", got, "10,000,000")
	}
}

// ── FormatPredictionAlert ──────────────────────────────────────────────────

func TestFormatPredictionAlert_ContainsShopName(t *testing.T) {
	msg := FormatPredictionAlert("Mega Market", 1500000)
	if !strings.Contains(msg, "Mega Market") {
		t.Fatalf("alert should contain shop name, got: %s", msg)
	}
}

func TestFormatPredictionAlert_ContainsFormattedAmount(t *testing.T) {
	msg := FormatPredictionAlert("Shop", 1500000)
	if !strings.Contains(msg, "1,500,000") {
		t.Fatalf("alert should contain formatted amount, got: %s", msg)
	}
}

func TestFormatPredictionAlert_ContainsHeader(t *testing.T) {
	msg := FormatPredictionAlert("Shop", 0)
	if !strings.Contains(msg, "AI Restock Alert") {
		t.Fatalf("alert should contain header, got: %s", msg)
	}
}

func TestFormatPredictionAlert_EmptyShopName(t *testing.T) {
	msg := FormatPredictionAlert("", 500000)
	if !strings.Contains(msg, "500,000") {
		t.Fatalf("alert with empty shop should still format amount, got: %s", msg)
	}
}

func TestFormatPredictionAlert_ZeroAmount(t *testing.T) {
	msg := FormatPredictionAlert("Shop", 0)
	if !strings.Contains(msg, "0") {
		t.Fatalf("alert with zero amount should show 0, got: %s", msg)
	}
}

func TestFormatPredictionAlert_MarkdownBoldLabels(t *testing.T) {
	msg := FormatPredictionAlert("Shop", 1000)
	if !strings.Contains(msg, "*Shop:*") || !strings.Contains(msg, "*Amount:*") {
		t.Fatalf("alert should use Markdown bold for labels, got: %s", msg)
	}
}

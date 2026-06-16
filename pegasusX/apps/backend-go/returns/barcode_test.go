package returns

import "testing"

func TestNormalizeBarcodeEAN13(t *testing.T) {
	code, err := NormalizeBarcode("5901234123457")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != "5901234123457" {
		t.Fatalf("got %q", code)
	}
}

func TestNormalizeBarcodeInvalidChecksum(t *testing.T) {
	_, err := NormalizeBarcode("5901234123450")
	if err == nil {
		t.Fatal("expected checksum error")
	}
}

func TestSuggestedDisposition(t *testing.T) {
	if SuggestedDisposition("DAMAGED") != DispositionWriteOff {
		t.Fatal("damaged should write off")
	}
	if SuggestedDisposition("MISSING") != DispositionRestock {
		t.Fatal("missing default restock")
	}
}

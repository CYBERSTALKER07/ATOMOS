package gs1

import (
	"strings"
	"testing"
)

func TestBuildAIElementString(t *testing.T) {
	s, err := BuildAIElementString(AIDataMatrixInput{
		GTIN:   "01234567890128",
		Lot:    "L1",
		Serial: "S9",
	})
	if err != nil {
		t.Fatal(err)
	}
	if s != "(01)01234567890128(10)L1(21)S9" {
		t.Fatalf("got %q", s)
	}
}

func TestBuildAIElementStringFNC1_NoParensLeadingFNC1(t *testing.T) {
	s, err := BuildAIElementStringFNC1(AIDataMatrixInput{
		GTIN:   "01234567890128",
		Lot:    "L1",
		Serial: "S9",
	})
	if err != nil {
		t.Fatal(err)
	}
	want := "\xf1010123456789012810L1\xf121S9"
	if s != want {
		t.Fatalf("got %q want %q", s, want)
	}
	if strings.ContainsAny(s, "()") {
		t.Fatal("machine form must not contain HRI parentheses")
	}
	m, err := EncodeDataMatrixModules(s)
	if err != nil || len(m) < 10 {
		t.Fatalf("modules=%d err=%v", len(m), err)
	}
	zpl, err := AIDataMatrixZPL(s, 5)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(zpl, "^FH_") || !strings.Contains(zpl, "_1D") || !strings.Contains(zpl, "^BX") {
		t.Fatalf("zpl missing GS1 escapes: %s", zpl)
	}
}

func TestAIDataMatrixZPL_RejectsHRI(t *testing.T) {
	_, err := AIDataMatrixZPL("(01)01234567890128", 5)
	if err == nil {
		t.Fatal("expected reject HRI form")
	}
}

func TestMultiLabelZPL_IncludesDataMatrixWhenGTIN(t *testing.T) {
	sscc, err := GenerateSSCC("0614141", 99)
	if err != nil {
		t.Fatal(err)
	}
	zpl, err := MultiLabelZPL([]LabelData{{
		SSCC: sscc, GTIN: "01234567890128", OrderID: "o1", ManifestID: "m1",
	}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(zpl, "^BCN") {
		t.Fatal("missing GS1-128")
	}
	if !strings.Contains(zpl, "^BX") || !strings.Contains(zpl, "^FH_") {
		t.Fatalf("missing DataMatrix: %s", zpl)
	}
}

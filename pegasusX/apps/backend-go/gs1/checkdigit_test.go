package gs1

import "testing"

func TestGTINKnown(t *testing.T) {
	// Classic GTIN-13 example 4006381333931
	code, err := NormalizeGTIN("4006381333931")
	if err != nil || code != "4006381333931" {
		t.Fatalf("got %q err=%v", code, err)
	}
	if ValidGTIN("4006381333930") {
		t.Fatal("bad checksum should fail")
	}
}

func TestGLNAndSSCCRoundTrip(t *testing.T) {
	// Build GLN: prefix-like body
	body := "061414100001" // 12 digits → check
	cd, err := mod10CheckDigit(body)
	if err != nil {
		t.Fatal(err)
	}
	gln := body + string(rune('0'+cd))
	if !ValidGLN(gln) {
		t.Fatalf("gln invalid %s", gln)
	}
	sscc, err := GenerateSSCC("0614141", 12345)
	if err != nil {
		t.Fatal(err)
	}
	if len(sscc) != 18 || !ValidSSCC(sscc) {
		t.Fatalf("sscc=%s", sscc)
	}
	if sscc[0] != '0' {
		t.Fatalf("extension digit want 0 got %c", sscc[0])
	}
}

func TestZPLContainsSSCC(t *testing.T) {
	sscc, err := GenerateSSCC("0614141", 99)
	if err != nil {
		t.Fatal(err)
	}
	zpl, err := AICode128ZPL(LabelData{SSCC: sscc, OrderID: "o1", ManifestID: "m1"})
	if err != nil {
		t.Fatal(err)
	}
	if !containsAll(zpl, "^XA", "^XZ", "(00)", sscc) {
		t.Fatalf("zpl=%s", zpl)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

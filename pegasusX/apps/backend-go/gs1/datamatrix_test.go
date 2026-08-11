package gs1

import "testing"

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
	m, err := EncodeDataMatrixModules(s)
	if err != nil || len(m) < 10 {
		t.Fatalf("modules=%d err=%v", len(m), err)
	}
	zpl, err := AIDataMatrixZPL(s, 5)
	if err != nil || zpl == "" {
		t.Fatalf("zpl err=%v", err)
	}
}

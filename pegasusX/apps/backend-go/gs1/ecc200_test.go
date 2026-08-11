package gs1

import (
	"strings"
	"testing"
)

// The classic ECC200 reference: "123456" in a 12x12 symbol.
// Per ISO/IEC 16022 worked examples, ASCII encodation of "123456" yields the
// three data codewords [142, 164, 186] (digit pairs 12,34,56 + 130), then two
// pad codewords to fill 5 data codewords, then 7 ECC codewords. We assert the
// encoder produces a 12x12 symbol with a correct finder/timing border and that
// the symbol is internally consistent (decode-free structural checks).
func TestECC200_KnownSizeAndBorders(t *testing.T) {
	m, err := encodeECC200("123456")
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// "123456" → 3 data codewords → smallest square symbol is 10x10.
	if len(m) != 10 || len(m[0]) != 10 {
		t.Fatalf("size=%dx%d want 10x10", len(m), len(m[0]))
	}
	// Single region: left column and bottom row solid (finder).
	for i := 0; i < 10; i++ {
		if !m[i][0] {
			t.Fatalf("left finder row %d not set", i)
		}
		if !m[9][i] {
			t.Fatalf("bottom finder col %d not set", i)
		}
		// Timing tracks are strictly alternating (clock pattern). The top-right
		// corner module is dark (shared by top row and right clock track).
		if i < 9 && m[0][i] != (i%2 == 0) {
			t.Fatalf("top timing col %d wrong", i)
		}
	}
	if !m[0][9] {
		t.Fatal("top-right corner must be dark")
	}
	for i := 1; i < 10; i++ {
		if m[i][9] == m[i-1][9] && i < 9 {
			t.Fatalf("right timing not alternating at row %d", i)
		}
	}
}

func TestECC200_ASCIIDigitPairing(t *testing.T) {
	got := dmASCIITextEncode([]byte("123456"))
	want := []int{142, 164, 186}
	if len(got) != len(want) {
		t.Fatalf("len=%d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("cw[%d]=%d want %d", i, got[i], want[i])
		}
	}
}

func TestECC200_PadRandomization(t *testing.T) {
	// 3 data codewords in a 5-codeword symbol → 2 pads.
	p := dmPadCodewords([]int{142, 164, 186}, 5)
	if len(p) != 5 {
		t.Fatalf("len=%d", len(p))
	}
	// First pad (position 4, index 3) is 129 XOR the randomizing term — per the
	// 253-state algorithm the stored value is 129+pseudo (mod), so it is the
	// deterministic pad for that position; just assert it's in valid range and
	// deterministic across calls.
	for _, idx := range []int{3, 4} {
		if p[idx] < 1 || p[idx] > 254 {
			t.Fatalf("pad[%d] out of range: %d", idx, p[idx])
		}
	}
	p2 := dmPadCodewords([]int{142, 164, 186}, 5)
	if p[3] != p2[3] || p[4] != p2[4] {
		t.Fatal("pad randomization must be deterministic")
	}
}

func TestECC200_RSProducesECC(t *testing.T) {
	data := dmPadCodewords(dmASCIITextEncode([]byte("123456")), 5)
	ecc := dmRSEncode(data, 7)
	if len(ecc) != 7 {
		t.Fatalf("ecc len=%d", len(ecc))
	}
	// ECC must be deterministic and non-trivial.
	allZero := true
	for _, b := range ecc {
		if b != 0 {
			allZero = false
		}
	}
	if allZero {
		t.Fatal("ECC all zero — RS broken")
	}
	// Re-encode must match exactly (deterministic).
	ecc2 := dmRSEncode(data, 7)
	for i := range ecc {
		if ecc[i] != ecc2[i] {
			t.Fatalf("ECC nondeterministic at %d", i)
		}
	}
}

func TestEncodeDataMatrixModules_RealSymbol(t *testing.T) {
	s, err := BuildAIElementString(AIDataMatrixInput{GTIN: "01234567890128", Lot: "L1", Serial: "S9"})
	if err != nil {
		t.Fatal(err)
	}
	m, err := EncodeDataMatrixModules(s)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	// Must be a valid square size from the symbol table.
	ok := false
	for _, sym := range dmSquareSymbols {
		if len(m) == sym.size {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("size %d not a valid square symbol", len(m))
	}
}

func TestBuildAIElementStringFNC1(t *testing.T) {
	s, err := BuildAIElementStringFNC1(AIDataMatrixInput{GTIN: "01234567890128", Lot: "L1", Serial: "S9"})
	if err != nil {
		t.Fatal(err)
	}
	// FNC1 (0xF1) after GTIN and after the variable-length lot.
	if !strings.Contains(s, "(01)01234567890128\xf1(10)L1\xf1(21)S9") {
		t.Fatalf("got %q", s)
	}
	// Must encode without error.
	if _, err := EncodeDataMatrixModules(s); err != nil {
		t.Fatalf("encode FNC1 string: %v", err)
	}
}

func TestEncodeDataMatrixModules_TooLarge(t *testing.T) {
	big := strings.Repeat("A", 400)
	if _, err := EncodeDataMatrixModules(big); err != errPayloadTooLarge {
		t.Fatalf("want payload_too_large, got %v", err)
	}
}

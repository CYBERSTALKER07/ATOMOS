package gs1

// ecc200.go is a real ISO/IEC 16022 (ECC200) DataMatrix encoder for the square
// symbol sizes, replacing the earlier deterministic placeholder. It implements:
//   - ASCII encodation (default) with Pad and Upper Shift, plus the FNC1
//     codeword (232) used to separate GS1 variable-length AIs.
//   - Reed–Solomon error correction over GF(256) with generator polynomial
//     x^8+x^5+x^3+x^2+1 (0b100101101 = 0x12D), interleaved per the ECC200 tables.
//   - Standard codeword module placement (the four corner cases + two "utah"
//     shapes) and the finder/timing pattern.
//
// Only the square sizes up to 44x44 are supported (single RS block; no
// interleave). These cover every GS1 AI element string this package builds
// (<= 144 data codewords). Larger payloads should use the ZPL ^BX path
// (AIDataMatrixZPL), which offloads ECC200 to the printer firmware.
//
// References: ISO/IEC 16022; the public-domain annex tables (symbol attributes,
// RS block counts, placement algorithm) reproduced below.

// dmSymbol describes one square symbol size.
type dmSymbol struct {
	size      int // module width/height (incl. finder)
	dataCW    int // number of data codewords
	eccCW      int // number of error codewords (single block; no interleave <= 44x44)
	matrixSize int // data-region module width (size without finder, per region grid)
	regionSize int // modules per region side (size/regionCount - 2)
	regions    int // regions per side
}

// Square ECC200 symbol table (single RS block up to 44x44).
var dmSquareSymbols = []dmSymbol{
	{size: 10, dataCW: 3, eccCW: 5, regions: 1, regionSize: 8},
	{size: 12, dataCW: 5, eccCW: 7, regions: 1, regionSize: 10},
	{size: 14, dataCW: 8, eccCW: 10, regions: 1, regionSize: 12},
	{size: 16, dataCW: 12, eccCW: 12, regions: 1, regionSize: 14},
	{size: 18, dataCW: 18, eccCW: 14, regions: 1, regionSize: 16},
	{size: 20, dataCW: 22, eccCW: 18, regions: 1, regionSize: 18},
	{size: 22, dataCW: 30, eccCW: 20, regions: 1, regionSize: 20},
	{size: 24, dataCW: 36, eccCW: 22, regions: 1, regionSize: 22},
	{size: 26, dataCW: 44, eccCW: 28, regions: 1, regionSize: 24},
	{size: 32, dataCW: 62, eccCW: 36, regions: 2, regionSize: 14},
	{size: 36, dataCW: 86, eccCW: 42, regions: 2, regionSize: 16},
	{size: 40, dataCW: 114, eccCW: 48, regions: 2, regionSize: 18},
	{size: 44, dataCW: 144, eccCW: 56, regions: 2, regionSize: 20},
	// 52x52 uses 2 interleaved RS blocks; we cap at 44x44 to keep the single-block
	// encoder correct and simple. Payloads larger than 144 codewords go to ZPL.
}

const dmFNC1 = 232 // ASCII-encodation FNC1 codeword (GS1 separator)

// dmPickSymbol returns the smallest square symbol that fits dataCW codewords.
func dmPickSymbol(need int) (dmSymbol, bool) {
	for _, s := range dmSquareSymbols {
		if s.dataCW >= need {
			return s, true
		}
	}
	return dmSymbol{}, false
}

// --- ASCII encodation -------------------------------------------------------

// dmASCIITextEncode encodes input into ECC200 ASCII encodation codewords.
// FNC1 bytes (0x1D group separators are mapped by callers to FNC1) are emitted
// as the 232 codeword. Returns data codewords (unpadded).
func dmASCIITextEncode(in []byte) []int {
	out := make([]int, 0, len(in)+2)
	for i := 0; i < len(in); i++ {
		c := in[i]
		if c >= '0' && c <= '9' && i+1 < len(in) && in[i+1] >= '0' && in[i+1] <= '9' {
			// Double-digit pair → 130 + value.
			v := int(c-'0')*10 + int(in[i+1]-'0')
			out = append(out, 130+v)
			i++
			continue
		}
		if c == 0xF1 { // explicit FNC1 marker byte
			out = append(out, dmFNC1)
			continue
		}
		if c <= 0x7F {
			out = append(out, int(c)+1)
		} else {
			// Upper shift for extended ASCII.
			out = append(out, 235, int(c)-0x7F)
		}
	}
	return out
}

// dmPadCodewords pads with 129 and the randomizing algorithm to exactly n.
func dmPadCodewords(cw []int, n int) []int {
	out := append([]int(nil), cw...)
	for len(out) < n {
		out = append(out, 129)
	}
	// Randomize pad codewords (253-state algorithm) for positions >= len(cw).
	for pos := len(cw); pos < n; pos++ {
		pseudo := ((149 * (pos + 1)) % 253) + 1
		v := 129 + pseudo
		if v > 254 {
			v -= 254
		}
		out[pos] = v
	}
	return out
}

// --- Reed–Solomon over GF(256) ----------------------------------------------

var (
	dmGFExp [512]int
	dmGFLog [256]int
)

func init() {
	// Primitive poly 0x12D (301): x^8+x^5+x^3+x^2+1.
	x := 1
	for i := 0; i < 255; i++ {
		dmGFExp[i] = x
		dmGFLog[x] = i
		x <<= 1
		if x&0x100 != 0 {
			x ^= 0x12D
		}
	}
	for i := 255; i < 512; i++ {
		dmGFExp[i] = dmGFExp[i-255]
	}
}

func dmGFMul(a, b int) int {
	if a == 0 || b == 0 {
		return 0
	}
	return dmGFExp[dmGFLog[a]+dmGFLog[b]]
}

// dmRSGenerator returns the RS generator polynomial coefficients (degree ecc).
func dmRSGenerator(ecc int) []int {
	poly := []int{1}
	for i := 0; i < ecc; i++ {
		next := make([]int, len(poly)+1)
		for j := range poly {
			next[j] ^= dmGFMul(poly[j], dmGFExp[i])
			next[j+1] ^= poly[j]
		}
		poly = next
	}
	return poly
}

// dmRSEncode appends ecc error-correction codewords for the single block.
func dmRSEncode(data []int, ecc int) []int {
	gen := dmRSGenerator(ecc)
	eccw := make([]int, ecc)
	for _, d := range data {
		factor := d ^ eccw[0]
		copy(eccw, eccw[1:])
		eccw[ecc-1] = 0
		for j := 0; j < ecc; j++ {
			eccw[j] ^= dmGFMul(gen[ecc-1-j], factor)
		}
	}
	return eccw
}

// --- Module placement --------------------------------------------------------

// dmPlaceData allocates the data-region module grid and runs the standard
// placement walk. n = data-region side (modules, excluding finder).
func dmPlaceData(codewords []int, n int) [][]bool {
	mod := make([][]bool, n)
	for i := range mod {
		mod[i] = make([]bool, n)
	}
	assigned := make([][]bool, n)
	for i := range assigned {
		assigned[i] = make([]bool, n)
	}
	return dmPlaceStandard(codewords, n, mod, assigned)
}

// dmPlaceStandard implements the ISO/IEC 16022 § placement: repeated
// corner/utah traversal moving up-left in 2-wide column bands.
func dmPlaceStandard(codewords []int, n int, mod, assigned [][]bool) [][]bool {
	bit := 0
	total := len(codewords) * 8

	var module func(row, col int)
	module = func(row, col int) {
		if row < 0 {
			row += n
			col += 4 - ((n + 4) % 8)
		}
		if col < 0 {
			col += n
			row += 4 - ((n + 4) % 8)
		}
		if !assigned[row][col] && bit < total {
			cw := bit / 8
			b := 7 - (bit % 8)
			mod[row][col] = (codewords[cw]>>b)&1 == 1
			assigned[row][col] = true
			bit++
		}
	}

	utah := func(row, col int) {
		module(row-2, col-2)
		module(row-2, col-1)
		module(row-1, col-2)
		module(row-1, col-1)
		module(row-1, col)
		module(row, col-2)
		module(row, col-1)
		module(row, col)
	}
	corner1 := func() {
		module(n-1, 0)
		module(n-1, 1)
		module(n-1, 2)
		module(0, n-2)
		module(0, n-1)
		module(1, n-1)
		module(2, n-1)
		module(3, n-1)
	}
	corner2 := func() {
		module(n-3, 0)
		module(n-2, 0)
		module(n-1, 0)
		module(0, n-4)
		module(0, n-3)
		module(0, n-2)
		module(0, n-1)
		module(1, n-1)
	}
	corner3 := func() {
		module(n-3, 0)
		module(n-2, 0)
		module(n-1, 0)
		module(0, n-2)
		module(0, n-1)
		module(1, n-1)
		module(2, n-1)
		module(3, n-1)
	}
	corner4 := func() {
		module(n-1, 0)
		module(n-1, n-1)
		module(0, n-3)
		module(0, n-2)
		module(0, n-1)
		module(1, n-3)
		module(1, n-2)
		module(1, n-1)
	}

	row, col := 4, 0
	for bit < total {
		if row == n && col == 0 {
			corner1()
		} else if row == n-2 && col == 0 && (n%8 != 0) {
			corner2()
		} else if row == n-2 && col == 0 && (n%8 == 4) {
			corner3()
		} else if row == n+4 && col == 2 && (n%8 == 0) {
			corner4()
		}
		// Sweep up.
		for row >= 0 && col < n {
			if row < n && col >= 0 && !assigned[row][col] {
				utah(row, col)
			}
			row -= 2
			col += 2
		}
		row++
		col += 3
		// Sweep down.
		for row < n && col >= 0 {
			if row >= 0 && col < n && !assigned[row][col] {
				utah(row, col)
			}
			row += 2
			col -= 2
		}
		row += 3
		col++
	}

	// Fixed bottom-right corner module is always dark if unset.
	if !assigned[n-1][n-1] {
		mod[n-1][n-1] = true
		mod[n-2][n-2] = true
	}
	return mod
}

// --- Symbol assembly ---------------------------------------------------------

// encodeECC200 encodes a GS1 AI element string into a full square module matrix
// (with finder/timing). Group separators are inserted as FNC1 between variable
// AIs by the caller via the 0xF1 marker byte.
func encodeECC200(aiString string) ([][]bool, error) {
	if aiString == "" {
		return nil, errEmptyAI
	}
	raw := []byte(aiString)
	data := dmASCIITextEncode(raw)
	sym, ok := dmPickSymbol(len(data))
	if !ok {
		return nil, errPayloadTooLarge
	}
	padded := dmPadCodewords(data, sym.dataCW)
	ecc := dmRSEncode(padded, sym.eccCW)
	// Codeword stream = data + ecc.
	cw := append(append([]int(nil), padded...), ecc...)

	regionModules := sym.regionSize
	dataSide := sym.regions * regionModules

	// Place into data-region grid.
	dataGrid := dmPlaceData(cw, dataSide)

	// Assemble full symbol with finder/timing patterns per region.
	full := make([][]bool, sym.size)
	for i := range full {
		full[i] = make([]bool, sym.size)
	}
	rs := regionModules + 2 // region pitch incl. finder rows/cols
	for rr := 0; rr < sym.regions; rr++ {
		for rc := 0; rc < sym.regions; rc++ {
			or := rr * rs
			oc := rc * rs
			// Finder: left column solid, bottom row solid.
			for i := 0; i < rs; i++ {
				full[or+rs-1][oc+i] = true // bottom
				full[or+i][oc] = true      // left
			}
			// Timing: top row alternating, right column alternating. Stop one
			// short of the bottom/right edges so we never overwrite the solid
			// finder bottom-right corner (it belongs to the bottom finder row).
			for i := 0; i < rs-1; i++ {
				full[or][oc+i] = i%2 == 0      // top (skip top-right corner)
				full[or+i][oc+rs-1] = i%2 == 0 // right (skip bottom-right corner)
			}
			// Data modules.
			for y := 0; y < regionModules; y++ {
				for x := 0; x < regionModules; x++ {
					full[or+1+y][oc+1+x] = dataGrid[rr*regionModules+y][rc*regionModules+x]
				}
			}
		}
	}
	return full, nil
}

// dmError is a sentinel set for shared errors.
type dmError string

func (e dmError) Error() string { return string(e) }

const (
	errEmptyAI         dmError = "empty_ai_string"
	errPayloadTooLarge dmError = "payload_too_large"
)

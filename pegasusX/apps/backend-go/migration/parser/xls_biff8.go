package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"sort"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/richardlehane/mscfb"
	"golang.org/x/text/encoding/charmap"
)

// BIFF8 record opcodes required for worksheet data & encryption guards.
const (
	biffOpcodeBOF8       uint16 = 0x0809
	biffOpcodeEOF        uint16 = 0x000A
	biffOpcodeFilePass   uint16 = 0x002F
	biffOpcodeCodepage   uint16 = 0x0042
	biffOpcodeContinue   uint16 = 0x003C
	biffOpcodeBoundSheet uint16 = 0x0085
	biffOpcodeSST        uint16 = 0x00FC
	biffOpcodeLabelSST   uint16 = 0x00FD
	biffOpcodeLabel      uint16 = 0x0204
	biffOpcodeRString    uint16 = 0x00D6
	biffOpcodeNumber     uint16 = 0x0203
	biffOpcodeRK         uint16 = 0x027E
	biffOpcodeMulRK      uint16 = 0x00BD
	biffOpcodeBoolErr    uint16 = 0x0205
	biffOpcodeFormula    uint16 = 0x0006
	biffOpcodeString     uint16 = 0x0207
)

type biffRecord struct {
	opcode uint16
	data   []byte
}

type boundSheetInfo struct {
	name   string
	offset uint32
}

// StreamXLS parses legacy BIFF8 Excel .xls OLE2 compound files in pure Go,
// enforcing FILEPASS encryption rejection and decoding cp1251/UTF-16LE strings.
func StreamXLS(data []byte, onChunk StreamRowHandler) ([]ParsedStream, error) {
	reader, err := mscfb.New(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("%w: open xls ole2 container: %v", ErrMalformedArtifact, err)
	}

	var workbookBytes []byte
	for entry, nextErr := reader.Next(); nextErr == nil; entry, nextErr = reader.Next() {
		nameLower := strings.ToLower(strings.TrimSpace(entry.Name))
		if nameLower == "encryptedpackage" {
			return nil, fmt.Errorf("%w: encrypted xls workbook", ErrPasswordProtectedWorkbook)
		}
		if nameLower == "workbook" || nameLower == "book" {
			buf, readErr := io.ReadAll(entry)
			if readErr != nil {
				return nil, fmt.Errorf("read xls workbook stream: %w", readErr)
			}
			workbookBytes = buf
		}
	}
	if len(workbookBytes) == 0 {
		return nil, fmt.Errorf("%w: xls container missing Workbook stream", ErrMalformedArtifact)
	}

	codepage := uint16(1251)
	sheets, sst, err := parseBIFFGlobals(workbookBytes, &codepage)
	if err != nil {
		return nil, err
	}
	if len(sheets) == 0 {
		return nil, fmt.Errorf("%w: xls workbook has no worksheets", ErrMalformedArtifact)
	}

	streams := make([]ParsedStream, 0, len(sheets))
	for _, sheet := range sheets {
		grid := parseBIFFWorksheet(workbookBytes, int(sheet.offset), sst, codepage)
		if len(grid) == 0 {
			continue
		}
		headerIdx := findHeaderRowIndex(grid)
		if isBlankRow(grid[headerIdx]) {
			continue
		}
		headers := normalizeHeaders(grid[headerIdx])

		meta := ParsedStream{
			StreamName: sheet.name,
			Format:     FormatXLS,
			Encoding:   fmt.Sprintf("biff8-cp%d", codepage),
			Headers:    headers,
		}
		chunk := make([]ParsedRow, 0, defaultStreamChunkSize)
		var logicalIdx int64

		for i := headerIdx + 1; i < len(grid); i++ {
			if isBlankRow(grid[i]) {
				continue
			}
			pr := ParsedRow{
				RowIndex: logicalIdx,
				Fields:   rowToMap(headers, grid[i]),
			}
			logicalIdx++
			if len(meta.SampleRows) < 20 {
				meta.SampleRows = append(meta.SampleRows, pr)
			}
			if onChunk != nil {
				chunk = append(chunk, pr)
				if len(chunk) >= defaultStreamChunkSize {
					if err := onChunk(sheet.name, headers, chunk); err != nil {
						return nil, err
					}
					chunk = chunk[:0]
				}
			}
		}
		if onChunk != nil && len(chunk) > 0 {
			if err := onChunk(sheet.name, headers, chunk); err != nil {
				return nil, err
			}
		}
		meta.TotalRows = logicalIdx
		if logicalIdx > 0 || len(headers) >= 2 {
			streams = append(streams, meta)
		}
	}

	if len(streams) == 0 {
		return nil, fmt.Errorf("%w: xls contained no non-empty sheets", ErrMalformedArtifact)
	}
	return streams, nil
}

func readBIFFRecord(stream []byte, pos int) (biffRecord, int, bool) {
	if pos+4 > len(stream) {
		return biffRecord{}, pos, false
	}
	opcode := binary.LittleEndian.Uint16(stream[pos : pos+2])
	length := int(binary.LittleEndian.Uint16(stream[pos+2 : pos+4]))
	pos += 4
	if pos+length > len(stream) {
		length = len(stream) - pos
	}
	return biffRecord{opcode: opcode, data: stream[pos : pos+length]}, pos + length, true
}

func parseBIFFGlobals(stream []byte, codepage *uint16) ([]boundSheetInfo, []string, error) {
	var sheets []boundSheetInfo
	var sst []string
	pos := 0
	for {
		rec, nextPos, ok := readBIFFRecord(stream, pos)
		if !ok {
			break
		}
		pos = nextPos
		switch rec.opcode {
		case biffOpcodeFilePass:
			return nil, nil, fmt.Errorf("%w: password-protected BIFF8 workbook", ErrPasswordProtectedWorkbook)
		case biffOpcodeCodepage:
			if len(rec.data) >= 2 {
				*codepage = binary.LittleEndian.Uint16(rec.data[:2])
			}
		case biffOpcodeBoundSheet:
			if len(rec.data) >= 8 {
				offset := binary.LittleEndian.Uint32(rec.data[0:4])
				sheetType := rec.data[5]
				if sheetType == 0x00 { // worksheet
					name, _ := decodeBIFF8ShortString(rec.data[6:], *codepage)
					if strings.TrimSpace(name) != "" {
						sheets = append(sheets, boundSheetInfo{name: name, offset: offset})
					}
				}
			}
		case biffOpcodeSST:
			// Gather immediately following CONTINUE records into segmented slice readers.
			chunks := [][]byte{rec.data}
			for {
				peekPos := pos
				nextRec, afterPeek, hasNext := readBIFFRecord(stream, peekPos)
				if !hasNext || nextRec.opcode != biffOpcodeContinue {
					break
				}
				chunks = append(chunks, nextRec.data)
				pos = afterPeek
			}
			sst = decodeBIFF8SST(chunks, *codepage)
		case biffOpcodeEOF:
			return sheets, sst, nil
		}
	}
	return sheets, sst, nil
}

func parseBIFFWorksheet(stream []byte, startOffset int, sst []string, codepage uint16) [][]string {
	if startOffset < 0 || startOffset >= len(stream) {
		return nil
	}
	cells := make(map[int]map[int]string)
	maxRow := -1
	maxCol := -1

	setCell := func(r, c int, val string) {
		rowMap, ok := cells[r]
		if !ok {
			rowMap = make(map[int]string)
			cells[r] = rowMap
		}
		rowMap[c] = val
		if r > maxRow {
			maxRow = r
		}
		if c > maxCol {
			maxCol = c
		}
	}

	pos := startOffset
	expectFormulaString := false
	var lastFormulaRow, lastFormulaCol int

	for {
		recStart := pos
		rec, nextPos, ok := readBIFFRecord(stream, pos)
		if !ok {
			break
		}
		pos = nextPos
		if rec.opcode == biffOpcodeBOF8 && recStart > startOffset {
			// Hit next substream without EOF.
			break
		}
		if rec.opcode == biffOpcodeEOF {
			break
		}

		if expectFormulaString && rec.opcode != biffOpcodeString {
			expectFormulaString = false
		}

		switch rec.opcode {
		case biffOpcodeLabelSST:
			if len(rec.data) >= 10 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				idx := int(binary.LittleEndian.Uint32(rec.data[6:10]))
				if idx >= 0 && idx < len(sst) {
					setCell(r, c, sst[idx])
				}
			}
		case biffOpcodeLabel, biffOpcodeRString:
			if len(rec.data) >= 9 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				txt, _ := decodeBIFF8Unicode16LenString(rec.data[6:], codepage)
				setCell(r, c, txt)
			}
		case biffOpcodeNumber:
			if len(rec.data) >= 14 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				bits := binary.LittleEndian.Uint64(rec.data[6:14])
				setCell(r, c, formatBIFFFloat(math.Float64frombits(bits)))
			}
		case biffOpcodeRK:
			if len(rec.data) >= 10 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				rk := binary.LittleEndian.Uint32(rec.data[6:10])
				setCell(r, c, formatBIFFFloat(decodeRK(rk)))
			}
		case biffOpcodeMulRK:
			if len(rec.data) >= 10 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				colFirst := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				n := (len(rec.data) - 6) / 6
				for i := 0; i < n; i++ {
					off := 4 + i*6 + 2
					if off+4 <= len(rec.data) {
						rk := binary.LittleEndian.Uint32(rec.data[off : off+4])
						setCell(r, colFirst+i, formatBIFFFloat(decodeRK(rk)))
					}
				}
			}
		case biffOpcodeBoolErr:
			if len(rec.data) >= 8 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				val := rec.data[6]
				isErr := rec.data[7] != 0
				if !isErr {
					if val == 0 {
						setCell(r, c, "FALSE")
					} else {
						setCell(r, c, "TRUE")
					}
				}
			}
		case biffOpcodeFormula:
			if len(rec.data) >= 14 {
				r := int(binary.LittleEndian.Uint16(rec.data[0:2]))
				c := int(binary.LittleEndian.Uint16(rec.data[2:4]))
				// String result marker: bytes 12..13 == 0xFFFF and byte 6 == 0x00.
				if rec.data[6] == 0x00 && rec.data[12] == 0xFF && rec.data[13] == 0xFF {
					expectFormulaString = true
					lastFormulaRow = r
					lastFormulaCol = c
				} else if rec.data[12] != 0xFF || rec.data[13] != 0xFF {
					bits := binary.LittleEndian.Uint64(rec.data[6:14])
					setCell(r, c, formatBIFFFloat(math.Float64frombits(bits)))
				}
			}
		case biffOpcodeString:
			if expectFormulaString && len(rec.data) >= 3 {
				txt, _ := decodeBIFF8Unicode16LenString(rec.data, codepage)
				setCell(lastFormulaRow, lastFormulaCol, txt)
				expectFormulaString = false
			}
		}
	}

	if maxRow < 0 || maxCol < 0 {
		return nil
	}
	rowIndices := make([]int, 0, len(cells))
	for r := range cells {
		rowIndices = append(rowIndices, r)
	}
	sort.Ints(rowIndices)

	grid := make([][]string, 0, maxRow+1)
	for r := 0; r <= maxRow; r++ {
		line := make([]string, maxCol+1)
		if rowMap, ok := cells[r]; ok {
			for c, val := range rowMap {
				if c >= 0 && c < len(line) {
					line[c] = val
				}
			}
		}
		grid = append(grid, line)
	}
	return grid
}

func decodeRK(rk uint32) float64 {
	var val float64
	if rk&0x02 != 0 {
		// 30-bit signed integer
		val = float64(int32(rk) >> 2)
	} else {
		// Top 30 bits of IEEE-754 float64
		bits := uint64(rk&0xFFFFFFFC) << 32
		val = math.Float64frombits(bits)
	}
	if rk&0x01 != 0 {
		val /= 100.0
	}
	return val
}

func formatBIFFFloat(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return ""
	}
	if v == math.Trunc(v) && v >= -9e15 && v <= 9e15 {
		return strconv.FormatInt(int64(v), 10)
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func decodeBIFF8ShortString(b []byte, codepage uint16) (string, int) {
	if len(b) < 2 {
		return "", 0
	}
	charCount := int(b[0])
	flags := b[1]
	isUTF16 := (flags & 0x01) != 0
	pos := 2
	if isUTF16 {
		need := charCount * 2
		if pos+need > len(b) {
			need = ((len(b) - pos) / 2) * 2
		}
		return decodeUTF16(b[pos:pos+need], binary.LittleEndian), pos + need
	}
	if pos+charCount > len(b) {
		charCount = len(b) - pos
	}
	return decodeBIFF8SingleByte(b[pos:pos+charCount], codepage), pos + charCount
}

func decodeBIFF8Unicode16LenString(b []byte, codepage uint16) (string, int) {
	if len(b) < 3 {
		return "", 0
	}
	charCount := int(binary.LittleEndian.Uint16(b[0:2]))
	flags := b[2]
	isUTF16 := (flags & 0x01) != 0
	pos := 3
	if (flags & 0x04) != 0 { // phonetic
		if pos+4 <= len(b) {
			pos += 4
		}
	}
	if (flags & 0x08) != 0 { // rich text
		if pos+2 <= len(b) {
			runs := int(binary.LittleEndian.Uint16(b[pos : pos+2]))
			pos += 2 + runs*4
		}
	}
	if pos > len(b) {
		return "", len(b)
	}
	if isUTF16 {
		need := charCount * 2
		if pos+need > len(b) {
			need = ((len(b) - pos) / 2) * 2
		}
		return decodeUTF16(b[pos:pos+need], binary.LittleEndian), pos + need
	}
	if pos+charCount > len(b) {
		charCount = len(b) - pos
	}
	return decodeBIFF8SingleByte(b[pos:pos+charCount], codepage), pos + charCount
}

// biffChunkReader reads across SST + CONTINUE boundaries, switching encoding flag on CONTINUE splits as spec requires.
type biffChunkReader struct {
	chunks [][]byte
	idx    int
	pos    int
}

func (r *biffChunkReader) remainingInChunk() int {
	if r.idx >= len(r.chunks) {
		return 0
	}
	return len(r.chunks[r.idx]) - r.pos
}

func (r *biffChunkReader) readBytes(n int) ([]byte, bool) {
	out := make([]byte, 0, n)
	for len(out) < n {
		for r.remainingInChunk() == 0 {
			r.idx++
			r.pos = 0
			if r.idx >= len(r.chunks) {
				return out, false
			}
		}
		take := n - len(out)
		if rem := r.remainingInChunk(); take > rem {
			take = rem
		}
		out = append(out, r.chunks[r.idx][r.pos:r.pos+take]...)
		r.pos += take
	}
	return out, true
}

func decodeBIFF8SST(chunks [][]byte, codepage uint16) []string {
	if len(chunks) == 0 || len(chunks[0]) < 8 {
		return nil
	}
	uniqueCount := int(binary.LittleEndian.Uint32(chunks[0][4:8]))
	r := &biffChunkReader{chunks: chunks, idx: 0, pos: 8}
	out := make([]string, 0, uniqueCount)

	for len(out) < uniqueCount {
		hdr, ok := r.readBytes(3)
		if !ok {
			break
		}
		charsLeft := int(binary.LittleEndian.Uint16(hdr[0:2]))
		flags := hdr[2]
		isUTF16 := (flags & 0x01) != 0
		var rtRuns, extBytes int
		if flags&0x08 != 0 {
			if rb, ok := r.readBytes(2); ok {
				rtRuns = int(binary.LittleEndian.Uint16(rb))
			}
		}
		if flags&0x04 != 0 {
			if eb, ok := r.readBytes(4); ok {
				extBytes = int(binary.LittleEndian.Uint32(eb))
			}
		}

		var sb strings.Builder
		for charsLeft > 0 {
			if r.remainingInChunk() == 0 {
				r.idx++
				r.pos = 0
				if r.idx >= len(r.chunks) {
					break
				}
				// CONTINUE chunk inside a string begins with a 1-byte high/low compression flag.
				if len(r.chunks[r.idx]) > 0 {
					isUTF16 = (r.chunks[r.idx][0] & 0x01) != 0
					r.pos = 1
				}
			}
			rem := r.remainingInChunk()
			if isUTF16 {
				canChars := rem / 2
				if canChars == 0 {
					// Boundary mid-char safety break
					break
				}
				takeChars := charsLeft
				if takeChars > canChars {
					takeChars = canChars
				}
				seg := r.chunks[r.idx][r.pos : r.pos+takeChars*2]
				r.pos += takeChars * 2
				u16 := make([]uint16, takeChars)
				for i := 0; i < takeChars; i++ {
					u16[i] = binary.LittleEndian.Uint16(seg[i*2 : i*2+2])
				}
				sb.WriteString(string(utf16.Decode(u16)))
				charsLeft -= takeChars
			} else {
				takeChars := charsLeft
				if takeChars > rem {
					takeChars = rem
				}
				seg := r.chunks[r.idx][r.pos : r.pos+takeChars]
				r.pos += takeChars
				sb.WriteString(decodeBIFF8SingleByte(seg, codepage))
				charsLeft -= takeChars
			}
		}

		if rtRuns > 0 {
			_, _ = r.readBytes(rtRuns * 4)
		}
		if extBytes > 0 {
			_, _ = r.readBytes(extBytes)
		}
		out = append(out, sb.String())
	}
	return out
}

func decodeBIFF8SingleByte(b []byte, codepage uint16) string {
	switch codepage {
	case 1251:
		s, err := charmap.Windows1251.NewDecoder().String(string(b))
		if err == nil {
			return s
		}
	case 866:
		s, err := charmap.CodePage866.NewDecoder().String(string(b))
		if err == nil {
			return s
		}
	}
	// BIFF8 compressed Unicode (latin1 fallback)
	runes := make([]rune, len(b))
	for i, c := range b {
		runes[i] = rune(c)
	}
	return string(runes)
}

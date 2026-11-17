package parser

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"golang.org/x/text/encoding/charmap"
)

// DecodedText contains UTF-8 normalized text along with detected encoding & confidence hints.
type DecodedText struct {
	Encoding          string
	Text              string
	Confidence        float64
	AmbiguousEncoding bool
	PreviewLines      []string
}

// DetectAndDecodeText detects UTF-8 BOM, UTF-16LE/BE, strict UTF-8, Windows-1251, or CP866,
// returning normalized UTF-8 text plus the top 5 preview lines for interactive confirmation if ambiguous.
func DetectAndDecodeText(data []byte, overrideEncoding string) (DecodedText, error) {
	if len(data) == 0 {
		return DecodedText{Encoding: "utf-8", Confidence: 1}, nil
	}

	if enc := normalizeEncodingName(overrideEncoding); enc != "" {
		txt, err := decodeWithNamedEncoding(data, enc)
		if err != nil {
			return DecodedText{}, err
		}
		return DecodedText{
			Encoding:     enc,
			Text:         txt,
			Confidence:   1.0,
			PreviewLines: firstNLines(txt, 5),
		}, nil
	}

	// 1. BOM checks
	switch {
	case bytes.HasPrefix(data, []byte{0xEF, 0xBB, 0xBF}):
		txt := string(data[3:])
		return DecodedText{Encoding: "utf-8", Text: txt, Confidence: 1.0, PreviewLines: firstNLines(txt, 5)}, nil
	case bytes.HasPrefix(data, []byte{0xFF, 0xFE}):
		txt := decodeUTF16(data[2:], binary.LittleEndian)
		return DecodedText{Encoding: "utf-16le", Text: txt, Confidence: 1.0, PreviewLines: firstNLines(txt, 5)}, nil
	case bytes.HasPrefix(data, []byte{0xFE, 0xFF}):
		txt := decodeUTF16(data[2:], binary.BigEndian)
		return DecodedText{Encoding: "utf-16be", Text: txt, Confidence: 1.0, PreviewLines: firstNLines(txt, 5)}, nil
	}

	// 2. Heuristic UTF-16 without BOM (lots of 0x00 bytes on alternating offsets)
	if isLikelyUTF16LE(data) {
		txt := decodeUTF16(data, binary.LittleEndian)
		return DecodedText{Encoding: "utf-16le", Text: txt, Confidence: 0.92, PreviewLines: firstNLines(txt, 5)}, nil
	}

	// 3. Valid UTF-8
	if utf8.Valid(data) {
		txt := string(data)
		return DecodedText{Encoding: "utf-8", Text: txt, Confidence: 0.99, PreviewLines: firstNLines(txt, 5)}, nil
	}

	// 4. Score 8-bit Cyrillic codepages (Windows-1251 vs CP866)
	txt1251, _ := charmap.Windows1251.NewDecoder().String(string(data))
	txt866, _ := charmap.CodePage866.NewDecoder().String(string(data))

	s1251 := scoreRussianUzbekPlausibility(txt1251)
	s866 := scoreRussianUzbekPlausibility(txt866)

	chosenEnc := "windows-1251"
	chosenTxt := txt1251
	conf := 0.90
	ambiguous := false

	if s866 > s1251 {
		chosenEnc = "cp866"
		chosenTxt = txt866
	}
	diff := s1251 - s866
	if diff < 0 {
		diff = -diff
	}
	if s1251+s866 > 0 && diff*10 <= (s1251+s866) {
		conf = 0.62
		ambiguous = true
	}

	return DecodedText{
		Encoding:          chosenEnc,
		Text:              chosenTxt,
		Confidence:        conf,
		AmbiguousEncoding: ambiguous,
		PreviewLines:      firstNLines(chosenTxt, 5),
	}, nil
}

func normalizeEncodingName(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "utf8", "utf-8":
		return "utf-8"
	case "utf16le", "utf-16le":
		return "utf-16le"
	case "utf16be", "utf-16be":
		return "utf-16be"
	case "cp1251", "windows-1251", "win1251":
		return "windows-1251"
	case "cp866", "ibm866", "866":
		return "cp866"
	default:
		return ""
	}
}

func decodeWithNamedEncoding(data []byte, enc string) (string, error) {
	switch enc {
	case "utf-8":
		return strings.TrimPrefix(string(data), "\uFEFF"), nil
	case "utf-16le":
		return strings.TrimPrefix(decodeUTF16(bytes.TrimPrefix(data, []byte{0xFF, 0xFE}), binary.LittleEndian), "\uFEFF"), nil
	case "utf-16be":
		return strings.TrimPrefix(decodeUTF16(bytes.TrimPrefix(data, []byte{0xFE, 0xFF}), binary.BigEndian), "\uFEFF"), nil
	case "windows-1251":
		return charmap.Windows1251.NewDecoder().String(string(data))
	case "cp866":
		return charmap.CodePage866.NewDecoder().String(string(data))
	default:
		return "", fmt.Errorf("unsupported encoding %q", enc)
	}
}

func decodeUTF16(b []byte, order binary.ByteOrder) string {
	n := len(b) / 2
	u16 := make([]uint16, 0, n)
	for i := 0; i+1 < len(b); i += 2 {
		u16 = append(u16, order.Uint16(b[i:i+2]))
	}
	return string(utf16.Decode(u16))
}

func isLikelyUTF16LE(data []byte) bool {
	limit := len(data)
	if limit > 256 {
		limit = 256
	}
	if limit < 8 {
		return false
	}
	oddZero := 0
	evenNonZero := 0
	for i := 0; i+1 < limit; i += 2 {
		if data[i] != 0 {
			evenNonZero++
		}
		if data[i+1] == 0 {
			oddZero++
		}
	}
	pairs := limit / 2
	return evenNonZero > pairs/2 && oddZero*2 >= pairs
}

// scoreRussianUzbekPlausibility rewards common lowercase Cyrillic & Kazakh/Uzbek Cyrillic letters
// and ASCII digits/punctuation, while heavily penalizing box-drawing characters typical of wrong codepage.
func scoreRussianUzbekPlausibility(txt string) int {
	score := 0
	for _, r := range txt {
		switch {
		case (r >= 'а' && r <= 'я') || r == 'ё' || r == 'ў' || r == 'қ' || r == 'ғ' || r == 'ҳ':
			score += 3
		case (r >= 'А' && r <= 'Я') || r == 'Ё' || r == 'Ў' || r == 'Қ' || r == 'Ғ' || r == 'Ҳ':
			score += 1
		case r >= 0x2500 && r <= 0x257F: // box drawing = cp866 decoded as 1251 or vice versa
			score -= 10
		case r == '\uFFFD':
			score -= 15
		}
	}
	return score
}

func firstNLines(txt string, n int) []string {
	rawLines := strings.Split(strings.ReplaceAll(txt, "\r\n", "\n"), "\n")
	out := make([]string, 0, n)
	for _, line := range rawLines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		out = append(out, line)
		if len(out) == n {
			break
		}
	}
	return out
}

// Package edi implements Pegasus EDI-lite (EDIFACT-ish UNA segment files).
// Not a certified EDIFACT/X12 implementation — see docs/PARTNER_EDI.md.
package edi

import (
	"fmt"
	"strings"
	"unicode"
)

const (
	DefaultUNA = "UNA:+.? '"

	DocTypeORDERS = "ORDERS"
	DocTypeORDRSP = "ORDRSP"
	DocTypeDESADV = "DESADV"
	DocTypeINVOIC = "INVOIC"
	DocTypeCONTRL = "CONTRL"
	DocTypeAPERAK = "APERAK"
	// Extended EDI-lite (P2-20) — still not certified EDIFACT.
	DocTypePRICAT = "PRICAT"
	DocTypeINVRPT = "INVRPT"
	DocTypeSLSRPT = "SLSRPT"
	DocTypeRECADV = "RECADV"
	DocTypeORDCHG = "ORDCHG"
	DocTypeDELFOR = "DELFOR"
	DocTypeREMADV = "REMADV"
)

// Segment is one EDI segment (tag + elements).
type Segment struct {
	Tag      string
	Elements []string
}

// ParseUNAFile splits a UNA-style EDI-lite document into segments.
func ParseUNAFile(raw string) (una string, segs []Segment, err error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil, fmt.Errorf("empty_edi")
	}
	// UNA layout: UNA + comp + elem + decimal + release + reserved + segmentTerm
	sep := '\''
	comp := ':'
	elem := '+'
	release := '?'
	una = DefaultUNA
	if strings.HasPrefix(raw, "UNA") && len(raw) >= 9 {
		una = raw[:9]
		comp = rune(una[3])
		elem = rune(una[4])
		release = rune(una[6])
		sep = rune(una[8])
		raw = strings.TrimSpace(raw[9:])
	}
	_ = comp
	var cur strings.Builder
	escaped := false
	flush := func() error {
		s := strings.TrimSpace(cur.String())
		cur.Reset()
		if s == "" {
			return nil
		}
		parts := splitElem(s, elem, release)
		if len(parts) == 0 || parts[0] == "" {
			return fmt.Errorf("empty_segment")
		}
		segs = append(segs, Segment{Tag: parts[0], Elements: parts[1:]})
		return nil
	}
	for _, r := range raw {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == release {
			escaped = true
			continue
		}
		if r == sep {
			if err := flush(); err != nil {
				return una, nil, err
			}
			continue
		}
		if r == '\n' || r == '\r' {
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		if err := flush(); err != nil {
			return una, nil, err
		}
	}
	if len(segs) == 0 {
		return una, nil, fmt.Errorf("no_segments")
	}
	return una, segs, nil
}

func splitElem(s string, elem, release rune) []string {
	var out []string
	var cur strings.Builder
	escaped := false
	for _, r := range s {
		if escaped {
			cur.WriteRune(r)
			escaped = false
			continue
		}
		if r == release {
			escaped = true
			continue
		}
		if r == elem {
			out = append(out, cur.String())
			cur.Reset()
			continue
		}
		cur.WriteRune(r)
	}
	out = append(out, cur.String())
	return out
}

// WriteUNAFile encodes segments with DefaultUNA terminators.
func WriteUNAFile(segs []Segment) string {
	var b strings.Builder
	b.WriteString(DefaultUNA)
	b.WriteByte('\n')
	for _, s := range segs {
		b.WriteString(s.Tag)
		for _, e := range s.Elements {
			b.WriteByte('+')
			b.WriteString(escapeElem(e))
		}
		b.WriteByte('\'')
		b.WriteByte('\n')
	}
	return b.String()
}

func escapeElem(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '+' || r == ':' || r == '?' || r == '\'' {
			b.WriteByte('?')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// Elem returns element i (0-based) or "".
func (s Segment) Elem(i int) string {
	if i < 0 || i >= len(s.Elements) {
		return ""
	}
	return strings.TrimSpace(s.Elements[i])
}

// Comp splits a composite on ':' (first component).
func Comp(v string, i int) string {
	parts := strings.Split(v, ":")
	if i < 0 || i >= len(parts) {
		return ""
	}
	return strings.TrimSpace(parts[i])
}

func digitsOnly(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsDigit(r) || r == '.' || r == '-' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

package edi

import (
	"fmt"
	"strings"
	"time"
)

// BuildCONTRL emits a minimal EDIFACT CONTRL (syntax acknowledgment) for an
// inbound interchange/reference. Action: 7=accepted, 4=rejected.
func BuildCONTRL(ourGLN, theirGLN, refDocID string, accepted bool, now time.Time) (string, error) {
	refDocID = strings.TrimSpace(refDocID)
	if refDocID == "" {
		return "", fmt.Errorf("ref_doc_id_required")
	}
	action := "7"
	if !accepted {
		action = "4"
	}
	una := "UNA:+.? '"
	unb := fmt.Sprintf("UNB+UNOC:3+%s:14+%s:14+%s:%s+%s'",
		sanitizeEDI(ourGLN), sanitizeEDI(theirGLN),
		now.UTC().Format("060102"), now.UTC().Format("1504"), sanitizeEDI(refDocID+"C"))
	unh := fmt.Sprintf("UNH+%s+CONTRL:D:96A:UN'", sanitizeEDI(refDocID+"H"))
	ucd := fmt.Sprintf("UCI+%s+%s:14+%s:14+%s'", sanitizeEDI(refDocID), sanitizeEDI(theirGLN), sanitizeEDI(ourGLN), action)
	unt := "UNT+3+" + sanitizeEDI(refDocID+"H") + "'"
	unz := "UNZ+1+" + sanitizeEDI(refDocID+"C") + "'"
	return una + unb + unh + ucd + unt + unz, nil
}

// BuildAPERAK emits a minimal EDIFACT APERAK (application acknowledgment).
// Code: 27=accepted, 29=rejected (aligned with ORDRSP response codes).
func BuildAPERAK(ourGLN, theirGLN, refDocID, reason string, accepted bool, now time.Time) (string, error) {
	refDocID = strings.TrimSpace(refDocID)
	if refDocID == "" {
		return "", fmt.Errorf("ref_doc_id_required")
	}
	code := "27"
	if !accepted {
		code = "29"
	}
	una := "UNA:+.? '"
	unb := fmt.Sprintf("UNB+UNOC:3+%s:14+%s:14+%s:%s+%s'",
		sanitizeEDI(ourGLN), sanitizeEDI(theirGLN),
		now.UTC().Format("060102"), now.UTC().Format("1504"), sanitizeEDI(refDocID+"A"))
	unh := fmt.Sprintf("UNH+%s+APERAK:D:96A:UN'", sanitizeEDI(refDocID+"H"))
	bgm := fmt.Sprintf("BGM+12+%s+%s'", sanitizeEDI(refDocID), code)
	ftx := ""
	if r := strings.TrimSpace(reason); r != "" && !accepted {
		ftx = fmt.Sprintf("FTX+AAO+++%s'", sanitizeEDI(r))
	}
	segCount := 3
	body := una + unb + unh + bgm
	if ftx != "" {
		body += ftx
		segCount++
	}
	unt := fmt.Sprintf("UNT+%d+%s'", segCount, sanitizeEDI(refDocID+"H"))
	unz := "UNZ+1+" + sanitizeEDI(refDocID+"A") + "'"
	return body + unt + unz, nil
}

func sanitizeEDI(s string) string {
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "+", " ")
	s = strings.ReplaceAll(s, ":", " ")
	return strings.TrimSpace(s)
}

package as2

import (
	"bytes"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"strings"
)

// MDNDisposition is the processing result reported in a sync MDN.
type MDNDisposition string

const (
	MDNProcessed MDNDisposition = "automatic-action/MDN-sent-automatically; processed"
	MDNFailed    MDNDisposition = "automatic-action/MDN-sent-automatically; failed"
)

// BuildSyncMDN builds an unsigned multipart/report MDN body and content-type.
func BuildSyncMDN(original MessageHeaders, mic string, disposition MDNDisposition, humanText string) (contentType string, body []byte, err error) {
	if humanText == "" {
		if disposition == MDNProcessed {
			humanText = "The AS2 message has been received and processed."
		} else {
			humanText = "The AS2 message could not be processed."
		}
	}
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	boundary := w.Boundary()

	// Part 1: human readable
	h1 := textproto.MIMEHeader{}
	h1.Set("Content-Type", "text/plain; charset=us-ascii")
	h1.Set("Content-Transfer-Encoding", "7bit")
	p1, err := w.CreatePart(h1)
	if err != nil {
		return "", nil, err
	}
	_, _ = p1.Write([]byte(humanText + "\r\n"))

	// Part 2: disposition-notification
	h2 := textproto.MIMEHeader{}
	h2.Set("Content-Type", "message/disposition-notification")
	h2.Set("Content-Transfer-Encoding", "7bit")
	p2, err := w.CreatePart(h2)
	if err != nil {
		return "", nil, err
	}
	var dn strings.Builder
	dn.WriteString("Reporting-UA: PegasusX-AS2/1.0\r\n")
	if original.AS2From != "" {
		dn.WriteString("Original-Recipient: rfc822; " + original.AS2From + "\r\n")
		dn.WriteString("Final-Recipient: rfc822; " + original.AS2From + "\r\n")
	}
	if original.MessageID != "" {
		dn.WriteString("Original-Message-ID: " + original.MessageID + "\r\n")
	}
	dn.WriteString("Disposition: " + string(disposition) + "\r\n")
	if mic != "" {
		dn.WriteString("Received-Content-MIC: " + mic + "\r\n")
	}
	_, _ = p2.Write([]byte(dn.String()))
	if err := w.Close(); err != nil {
		return "", nil, err
	}

	ct := fmt.Sprintf("multipart/report; report-type=disposition-notification; boundary=\"%s\"", boundary)
	return ct, buf.Bytes(), nil
}

// WriteSyncMDN writes AS2 MDN headers + body to the response.
func WriteSyncMDN(w http.ResponseWriter, ourAS2ID, partnerAS2ID string, original MessageHeaders, mic string, disposition MDNDisposition, humanText string) error {
	ct, body, err := BuildSyncMDN(original, mic, disposition, humanText)
	if err != nil {
		return err
	}
	w.Header().Set(HeaderAS2From, quoteAS2(ourAS2ID))
	w.Header().Set(HeaderAS2To, quoteAS2(partnerAS2ID))
	w.Header().Set(HeaderAS2Version, "1.2")
	w.Header().Set(HeaderMessageID, NewMessageID("mdn.pegasusx.local"))
	w.Header().Set("Content-Type", ct)
	w.WriteHeader(http.StatusOK)
	_, err = w.Write(body)
	return err
}

// ParsedMDN is the disposition-notification fields we verify on outbound send.
type ParsedMDN struct {
	Disposition       string
	ReceivedContentMIC string
	OriginalMessageID string
}

// ParseSyncMDN extracts Disposition + Received-Content-MIC from a sync MDN body.
func ParseSyncMDN(contentType string, body []byte) (ParsedMDN, error) {
	media, params, err := mime.ParseMediaType(strings.TrimSpace(contentType))
	if err != nil {
		// Some partners omit parameters on Content-Type; sniff multipart boundary.
		media = strings.ToLower(strings.TrimSpace(contentType))
		params = map[string]string{}
	}
	var out ParsedMDN
	if !strings.HasPrefix(media, "multipart/") {
		// Flat disposition-notification body (rare).
		parseDispositionNotification(string(body), &out)
		if out.Disposition == "" && out.ReceivedContentMIC == "" {
			return out, fmt.Errorf("as2_mdn_not_multipart")
		}
		return out, nil
	}
	boundary := params["boundary"]
	if boundary == "" {
		return out, fmt.Errorf("as2_mdn_boundary_missing")
	}
	r := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := r.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return out, fmt.Errorf("as2_mdn_part: %w", err)
		}
		ct := strings.ToLower(part.Header.Get("Content-Type"))
		raw, _ := io.ReadAll(io.LimitReader(part, 1<<20))
		_ = part.Close()
		if strings.Contains(ct, "disposition-notification") {
			parseDispositionNotification(string(raw), &out)
		}
	}
	if out.Disposition == "" {
		return out, fmt.Errorf("as2_mdn_disposition_missing")
	}
	return out, nil
}

func parseDispositionNotification(raw string, out *ParsedMDN) {
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		lower := strings.ToLower(line)
		switch {
		case strings.HasPrefix(lower, "disposition:"):
			out.Disposition = strings.TrimSpace(line[len("Disposition:"):])
		case strings.HasPrefix(lower, "received-content-mic:"):
			out.ReceivedContentMIC = strings.TrimSpace(line[len("Received-Content-MIC:"):])
		case strings.HasPrefix(lower, "original-message-id:"):
			out.OriginalMessageID = strings.TrimSpace(line[len("Original-Message-ID:"):])
		}
	}
}

// VerifyMDN fails closed unless disposition is processed and MIC matches expected.
func VerifyMDN(expectedMIC string, mdn ParsedMDN) error {
	disp := strings.ToLower(mdn.Disposition)
	if !strings.Contains(disp, "processed") || strings.Contains(disp, "failed") || strings.Contains(disp, "error") {
		return fmt.Errorf("as2_mdn_not_processed")
	}
	want := normalizeMIC(expectedMIC)
	got := normalizeMIC(mdn.ReceivedContentMIC)
	if want == "" || got == "" {
		return fmt.Errorf("as2_mdn_mic_missing")
	}
	if !strings.EqualFold(want, got) {
		return fmt.Errorf("as2_mdn_mic_mismatch")
	}
	return nil
}

func normalizeMIC(s string) string {
	s = strings.TrimSpace(s)
	// "base64…, sha-256" — compare digest + alg case-insensitively, collapse spaces.
	parts := strings.Split(s, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return strings.ToLower(strings.Join(parts, ", "))
}

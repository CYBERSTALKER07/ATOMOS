package as2

import (
	"bytes"
	"fmt"
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

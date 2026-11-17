package as2

import (
	"fmt"
	"net/http"
	"strings"
)

// Header names (RFC 4130).
const (
	HeaderAS2From                      = "AS2-From"
	HeaderAS2To                        = "AS2-To"
	HeaderAS2Version                   = "AS2-Version"
	HeaderMessageID                    = "Message-ID"
	HeaderDispositionNotificationTo    = "Disposition-Notification-To"
	HeaderDispositionNotificationOpts  = "Disposition-Notification-Options"
	HeaderReceiptDeliveryOption        = "Receipt-Delivery-Option"
	ContentTypeEDIFACT                 = "application/edifact"
	ContentTypePKCS7MIME               = "application/pkcs7-mime"
)

// MessageHeaders are the AS2 identity headers on a request/response.
type MessageHeaders struct {
	AS2From   string
	AS2To     string
	MessageID string
	Version   string
	MDNTo     string
}

// ParseHeaders extracts AS2 headers from an HTTP request.
func ParseHeaders(h http.Header) MessageHeaders {
	return MessageHeaders{
		AS2From:   NormalizeAS2ID(h.Get(HeaderAS2From)),
		AS2To:     NormalizeAS2ID(h.Get(HeaderAS2To)),
		MessageID: strings.TrimSpace(h.Get(HeaderMessageID)),
		Version:   strings.TrimSpace(h.Get(HeaderAS2Version)),
		MDNTo:     strings.TrimSpace(h.Get(HeaderDispositionNotificationTo)),
	}
}

// ApplyOutbound sets AS2 headers on an outbound request.
func ApplyOutbound(h http.Header, from, to, messageID string, requestMDN bool) {
	h.Set(HeaderAS2From, quoteAS2(from))
	h.Set(HeaderAS2To, quoteAS2(to))
	h.Set(HeaderAS2Version, "1.2")
	h.Set(HeaderMessageID, messageID)
	if requestMDN {
		h.Set(HeaderDispositionNotificationTo, "pegasusx@localhost")
		h.Set(HeaderDispositionNotificationOpts, "signed-receipt-protocol=optional, pkcs7-signature; signed-receipt-micalg=optional, sha-256")
	}
}

func quoteAS2(id string) string {
	id = NormalizeAS2ID(id)
	if id == "" {
		return `""`
	}
	if strings.ContainsAny(id, ` "<>`) {
		return `"` + strings.ReplaceAll(id, `"`, "") + `"`
	}
	return id
}

// NewMessageID builds a Message-ID value.
func NewMessageID(domain string) string {
	if domain == "" {
		domain = "pegasusx.local"
	}
	return fmt.Sprintf("<%d@%s>", timeUnixNano(), domain)
}

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

// Payload is the unwrapped EDI content plus MIC of that content.
type Payload struct {
	Content     []byte
	MIC         string
	ContentType string
	Filename    string
}

// DetectContentType returns a MIME type for an AS2 body.
func DetectContentType(h http.Header) string {
	ct := h.Get("Content-Type")
	if ct == "" {
		return ""
	}
	media, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return strings.ToLower(ct)
	}
	return strings.ToLower(media)
}

// FilenameFromDisposition extracts filename= from Content-Disposition.
func FilenameFromDisposition(h http.Header) string {
	cd := h.Get("Content-Disposition")
	if cd == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(cd)
	if err != nil {
		return ""
	}
	return params["filename"]
}

// BuildPlainBody returns content-type and body for insecure plain AS2.
func BuildPlainBody(edi []byte, filename string) (contentType string, body []byte, hdr textproto.MIMEHeader) {
	if filename == "" {
		filename = "message.edi"
	}
	hdr = textproto.MIMEHeader{}
	hdr.Set("Content-Type", ContentTypeEDIFACT)
	hdr.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	return ContentTypeEDIFACT, edi, hdr
}

// BuildEncryptedBody wraps EDI as sign-then-encrypt PKCS7 MIME.
func BuildEncryptedBody(edi []byte, signer Material, recipientCertPEM []byte, filename string) (contentType string, body []byte, err error) {
	recip, err := LoadCertificatePEM(recipientCertPEM)
	if err != nil {
		return "", nil, err
	}
	enveloped, err := SignThenEncrypt(edi, signer, recip)
	if err != nil {
		return "", nil, err
	}
	if filename == "" {
		filename = "smime.p7m"
	}
	ct := ContentTypePKCS7MIME + `; smime-type=enveloped-data; name="` + filename + `"`
	return ct, enveloped, nil
}

// UnwrapInbound extracts EDI bytes from an AS2 HTTP body.
// When insecurePlain is true, raw body is treated as EDI if Content-Type is edifact/octet-stream/empty.
func UnwrapInbound(h http.Header, body []byte, our Material, partnerCertPEM []byte, insecurePlain bool) (Payload, error) {
	ct := DetectContentType(h)
	filename := FilenameFromDisposition(h)

	switch {
	case insecurePlain && (ct == "" || ct == ContentTypeEDIFACT || ct == "application/octet-stream" || ct == "text/plain"):
		return Payload{Content: body, MIC: MICSHA256(body), ContentType: ct, Filename: filename}, nil

	case strings.Contains(ct, "pkcs7-mime") || strings.Contains(ct, "x-pkcs7-mime"):
		if len(partnerCertPEM) == 0 {
			return Payload{}, fmt.Errorf("partner_cert_required")
		}
		partnerCert, err := LoadCertificatePEM(partnerCertPEM)
		if err != nil {
			return Payload{}, err
		}
		edi, err := DecryptThenVerify(body, our, partnerCert)
		if err != nil {
			return Payload{}, err
		}
		return Payload{Content: edi, MIC: MICSHA256(edi), ContentType: ct, Filename: filename}, nil

	case strings.HasPrefix(ct, "multipart/"):
		edi, err := extractMultipartEDI(ct, body)
		if err != nil {
			return Payload{}, err
		}
		return Payload{Content: edi, MIC: MICSHA256(edi), ContentType: ct, Filename: filename}, nil

	default:
		if insecurePlain {
			return Payload{Content: body, MIC: MICSHA256(body), ContentType: ct, Filename: filename}, nil
		}
		return Payload{}, fmt.Errorf("unsupported_content_type:%s", ct)
	}
}

func extractMultipartEDI(contentType string, body []byte) ([]byte, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("multipart_boundary_missing")
	}
	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		pct := part.Header.Get("Content-Type")
		data, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		media, _, _ := mime.ParseMediaType(pct)
		media = strings.ToLower(media)
		if media == ContentTypeEDIFACT || media == "application/edi-x12" || media == "application/octet-stream" || media == "text/plain" {
			return data, nil
		}
	}
	return nil, fmt.Errorf("multipart_edi_part_missing")
}

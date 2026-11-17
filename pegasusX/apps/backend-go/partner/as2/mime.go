package as2

import (
	"bytes"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
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

// BuildMultipartSignedBody wraps EDI as RFC 4130 multipart/signed with a detached SHA-256 PKCS7 signature.
func BuildMultipartSignedBody(edi []byte, signer Material, filename string) (contentType string, body []byte, err error) {
	if filename == "" {
		filename = "message.edi"
	}
	var rnd [12]byte
	if _, err := rand.Read(rnd[:]); err != nil {
		return "", nil, err
	}
	boundary := "=_as2_" + hex.EncodeToString(rnd[:])
	signedEntity := []byte(fmt.Sprintf(
		"Content-Type: %s\r\nContent-Disposition: attachment; filename=%q\r\n\r\n",
		ContentTypeEDIFACT, filename,
	))
	signedEntity = append(signedEntity, edi...)

	sigDER, err := SignDetached(signedEntity, signer)
	if err != nil {
		return "", nil, err
	}
	var out bytes.Buffer
	out.WriteString("--" + boundary + "\r\n")
	out.Write(signedEntity)
	out.WriteString("\r\n--" + boundary + "\r\n")
	out.WriteString("Content-Type: application/pkcs7-signature; name=\"smime.p7s\"\r\n")
	out.WriteString("Content-Transfer-Encoding: base64\r\n")
	out.WriteString("Content-Disposition: attachment; filename=\"smime.p7s\"\r\n\r\n")
	out.WriteString(base64.StdEncoding.EncodeToString(sigDER))
	out.WriteString("\r\n--" + boundary + "--\r\n")

	ct := fmt.Sprintf(`multipart/signed; protocol="application/pkcs7-signature"; micalg=sha-256; boundary="%s"`, boundary)
	return ct, out.Bytes(), nil
}

// UnwrapInbound extracts EDI bytes from an AS2 HTTP body.
// When insecurePlain is true, raw body is treated as EDI if Content-Type is edifact/octet-stream/empty.
func UnwrapInbound(h http.Header, body []byte, our Material, partnerCertPEM []byte, insecurePlain bool) (Payload, error) {
	rawCT := h.Get("Content-Type")
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

	case ct == "multipart/signed":
		if len(partnerCertPEM) == 0 {
			return Payload{}, fmt.Errorf("partner_cert_required")
		}
		partnerCert, err := LoadCertificatePEM(partnerCertPEM)
		if err != nil {
			return Payload{}, err
		}
		edi, err := verifyAndExtractMultipartSigned(rawCT, body, partnerCert)
		if err != nil {
			return Payload{}, err
		}
		return Payload{Content: edi, MIC: MICSHA256(edi), ContentType: ct, Filename: filename}, nil

	case strings.HasPrefix(ct, "multipart/"):
		if !insecurePlain {
			return Payload{}, fmt.Errorf("as2_signature_required")
		}
		edi, err := extractMultipartEDI(rawCT, body)
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

func verifyAndExtractMultipartSigned(contentType string, body []byte, partnerCert *x509.Certificate) ([]byte, error) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, err
	}
	boundary := params["boundary"]
	if boundary == "" {
		return nil, fmt.Errorf("multipart_boundary_missing")
	}
	signedEntityRaw, err := extractRawFirstMultipartEntity(body, boundary)
	if err != nil {
		return nil, err
	}

	mr := multipart.NewReader(bytes.NewReader(body), boundary)
	var ediData []byte
	var sigData []byte
	partCount := 0
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		partCount++
		pct := part.Header.Get("Content-Type")
		cte := strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Transfer-Encoding")))
		data, err := io.ReadAll(part)
		if err != nil {
			return nil, err
		}
		media, _, _ := mime.ParseMediaType(pct)
		media = strings.ToLower(media)
		switch partCount {
		case 1:
			if !isAcceptedEDIMediaType(media) {
				return nil, fmt.Errorf("multipart_edi_part_missing")
			}
			ediData = data
		case 2:
			if media != "application/pkcs7-signature" && media != "application/x-pkcs7-signature" {
				return nil, fmt.Errorf("as2_signature_required")
			}
			if cte == "base64" {
				clean := strings.Map(func(r rune) rune {
					if r == '\r' || r == '\n' || r == ' ' || r == '\t' {
						return -1
					}
					return r
				}, string(data))
				dec, decErr := base64.StdEncoding.DecodeString(clean)
				if decErr != nil {
					return nil, fmt.Errorf("invalid_pkcs7_signature: %w", decErr)
				}
				sigData = dec
			} else {
				sigData = data
			}
		default:
			return nil, fmt.Errorf("as2_invalid_multipart_signed")
		}
	}
	if partCount != 2 || len(ediData) == 0 || len(sigData) == 0 {
		return nil, fmt.Errorf("as2_signature_required")
	}
	if err := VerifyDetached(signedEntityRaw, sigData, partnerCert); err != nil {
		return nil, fmt.Errorf("as2_signature_invalid: %w", err)
	}
	return ediData, nil
}

func extractRawFirstMultipartEntity(body []byte, boundary string) ([]byte, error) {
	openCRLF := []byte("--" + boundary + "\r\n")
	openLF := []byte("--" + boundary + "\n")
	start := bytes.Index(body, openCRLF)
	delimLen := len(openCRLF)
	if start < 0 {
		start = bytes.Index(body, openLF)
		delimLen = len(openLF)
	}
	if start < 0 {
		return nil, fmt.Errorf("multipart_boundary_missing")
	}
	rest := body[start+delimLen:]
	nextCRLF := []byte("\r\n--" + boundary)
	nextLF := []byte("\n--" + boundary)
	end := bytes.Index(rest, nextCRLF)
	if end < 0 {
		end = bytes.Index(rest, nextLF)
	}
	if end < 0 {
		return nil, fmt.Errorf("multipart_boundary_missing")
	}
	return rest[:end], nil
}

func isAcceptedEDIMediaType(media string) bool {
	return media == ContentTypeEDIFACT || media == "application/edi-x12" || media == "application/octet-stream" || media == "text/plain"
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
		if isAcceptedEDIMediaType(media) {
			return data, nil
		}
	}
	return nil, fmt.Errorf("multipart_edi_part_missing")
}

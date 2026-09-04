package orgoidc

import (
	"crypto"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type idClaims struct {
	Iss   string `json:"iss"`
	Aud   any    `json:"aud"`
	Exp   int64  `json:"exp"`
	Nbf   int64  `json:"nbf"`
	Sub   string `json:"sub"`
	Email string `json:"email"`
	Nonce string `json:"nonce"`
}

func (c idClaims) audiences() []string {
	switch v := c.Aud.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return nil
		}
		return []string{v}
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, _ := item.(string)
			if strings.TrimSpace(s) != "" {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// ExtractKID extracts the Key ID from the JWT header.
func ExtractKID(idToken string) string {
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return ""
	}
	headerJSON, err := b64JSON(parts[0])
	if err != nil {
		return ""
	}
	var header struct {
		KID string `json:"kid"`
	}
	_ = json.Unmarshal(headerJSON, &header)
	return header.KID
}

// VerifyIDToken checks RS256 signature + iss/aud/exp. key must be the IdP public key.
func VerifyIDToken(idToken string, cfg Config, key *rsa.PublicKey, now time.Time, nonce string) (subject, email string, err error) {
	if key == nil {
		return "", "", fmt.Errorf("%w: missing key", ErrInvalidToken)
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	parts := strings.Split(idToken, ".")
	if len(parts) != 3 {
		return "", "", ErrInvalidToken
	}
	headerJSON, err := b64JSON(parts[0])
	if err != nil {
		return "", "", ErrInvalidToken
	}
	var header struct {
		Alg string `json:"alg"`
	}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return "", "", ErrInvalidToken
	}
	if !strings.EqualFold(header.Alg, "RS256") {
		return "", "", fmt.Errorf("%w: alg", ErrInvalidToken)
	}
	payloadJSON, err := b64JSON(parts[1])
	if err != nil {
		return "", "", ErrInvalidToken
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return "", "", ErrInvalidToken
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	if err := rsa.VerifyPKCS1v15(key, crypto.SHA256, sum[:], sig); err != nil {
		return "", "", fmt.Errorf("%w: sig", ErrInvalidToken)
	}
	var claims idClaims
	if err := json.Unmarshal(payloadJSON, &claims); err != nil {
		return "", "", ErrInvalidToken
	}
	if strings.TrimRight(strings.TrimSpace(claims.Iss), "/") != cfg.Issuer {
		return "", "", ErrIssuerMismatch
	}
	wantAud := cfg.audience()
	okAud := false
	for _, a := range claims.audiences() {
		if a == wantAud {
			okAud = true
			break
		}
	}
	if !okAud {
		return "", "", ErrAudienceMismatch
	}
	if claims.Exp <= now.Unix() {
		return "", "", fmt.Errorf("%w: expired", ErrInvalidToken)
	}
	if claims.Nbf > 0 && claims.Nbf > now.Unix()+60 {
		return "", "", fmt.Errorf("%w: nbf", ErrInvalidToken)
	}
	if n := strings.TrimSpace(nonce); n != "" && n != strings.TrimSpace(claims.Nonce) {
		return "", "", ErrNonceMismatch
	}
	sub := strings.TrimSpace(claims.Email)
	if sub == "" {
		sub = strings.TrimSpace(claims.Sub)
	}
	if sub == "" {
		return "", "", ErrMissingSubject
	}
	return sub, strings.TrimSpace(claims.Email), nil
}

func b64JSON(seg string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(seg)
}

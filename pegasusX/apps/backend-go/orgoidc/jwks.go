package orgoidc

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

type jwksDoc struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksCacheEntry struct {
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

var (
	jwksCache   sync.Map
	jwksCacheMu sync.Mutex
)

// FetchJWKS loads the RSA signing key matching the kid from {issuer}/.well-known/jwks.json.
func FetchJWKS(ctx context.Context, issuer, kid string) (*rsa.PublicKey, error) {
	now := time.Now()
	if entry, ok := jwksCache.Load(issuer); ok {
		cached := entry.(*jwksCacheEntry)
		if now.Before(cached.expiresAt) {
			if key, found := cached.keys[kid]; found {
				return key, nil
			}
		}
	}

	jwksCacheMu.Lock()
	defer jwksCacheMu.Unlock()

	// Double-check cache inside lock
	if entry, ok := jwksCache.Load(issuer); ok {
		cached := entry.(*jwksCacheEntry)
		if now.Before(cached.expiresAt) {
			if key, found := cached.keys[kid]; found {
				return key, nil
			}
		}
	}

	u := strings.TrimRight(strings.TrimSpace(issuer), "/") + "/.well-known/jwks.json"
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("oidc jwks: status %d", resp.StatusCode)
	}
	var doc jwksDoc
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return nil, err
	}

	keys := make(map[string]*rsa.PublicKey)
	for _, k := range doc.Keys {
		if !strings.EqualFold(k.Kty, "RSA") {
			continue
		}
		if k.Use != "" && !strings.EqualFold(k.Use, "sig") {
			continue
		}
		if pub, err := rsaPublicFromJWK(k.N, k.E); err == nil {
			keys[k.Kid] = pub
		}
	}

	jwksCache.Store(issuer, &jwksCacheEntry{
		keys:      keys,
		expiresAt: now.Add(15 * time.Minute), // Cache for 15 minutes
	})

	if key, found := keys[kid]; found {
		return key, nil
	}

	return nil, fmt.Errorf("oidc jwks: no rsa key matched kid %s", kid)
}

func rsaPublicFromJWK(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	var e int
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	if e == 0 {
		return nil, fmt.Errorf("oidc: bad jwk e")
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

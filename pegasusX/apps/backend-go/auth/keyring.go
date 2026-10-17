package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"strings"
	"sync"
)

// Keyring defines the contract for signing and multi-key verification.
type Keyring interface {
	// CurrentKey returns the active signing key and its key ID (if any).
	CurrentKey() (secret string, kid string)
	// VerifyCandidateKeys returns candidate secrets to verify a token against.
	// If kid is non-empty, it returns the key for that kid if known.
	// If kid is empty or unknown, it returns all candidate keys (current, then fallbacks).
	VerifyCandidateKeys(kid string) []string
	// HasKeys reports whether the keyring contains at least one non-empty secret.
	HasKeys() bool
}

// MultiKeyring implements Keyring with thread-safe rotation and candidate fallback.
type MultiKeyring struct {
	mu         sync.RWMutex
	currentKID string
	currentKey string
	keysByID   map[string]string
	fallbacks  []string
}

// NewKeyring creates a Keyring with a primary secret and optional fallback secrets.
func NewKeyring(primarySecret string, fallbackSecrets ...string) *MultiKeyring {
	primarySecret = strings.TrimSpace(primarySecret)
	kr := &MultiKeyring{
		currentKey: primarySecret,
		keysByID:   make(map[string]string),
	}
	if primarySecret != "" {
		kr.currentKID = deriveKeyID(primarySecret)
		kr.keysByID[kr.currentKID] = primarySecret
	}
	for _, fb := range fallbackSecrets {
		fb = strings.TrimSpace(fb)
		if fb != "" && fb != primarySecret {
			kid := deriveKeyID(fb)
			kr.keysByID[kid] = fb
			kr.fallbacks = append(kr.fallbacks, fb)
		}
	}
	return kr
}

// NewKeyringWithKID creates a Keyring with an explicit Key ID for the primary secret.
func NewKeyringWithKID(primarySecret, primaryKID string, fallbackSecrets ...string) *MultiKeyring {
	kr := NewKeyring(primarySecret, fallbackSecrets...)
	if primaryKID != "" && kr.currentKey != "" {
		kr.currentKID = primaryKID
		kr.keysByID[primaryKID] = kr.currentKey
	}
	return kr
}

// NewKeyringFromEnv initializes a Keyring from environment variables:
// - JWT_SECRET_CURRENT (or fallback JWT_SECRET)
// - JWT_KEY_ID (optional)
// - JWT_SECRET_PREVIOUS (optional)
// - JWT_SECRETS (optional comma-separated list of [kid:]secret)
func NewKeyringFromEnv() *MultiKeyring {
	current := strings.TrimSpace(os.Getenv("JWT_SECRET_CURRENT"))
	if current == "" {
		current = strings.TrimSpace(os.Getenv("JWT_SECRET"))
	}
	kid := strings.TrimSpace(os.Getenv("JWT_KEY_ID"))
	prev := strings.TrimSpace(os.Getenv("JWT_SECRET_PREVIOUS"))
	csv := strings.TrimSpace(os.Getenv("JWT_SECRETS"))

	var fallbacks []string
	if prev != "" && prev != current {
		fallbacks = append(fallbacks, prev)
	}
	if csv != "" {
		for _, part := range strings.Split(csv, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			if strings.Contains(part, ":") {
				sub := strings.SplitN(part, ":", 2)
				fallbacks = append(fallbacks, strings.TrimSpace(sub[1]))
			} else {
				fallbacks = append(fallbacks, part)
			}
		}
	}

	if kid != "" {
		return NewKeyringWithKID(current, kid, fallbacks...)
	}
	return NewKeyring(current, fallbacks...)
}

// CurrentKey returns the active primary secret and its key ID.
func (kr *MultiKeyring) CurrentKey() (secret string, kid string) {
	if kr == nil {
		return "", ""
	}
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.currentKey, kr.currentKID
}

// VerifyCandidateKeys returns matching secrets based on key ID, or all candidates.
func (kr *MultiKeyring) VerifyCandidateKeys(kid string) []string {
	if kr == nil {
		return nil
	}
	kr.mu.RLock()
	defer kr.mu.RUnlock()

	kid = strings.TrimSpace(kid)
	if kid != "" {
		if secret, exists := kr.keysByID[kid]; exists {
			return []string{secret}
		}
	}

	// If kid is empty or not found by exact ID, search all active keys starting with current
	seen := make(map[string]struct{}, len(kr.fallbacks)+1)
	var candidates []string
	if kr.currentKey != "" {
		candidates = append(candidates, kr.currentKey)
		seen[kr.currentKey] = struct{}{}
	}
	for _, fb := range kr.fallbacks {
		if _, exists := seen[fb]; !exists && fb != "" {
			candidates = append(candidates, fb)
			seen[fb] = struct{}{}
		}
	}
	return candidates
}

// HasKeys reports whether the keyring contains at least one secret.
func (kr *MultiKeyring) HasKeys() bool {
	if kr == nil {
		return false
	}
	kr.mu.RLock()
	defer kr.mu.RUnlock()
	return kr.currentKey != "" || len(kr.fallbacks) > 0
}

// Rotate switches the primary signing key to newSecret and moves the old key to fallbacks.
func (kr *MultiKeyring) Rotate(newSecret, newKID string) {
	if kr == nil {
		return
	}
	newSecret = strings.TrimSpace(newSecret)
	if newSecret == "" {
		return
	}
	kr.mu.Lock()
	defer kr.mu.Unlock()

	// Demote old key to fallback if non-empty
	if kr.currentKey != "" && kr.currentKey != newSecret {
		kr.fallbacks = append([]string{kr.currentKey}, kr.fallbacks...)
	}

	kr.currentKey = newSecret
	if newKID != "" {
		kr.currentKID = newKID
	} else {
		kr.currentKID = deriveKeyID(newSecret)
	}
	kr.keysByID[kr.currentKID] = newSecret
}

// singleKeyring implements Keyring for a single static secret.
type singleKeyring string

func (s singleKeyring) CurrentKey() (string, string) {
	str := string(s)
	return str, deriveKeyID(str)
}

func (s singleKeyring) VerifyCandidateKeys(_ string) []string {
	str := string(s)
	if str == "" {
		return nil
	}
	return []string{str}
}

func (s singleKeyring) HasKeys() bool {
	return string(s) != ""
}

func deriveKeyID(secret string) string {
	h := sha256.Sum256([]byte(secret))
	// 8-byte hex prefix for compact kid
	return hex.EncodeToString(h[:8])
}

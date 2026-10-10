package idempotency

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// ScopeKey binds an idempotency key to principal + route so callers cannot
// replay another actor's cached response by reusing the raw header value.
func ScopeKey(principalID, routePattern, rawKey string) string {
	principalID = strings.TrimSpace(principalID)
	routePattern = strings.TrimSpace(routePattern)
	rawKey = strings.TrimSpace(rawKey)
	if rawKey == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(principalID + "|" + routePattern + "|" + rawKey))
	return hex.EncodeToString(sum[:])
}

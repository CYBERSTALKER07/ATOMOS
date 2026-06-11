package handoff

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strings"

	"github.com/google/uuid"
)

// Config controls token generation and backward-compatible dev behaviour.
type Config struct {
	// LegacyOrderIDFallback accepts order_id as the delivery token when no
	// persisted DeliveryToken exists. Defaults to true for local docker sims.
	LegacyOrderIDFallback bool
	Mint                  func() string
}

// Engine mints and validates per-order delivery handoff tokens.
type Engine struct {
	legacyOrderIDFallback bool
	mint                  func() string
}

// New constructs an Engine from Config.
func New(cfg Config) *Engine {
	mint := cfg.Mint
	if mint == nil {
		mint = func() string { return uuid.NewString() }
	}
	return &Engine{
		legacyOrderIDFallback: cfg.LegacyOrderIDFallback,
		mint:                  mint,
	}
}

// FromEnv builds the process-default engine. HANDOFF_LEGACY_ORDER_ID_FALLBACK
// defaults to true so existing docker simulations keep working without migration
// backfill for rows that predate DeliveryToken.
func FromEnv() *Engine {
	return New(Config{
		LegacyOrderIDFallback: envBool("HANDOFF_LEGACY_ORDER_ID_FALLBACK", true),
	})
}

// StatusesWithPublicToken lists order states where retailers may show a QR code.
func StatusesWithPublicToken() []string {
	return []string{
		"LOADED",
		"IN_TRANSIT",
		"ARRIVED",
		"ARRIVING",
		"AWAITING_PAYMENT",
		"PENDING_CASH_COLLECTION",
	}
}

// ShouldMintOnTransition reports whether entering nextStatus requires a token.
func (e *Engine) ShouldMintOnTransition(previousStatus, nextStatus string) bool {
	prev := strings.TrimSpace(previousStatus)
	next := strings.TrimSpace(nextStatus)
	if next == "LOADED" && prev != "LOADED" {
		return true
	}
	return false
}

// ShouldClearOnTransition reports terminal states that invalidate the token.
func (e *Engine) ShouldClearOnTransition(nextStatus string) bool {
	switch strings.TrimSpace(nextStatus) {
	case "COMPLETED", "CANCELLED":
		return true
	default:
		return false
	}
}

// ShouldRotateOnReassign reports whether driver reassignment must rotate token.
func (e *Engine) ShouldRotateOnReassign(previousDriverID, nextDriverID string, storedToken string) bool {
	if strings.TrimSpace(storedToken) == "" {
		return false
	}
	prev := strings.TrimSpace(previousDriverID)
	next := strings.TrimSpace(nextDriverID)
	return prev != "" && next != "" && prev != next
}

// Mint returns a new opaque delivery token.
func (e *Engine) Mint() string {
	return strings.TrimSpace(e.mint())
}

// ApplyTransition mutates storedToken in place for a lifecycle or reassignment change.
func (e *Engine) ApplyTransition(storedToken *string, previousStatus, nextStatus, previousDriverID, nextDriverID string) {
	if storedToken == nil {
		return
	}
	if e.ShouldClearOnTransition(nextStatus) {
		*storedToken = ""
		return
	}
	if e.ShouldRotateOnReassign(previousDriverID, nextDriverID, *storedToken) {
		*storedToken = e.Mint()
		return
	}
	if e.ShouldMintOnTransition(previousStatus, nextStatus) && strings.TrimSpace(*storedToken) == "" {
		*storedToken = e.Mint()
	}
}

// PublicToken resolves the token retailers/drivers should encode in QR payloads.
func (e *Engine) PublicToken(orderID, storedToken, status string) string {
	if !e.statusExposesToken(status) {
		return ""
	}
	stored := strings.TrimSpace(storedToken)
	if stored != "" {
		return stored
	}
	if e.legacyOrderIDFallback {
		return strings.TrimSpace(orderID)
	}
	return ""
}

// ExpectedToken resolves the canonical token for server-side validation.
func (e *Engine) ExpectedToken(orderID, storedToken string) string {
	stored := strings.TrimSpace(storedToken)
	if stored != "" {
		return stored
	}
	if e.legacyOrderIDFallback {
		return strings.TrimSpace(orderID)
	}
	return ""
}

// Validate checks a scanned token against the persisted order token.
func (e *Engine) Validate(orderID, storedToken, presented string) error {
	expected := e.ExpectedToken(orderID, storedToken)
	if expected == "" {
		return errors.New("delivery token not active")
	}
	if strings.TrimSpace(presented) != expected {
		return errors.New("invalid qr token")
	}
	return nil
}

// HashToken returns SHA-256 hex of the raw token for offline manifest hashes.
func (e *Engine) HashToken(token string) string {
	trimmed := strings.TrimSpace(token)
	if trimmed == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(trimmed))
	return hex.EncodeToString(sum[:])
}

func (e *Engine) statusExposesToken(status string) bool {
	normalized := strings.TrimSpace(status)
	for _, allowed := range StatusesWithPublicToken() {
		if normalized == allowed {
			return true
		}
	}
	return false
}

func envBool(key string, fallback bool) bool {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	switch strings.ToLower(raw) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

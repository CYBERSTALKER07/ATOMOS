package retailer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

var (
	ErrTradingPartnerRequired = errors.New("trading_partner_required")
	ErrSeedAttachForbidden    = errors.New("seed_attach_forbidden")
	ErrUnknownTradingPartner  = errors.New("unknown_trading_partner")
	ErrInviteInvalid          = errors.New("invite_invalid")
	ErrInviteExpired          = errors.New("invite_expired")
	ErrInviteSecretMissing    = errors.New("invite_secret_missing")
)

// TradingPartnerLookup reports whether a supplier id exists.
type TradingPartnerLookup func(ctx context.Context, supplierID string) (bool, error)

// SetTradingPartnerLookup wires GS-T4 existence checks (bootstrap: supplier profile).
func (s *Service) SetTradingPartnerLookup(fn TradingPartnerLookup) {
	if s != nil {
		s.partners = fn
	}
}

func retailerDemoLoginAllowed() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "ssmr")
}

func retailerSeedAttachAllowed() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("PEGASUSX_ENV")), "ssmr")
}

func demoRetailerSecret() string {
	if !retailerDemoLoginAllowed() {
		return ""
	}
	expect := strings.TrimSpace(os.Getenv("RETAILER_DEMO_PASSWORD"))
	if expect == "" {
		expect = strings.TrimSpace(os.Getenv("RETAILER_DEMO_PIN"))
	}
	if expect == "" {
		expect = "1234"
	}
	return expect
}

func (s *Service) resolveTradingPartner(ctx context.Context, supplierID, inviteToken string) (string, error) {
	if tok := strings.TrimSpace(inviteToken); tok != "" {
		sid, err := ParseTradingPartnerInvite(s.jwtSecret, tok, s.now())
		if err != nil {
			return "", err
		}
		return s.guardTradingPartner(ctx, sid)
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return "", ErrTradingPartnerRequired
	}
	return s.guardTradingPartner(ctx, sid)
}

func (s *Service) guardTradingPartner(ctx context.Context, supplierID string) (string, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return "", ErrTradingPartnerRequired
	}
	if seed := strings.TrimSpace(s.seedSupplierID); seed != "" && sid == seed && !retailerSeedAttachAllowed() {
		return "", ErrSeedAttachForbidden
	}
	if s.partners != nil {
		ok, err := s.partners(ctx, sid)
		if err != nil {
			return "", fmt.Errorf("lookup trading partner: %w", err)
		}
		if !ok {
			return "", ErrUnknownTradingPartner
		}
	}
	return sid, nil
}

// MintTradingPartnerInvite signs a time-limited attach token for one supplier.
func MintTradingPartnerInvite(secret, supplierID string, ttl time.Duration, now time.Time) (string, time.Time, error) {
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return "", time.Time{}, ErrTradingPartnerRequired
	}
	if strings.TrimSpace(secret) == "" {
		return "", time.Time{}, ErrInviteSecretMissing
	}
	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	exp := now.Add(ttl)
	payload := sid + "|" + strconv.FormatInt(exp.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token, exp, nil
}

// ParseTradingPartnerInvite verifies an invite and returns the supplier id.
func ParseTradingPartnerInvite(secret, token string, now time.Time) (string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", ErrInviteSecretMissing
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return "", ErrInviteInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", ErrInviteInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", ErrInviteInvalid
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", ErrInviteInvalid
	}
	payload := string(raw)
	cut := strings.LastIndex(payload, "|")
	if cut <= 0 {
		return "", ErrInviteInvalid
	}
	sid := strings.TrimSpace(payload[:cut])
	expUnix, err := strconv.ParseInt(payload[cut+1:], 10, 64)
	if err != nil || sid == "" {
		return "", ErrInviteInvalid
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !now.Before(time.Unix(expUnix, 0).UTC()) {
		return "", ErrInviteExpired
	}
	return sid, nil
}

// HandleCreateTradingPartnerInvite is POST /v1/supplier/retailer-invites (ADMIN).
func (s *Service) HandleCreateTradingPartnerInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	sid := strings.TrimSpace(claims.SupplierID)
	if sid == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": ErrTradingPartnerRequired.Error()})
		return
	}
	if _, err := s.guardTradingPartner(r.Context(), sid); err != nil {
		writeAttachError(w, err)
		return
	}
	token, exp, err := MintTradingPartnerInvite(s.jwtSecret, sid, 7*24*time.Hour, s.now())
	if err != nil {
		writeAttachError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"supplier_id": sid,
		"token":       token,
		"expires_at":  exp.UTC().Format(time.RFC3339),
	})
}

func writeAttachError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrTradingPartnerRequired), errors.Is(err, ErrInviteInvalid),
		errors.Is(err, ErrInviteExpired):
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrSeedAttachForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrUnknownTradingPartner):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
	case errors.Is(err, ErrInviteSecretMissing):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

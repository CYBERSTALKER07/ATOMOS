package retailer

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/entityresolution"
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
	return auth.IsSandbox()
}

func retailerSeedAttachAllowed() bool {
	return auth.IsSandbox()
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
	sid, _, err := s.resolveTradingPartnerPack(ctx, supplierID, inviteToken)
	return sid, err
}

func (s *Service) resolveTradingPartnerPack(ctx context.Context, supplierID, inviteToken string) (string, string, error) {
	if tok := strings.TrimSpace(inviteToken); tok != "" {
		sid, market, err := ParseTradingPartnerInvite(s.jwtSecret, tok, s.now())
		if err != nil {
			return "", "", err
		}
		sid, err = s.guardTradingPartner(ctx, sid)
		if err != nil {
			return "", "", err
		}
		return sid, market, nil
	}
	sid := strings.TrimSpace(supplierID)
	if sid == "" {
		return "", "", ErrTradingPartnerRequired
	}
	sid, err := s.guardTradingPartner(ctx, sid)
	if err != nil {
		return "", "", err
	}
	return sid, "", nil
}

func assertAttachMarket(pack auth.MarketPack, requestedCountry, inviteMarket string) error {
	packCountry, err := auth.PackCountryCode(pack)
	if err != nil {
		return err
	}
	if inviteMarket != "" && auth.NormalizeMarketCode(inviteMarket) != pack.Code {
		return auth.ErrCrossMarketDeferred
	}
	return auth.AssertSameMarket(packCountry, requestedCountry)
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
// Payload is supplierID|unixExp[|marketCode]. Old two-field tokens still parse.
func MintTradingPartnerInvite(secret, supplierID, marketCode string, ttl time.Duration, now time.Time) (string, time.Time, error) {
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
	if market := auth.NormalizeMarketCode(marketCode); market != "" {
		payload += "|" + market
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	token := base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return token, exp, nil
}

// ParseTradingPartnerInvite verifies an invite and returns supplier id + pack code.
func ParseTradingPartnerInvite(secret, token string, now time.Time) (string, string, error) {
	if strings.TrimSpace(secret) == "" {
		return "", "", ErrInviteSecretMissing
	}
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return "", "", ErrInviteInvalid
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", "", ErrInviteInvalid
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", "", ErrInviteInvalid
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	if !hmac.Equal(sig, mac.Sum(nil)) {
		return "", "", ErrInviteInvalid
	}
	fields := strings.Split(string(raw), "|")
	if len(fields) < 2 {
		return "", "", ErrInviteInvalid
	}
	sid := strings.TrimSpace(fields[0])
	expUnix, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || sid == "" {
		return "", "", ErrInviteInvalid
	}
	market := ""
	if len(fields) >= 3 {
		market = auth.NormalizeMarketCode(fields[2])
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if !now.Before(time.Unix(expUnix, 0).UTC()) {
		return "", "", ErrInviteExpired
	}
	return sid, market, nil
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
	pack, err := auth.FiscalPackForSupplier(sid)
	if err != nil {
		writeAttachError(w, err)
		return
	}
	token, exp, err := MintTradingPartnerInvite(s.jwtSecret, sid, pack.Code, 7*24*time.Hour, s.now())
	if err != nil {
		writeAttachError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"supplier_id": sid,
		"market_code": pack.Code,
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
	case errors.Is(err, ErrUnknownTradingPartner),
		errors.Is(err, auth.ErrMarketPackUnknown), errors.Is(err, auth.ErrMarketPackNotShipped):
		st, code := auth.CheckoutPackHTTPStatus(err)
		if errors.Is(err, ErrUnknownTradingPartner) {
			st, code = http.StatusNotFound, err.Error()
		}
		writeJSON(w, st, map[string]string{"error": code})
	case errors.Is(err, auth.ErrCrossMarketDeferred), errors.Is(err, auth.ErrGeographyIncomplete):
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
	case errors.Is(err, ErrInviteSecretMissing):
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": err.Error()})
	default:
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
	}
}

// RetailerLinkInput holds identity fields recorded into SupplierRetailerLinks upon attach/invite redemption.
type RetailerLinkInput struct {
	SupplierID string
	RetailerID string
	TaxIDTIN   string
	Phone      string
	LegalName  string
}

// NormalizeTIN normalizes a tax identifier (INN/STIR/PINFL) using the canonical entityresolution rules.
func NormalizeTIN(raw string) string {
	return entityresolution.NormalizeTIN(raw)
}

// NormalizeUZPhone normalizes a Uzbekistan phone number (+998...) using the canonical entityresolution rules.
func NormalizeUZPhone(raw string) string {
	return entityresolution.NormalizeUZPhone(raw)
}

// UpsertSupplierRetailerLink writes a tenant-scoped SupplierRetailerLinks row and flags exact TIN/phone duplicates within SupplierId.
func UpsertSupplierRetailerLink(ctx context.Context, client *spanner.Client, in RetailerLinkInput, now time.Time) (entityresolution.SupplierRetailerLink, []entityresolution.SupplierRetailerLink, error) {
	return entityresolution.UpsertSupplierRetailerLink(ctx, client, entityresolution.SupplierRetailerLink{
		SupplierID:      strings.TrimSpace(in.SupplierID),
		RetailerID:      strings.TrimSpace(in.RetailerID),
		TaxIDTIN:        in.TaxIDTIN,
		NormalizedPhone: in.Phone,
		LegalName:       in.LegalName,
	}, now)
}

// SyncSupplierRetailerLink normalizes and persists a retailer attach link into SupplierRetailerLinks when Spanner is configured.
func (s *Service) SyncSupplierRetailerLink(ctx context.Context, in RetailerLinkInput) (entityresolution.SupplierRetailerLink, []entityresolution.SupplierRetailerLink, error) {
	sid := strings.TrimSpace(in.SupplierID)
	rid := strings.TrimSpace(in.RetailerID)
	if sid == "" || rid == "" {
		return entityresolution.SupplierRetailerLink{}, nil, entityresolution.ErrInvalidRetailerLinkInput
	}
	now := time.Now().UTC()
	if s != nil && s.now != nil {
		now = s.now().UTC()
	}
	link := entityresolution.SupplierRetailerLink{
		SupplierID:      sid,
		RetailerID:      rid,
		TaxIDTIN:        in.TaxIDTIN,
		NormalizedPhone: in.Phone,
		LegalName:       in.LegalName,
	}
	if s != nil && s.spannerClient != nil {
		return entityresolution.UpsertSupplierRetailerLink(ctx, s.spannerClient, link, now)
	}
	saved, peers := entityresolution.ApplyRetailerLinkUpsert(now, link, nil, nil, nil)
	return saved, peers, nil
}

// HandleAcceptTradingPartnerInvite binds the calling retailer to a supplier using an invite token or explicit supplier_id
// and records normalized TIN/phone identity into SupplierRetailerLinks.
func (s *Service) HandleAcceptTradingPartnerInvite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleRetailer {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	retailerID := strings.TrimSpace(claims.Subject)
	if retailerID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "retailer_id_required"})
		return
	}
	var body struct {
		SupplierID  string `json:"supplier_id"`
		InviteToken string `json:"invite_token"`
		TaxIDTIN    string `json:"tax_id_tin"`
		TIN         string `json:"tin"`
		Phone       string `json:"phone"`
		LegalName   string `json:"legal_name"`
		CountryCode string `json:"country_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json_body"})
		return
	}
	partnerID, inviteMarket, err := s.resolveTradingPartnerPack(r.Context(), body.SupplierID, body.InviteToken)
	if err != nil {
		writeAttachError(w, err)
		return
	}
	pack, err := auth.FiscalPackForSupplier(partnerID)
	if err != nil {
		writeAttachError(w, err)
		return
	}
	if err := assertAttachMarket(pack, body.CountryCode, inviteMarket); err != nil {
		writeAttachError(w, err)
		return
	}
	tin := strings.TrimSpace(body.TaxIDTIN)
	if tin == "" {
		tin = strings.TrimSpace(body.TIN)
	}
	phone := strings.TrimSpace(body.Phone)
	legalName := strings.TrimSpace(body.LegalName)
	if phone == "" {
		phone = strings.TrimSpace(claims.PhoneNumber)
	}
	if (phone == "" || legalName == "") && s.repo != nil {
		if ret, found, lookupErr := s.repo.GetRetailer(r.Context(), retailerID); lookupErr == nil && found {
			if phone == "" {
				phone = ret.Phone
			}
			if legalName == "" {
				legalName = ret.Name
			}
		}
	}
	saved, peers, err := s.SyncSupplierRetailerLink(r.Context(), RetailerLinkInput{
		SupplierID: partnerID,
		RetailerID: retailerID,
		TaxIDTIN:   tin,
		Phone:      phone,
		LegalName:  legalName,
	})
	if err != nil {
		writeAttachError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":          "attached",
		"supplier_id":     partnerID,
		"retailer_id":     retailerID,
		"link":            saved,
		"duplicate_peers": peers,
	})
}

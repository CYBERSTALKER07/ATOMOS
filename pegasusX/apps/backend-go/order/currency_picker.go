package order

import (
	"context"
	"errors"
	"net/http"
	"os"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/fxrates"
)

var (
	// ErrCurrencyNotAllowed is returned when request currency is outside the allowlist.
	ErrCurrencyNotAllowed = errors.New("currency_not_allowed")
	// ErrCurrencyPickerDisabled is returned when a non-empty currency is sent while the flag is off.
	// Callers typically ignore request currency when disabled instead of surfacing this.
	ErrCurrencyPickerDisabled = errors.New("currency_picker_disabled")
)

// OrderCurrencyPickerEnabled gates accepting CreateRequest/UnifiedCheckout currency.
func OrderCurrencyPickerEnabled() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("ORDER_CURRENCY_PICKER_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// ParseCurrencyAllowlist parses ORDER_CURRENCY_ALLOWLIST (comma-separated ISO-4217).
// Always includes operatingCurrency. Empty env → only operating.
func ParseCurrencyAllowlist(raw, operatingCurrency string) []string {
	op := fxrates.NormalizeCurrency(operatingCurrency)
	if op == "" {
		if cur, err := auth.CurrencyFromContext(context.Background(), ""); err == nil {
			op = cur
		}
	}
	if op == "" {
		return nil
	}
	seen := map[string]struct{}{op: {}}
	out := []string{op}
	for _, part := range strings.Split(raw, ",") {
		c := fxrates.NormalizeCurrency(part)
		if len(c) != 3 {
			continue
		}
		if _, ok := seen[c]; ok {
			continue
		}
		seen[c] = struct{}{}
		out = append(out, c)
	}
	return out
}

// CurrencyOptions is GET /v1/order/currencies response.
type CurrencyOptions struct {
	Enabled           bool     `json:"enabled"`
	OperatingCurrency string   `json:"operating_currency"`
	Allowlist         []string `json:"allowlist"`
}

// resolveOrderCurrency picks the currency stamped on a new order.
// When picker is disabled, always returns operating currency (ignores request).
// When enabled: empty → operating; non-empty must be on allowlist.
func (s *Service) operatingCurrency(ctx context.Context, supplierID string) (string, error) {
	operating := fxrates.NormalizeCurrency(s.currency)
	if operating != "" {
		return operating, nil
	}
	return auth.CurrencyFromContext(ctx, supplierID)
}

func (s *Service) resolveOrderCurrency(ctx context.Context, supplierID, requested string) (string, error) {
	operating, err := s.operatingCurrency(ctx, supplierID)
	if err != nil {
		return "", err
	}
	req := fxrates.NormalizeCurrency(requested)
	if req != "" && auth.IsShippedPackCurrency(req) {
		return req, nil
	}
	if !s.currencyPickerEnabled {
		return operating, nil
	}
	if req == "" {
		return operating, nil
	}
	if len(req) != 3 {
		return "", ErrCurrencyNotAllowed
	}
	for _, c := range s.currencyAllowlist {
		if c == req {
			return req, nil
		}
	}
	return "", ErrCurrencyNotAllowed
}

// CurrencyOptions returns picker config for retailer clients.
func (s *Service) CurrencyOptions() CurrencyOptions {
	opts, err := s.currencyOptions(context.Background())
	if err != nil {
		return CurrencyOptions{Enabled: s.currencyPickerEnabled}
	}
	return opts
}

func (s *Service) currencyOptions(ctx context.Context) (CurrencyOptions, error) {
	operating, err := s.operatingCurrency(ctx, "")
	if err != nil {
		return CurrencyOptions{}, err
	}
	allow := append([]string(nil), s.currencyAllowlist...)
	if len(allow) == 0 {
		allow = []string{operating}
	}
	return CurrencyOptions{
		Enabled:           s.currencyPickerEnabled,
		OperatingCurrency: operating,
		Allowlist:         allow,
	}, nil
}

// HandleOrderCurrencies serves GET /v1/order/currencies.
func (s *Service) HandleOrderCurrencies(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	opts, err := s.currencyOptions(r.Context())
	if err != nil {
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, opts)
}

// fiscalCurrency is empty-currency law for receipts: stored ISO code, else the
// shipped pack. Planned/unknown packs fail closed — never invent UZS.
func fiscalCurrency(ctx context.Context, supplierID, stored string) (string, error) {
	c, err := auth.CoalesceCurrency(ctx, supplierID, stored)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(c) == "" {
		return "", auth.ErrMarketPackNotShipped
	}
	return c, nil
}

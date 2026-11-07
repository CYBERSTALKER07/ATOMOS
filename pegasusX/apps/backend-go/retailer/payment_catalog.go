package retailer

import (
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/payment"
)

// HandlePaymentCatalog serves GET /v1/retailer/payment-catalog (GS-R).
// Pack ∩ registry only. UZ never lists Stripe/Adyen.
func (s *Service) HandlePaymentCatalog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if _, ok := auth.FromContext(r.Context()); !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	pack, err := auth.CheckoutPackFromContext(r.Context())
	if err != nil {
		st, code := auth.CheckoutPackHTTPStatus(err)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"currency_code": pack.CurrencyCode,
		"market_code":   pack.Code,
		"catalog":       payment.AvailablePSPs(pack),
	})
}

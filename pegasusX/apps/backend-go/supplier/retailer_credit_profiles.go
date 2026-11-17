package supplier

import (
	"net/http"

	"github.com/pegasusx/pegasusx/apps/backend-go/credit"
)

// HandleRetailerCreditProfiles serves GET /v1/supplier/retailer-credit-profiles as an alias
// backed by the canonical credit desk enrichment logic.
func (s *Service) HandleRetailerCreditProfiles(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	if s.portalSpanner != nil {
		creditSvc := credit.NewService(credit.NewSpannerRepository(s.portalSpanner))
		creditSvc.HandleListSupplierProfiles(w, r)
		return
	}
	sid := s.scopedSupplierID(r)
	if sid == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"profiles":                  []any{},
		"supplier_id":               sid,
		"count":                     0,
		"pending_credit_note_count": 0,
	})
}

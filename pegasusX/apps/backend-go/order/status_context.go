package order

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/platform"
)

// HandleGetOrderStatusContext serves GET /v1/order/{orderID}/status-context.
func (s *Service) HandleGetOrderStatusContext(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orderID := strings.TrimSpace(chi.URLParam(r, "orderID"))
	if orderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}
	o, found, err := s.loadOrderForRequest(r.Context(), orderID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal_error"})
		return
	}
	if !found {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if err := assertTimelineAccess(claims, o); err != nil {
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	loc, locErr := resolveCalendarLocation(r.Context(), o.SupplierID, o.Timezone)
	if locErr != nil {
		st, code := auth.TimezonePackHTTPStatus(locErr)
		writeJSON(w, st, map[string]string{"error": code})
		return
	}
	exp := ComputeDeliveryExpectation(s.now(), loc, o)
	writeJSON(w, http.StatusOK, map[string]any{
		"order_id":             orderID,
		"status":               o.Status,
		"delivery_expectation": exp,
		"explain":              platform.ExplainForCode(string(o.Status)),
	})
}

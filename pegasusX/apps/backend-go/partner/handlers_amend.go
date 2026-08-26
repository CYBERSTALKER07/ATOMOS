package partner

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

// HandleCancelOrder POST /partner/v1/orders/{orderID}/cancel
func (h *Handlers) HandleCancelOrder(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFromContext(r.Context())
	orderID := chi.URLParam(r, "orderID")

	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}

	var req struct {
		Reason string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	resp, err := h.Svc.CancelOrder(r.Context(), p, orderID, req.Reason)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleUpdateOrderStatus POST /partner/v1/orders/{orderID}/status
func (h *Handlers) HandleUpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	p, _ := PrincipalFromContext(r.Context())
	orderID := chi.URLParam(r, "orderID")

	body, err := readBody(r)
	if err != nil {
		writePartnerError(w, http.StatusBadRequest, "read_body_error")
		return
	}

	var req order.UpdateStatusRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writePartnerError(w, http.StatusBadRequest, "invalid_json")
		return
	}

	resp, err := h.Svc.UpdateOrderStatus(r.Context(), p, orderID, req)
	if err != nil {
		writePartnerError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

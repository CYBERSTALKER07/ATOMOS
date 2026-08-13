package deliveryroutes

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
)

type Deps struct {
	Service             *order.Service
	FirebaseAuthEnabled bool
	FirebaseVerifier    auth.FirebaseVerifier
	AllowAuthBypass     bool
}

func RegisterRoutes(r chi.Router, d Deps) {
	r.Route("/v1/delivery", func(r chi.Router) {
		if d.FirebaseAuthEnabled && d.FirebaseVerifier != nil {
			r.Use(auth.FirebaseAuth(d.FirebaseVerifier))
		}
		r.Use(auth.RequireRole(auth.RoleDriver))

		r.Post("/verify-handshake", handleVerifyHandshake(d.Service))
		r.Post("/update-order-during-delivery", handleUpdateOrderDuringDelivery(d.Service))
	})
}

func handleVerifyHandshake(svc *order.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req order.VerifyHandshakeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}

		claims, ok := auth.FromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		resp, err := svc.VerifyHandshake(r.Context(), claims, req)
		if err != nil {
			status := http.StatusUnprocessableEntity
			if errors.Is(err, order.ErrOrderNotFound) {
				status = http.StatusNotFound
			} else if errors.Is(err, order.ErrOrderForbidden) {
				status = http.StatusForbidden
			}
			writeJSON(w, status, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func handleUpdateOrderDuringDelivery(svc *order.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req order.UpdateOrderDuringDeliveryRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}

		claims, ok := auth.FromContext(r.Context())
		if !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}

		resp, err := svc.UpdateOrderDuringDelivery(r.Context(), claims, req)
		if err != nil {
			status := http.StatusUnprocessableEntity
			msg := err.Error()
			if errors.Is(err, order.ErrOrderNotFound) {
				status = http.StatusNotFound
			} else if errors.Is(err, order.ErrOrderForbidden) {
				status = http.StatusForbidden
			} else if strings.HasPrefix(msg, "not_implemented") {
				status = http.StatusNotImplemented
			}
			writeJSON(w, status, map[string]string{"error": msg})
			return
		}

		writeJSON(w, http.StatusOK, resp)
	}
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

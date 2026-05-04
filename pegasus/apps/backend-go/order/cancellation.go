package order

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"backend-go/auth"
	"backend-go/telemetry"
)

// HandleCancelOrder cancels a retailer-owned order using JWT-bound scope.
func HandleCancelOrder(service *OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.Role != "RETAILER" || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req CancelOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.RetailerID != "" && req.RetailerID != claims.UserID {
			http.Error(w, `{"error":"forbidden: cannot cancel another retailer's order"}`, http.StatusForbidden)
			return
		}
		req.RetailerID = claims.UserID
		if req.OrderID == "" {
			http.Error(w, "order_id is required", http.StatusBadRequest)
			return
		}

		if err := service.CancelOrder(r.Context(), req); err != nil {
			writeCancelOrderError(w, r, req.OrderID, err)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(fmt.Sprintf(`{"status":"ORDER_CANCELLED","order_id":"%s"}`, req.OrderID)))
	}
}

func writeCancelOrderError(w http.ResponseWriter, r *http.Request, orderID string, err error) {
	var stateConflict *ErrStateConflict
	var versionConflict *ErrVersionConflict
	var freezeLock *ErrFreezeLock
	var forbidden *ErrCancelForbidden

	w.Header().Set("Content-Type", "application/json")
	switch {
	case errors.As(err, &stateConflict):
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": stateConflict.Error()})
	case errors.As(err, &versionConflict):
		w.WriteHeader(http.StatusConflict)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": versionConflict.Error()})
	case errors.As(err, &freezeLock):
		w.WriteHeader(http.StatusLocked)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": freezeLock.Error()})
	case errors.As(err, &forbidden):
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": forbidden.Reason})
	case strings.Contains(err.Error(), "not found"):
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
	default:
		slog.ErrorContext(r.Context(), "order.cancel_failed",
			"trace_id", telemetry.TraceIDFromContext(r.Context()),
			"order_id", orderID,
			"err", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "Internal Server Error"})
	}
}

// Package orderroutes owns legacy-compatible order route composition that is
// still shared by retailer, supplier, admin portal, and driver clients.
package orderroutes

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"backend-go/auth"
	"backend-go/order"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/telemetry"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
)

// Middleware is the handler-wrap contract supplied by main.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps contains only collaborators needed by the shared order routes.
type Deps struct {
	Spanner    *spanner.Client
	ReadRouter proximity.ReadRouter
	Order      *order.OrderService
	Refund     *payment.RefundService
	Log        Middleware
}

// RegisterRoutes mounts the extracted legacy-compatible order surface.
func RegisterRoutes(r chi.Router, d Deps) {
	r.HandleFunc("/v1/order/refunds",
		auth.RequireRole([]string{"ADMIN", "SUPPLIER", "RETAILER"}, d.Log(handleOrderRefunds(d))))
	r.HandleFunc("/v1/orders",
		auth.RequireRole([]string{"ADMIN", "RETAILER", "SUPPLIER", "PAYLOADER"}, d.Log(handleOrdersList(d))))
	r.HandleFunc("/v1/orders/",
		auth.RequireRole([]string{"ADMIN", "DRIVER", "RETAILER", "SUPPLIER"}, d.Log(order.HandleLegacyOrdersPath(d.Order))))
	r.HandleFunc("/v1/orders/line-items/history",
		auth.RequireRole([]string{"RETAILER", "ADMIN"}, d.Log(order.HandleLineItemHistory(d.Spanner, d.ReadRouter))))
}

func handleOrdersList(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed - use POST /v1/order/create", http.StatusMethodNotAllowed)
			return
		}

		claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		retailerID := r.URL.Query().Get("retailer_id")
		if claims != nil && claims.Role == "RETAILER" {
			retailerID = claims.UserID
		}
		supplierID, ok := supplierScopeForList(claims)
		if !ok {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}

		limit, offset, err := parsePagination(r)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		orders, err := d.Order.ListOrdersPaginatedScoped(
			r.Context(),
			r.URL.Query().Get("route_id"),
			r.URL.Query().Get("state"),
			retailerID,
			supplierID,
			limit,
			offset,
		)
		if err != nil {
			slog.ErrorContext(r.Context(), "order_routes.list_failed",
				"trace_id", telemetry.TraceIDFromContext(r.Context()),
				"role", roleOf(claims),
				"actor_id", actorOf(claims),
				"err", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(orders); err != nil {
			slog.ErrorContext(r.Context(), "order_routes.list_write_failed", "err", err)
		}
	}
}

func handleOrderRefunds(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		orderID := r.URL.Query().Get("order_id")
		if orderID == "" {
			http.Error(w, `{"error":"order_id query param required"}`, http.StatusBadRequest)
			return
		}

		claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if err := verifyRefundScope(r, d.Spanner, claims, orderID); err != nil {
			writeRefundScopeError(w, r, orderID, err)
			return
		}
		refunds, err := d.Refund.GetRefundsByOrder(r.Context(), orderID)
		if err != nil {
			slog.ErrorContext(r.Context(), "order_routes.refunds_failed",
				"trace_id", telemetry.TraceIDFromContext(r.Context()),
				"order_id", orderID,
				"err", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(refunds); err != nil {
			slog.ErrorContext(r.Context(), "order_routes.refunds_write_failed", "order_id", orderID, "err", err)
		}
	}
}

func supplierScopeForList(claims *auth.PegasusClaims) (string, bool) {
	if claims == nil {
		return "", true
	}
	if claims.Role != "ADMIN" && claims.Role != "SUPPLIER" {
		return "", true
	}
	supplierID := claims.ResolveSupplierID()
	return supplierID, supplierID != ""
}

func parsePagination(r *http.Request) (int, int64, error) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid limit")
		}
		limit = parsed
	}
	offset := int64(0)
	if raw := r.URL.Query().Get("offset"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return 0, 0, fmt.Errorf("Invalid offset")
		}
		offset = parsed
	}
	return limit, offset, nil
}

func verifyRefundScope(r *http.Request, client *spanner.Client, claims *auth.PegasusClaims, orderID string) error {
	row, err := client.Single().ReadRow(r.Context(), "Orders", spanner.Key{orderID}, []string{"RetailerId", "SupplierId"})
	if err != nil {
		return err
	}
	var ownerRetailerID string
	var ownerSupplierID spanner.NullString
	if err := row.Columns(&ownerRetailerID, &ownerSupplierID); err != nil {
		return fmt.Errorf("decode refund order scope: %w", err)
	}
	if claims != nil && claims.Role == "RETAILER" && ownerRetailerID != claims.UserID {
		return errRefundForbidden
	}
	if claims != nil && (claims.Role == "ADMIN" || claims.Role == "SUPPLIER") {
		supplierID := claims.ResolveSupplierID()
		if supplierID == "" || !ownerSupplierID.Valid || ownerSupplierID.StringVal != supplierID {
			return errRefundForbidden
		}
	}
	return nil
}

var errRefundForbidden = errors.New("refund outside caller scope")

func writeRefundScopeError(w http.ResponseWriter, r *http.Request, orderID string, err error) {
	switch {
	case errors.Is(err, spanner.ErrRowNotFound):
		http.Error(w, `{"error":"order not found"}`, http.StatusNotFound)
	case errors.Is(err, errRefundForbidden):
		http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
	default:
		slog.ErrorContext(r.Context(), "order_routes.refunds_scope_failed",
			"trace_id", telemetry.TraceIDFromContext(r.Context()),
			"order_id", orderID,
			"err", err)
		http.Error(w, `{"error":"failed to verify order scope"}`, http.StatusInternalServerError)
	}
}

func roleOf(claims *auth.PegasusClaims) string {
	if claims == nil {
		return ""
	}
	return claims.Role
}

func actorOf(claims *auth.PegasusClaims) string {
	if claims == nil {
		return ""
	}
	return claims.UserID
}

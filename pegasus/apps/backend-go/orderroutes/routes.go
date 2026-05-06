// Package orderroutes owns legacy-compatible order route composition that is
// still shared by retailer, supplier, admin portal, and driver clients.
package orderroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/fleet"
	"backend-go/order"
	"backend-go/payment"
	"backend-go/proximity"
	"backend-go/telemetry"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
)

// Middleware is the handler-wrap contract supplied by main.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps contains only collaborators needed by the shared order routes.
type Deps struct {
	Spanner     *spanner.Client
	ReadRouter  proximity.ReadRouter
	Order       *order.OrderService
	Refund      *payment.RefundService
	RetailerHub *ws.RetailerHub
	DriverHub   *ws.DriverHub
	FleetHub    *telemetry.Hub
	MapsAPIKey  string
	Log         Middleware
	Idempotency Middleware
}

// RegisterRoutes mounts the extracted legacy-compatible order surface.
func RegisterRoutes(r chi.Router, d Deps) {
	logWrap := d.Log
	if logWrap == nil {
		logWrap = passthrough
	}
	idemWrap := d.Idempotency
	if idemWrap == nil {
		idemWrap = passthrough
	}

	r.HandleFunc("/v1/order/refunds",
		auth.RequireRole([]string{"ADMIN", "SUPPLIER", "RETAILER"}, logWrap(handleOrderRefunds(d))))
	r.HandleFunc("/v1/orders",
		auth.RequireRole([]string{"ADMIN", "RETAILER", "SUPPLIER", "PAYLOADER"}, logWrap(handleOrdersList(d))))
	r.HandleFunc("/v1/orders/",
		auth.RequireRole([]string{"ADMIN", "DRIVER", "RETAILER", "SUPPLIER"}, logWrap(order.HandleLegacyOrdersPath(d.Order))))
	r.HandleFunc("/v1/orders/line-items/history",
		auth.RequireRole([]string{"RETAILER", "ADMIN"}, logWrap(order.HandleLineItemHistory(d.Spanner, d.ReadRouter))))

	// Legacy compatibility endpoints extracted from infraroutes.
	r.HandleFunc("/v1/order/deliver",
		auth.RequireRole([]string{"DRIVER"}, logWrap(idemWrap(handleOrderDeliver(d)))))
	r.HandleFunc("/v1/order/validate-qr",
		auth.RequireRole([]string{"DRIVER"}, logWrap(handleOrderValidateQR(d.Order))))
	r.HandleFunc("/v1/order/confirm-offload",
		auth.RequireRole([]string{"DRIVER"}, logWrap(idemWrap(handleOrderConfirmOffload(d)))))
	r.HandleFunc("/v1/order/complete",
		auth.RequireRole([]string{"DRIVER"}, logWrap(idemWrap(handleOrderComplete(d)))))
	r.HandleFunc("/v1/order/collect-cash",
		auth.RequireRole([]string{"DRIVER"}, logWrap(idemWrap(handleOrderCollectCash(d)))))

	r.HandleFunc("/v1/routes",
		auth.RequireRole([]string{"ADMIN", "SUPPLIER", "PAYLOADER"}, logWrap(handleRoutes(d.Order))))
	r.HandleFunc("/v1/prediction/create",
		auth.RequireRole([]string{"RETAILER"}, logWrap(handlePredictionCreate(d.Order))))

	r.HandleFunc("/v1/order/refund",
		auth.RequireRole([]string{"ADMIN", "SUPPLIER"}, logWrap(idemWrap(handleOrderRefund(d)))))
	r.HandleFunc("/v1/order/amend",
		auth.RequireRole([]string{"DRIVER", "ADMIN"}, logWrap(handleOrderAmend(d))))

	if d.Order != nil {
		r.HandleFunc("/v1/vehicle/*",
			auth.RequireRole([]string{"ADMIN", "SUPPLIER"}, logWrap(idemWrap(d.Order.HandleClearReturns))))
	}
}

func passthrough(next http.HandlerFunc) http.HandlerFunc {
	return next
}

func withDetachedRequestContext(r *http.Request) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(r.Context()), 30*time.Second)
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

func handleOrderDeliver(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if d.Order == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			OrderId      string  `json:"order_id"`
			ScannedToken string  `json:"scanned_token"`
			Latitude     float64 `json:"latitude"`
			Longitude    float64 `json:"longitude"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderId == "" || req.ScannedToken == "" {
			http.Error(w, "Invalid payload. order_id and scanned_token required.", http.StatusBadRequest)
			return
		}

		supplierID, err := d.Order.CompleteDeliveryWithToken(r.Context(), req.OrderId, req.ScannedToken, req.Latitude, req.Longitude)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}

		go func(orderID string) {
			ctx, cancel := withDetachedRequestContext(r)
			defer cancel()
			d.Order.InvalidateDeliveryToken(ctx, orderID)
		}(req.OrderId)

		if d.FleetHub != nil && supplierID != "" {
			go d.FleetHub.BroadcastOrderStateChange(supplierID, req.OrderId, "COMPLETED", "")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETED",
			"message": "Handshake successful. Delivery completed.",
		})

		if d.Spanner != nil {
			go func(orderID string) {
				ctx, cancel := withDetachedRequestContext(r)
				defer cancel()
				fleet.CheckAndAutoReleaseTruck(ctx, d.Spanner, orderID, d.MapsAPIKey)
			}(req.OrderId)
		}
	}
}

func handleOrderValidateQR(orderSvc *order.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if orderSvc == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			OrderID      string `json:"order_id"`
			ScannedToken string `json:"scanned_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || req.ScannedToken == "" {
			http.Error(w, "order_id and scanned_token required", http.StatusBadRequest)
			return
		}
		resp, err := orderSvc.ValidateQRToken(r.Context(), req.OrderID, req.ScannedToken)
		if err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleOrderConfirmOffload(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.Order == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}
		resp, err := d.Order.ConfirmOffload(r.Context(), req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		if d.RetailerHub != nil {
			d.RetailerHub.PushToRetailer(resp.RetailerID, map[string]interface{}{
				"type":                    ws.EventPaymentRequired,
				"order_id":                resp.OrderID,
				"invoice_id":              resp.InvoiceID,
				"session_id":              resp.SessionID,
				"amount":                  resp.Amount,
				"original_amount":         resp.OriginalAmount,
				"payment_method":          resp.PaymentMethod,
				"available_card_gateways": resp.AvailableCardGateways,
				"message":                 fmt.Sprintf("Payment of %d required for order %s", resp.Amount, resp.OrderID),
			})
		}

		if d.FleetHub != nil && resp.SupplierID != "" {
			go d.FleetHub.BroadcastOrderStateChange(resp.SupplierID, resp.OrderID, "AWAITING_PAYMENT", "")
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleOrderComplete(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.Order == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		var req struct {
			OrderID string `json:"order_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}
		supplierID, err := d.Order.CompleteOrder(r.Context(), req.OrderID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		go func(orderID string) {
			ctx, cancel := withDetachedRequestContext(r)
			defer cancel()
			d.Order.InvalidateDeliveryToken(ctx, orderID)
		}(req.OrderID)

		if d.FleetHub != nil && supplierID != "" {
			go d.FleetHub.BroadcastOrderStateChange(supplierID, req.OrderID, "COMPLETED", "")
		}

		if d.Spanner != nil {
			go func(orderID string) {
				ctx, cancel := withDetachedRequestContext(r)
				defer cancel()
				fleet.CheckAndAutoReleaseTruck(ctx, d.Spanner, orderID, d.MapsAPIKey)
			}(req.OrderID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETED",
			"message": "Delivery finalized.",
		})
	}
}

func handleOrderCollectCash(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.Order == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			OrderID   string  `json:"order_id"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" {
			http.Error(w, "order_id required", http.StatusBadRequest)
			return
		}
		if req.Latitude == 0 && req.Longitude == 0 {
			http.Error(w, "GPS coordinates required (latitude, longitude)", http.StatusBadRequest)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims.UserID == "" {
			http.Error(w, "driver identity missing from token", http.StatusUnauthorized)
			return
		}

		resp, err := d.Order.CollectCash(r.Context(), order.CollectCashRequest{
			OrderID:   req.OrderID,
			DriverID:  claims.UserID,
			Latitude:  req.Latitude,
			Longitude: req.Longitude,
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}

		if d.RetailerHub != nil {
			d.RetailerHub.PushToRetailer(resp.RetailerID, map[string]interface{}{
				"type":     ws.EventOrderCompleted,
				"order_id": resp.OrderID,
				"amount":   resp.Amount,
				"message":  resp.Message,
			})
		}

		if d.Spanner != nil {
			go func(orderID string) {
				ctx, cancel := withDetachedRequestContext(r)
				defer cancel()
				fleet.CheckAndAutoReleaseTruck(ctx, d.Spanner, orderID, d.MapsAPIKey)
			}(req.OrderID)
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleRoutes(orderSvc *order.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if orderSvc == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}
		routes, err := orderSvc.ListRoutes(r.Context())
		if err != nil {
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(routes)
	}
}

func handlePredictionCreate(orderSvc *order.OrderService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if orderSvc == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		var req struct {
			RetailerId  string `json:"retailer_id"`
			Amount      int64  `json:"amount"`
			TriggerDate string `json:"trigger_date"`
			Status      string `json:"status,omitempty"`
			WarehouseId string `json:"warehouse_id,omitempty"`
			Items       []struct {
				SkuID    string `json:"sku_id"`
				Quantity int64  `json:"quantity"`
				Price    int64  `json:"price"`
			} `json:"items,omitempty"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid payload", http.StatusBadRequest)
			return
		}

		if len(req.Items) > 0 {
			var items []order.PredictionItem
			for _, it := range req.Items {
				items = append(items, order.PredictionItem{
					SkuID: it.SkuID, Quantity: it.Quantity, Price: it.Price,
				})
			}
			err := orderSvc.SavePredictionWithItems(r.Context(), req.RetailerId, req.Amount, req.TriggerDate, items, req.Status, req.WarehouseId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		} else {
			err := orderSvc.SavePrediction(r.Context(), req.RetailerId, req.Amount, req.TriggerDate, req.WarehouseId)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}

		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "PREDICTION_LOCKED"})
	}
}

func handleOrderRefund(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.Refund == nil {
			http.Error(w, "refund service unavailable", http.StatusServiceUnavailable)
			return
		}

		claims, _ := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		var req payment.RefundRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON payload", http.StatusBadRequest)
			return
		}
		if req.OrderID == "" {
			http.Error(w, `{"error":"order_id is required"}`, http.StatusBadRequest)
			return
		}

		actorID := ""
		if claims != nil {
			actorID = claims.UserID
		}
		result, err := d.Refund.InitiateRefund(r.Context(), req, actorID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(result)
	}
}

func handleOrderAmend(d Deps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if d.Order == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		var req order.AmendOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON body", http.StatusBadRequest)
			return
		}
		if req.OrderID == "" || len(req.Items) == 0 {
			http.Error(w, "order_id and items are required", http.StatusBadRequest)
			return
		}

		resp, err := d.Order.AmendOrder(r.Context(), req)
		if err != nil {
			var versionConflict *order.ErrVersionConflict
			if errors.As(err, &versionConflict) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusConflict)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": versionConflict.Error()})
				return
			}
			var freezeLock *order.ErrFreezeLock
			if errors.As(err, &freezeLock) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(423)
				_ = json.NewEncoder(w).Encode(map[string]string{"error": freezeLock.Error()})
				return
			}
			if strings.Contains(err.Error(), "cannot be amended") {
				http.Error(w, err.Error(), http.StatusConflict)
			} else if strings.Contains(err.Error(), "not found") {
				http.Error(w, err.Error(), http.StatusNotFound)
			} else {
				http.Error(w, "internal error: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}

		if d.RetailerHub != nil && resp.RetailerID != "" {
			go d.RetailerHub.PushToRetailer(resp.RetailerID, map[string]interface{}{
				"type":         ws.EventOrderAmended,
				"order_id":     req.OrderID,
				"amendment_id": resp.AmendmentID,
				"new_total":    resp.AdjustedTotal,
				"message":      resp.Message,
			})
		}
		if d.DriverHub != nil && resp.DriverID != "" {
			go d.DriverHub.PushToDriver(resp.DriverID, map[string]interface{}{
				"type":         ws.EventOrderAmended,
				"order_id":     req.OrderID,
				"amendment_id": resp.AmendmentID,
				"new_total":    resp.AdjustedTotal,
				"message":      resp.Message,
			})
		}
		if d.FleetHub != nil && resp.SupplierID != "" {
			go d.FleetHub.BroadcastOrderStateChange(resp.SupplierID, req.OrderID, "AMENDED", "")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(resp)
	}
}

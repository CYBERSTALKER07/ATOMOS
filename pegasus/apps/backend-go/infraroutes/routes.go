package infraroutes
// Package infraroutes owns legacy-compatible infrastructure and compatibility
// endpoints that were previously mounted inline in main.go via http.HandleFunc.
//
// This package is a thin route composer: it keeps existing handler behavior
// while removing direct route registration from the composition root.
package infraroutes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"google.golang.org/api/iterator"

	"backend-go/auth"
	"backend-go/cache"
	"backend-go/fleet"
	"backend-go/order"
	"backend-go/payment"
	"backend-go/ws"
)

// Middleware is the handler-wrap contract supplied by the caller.
type Middleware func(http.HandlerFunc) http.HandlerFunc

// Deps bundles collaborators required to register extracted legacy endpoints.
type Deps struct {
	Spanner             *spanner.Client
	Order               *order.OrderService
	Refund              *payment.RefundService
	FleetHub            *ws.FleetHub
	RetailerHub         *ws.RetailerHub
	DriverHub           *ws.DriverHub
	WarehouseHub        *ws.WarehouseHub
	MapsAPIKey          string
	Log                 Middleware
	Idempotency         Middleware
	EnableDebugMintToken bool
}

// RegisterRoutes mounts the extracted legacy-compatible endpoints that were
// previously registered directly in main.go.
func RegisterRoutes(r chi.Router, d Deps) {
	logWrap := d.Log
	if logWrap == nil {
		logWrap = passthrough
	}
	idemWrap := d.Idempotency
	if idemWrap == nil {
		idemWrap = passthrough
	}

	if d.FleetHub != nil {
		r.HandleFunc("/ws/telemetry",
			auth.RequireRoleWithGrace([]string{"DRIVER", "ADMIN", "SUPPLIER"}, 2*time.Hour, d.FleetHub.HandleConnection))
		r.HandleFunc("/ws/fleet",
			auth.RequireRoleWithGrace([]string{"DRIVER", "ADMIN", "SUPPLIER"}, 2*time.Hour, d.FleetHub.HandleConnection))
	}

	r.HandleFunc("/v1/health", handleHealth(d.Spanner))

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
	r.HandleFunc("/v1/products",
		auth.RequireRole([]string{"RETAILER", "ADMIN"}, logWrap(handleProducts(d.Spanner))))

	r.HandleFunc("/v1/order/amend",
		auth.RequireRole([]string{"DRIVER", "ADMIN"}, logWrap(handleOrderAmend(d))))

	if d.Order != nil {
		r.HandleFunc("/v1/vehicle/*",
			auth.RequireRole([]string{"ADMIN", "SUPPLIER"}, logWrap(idemWrap(d.Order.HandleClearReturns))))
	}

	if d.WarehouseHub != nil {
		r.HandleFunc("/ws/warehouse",
			auth.RequireRoleWithGrace([]string{"WAREHOUSE", "SUPPLIER", "ADMIN"}, 2*time.Hour, d.WarehouseHub.HandleConnection))
	}

	if d.EnableDebugMintToken {
		r.HandleFunc("/debug/mint-token", handleDebugMintToken)
	}
}

func passthrough(next http.HandlerFunc) http.HandlerFunc {
	return next
}

func handleHealth(spannerClient *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		spannerOK := false
		if spannerClient != nil {
			row, err := spannerClient.Single().ReadRow(r.Context(), "Suppliers", spanner.Key{"health-check-probe"}, []string{"SupplierId"})
			spannerOK = err != nil || row != nil
		}

		redisOK := cache.Client != nil
		if redisOK {
			if err := cache.Client.Ping(r.Context()).Err(); err != nil {
				redisOK = false
			}
		}

		status := "healthy"
		code := http.StatusOK
		if !spannerOK {
			status = "degraded"
			code = http.StatusServiceUnavailable
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(code)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  status,
			"spanner": spannerOK,
			"redis":   redisOK,
			"time":    time.Now().UTC().Format(time.RFC3339),
		})
	}
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

		go d.Order.InvalidateDeliveryToken(context.Background(), req.OrderId)

		if d.FleetHub != nil && supplierID != "" {
			go d.FleetHub.BroadcastOrderStateChange(supplierID, req.OrderId, "COMPLETED", "")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{
			"status":  "COMPLETED",
			"message": "Handshake successful. Delivery completed.",
		})

		if d.Spanner != nil {
			go fleet.CheckAndAutoReleaseTruck(context.Background(), d.Spanner, req.OrderId, d.MapsAPIKey)
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
		json.NewEncoder(w).Encode(resp)
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
		json.NewEncoder(w).Encode(resp)
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

		go d.Order.InvalidateDeliveryToken(context.Background(), req.OrderID)

		if d.FleetHub != nil && supplierID != "" {
			go d.FleetHub.BroadcastOrderStateChange(supplierID, req.OrderID, "COMPLETED", "")
		}

		if d.Spanner != nil {
			go fleet.CheckAndAutoReleaseTruck(context.Background(), d.Spanner, req.OrderID, d.MapsAPIKey)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
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
			go fleet.CheckAndAutoReleaseTruck(context.Background(), d.Spanner, req.OrderID, d.MapsAPIKey)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
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
			log.Printf("Failed to list routes: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(routes)
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
		json.NewEncoder(w).Encode(map[string]string{"status": "PREDICTION_LOCKED"})
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
			log.Printf("[REFUND] Failed for order %s: %v", req.OrderID, err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(result)
	}
}

func handleProducts(spannerClient *spanner.Client) http.HandlerFunc {
	type variant struct {
		ID            string  `json:"id"`
		Size          string  `json:"size"`
		Pack          string  `json:"pack"`
		PackCount     int64   `json:"pack_count"`
		WeightPerUnit string  `json:"weight_per_unit"`
		Price         float64 `json:"price"`
	}

	type product struct {
		ID               string    `json:"id"`
		Name             string    `json:"name"`
		Description      string    `json:"description"`
		Nutrition        string    `json:"nutrition"`
		ImageURL         string    `json:"image_url"`
		Variants         []variant `json:"variants"`
		SupplierID       string    `json:"supplier_id"`
		SupplierName     string    `json:"supplier_name"`
		SupplierCategory string    `json:"supplier_category"`
		CategoryID       string    `json:"category_id"`
		CategoryName     string    `json:"category_name"`
		SellByBlock      bool      `json:"sell_by_block"`
		UnitsPerBlock    int64     `json:"units_per_block"`
		Price            int64     `json:"price"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		if spannerClient == nil {
			http.Error(w, "service unavailable", http.StatusServiceUnavailable)
			return
		}

		stmt := spanner.Statement{
			SQL: `SELECT sp.SkuId, sp.SupplierId, sp.Name, sp.Description, sp.ImageUrl,
			             sp.SellByBlock, sp.UnitsPerBlock, sp.BasePrice, sp.CategoryId,
			             COALESCE(c.Name, '') AS CategoryName,
			             COALESCE(s.Name, '') AS SupplierName,
			             COALESCE(s.Category, '') AS SupplierCategory
			      FROM SupplierProducts sp
			      LEFT JOIN Suppliers s ON sp.SupplierId = s.SupplierId
			      LEFT JOIN Categories c ON c.CategoryId = sp.CategoryId
			      WHERE sp.IsActive = TRUE
			      ORDER BY sp.Name ASC`,
		}

		iter := spannerClient.Single().Query(r.Context(), stmt)
		defer iter.Stop()

		var productList []product
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				log.Printf("Failed to query products: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			var skuId, supplierId, name string
			var desc, imageURL, catID, categoryName, supplierName, supplierCategory spanner.NullString
			var sellByBlock bool
			var unitsPerBlock, basePrice int64

			if err := row.Columns(&skuId, &supplierId, &name, &desc, &imageURL,
				&sellByBlock, &unitsPerBlock, &basePrice, &catID, &categoryName, &supplierName, &supplierCategory); err != nil {
				log.Printf("Failed to parse product row: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			p := product{
				ID:            skuId,
				Name:          name,
				SellByBlock:   sellByBlock,
				UnitsPerBlock: unitsPerBlock,
				Price:         basePrice,
				SupplierID:    supplierId,
			}
			if desc.Valid {
				p.Description = desc.StringVal
			}
			if imageURL.Valid {
				p.ImageURL = imageURL.StringVal
			}
			if catID.Valid {
				p.CategoryID = catID.StringVal
			}
			if categoryName.Valid {
				p.CategoryName = categoryName.StringVal
			}
			if supplierName.Valid {
				p.SupplierName = supplierName.StringVal
			}
			if supplierCategory.Valid {
				p.SupplierCategory = supplierCategory.StringVal
			}

			packLabel := "Per unit"
			if sellByBlock && unitsPerBlock > 1 {
				packLabel = fmt.Sprintf("Block of %d", unitsPerBlock)
			}
			p.Variants = []variant{{
				ID:            skuId,
				Size:          "Standard",
				Pack:          packLabel,
				PackCount:     1,
				WeightPerUnit: "1 unit",
				Price:         float64(basePrice),
			}}

			productList = append(productList, p)
		}

		if productList == nil {
			productList = []product{}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(productList); err != nil {
			log.Printf("Failed to write products response payload: %v", err)
		}
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
				json.NewEncoder(w).Encode(map[string]string{"error": versionConflict.Error()})
				return
			}
			var freezeLock *order.ErrFreezeLock
			if errors.As(err, &freezeLock) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(423)
				json.NewEncoder(w).Encode(map[string]string{"error": freezeLock.Error()})
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
		json.NewEncoder(w).Encode(resp)
	}
}

func handleDebugMintToken(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "" {
		role = "RETAILER"
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		userID = "TEST-USER-99"
	}

	tokenString, err := auth.GenerateTestToken(userID, role)
	if err != nil {
		http.Error(w, "Failed to forge token", http.StatusInternalServerError)
		return
	}

	w.Write([]byte(tokenString))
}
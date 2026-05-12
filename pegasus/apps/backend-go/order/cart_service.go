package order

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"backend-go/auth"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// ── Server-Side Cart Persistence ────────────────────────────────────────────
// POST /v1/retailer/cart/sync — Full cart sync (client sends entire cart, server replaces)
// GET  /v1/retailer/cart/sync — Returns current server-side cart

type CartItem struct {
	CartID     string `json:"cart_id,omitempty"`
	SkuID      string `json:"sku_id"`
	SupplierID string `json:"supplier_id"`
	Quantity   int64  `json:"quantity"`
	UnitPrice  int64  `json:"unit_price"`
	Currency   string `json:"currency"`
}

type CartSyncRequest struct {
	Items []CartItem `json:"items"`
}

type CartSyncResponse struct {
	Items []CartItem `json:"items"`
	Total int        `json:"total"`
}

func HandleCartSync(client *spanner.Client, retailerHub *ws.RetailerHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok || claims == nil || claims.UserID == "" {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		switch r.Method {
		case http.MethodGet:
			handleCartGet(w, r, client, claims.UserID)
		case http.MethodPost:
			handleCartPut(w, r, client, retailerHub, claims.UserID)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleCartGet(w http.ResponseWriter, r *http.Request, client *spanner.Client, retailerID string) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stmt := spanner.Statement{
		SQL: `SELECT CartId, SkuId, SupplierId, Quantity, UnitPrice, Currency
			FROM RetailerCarts
			WHERE RetailerId = @retailerId
			ORDER BY AddedAt DESC`,
		Params: map[string]interface{}{"retailerId": retailerID},
	}

	resp := CartSyncResponse{Items: []CartItem{}}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			slog.ErrorContext(ctx, "cart sync query failed", "retailer_id", retailerID, "err", err)
			http.Error(w, `{"error":"query_failed"}`, http.StatusInternalServerError)
			return
		}
		var item CartItem
		if err := row.Columns(&item.CartID, &item.SkuID, &item.SupplierID, &item.Quantity, &item.UnitPrice, &item.Currency); err != nil {
			slog.WarnContext(ctx, "cart sync decode row failed", "retailer_id", retailerID, "err", err)
			continue
		}
		resp.Items = append(resp.Items, item)
	}
	resp.Total = len(resp.Items)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func handleCartPut(w http.ResponseWriter, r *http.Request, client *spanner.Client, retailerHub *ws.RetailerHub, retailerID string) {
	var req CartSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid JSON"}`, http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()
	insertedCount := 0

	_, err := client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Delete all existing cart items for this retailer
		deleteStmt := spanner.Statement{
			SQL:    `DELETE FROM RetailerCarts WHERE RetailerId = @retailerId`,
			Params: map[string]interface{}{"retailerId": retailerID},
		}
		if _, err := txn.Update(ctx, deleteStmt); err != nil {
			return err
		}

		// 2. Insert new items
		now := time.Now().UTC()
		for _, item := range req.Items {
			if item.SkuID == "" || item.SupplierID == "" || item.Quantity <= 0 {
				continue
			}
			currency := strings.TrimSpace(item.Currency)
			if currency == "" {
				currency = "UZS"
			}
			cartID := uuid.NewString()
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("RetailerCarts",
					[]string{"CartId", "RetailerId", "SupplierId", "SkuId", "Quantity", "UnitPrice", "Currency", "AddedAt"},
					[]interface{}{cartID, retailerID, item.SupplierID, item.SkuID, item.Quantity, item.UnitPrice, currency, now},
				),
			}); err != nil {
				return err
			}
			insertedCount++
		}
		return nil
	})
	if err != nil {
		slog.ErrorContext(ctx, "cart sync transaction failed", "retailer_id", retailerID, "err", err)
		http.Error(w, `{"error":"sync_failed"}`, http.StatusInternalServerError)
		return
	}

	if retailerHub != nil {
		retailerHub.PushToRetailer(retailerID, map[string]interface{}{
			"type":        ws.EventCartSyncUpdated,
			"retailer_id": retailerID,
			"item_count":  insertedCount,
			"updated_at":  time.Now().UTC().Format(time.RFC3339Nano),
		})
	}

	// Return the synced cart
	handleCartGet(w, r, client, retailerID)
}

package sync

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"backend-go/auth"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ── Reconnection Catch-Up ───────────────────────────────────────────────────
// When a client reconnects after losing WebSocket/network, it calls
// GET /v1/sync/catchup?since=<RFC3339> to receive a compact delta of all
// changes since the last known good state. This avoids replaying hundreds
// of missed real-time events — the client gets a single "current state" payload.

// CatchupResponse is the delta payload returned to reconnecting clients.
type CatchupResponse struct {
	Orders              []OrderDelta `json:"orders"`
	Fleet               []FleetDelta `json:"fleet,omitempty"`
	UnreadNotifications int64        `json:"unread_notifications"`
	ServerTime          string       `json:"server_time"`
}

// OrderDelta is a minimal order state change record.
type OrderDelta struct {
	OrderID   string `json:"order_id"`
	State     string `json:"state"`
	UpdatedAt string `json:"updated_at"`
}

// FleetDelta is a minimal driver status change record.
type FleetDelta struct {
	DriverID  string `json:"driver_id"`
	Status    string `json:"status"`
	IsOffline bool   `json:"is_offline"`
}

// HandleCatchup returns a compact delta of changes since the given timestamp.
// Used by all client roles to recover state after a WebSocket disconnect.
func HandleCatchup(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		sinceStr := r.URL.Query().Get("since")
		if sinceStr == "" {
			http.Error(w, `{"error":"'since' query parameter required (RFC3339)"}`, http.StatusBadRequest)
			return
		}

		since, err := time.Parse(time.RFC3339, sinceStr)
		if err != nil {
			http.Error(w, `{"error":"invalid RFC3339 timestamp"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		resp := CatchupResponse{
			ServerTime: time.Now().UTC().Format(time.RFC3339),
		}

		// ── Order deltas ────────────────────────────────────────────────
		// Scope by role: SUPPLIER/ADMIN sees their orders, RETAILER sees theirs, DRIVER sees assigned
		var orderSQL string
		params := map[string]interface{}{"since": since}

		switch claims.Role {
		case "ADMIN", "SUPPLIER":
			supplierID := claims.ResolveSupplierID()
			orderSQL = `SELECT OrderId, State, UpdatedAt FROM Orders
				WHERE SupplierId = @scopeID AND UpdatedAt > @since
				ORDER BY UpdatedAt DESC LIMIT 100`
			params["scopeID"] = supplierID

		case "RETAILER":
			orderSQL = `SELECT OrderId, State, UpdatedAt FROM Orders
				WHERE RetailerId = @scopeID AND UpdatedAt > @since
				ORDER BY UpdatedAt DESC LIMIT 100`
			params["scopeID"] = claims.UserID

		case "DRIVER":
			orderSQL = `SELECT OrderId, State, UpdatedAt FROM Orders
				WHERE DriverId = @scopeID AND UpdatedAt > @since
				ORDER BY UpdatedAt DESC LIMIT 100`
			params["scopeID"] = claims.UserID

		default:
			orderSQL = ""
		}

		if orderSQL != "" {
			iter := client.Single().Query(ctx, spanner.Statement{SQL: orderSQL, Params: params})
			for {
				row, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					slog.Error("sync.catchup.order_query_failed", "role", claims.Role, "err", err)
					break
				}
				var od OrderDelta
				var updatedAt time.Time
				if err := row.Columns(&od.OrderID, &od.State, &updatedAt); err != nil {
					continue
				}
				od.UpdatedAt = updatedAt.UTC().Format(time.RFC3339)
				resp.Orders = append(resp.Orders, od)
			}
			iter.Stop()
		}

		// ── Fleet deltas (SUPPLIER/ADMIN only) ──────────────────────────
		if claims.Role == "ADMIN" || claims.Role == "SUPPLIER" {
			supplierID := claims.ResolveSupplierID()
			fleetIter := client.Single().Query(ctx, spanner.Statement{
				SQL: `SELECT DriverId, COALESCE(TruckStatus, 'UNKNOWN'), COALESCE(IsOffline, false)
					FROM Drivers WHERE SupplierId = @sid AND UpdatedAt > @since LIMIT 100`,
				Params: map[string]interface{}{
					"sid":   supplierID,
					"since": since,
				},
			})
			for {
				row, err := fleetIter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					slog.Error("sync.catchup.fleet_query_failed", "supplier_id", supplierID, "err", err)
					break
				}
				var fd FleetDelta
				if err := row.Columns(&fd.DriverID, &fd.Status, &fd.IsOffline); err != nil {
					continue
				}
				resp.Fleet = append(resp.Fleet, fd)
			}
			fleetIter.Stop()
		}

		// ── Unread notification count ───────────────────────────────────
		countIter := client.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT COUNT(*) FROM Notifications
				WHERE RecipientId = @uid AND ReadAt IS NULL AND (DeletedAt IS NULL)`,
			Params: map[string]interface{}{"uid": claims.UserID},
		})
		row, err := countIter.Next()
		if err == nil {
			row.Columns(&resp.UnreadNotifications)
		}
		countIter.Stop()

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			fmt.Fprintf(w, `{"error":"encode: %v"}`, err)
		}
	}
}

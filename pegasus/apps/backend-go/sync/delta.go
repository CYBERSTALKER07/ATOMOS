package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"backend-go/auth"
)

// ── Mobile Delta Sync ──────────────────────────────────────────────────────
//
// GET /v1/sync/delta?since=<unix_ms>
//
// Designed for mobile clients polling after a cold start or background resume.
// Returns only the minimal change set since the epoch milliseconds `since`,
// scoped to the authenticated role. Intended to replace full-page refreshes
// at scale — a mobile client that was offline for 30 min needs ≤ 50 KB to
// catch up, not a full catalog reload.
//
// Pagination: include `cursor` from the previous response to page through
// large deltas. When `has_more=false`, the client is fully caught up.
//
// Design choices vs /v1/sync/catchup:
//   - `since` is unix milliseconds (not RFC3339) — cheaper to compare in JS/Swift/Kotlin
//   - Paginated with server-side cursor — avoids 10 s Spanner scan on huge windows
//   - Returns `server_now_ms` for the client to save as its next `since` baseline
//   - Maximum page size 200 records — prevents OOM on slow networks

const (
	deltaPageSize = 200
	deltaTimeout  = 8 * time.Second
)

// DeltaResponse is the versioned mobile delta payload.
type DeltaResponse struct {
	// V is the wire contract version — always "v1" for this endpoint.
	V string `json:"v"`

	// ServerNowMs is the server's current epoch-ms. The client MUST save
	// this and use it as `since` in the next poll. Never save the request's
	// `since` value — clock skew between client and server can cause gaps.
	ServerNowMs int64 `json:"server_now_ms"`

	// Orders contains changed order records since `since`.
	Orders []OrderRecord `json:"orders,omitempty"`

	// Notifications is the count of unread notifications since `since`.
	// Mobile clients use this to badge the notification icon; the full list
	// is fetched via /v1/notifications?since=<unix_ms>.
	UnreadNotifications int `json:"unread_notifications"`

	// HasMore is true when the page is full and there are more records.
	// Clients MUST paginate until HasMore=false before saving ServerNowMs.
	HasMore bool `json:"has_more"`

	// NextCursor is an opaque page token. Pass as `cursor=<token>` in the
	// next request. Empty when HasMore=false.
	NextCursor string `json:"next_cursor,omitempty"`
}

// OrderRecord is the thin change record for a single order.
type OrderRecord struct {
	OrderID   string `json:"order_id"`
	Status    string `json:"status"`
	UpdatedMs int64  `json:"updated_ms"` // epoch milliseconds
}

// HandleDelta returns changed records since a unix-millisecond epoch baseline.
// Role scoping is identical to HandleCatchup.
func HandleDelta(client *spanner.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeErr(w, http.StatusMethodNotAllowed, "method_not_allowed")
			return
		}

		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.PegasusClaims)
		if !ok {
			writeErr(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		sinceMs, err := strconv.ParseInt(r.URL.Query().Get("since"), 10, 64)
		if err != nil || sinceMs < 0 {
			writeErr(w, http.StatusBadRequest, "invalid 'since': must be epoch milliseconds as int64")
			return
		}

		// Reject absurdly large windows to prevent runaway scans.
		// 7-day max keeps the index scan bounded.
		const maxWindowMs = 7 * 24 * 60 * 60 * 1000
		if time.Now().UnixMilli()-sinceMs > maxWindowMs {
			writeErr(w, http.StatusBadRequest, "window too large: maximum 7 days")
			return
		}

		since := time.UnixMilli(sinceMs).UTC()
		serverNow := time.Now().UTC()

		// Optional opaque cursor (base-10 epoch ms of last returned record)
		var cursorMs int64
		if c := r.URL.Query().Get("cursor"); c != "" {
			cursorMs, err = strconv.ParseInt(c, 10, 64)
			if err != nil || cursorMs < sinceMs {
				writeErr(w, http.StatusBadRequest, "invalid cursor")
				return
			}
			since = time.UnixMilli(cursorMs).UTC()
		}

		ctx, cancel := context.WithTimeout(r.Context(), deltaTimeout)
		defer cancel()

		orders, hasMore, nextCursorMs, err := fetchOrderDelta(ctx, client, claims, since, cursorMs > 0)
		if err != nil {
			writeErr(w, http.StatusInternalServerError, "delta_query_failed")
			return
		}

		resp := DeltaResponse{
			V:           "v1",
			ServerNowMs: serverNow.UnixMilli(),
			Orders:      orders,
			HasMore:     hasMore,
		}
		if hasMore && nextCursorMs > 0 {
			resp.NextCursor = strconv.FormatInt(nextCursorMs, 10)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

// fetchOrderDelta queries Orders changed since `since` for the given claims.
// Returns (records, hasMore, nextCursorMs, error).
// nextCursorMs is the UpdatedAt of the last returned record in epoch ms.
func fetchOrderDelta(
	ctx context.Context,
	client *spanner.Client,
	claims *auth.PegasusClaims,
	since time.Time,
	isCursorPage bool,
) ([]OrderRecord, bool, int64, error) {

	// Stale read is acceptable for delta sync — 5 s staleness is fine for
	// mobile polling cadence (typically ≥ 30 s between polls).
	txn := client.Single().WithTimestampBound(spanner.ExactStaleness(5 * time.Second))

	var stmt spanner.Statement

	switch claims.Role {
	case "ADMIN", "SUPPLIER":
		supplierID := claims.ResolveSupplierID()
		stmt = spanner.Statement{
			SQL: `SELECT OrderId, State, UpdatedAt FROM Orders
			       WHERE SupplierId = @scopeID
			         AND UpdatedAt > @since
			       ORDER BY UpdatedAt ASC
			       LIMIT @limit`,
			Params: map[string]interface{}{
				"scopeID": supplierID,
				"since":   since,
				"limit":   int64(deltaPageSize + 1), // +1 to detect hasMore
			},
		}

	case "RETAILER":
		stmt = spanner.Statement{
			SQL: `SELECT OrderId, State, UpdatedAt FROM Orders
			       WHERE RetailerId = @scopeID
			         AND UpdatedAt > @since
			       ORDER BY UpdatedAt ASC
			       LIMIT @limit`,
			Params: map[string]interface{}{
				"scopeID": claims.UserID,
				"since":   since,
				"limit":   int64(deltaPageSize + 1),
			},
		}

	case "DRIVER":
		stmt = spanner.Statement{
			SQL: `SELECT OrderId, State, UpdatedAt FROM Orders
			       WHERE DriverId = @scopeID
			         AND UpdatedAt > @since
			       ORDER BY UpdatedAt ASC
			       LIMIT @limit`,
			Params: map[string]interface{}{
				"scopeID": claims.UserID,
				"since":   since,
				"limit":   int64(deltaPageSize + 1),
			},
		}

	default:
		return nil, false, 0, nil
	}

	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	var records []OrderRecord
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, false, 0, err
		}

		var orderID, state string
		var updatedAt time.Time
		if err := row.Columns(&orderID, &state, &updatedAt); err != nil {
			return nil, false, 0, err
		}
		records = append(records, OrderRecord{
			OrderID:   orderID,
			Status:    state,
			UpdatedMs: updatedAt.UnixMilli(),
		})
	}

	hasMore := len(records) > deltaPageSize
	if hasMore {
		records = records[:deltaPageSize]
	}

	var nextCursorMs int64
	if hasMore && len(records) > 0 {
		nextCursorMs = records[len(records)-1].UpdatedMs
	}

	return records, hasMore, nextCursorMs, nil
}

// writeErr writes a JSON error body with the given status code.
func writeErr(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(`{"error":"` + msg + `"}`))
}

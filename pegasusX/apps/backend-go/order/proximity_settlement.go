package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/uber/h3-go/v4"
)

// Settlement proximity: payment modes unlock only when driver is physically
// at the stop (design §4.1). Tighter than pack breach_radius_meters.
const (
	SettlementProximityRadiusM = 100.0
	// SettlementH3Resolution: res 9 ~174 m edge — cell match ≈ doorstep.
	SettlementH3Resolution = 9
	// MaxTelemetryAge for anti-spoof: location sample must be recent.
	SettlementTelemetryMaxAge = 2 * time.Minute
)

// ProximityMethod codes stored on Orders.ProximityMethod.
const (
	ProximityMethodH3          = "H3"
	ProximityMethodGeofence100 = "GEOFENCE_100M"
	ProximityMethodManual      = "MANUAL"
	ProximityMethodForceBypass = "FORCE_BYPASS"
)

var (
	// ErrProximityLocked: cash collect / credit leave blocked until unlock.
	ErrProximityLocked = errors.New("proximity_locked")
	// ErrProximityTelemetryStale: client timestamp too old (spoof guard).
	ErrProximityTelemetryStale = errors.New("proximity_telemetry_stale")
	// ErrProximityMismatch: neither H3 nor 100 m geofence satisfied.
	ErrProximityMismatch = errors.New("proximity_mismatch")
)

// ProximityUnlockRequest is POST /v1/delivery/proximity-unlock.
type ProximityUnlockRequest struct {
	OrderID   string  `json:"order_id"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	// ClientTimestamp RFC3339 — when the location sample was taken (offline replay).
	ClientTimestamp string `json:"client_timestamp,omitempty"`
	// ForceBypassToken optional supervisor token (supervised MANUAL/FORCE_BYPASS).
	ForceBypassToken string `json:"force_bypass_token,omitempty"`
	// Idempotency is via Idempotency-Key header (existing guard).
}

// ProximityUnlockResponse is the unlock wire shape.
type ProximityUnlockResponse struct {
	OrderID             string  `json:"order_id"`
	ProximityUnlocked   bool    `json:"proximity_unlocked"`
	ProximityMethod     string  `json:"proximity_method,omitempty"`
	DistanceM           float64 `json:"distance_m,omitempty"`
	UnlockedAt          string  `json:"unlocked_at,omitempty"`
	PaymentModesEnabled bool    `json:"payment_modes_enabled"`
	Message             string  `json:"message,omitempty"`
}

// SettlementH3Cell returns the resolution 9 H3 cell for doorstep settlement / perimeter checks.
func SettlementH3Cell(lat, lng float64) string {
	return proximity.SettlementH3Cell(lat, lng)
}

// EvaluateSettlementProximity returns method + distance if within settlement radius or H3 match.
// Pure function — no I/O. Used by unlock handler and tests.
func EvaluateSettlementProximity(driverLat, driverLng, orderLat, orderLng float64, orderH3Cell string) (method string, distanceM float64, ok bool) {
	if driverLat == 0 && driverLng == 0 {
		return "", 0, false
	}
	if orderLat == 0 && orderLng == 0 && strings.TrimSpace(orderH3Cell) == "" {
		return "", 0, false
	}

	// H3 cell match (preferred when order has cell).
	if strings.TrimSpace(orderH3Cell) != "" {
		driverCell := proximity.SettlementH3Cell(driverLat, driverLng)
		if driverCell != "" {
			// Accept exact match at settlement res or prefix match against stored cell.
			if driverCell == orderH3Cell || h3CellsCompatible(driverCell, orderH3Cell) {
				if orderLat != 0 || orderLng != 0 {
					distanceM = distanceMeters(driverLat, driverLng, orderLat, orderLng)
				}
				return ProximityMethodH3, distanceM, true
			}
		}
	}

	if orderLat != 0 || orderLng != 0 {
		distanceM = distanceMeters(driverLat, driverLng, orderLat, orderLng)
		if distanceM <= SettlementProximityRadiusM {
			return ProximityMethodGeofence100, distanceM, true
		}
	}
	return "", distanceM, false
}

func h3CellsCompatible(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	ca := h3.Cell(h3.IndexFromString(a))
	cb := h3.Cell(h3.IndexFromString(b))
	if !ca.IsValid() || !cb.IsValid() {
		return false
	}
	ra := ca.Resolution()
	rb := cb.Resolution()
	if ra == rb {
		return false
	}
	if ra > rb {
		parent, err := ca.Parent(rb)
		return err == nil && parent == cb
	}
	parent, err := cb.Parent(ra)
	return err == nil && parent == ca
}

// ValidateTelemetryFreshness rejects spoofed old timestamps.
// Empty clientTS → use server now (online path).
func ValidateTelemetryFreshness(clientTS string, now time.Time, maxAge time.Duration) error {
	if strings.TrimSpace(clientTS) == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339Nano, clientTS)
	if err != nil {
		t, err = time.Parse(time.RFC3339, clientTS)
		if err != nil {
			return fmt.Errorf("%w: invalid client_timestamp", ErrProximityTelemetryStale)
		}
	}
	age := now.Sub(t.UTC())
	if age < 0 {
		age = -age // clock skew forward
	}
	if age > maxAge {
		return fmt.Errorf("%w: age %s > max %s", ErrProximityTelemetryStale, age, maxAge)
	}
	return nil
}

// HandleProximityUnlock is POST /v1/delivery/proximity-unlock.
// Unlocks payment modes (cash / card / credit / split) for the stop.
func (s *Service) HandleProximityUnlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "proximity_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req ProximityUnlockRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id_required"})
		return
	}

	now := s.now()
	if err := ValidateTelemetryFreshness(req.ClientTimestamp, now, SettlementTelemetryMaxAge); err != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()

	var (
		method     string
		distanceM  float64
		unlockedAt time.Time
		already    bool
		retailerID string
		supplierID string
	)

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID}, []string{
			"Status", "Version", "DriverId", "RetailerId", "SupplierId", "Lat", "Lng", "H3Cell",
			"ProximityUnlockedAt", "ProximityMethod",
		})
		if err != nil {
			return fmt.Errorf("order %s not found: %w", req.OrderID, err)
		}
		var (
			status                              string
			version                             int64
			driverCol, retailerCol, supplierCol spanner.NullString
			orderLat, orderLng                  float64
			h3Cell                              spanner.NullString
			proxAt                              spanner.NullTime
			proxMethod                          spanner.NullString
		)
		if err := row.Columns(&status, &version, &driverCol, &retailerCol, &supplierCol,
			&orderLat, &orderLng, &h3Cell, &proxAt, &proxMethod); err != nil {
			return err
		}
		if !driverCol.Valid || driverCol.StringVal != driverID {
			return fmt.Errorf("driver is not assigned to order %s", req.OrderID)
		}
		// Unlock only on operational delivery legs before fiscal hard-gate.
		switch Status(status) {
		case StatusArrived, StatusShopClosedPending, StatusAwaitingPayment,
			StatusPendingCashCollection, StatusDeliveredOnCredit:
		default:
			return fmt.Errorf("proximity unlock not allowed in status %s", status)
		}
		if retailerCol.Valid {
			retailerID = retailerCol.StringVal
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}

		if proxAt.Valid {
			already = true
			unlockedAt = proxAt.Time.UTC()
			if proxMethod.Valid {
				method = proxMethod.StringVal
			}
			return nil
		}

		// Supervised force path.
		token := strings.TrimSpace(req.ForceBypassToken)
		if token != "" {
			// Reuse shop-closed bypass token table when present; else accept non-empty supervisor token
			// only if status is ARRIVED_SHOP_CLOSED with matching BypassToken on open attempt.
			okToken, tokErr := assertShopClosedBypassToken(ctx, txn, req.OrderID, token)
			if tokErr != nil {
				return tokErr
			}
			if !okToken {
				return fmt.Errorf("invalid force_bypass_token")
			}
			method = ProximityMethodForceBypass
			if orderLat != 0 || orderLng != 0 {
				distanceM = distanceMeters(req.Latitude, req.Longitude, orderLat, orderLng)
			}
		} else {
			cell := ""
			if h3Cell.Valid {
				cell = h3Cell.StringVal
			}
			var okProx bool
			method, distanceM, okProx = EvaluateSettlementProximity(req.Latitude, req.Longitude, orderLat, orderLng, cell)
			if !okProx {
				return fmt.Errorf("%w: %.0fm from stop (need ≤ %.0fm or H3 match)", ErrProximityMismatch, distanceM, SettlementProximityRadiusM)
			}
		}

		unlockedAt = now.UTC()
		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, req.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventProximityUnlocked, Timestamp: unlockedAt.Format(time.RFC3339Nano)},
			OrderID:    req.OrderID,
			DriverID:   driverID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			Status:     method,
		}); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{
			spanner.UpdateMap("Orders", map[string]any{
				"OrderId":             req.OrderID,
				"ProximityUnlockedAt": unlockedAt,
				"ProximityMethod":     method,
				"Version":             version + 1,
				"UpdatedAt":           unlockedAt,
			}),
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		msg := err.Error()
		code := http.StatusConflict
		if errors.Is(err, ErrProximityMismatch) || strings.Contains(msg, "proximity_mismatch") {
			code = http.StatusForbidden
		}
		writeJSON(w, code, map[string]string{"error": msg})
		return
	}

	s.invalidateOrderCache(ctx, req.OrderID)
	if !already {
		s.broadcastShopClosed(ctx, supplierID, retailerID, driverID, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventProximityUnlocked, Timestamp: unlockedAt.Format(time.RFC3339Nano)},
			OrderID:    req.OrderID,
			DriverID:   driverID,
			RetailerID: retailerID,
			SupplierID: supplierID,
			Status:     method,
		})
	}

	msg := "Payment modes unlocked."
	if already {
		msg = "Proximity already unlocked."
	}
	resp := ProximityUnlockResponse{
		OrderID:             req.OrderID,
		ProximityUnlocked:   true,
		ProximityMethod:     method,
		DistanceM:           distanceM,
		UnlockedAt:          unlockedAt.Format(time.RFC3339Nano),
		PaymentModesEnabled: true,
		Message:             msg,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func assertShopClosedBypassToken(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID, token string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT BypassToken FROM ShopClosedAttempts
		      WHERE OrderId = @oid AND Resolution = 'BYPASS_ISSUED'
		      ORDER BY ReportedAt DESC LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := txn.Query(ctx, stmt)
	row, err := iter.Next()
	iter.Stop()
	if err != nil {
		return false, nil
	}
	var stored spanner.NullString
	if err := row.Columns(&stored); err != nil {
		return false, err
	}
	return stored.Valid && stored.StringVal == token, nil
}

// requireProximityUnlocked fails if settlement proximity is not open.
// forceBypassToken allows supervised credit leave without prior unlock.
func (s *Service) requireProximityUnlocked(ctx context.Context, orderID, forceBypassToken string) error {
	if s.spannerClient == nil {
		// Unit tests without Spanner: fail-open only when no client (legacy harness).
		return nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{orderID},
		[]string{"ProximityUnlockedAt", "ProximityMethod"})
	if err != nil {
		return fmt.Errorf("%w: order read failed", ErrProximityLocked)
	}
	var proxAt spanner.NullTime
	var method spanner.NullString
	if err := row.Columns(&proxAt, &method); err != nil {
		return err
	}
	if proxAt.Valid {
		return nil
	}
	token := strings.TrimSpace(forceBypassToken)
	if token == "" {
		return ErrProximityLocked
	}
	// Token path: verify in a read-only query.
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		ok, e := assertShopClosedBypassToken(ctx, txn, orderID, token)
		if e != nil {
			return e
		}
		if !ok {
			return ErrProximityLocked
		}
		return nil
	})
	if err != nil {
		return ErrProximityLocked
	}
	return nil
}

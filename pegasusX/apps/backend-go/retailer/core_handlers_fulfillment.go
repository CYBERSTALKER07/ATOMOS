package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"github.com/pegasusx/pegasusx/apps/backend-go/proximity"
	"github.com/pegasusx/pegasusx/apps/backend-go/routing"
	"github.com/pegasusx/pegasusx/apps/backend-go/telemetry"
	"google.golang.org/api/iterator"
)

func (s *Service) HandleAIPredictions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	limit := 25
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "limit must be a positive integer"})
			return
		}
		limit = parsed
	}
	items, err := s.orders.ListRetailerAIPredictions(r.Context(), retailerID, limit)
	if err != nil {
		s.log.Warn("retailer ai predictions read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_ai_predictions_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// HandleConfirmAIOrder confirms a retailer AI preorder.
func (s *Service) HandleConfirmAIOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.ConfirmAIOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.ConfirmAIOrder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleRejectAIOrder rejects a retailer AI preorder.
func (s *Service) HandleRejectAIOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.RejectAIOrderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectAIOrder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleEditPreorder edits a retailer draft preorder.
func (s *Service) HandleEditPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.EditPreorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.EditPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleConfirmPreorder confirms a retailer draft preorder.
func (s *Service) HandleConfirmPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.ConfirmPreorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.ConfirmPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleAcceptDeliveryProposal applies a warehouse-proposed delivery date.
func (s *Service) HandleAcceptDeliveryProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.AcceptDeliveryProposalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.AcceptDeliveryProposal(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleRejectDeliveryProposal cancels an order when the retailer declines a warehouse date proposal.
func (s *Service) HandleRejectDeliveryProposal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.RejectDeliveryProposalRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectDeliveryProposal(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

// HandleRejectPreorder lets a retailer cancel a draft or scheduled manual pre-order.
func (s *Service) HandleRejectPreorder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	body, ok := readLimitedBody(w, r, 64*1024)
	if !ok {
		return
	}
	if s.guardIdempotencyStrict(w, r, body) {
		return
	}
	if s.orders == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "order_lifecycle_unavailable"})
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()
	var req order.RejectPreorderRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	resp, err := s.orders.RejectPreorder(r.Context(), retailerID, req)
	if err != nil {
		writeRetailerOrderLifecycleError(w, err)
		return
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
	s.saveIdempotency(r.Context(), r, body, http.StatusOK, respBytes)
}

func writeRetailerOrderLifecycleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, order.ErrOrderNotFound):
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
	case errors.Is(err, order.ErrOrderForbidden):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	case errors.Is(err, order.ErrInvalidStatusTransition):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_status_transition"})
	case errors.Is(err, order.ErrDeliveryProposalRequired):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "delivery_proposal_required"})
	case errors.Is(err, order.ErrDeliveryProposalPending):
		writeJSON(w, http.StatusConflict, map[string]string{"error": "delivery_proposal_pending"})
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
	}
}

// HandlePendingPayments returns pending post-offload payment sessions.
func (s *Service) HandlePendingPayments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	orders, err := s.listRetailerTrackingOrders(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer pending payments read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_pending_payments_failed"})
		return
	}
	pending := filterTrackingOrders(orders, isPendingPaymentStatus)
	sessions := make([]map[string]any, 0, len(pending))
	for i := range pending {
		row := map[string]any{
			"order_id":      pending[i].OrderID,
			"retailer_id":   retailerID,
			"supplier_id":   pending[i].SupplierID,
			"locked_amount": pending[i].TotalMinor,
			"currency":      pending[i].Currency,
			"status":        pending[i].Status,
			"created_at":    pending[i].CreatedAt,
		}
		if s.paymentSessionByOrder != nil {
			sessionID, gateway, ok, err := s.paymentSessionByOrder(r.Context(), pending[i].OrderID)
			if err != nil {
				s.log.Warn("pending payment session lookup failed", "order_id", pending[i].OrderID, "err", err)
			} else if ok {
				if sessionID != "" {
					row["session_id"] = sessionID
				}
				if gateway != "" {
					row["gateway"] = gateway
				}
			}
		}
		sessions = append(sessions, row)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":           "pending",
		"pending":          sessions,
		"pending_payments": sessions,
		"count":            len(sessions),
	})
}

// HandleActiveFulfillment returns active fulfillment list for retailer UI.
func (s *Service) HandleActiveFulfillment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	orders, err := s.listRetailerTrackingOrders(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer active fulfillment read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_active_fulfillment_failed"})
		return
	}
	fulfillments := make([]map[string]any, 0, len(orders))
	for i := range orders {
		fulfillments = append(fulfillments, mobileActiveFulfillment(orders[i]))
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"status":       "active",
		"fulfillments": fulfillments,
		"count":        len(fulfillments),
	})
}

// HandleTracking returns active assigned-order tracking projections.
func (s *Service) HandleTracking(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	retailerID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	orders, err := s.listRetailerTrackingOrders(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer tracking read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_tracking_failed"})
		return
	}

	receipts, err := s.listRetailerRecentReceipts(r.Context(), retailerID)
	if err != nil {
		s.log.Warn("retailer recent receipts read failed", "retailer_id", retailerID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_receipts_failed"})
		return
	}

	merged := mergeTrackingOrders(orders, receipts)
	events := deriveTrackingEvents(merged)

	status := "idle"
	if len(orders) > 0 {
		status = "active"
	}

	mobileOrders := make([]map[string]any, 0, len(orders))
	for i := range orders {
		mobileOrders = append(mobileOrders, mobileTrackingOrder(orders[i]))
	}

	mobileReceipts := make([]map[string]any, 0, len(receipts))
	for i := range receipts {
		mobileReceipts = append(mobileReceipts, mobileTrackingOrder(receipts[i]))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"status":          status,
		"orders":          mobileOrders,
		"recent_receipts": mobileReceipts,
		"events":          events,
	})
}

func (s *Service) listRetailerTrackingOrders(ctx context.Context, retailerID string) ([]TrackingOrder, error) {
	const pageSize = 500
	all := make([]TrackingOrder, 0)
	for offset := 0; ; offset += pageSize {
		batch, err := s.repo.ListTrackingOrders(ctx, retailerID, pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, batch...)
		if len(batch) < pageSize {
			break
		}
	}
	return normalizeTrackingOrders(s.attachRouteGeometry(ctx, s.attachLiveLocations(ctx, all))), nil
}

func (s *Service) listRetailerRecentReceipts(ctx context.Context, retailerID string) ([]TrackingOrder, error) {
	receipts, err := s.repo.ListRecentReceipts(ctx, retailerID, recentReceiptLimit)
	if err != nil {
		return nil, err
	}
	return normalizeTrackingOrders(receipts), nil
}

func mergeTrackingOrders(primary []TrackingOrder, secondary []TrackingOrder) []TrackingOrder {
	if len(primary) == 0 && len(secondary) == 0 {
		return []TrackingOrder{}
	}
	merged := make([]TrackingOrder, 0, len(primary)+len(secondary))
	seen := make(map[string]struct{}, len(primary)+len(secondary))
	appendUnique := func(orders []TrackingOrder) {
		for _, order := range orders {
			orderID := strings.TrimSpace(order.OrderID)
			if orderID != "" {
				if _, ok := seen[orderID]; ok {
					continue
				}
				seen[orderID] = struct{}{}
			}
			merged = append(merged, order)
		}
	}
	appendUnique(primary)
	appendUnique(secondary)
	return merged
}

func normalizeTrackingOrders(orders []TrackingOrder) []TrackingOrder {
	if len(orders) == 0 {
		return []TrackingOrder{}
	}
	return orders
}

func filterTrackingOrders(orders []TrackingOrder, keep func(TrackingOrder) bool) []TrackingOrder {
	if len(orders) == 0 {
		return []TrackingOrder{}
	}
	filtered := make([]TrackingOrder, 0, len(orders))
	for _, order := range orders {
		if keep(order) {
			filtered = append(filtered, order)
		}
	}
	if len(filtered) == 0 {
		return []TrackingOrder{}
	}
	return filtered
}

func isPendingPaymentStatus(order TrackingOrder) bool {
	switch strings.TrimSpace(order.Status) {
	case "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION":
		return true
	default:
		return false
	}
}

func deriveTrackingEvents(orders []TrackingOrder) []TrackingEvent {
	if len(orders) == 0 {
		return []TrackingEvent{}
	}
	events := make([]TrackingEvent, 0, len(orders)*2)
	for _, order := range orders {
		createdAt := strings.TrimSpace(order.CreatedAt)
		updatedAt := strings.TrimSpace(order.UpdatedAt)
		status := strings.TrimSpace(order.Status)
		if createdAt != "" {
			events = append(events, TrackingEvent{
				EventType:  TrackingEventOrderCreated,
				OrderID:    order.OrderID,
				OccurredAt: createdAt,
				Derived:    true,
				Source:     trackingEventSourceOrderRow,
			})
		}
		if updatedAt == "" || status == "" {
			continue
		}
		if updatedAt == createdAt && status == "PENDING" {
			continue
		}
		if updatedAt == createdAt && createdAt == "" {
			continue
		}
		events = append(events, TrackingEvent{
			EventType:  TrackingEventOrderStatusSnapshot,
			OrderID:    order.OrderID,
			Status:     status,
			OccurredAt: updatedAt,
			Derived:    true,
			Source:     trackingEventSourceOrderRow,
		})
	}
	sort.SliceStable(events, func(i, j int) bool {
		left := parseTrackingEventTime(events[i].OccurredAt)
		right := parseTrackingEventTime(events[j].OccurredAt)
		if left.Equal(right) {
			if events[i].OrderID == events[j].OrderID {
				return events[i].EventType < events[j].EventType
			}
			return events[i].OrderID < events[j].OrderID
		}
		return left.After(right)
	})
	if len(events) == 0 {
		return []TrackingEvent{}
	}
	return events
}

func parseTrackingEventTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return time.Time{}
	}
	return parsed
}

type trackingLocationLookup struct {
	location telemetry.DriverLocation
	found    bool
}

func (s *Service) attachLiveLocations(ctx context.Context, orders []TrackingOrder) []TrackingOrder {
	if s.locations == nil || len(orders) == 0 {
		for i := range orders {
			if strings.TrimSpace(orders[i].DriverID) == "" {
				orders[i].LocationFreshness = "AWAITING_TELEMETRY"
			} else {
				orders[i].LocationFreshness = "AWAITING_TELEMETRY"
			}
			orders[i].LiveLocationAvailable = false
		}
		return orders
	}
	now := s.now()
	lookups := make(map[string]trackingLocationLookup, len(orders))
	for i := range orders {
		orders[i].LiveLocationAvailable = false
		orders[i].DriverLocation = nil
		orders[i].LocationFreshness = "AWAITING_TELEMETRY"
		driverID := strings.TrimSpace(orders[i].DriverID)
		if driverID == "" {
			continue
		}
		lookup, ok := lookups[driverID]
		if !ok {
			location, found, err := s.locations.GetDriverLocation(ctx, driverID)
			if err != nil {
				s.log.Warn("retailer tracking location read failed", "driver_id", driverID, "err", err)
				continue
			}
			lookup = trackingLocationLookup{location: location, found: found}
			lookups[driverID] = lookup
		}
		if !lookup.found {
			continue
		}
		if strings.TrimSpace(lookup.location.SupplierID) != strings.TrimSpace(orders[i].SupplierID) {
			continue
		}
		// G3.C: always surface last-known GPS with honesty labels (live vs stale).
		orders[i].DriverLocation = trackingLocationFromTelemetry(lookup.location)
		if lookup.location.IsLive(now) {
			orders[i].LiveLocationAvailable = true
			orders[i].LocationFreshness = "LIVE"
			if orders[i].DeliveryLat != 0 && orders[i].DeliveryLng != 0 {
				driverLat := lookup.location.Lat
				driverLng := lookup.location.Lng
				if driverLat == 0 && lookup.location.Latitude != 0 {
					driverLat = lookup.location.Latitude
				}
				if driverLng == 0 && lookup.location.Longitude != 0 {
					driverLng = lookup.location.Longitude
				}
				distKm := proximity.HaversineDistance(driverLat, driverLng, orders[i].DeliveryLat, orders[i].DeliveryLng)
				if proximity.WithinDeliveryApproachForSupplier(ctx, orders[i].SupplierID, distKm) {
					orders[i].IsApproaching = true
				}
			}
		} else {
			orders[i].LiveLocationAvailable = false
			orders[i].LocationFreshness = "LAST_KNOWN"
		}
	}
	return orders
}

func trackingLocationFromTelemetry(location telemetry.DriverLocation) *TrackingLocation {
	return &TrackingLocation{
		DriverID:          location.DriverID,
		SupplierID:        location.SupplierID,
		Lat:               location.Lat,
		Lng:               location.Lng,
		Latitude:          location.Latitude,
		Longitude:         location.Longitude,
		Velocity:          location.Velocity,
		Heading:           location.Heading,
		ReportedAt:        location.ReportedAt.UTC().Format(time.RFC3339Nano),
		ReceivedAt:        location.ReceivedAt.UTC().Format(time.RFC3339Nano),
		StaleAfterSeconds: location.StaleAfterSeconds,
	}
}

// attachRouteGeometry hydrates planned polylines from SupplierTruckManifests (fail-open).
func (s *Service) attachRouteGeometry(ctx context.Context, orders []TrackingOrder) []TrackingOrder {
	if s == nil || s.spannerClient == nil || len(orders) == 0 {
		return orders
	}
	manifestIDs := make([]string, 0, len(orders))
	seen := make(map[string]struct{}, len(orders))
	for i := range orders {
		mid := strings.TrimSpace(orders[i].ManifestID)
		if mid == "" {
			continue
		}
		if _, ok := seen[mid]; ok {
			continue
		}
		seen[mid] = struct{}{}
		manifestIDs = append(manifestIDs, mid)
	}
	if len(manifestIDs) == 0 {
		return orders
	}
	stmt := spanner.Statement{
		SQL: `SELECT ManifestId, RouteId, EncodedRoutePolyline, RouteGeometrySource, StopCount
		      FROM SupplierTruckManifests
		      WHERE ManifestId IN UNNEST(@ids)
		        AND EncodedRoutePolyline IS NOT NULL
		        AND EncodedRoutePolyline != ''`,
		Params: map[string]any{"ids": manifestIDs},
	}
	iter := s.spannerClient.Single().
		WithTimestampBound(spanner.ExactStaleness(15*time.Second)).
		Query(ctx, stmt)
	defer iter.Stop()

	byManifest := make(map[string]routing.RouteGeometryWire, len(manifestIDs))
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.Warn("retailer tracking route geometry query failed", "err", err)
			return orders
		}
		var manifestID, routeID string
		var encoded, source spanner.NullString
		var stopCount int64
		if err := row.Columns(&manifestID, &routeID, &encoded, &source, &stopCount); err != nil {
			s.log.Warn("retailer tracking route geometry scan failed", "err", err)
			continue
		}
		if !encoded.Valid || encoded.StringVal == "" {
			continue
		}
		geometry, decodeErr := routing.GeometryFromStoredPolyline(
			routeID,
			encoded.StringVal,
			source.StringVal,
			int(stopCount),
		)
		if decodeErr != nil {
			continue
		}
		wire := routing.ToRouteGeometryWire(geometry)
		byManifest[strings.TrimSpace(manifestID)] = wire
	}
	for i := range orders {
		mid := strings.TrimSpace(orders[i].ManifestID)
		if mid == "" {
			continue
		}
		if wire, ok := byManifest[mid]; ok {
			copy := wire
			orders[i].RouteGeometry = &copy
		}
	}
	return orders
}

func retailerIDFromRequest(r *http.Request) (string, error) {
	if claims, ok := auth.FromContext(r.Context()); ok {
		// Retail OS v2: tenant id is RetailerOrgID; legacy v1 uses Subject as retailer id.
		orgID := auth.ResolveRetailerOrgID(claims)
		if orgID != "" {
			pathRetailerID := strings.TrimSpace(chi.URLParam(r, "retailerID"))
			if pathRetailerID != "" && pathRetailerID != orgID {
				return "", errRetailerScopeMismatch
			}
			queryRetailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
			if queryRetailerID != "" && queryRetailerID != orgID {
				return "", errRetailerScopeMismatch
			}
			return orgID, nil
		}
	}
	if id := strings.TrimSpace(chi.URLParam(r, "retailerID")); id != "" {
		return id, nil
	}
	if id := strings.TrimSpace(r.URL.Query().Get("retailer_id")); id != "" {
		return id, nil
	}
	return "", errRetailerUnauthorized
}

func writeRetailerIdentityError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errRetailerScopeMismatch):
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
	default:
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
}

// HandleCheckShelfAlerts handles POST /v1/retailer/shelf/check
func (s *Service) HandleCheckShelfAlerts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "shelf_alerts_unavailable"})
		return
	}
	var req struct {
		LocationID        string `json:"location_id"`
		GlobalProductID   string `json:"global_product_id"`
		CurrentStock      int64  `json:"current_stock"`
		CapacityThreshold int64  `json:"capacity_threshold"`
	}
	body, _ := io.ReadAll(io.LimitReader(r.Body, 16*1024))
	_ = json.Unmarshal(body, &req)

	locID := strings.TrimSpace(req.LocationID)
	if locID == "" {
		locID = strings.TrimSpace(claims.ActiveLocationID)
	}

	if locID != "" && req.GlobalProductID != "" && req.CapacityThreshold > 0 && req.CurrentStock <= req.CapacityThreshold {
		alertID := uuid.New().String()
		_, err := s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return txn.BufferWrite([]*spanner.Mutation{
				spanner.InsertOrUpdateMap("RetailerShelfAlerts", map[string]any{
					"AlertId":           alertID,
					"RetailerId":        orgID,
					"LocationId":        locID,
					"GlobalProductId":   req.GlobalProductID,
					"Status":            "OPEN",
					"CurrentStock":      req.CurrentStock,
					"CapacityThreshold": req.CapacityThreshold,
					"CreatedAt":         spanner.CommitTimestamp,
				}),
			})
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed_to_save_alert"})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"status": "alert_created", "alert_id": alertID})
		return
	}

	// Query open alerts for this retailer
	sql := `SELECT AlertId, LocationId, GlobalProductId, Status, CurrentStock, CapacityThreshold, CreatedAt
	        FROM RetailerShelfAlerts
	        WHERE RetailerId = @rid AND Status = 'OPEN'
	        ORDER BY CreatedAt DESC LIMIT 50`
	params := map[string]any{"rid": orgID}
	iter := s.spannerClient.Single().WithTimestampBound(spanner.ExactStaleness(15*time.Second)).Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	type alertRow struct {
		AlertId           string    `json:"alert_id"`
		LocationId        string    `json:"location_id"`
		GlobalProductId   string    `json:"global_product_id"`
		Status            string    `json:"status"`
		CurrentStock      int64     `json:"current_stock"`
		CapacityThreshold int64     `json:"capacity_threshold"`
		CreatedAt         time.Time `json:"created_at"`
	}
	alerts := make([]alertRow, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "query_alerts_failed"})
			return
		}
		var a alertRow
		if err := row.Columns(&a.AlertId, &a.LocationId, &a.GlobalProductId, &a.Status, &a.CurrentStock, &a.CapacityThreshold, &a.CreatedAt); err != nil {
			continue
		}
		alerts = append(alerts, a)
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok", "alerts": alerts})
}

// HandleResolveShelfAlert handles POST /v1/retailer/shelf/alerts/{alertID}/resolve
func (s *Service) HandleResolveShelfAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	_, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	alertID := strings.TrimSpace(chi.URLParam(r, "alertID"))
	if alertID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "alert_id_required"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "shelf_alerts_unavailable"})
		return
	}

	_, err = s.spannerClient.ReadWriteTransaction(r.Context(), func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{
			SQL:    `SELECT LocationId FROM RetailerShelfAlerts WHERE RetailerId = @rid AND AlertId = @aid`,
			Params: map[string]any{"rid": orgID, "aid": alertID},
		}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()
		row, err := iter.Next()
		if err == iterator.Done {
			return errors.New("alert_not_found")
		}
		if err != nil {
			return err
		}
		var locID string
		if err := row.Columns(&locID); err != nil {
			return err
		}
		return txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("RetailerShelfAlerts", map[string]any{
				"RetailerId": orgID,
				"LocationId": locID,
				"AlertId":    alertID,
				"Status":     "RESOLVED",
				"ResolvedAt": spanner.CommitTimestamp,
			}),
		})
	})
	if err != nil {
		if err.Error() == "alert_not_found" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "alert_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "resolve_alert_failed"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "resolved", "alert_id": alertID})
}

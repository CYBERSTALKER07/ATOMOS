package order

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
)

const (
	cacheKeyPaymentBypassPrefix = "payment_bypass:"
	cacheKeyEarlyCompletePrefix = "early_complete:"
	paymentBypassCacheTTL       = 24 * time.Hour
	earlyCompleteCacheTTL       = 6 * time.Hour
)

type paymentBypassRecord struct {
	Token      string `json:"token"`
	SupplierID string `json:"supplier_id"`
	IssuedBy   string `json:"issued_by"`
	Reason     string `json:"reason,omitempty"`
	IssuedAt   string `json:"issued_at"`
}

type earlyCompleteRecord struct {
	OrderIDs []string `json:"order_ids"`
	Reason   string   `json:"reason,omitempty"`
	Note     string   `json:"note,omitempty"`
	DriverID string   `json:"driver_id"`
}

// HandleIssuePaymentBypass serves POST /v1/supplier/orders/payment-bypass.
func (s *Service) HandleIssuePaymentBypass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment_bypass_unavailable"})
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

	var req struct {
		OrderID string `json:"order_id"`
		Reason  string `json:"reason"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id required"})
		return
	}

	supplierID, ok := auth.ResolveSupplierID(r.Context())
	if !ok {
		supplierID = s.resolveSupplierScope(r.Context())
	}
	ctx := r.Context()
	now := s.now()

	var status string
	err = s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL:    `SELECT Status FROM Orders WHERE OrderId = @oid AND SupplierId = @sid`,
		Params: map[string]any{"oid": req.OrderID, "sid": supplierID},
	}).Do(func(row *spanner.Row) error {
		return row.Columns(&status)
	})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
		return
	}
	if status != string(StatusAwaitingPayment) {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "order_must_be_awaiting_payment"})
		return
	}

	token, err := generatePaymentBypassToken()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "token_generation_failed"})
		return
	}

	rec := paymentBypassRecord{
		Token:      token,
		SupplierID: supplierID,
		IssuedBy:   strings.TrimSpace(claims.Subject),
		Reason:     strings.TrimSpace(req.Reason),
		IssuedAt:   now.Format(time.RFC3339Nano),
	}
	raw, _ := json.Marshal(rec)
	if err := s.cache.Set(ctx, cacheKeyPaymentBypassPrefix+req.OrderID, raw, paymentBypassCacheTTL); err != nil {
		s.log.ErrorContext(ctx, "payment bypass cache set failed", "order_id", req.OrderID, "err", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cache_write_failed"})
		return
	}

	s.log.InfoContext(ctx, "payment bypass issued", "order_id", req.OrderID, "supplier_id", supplierID)
	resp := map[string]any{
		"status":       "bypass_issued",
		"bypass_token": token,
		"order_id":     req.OrderID,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleConfirmPaymentBypass serves POST /v1/delivery/confirm-payment-bypass.
func (s *Service) HandleConfirmPaymentBypass(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "payment_bypass_unavailable"})
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

	var req struct {
		OrderID     string `json:"order_id"`
		BypassToken string `json:"bypass_token"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	req.BypassToken = strings.TrimSpace(req.BypassToken)
	if req.OrderID == "" || req.BypassToken == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and bypass_token required"})
		return
	}

	ctx := r.Context()
	raw, found, err := s.cache.Get(ctx, cacheKeyPaymentBypassPrefix+req.OrderID)
	if err != nil || !found || len(raw) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "bypass_not_found"})
		return
	}
	var rec paymentBypassRecord
	if err := json.Unmarshal(raw, &rec); err != nil || rec.Token != req.BypassToken {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "invalid_bypass_token"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	// Gate-0: never write COMPLETED from bypass — open fiscal gate (ADR-009).
	if err := s.ConfirmPaymentBypass(ctx, req.OrderID, driverID, rec.SupplierID); err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "order_not_found"})
			return
		}
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "order_forbidden"})
			return
		}
		if errors.Is(err, ErrInvalidStatusTransition) {
			writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.cache.Invalidate(ctx, cacheKeyPaymentBypassPrefix+req.OrderID)
	s.invalidateOrderCache(ctx, req.OrderID)
	resp := map[string]any{"status": "FISCALIZING", "order_id": req.OrderID}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// HandleApproveEarlyComplete serves POST /v1/supplier/route/approve-early-complete.
func (s *Service) HandleApproveEarlyComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "early_complete_unavailable"})
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

	var req struct {
		DriverID string `json:"driver_id"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.DriverID = strings.TrimSpace(req.DriverID)
	if req.DriverID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "driver_id required"})
		return
	}

	ctx := r.Context()
	raw, found, err := s.cache.Get(ctx, cacheKeyEarlyCompletePrefix+req.DriverID)
	if err != nil || !found || len(raw) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_pending_early_complete"})
		return
	}
	var rec earlyCompleteRecord
	if err := json.Unmarshal(raw, &rec); err != nil || len(rec.OrderIDs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_pending_early_complete"})
		return
	}

	supplierID, _ := auth.ResolveSupplierID(ctx)
	if supplierID == "" {
		supplierID = s.resolveSupplierScope(ctx)
	}
	// Gate-0: early-complete must open fiscalization, never jump to COMPLETED.
	approved := 0
	for _, orderID := range rec.OrderIDs {
		orderID = strings.TrimSpace(orderID)
		if orderID == "" {
			continue
		}
		o, found, gerr := s.repo.GetOrder(ctx, orderID)
		if gerr != nil || !found {
			continue
		}
		if strings.TrimSpace(o.SupplierID) != supplierID {
			continue
		}
		if o.Status != StatusAwaitingPayment {
			continue
		}
		if err := s.beginFiscalFromAwaitingPayment(ctx, o, "EARLY_COMPLETE", "early_complete_approved", nil); err != nil {
			s.log.WarnContext(ctx, "early complete fiscal open failed", "order_id", orderID, "err", err)
			continue
		}
		approved++
	}
	s.cache.Invalidate(ctx, cacheKeyEarlyCompletePrefix+req.DriverID)
	resp := map[string]any{
		"status":          "approved",
		"driver_id":       req.DriverID,
		"orders_approved": approved,
		"next_status":     "FISCALIZING",
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

// StoreEarlyCompleteRequest caches a driver early-complete request for supplier approval.
func (s *Service) StoreEarlyCompleteRequest(ctx context.Context, rec earlyCompleteRecord) error {
	if s.cache == nil {
		return errors.New("cache_unavailable")
	}
	rec.DriverID = strings.TrimSpace(rec.DriverID)
	if rec.DriverID == "" {
		return errors.New("driver_id required")
	}
	raw, err := json.Marshal(rec)
	if err != nil {
		return err
	}
	return s.cache.Set(ctx, cacheKeyEarlyCompletePrefix+rec.DriverID, raw, earlyCompleteCacheTTL)
}

// HandleRequestEarlyComplete serves POST /v1/fleet/route/request-early-complete.
func (s *Service) HandleRequestEarlyComplete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil || s.cache == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "early_complete_unavailable"})
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

	var req struct {
		RouteID string `json:"route_id"`
		Reason  string `json:"reason"`
		Note    string `json:"note"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	ctx := r.Context()
	var orderIDs []string
	err = s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT OrderId FROM Orders
		      WHERE DriverId = @did AND Status NOT IN ('COMPLETED', 'CANCELLED')
		      ORDER BY CreatedAt DESC`,
		Params: map[string]any{"did": driverID},
	}).Do(func(row *spanner.Row) error {
		var oid string
		if err := row.Columns(&oid); err != nil {
			return err
		}
		orderIDs = append(orderIDs, oid)
		return nil
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "load_route_orders_failed"})
		return
	}
	if len(orderIDs) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "no_active_orders"})
		return
	}

	rec := earlyCompleteRecord{
		OrderIDs: orderIDs,
		Reason:   strings.TrimSpace(req.Reason),
		Note:     strings.TrimSpace(req.Note),
		DriverID: driverID,
	}
	if err := s.StoreEarlyCompleteRequest(ctx, rec); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "cache_write_failed"})
		return
	}
	resp := map[string]any{
		"status":      "REQUESTED",
		"order_count": len(orderIDs),
		"order_ids":   orderIDs,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func generatePaymentBypassToken() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(900000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()+100000), nil
}

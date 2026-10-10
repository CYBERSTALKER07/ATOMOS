package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

var (
	errRetailerPriceOverrideNotFound     = errors.New("retailer price override not found")
	errRetailerPriceOverrideAccessDenied = errors.New("retailer price override access denied")
)

type createRetailerPriceOverrideRequest struct {
	RetailerID string `json:"retailer_id"`
	ProductID  string `json:"product_id"`
	SkuID      string `json:"sku_id"`
	Price      int64  `json:"price"`
	Currency   string `json:"currency"`
	Notes      string `json:"notes"`
	ExpiresAt  string `json:"expires_at"`
}

type retailerPriceOverrideResponse struct {
	OverrideID string  `json:"override_id"`
	SupplierID string  `json:"supplier_id"`
	RetailerID string  `json:"retailer_id"`
	ProductID  string  `json:"product_id"`
	Price      int64   `json:"price"`
	SetBy      string  `json:"set_by"`
	SetByRole  string  `json:"set_by_role"`
	IsActive   bool    `json:"is_active"`
	Notes      string  `json:"notes,omitempty"`
	ExpiresAt  *string `json:"expires_at,omitempty"`
	CreatedAt  string  `json:"created_at"`
}

// HandleRetailerPricingOverrides serves list and create for per-retailer prices.
func (s *Service) HandleRetailerPricingOverrides(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		s.listRetailerPriceOverrides(w, r)
	case http.MethodPost:
		s.createRetailerPriceOverride(w, r)
	default:
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
	}
}

// HandleRetailerPricingOverrideDelete deactivates one override by id.
func (s *Service) HandleRetailerPricingOverrideDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	overrideID := strings.TrimSpace(chi.URLParam(r, "overrideID"))
	if overrideID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "override_id_required"})
		return
	}
	s.deactivateRetailerPriceOverride(w, r, overrideID)
}

func (s *Service) listRetailerPriceOverrides(w http.ResponseWriter, r *http.Request) {
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pricing_unavailable"})
		return
	}
	sid := s.scopedSupplierID(r)
	retailerID := strings.TrimSpace(r.URL.Query().Get("retailer_id"))
	productID := strings.TrimSpace(r.URL.Query().Get("product_id"))
	if productID == "" {
		productID = strings.TrimSpace(r.URL.Query().Get("sku_id"))
	}

	sql := `SELECT OverrideId, SupplierId, RetailerId, ProductId, OverridePrice,
	               SetBy, SetByRole, IsActive, Notes, ExpiresAt, CreatedAt
	        FROM RetailerPricingOverrides
	        WHERE SupplierId = @supplierId AND IsActive = true`
	params := map[string]any{"supplierId": sid}
	if retailerID != "" {
		sql += ` AND RetailerId = @retailerId`
		params["retailerId"] = retailerID
	}
	if productID != "" {
		sql += ` AND ProductId = @productId`
		params["productId"] = productID
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT 500`

	iter := s.portalSpanner.Single().Query(r.Context(), spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	overrides := make([]retailerPriceOverrideResponse, 0)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.ErrorContext(r.Context(), "list retailer price overrides failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_overrides_failed"})
			return
		}
		item, err := scanRetailerPriceOverrideRow(row)
		if err != nil {
			s.log.ErrorContext(r.Context(), "parse retailer price override row failed", "err", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_overrides_failed"})
			return
		}
		overrides = append(overrides, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"overrides": overrides, "total": len(overrides)})
}

func (s *Service) createRetailerPriceOverride(w http.ResponseWriter, r *http.Request) {
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pricing_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || strings.TrimSpace(claims.Subject) == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, ok := readMutationBody(w, r, 32*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	var req createRetailerPriceOverrideRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}

	productID := strings.TrimSpace(req.ProductID)
	if productID == "" {
		productID = strings.TrimSpace(req.SkuID)
	}
	if req.RetailerID == "" || productID == "" || req.Price <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "retailer_id_product_id_and_positive_price_required"})
		return
	}

	sid := s.scopedSupplierID(r)
	ctx := r.Context()
	if err := s.ensureSupplierProduct(ctx, sid, productID); err != nil {
		if errors.Is(err, errProductNotFound) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "product_not_found"})
			return
		}
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "validate_product_failed"})
		return
	}

	// Resolve the supplier's canonical currency and validate the override matches.
	supplierCurrency := s.currency
	if supplierCurrency == "" {
		supplierCurrency = "UZS"
	}
	reqCurrency := strings.ToUpper(strings.TrimSpace(req.Currency))
	if reqCurrency == "" {
		reqCurrency = supplierCurrency
	}
	if reqCurrency != supplierCurrency {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error":             "currency_mismatch",
			"supplier_currency": supplierCurrency,
			"provided_currency": reqCurrency,
		})
		return
	}

	expiresAt, err := parseOptionalRFC3339(req.ExpiresAt)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expires_at_must_be_rfc3339"})
		return
	}

	overrideID := uuid.NewString()
	setBy := strings.TrimSpace(claims.Subject)
	setByRole := string(claims.Role)

	_, err = s.portalSpanner.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		if err := deactivateActiveOverrides(txnCtx, txn, sid, req.RetailerID, productID); err != nil {
			return err
		}
		cols := []string{"OverrideId", "SupplierId", "RetailerId", "ProductId", "OverridePrice", "Currency", "SetBy", "SetByRole", "IsActive", "CreatedAt", "UpdatedAt"}
		vals := []any{overrideID, sid, req.RetailerID, productID, req.Price, reqCurrency, setBy, setByRole, true, spanner.CommitTimestamp, spanner.CommitTimestamp}
		if strings.TrimSpace(req.Notes) != "" {
			cols = append(cols, "Notes")
			vals = append(vals, strings.TrimSpace(req.Notes))
		}
		if expiresAt.Valid {
			cols = append(cols, "ExpiresAt")
			vals = append(vals, expiresAt.Time)
		}

		buf := &supplierSpannerTxnBuf{}
		event := events.RetailerPriceOverrideEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventRetailerPriceOverride,
				Timestamp: s.now().UTC().Format(time.RFC3339Nano),
			},
			OverrideID: overrideID,
			SupplierID: sid,
			RetailerID: req.RetailerID,
			ProductID:  productID,
			PriceMinor: req.Price,
			Action:     "CREATED",
			SetBy:      setBy,
			SetByRole:  setByRole,
		}
		if err := outbox.EmitJSON(txnCtx, buf, events.AggregateRetailerPriceOverride, overrideID, events.TopicMain, event); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{spanner.Insert("RetailerPricingOverrides", cols, vals)}
		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		s.log.ErrorContext(ctx, "create retailer price override failed", "err", err, "retailer_id", req.RetailerID, "product_id", productID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "create_override_failed"})
		return
	}
	s.invalidateRetailerPricingCaches(ctx, sid, req.RetailerID)

	resp := map[string]any{
		"status":      "OVERRIDE_ACTIVE",
		"override_id": overrideID,
		"retailer_id": req.RetailerID,
		"product_id":  productID,
		"price":       req.Price,
	}
	respBytes, _ := json.Marshal(resp)
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusCreated, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(respBytes)
}

func (s *Service) deactivateRetailerPriceOverride(w http.ResponseWriter, r *http.Request, overrideID string) {
	if s.portalSpanner == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "pricing_unavailable"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	body, ok := readMutationBody(w, r, 4*1024)
	if !ok {
		return
	}
	key, handled := s.guardMutationReplay(w, r, body)
	if handled {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseMutationReplay(r.Context(), r)
		}
	}()

	sid := s.scopedSupplierID(r)
	ctx := r.Context()

	var retailerID, productID string
	var price int64
	_, err := s.portalSpanner.ReadWriteTransaction(ctx, func(txnCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txnCtx, "RetailerPricingOverrides", spanner.Key{overrideID},
			[]string{"SupplierId", "RetailerId", "ProductId", "OverridePrice"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return errRetailerPriceOverrideNotFound
			}
			return fmt.Errorf("read override %s: %w", overrideID, err)
		}
		var ownerSid string
		if err := row.Columns(&ownerSid, &retailerID, &productID, &price); err != nil {
			return fmt.Errorf("parse override %s: %w", overrideID, err)
		}
		if ownerSid != sid {
			return errRetailerPriceOverrideAccessDenied
		}

		buf := &supplierSpannerTxnBuf{}
		event := events.RetailerPriceOverrideEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventRetailerPriceOverride,
				Timestamp: s.now().UTC().Format(time.RFC3339Nano),
			},
			OverrideID: overrideID,
			SupplierID: sid,
			RetailerID: retailerID,
			ProductID:  productID,
			PriceMinor: price,
			Action:     "DEACTIVATED",
			SetBy:      strings.TrimSpace(claims.Subject),
			SetByRole:  string(claims.Role),
		}
		if err := outbox.EmitJSON(txnCtx, buf, events.AggregateRetailerPriceOverride, overrideID, events.TopicMain, event); err != nil {
			return err
		}
		mutations := []*spanner.Mutation{spanner.Update("RetailerPricingOverrides",
			[]string{"OverrideId", "IsActive", "UpdatedAt"},
			[]any{overrideID, false, spanner.CommitTimestamp},
		)}
		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		switch {
		case errors.Is(err, errRetailerPriceOverrideAccessDenied):
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		case errors.Is(err, errRetailerPriceOverrideNotFound):
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "override_not_found"})
		default:
			s.log.ErrorContext(ctx, "deactivate retailer price override failed", "err", err, "override_id", overrideID)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "deactivate_override_failed"})
		}
		return
	}
	s.invalidateRetailerPricingCaches(ctx, sid, retailerID)
	respBytes, _ := json.Marshal(map[string]string{"status": "deactivated", "override_id": overrideID})
	idemCommitted = true
	s.saveMutationReplay(r.Context(), key, body, http.StatusOK, respBytes)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(respBytes)
}

var errProductNotFound = errors.New("product not found")

func (s *Service) ensureSupplierProduct(ctx context.Context, supplierID, productID string) error {
	row, err := s.portalSpanner.Single().ReadRow(ctx, "Products", spanner.Key{productID}, []string{"SupplierId", "IsActive"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return errProductNotFound
		}
		return err
	}
	var owner string
	var active bool
	if err := row.Columns(&owner, &active); err != nil {
		return err
	}
	if owner != supplierID || !active {
		return errProductNotFound
	}
	return nil
}

func deactivateActiveOverrides(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID, retailerID, productID string) error {
	stmt := spanner.Statement{
		SQL: `SELECT OverrideId FROM RetailerPricingOverrides
		      WHERE SupplierId = @sid AND RetailerId = @rid AND ProductId = @pid AND IsActive = true`,
		Params: map[string]any{"sid": supplierID, "rid": retailerID, "pid": productID},
	}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()
	mutations := make([]*spanner.Mutation, 0, 1)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}
		var oldID string
		if err := row.Columns(&oldID); err != nil {
			return err
		}
		mutations = append(mutations, spanner.Update("RetailerPricingOverrides",
			[]string{"OverrideId", "IsActive", "UpdatedAt"},
			[]any{oldID, false, spanner.CommitTimestamp},
		))
	}
	if len(mutations) == 0 {
		return nil
	}
	return txn.BufferWrite(mutations)
}

func scanRetailerPriceOverrideRow(row *spanner.Row) (retailerPriceOverrideResponse, error) {
	var item retailerPriceOverrideResponse
	var notes spanner.NullString
	var expiresAt spanner.NullTime
	var createdAt time.Time
	if err := row.Columns(
		&item.OverrideID, &item.SupplierID, &item.RetailerID, &item.ProductID, &item.Price,
		&item.SetBy, &item.SetByRole, &item.IsActive, &notes, &expiresAt, &createdAt,
	); err != nil {
		return retailerPriceOverrideResponse{}, err
	}
	if notes.Valid {
		item.Notes = notes.StringVal
	}
	if expiresAt.Valid {
		formatted := expiresAt.Time.UTC().Format(time.RFC3339Nano)
		item.ExpiresAt = &formatted
	}
	if !createdAt.IsZero() {
		item.CreatedAt = createdAt.UTC().Format(time.RFC3339Nano)
	}
	return item, nil
}

func parseOptionalRFC3339(raw string) (spanner.NullTime, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return spanner.NullTime{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return spanner.NullTime{}, err
	}
	return spanner.NullTime{Time: parsed.UTC(), Valid: true}, nil
}

func (s *Service) invalidateRetailerPricingCaches(ctx context.Context, supplierID, retailerID string) {
	if s.cache == nil {
		return
	}
	s.cache.Invalidate(ctx,
		"promotions:supplier:"+supplierID,
		"catalog:products:"+supplierID,
		"pricing:overrides:"+supplierID+":"+retailerID,
	)
}

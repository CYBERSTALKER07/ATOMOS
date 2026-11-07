package supplier

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
)

// SupplierServicePolicy is the durable commitment contract published by a supplier.
type SupplierServicePolicy struct {
	SupplierID             string    `json:"supplier_id"`
	LeadTimeDays           int64     `json:"lead_time_days"`
	SameDayCutoffTime      string    `json:"same_day_cutoff_time,omitempty"`
	NextDayCutoffTime      string    `json:"next_day_cutoff_time,omitempty"`
	MinOrderMinor          int64     `json:"min_order_minor"`
	Currency               string    `json:"currency"`
	FillRateGuaranteeBps   int64     `json:"fill_rate_guarantee_bps"`
	AllowScheduledDelivery bool      `json:"allow_scheduled_delivery"`
	MaxScheduleAdvanceDays int64     `json:"max_schedule_advance_days"`
	AssignedManagerName    string    `json:"assigned_manager_name,omitempty"`
	AssignedManagerPhone   string    `json:"assigned_manager_phone,omitempty"`
	UpdatedAt              time.Time `json:"updated_at"`
	UpdatedByUserID        string    `json:"updated_by_user_id,omitempty"`
}

// OrderServicePromiseSnapshot is the immutable snapshot of the promise made for an order.
type OrderServicePromiseSnapshot struct {
	OrderID                string    `json:"order_id"`
	SupplierID             string    `json:"supplier_id"`
	RetailerID             string    `json:"retailer_id"`
	WarehouseID            string    `json:"warehouse_id"`
	PromiseType            string    `json:"promise_type"` // SAME_DAY, NEXT_DAY, SCHEDULED
	GuaranteedDeliveryDate time.Time `json:"guaranteed_delivery_date"`
	CutoffAppliedAt        time.Time `json:"cutoff_applied_at,omitempty"`
	FillRateTargetBps      int64     `json:"fill_rate_target_bps"`
	MinOrderMinor          int64     `json:"min_order_minor"`
	Currency               string    `json:"currency"`
	SLAHours               int64     `json:"sla_hours"`
	Status                 string    `json:"status"` // PENDING, MET, BREACHED
	BreachedAt             time.Time `json:"breached_at,omitempty"`
	BreachReason           string    `json:"breach_reason,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

// PromiseEvaluationRequest is sent when calculating delivery promise at checkout or cart view.
type PromiseEvaluationRequest struct {
	SupplierID            string `json:"supplier_id"`
	RetailerID            string `json:"retailer_id"`
	WarehouseID           string `json:"warehouse_id"`
	TotalMinor            int64  `json:"total_minor"`
	Currency              string `json:"currency"`
	RequestedDeliveryDate string `json:"requested_delivery_date,omitempty"`
}

// PromiseEvaluationResult is the evaluated delivery guarantee.
type PromiseEvaluationResult struct {
	Eligible               bool      `json:"eligible"`
	PromiseType            string    `json:"promise_type"`
	GuaranteedDeliveryDate time.Time `json:"guaranteed_delivery_date"`
	SLAHours               int64     `json:"sla_hours"`
	FillRateTargetBps      int64     `json:"fill_rate_target_bps"`
	MinOrderMinor          int64     `json:"min_order_minor"`
	Currency               string    `json:"currency"`
	CutoffTime             string    `json:"cutoff_time,omitempty"`
	Reason                 string    `json:"reason,omitempty"`
}

// GetServicePolicy loads the policy for a supplier, or default if unconfigured.
func (s *Service) GetServicePolicy(ctx context.Context, supplierID string) (SupplierServicePolicy, error) {
	supplierID = strings.TrimSpace(supplierID)
	if supplierID == "" {
		return SupplierServicePolicy{}, errors.New("supplier_id_required")
	}

	pack, ok := auth.ResolveMarketPack(auth.DefaultMarketCode)
	defaultCurrency := "UZS"
	if ok && pack.CurrencyCode != "" {
		defaultCurrency = pack.CurrencyCode
	}

	defaultPolicy := SupplierServicePolicy{
		SupplierID:             supplierID,
		LeadTimeDays:           1,
		SameDayCutoffTime:      "14:00",
		NextDayCutoffTime:      "18:00",
		MinOrderMinor:          0,
		Currency:               defaultCurrency,
		FillRateGuaranteeBps:   9500,
		AllowScheduledDelivery: true,
		MaxScheduleAdvanceDays: 14,
		UpdatedAt:              time.Now().UTC(),
	}

	if s.portalSpanner == nil {
		return defaultPolicy, nil
	}

	stmt := spanner.Statement{
		SQL: `SELECT SupplierId, LeadTimeDays, SameDayCutoffTime, NextDayCutoffTime,
		             MinOrderMinor, Currency, FillRateGuaranteeBps, AllowScheduledDelivery,
		             MaxScheduleAdvanceDays, AssignedManagerName, AssignedManagerPhone,
		             UpdatedAt, UpdatedByUserId
		      FROM SupplierServicePolicies
		      WHERE SupplierId = @supplier_id`,
		Params: map[string]any{"supplier_id": supplierID},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err == iterator.Done {
		return defaultPolicy, nil
	}
	if err != nil {
		return defaultPolicy, fmt.Errorf("read service policy: %w", err)
	}

	var p SupplierServicePolicy
	var sameDayCutoff, nextDayCutoff, mgrName, mgrPhone, updatedBy spanner.NullString
	if err := row.Columns(
		&p.SupplierID, &p.LeadTimeDays, &sameDayCutoff, &nextDayCutoff,
		&p.MinOrderMinor, &p.Currency, &p.FillRateGuaranteeBps, &p.AllowScheduledDelivery,
		&p.MaxScheduleAdvanceDays, &mgrName, &mgrPhone, &p.UpdatedAt, &updatedBy,
	); err != nil {
		return defaultPolicy, err
	}

	p.SameDayCutoffTime = sameDayCutoff.StringVal
	p.NextDayCutoffTime = nextDayCutoff.StringVal
	p.AssignedManagerName = mgrName.StringVal
	p.AssignedManagerPhone = mgrPhone.StringVal
	p.UpdatedByUserID = updatedBy.StringVal

	return p, nil
}

// UpsertServicePolicy creates or replaces the supplier service policy.
func (s *Service) UpsertServicePolicy(ctx context.Context, p SupplierServicePolicy, actorID string) error {
	p.SupplierID = strings.TrimSpace(p.SupplierID)
	if p.SupplierID == "" {
		return errors.New("supplier_id_required")
	}
	if p.LeadTimeDays < 0 {
		return errors.New("lead_time_days_invalid")
	}
	if p.MinOrderMinor < 0 {
		return errors.New("min_order_minor_invalid")
	}
	if p.FillRateGuaranteeBps < 0 || p.FillRateGuaranteeBps > 10000 {
		return errors.New("fill_rate_guarantee_bps_invalid")
	}

	pack, ok := auth.ResolveMarketPack(auth.DefaultMarketCode)
	if ok && pack.CurrencyCode != "" {
		if p.Currency == "" {
			p.Currency = pack.CurrencyCode
		} else if p.Currency != pack.CurrencyCode {
			return fmt.Errorf("currency_mismatch: expected %s, got %s", pack.CurrencyCode, p.Currency)
		}
	} else if p.Currency == "" {
		p.Currency = "UZS"
	}

	if s.portalSpanner == nil {
		return nil
	}

	_, err := s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mut := spanner.InsertOrUpdateMap("SupplierServicePolicies", map[string]any{
			"SupplierId":             p.SupplierID,
			"LeadTimeDays":           p.LeadTimeDays,
			"SameDayCutoffTime":      spanner.NullString{StringVal: p.SameDayCutoffTime, Valid: p.SameDayCutoffTime != ""},
			"NextDayCutoffTime":      spanner.NullString{StringVal: p.NextDayCutoffTime, Valid: p.NextDayCutoffTime != ""},
			"MinOrderMinor":          p.MinOrderMinor,
			"Currency":               p.Currency,
			"FillRateGuaranteeBps":   p.FillRateGuaranteeBps,
			"AllowScheduledDelivery": p.AllowScheduledDelivery,
			"MaxScheduleAdvanceDays": p.MaxScheduleAdvanceDays,
			"AssignedManagerName":    spanner.NullString{StringVal: p.AssignedManagerName, Valid: p.AssignedManagerName != ""},
			"AssignedManagerPhone":   spanner.NullString{StringVal: p.AssignedManagerPhone, Valid: p.AssignedManagerPhone != ""},
			"UpdatedAt":              spanner.CommitTimestamp,
			"UpdatedByUserId":        spanner.NullString{StringVal: actorID, Valid: actorID != ""},
		})
		return txn.BufferWrite([]*spanner.Mutation{mut})
	})
	if err != nil {
		return fmt.Errorf("upsert service policy: %w", err)
	}

	if s.cache != nil {
		s.cache.Invalidate(ctx, supplierCacheKey(p.SupplierID), "supplier:service_policy:"+p.SupplierID)
	}
	return nil
}

// EvaluatePromise calculates the delivery commitment based on the supplier policy and current time.
func (s *Service) EvaluatePromise(ctx context.Context, req PromiseEvaluationRequest, now time.Time) (PromiseEvaluationResult, error) {
	policy, err := s.GetServicePolicy(ctx, req.SupplierID)
	if err != nil {
		return PromiseEvaluationResult{}, err
	}

	loc, locErr := auth.TimezoneFromContext(ctx, req.SupplierID)
	if locErr != nil || loc == nil {
		loc = time.UTC
	}
	localNow := now.In(loc)

	result := PromiseEvaluationResult{
		Eligible:          true,
		FillRateTargetBps: policy.FillRateGuaranteeBps,
		MinOrderMinor:     policy.MinOrderMinor,
		Currency:          policy.Currency,
	}

	if req.TotalMinor > 0 && req.TotalMinor < policy.MinOrderMinor {
		result.Eligible = false
		result.Reason = fmt.Sprintf("Order total %d %s is below minimum order requirement of %d %s",
			req.TotalMinor, policy.Currency, policy.MinOrderMinor, policy.Currency)
		return result, nil
	}

	// If a specific scheduled date was requested
	if req.RequestedDeliveryDate != "" {
		reqDate, err := time.Parse(time.RFC3339, req.RequestedDeliveryDate)
		if err != nil {
			reqDate, err = time.Parse("2006-01-02", req.RequestedDeliveryDate)
		}
		if err == nil {
			if !policy.AllowScheduledDelivery {
				result.Eligible = false
				result.Reason = "Scheduled delivery is disabled by supplier policy"
				return result, nil
			}
			maxAdvance := time.Duration(policy.MaxScheduleAdvanceDays*24) * time.Hour
			if reqDate.After(localNow.Add(maxAdvance)) {
				result.Eligible = false
				result.Reason = fmt.Sprintf("Scheduled delivery cannot exceed %d days in advance", policy.MaxScheduleAdvanceDays)
				return result, nil
			}
			if reqDate.Before(localNow) {
				result.Eligible = false
				result.Reason = "Requested delivery date cannot be in the past"
				return result, nil
			}
			result.PromiseType = "SCHEDULED"
			result.GuaranteedDeliveryDate = reqDate
			result.SLAHours = int64(reqDate.Sub(localNow).Hours())
			return result, nil
		}
	}

	// Parse Same-Day Cutoff
	sameDayPossible := false
	if policy.SameDayCutoffTime != "" {
		parts := strings.Split(policy.SameDayCutoffTime, ":")
		if len(parts) == 2 {
			h, _ := strconv.Atoi(parts[0])
			m, _ := strconv.Atoi(parts[1])
			cutoff := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), h, m, 0, 0, loc)
			if localNow.Before(cutoff) {
				sameDayPossible = true
				result.PromiseType = "SAME_DAY"
				result.GuaranteedDeliveryDate = time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 20, 0, 0, 0, loc)
				result.SLAHours = 8
				result.CutoffTime = policy.SameDayCutoffTime
				return result, nil
			}
		}
	}

	// Next-Day evaluation
	if !sameDayPossible {
		result.PromiseType = "NEXT_DAY"
		leadDays := policy.LeadTimeDays
		if leadDays <= 0 {
			leadDays = 1
		}
		deliveryDay := localNow.Add(time.Duration(leadDays*24) * time.Hour)
		result.GuaranteedDeliveryDate = time.Date(deliveryDay.Year(), deliveryDay.Month(), deliveryDay.Day(), 18, 0, 0, 0, loc)
		result.SLAHours = leadDays * 24
		result.CutoffTime = policy.NextDayCutoffTime
	}

	return result, nil
}

// SnapshotPromiseTxn persists the promise snapshot into Spanner and emits an outbox event.
func (s *Service) SnapshotPromiseTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, snap OrderServicePromiseSnapshot) error {
	if snap.OrderID == "" || snap.SupplierID == "" || snap.RetailerID == "" {
		return errors.New("missing_required_promise_fields")
	}

	now := time.Now().UTC()
	if snap.CreatedAt.IsZero() {
		snap.CreatedAt = now
	}
	if snap.UpdatedAt.IsZero() {
		snap.UpdatedAt = now
	}
	if snap.Status == "" {
		snap.Status = "PENDING"
	}

	mut := spanner.InsertMap("OrderServicePromiseSnapshots", map[string]any{
		"OrderId":                snap.OrderID,
		"SupplierId":             snap.SupplierID,
		"RetailerId":             snap.RetailerID,
		"WarehouseId":            snap.WarehouseID,
		"PromiseType":            snap.PromiseType,
		"GuaranteedDeliveryDate": snap.GuaranteedDeliveryDate,
		"CutoffAppliedAt":        spanner.NullTime{Time: snap.CutoffAppliedAt, Valid: !snap.CutoffAppliedAt.IsZero()},
		"FillRateTargetBps":      snap.FillRateTargetBps,
		"MinOrderMinor":          snap.MinOrderMinor,
		"Currency":               snap.Currency,
		"SLAHours":               snap.SLAHours,
		"Status":                 snap.Status,
		"BreachedAt":             spanner.NullTime{Time: snap.BreachedAt, Valid: !snap.BreachedAt.IsZero()},
		"BreachReason":           spanner.NullString{StringVal: snap.BreachReason, Valid: snap.BreachReason != ""},
		"CreatedAt":              spanner.CommitTimestamp,
		"UpdatedAt":              spanner.CommitTimestamp,
	})

	if err := txn.BufferWrite([]*spanner.Mutation{mut}); err != nil {
		return err
	}

	if buf != nil {
		eventPayload := events.SupplierServicePromiseEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventSupplierServicePromiseCreated,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			OrderID:                snap.OrderID,
			SupplierID:             snap.SupplierID,
			RetailerID:             snap.RetailerID,
			WarehouseID:            snap.WarehouseID,
			PromiseType:            snap.PromiseType,
			GuaranteedDeliveryDate: snap.GuaranteedDeliveryDate.Format(time.RFC3339),
			FillRateTargetBps:      snap.FillRateTargetBps,
			MinOrderMinor:          snap.MinOrderMinor,
			Currency:               snap.Currency,
			SLAHours:               snap.SLAHours,
			Status:                 snap.Status,
		}
		if !snap.CutoffAppliedAt.IsZero() {
			eventPayload.CutoffAppliedAt = snap.CutoffAppliedAt.Format(time.RFC3339)
		}
		return outbox.EmitJSON(ctx, buf, events.AggregateOrder, snap.OrderID, events.TopicMain, eventPayload)
	}

	return nil
}

// RecordPromiseBreach marks an order promise as breached.
func (s *Service) RecordPromiseBreach(ctx context.Context, orderID, reason string) error {
	orderID = strings.TrimSpace(orderID)
	if orderID == "" {
		return errors.New("order_id_required")
	}

	if s.portalSpanner == nil {
		return nil
	}

	now := time.Now().UTC()
	var supplierID, retailerID string
	var currentStatus string

	_, err := s.portalSpanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		buf := &supplierSpannerTxnBuf{}
		row, err := txn.ReadRow(ctx, "OrderServicePromiseSnapshots", spanner.Key{orderID},
			[]string{"SupplierId", "RetailerId", "Status"})
		if err != nil {
			if spanner.ErrCode(err) == codes.NotFound {
				return errors.New("promise_snapshot_not_found")
			}
			return err
		}
		if err := row.Columns(&supplierID, &retailerID, &currentStatus); err != nil {
			return err
		}

		if currentStatus == "BREACHED" {
			return nil // Already breached
		}

		mut := spanner.UpdateMap("OrderServicePromiseSnapshots", map[string]any{
			"OrderId":      orderID,
			"Status":       "BREACHED",
			"BreachedAt":   spanner.CommitTimestamp,
			"BreachReason": spanner.NullString{StringVal: strings.TrimSpace(reason), Valid: reason != ""},
			"UpdatedAt":    spanner.CommitTimestamp,
		})

		eventPayload := events.SupplierServicePromiseEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventSupplierServicePromiseBreached,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			OrderID:      orderID,
			SupplierID:   supplierID,
			RetailerID:   retailerID,
			Status:       "BREACHED",
			BreachedAt:   now.Format(time.RFC3339),
			BreachReason: strings.TrimSpace(reason),
		}

		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, eventPayload); err != nil {
			return err
		}

		mutations := []*spanner.Mutation{mut}
		for _, e := range buf.events {
			mutations = append(mutations, portalOutboxMutation(e))
		}

		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return fmt.Errorf("record promise breach: %w", err)
	}

	if s.cache != nil && supplierID != "" {
		s.cache.Invalidate(ctx, supplierCacheKey(supplierID), "supplier:promises:"+supplierID)
	}
	return nil
}

// HandleGetServicePolicy serves GET /v1/supplier/service-policy.
func (s *Service) HandleGetServicePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	policy, err := s.GetServicePolicy(r.Context(), supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

// HandleUpsertServicePolicy serves PUT /v1/supplier/service-policy.
func (s *Service) HandleUpsertServicePolicy(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	var req SupplierServicePolicy
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.SupplierID = supplierID

	actorID := strings.TrimSpace(auth.ActorFromContext(r.Context()))
	if actorID == "unknown" {
		actorID = ""
	}
	if err := s.UpsertServicePolicy(r.Context(), req, actorID); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	policy, err := s.GetServicePolicy(r.Context(), supplierID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, policy)
}

// HandleEvaluateServicePromise serves GET /v1/retailer/service-promise.
func (s *Service) HandleEvaluateServicePromise(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}

	var req PromiseEvaluationRequest
	if r.Method == http.MethodPost {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
			return
		}
	} else {
		req.SupplierID = strings.TrimSpace(r.URL.Query().Get("supplier_id"))
		req.RetailerID = strings.TrimSpace(r.URL.Query().Get("retailer_id"))
		req.WarehouseID = strings.TrimSpace(r.URL.Query().Get("warehouse_id"))
		req.Currency = strings.TrimSpace(r.URL.Query().Get("currency"))
		req.RequestedDeliveryDate = strings.TrimSpace(r.URL.Query().Get("requested_delivery_date"))
		if totalStr := strings.TrimSpace(r.URL.Query().Get("total_minor")); totalStr != "" {
			req.TotalMinor, _ = strconv.ParseInt(totalStr, 10, 64)
		}
	}

	if req.SupplierID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "supplier_id_required"})
		return
	}

	res, err := s.EvaluatePromise(r.Context(), req, time.Now().UTC())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

// HandleListBreachedPromises serves GET /v1/supplier/service-promises/breaches.
func (s *Service) HandleListBreachedPromises(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	supplierID := strings.TrimSpace(s.scopedSupplierID(r))
	if supplierID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}

	if s.portalSpanner == nil {
		writeJSON(w, http.StatusOK, map[string]any{"breaches": []any{}, "total": 0})
		return
	}

	limit := 50
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil && parsed > 0 && parsed <= 500 {
			limit = parsed
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId, WarehouseId, PromiseType,
		             GuaranteedDeliveryDate, FillRateTargetBps, MinOrderMinor, Currency,
		             SLAHours, Status, BreachedAt, BreachReason, CreatedAt
		      FROM OrderServicePromiseSnapshots
		      WHERE SupplierId = @supplier_id AND Status = 'BREACHED'
		      ORDER BY BreachedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{
			"supplier_id": supplierID,
			"limit":       int64(limit),
		},
	}
	iter := s.portalSpanner.Single().Query(ctx, stmt)
	defer iter.Stop()

	var breaches []OrderServicePromiseSnapshot
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		var b OrderServicePromiseSnapshot
		var breachedAt spanner.NullTime
		var breachReason spanner.NullString
		if err := row.Columns(
			&b.OrderID, &b.SupplierID, &b.RetailerID, &b.WarehouseID, &b.PromiseType,
			&b.GuaranteedDeliveryDate, &b.FillRateTargetBps, &b.MinOrderMinor, &b.Currency,
			&b.SLAHours, &b.Status, &breachedAt, &breachReason, &b.CreatedAt,
		); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		b.BreachedAt = breachedAt.Time
		b.BreachReason = breachReason.StringVal
		breaches = append(breaches, b)
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"breaches": breaches,
		"total":    len(breaches),
	})
}

package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/order"
	"google.golang.org/api/iterator"
)

// AutoOrderPlacedOrder is one procurement order created in place mode.
type AutoOrderPlacedOrder struct {
	OrderID    string   `json:"order_id"`
	SupplierID string   `json:"supplier_id,omitempty"`
	LineCount  int      `json:"line_count"`
	TotalMinor int64    `json:"total_minor"`
	SKUs       []string `json:"skus,omitempty"`
}

// AutoOrderRun is an audit record for one worker tick per retailer.
type AutoOrderRun struct {
	RunID           string                `json:"run_id"`
	RetailerID      string                `json:"retailer_id"`
	StartedAt       string                `json:"started_at"`
	FinishedAt      string                `json:"finished_at,omitempty"`
	Mode            string                `json:"mode"`
	DraftLines      int                   `json:"draft_lines"`
	PlacedLines     int                   `json:"placed_lines,omitempty"`
	PlacedOrders    []AutoOrderPlacedOrder `json:"placed_orders,omitempty"`
	Skipped         []AutoOrderSkip       `json:"skipped"`
	Status          string                `json:"status"`
	Message         string                `json:"message,omitempty"`
	Suggestions     int                   `json:"suggestions_seen"`
	ScheduleBucket  string                `json:"schedule_bucket"`
	CandidateSource string                `json:"candidate_source,omitempty"` // seed | reorder_suggestions | ai_predictions
}

// AutoOrderSkip records why a candidate was not ordered.
type AutoOrderSkip struct {
	SKU    string `json:"sku,omitempty"`
	Reason string `json:"reason"`
}

// AutoOrderCandidate is a line the worker might act on.
type AutoOrderCandidate struct {
	SKU        string   `json:"sku"`
	ProductID  string   `json:"product_id,omitempty"`
	SupplierID string   `json:"supplier_id,omitempty"`
	CategoryID string   `json:"category_id,omitempty"`
	Qty        int64    `json:"qty"`
	Name       string   `json:"name,omitempty"`
	Sources    []string `json:"sources,omitempty"` // L3.5 from reorder suggestions
	// Inventory-grounded metadata (optional).
	IP           float64 `json:"ip,omitempty"`
	ReorderPoint float64 `json:"reorder_point,omitempty"`
	OrderUpTo    float64 `json:"order_up_to,omitempty"`
	Confidence   float64 `json:"confidence,omitempty"`
	Reason       string  `json:"reason,omitempty"`
}

type autoOrderWorkerState struct {
	mu         sync.Mutex
	runs       []AutoOrderRun
	bucketDone map[string]bool // retailer|day|sku — in-process idempotency
}

func (s *Service) aoWorker() *autoOrderWorkerState {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.autoOrderWorker == nil {
		s.autoOrderWorker = &autoOrderWorkerState{bucketDone: map[string]bool{}}
	}
	return s.autoOrderWorker
}

// HandleAutoOrderRun serves POST /v1/retailer/settings/auto-order/run
// Query: mode=off|shadow|draft|place (default settings.execution_mode, else draft).
// place creates real orders when OrderCreator is wired and AUTO_ORDER_PLACE_ENABLED (or test).
func (s *Service) HandleAutoOrderRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermOrderPlace) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermOrderPlace})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	rawMode := strings.TrimSpace(r.URL.Query().Get("mode"))
	mode := NormalizeExecutionMode(rawMode)
	if rawMode != "" && mode == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "invalid_mode", "allowed": "shadow,draft,place",
		})
		return
	}
	if mode == "" {
		settings := s.loadAutoOrderDurable(r.Context(), orgID)
		mode = NormalizeExecutionMode(settings.ExecutionMode)
		if mode == "" || mode == AutoOrderModeOff {
			mode = AutoOrderModeDraft
		}
	}
	if mode != AutoOrderModeShadow && mode != AutoOrderModeDraft && mode != AutoOrderModePlace {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{
			"error": "invalid_mode", "allowed": "shadow,draft,place",
		})
		return
	}
	if mode == AutoOrderModePlace {
		role := auth.EffectiveRetailerRole(claims)
		if role != "OWNER" && role != "ADMIN" && role != "MANAGER" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "place_requires_manager"})
			return
		}
	}
	run := s.RunAutoOrderForRetailer(r.Context(), orgID, mode)
	writeJSON(w, http.StatusOK, run)
}

// HandleAutoOrderRuns serves GET /v1/retailer/settings/auto-order/runs
// Returns the last N audit runs for the retailer (newest first).
// Prefers Spanner RetailerAutoOrderRuns when available; falls back to in-process memory.
func (s *Service) HandleAutoOrderRuns(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || !auth.HasRetailerPerm(claims, auth.PermOrderPlace) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden", "permission": auth.PermOrderPlace})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	limit := 20
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		if n, e := strconv.Atoi(raw); e == nil {
			limit = n
			if limit < 1 {
				limit = 1
			}
			if limit > 50 {
				limit = 50
			}
		}
	}
	items := s.listAutoOrderRuns(r.Context(), orgID, limit)
	if items == nil {
		items = []AutoOrderRun{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

func (s *Service) listAutoOrderRuns(ctx context.Context, orgID string, limit int) []AutoOrderRun {
	if limit <= 0 {
		limit = 20
	}
	if s.spannerClient != nil {
		if rows, err := s.listAutoOrderRunsSpanner(ctx, orgID, limit); err == nil && len(rows) > 0 {
			return rows
		}
	}
	st := s.aoWorker()
	st.mu.Lock()
	defer st.mu.Unlock()
	var items []AutoOrderRun
	for i := len(st.runs) - 1; i >= 0; i-- {
		if st.runs[i].RetailerID == orgID {
			items = append(items, st.runs[i])
		}
		if len(items) >= limit {
			break
		}
	}
	return items
}

func (s *Service) listAutoOrderRunsSpanner(ctx context.Context, orgID string, limit int) ([]AutoOrderRun, error) {
	stmt := spanner.Statement{
		SQL: `SELECT RunId, RetailerId, Mode, Status, COALESCE(Message, ''), ScheduleBucket,
			COALESCE(CandidateSource, ''), SuggestionsSeen, DraftLines, PlacedLines,
			SkippedJson, PlacedOrdersJson, StartedAt, FinishedAt
			FROM RetailerAutoOrderRuns
			WHERE RetailerId = @rid
			ORDER BY CreatedAt DESC
			LIMIT @lim`,
		Params: map[string]any{"rid": orgID, "lim": int64(limit)},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	var out []AutoOrderRun
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var run AutoOrderRun
		var msg, cand string
		var skippedRaw, placedRaw []byte
		var startedAt time.Time
		var finishedAt spanner.NullTime
		if err := row.Columns(
			&run.RunID, &run.RetailerID, &run.Mode, &run.Status, &msg, &run.ScheduleBucket,
			&cand, &run.Suggestions, &run.DraftLines, &run.PlacedLines,
			&skippedRaw, &placedRaw, &startedAt, &finishedAt,
		); err != nil {
			return nil, err
		}
		run.Message = msg
		run.CandidateSource = cand
		run.StartedAt = startedAt.UTC().Format(time.RFC3339Nano)
		if finishedAt.Valid {
			run.FinishedAt = finishedAt.Time.UTC().Format(time.RFC3339Nano)
		}
		if len(skippedRaw) > 0 {
			_ = json.Unmarshal(skippedRaw, &run.Skipped)
		}
		if len(placedRaw) > 0 {
			_ = json.Unmarshal(placedRaw, &run.PlacedOrders)
		}
		if run.Skipped == nil {
			run.Skipped = []AutoOrderSkip{}
		}
		out = append(out, run)
	}
	return out, nil
}

// SeedAutoOrderCandidates injects candidates for unit tests.
func (s *Service) SeedAutoOrderCandidates(orgID string, cands []AutoOrderCandidate) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.autoOrderCandidates == nil {
		s.autoOrderCandidates = map[string][]AutoOrderCandidate{}
	}
	s.autoOrderCandidates[orgID] = append([]AutoOrderCandidate(nil), cands...)
}

// RunAutoOrderForRetailer executes one tick (testable without HTTP).
func (s *Service) RunAutoOrderForRetailer(ctx context.Context, orgID, mode string) AutoOrderRun {
	mode = NormalizeExecutionMode(mode)
	if mode == "" || mode == AutoOrderModeOff {
		if mode == "" {
			mode = AutoOrderModeDraft
		}
	}
	now := s.now().UTC()
	bucket := now.Format("2006-01-02")
	run := AutoOrderRun{
		RunID:          s.newID(),
		RetailerID:     orgID,
		StartedAt:      now.Format(time.RFC3339Nano),
		Mode:           mode,
		Skipped:        []AutoOrderSkip{},
		ScheduleBucket: bucket,
		Status:         "OK",
	}

	settings := s.loadAutoOrderDurable(ctx, orgID)
	settingsMode := NormalizeExecutionMode(settings.ExecutionMode)
	if mode == AutoOrderModeOff || settingsMode == AutoOrderModeOff {
		run.Mode = AutoOrderModeOff
		run.Status = "SKIPPED_ALL"
		run.Message = "execution_mode_off"
		run.Skipped = append(run.Skipped, AutoOrderSkip{Reason: "execution_mode_off"})
		run.FinishedAt = s.now().UTC().Format(time.RFC3339Nano)
		s.recordAutoOrderRun(run)
		return run
	}
	if !settings.GlobalEnabled && !hasAnyScopedEnable(settings) {
		run.Status = "SKIPPED_ALL"
		run.Message = "auto_order_disabled"
		run.Skipped = append(run.Skipped, AutoOrderSkip{Reason: "auto_order_disabled"})
		run.FinishedAt = s.now().UTC().Format(time.RFC3339Nano)
		s.recordAutoOrderRun(run)
		return run
	}

	cands, candSrc := s.loadAutoOrderCandidatesWithSource(ctx, orgID)
	run.Suggestions = len(cands)
	run.CandidateSource = candSrc
	if len(cands) == 0 {
		run.Status = "SKIPPED_ALL"
		run.Message = "no_suggestions"
		run.Skipped = append(run.Skipped, AutoOrderSkip{Reason: "no_suggestions"})
		run.FinishedAt = s.now().UTC().Format(time.RFC3339Nano)
		s.recordAutoOrderRun(run)
		return run
	}

	// Normalize + filter candidates first
	var actionable []AutoOrderCandidate
	st := s.aoWorker()
	for _, c := range cands {
		if c.SKU == "" && c.ProductID != "" {
			c.SKU = c.ProductID
		}
		if c.SKU == "" {
			run.Skipped = append(run.Skipped, AutoOrderSkip{Reason: "missing_sku"})
			continue
		}
		if c.Qty <= 0 {
			c.Qty = 1
		}
		if c.CategoryID == "" {
			c.CategoryID = s.categoryIDForSKU(ctx, c.SKU)
		}
		if !candidateAllowed(settings, c) {
			run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "scoped_disabled"})
			continue
		}
		key := orgID + "|" + bucket + "|" + mode + "|" + c.SKU
		if s.bucketTaken(key) {
			run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "already_processed_bucket"})
			continue
		}
		if c.SupplierID == "" {
			c.SupplierID = s.supplierIDForRetailerSKU(ctx, orgID, c.SKU)
		}
		actionable = append(actionable, c)
	}

	switch mode {
	case AutoOrderModeShadow:
		s.runAutoOrderShadow(ctx, orgID, bucket, actionable, &run)
	case AutoOrderModePlace:
		s.runAutoOrderPlace(ctx, orgID, bucket, actionable, &run)
	default: // draft
		mode = AutoOrderModeDraft
		run.Mode = mode
		for _, c := range actionable {
			key := orgID + "|" + bucket + "|" + mode + "|" + c.SKU
			if s.cartRepo != nil {
				if c.SupplierID == "" {
					run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "missing_supplier"})
					continue
				}
				item := CartItem{
					CartItemID:    s.newID(),
					RetailerID:    orgID,
					SupplierID:    c.SupplierID,
					ProductID:     aoFirstNonEmpty(c.ProductID, c.SKU),
					Quantity:      c.Qty,
					PriceSnapshot: 0,
					Currency:      "UZS",
					UpdatedAt:     now,
				}
				if err := s.cartRepo.UpsertItems(ctx, []CartItem{item}); err != nil {
					run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "cart_upsert_failed"})
					continue
				}
			}
			s.markBucket(key, run.RunID)
			st.mu.Lock()
			st.bucketDone[key] = true
			st.mu.Unlock()
			run.DraftLines++
		}
	}

	if mode != AutoOrderModeShadow {
		if run.DraftLines == 0 && run.PlacedLines == 0 {
			run.Status = "SKIPPED_ALL"
			if run.Message == "" {
				run.Message = "all_candidates_skipped"
			}
		} else if run.PlacedLines > 0 && len(run.Skipped) > 0 {
			run.Status = "PARTIAL"
			run.Message = "orders_placed_partial"
		} else if run.PlacedLines > 0 {
			run.Status = "OK"
			run.Message = "orders_placed"
		} else if run.DraftLines > 0 {
			run.Status = "OK"
			run.Message = "draft_cart_lines"
		}
	}

	if run.PlacedLines > 0 {
		if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
			_ = w.CreateNotification(ctx, orgID, "RETAILER", events.EventRetailerAutoOrderUpdated,
				"Auto-order placed",
				fmt.Sprintf("Auto-order placed %d line(s) across %d order(s).", run.PlacedLines, len(run.PlacedOrders)),
				"/orders")
		}
	} else if mode == AutoOrderModeShadow && run.DraftLines > 0 {
		if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
			_ = w.CreateNotification(ctx, orgID, "RETAILER", events.EventRetailerAutoOrderUpdated,
				"Auto-order shadow proposals",
				fmt.Sprintf("Shadow recorded %d proposal(s) — no cart or orders created.", run.DraftLines),
				"/auto-order")
		}
	} else if run.DraftLines > 0 {
		if w, ok := s.notifSvc.(NotificationWriter); ok && w != nil {
			_ = w.CreateNotification(ctx, orgID, "RETAILER", events.EventRetailerAutoOrderUpdated,
				"Auto-order draft ready",
				"Auto-order added draft lines — review cart before checkout.",
				"/cart")
		}
	}
	run.FinishedAt = s.now().UTC().Format(time.RFC3339Nano)
	s.recordAutoOrderRun(run)
	_ = s.emitPosEvent(ctx, orgID, events.EventRetailerAutoOrderUpdated, map[string]any{
		"run_id":           run.RunID,
		"draft_lines":      run.DraftLines,
		"placed_lines":     run.PlacedLines,
		"placed_orders":    len(run.PlacedOrders),
		"status":           run.Status,
		"mode":             run.Mode,
		"candidate_source": run.CandidateSource,
	})
	return run
}

func (s *Service) bucketTaken(key string) bool {
	s.mu.RLock()
	if s.autoOrderBucket != nil {
		if _, ok := s.autoOrderBucket[key]; ok {
			s.mu.RUnlock()
			return true
		}
	}
	s.mu.RUnlock()
	st := s.aoWorker()
	st.mu.Lock()
	if st.bucketDone[key] {
		st.mu.Unlock()
		return true
	}
	st.mu.Unlock()
	// Multi-pod: Spanner durable bucket (no ctx on this helper — use Background for read).
	return s.loadDurableBucketTaken(context.Background(), key)
}

func (s *Service) markBucket(key, value string) {
	s.mu.Lock()
	if s.autoOrderBucket == nil {
		s.autoOrderBucket = map[string]string{}
	}
	s.autoOrderBucket[key] = value
	s.mu.Unlock()
	st := s.aoWorker()
	st.mu.Lock()
	st.bucketDone[key] = true
	st.mu.Unlock()
}

// runAutoOrderPlace groups candidates by supplier and creates real orders via OrderCreator.
func (s *Service) runAutoOrderPlace(ctx context.Context, orgID, bucket string, cands []AutoOrderCandidate, run *AutoOrderRun) {
	if s.orderCreator == nil || !s.autoOrderPlaceEnabled {
		// Soft fall-through: still draft so operators are not stuck
		for _, c := range cands {
			run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "place_unavailable"})
		}
		if s.orderCreator == nil {
			run.Message = "place_unavailable_no_order_creator"
		} else {
			run.Message = "place_disabled_set_AUTO_ORDER_PLACE_ENABLED"
		}
		// Draft fallback for ops continuity
		now := s.now().UTC()
		for _, c := range cands {
			if c.SupplierID == "" {
				continue
			}
			if s.cartRepo != nil {
				_ = s.cartRepo.UpsertItems(ctx, []CartItem{{
					CartItemID: s.newID(), RetailerID: orgID, SupplierID: c.SupplierID,
					ProductID: aoFirstNonEmpty(c.ProductID, c.SKU), Quantity: c.Qty,
					Currency: "UZS", UpdatedAt: now,
				}})
			}
			key := orgID + "|" + bucket + "|" + AutoOrderModePlace + "|" + c.SKU
			s.markBucket(key, run.RunID)
			run.DraftLines++
		}
		if run.DraftLines > 0 {
			run.Message = "drafted_place_unavailable_fallback"
		}
		return
	}

	// Group by supplier
	groups := map[string][]AutoOrderCandidate{}
	for _, c := range cands {
		if c.SupplierID == "" {
			run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "missing_supplier"})
			continue
		}
		groups[c.SupplierID] = append(groups[c.SupplierID], c)
	}

	lat, lng, h3, locID := s.autoOrderDeliveryGeo(ctx, orgID)
	if !autoOrderGeoValid(lat, lng, h3) {
		for sid := range groups {
			for _, c := range groups[sid] {
				run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "retailer_geo_missing"})
			}
		}
		run.Message = "retailer_geo_missing"
		return
	}

	placedAny := false
	skippedAny := false
	for supplierID, lines := range groups {
		var orderLines []order.LineItem
		var skus []string
		for _, c := range lines {
			unitPrice := int64(100) // placeholder; Create normalizes/quotes when Spanner products exist
			if c.Name == "" {
				c.Name = c.SKU
			}
			orderLines = append(orderLines, order.LineItem{
				SKU: c.SKU, Name: c.Name, Quantity: c.Qty, UnitPrice: unitPrice,
			})
			skus = append(skus, c.SKU)
		}
		req := order.CreateRequest{
			LineItems:  orderLines,
			H3Cell:     h3,
			Lat:        lat,
			Lng:        lng,
			LocationID: locID,
			SupplierID: supplierID,
			Source:     order.OrderSourceAutoOrder,
		}
		resp, err := s.orderCreator.Create(ctx, orgID, req)
		if err != nil {
			skippedAny = true
			reason := "create_failed"
			if errors.Is(err, order.ErrCreditLimitBreached) {
				reason = "credit_block"
			}
			for _, c := range lines {
				run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: reason})
			}
			continue
		}
		placedAny = true
		for _, c := range lines {
			key := orgID + "|" + bucket + "|" + AutoOrderModePlace + "|" + c.SKU
			s.markBucket(key, resp.OrderID)
			s.persistAutoOrderBucket(ctx, orgID, bucket, AutoOrderModePlace, c.SKU, run.RunID, resp.OrderID)
			run.PlacedLines++
			s.markReorderSuggestionConverted(ctx, orgID, c.SKU)
		}
		run.PlacedOrders = append(run.PlacedOrders, AutoOrderPlacedOrder{
			OrderID:    resp.OrderID,
			SupplierID: supplierID,
			LineCount:  len(lines),
			TotalMinor: resp.TotalMinor,
			SKUs:       skus,
		})
	}
	if placedAny && skippedAny {
		run.Status = "PARTIAL"
		run.Message = "orders_placed_partial"
	} else if placedAny {
		run.Message = "orders_placed"
	}
}

func autoOrderGeoValid(lat, lng float64, h3 string) bool {
	return len(strings.TrimSpace(h3)) == 15 && !(lat == 0 && lng == 0)
}

// autoOrderDeliveryGeo resolves delivery coordinates for place mode.
// Prod (Spanner): incomplete primary location → empty (caller skips).
// Memory/tests (no Spanner): synthetic Tashkent-ish coords so Create Validate can pass.
func (s *Service) autoOrderDeliveryGeo(ctx context.Context, orgID string) (lat, lng float64, h3, locID string) {
	loc, err := s.EnsurePrimaryLocation(ctx, orgID)
	if err == nil && autoOrderGeoValid(loc.Lat, loc.Lng, loc.H3Cell) {
		return loc.Lat, loc.Lng, loc.H3Cell, loc.LocationID
	}
	if s.spannerClient != nil {
		// Production: never invent geo — operator must set primary location lat/lng/h3.
		if err == nil {
			return loc.Lat, loc.Lng, loc.H3Cell, loc.LocationID
		}
		return 0, 0, "", ""
	}
	// Unit-test / memory path: valid-looking 15-char h3 + coords for Create.Validate.
	const testH3 = "88283082bffffff"
	if err == nil && loc.LocationID != "" {
		return 41.2995, 69.2401, testH3, loc.LocationID
	}
	return 41.2995, 69.2401, testH3, ""
}

// persistAutoOrderBucket writes durable multi-pod idempotency when Spanner is available.
func (s *Service) persistAutoOrderBucket(ctx context.Context, retailerID, day, mode, sku, runID, orderID string) {
	if s.spannerClient == nil {
		return
	}
	_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerAutoOrderBucket", map[string]any{
			"RetailerId": retailerID,
			"Day":        day,
			"Mode":       mode,
			"Sku":        sku,
			"RunId":      runID,
			"OrderId":    orderID,
			"CreatedAt":  spanner.CommitTimestamp,
		}),
	})
}

// loadDurableBucketTaken returns true if Spanner already has a bucket row (multi-pod safe).
func (s *Service) loadDurableBucketTaken(ctx context.Context, key string) bool {
	// key format: retailer|day|mode|sku
	if s.spannerClient == nil {
		return false
	}
	parts := strings.Split(key, "|")
	if len(parts) != 4 {
		return false
	}
	_, err := s.spannerClient.Single().ReadRow(ctx, "RetailerAutoOrderBucket",
		spanner.Key{parts[0], parts[1], parts[2], parts[3]},
		[]string{"OrderId"},
	)
	return err == nil
}

func (s *Service) markReorderSuggestionConverted(ctx context.Context, orgID, sku string) {
	// Memory seed
	s.mu.Lock()
	if s.reorderSuggestionSeed != nil {
		rows := s.reorderSuggestionSeed[orgID]
		for i := range rows {
			if rows[i].SKU == sku {
				rows[i].Status = "CONVERTED"
			}
		}
		s.reorderSuggestionSeed[orgID] = rows
	}
	s.mu.Unlock()
	if s.spannerClient == nil {
		return
	}
	_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.UpdateMap("ReorderSuggestions", map[string]any{
			"RetailerId": orgID,
			"Sku":        sku,
			"Status":     "CONVERTED",
			"ComputedAt": spanner.CommitTimestamp,
		}),
	})
}

func (s *Service) recordAutoOrderRun(run AutoOrderRun) {
	st := s.aoWorker()
	st.mu.Lock()
	st.runs = append(st.runs, run)
	if len(st.runs) > 200 {
		st.runs = st.runs[len(st.runs)-200:]
	}
	st.mu.Unlock()
	s.persistAutoOrderRun(context.Background(), run)
}

// persistAutoOrderRun dual-writes audit to Spanner for multi-pod history.
func (s *Service) persistAutoOrderRun(ctx context.Context, run AutoOrderRun) {
	if s.spannerClient == nil {
		return
	}
	skippedRaw, _ := json.Marshal(run.Skipped)
	if run.Skipped == nil {
		skippedRaw = []byte("[]")
	}
	placedRaw, _ := json.Marshal(run.PlacedOrders)
	if run.PlacedOrders == nil {
		placedRaw = []byte("[]")
	}
	startedAt := s.now().UTC()
	if t, err := time.Parse(time.RFC3339Nano, run.StartedAt); err == nil {
		startedAt = t.UTC()
	} else if t, err := time.Parse(time.RFC3339, run.StartedAt); err == nil {
		startedAt = t.UTC()
	}
	var finishedAt any
	if strings.TrimSpace(run.FinishedAt) != "" {
		if t, err := time.Parse(time.RFC3339Nano, run.FinishedAt); err == nil {
			finishedAt = t.UTC()
		} else if t, err := time.Parse(time.RFC3339, run.FinishedAt); err == nil {
			finishedAt = t.UTC()
		}
	}
	cols := map[string]any{
		"RunId":            run.RunID,
		"RetailerId":       run.RetailerID,
		"Mode":             run.Mode,
		"Status":           run.Status,
		"Message":          run.Message,
		"ScheduleBucket":   run.ScheduleBucket,
		"CandidateSource":  run.CandidateSource,
		"SuggestionsSeen":  int64(run.Suggestions),
		"DraftLines":       int64(run.DraftLines),
		"PlacedLines":      int64(run.PlacedLines),
		"SkippedJson":      skippedRaw,
		"PlacedOrdersJson": placedRaw,
		"StartedAt":        startedAt,
		"CreatedAt":        spanner.CommitTimestamp,
	}
	if finishedAt != nil {
		cols["FinishedAt"] = finishedAt
	}
	_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerAutoOrderRuns", cols),
	})
}

func (s *Service) loadAutoOrderCandidates(ctx context.Context, orgID string) []AutoOrderCandidate {
	c, _ := s.loadAutoOrderCandidatesWithSource(ctx, orgID)
	return c
}

// loadAutoOrderCandidatesWithSource prefers test seeds, then inventory (R,s,S) proposals,
// then OPEN ReorderSuggestions (L3.5), then AI prediction lines (legacy when not inventory-grounded).
func (s *Service) loadAutoOrderCandidatesWithSource(ctx context.Context, orgID string) ([]AutoOrderCandidate, string) {
	s.mu.RLock()
	if s.autoOrderCandidates != nil {
		if c := s.autoOrderCandidates[orgID]; len(c) > 0 {
			out := append([]AutoOrderCandidate(nil), c...)
			s.mu.RUnlock()
			return out, "seed"
		}
	}
	s.mu.RUnlock()

	// Inventory-grounded (R,s,S) proposals — preferred when stock/suggestions yield qty.
	if props := s.loadInventoryProposals(ctx, orgID); len(props) > 0 {
		return proposalsToCandidates(props), "inventory_rs"
	}

	// L3.5: sell-through-aware reorder suggestions (OPEN, SuggestedQty > 0)
	if cands := s.candidatesFromReorderSuggestions(ctx, orgID); len(cands) > 0 {
		return cands, "reorder_suggestions"
	}

	if AutoOrderInventoryGrounded() {
		// Skip AI /2-style prediction lines when inventory path is the source of truth.
		return nil, ""
	}

	if s.orders == nil {
		return nil, ""
	}
	items, err := s.orders.ListRetailerAIPredictions(ctx, orgID, 25)
	if err != nil || len(items) == 0 {
		return nil, ""
	}

	// Aggregate qty per SKU (last non-empty supplier/name wins).
	type agg struct {
		sku, productID, supplierID, name string
		qty                              int64
	}
	bySKU := map[string]*agg{}
	orderKeys := make([]string, 0)

	for _, p := range items {
		supplierID := strings.TrimSpace(p.SupplierID)
		for _, li := range p.Items {
			sku := strings.TrimSpace(li.SKU)
			if sku == "" {
				continue
			}
			a, ok := bySKU[sku]
			if !ok {
				a = &agg{sku: sku, productID: sku}
				bySKU[sku] = a
				orderKeys = append(orderKeys, sku)
			}
			qty := li.Quantity
			if qty <= 0 {
				qty = 1
			}
			a.qty += qty
			if name := strings.TrimSpace(li.Name); name != "" {
				a.name = name
			}
			if supplierID != "" {
				a.supplierID = supplierID
			}
		}
	}

	out := make([]AutoOrderCandidate, 0, len(orderKeys))
	for _, sku := range orderKeys {
		a := bySKU[sku]
		out = append(out, AutoOrderCandidate{
			SKU:        a.sku,
			ProductID:  a.productID,
			SupplierID: a.supplierID,
			Qty:        a.qty,
			Name:       a.name,
		})
	}
	return out, "ai_predictions"
}

func (s *Service) candidatesFromReorderSuggestions(ctx context.Context, orgID string) []AutoOrderCandidate {
	// Worker loads all OPEN suggestions — never apply UI source filters.
	items, err := s.listRetailerReorderSuggestions(ctx, orgID, nil)
	if err != nil || len(items) == 0 {
		return nil
	}
	out := make([]AutoOrderCandidate, 0, len(items))
	for _, it := range items {
		sku := strings.TrimSpace(it.SKU)
		if sku == "" || it.SuggestedQty <= 0 {
			continue
		}
		if strings.HasPrefix(strings.ToLower(sku), "local:") {
			continue
		}
		sup := s.supplierIDForRetailerSKU(ctx, orgID, sku)
		out = append(out, AutoOrderCandidate{
			SKU:        sku,
			ProductID:  sku,
			SupplierID: sup,
			Qty:        it.SuggestedQty,
			Sources:    it.Sources,
		})
	}
	return out
}

// supplierIDForRetailerSKU best-effort: last order for this sku, else first favorite.
func (s *Service) supplierIDForRetailerSKU(ctx context.Context, orgID, sku string) string {
	if s.spannerClient != nil {
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT o.SupplierId FROM Orders o
				JOIN OrderLines l ON l.OrderId = o.OrderId
				WHERE o.RetailerId = @rid AND l.Sku = @sku
				  AND o.SupplierId IS NOT NULL AND o.SupplierId != ''
				ORDER BY o.UpdatedAt DESC LIMIT 1`,
			Params: map[string]any{"rid": orgID, "sku": sku},
		})
		defer iter.Stop()
		if row, err := iter.Next(); err == nil {
			var sid string
			if err := row.Column(0, &sid); err == nil && strings.TrimSpace(sid) != "" {
				return strings.TrimSpace(sid)
			}
		}
	}
	// Favorites memory map
	s.mu.RLock()
	if s.favoriteSuppliers != nil {
		if favs := s.favoriteSuppliers[orgID]; len(favs) > 0 {
			for sid, ok := range favs {
				if ok && strings.TrimSpace(sid) != "" {
					s.mu.RUnlock()
					return strings.TrimSpace(sid)
				}
			}
		}
	}
	s.mu.RUnlock()
	// Catalog product supplier (when Spanner products exist)
	if s.spannerClient != nil && strings.TrimSpace(sku) != "" {
		row, err := s.spannerClient.Single().ReadRow(ctx, "Products", spanner.Key{sku}, []string{"SupplierId"})
		if err == nil {
			var sid string
			if err := row.Columns(&sid); err == nil && strings.TrimSpace(sid) != "" {
				return strings.TrimSpace(sid)
			}
		}
	}
	// SSMR / single-supplier seed fallback (request TenantContext wins when present).
	return s.resolveSupplierScope(ctx)
}

// autoOrderWorkerEnabled gates the background ticker (default off).
func autoOrderWorkerEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("AUTO_ORDER_WORKER_ENABLED")))
	return v == "1" || v == "true" || v == "yes" || v == "on"
}

// RunAutoOrderWorker periodically drafts carts for retailers with auto-order enabled.
// Default mode is draft; place mode requires AUTO_ORDER_PLACE_ENABLED separately.
func (s *Service) RunAutoOrderWorker(ctx context.Context, interval time.Duration) {
	if !autoOrderWorkerEnabled() {
		return
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.sweepAutoOrderEnabled(ctx)
		}
	}
}

func (s *Service) sweepAutoOrderEnabled(ctx context.Context) {
	ids := s.listAutoOrderEnabledRetailerIDs(ctx)
	for _, rid := range ids {
		settings := s.loadAutoOrderDurable(ctx, rid)
		mode := NormalizeExecutionMode(settings.ExecutionMode)
		if mode == "" {
			mode = AutoOrderModeDraft
		}
		if mode == AutoOrderModeOff {
			continue
		}
		if mode == AutoOrderModePlace && !s.autoOrderPlaceEnabled {
			mode = AutoOrderModeDraft
		}
		_ = s.RunAutoOrderForRetailer(ctx, rid, mode)
	}
}

func (s *Service) listAutoOrderEnabledRetailerIDs(ctx context.Context) []string {
	seen := map[string]struct{}{}
	var out []string
	add := func(id string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	s.autoOrderMu.RLock()
	for rid, settings := range s.autoOrderByRetailer {
		if settings != nil && (settings.GlobalEnabled || hasAnyScopedEnable(*settings)) {
			add(rid)
		}
	}
	s.autoOrderMu.RUnlock()
	if s.spannerClient != nil {
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT RetailerId FROM RetailerAutoOrderSettings
			      WHERE GlobalEnabled = true LIMIT 200`,
		})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				break
			}
			var rid string
			if err := row.Column(0, &rid); err == nil {
				add(rid)
			}
		}
	}
	return out
}

func hasAnyScopedEnable(s AutoOrderSettings) bool {
	for _, o := range s.SupplierOverrides {
		if o.Enabled {
			return true
		}
	}
	for _, o := range s.ProductOverrides {
		if o.Enabled {
			return true
		}
	}
	for _, o := range s.VariantOverrides {
		if o.Enabled {
			return true
		}
	}
	for _, o := range s.CategoryOverrides {
		if o.Enabled {
			return true
		}
	}
	return false
}

func aoFirstNonEmpty(a, b string) string {
	if strings.TrimSpace(a) != "" {
		return a
	}
	return b
}

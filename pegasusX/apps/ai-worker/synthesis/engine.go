// Package synthesis restores enterprise AI recommendation + preorder drafting
// from live order events into Spanner (AIPredictions + AI_PREORDER orders).
package synthesis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"google.golang.org/api/iterator"
)

// OrderSignal is the minimal order-event payload used for synthesis.
type OrderSignal struct {
	OrderID      string  `json:"order_id"`
	SupplierID   string  `json:"supplier_id"`
	RetailerID   string  `json:"retailer_id"`
	WarehouseID  string  `json:"warehouse_id"`
	Status       string  `json:"status"`
	TotalMinor   int64   `json:"total_minor"`
	Currency     string  `json:"currency"`
	OrderSource  string  `json:"order_source"`
	LineItems    []Line  `json:"line_items"`
	H3Cell       string  `json:"h3_cell"`
	Lat          float64 `json:"lat"`
	Lng          float64 `json:"lng"`
}

// Line is one SKU line on an order signal.
type Line struct {
	ProductID string `json:"product_id"`
	SKU       string `json:"sku"`
	Quantity  int64  `json:"quantity"`
	UnitPrice int64  `json:"unit_price_minor"`
}

// Engine writes advisory predictions and optional AI preorder drafts.
type Engine struct {
	Client *spanner.Client
	Log    *slog.Logger
	Now    func() time.Time
	// MinConfidence gates weak signals (0–1). Default 0.55.
	MinConfidence float64
	// PreorderHorizonDays sets RequestedDeliveryDate offset. Default 3.
	PreorderHorizonDays int
	// AutoConfirmHours after which pending AI preorders auto-confirm. Default 24.
	AutoConfirmHours int
	// CreatePreorders enables AI_PREORDER order draft writes. Default true.
	CreatePreorders bool
}

// New constructs a synthesis engine.
func New(client *spanner.Client, log *slog.Logger) *Engine {
	if log == nil {
		log = slog.Default()
	}
	return &Engine{
		Client:              client,
		Log:                 log,
		Now:                 time.Now,
		MinConfidence:       0.55,
		PreorderHorizonDays: 3,
		AutoConfirmHours:    24,
		CreatePreorders:     true,
	}
}

// HandleOrderEvent synthesizes supplier recommendations (and optional retailer
// AI preorders) from ORDER_CREATED / completed-order envelopes.
func (e *Engine) HandleOrderEvent(ctx context.Context, eventType string, payload []byte) error {
	if e == nil || e.Client == nil {
		return fmt.Errorf("synthesis engine: nil client")
	}
	signal, err := ParseOrderSignal(payload)
	if err != nil {
		return err
	}
	if strings.TrimSpace(signal.SupplierID) == "" || strings.TrimSpace(signal.RetailerID) == "" {
		e.Log.Debug("synthesis skip: missing supplier/retailer", "event_type", eventType)
		return nil
	}
	// Do not re-synthesize from AI drafts (prevents feedback loops).
	if strings.EqualFold(signal.OrderSource, "AI_PREORDER") {
		return nil
	}

	history, err := e.loadRecentRetailerOrders(ctx, signal.SupplierID, signal.RetailerID, 30)
	if err != nil {
		return fmt.Errorf("load retailer order history: %w", err)
	}

	rec := BuildSupplierRecommendation(signal, history, e.now())
	minConf := e.minConfidence()
	// MinConfidence >= 1.0 is an explicit reject-all control (Gate-0 CONTROL proof).
	if minConf >= 1.0 || rec.Score < minConf {
		e.Log.Debug("synthesis below confidence floor",
			"score", rec.Score,
			"min_confidence", minConf,
			"supplier_id", signal.SupplierID,
			"retailer_id", signal.RetailerID,
		)
		return nil
	}

	mutations := make([]*spanner.Mutation, 0, 2)
	predMut, err := e.predictionMutation(rec)
	if err != nil {
		return err
	}
	mutations = append(mutations, predMut)

	if e.CreatePreorders && len(signal.LineItems) > 0 && rec.Action == "SUGGEST_REORDER" {
		preMut, ok, err := e.maybePreorderMutation(ctx, signal, rec)
		if err != nil {
			return err
		}
		if ok {
			mutations = append(mutations, preMut)
		}
	}

	if _, err := e.Client.Apply(ctx, mutations); err != nil {
		return fmt.Errorf("apply synthesis mutations: %w", err)
	}
	e.Log.Info("ai synthesis applied",
		"event_type", eventType,
		"recommendation_id", rec.PredictionID,
		"action", rec.Action,
		"score", rec.Score,
		"supplier_id", signal.SupplierID,
		"retailer_id", signal.RetailerID,
		"preorder", len(mutations) > 1,
	)
	return nil
}

// Recommendation is the internal advisory model persisted to AIPredictions.
type Recommendation struct {
	PredictionID  string
	AggregateID   string
	AggregateType string
	SupplierID    string
	Action        string
	Score         float64
	Confidence    float64
	Source        string
	Explanation   string
	ReasonCodes   []string
	Evidence      []map[string]string
	ExpiresAt     time.Time
	GeneratedAt   time.Time
	OrderID       string
	RetailerID    string
	WarehouseID   string
	ProductHints  []string
}

// BuildSupplierRecommendation is pure logic for tests.
func BuildSupplierRecommendation(signal OrderSignal, history []OrderSignal, now time.Time) Recommendation {
	orderCount := len(history) + 1
	var totalQty int64
	productSet := map[string]struct{}{}
	for _, line := range signal.LineItems {
		totalQty += line.Quantity
		if pid := strings.TrimSpace(line.ProductID); pid != "" {
			productSet[pid] = struct{}{}
		}
	}
	// Recency + volume heuristic (deterministic, no external model call).
	// Base/recency sized so a first thin order scores below default MinConfidence(0.55).
	recencyBoost := 0.05
	if orderCount >= 3 {
		recencyBoost = 0.30
	} else if orderCount >= 2 {
		recencyBoost = 0.18
	} else if orderCount >= 1 {
		recencyBoost = 0.10
	}
	volumeBoost := math.Min(0.3, float64(totalQty)/100.0)
	valueBoost := math.Min(0.2, float64(signal.TotalMinor)/1_000_000.0)
	// Base 0.15: first thin order (qty~15, ~100k minor) lands ~0.50 < default 0.55 gate.
	score := clamp01(0.15 + recencyBoost + volumeBoost + valueBoost)

	action := "SUGGEST_REORDER"
	reason := []string{"ORDER_SIGNAL", "RETAILER_HISTORY"}
	explanation := fmt.Sprintf(
		"Retailer %s placed order %s (%d lines, %d units). Pattern over last %d orders suggests replenishment.",
		signal.RetailerID, signal.OrderID, len(signal.LineItems), totalQty, orderCount,
	)
	if totalQty >= 50 || signal.TotalMinor >= 500_000 {
		action = "FLAG_HIGH_DEMAND"
		reason = append(reason, "HIGH_VOLUME")
		explanation = fmt.Sprintf(
			"High-demand signal for retailer %s on order %s (units=%d total_minor=%d). Consider warehouse buffer increase.",
			signal.RetailerID, signal.OrderID, totalQty, signal.TotalMinor,
		)
	}

	hints := make([]string, 0, len(productSet))
	for pid := range productSet {
		hints = append(hints, pid)
	}

	aggID := shortUUID()
	return Recommendation{
		PredictionID:  uuid.NewString(),
		AggregateID:   aggID,
		AggregateType: "ORDER",
		SupplierID:    signal.SupplierID,
		Action:        action,
		Score:         score,
		Confidence:    score,
		Source:        "ai-worker/synthesis",
		Explanation:   explanation,
		ReasonCodes:   reason,
		Evidence: []map[string]string{
			{"label": "source_order_id", "value": signal.OrderID},
			{"label": "retailer_id", "value": signal.RetailerID},
			{"label": "history_orders", "value": fmt.Sprintf("%d", orderCount)},
			{"label": "line_units", "value": fmt.Sprintf("%d", totalQty)},
		},
		ExpiresAt:    now.UTC().Add(7 * 24 * time.Hour),
		GeneratedAt:  now.UTC(),
		OrderID:      signal.OrderID,
		RetailerID:   signal.RetailerID,
		WarehouseID:  signal.WarehouseID,
		ProductHints: hints,
	}
}

// ParseOrderSignal extracts an OrderSignal from a Kafka/outbox JSON body.
// Accepts both nested envelopes and flat OrderEvent-shaped payloads.
func ParseOrderSignal(payload []byte) (OrderSignal, error) {
	var envelope struct {
		Type      string          `json:"type"`
		Payload   json.RawMessage `json:"payload"`
		OrderID   string          `json:"order_id"`
		SupplierID string         `json:"supplier_id"`
	}
	_ = json.Unmarshal(payload, &envelope)

	raw := payload
	if len(envelope.Payload) > 0 && envelope.Payload[0] == '{' {
		raw = envelope.Payload
	}

	var signal OrderSignal
	if err := json.Unmarshal(raw, &signal); err != nil {
		return OrderSignal{}, fmt.Errorf("parse order signal: %w", err)
	}
	// Some producers nest line items as raw JSON blob under LineItems field already decoded.
	if signal.OrderID == "" && envelope.OrderID != "" {
		signal.OrderID = envelope.OrderID
	}
	if signal.SupplierID == "" && envelope.SupplierID != "" {
		signal.SupplierID = envelope.SupplierID
	}
	return signal, nil
}

func (e *Engine) predictionMutation(rec Recommendation) (*spanner.Mutation, error) {
	data, err := json.Marshal(map[string]any{
		"action":        rec.Action,
		"confidence":    rec.Confidence,
		"source":        rec.Source,
		"explanation":   rec.Explanation,
		"reason_codes":  rec.ReasonCodes,
		"evidence":      rec.Evidence,
		"expires_at":    rec.ExpiresAt.Format(time.RFC3339Nano),
		"order_id":      rec.OrderID,
		"retailer_id":   rec.RetailerID,
		"warehouse_id":  rec.WarehouseID,
		"product_hints": rec.ProductHints,
	})
	if err != nil {
		return nil, err
	}
	// AggregateId is STRING(36) in DDL — use UUID only.
	return spanner.InsertOrUpdateMap("AIPredictions", map[string]any{
		"PredictionId":   rec.PredictionID,
		"AggregateId":    rec.AggregateID,
		"AggregateType":  rec.AggregateType,
		"SupplierId":     rec.SupplierID,
		"PredictionData": data,
		"Score":          rec.Score,
		"Status":         "PENDING",
		"CreatedAt":      spanner.CommitTimestamp,
		"UpdatedAt":      spanner.CommitTimestamp,
	}), nil
}

func (e *Engine) maybePreorderMutation(ctx context.Context, signal OrderSignal, rec Recommendation) (*spanner.Mutation, bool, error) {
	// Skip if a pending AI preorder already exists for this retailer recently.
	exists, err := e.hasOpenAIPreorder(ctx, signal.RetailerID, signal.SupplierID)
	if err != nil {
		return nil, false, err
	}
	if exists {
		return nil, false, nil
	}

	now := e.now()
	delivery := now.AddDate(0, 0, e.preorderHorizonDays()).UTC()
	autoConfirm := now.Add(time.Duration(e.autoConfirmHours()) * time.Hour).UTC()
	orderID := uuid.NewString()

	// Scale suggested qty: 50% of last order, min 1 per line.
	lines := make([]map[string]any, 0, len(signal.LineItems))
	var total int64
	for _, line := range signal.LineItems {
		qty := line.Quantity / 2
		if qty < 1 {
			qty = 1
		}
		unit := line.UnitPrice
		if unit <= 0 {
			unit = 0
		}
		total += qty * unit
		lines = append(lines, map[string]any{
			"product_id":        line.ProductID,
			"sku":               line.SKU,
			"quantity":          qty,
			"unit_price_minor":  unit,
			"line_total_minor":  qty * unit,
		})
	}
	lineJSON, err := json.Marshal(lines)
	if err != nil {
		return nil, false, err
	}
	currency := strings.TrimSpace(signal.Currency)
	if currency == "" {
		currency = "UZS"
	}

	mut := spanner.InsertMap("Orders", map[string]any{
		"OrderId":              orderID,
		"SupplierId":           signal.SupplierID,
		"RetailerId":           signal.RetailerID,
		"WarehouseId":          nullable(signal.WarehouseID),
		"Status":               "PENDING",
		"OrderSource":          "AI_PREORDER",
		"ConfirmationStatus":   "PENDING",
		"LineItemsJson":        lineJSON,
		"TotalMinor":           total,
		"OriginalTotalMinor":   total,
		"Currency":             currency,
		"H3Cell":               nullable(signal.H3Cell),
		"Lat":                  signal.Lat,
		"Lng":                  signal.Lng,
		"RequestedDeliveryDate": delivery,
		"AutoConfirmAt":        autoConfirm,
		"DerivedFromOrderId":   nullable(signal.OrderID),
		"Version":              int64(1),
		"CreatedAt":            now,
		"UpdatedAt":            now,
	})
	_ = rec // score already gated
	return mut, true, nil
}

func (e *Engine) hasOpenAIPreorder(ctx context.Context, retailerID, supplierID string) (bool, error) {
	stmt := spanner.Statement{
		SQL: `SELECT OrderId FROM Orders
		      WHERE RetailerId = @retailer_id AND SupplierId = @supplier_id
		        AND OrderSource = 'AI_PREORDER' AND ConfirmationStatus = 'PENDING'
		      LIMIT 1`,
		Params: map[string]any{
			"retailer_id": retailerID,
			"supplier_id": supplierID,
		},
	}
	iter := e.Client.Single().Query(ctx, stmt)
	defer iter.Stop()
	_, err := iter.Next()
	if err == iterator.Done {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (e *Engine) loadRecentRetailerOrders(ctx context.Context, supplierID, retailerID string, limit int) ([]OrderSignal, error) {
	if limit <= 0 {
		limit = 20
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, RetailerId, WarehouseId, Status, TotalMinor, Currency, OrderSource, LineItemsJson
		      FROM Orders@{FORCE_INDEX=Idx_Orders_ByRetailerCreated}
		      WHERE RetailerId = @retailer_id
		      ORDER BY CreatedAt DESC
		      LIMIT @limit`,
		Params: map[string]any{
			"retailer_id": retailerID,
			"limit":       int64(limit),
		},
	}
	iter := e.Client.Single().Query(ctx, stmt)
	defer iter.Stop()

	out := make([]OrderSignal, 0, limit)
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var (
			orderID, supID, retID, whID, status, currency, source string
			totalMinor                                           int64
			lineJSON                                             []byte
			whNull                                               spanner.NullString
		)
		if err := row.Columns(&orderID, &supID, &retID, &whNull, &status, &totalMinor, &currency, &source, &lineJSON); err != nil {
			return nil, err
		}
		if whNull.Valid {
			whID = whNull.StringVal
		}
		// Scope to supplier when present.
		if supplierID != "" && supID != supplierID {
			continue
		}
		sig := OrderSignal{
			OrderID:     orderID,
			SupplierID:  supID,
			RetailerID:  retID,
			WarehouseID: whID,
			Status:      status,
			TotalMinor:  totalMinor,
			Currency:    currency,
			OrderSource: source,
		}
		_ = json.Unmarshal(lineJSON, &sig.LineItems)
		out = append(out, sig)
	}
	return out, nil
}

func (e *Engine) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Engine) minConfidence() float64 {
	if e.MinConfidence <= 0 {
		return 0.55
	}
	return e.MinConfidence
}

func (e *Engine) preorderHorizonDays() int {
	if e.PreorderHorizonDays <= 0 {
		return 3
	}
	return e.PreorderHorizonDays
}

func (e *Engine) autoConfirmHours() int {
	if e.AutoConfirmHours <= 0 {
		return 24
	}
	return e.AutoConfirmHours
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func shortUUID() string {
	return uuid.NewString()
}

func nullable(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

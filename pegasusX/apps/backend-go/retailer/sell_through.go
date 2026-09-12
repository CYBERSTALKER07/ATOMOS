package retailer

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// SellThroughDayDTO is one daily sell-through rollup row (L3 flywheel).
type SellThroughDayDTO struct {
	RetailerID   string `json:"retailer_id"`
	LocationID   string `json:"location_id"`
	SkuID        string `json:"sku_id"`
	Day          string `json:"day"` // YYYY-MM-DD
	QtySold      int64  `json:"qty_sold"`
	QtyVoided    int64  `json:"qty_voided"`
	QtyOnHandEod *int64 `json:"qty_on_hand_eod,omitempty"`
	NetSold      int64  `json:"net_sold"` // qty_sold - qty_voided
	Source       string `json:"source"`   // STORE_POS
}

type sellThroughKey struct {
	RetailerID string
	LocationID string
	SkuID      string
	Day        string
}

// recordSellThroughSale increments daily sold qty for each line and updates demand factor.
func (s *Service) recordSellThroughSale(ctx context.Context, retailerID, locationID string, lines []PosSaleLine) {
	day := s.now().UTC().Format("2006-01-02")
	for _, l := range lines {
		sku := strings.TrimSpace(l.Sku)
		if sku == "" || l.Qty <= 0 {
			continue
		}
		// local: SKUs never feed supplier DemandAdjustments / reorder suggestions.
		if IsLocalSKU(sku) {
			continue
		}
		_ = s.upsertSellThroughDelta(ctx, retailerID, locationID, sku, day, l.Qty, 0, "sale")
		_ = s.applySellThroughDemandFactor(ctx, retailerID, sku, day, float64(l.Qty))
	}
	_ = s.emitPosEvent(ctx, retailerID, events.EventRetailerSellThroughUpdated, map[string]any{
		"location_id": locationID,
		"day":         day,
		"kind":        "sale",
		"lines":       len(lines),
	})
}

// recordSellThroughVoid increments voided qty (reverses net sold) and demand factor.
func (s *Service) recordSellThroughVoid(ctx context.Context, retailerID, locationID string, lines []PosSaleLine) {
	day := s.now().UTC().Format("2006-01-02")
	for _, l := range lines {
		sku := strings.TrimSpace(l.Sku)
		if sku == "" || l.Qty <= 0 {
			continue
		}
		if IsLocalSKU(sku) {
			continue
		}
		_ = s.upsertSellThroughDelta(ctx, retailerID, locationID, sku, day, 0, l.Qty, "void")
		_ = s.applySellThroughDemandFactor(ctx, retailerID, sku, day, -float64(l.Qty))
	}
	_ = s.emitPosEvent(ctx, retailerID, events.EventRetailerSellThroughUpdated, map[string]any{
		"location_id": locationID,
		"day":         day,
		"kind":        "void",
		"lines":       len(lines),
	})
}

// emitDemandSignal records an in-memory DEMAND_SIGNAL for tests and, when no
// Spanner client is configured, returns. Spanner path emits inside
// upsertSellThroughDelta (same txn as the sell-through row — P2-12).
func (s *Service) emitDemandSignal(ctx context.Context, retailerID, locationID, sku, day, kind string, qtyDelta int64) {
	if qtyDelta == 0 || IsLocalSKU(sku) {
		return
	}
	net := s.dayNetSold(ctx, retailerID, locationID, sku, day)
	ev := events.DemandSignalEvent{
		BaseEvent: events.BaseEvent{
			Type:      events.EventDemandSignal,
			Timestamp: s.now().UTC().Format(time.RFC3339Nano),
		},
		RetailerID: retailerID,
		LocationID: locationID,
		SKU:        sku,
		Day:        day,
		QtyDelta:   qtyDelta,
		NetSold:    net,
		Source:     "STORE_POS",
		Kind:       kind,
		SupplierID: s.supplierIDForRetailerSKU(ctx, retailerID, sku),
	}
	s.mu.Lock()
	s.demandSignalsEmitted = append(s.demandSignalsEmitted, ev)
	if len(s.demandSignalsEmitted) > 500 {
		s.demandSignalsEmitted = s.demandSignalsEmitted[len(s.demandSignalsEmitted)-250:]
	}
	s.mu.Unlock()
}

func (s *Service) upsertSellThroughDelta(ctx context.Context, retailerID, locationID, sku, day string, soldDelta, voidDelta int64, kind string) error {
	onHand, _ := s.SumOnHandForSKU(ctx, retailerID, sku)
	qtyDelta := soldDelta - voidDelta

	if s.spannerClient == nil {
		s.mu.Lock()
		if s.sellThroughDaily == nil {
			s.sellThroughDaily = map[sellThroughKey]SellThroughDayDTO{}
		}
		k := sellThroughKey{RetailerID: retailerID, LocationID: locationID, SkuID: sku, Day: day}
		row := s.sellThroughDaily[k]
		row.RetailerID = retailerID
		row.LocationID = locationID
		row.SkuID = sku
		row.Day = day
		row.QtySold += soldDelta
		row.QtyVoided += voidDelta
		row.NetSold = row.QtySold - row.QtyVoided
		row.Source = "STORE_POS"
		if onHand >= 0 {
			v := onHand
			row.QtyOnHandEod = &v
		}
		s.sellThroughDaily[k] = row
		s.mu.Unlock()
		s.emitDemandSignal(ctx, retailerID, locationID, sku, day, kind, qtyDelta)
		return nil
	}

	dayTime, err := time.Parse("2006-01-02", day)
	if err != nil {
		dayTime = s.now().UTC()
	}
	supplierID := s.supplierIDForRetailerSKU(ctx, retailerID, sku)
	signalID := s.newID()
	aggID := retailerID + "|" + sku + "|" + day
	if strings.TrimSpace(signalID) == "" {
		signalID = aggID + "|" + s.now().UTC().Format(time.RFC3339Nano)
	}
	ts := s.now().UTC().Format(time.RFC3339Nano)

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var sold, voided int64
		row, err := txn.ReadRow(ctx, "RetailerSellThroughDaily",
			spanner.Key{retailerID, locationID, sku, dayTime},
			[]string{"QtySold", "QtyVoided"})
		if err == nil {
			_ = row.Columns(&sold, &voided)
		}
		sold += soldDelta
		voided += voidDelta
		net := sold - voided
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerSellThroughDaily", map[string]any{
				"RetailerId":   retailerID,
				"LocationId":   locationID,
				"SkuId":        sku,
				"Day":          dayTime,
				"QtySold":      sold,
				"QtyVoided":    voided,
				"QtyOnHandEod": onHand,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
		}
		if qtyDelta != 0 && kind != "" {
			payload := map[string]any{
				"type":        events.EventDemandSignal,
				"timestamp":   ts,
				"retailer_id": retailerID,
				"location_id": locationID,
				"sku":         sku,
				"day":         day,
				"qty_delta":   qtyDelta,
				"net_sold":    net,
				"source":      "STORE_POS",
				"kind":        kind,
				"signal_id":   signalID,
			}
			if supplierID != "" {
				payload["supplier_id"] = supplierID
			}
			buf := &spannerTxnBuffer{}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateDemandSignal, aggID, events.TopicMain, payload); err != nil {
				return err
			}
			if err := buf.Flush(txn); err != nil {
				return err
			}
		}
		return txn.BufferWrite(muts)
	})
	if err != nil {
		return err
	}

	// Mirror into memory ring for tests / diagnostics (post-commit).
	s.emitDemandSignal(ctx, retailerID, locationID, sku, day, kind, qtyDelta)

	// Flywheel feed stays best-effort separate so missing DDL cannot roll back
	// the sell-through + outbox commit (coverage rule satisfied by outbox).
	if qtyDelta != 0 && kind != "" {
		_, _ = s.spannerClient.Apply(ctx, []*spanner.Mutation{
			spanner.InsertOrUpdateMap("FlywheelDemandFeed", map[string]any{
				"SignalId":   signalID,
				"SupplierId": nullableStr(supplierID),
				"RetailerId": retailerID,
				"LocationId": nullableStr(locationID),
				"SkuId":      sku,
				"Day":        dayTime,
				"QtyDelta":   qtyDelta,
				"NetSold":    s.dayNetSold(ctx, retailerID, locationID, sku, day),
				"Kind":       kind,
				"Source":     "STORE_POS",
				"CreatedAt":  spanner.CommitTimestamp,
			}),
		})
	}
	return nil
}

// dayNetSold returns cumulative net sold for the day (memory or Spanner).
func (s *Service) dayNetSold(ctx context.Context, retailerID, locationID, sku, day string) int64 {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.sellThroughDaily == nil {
			return 0
		}
		row := s.sellThroughDaily[sellThroughKey{RetailerID: retailerID, LocationID: locationID, SkuID: sku, Day: day}]
		return row.NetSold
	}
	dayTime, err := time.Parse("2006-01-02", day)
	if err != nil {
		return 0
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerSellThroughDaily",
		spanner.Key{retailerID, locationID, sku, dayTime},
		[]string{"QtySold", "QtyVoided"})
	if err != nil {
		return 0
	}
	var sold, voided int64
	if err := row.Columns(&sold, &voided); err != nil {
		return 0
	}
	return sold - voided
}

// EmittedDemandSignals returns a copy of in-memory DEMAND_SIGNAL emissions (tests).
func (s *Service) EmittedDemandSignals() []events.DemandSignalEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.demandSignalsEmitted) == 0 {
		return nil
	}
	out := make([]events.DemandSignalEvent, len(s.demandSignalsEmitted))
	copy(out, s.demandSignalsEmitted)
	return out
}

// applySellThroughDemandFactor merges SELL_THROUGH into DemandAdjustments.FactorsJson for the day.
func (s *Service) applySellThroughDemandFactor(ctx context.Context, retailerID, sku, day string, delta float64) error {
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.sellThroughFactors == nil {
			s.sellThroughFactors = map[string]float64{}
		}
		key := retailerID + "|" + sku + "|" + day
		s.sellThroughFactors[key] += delta
		return nil
	}

	// Parse day as civil date for Spanner DATE
	dayTime, err := time.Parse("2006-01-02", day)
	if err != nil {
		dayTime = s.now().UTC()
	}

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		var base, adj, adjusted float64
		var factorsRaw spanner.NullString
		row, err := txn.ReadRow(ctx, "DemandAdjustments",
			spanner.Key{retailerID, sku, dayTime},
			[]string{"BaseVelocity", "Adjustment", "AdjustedDemand", "FactorsJson"})
		factors := map[string]float64{}
		if err == nil {
			_ = row.Columns(&base, &adj, &adjusted, &factorsRaw)
			if factorsRaw.Valid && factorsRaw.StringVal != "" {
				_ = json.Unmarshal([]byte(factorsRaw.StringVal), &factors)
			}
		}
		factors["SELL_THROUGH"] = factors["SELL_THROUGH"] + delta
		sumF := 0.0
		for _, v := range factors {
			sumF += v
		}
		adj = sumF
		adjusted = base + adj
		raw, _ := json.Marshal(factors)
		supplierID := ""
		if prow, perr := txn.ReadRow(ctx, "Products", spanner.Key{sku}, []string{"SupplierId"}); perr == nil {
			_ = prow.Column(0, &supplierID)
		}
		cols := map[string]any{
			"RetailerId":     retailerID,
			"Sku":            sku,
			"Date":           dayTime,
			"BaseVelocity":   base,
			"Adjustment":     adj,
			"AdjustedDemand": adjusted,
			"FactorsJson":    string(raw),
			"ComputedAt":     spanner.CommitTimestamp,
		}
		if strings.TrimSpace(supplierID) != "" {
			cols["SupplierId"] = supplierID
		}
		m := spanner.InsertOrUpdateMap("DemandAdjustments", cols)
		return txn.BufferWrite([]*spanner.Mutation{m})
	})
	return err
}

// GetSellThroughFactor returns memory-mode SELL_THROUGH for tests.
func (s *Service) GetSellThroughFactor(retailerID, sku, day string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.sellThroughFactors == nil {
		return 0
	}
	return s.sellThroughFactors[retailerID+"|"+sku+"|"+day]
}

// ListSellThrough returns rollups for a retailer (optional location + day range).
func (s *Service) ListSellThrough(ctx context.Context, retailerID, locationID, fromDay, toDay string, limit int) ([]SellThroughDayDTO, error) {
	if limit <= 0 {
		limit = 100
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []SellThroughDayDTO
		for _, row := range s.sellThroughDaily {
			if row.RetailerID != retailerID {
				continue
			}
			if locationID != "" && row.LocationID != locationID {
				continue
			}
			if fromDay != "" && row.Day < fromDay {
				continue
			}
			if toDay != "" && row.Day > toDay {
				continue
			}
			row.NetSold = row.QtySold - row.QtyVoided
			row.Source = "STORE_POS"
			out = append(out, row)
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Day != out[j].Day {
				return out[i].Day > out[j].Day
			}
			return out[i].SkuID < out[j].SkuID
		})
		if len(out) > limit {
			out = out[:limit]
		}
		return out, nil
	}

	sql := `SELECT RetailerId, LocationId, SkuId, CAST(Day AS STRING), QtySold, QtyVoided, QtyOnHandEod
		FROM RetailerSellThroughDaily
		WHERE RetailerId = @rid`
	params := map[string]any{"rid": retailerID, "lim": int64(limit)}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	if fromDay != "" {
		sql += ` AND Day >= @from`
		params["from"] = fromDay
	}
	if toDay != "" {
		sql += ` AND Day <= @to`
		params["to"] = toDay
	}
	sql += ` ORDER BY Day DESC, SkuId LIMIT @lim`

	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()
	var out []SellThroughDayDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var dto SellThroughDayDTO
		var onHand spanner.NullInt64
		if err := row.Columns(&dto.RetailerID, &dto.LocationID, &dto.SkuID, &dto.Day, &dto.QtySold, &dto.QtyVoided, &onHand); err != nil {
			return nil, err
		}
		dto.NetSold = dto.QtySold - dto.QtyVoided
		dto.Source = "STORE_POS"
		if onHand.Valid {
			v := onHand.Int64
			dto.QtyOnHandEod = &v
		}
		out = append(out, dto)
	}
	return out, nil
}

// HandleSellThroughInsights serves GET /v1/retailer/insights/sell-through
func (s *Service) HandleSellThroughInsights(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	allowed := auth.HasRetailerPerm(claims, auth.PermPosSell) ||
		auth.HasRetailerPerm(claims, auth.PermStockView) ||
		auth.HasRetailerPerm(claims, auth.PermReportsView) ||
		auth.HasRetailerPerm(claims, auth.PermCapManage)
	if !allowed {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "forbidden"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	q := r.URL.Query()
	items, err := s.ListSellThrough(r.Context(), orgID,
		strings.TrimSpace(q.Get("location_id")),
		strings.TrimSpace(q.Get("from")),
		strings.TrimSpace(q.Get("to")),
		100,
	)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed", "detail": err.Error()})
		return
	}
	if items == nil {
		items = []SellThroughDayDTO{}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"source": "STORE_POS",
	})
}

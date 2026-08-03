package retailer

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// Wave C2.1 HQ projections.
// Non-negotiable: HQ sales daily deltas are applied in the same Spanner Apply
// batch as the POS sale/void ledger write (see savePosSale).

type hqSalesKey struct {
	RetailerID string
	LocationID string
	Day        string // YYYY-MM-DD UTC
	SkuID      string
}

// HqSalesDayDTO is one SKU rollup for a store-day (C2.2 REST will expose).
type HqSalesDayDTO struct {
	RetailerID string `json:"retailer_id"`
	LocationID string `json:"location_id"`
	Day        string `json:"day"`
	SkuID      string `json:"sku_id"`
	QtySold    int64  `json:"qty_sold"`
	QtyVoided  int64  `json:"qty_voided"`
	GrossMinor int64  `json:"gross_minor"`
	NetMinor   int64  `json:"net_minor"`
	Currency   string `json:"currency"`
}

type hqStockKey struct {
	RetailerID string
	LocationID string
	SkuID      string
}

type hqStockSnap struct {
	OnHand   int64
	Reserved int64
	AsOf     time.Time
}

func hqDayUTC(t time.Time) string {
	return t.UTC().Format("2006-01-02")
}

func parseHQDay(day string) time.Time {
	day = strings.TrimSpace(day)
	if t, err := time.Parse("2006-01-02", day); err == nil {
		return t
	}
	return time.Now().UTC().Truncate(24 * time.Hour)
}

// applyHqSalesDeltaMemory mutates in-memory HQ sales (tests / no Spanner).
func (s *Service) applyHqSalesDeltaMemory(k hqSalesKey, qtySold, qtyVoided, grossDelta, netDelta int64, currency string) {
	if s.hqSalesDaily == nil {
		s.hqSalesDaily = map[hqSalesKey]HqSalesDayDTO{}
	}
	cur := s.hqSalesDaily[k]
	if cur.RetailerID == "" {
		cur = HqSalesDayDTO{
			RetailerID: k.RetailerID,
			LocationID: k.LocationID,
			Day:        k.Day,
			SkuID:      k.SkuID,
			Currency:   currency,
		}
	}
	if currency != "" {
		cur.Currency = currency
	}
	cur.QtySold += qtySold
	cur.QtyVoided += qtyVoided
	cur.GrossMinor += grossDelta
	cur.NetMinor += netDelta
	s.hqSalesDaily[k] = cur
}

// buildHqSalesMutationsFromSale returns Spanner mutations for sale completion
// (qty sold + gross/net). Includes local: SKUs.
func (s *Service) buildHqSalesMutationsFromSale(ctx context.Context, sale PosSaleDTO) ([]*spanner.Mutation, error) {
	day := hqDayUTC(s.now())
	if sale.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, sale.CreatedAt); err == nil {
			day = hqDayUTC(t)
		}
	}
	currency := sale.Currency
	if currency == "" {
		currency = "UZS"
	}
	var muts []*spanner.Mutation
	for _, l := range sale.Lines {
		sku := strings.TrimSpace(l.Sku)
		if sku == "" || l.Qty <= 0 {
			continue
		}
		m, err := s.hqSalesDailyUpsertMutation(ctx, sale.RetailerID, sale.LocationID, day, sku,
			l.Qty, 0, l.LineTotalMinor, l.LineTotalMinor, currency)
		if err != nil {
			return nil, err
		}
		if m != nil {
			muts = append(muts, m)
		}
	}
	return muts, nil
}

// buildHqSalesMutationsFromVoid returns Spanner mutations for void
// (qty voided + net decrease). Gross unchanged.
func (s *Service) buildHqSalesMutationsFromVoid(ctx context.Context, sale PosSaleDTO) ([]*spanner.Mutation, error) {
	day := hqDayUTC(s.now())
	if sale.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, sale.CreatedAt); err == nil {
			day = hqDayUTC(t)
		}
	}
	currency := sale.Currency
	if currency == "" {
		currency = "UZS"
	}
	var muts []*spanner.Mutation
	for _, l := range sale.Lines {
		sku := strings.TrimSpace(l.Sku)
		if sku == "" || l.Qty <= 0 {
			continue
		}
		m, err := s.hqSalesDailyUpsertMutation(ctx, sale.RetailerID, sale.LocationID, day, sku,
			0, l.Qty, 0, -l.LineTotalMinor, currency)
		if err != nil {
			return nil, err
		}
		if m != nil {
			muts = append(muts, m)
		}
	}
	return muts, nil
}

// hqSalesDailyUpsertMutation reads current row (if any) and builds InsertOrUpdate.
// On memory backend, applies delta in-process and returns nil mutation.
func (s *Service) hqSalesDailyUpsertMutation(ctx context.Context, retailerID, locationID, day, sku string,
	qtySold, qtyVoided, grossDelta, netDelta int64, currency string) (*spanner.Mutation, error) {
	retailerID = strings.TrimSpace(retailerID)
	locationID = strings.TrimSpace(locationID)
	sku = strings.TrimSpace(sku)
	if retailerID == "" || locationID == "" || day == "" || sku == "" {
		return nil, nil
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.applyHqSalesDeltaMemory(hqSalesKey{
			RetailerID: retailerID, LocationID: locationID, Day: day, SkuID: sku,
		}, qtySold, qtyVoided, grossDelta, netDelta, currency)
		return nil, nil
	}

	curSold, curVoid, curGross, curNet, curCur, err := s.readHqSalesDaily(ctx, retailerID, locationID, day, sku)
	if err != nil {
		// Table missing pre-migration: skip HQ write so sale still succeeds.
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerHqSalesDaily") {
			return nil, nil
		}
		return nil, err
	}
	if curCur == "" {
		curCur = currency
	}
	if currency == "" {
		currency = curCur
	}
	return spanner.InsertOrUpdateMap("RetailerHqSalesDaily", map[string]any{
		"RetailerId": retailerID,
		"LocationId": locationID,
		"Day":        parseHQDay(day),
		"SkuId":      sku,
		"QtySold":    curSold + qtySold,
		"QtyVoided":  curVoid + qtyVoided,
		"GrossMinor": curGross + grossDelta,
		"NetMinor":   curNet + netDelta,
		"Currency":   currency,
		"UpdatedAt":  spanner.CommitTimestamp,
	}), nil
}

func (s *Service) readHqSalesDaily(ctx context.Context, retailerID, locationID, day, sku string) (sold, voided, gross, net int64, currency string, err error) {
	dayT := parseHQDay(day)
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerHqSalesDaily",
		spanner.Key{retailerID, dayT, locationID, sku},
		[]string{"QtySold", "QtyVoided", "GrossMinor", "NetMinor", "Currency"})
	if err != nil {
		if isNotFound(err) {
			return 0, 0, 0, 0, "", nil
		}
		return 0, 0, 0, 0, "", err
	}
	if err := row.Columns(&sold, &voided, &gross, &net, &currency); err != nil {
		return 0, 0, 0, 0, "", err
	}
	return sold, voided, gross, net, currency, nil
}

// applyHqFromSaleMemory is called when sale is saved without Spanner.
func (s *Service) applyHqFromSaleMemory(sale PosSaleDTO) {
	day := hqDayUTC(s.now())
	if sale.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, sale.CreatedAt); err == nil {
			day = hqDayUTC(t)
		}
	}
	cur := sale.Currency
	if cur == "" {
		cur = "UZS"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range sale.Lines {
		s.applyHqSalesDeltaMemory(hqSalesKey{
			RetailerID: sale.RetailerID, LocationID: sale.LocationID, Day: day, SkuID: l.Sku,
		}, l.Qty, 0, l.LineTotalMinor, l.LineTotalMinor, cur)
	}
}

func (s *Service) applyHqFromVoidMemory(sale PosSaleDTO) {
	day := hqDayUTC(s.now())
	if sale.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339Nano, sale.CreatedAt); err == nil {
			day = hqDayUTC(t)
		}
	}
	cur := sale.Currency
	if cur == "" {
		cur = "UZS"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, l := range sale.Lines {
		s.applyHqSalesDeltaMemory(hqSalesKey{
			RetailerID: sale.RetailerID, LocationID: sale.LocationID, Day: day, SkuID: l.Sku,
		}, 0, l.Qty, 0, -l.LineTotalMinor, cur)
	}
}

// GetHqSalesDay returns memory aggregate for tests / C2.2.
func (s *Service) GetHqSalesDay(retailerID, locationID, day, sku string) (HqSalesDayDTO, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.hqSalesDaily == nil {
		return HqSalesDayDTO{}, false
	}
	v, ok := s.hqSalesDaily[hqSalesKey{
		RetailerID: retailerID, LocationID: locationID, Day: day, SkuID: sku,
	}]
	return v, ok
}

// ListHqSalesByLocation sums NetMinor by location for a day (property: sum locations = org).
func (s *Service) ListHqSalesByLocation(ctx context.Context, retailerID, day string) (map[string]int64, error) {
	retailerID = strings.TrimSpace(retailerID)
	day = strings.TrimSpace(day)
	if retailerID == "" || day == "" {
		return nil, nil
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		out := map[string]int64{}
		for k, v := range s.hqSalesDaily {
			if k.RetailerID == retailerID && k.Day == day {
				out[k.LocationID] += v.NetMinor
			}
		}
		return out, nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT LocationId, SUM(NetMinor) FROM RetailerHqSalesDaily
			WHERE RetailerId = @rid AND Day = @day
			GROUP BY LocationId`,
		Params: map[string]any{"rid": retailerID, "day": parseHQDay(day)},
	})
	defer iter.Stop()
	out := map[string]int64{}
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			if strings.Contains(err.Error(), "RetailerHqSalesDaily") {
				return out, nil
			}
			return nil, err
		}
		var loc string
		var net int64
		if err := row.Columns(&loc, &net); err != nil {
			return nil, err
		}
		out[loc] = net
	}
	return out, nil
}

// UpsertHqStockSnapshot writes current on-hand for a SKU at a location (post stock move).
// Best-effort; not required to share sale txn (stock already applied separately today).
func (s *Service) UpsertHqStockSnapshot(ctx context.Context, retailerID, locationID, sku string, onHand, reserved int64) error {
	retailerID = strings.TrimSpace(retailerID)
	locationID = strings.TrimSpace(locationID)
	sku = strings.TrimSpace(sku)
	if retailerID == "" || locationID == "" || sku == "" {
		return nil
	}
	now := s.now().UTC()
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.hqStockSnap == nil {
			s.hqStockSnap = map[hqStockKey]hqStockSnap{}
		}
		s.hqStockSnap[hqStockKey{RetailerID: retailerID, LocationID: locationID, SkuID: sku}] = hqStockSnap{
			OnHand: onHand, Reserved: reserved, AsOf: now,
		}
		return nil
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerHqStockSnapshot", map[string]any{
			"RetailerId": retailerID,
			"LocationId": locationID,
			"SkuId":      sku,
			"OnHand":     onHand,
			"Reserved":   reserved,
			"AsOf":       now,
			"UpdatedAt":  spanner.CommitTimestamp,
		}),
	})
	if err != nil && (strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "RetailerHqStockSnapshot")) {
		return nil
	}
	return err
}

// refreshHqStockSnapshotsForSale updates HQ stock snapshots for sale lines (after stock delta).
func (s *Service) refreshHqStockSnapshotsForSale(ctx context.Context, retailerID, locationID, bin string, lines []PosSaleLine) {
	for _, l := range lines {
		onHand, _ := s.getOnHand(ctx, locationID, normalizeBin(bin), l.Sku)
		_ = s.UpsertHqStockSnapshot(ctx, retailerID, locationID, l.Sku, onHand, 0)
	}
}

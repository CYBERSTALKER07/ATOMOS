package retailer

import (
	"context"
	"math"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/replenishment"
	"google.golang.org/api/iterator"
)

const defaultReviewIntervalDays = 1.0

// InventoryProposal is one inventory-grounded (R,s,S) candidate.
type InventoryProposal struct {
	SKU          string
	ProductID    string
	SupplierID   string
	CategoryID   string
	Qty          int64
	IP           float64
	ReorderPoint float64 // s
	OrderUpTo    float64 // S
	DemandPerDay float64
	Confidence   float64
	Reason       string
	UnitsPerPack int64
}

// ComputeOrderUpToQty returns order quantity when IP ≤ s using S = s + d̄·R.
// Pure helper for tests. pack rounds up to UnitsPerPack when > 1.
func ComputeOrderUpToQty(ip, reorderPoint, demandPerDay, reviewDays float64, pack int64) (qty int64, orderUpTo float64, shouldOrder bool) {
	if reviewDays <= 0 {
		reviewDays = defaultReviewIntervalDays
	}
	if demandPerDay < 0 {
		demandPerDay = 0
	}
	if reorderPoint < 0 {
		reorderPoint = 0
	}
	orderUpTo = reorderPoint + demandPerDay*reviewDays
	if ip > reorderPoint {
		return 0, orderUpTo, false
	}
	need := orderUpTo - ip
	if need <= 0 {
		return 0, orderUpTo, false
	}
	qty = int64(math.Ceil(need - 1e-9))
	if qty < 1 {
		qty = 1
	}
	if pack > 1 {
		rem := qty % pack
		if rem != 0 {
			qty += pack - rem
		}
	}
	return qty, orderUpTo, true
}

// ConfidenceFromStockAge decays confidence with days since last stock update (option c).
func ConfidenceFromStockAge(updatedAt time.Time, now time.Time) float64 {
	if updatedAt.IsZero() {
		return 0.55
	}
	days := now.Sub(updatedAt).Hours() / 24
	if days < 0 {
		days = 0
	}
	c := 1.0 - 0.05*days
	if c < 0.35 {
		c = 0.35
	}
	if c > 1 {
		c = 1
	}
	return c
}

// loadInventoryProposals builds (R,s,S) candidates from stock + OPEN suggestions / sell-through.
func (s *Service) loadInventoryProposals(ctx context.Context, orgID string) []InventoryProposal {
	if s == nil || s.spannerClient == nil || strings.TrimSpace(orgID) == "" {
		return nil
	}
	now := s.now().UTC()
	today := civil.DateOf(now)

	type stockRow struct {
		sku       string
		onHand    int64
		reserved  int64
		updatedAt time.Time
	}
	stockBySKU := map[string]*stockRow{}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT Sku, COALESCE(SUM(OnHand), 0), COALESCE(SUM(Reserved), 0), MAX(UpdatedAt)
			FROM RetailerStockBalances
			WHERE RetailerId = @rid
			GROUP BY Sku
			LIMIT 500`,
		Params: map[string]any{"rid": orgID},
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
		var sku string
		var onHand, reserved int64
		var updated spanner.NullTime
		if err := row.Columns(&sku, &onHand, &reserved, &updated); err != nil {
			continue
		}
		sku = strings.TrimSpace(sku)
		if sku == "" || strings.HasPrefix(strings.ToLower(sku), "local:") {
			continue
		}
		sr := &stockRow{sku: sku, onHand: onHand, reserved: reserved}
		if updated.Valid {
			sr.updatedAt = updated.Time.UTC()
		}
		stockBySKU[sku] = sr
	}

	// Merge OPEN reorder suggestions (demand + safety → ROP).
	sugBySKU := map[string]RetailerReorderSuggestion{}
	if items, err := s.listRetailerReorderSuggestions(ctx, orgID, nil); err == nil {
		for _, it := range items {
			sku := strings.TrimSpace(it.SKU)
			if sku == "" {
				continue
			}
			sugBySKU[sku] = it
			if _, ok := stockBySKU[sku]; !ok {
				stockBySKU[sku] = &stockRow{sku: sku, onHand: it.CurrentStock}
			}
		}
	}

	out := make([]InventoryProposal, 0)
	for sku, sr := range stockBySKU {
		sug, hasSug := sugBySKU[sku]
		inFlight := int64(0)
		if hasSug {
			inFlight = sug.InFlightQty
		}
		if inFlight == 0 {
			inFlight = s.inFlightQtyForSKU(ctx, orgID, sku)
		}
		onHand := float64(sr.onHand)
		if hasSug && sug.CurrentStock > 0 && onHand == 0 {
			onHand = float64(sug.CurrentStock)
		}
		backorders := float64(sr.reserved)
		if backorders < 0 {
			backorders = 0
		}
		ip := onHand + float64(inFlight) - backorders

		dBar := 0.0
		if hasSug && sug.AdjustedDemand > 0 {
			dBar = sug.AdjustedDemand
		} else if hasSug && sug.SellThroughVel > 0 {
			dBar = sug.SellThroughVel
		} else {
			dBar = s.sellThroughDemandPerDay(ctx, orgID, sku, today, 7)
		}
		if dBar <= 0 {
			continue
		}

		leadDays := 2.0
		safety := dBar * 0.15
		if hasSug && sug.SafetyStock > 0 {
			safety = sug.SafetyStock
		} else if replenishment.SafetyStockV2Enabled() {
			safety = replenishment.SafetyStockUnits(replenishment.SafetyStockInputs{
				DBar:             dBar,
				SigmaD:           math.Max(dBar*0.25, 1),
				SigmaDAssumed:    true,
				L:                leadDays,
				SigmaL:           1.0,
				LeadSigmaAssumed: true,
				ServiceLevel:     0.98,
			})
		}
		rop := dBar*leadDays + safety
		if hasSug && sug.SuggestedQty > 0 && sug.CurrentStock+sug.InFlightQty >= 0 {
			// Prefer ROP implied by suggestion engine when available:
			// SuggestedQty ≈ ROP - stock - inFlight → ROP ≈ SuggestedQty + stock + inFlight
			implied := float64(sug.SuggestedQty) + float64(sug.CurrentStock) + float64(sug.InFlightQty)
			if implied > rop {
				rop = implied
			}
		}

		pack := s.unitsPerPackForSKU(ctx, sku)
		qty, orderUpTo, ok := ComputeOrderUpToQty(ip, rop, dBar, defaultReviewIntervalDays, pack)
		if !ok || qty <= 0 {
			continue
		}

		sup := s.supplierIDForRetailerSKU(ctx, orgID, sku)
		cat := s.categoryIDForSKU(ctx, sku)
		conf := ConfidenceFromStockAge(sr.updatedAt, now)
		out = append(out, InventoryProposal{
			SKU:          sku,
			ProductID:    sku,
			SupplierID:   sup,
			CategoryID:   cat,
			Qty:          qty,
			IP:           ip,
			ReorderPoint: rop,
			OrderUpTo:    orderUpTo,
			DemandPerDay: dBar,
			Confidence:   conf,
			Reason:       "inventory_rs",
			UnitsPerPack: pack,
		})
	}
	return out
}

func (s *Service) inFlightQtyForSKU(ctx context.Context, retailerID, sku string) int64 {
	if s.spannerClient == nil {
		return 0
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(SUM(l.Quantity), 0)
			FROM Orders o
			JOIN OrderLines l ON l.OrderId = o.OrderId
			WHERE o.RetailerId = @rid AND l.Sku = @sku
			  AND UPPER(o.Status) IN ('PENDING', 'CONFIRMED', 'PICKING', 'PACKED', 'IN_TRANSIT', 'DISPATCHED')`,
		Params: map[string]any{"rid": retailerID, "sku": sku},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0
	}
	var n int64
	_ = row.Column(0, &n)
	return n
}

func (s *Service) sellThroughDemandPerDay(ctx context.Context, retailerID, sku string, today civil.Date, windowDays int) float64 {
	if s.spannerClient == nil || windowDays <= 0 {
		return 0
	}
	from := today.AddDays(-(windowDays - 1))
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT COALESCE(SUM(QtySold - QtyVoided), 0)
			FROM RetailerSellThroughDaily
			WHERE RetailerId = @rid AND SkuId = @sku AND Day >= @from AND Day <= @to`,
		Params: map[string]any{"rid": retailerID, "sku": sku, "from": from, "to": today},
	})
	defer iter.Stop()
	row, err := iter.Next()
	if err != nil {
		return 0
	}
	var units int64
	if err := row.Column(0, &units); err != nil || units <= 0 {
		return 0
	}
	return float64(units) / float64(windowDays)
}

func (s *Service) unitsPerPackForSKU(ctx context.Context, sku string) int64 {
	if s.spannerClient == nil || sku == "" {
		return 1
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Products", spanner.Key{sku}, []string{"UnitsPerPack"})
	if err != nil {
		return 1
	}
	var pack spanner.NullInt64
	if err := row.Columns(&pack); err != nil || !pack.Valid || pack.Int64 <= 1 {
		return 1
	}
	return pack.Int64
}

func (s *Service) categoryIDForSKU(ctx context.Context, sku string) string {
	if s.spannerClient == nil || sku == "" {
		return ""
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Products", spanner.Key{sku}, []string{"CategoryId"})
	if err != nil {
		return ""
	}
	var cat string
	if err := row.Columns(&cat); err != nil {
		return ""
	}
	return strings.TrimSpace(cat)
}

func proposalsToCandidates(props []InventoryProposal) []AutoOrderCandidate {
	out := make([]AutoOrderCandidate, 0, len(props))
	for _, p := range props {
		out = append(out, AutoOrderCandidate{
			SKU:          p.SKU,
			ProductID:    p.ProductID,
			SupplierID:   p.SupplierID,
			CategoryID:   p.CategoryID,
			Qty:          p.Qty,
			Sources:      []string{"inventory_rs"},
			IP:           p.IP,
			ReorderPoint: p.ReorderPoint,
			OrderUpTo:    p.OrderUpTo,
			Confidence:   p.Confidence,
			Reason:       p.Reason,
		})
	}
	return out
}

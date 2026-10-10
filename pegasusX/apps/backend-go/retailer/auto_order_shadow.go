package retailer

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/civil"
	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// ShadowProposalStatusOpen is written by shadow mode runs.
const ShadowProposalStatusOpen = "OPEN"

// AutoOrderShadowProposal is one persisted shadow line.
type AutoOrderShadowProposal struct {
	ProposalID   string  `json:"proposal_id"`
	RetailerID   string  `json:"retailer_id"`
	SKU          string  `json:"sku"`
	SupplierID   string  `json:"supplier_id,omitempty"`
	ProposedQty  int64   `json:"proposed_qty"`
	IP           float64 `json:"ip"`
	ReorderPoint float64 `json:"reorder_point"`
	OrderUpTo    float64 `json:"order_up_to"`
	Confidence   float64 `json:"confidence,omitempty"`
	Reason       string  `json:"reason,omitempty"`
	BucketDate   string  `json:"bucket_date"`
	Status       string  `json:"status"`
	RunID        string  `json:"run_id,omitempty"`
	CreatedAt    string  `json:"created_at,omitempty"`
}

func (s *Service) runAutoOrderShadow(ctx context.Context, orgID, bucket string, cands []AutoOrderCandidate, run *AutoOrderRun) {
	if !AutoOrderShadowEnabled() {
		run.Status = "SKIPPED_ALL"
		run.Message = "shadow_disabled_set_AUTO_ORDER_SHADOW_ENABLED"
		for _, c := range cands {
			run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "shadow_disabled"})
		}
		return
	}
	day, err := civil.ParseDate(bucket)
	if err != nil {
		day = civil.DateOf(s.now().UTC())
	}
	written := 0
	for _, c := range cands {
		key := orgID + "|" + bucket + "|" + AutoOrderModeShadow + "|" + c.SKU
		proposalID := s.newID()
		if err := s.persistShadowProposal(ctx, orgID, proposalID, day, run.RunID, c); err != nil {
			run.Skipped = append(run.Skipped, AutoOrderSkip{SKU: c.SKU, Reason: "shadow_persist_failed"})
			continue
		}
		s.markBucket(key, run.RunID)
		st := s.aoWorker()
		st.mu.Lock()
		st.bucketDone[key] = true
		st.mu.Unlock()
		written++
	}
	run.DraftLines = written // reuse draft_lines as "proposals written" for audit compatibility
	if written == 0 {
		run.Status = "SKIPPED_ALL"
		if run.Message == "" {
			run.Message = "no_shadow_proposals"
		}
	} else {
		run.Status = "OK"
		run.Message = "shadow_proposals_recorded"
	}
}

func (s *Service) persistShadowProposal(ctx context.Context, orgID, proposalID string, day civil.Date, runID string, c AutoOrderCandidate) error {
	// Memory fallback for tests without Spanner
	if s.spannerClient == nil {
		s.mu.Lock()
		if s.shadowProposalsMem == nil {
			s.shadowProposalsMem = map[string][]AutoOrderShadowProposal{}
		}
		s.shadowProposalsMem[orgID] = append(s.shadowProposalsMem[orgID], AutoOrderShadowProposal{
			ProposalID:   proposalID,
			RetailerID:   orgID,
			SKU:          c.SKU,
			SupplierID:   c.SupplierID,
			ProposedQty:  c.Qty,
			IP:           c.IP,
			ReorderPoint: c.ReorderPoint,
			OrderUpTo:    c.OrderUpTo,
			Confidence:   c.Confidence,
			Reason:       aoFirstNonEmpty(c.Reason, "inventory_rs"),
			BucketDate:   day.String(),
			Status:       ShadowProposalStatusOpen,
			RunID:        runID,
			CreatedAt:    s.now().UTC().Format(time.RFC3339Nano),
		})
		s.mu.Unlock()
		return nil
	}
	cols := map[string]any{
		"RetailerId":   orgID,
		"ProposalId":   proposalID,
		"Sku":          c.SKU,
		"SupplierId":   nullableStr(c.SupplierID),
		"ProposedQty":  c.Qty,
		"IP":           c.IP,
		"ReorderPoint": c.ReorderPoint,
		"OrderUpTo":    c.OrderUpTo,
		"Confidence":   c.Confidence,
		"Reason":       aoFirstNonEmpty(c.Reason, "inventory_rs"),
		"BucketDate":   day,
		"Status":       ShadowProposalStatusOpen,
		"RunId":        nullableStr(runID),
		"CreatedAt":    spanner.CommitTimestamp,
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerAutoOrderShadowProposals", cols),
	})
	return err
}

func (s *Service) listShadowProposals(ctx context.Context, orgID string, limit int) ([]AutoOrderShadowProposal, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		items := s.shadowProposalsMem[orgID]
		if items == nil {
			return []AutoOrderShadowProposal{}, nil
		}
		out := append([]AutoOrderShadowProposal(nil), items...)
		if len(out) > limit {
			out = out[len(out)-limit:]
		}
		// newest last in mem — reverse
		for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
			out[i], out[j] = out[j], out[i]
		}
		return out, nil
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT ProposalId, RetailerId, Sku, COALESCE(SupplierId, ''), ProposedQty,
			COALESCE(IP, 0), COALESCE(ReorderPoint, 0), COALESCE(OrderUpTo, 0),
			COALESCE(Confidence, 0), COALESCE(Reason, ''), CAST(BucketDate AS STRING), Status,
			COALESCE(RunId, ''), CreatedAt
			FROM RetailerAutoOrderShadowProposals
			WHERE RetailerId = @rid
			ORDER BY CreatedAt DESC
			LIMIT @lim`,
		Params: map[string]any{"rid": orgID, "lim": int64(limit)},
	})
	defer iter.Stop()
	var out []AutoOrderShadowProposal
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var p AutoOrderShadowProposal
		var created time.Time
		if err := row.Columns(
			&p.ProposalID, &p.RetailerID, &p.SKU, &p.SupplierID, &p.ProposedQty,
			&p.IP, &p.ReorderPoint, &p.OrderUpTo, &p.Confidence, &p.Reason,
			&p.BucketDate, &p.Status, &p.RunID, &created,
		); err != nil {
			return nil, err
		}
		p.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, p)
	}
	return out, nil
}

func (s *Service) loadShadowStats(ctx context.Context, orgID string, windowDays int) (AutoOrderShadowStats, error) {
	if windowDays <= 0 {
		windowDays = 30
	}
	stats := AutoOrderShadowStats{WindowDays: windowDays}
	if err := s.computeShadowAcceptance(ctx, orgID, windowDays, &stats); err != nil {
		return stats, err
	}
	return stats, nil
}

// computeShadowAcceptance joins shadow proposals to COMPLETED orders in ±3d window.
func (s *Service) computeShadowAcceptance(ctx context.Context, orgID string, windowDays int, stats *AutoOrderShadowStats) error {
	if stats == nil {
		return nil
	}
	from := civil.DateOf(s.now().UTC().AddDate(0, 0, -(windowDays - 1)))

	type prop struct {
		sku string
		qty int64
		day civil.Date
	}
	var props []prop

	if s.spannerClient == nil {
		s.mu.RLock()
		for _, p := range s.shadowProposalsMem[orgID] {
			d, err := civil.ParseDate(p.BucketDate)
			if err != nil {
				continue
			}
			if d.Before(from) {
				continue
			}
			props = append(props, prop{sku: p.SKU, qty: p.ProposedQty, day: d})
		}
		s.mu.RUnlock()
	} else {
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT Sku, ProposedQty, BucketDate
				FROM RetailerAutoOrderShadowProposals
				WHERE RetailerId = @rid AND BucketDate >= @from`,
			Params: map[string]any{"rid": orgID, "from": from},
		})
		defer iter.Stop()
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var sku string
			var qty int64
			var day civil.Date
			if err := row.Columns(&sku, &qty, &day); err != nil {
				return err
			}
			props = append(props, prop{sku: sku, qty: qty, day: day})
		}
	}
	stats.ProposalCount = int64(len(props))
	if len(props) == 0 {
		return nil
	}

	// Load completed order line qtys by sku+day from LineItemsJson (best-effort).
	type orderLine struct {
		sku string
		qty int64
		day civil.Date
	}
	var lines []orderLine
	if s.spannerClient != nil {
		iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
			SQL: `SELECT LineItemsJson, CAST(DATE(CreatedAt) AS STRING)
				FROM Orders
				WHERE RetailerId = @rid
				  AND UPPER(Status) IN ('COMPLETED', 'DELIVERED', 'CLOSED')
				  AND CreatedAt >= TIMESTAMP(@from)
				LIMIT 500`,
			Params: map[string]any{
				"rid":  orgID,
				"from": from.In(time.UTC).Format(time.RFC3339),
			},
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
			var raw []byte
			var dayStr string
			if err := row.Columns(&raw, &dayStr); err != nil {
				continue
			}
			day, err := civil.ParseDate(dayStr)
			if err != nil {
				continue
			}
			var items []struct {
				SKU      string `json:"sku"`
				Quantity int64  `json:"quantity"`
			}
			if err := json.Unmarshal(raw, &items); err != nil {
				continue
			}
			for _, it := range items {
				sku := strings.TrimSpace(it.SKU)
				if sku == "" || it.Quantity <= 0 {
					continue
				}
				lines = append(lines, orderLine{sku: sku, qty: it.Quantity, day: day})
			}
		}
	}

	var absErr, sumProp float64
	var matched, unmodified int64
	for _, p := range props {
		sumProp += float64(p.qty)
		bestQty := int64(0)
		found := false
		for _, l := range lines {
			if l.sku != p.sku {
				continue
			}
			delta := l.day.DaysSince(p.day)
			if delta < -3 || delta > 3 {
				continue
			}
			found = true
			if bestQty == 0 || absInt64(l.qty-p.qty) < absInt64(bestQty-p.qty) {
				bestQty = l.qty
			}
		}
		if !found {
			absErr += float64(p.qty)
			continue
		}
		matched++
		absErr += math.Abs(float64(bestQty - p.qty))
		if bestQty == p.qty {
			unmodified++
		}
	}
	stats.MatchedOrders = matched
	if sumProp > 0 {
		stats.WAPE = absErr / sumProp
	}
	if matched > 0 {
		stats.UnmodifiedRate = float64(unmodified) / float64(matched)
	}
	return nil
}

func absInt64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

// HandleAutoOrderShadowProposals GET /v1/retailer/settings/auto-order/shadow-proposals
func (s *Service) HandleAutoOrderShadowProposals(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	items, err := s.listShadowProposals(r.Context(), orgID, 50)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "list_failed"})
		return
	}
	if items == nil {
		items = []AutoOrderShadowProposal{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// HandleAutoOrderShadowStats GET /v1/retailer/settings/auto-order/shadow-stats
func (s *Service) HandleAutoOrderShadowStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	orgID, err := retailerIDFromRequest(r)
	if err != nil {
		writeRetailerIdentityError(w, err)
		return
	}
	stats, err := s.loadShadowStats(r.Context(), orgID, 30)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stats_failed"})
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

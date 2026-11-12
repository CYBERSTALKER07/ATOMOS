package retailer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
	"google.golang.org/api/iterator"
)

func normalizeBin(b string) string {
	b = strings.ToUpper(strings.TrimSpace(b))
	switch b {
	case BinBackroom, BinFloor, BinQuarantine:
		return b
	default:
		return ""
	}
}

func (s *Service) assertLocationInOrg(ctx context.Context, orgID, locID string) error {
	loc, found, err := s.getLocation(ctx, locID)
	if err != nil {
		return err
	}
	if !found || loc.RetailerID != orgID {
		return errors.New("location_not_found")
	}
	return nil
}

func (s *Service) available(onHand, reserved int64) int64 {
	a := onHand - reserved
	if a < 0 {
		return 0
	}
	return a
}

func (s *Service) getOnHand(ctx context.Context, locationID, bin, sku string) (int64, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.stockBalances == nil {
			return 0, nil
		}
		b, ok := s.stockBalances[stockBalanceKey{locationID, bin, sku}]
		if !ok {
			return 0, nil
		}
		return b.OnHand, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerStockBalances",
		spanner.Key{locationID, bin, sku}, []string{"OnHand"})
	if err != nil {
		if isNotFound(err) {
			return 0, nil
		}
		return 0, err
	}
	var onHand int64
	if err := row.Columns(&onHand); err != nil {
		return 0, err
	}
	return onHand, nil
}

// SumOnHandForSKU across all bins/locations for a retailer (reorder AI).
func (s *Service) SumOnHandForSKU(ctx context.Context, retailerID, sku string) (int64, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var total int64
		for k, b := range s.stockBalances {
			if b.RetailerID == retailerID && k.Sku == sku {
				total += b.OnHand
			}
		}
		return total, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT COALESCE(SUM(OnHand), 0) FROM RetailerStockBalances
			WHERE RetailerId = @rid AND Sku = @sku`,
		Params: map[string]any{"rid": retailerID, "sku": sku},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	var total int64
	if err := row.Column(0, &total); err != nil {
		return 0, err
	}
	return total, nil
}

func (s *Service) listStockBalances(ctx context.Context, retailerID, locationID string) ([]StockBalanceDTO, error) {
	if s != nil && s.stockBalancesQuery != nil {
		return s.stockBalancesQuery(ctx, retailerID, locationID)
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []StockBalanceDTO
		for k, b := range s.stockBalances {
			if b.RetailerID != retailerID || k.LocationID != locationID {
				continue
			}
			out = append(out, StockBalanceDTO{
				LocationID: k.LocationID, StockBin: k.StockBin, Sku: k.Sku,
				OnHand: b.OnHand, Reserved: b.Reserved, Available: s.available(b.OnHand, b.Reserved),
			})
		}
		sort.Slice(out, func(i, j int) bool {
			if out[i].Sku != out[j].Sku {
				return out[i].Sku < out[j].Sku
			}
			return out[i].StockBin < out[j].StockBin
		})
		return out, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT LocationId, StockBin, Sku, OnHand, Reserved FROM RetailerStockBalances
			WHERE RetailerId = @rid AND LocationId = @lid`,
		Params: map[string]any{"rid": retailerID, "lid": locationID},
	}
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, stmt)
	} else {
		iter = s.spannerClient.Single().Query(ctx, stmt)
	}
	defer iter.Stop()
	var out []StockBalanceDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var loc, bin, sku string
		var onHand, reserved int64
		if err := row.Columns(&loc, &bin, &sku, &onHand, &reserved); err != nil {
			return nil, err
		}
		out = append(out, StockBalanceDTO{
			LocationID: loc, StockBin: bin, Sku: sku,
			OnHand: onHand, Reserved: reserved, Available: s.available(onHand, reserved),
		})
	}
	return out, nil
}

func (s *Service) applyDelta(ctx context.Context, retailerID, locationID, bin, sku string, delta int64, moveType, refType, refID, actor, note string) error {
	if delta == 0 {
		return nil
	}
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.stockBalances == nil {
			s.stockBalances = map[stockBalanceKey]memStockBalance{}
		}
		if s.stockMovements == nil {
			s.stockMovements = []StockMovementDTO{}
		}
		key := stockBalanceKey{locationID, bin, sku}
		cur := s.stockBalances[key]
		cur.RetailerID = retailerID
		next := cur.OnHand + delta
		if next < 0 {
			return errors.New("insufficient_stock")
		}
		cur.OnHand = next
		s.stockBalances[key] = cur
		s.stockMovements = append(s.stockMovements, StockMovementDTO{
			MovementID: s.newID(), LocationID: locationID, StockBin: bin, Sku: sku,
			Qty: delta, MovementType: moveType, RefType: refType, RefID: refID,
			ActorUserID: actor, Note: note, CreatedAt: s.now().UTC().Format(time.RFC3339Nano),
		})
		// C3.3: bump location inventory etag (caller holds s.mu)
		s.bumpStockLocationVersionLocked(retailerID, locationID, bin)
		return nil
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// Read current
		var onHand, reserved int64
		row, err := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{locationID, bin, sku},
			[]string{"OnHand", "Reserved"})
		if err != nil && !isNotFound(err) {
			return err
		}
		if err == nil {
			_ = row.Columns(&onHand, &reserved)
		}
		next := onHand + delta
		if next < 0 {
			return errors.New("insufficient_stock")
		}
		muts := []*spanner.Mutation{
			spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
				"LocationId": locationID,
				"StockBin":   bin,
				"Sku":        sku,
				"RetailerId": retailerID,
				"OnHand":     next,
				"Reserved":   reserved,
				"UpdatedAt":  spanner.CommitTimestamp,
			}),
			spanner.InsertMap("RetailerStockMovements", map[string]any{
				"MovementId":   s.newID(),
				"RetailerId":   retailerID,
				"LocationId":   locationID,
				"StockBin":     bin,
				"Sku":          sku,
				"Qty":          delta,
				"MovementType": moveType,
				"RefType":      nullableStr(refType),
				"RefId":        nullableStr(refID),
				"ActorUserId":  nullableStr(actor),
				"Note":         nullableStr(note),
				"CreatedAt":    spanner.CommitTimestamp,
			}),
		}
		if err := txn.BufferWrite(muts); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		evType := events.EventStoreStockAdjusted
		switch moveType {
		case MoveReceive:
			evType = events.EventStoreStockReceived
		case MoveTransferBin:
			evType = events.EventStoreStockTransferred
		case MoveCountVariance:
			evType = events.EventStoreStockCounted
		}
		payload := map[string]any{
			"type":        evType,
			"timestamp":   s.now().Format(time.RFC3339Nano),
			"retailer_id": retailerID,
			"location_id": locationID,
			"stock_bin":   bin,
			"sku":         sku,
			"qty":         delta,
			"movement":    moveType,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateRetailer, retailerID, events.TopicMain, payload); err != nil {
			return err
		}
		return buf.Flush(txn)
	})
	if err == nil {
		// C3.3: location inventory etag (best-effort separate write if version table present)
		s.bumpStockLocationVersion(ctx, retailerID, locationID, bin)
	}
	return err
}

func (s *Service) applyTransfer(ctx context.Context, retailerID, fromLoc, toLoc, fromBin, toBin, sku string, qty int64, actor, note string) error {
	return s.applyTransferTyped(ctx, retailerID, fromLoc, toLoc, fromBin, toBin, sku, qty, MoveTransferBin, "TRANSFER", actor, note)
}

func (s *Service) applyTransferTyped(ctx context.Context, retailerID, fromLoc, toLoc, fromBin, toBin, sku string, qty int64, moveType, refType, actor, note string) error {
	if err := s.applyDelta(ctx, retailerID, fromLoc, fromBin, sku, -qty, moveType, refType, toLoc+":"+toBin, actor, note); err != nil {
		return err
	}
	if err := s.applyDelta(ctx, retailerID, toLoc, toBin, sku, qty, moveType, refType, fromLoc+":"+fromBin, actor, note); err != nil {
		_ = s.applyDelta(ctx, retailerID, fromLoc, fromBin, sku, qty, MoveAdjust, "TRANSFER_ROLLBACK", "", actor, "rollback")
		return err
	}
	return nil
}

// ClaimHoldLine is one SKU qty to quarantine or resolve.
type ClaimHoldLine struct {
	SKU string
	Qty int64
}

// HoldForClaim moves sellable stock (FLOOR then BACKROOM) into QUARANTINE for a filed claim.
// Best-effort per line: insufficient stock is skipped (no receive / already sold) without failing the claim.
func (s *Service) HoldForClaim(ctx context.Context, retailerID, claimID, orderID string, lines []ClaimHoldLine, actor string) error {
	if s == nil || retailerID == "" || claimID == "" || len(lines) == 0 {
		return nil
	}
	loc, err := s.EnsurePrimaryLocation(ctx, retailerID)
	if err != nil {
		return err
	}
	locID := loc.LocationID
	note := "claim_hold:" + claimID
	if orderID != "" {
		note += " order:" + orderID
	}
	for _, ln := range lines {
		sku := strings.TrimSpace(ln.SKU)
		need := ln.Qty
		if sku == "" || need <= 0 {
			continue
		}
		// Prefer FLOOR, then BACKROOM.
		for _, bin := range []string{BinFloor, BinBackroom} {
			if need <= 0 {
				break
			}
			avail := s.onHandInBin(ctx, locID, bin, sku)
			if avail <= 0 {
				continue
			}
			take := need
			if take > avail {
				take = avail
			}
			if err := s.applyTransferTyped(ctx, retailerID, locID, locID, bin, BinQuarantine, sku, take, MoveClaimHold, "CLAIM", actor, note); err != nil {
				s.log.Warn("claim hold transfer failed", "sku", sku, "bin", bin, "qty", take, "err", err)
				continue
			}
			need -= take
		}
		_ = s.syncReorderCurrentStock(ctx, retailerID, sku)
	}
	_ = s.emitPosEvent(ctx, retailerID, events.EventStoreStockTransferred, map[string]any{
		"reason":   "CLAIM_HOLD",
		"claim_id": claimID,
		"order_id": orderID,
	})
	return nil
}

// ResolveClaimStock disposes quarantined units after claim adjudication.
// disposition: RETURN | RESTORE | WASTE
func (s *Service) ResolveClaimStock(ctx context.Context, retailerID, claimID string, lines []ClaimHoldLine, disposition, actor string) error {
	if s == nil || retailerID == "" || claimID == "" || len(lines) == 0 {
		return nil
	}
	disposition = strings.ToUpper(strings.TrimSpace(disposition))
	if disposition == "" {
		disposition = ClaimStockReturn
	}
	loc, err := s.EnsurePrimaryLocation(ctx, retailerID)
	if err != nil {
		return err
	}
	locID := loc.LocationID
	note := "claim_" + strings.ToLower(disposition) + ":" + claimID
	for _, ln := range lines {
		sku := strings.TrimSpace(ln.SKU)
		need := ln.Qty
		if sku == "" || need <= 0 {
			continue
		}
		avail := s.onHandInBin(ctx, locID, BinQuarantine, sku)
		if avail <= 0 {
			continue
		}
		take := need
		if take > avail {
			take = avail
		}
		switch disposition {
		case ClaimStockRestore:
			_ = s.applyTransferTyped(ctx, retailerID, locID, locID, BinQuarantine, BinFloor, sku, take, MoveClaimRestore, "CLAIM", actor, note)
		case ClaimStockWaste:
			_ = s.applyDelta(ctx, retailerID, locID, BinQuarantine, sku, -take, MoveClaimRelease, "CLAIM", claimID, actor, note+":waste")
		default: // RETURN — leave store via reverse logistics
			_ = s.applyDelta(ctx, retailerID, locID, BinQuarantine, sku, -take, MoveClaimRelease, "CLAIM", claimID, actor, note+":return")
		}
		_ = s.syncReorderCurrentStock(ctx, retailerID, sku)
	}
	return nil
}

func (s *Service) onHandInBin(ctx context.Context, locationID, bin, sku string) int64 {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		if s.stockBalances == nil {
			return 0
		}
		b := s.stockBalances[stockBalanceKey{locationID, bin, sku}]
		return b.OnHand
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerStockBalances",
		spanner.Key{locationID, bin, sku}, []string{"OnHand"})
	if err != nil {
		return 0
	}
	var onHand int64
	_ = row.Columns(&onHand)
	return onHand
}

func (s *Service) applyAdjust(ctx context.Context, retailerID, locationID, bin, sku string, delta int64, actor, note string) error {
	moveType := MoveAdjust
	if strings.HasPrefix(note, "cycle_count:") {
		moveType = MoveCountVariance
	}
	return s.applyDelta(ctx, retailerID, locationID, bin, sku, delta, moveType, "ADJUST", "", actor, note)
}

func (s *Service) confirmReceiveSession(ctx context.Context, session ReceiveSessionDTO, bin, actor string) error {
	// Persist session as confirmed and apply RECEIVE deltas.
	buf := &spannerTxnBuffer{}
	for _, line := range session.Lines {
		if line.AcceptedQty > 0 {
			if err := s.applyDelta(ctx, session.RetailerID, session.LocationID, bin, line.Sku, line.AcceptedQty, MoveReceive, "ORDER", session.OrderID, actor, "receive:accepted"); err != nil {
				return err
			}
		}

		varianceQty := int64(0)
		if line.DamagedQty > 0 {
			varianceQty += line.DamagedQty
			// Route damage direct to QUARANTINE bin so the claim system can evaluate it
			if err := s.applyDelta(ctx, session.RetailerID, session.LocationID, BinQuarantine, line.Sku, line.DamagedQty, MoveReceive, "ORDER", session.OrderID, actor, "receive:damaged"); err != nil {
				return err
			}

			// Bridge to claims/quarantine via Outbox event
			_ = outbox.EmitJSON(ctx, buf, events.AggregateRetailer, session.RetailerID, events.TopicMain, map[string]any{
				"type":        events.EventReceivingVarianceReported,
				"timestamp":   s.now().Format(time.RFC3339Nano),
				"retailer_id": session.RetailerID,
				"location_id": session.LocationID,
				"order_id":    session.OrderID,
				"sku":         line.Sku,
				"qty":         line.DamagedQty,
				"condition":   "DAMAGED",
				"reported_by": actor,
			})
		}

		if line.MissingQty > 0 {
			varianceQty += line.MissingQty
			// Missing goods don't enter inventory, but we must report the variance
			_ = outbox.EmitJSON(ctx, buf, events.AggregateRetailer, session.RetailerID, events.TopicMain, map[string]any{
				"type":        events.EventReceivingVarianceReported,
				"timestamp":   s.now().Format(time.RFC3339Nano),
				"retailer_id": session.RetailerID,
				"location_id": session.LocationID,
				"order_id":    session.OrderID,
				"sku":         line.Sku,
				"qty":         line.MissingQty,
				"condition":   "MISSING",
				"reported_by": actor,
			})
		}

		_ = s.syncReorderCurrentStock(ctx, session.RetailerID, line.Sku)
	}

	if s.spannerClient != nil && (len(buf.events) > 0 || len(buf.audits) > 0) {
		_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			return buf.Flush(txn)
		})
		if err != nil {
			return err
		}
	}

	session.Status = "CONFIRMED"
	return s.saveReceiveSession(ctx, session, actor)
}

func (s *Service) saveReceiveSession(ctx context.Context, session ReceiveSessionDTO, actor string) error {
	linesJSON, _ := json.Marshal(session.Lines)
	if s.spannerClient == nil {
		s.mu.Lock()
		defer s.mu.Unlock()
		if s.receiveSessions == nil {
			s.receiveSessions = map[string]ReceiveSessionDTO{}
		}
		s.receiveSessions[session.SessionID] = session
		s.receiveByOrder[session.OrderID] = session.SessionID
		return nil
	}
	row := map[string]any{
		"SessionId":  session.SessionID,
		"RetailerId": session.RetailerID,
		"LocationId": session.LocationID,
		"OrderId":    session.OrderID,
		"Status":     session.Status,
		"LinesJson":  string(linesJSON),
		"CreatedBy":  nullableStr(actor),
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if session.Status == "CONFIRMED" {
		row["ConfirmedAt"] = spanner.CommitTimestamp
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertOrUpdateMap("RetailerReceiveSessions", row)})
	return err
}

func (s *Service) getReceiveSession(ctx context.Context, sessionID string) (ReceiveSessionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		ss, ok := s.receiveSessions[sessionID]
		return ss, ok, nil
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "RetailerReceiveSessions", spanner.Key{sessionID},
		[]string{"SessionId", "RetailerId", "LocationId", "OrderId", "Status", "LinesJson", "CreatedAt", "ConfirmedAt"})
	if err != nil {
		if isNotFound(err) {
			return ReceiveSessionDTO{}, false, nil
		}
		return ReceiveSessionDTO{}, false, err
	}
	return decodeReceiveSessionRow(row)
}

func (s *Service) getReceiveSessionByOrder(ctx context.Context, orderID string) (ReceiveSessionDTO, bool, error) {
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		sid, ok := s.receiveByOrder[orderID]
		if !ok {
			return ReceiveSessionDTO{}, false, nil
		}
		ss, ok2 := s.receiveSessions[sid]
		return ss, ok2, nil
	}
	stmt := spanner.Statement{
		SQL: `SELECT SessionId, RetailerId, LocationId, OrderId, Status, LinesJson, CreatedAt, ConfirmedAt
			FROM RetailerReceiveSessions@{FORCE_INDEX=UQ_RetailerReceiveSessions_ByOrder}
			WHERE OrderId = @oid LIMIT 1`,
		Params: map[string]any{"oid": orderID},
	}
	iter := s.spannerClient.Single().Query(ctx, stmt)
	defer iter.Stop()
	row, err := iter.Next()
	if err == iterator.Done {
		return ReceiveSessionDTO{}, false, nil
	}
	if err != nil {
		return ReceiveSessionDTO{}, false, err
	}
	return decodeReceiveSessionRow(row)
}

func decodeReceiveSessionRow(row *spanner.Row) (ReceiveSessionDTO, bool, error) {
	var ss ReceiveSessionDTO
	var linesJSON string
	var created time.Time
	var confirmed spanner.NullTime
	if err := row.Columns(&ss.SessionID, &ss.RetailerID, &ss.LocationID, &ss.OrderID, &ss.Status, &linesJSON, &created, &confirmed); err != nil {
		return ReceiveSessionDTO{}, false, err
	}
	_ = json.Unmarshal([]byte(linesJSON), &ss.Lines)
	ss.CreatedAt = created.UTC().Format(time.RFC3339Nano)
	if confirmed.Valid {
		ss.ConfirmedAt = confirmed.Time.UTC().Format(time.RFC3339Nano)
	}
	return ss, true, nil
}

func (s *Service) loadOrderLinesForReceive(ctx context.Context, retailerID, orderID string) ([]ReceiveLine, error) {
	if s.spannerClient == nil {
		// Memory: allow synthetic lines from test/manual inject via empty error
		return nil, errors.New("order_not_found")
	}
	row, err := s.spannerClient.Single().ReadRow(ctx, "Orders", spanner.Key{orderID},
		[]string{"RetailerId", "Status", "LineItemsJson"})
	if err != nil {
		if isNotFound(err) {
			return nil, errors.New("order_not_found")
		}
		return nil, err
	}
	var rid, status string
	var lineBytes []byte
	if err := row.Columns(&rid, &status, &lineBytes); err != nil {
		return nil, err
	}
	if rid != retailerID {
		return nil, errors.New("order_not_owned")
	}
	// Allow receive when delivered-ish or completed.
	st := strings.ToUpper(status)
	switch st {
	case "COMPLETED", "ARRIVED", "AWAITING_PAYMENT", "PENDING_CASH_COLLECTION", "DELIVERED_ON_CREDIT", "FISCALIZING", "FISCAL_FAILED":
		// ok
	default:
		return nil, fmt.Errorf("order_status_not_receivable:%s", st)
	}
	// LineItemsJson is the order qty source of truth for receive.
	// Cap to delivered − remaining/offload − open logistics claim qty (G9).
	var items []struct {
		SKU         string `json:"sku"`
		Name        string `json:"name"`
		ProductName string `json:"product_name"`
		Quantity    int64  `json:"quantity"`
		Qty         int64  `json:"qty"`
		Delivered   int64  `json:"delivered_qty"`
		Remaining   int64  `json:"remaining_qty"`
		OffloadQty  int64  `json:"offload_qty"`
	}
	if err := json.Unmarshal(lineBytes, &items); err != nil {
		return nil, errors.New("order_lines_invalid")
	}
	claimedBySKU := s.openClaimQtyBySKU(ctx, orderID)
	var lines []ReceiveLine
	for _, it := range items {
		sku := strings.TrimSpace(it.SKU)
		if sku == "" {
			continue
		}
		ordered := it.Quantity
		if ordered == 0 {
			ordered = it.Qty
		}
		qty := ReceivableQty(ordered, it.Delivered, it.Remaining, it.OffloadQty, claimedBySKU[sku])
		name := it.Name
		if name == "" {
			name = it.ProductName
		}
		lines = append(lines, ReceiveLine{Sku: sku, ProductName: name, OrderedQty: qty, AcceptedQty: qty})
	}
	if len(lines) == 0 {
		return nil, errors.New("order_has_no_lines")
	}
	return lines, nil
}

// ReceivableQty caps putaway to units actually left with the retailer.
// When delivered_qty is set it already excludes remaining/offload (Delivered+Remaining==Quantity).
// Otherwise subtract remaining/offload from ordered. Always subtract open logistics claim qty.
func ReceivableQty(ordered, delivered, remaining, offload, openClaimed int64) int64 {
	var base int64
	if delivered > 0 {
		base = delivered
	} else {
		excepted := remaining
		if offload > excepted {
			excepted = offload
		}
		base = ordered - excepted
	}
	out := base - openClaimed
	if out < 0 {
		return 0
	}
	return out
}

// openClaimQtyBySKU sums OPEN/UNDER_REVIEW/APPROVED claim line qtys for an order (best-effort).
func (s *Service) openClaimQtyBySKU(ctx context.Context, orderID string) map[string]int64 {
	out := map[string]int64{}
	if s == nil || s.spannerClient == nil || strings.TrimSpace(orderID) == "" {
		return out
	}
	iter := s.spannerClient.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT Status, LineItemsJSON FROM Claims@{FORCE_INDEX=Idx_Claims_ByOrderCreated}
			WHERE OrderId = @oid ORDER BY CreatedAt DESC LIMIT 50`,
		Params: map[string]any{"oid": strings.TrimSpace(orderID)},
	})
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			s.log.Warn("open claim qty lookup failed", "order_id", orderID, "err", err)
			return out
		}
		var status string
		var lineBytes []byte
		if err := row.Columns(&status, &lineBytes); err != nil {
			continue
		}
		switch strings.ToUpper(strings.TrimSpace(status)) {
		case "OPEN", "UNDER_REVIEW", "APPROVED", "RESOLVED":
		default:
			continue
		}
		var lines []struct {
			SKU      string `json:"sku"`
			Quantity int64  `json:"quantity"`
			Qty      int64  `json:"qty"`
		}
		if err := json.Unmarshal(lineBytes, &lines); err != nil {
			continue
		}
		for _, ln := range lines {
			sku := strings.TrimSpace(ln.SKU)
			q := ln.Quantity
			if q == 0 {
				q = ln.Qty
			}
			if sku == "" || q <= 0 {
				continue
			}
			out[sku] += q
		}
	}
	return out
}

func (s *Service) listStockMovements(ctx context.Context, retailerID, locationID, sku string, limit int) ([]StockMovementDTO, error) {
	if limit <= 0 {
		limit = 50
	}
	if s.spannerClient == nil {
		s.mu.RLock()
		defer s.mu.RUnlock()
		var out []StockMovementDTO
		for i := len(s.stockMovements) - 1; i >= 0 && len(out) < limit; i-- {
			m := s.stockMovements[i]
			// no retailer on DTO in memory — filter loosely
			if locationID != "" && m.LocationID != locationID {
				continue
			}
			if sku != "" && m.Sku != sku {
				continue
			}
			out = append(out, m)
		}
		return out, nil
	}
	var indexHint string
	if sku != "" {
		indexHint = "@{FORCE_INDEX=Idx_RetailerStockMovements_ByRetailerSku}"
	} else if locationID != "" {
		indexHint = "@{FORCE_INDEX=Idx_RetailerStockMovements_ByLocationCreated}"
	}
	sql := fmt.Sprintf(`SELECT MovementId, LocationId, StockBin, Sku, Qty, MovementType, IFNULL(RefType,''), IFNULL(RefId,''), IFNULL(ActorUserId,''), IFNULL(Note,''), CreatedAt
		FROM RetailerStockMovements%s
		WHERE RetailerId = @rid`, indexHint)
	params := map[string]any{"rid": retailerID}
	if locationID != "" {
		sql += ` AND LocationId = @lid`
		params["lid"] = locationID
	}
	if sku != "" {
		sql += ` AND Sku = @sku`
		params["sku"] = sku
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`
	params["lim"] = int64(limit)
	var iter *spanner.RowIterator
	if txn := spannerutils.ReadOnlyTxnFromContext(ctx); txn != nil {
		iter = txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	} else {
		iter = s.spannerClient.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	}
	defer iter.Stop()
	var out []StockMovementDTO
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var m StockMovementDTO
		var created time.Time
		if err := row.Columns(&m.MovementID, &m.LocationID, &m.StockBin, &m.Sku, &m.Qty, &m.MovementType, &m.RefType, &m.RefID, &m.ActorUserID, &m.Note, &created); err != nil {
			return nil, err
		}
		m.CreatedAt = created.UTC().Format(time.RFC3339Nano)
		out = append(out, m)
	}
	return out, nil
}

func (s *Service) saveStockCount(ctx context.Context, countID, retailerID, locationID, status, linesJSON, actor string, committed bool) error {
	if s.spannerClient == nil {
		return nil
	}
	row := map[string]any{
		"CountId":    countID,
		"RetailerId": retailerID,
		"LocationId": locationID,
		"Status":     status,
		"LinesJson":  linesJSON,
		"CreatedBy":  nullableStr(actor),
		"CreatedAt":  spanner.CommitTimestamp,
	}
	if committed {
		row["CommittedAt"] = spanner.CommitTimestamp
	}
	_, err := s.spannerClient.Apply(ctx, []*spanner.Mutation{spanner.InsertMap("RetailerStockCounts", row)})
	return err
}

// syncReorderCurrentStock writes store on-hand into ReorderSuggestions.CurrentStock when row exists.
func (s *Service) syncReorderCurrentStock(ctx context.Context, retailerID, sku string) error {
	if s.spannerClient == nil {
		return nil
	}
	total, err := s.SumOnHandForSKU(ctx, retailerID, sku)
	if err != nil {
		return err
	}
	// Best-effort update; ignore missing suggestion rows.
	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		_, err := txn.Update(ctx, spanner.Statement{
			SQL: `UPDATE ReorderSuggestions SET CurrentStock = @qty, ComputedAt = CURRENT_TIMESTAMP()
				WHERE RetailerId = @rid AND Sku = @sku`,
			Params: map[string]any{"qty": total, "rid": retailerID, "sku": sku},
		})
		return err
	})
	return err
}

// InjectTestOrderLines seeds order line load for memory tests via receive without Spanner.
// Not used in production.
func (s *Service) injectMemoryReceive(orgID, locID, orderID string, lines []ReceiveLine) error {
	session := ReceiveSessionDTO{
		SessionID: s.newID(), RetailerID: orgID, LocationID: locID, OrderID: orderID,
		Status: "DRAFT", Lines: lines, CreatedAt: s.now().UTC().Format(time.RFC3339Nano),
	}
	return s.confirmReceiveSession(context.Background(), session, BinBackroom, "test")
}

func (s *Service) applyDeltaInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, retailerID, locationID, bin, sku string, delta int64, moveType, refType, refID, actor, note string) error {
	var onHand, reserved int64
	row, err := txn.ReadRow(ctx, "RetailerStockBalances", spanner.Key{locationID, bin, sku},
		[]string{"OnHand", "Reserved"})
	if err != nil && !isNotFound(err) {
		return err
	}
	if err == nil {
		_ = row.Columns(&onHand, &reserved)
	}
	next := onHand + delta
	if next < 0 {
		return errors.New("insufficient_stock")
	}
	muts := []*spanner.Mutation{
		spanner.InsertOrUpdateMap("RetailerStockBalances", map[string]any{
			"LocationId": locationID,
			"StockBin":   bin,
			"Sku":        sku,
			"RetailerId": retailerID,
			"OnHand":     next,
			"Reserved":   reserved,
			"UpdatedAt":  spanner.CommitTimestamp,
		}),
		spanner.InsertMap("RetailerStockMovements", map[string]any{
			"MovementId":   s.newID(),
			"RetailerId":   retailerID,
			"LocationId":   locationID,
			"StockBin":     bin,
			"Sku":          sku,
			"Qty":          delta,
			"MovementType": moveType,
			"RefType":      nullableStr(refType),
			"RefId":        nullableStr(refID),
			"ActorUserId":  nullableStr(actor),
			"Note":         nullableStr(note),
			"CreatedAt":    spanner.CommitTimestamp,
		}),
	}
	return txn.BufferWrite(muts)
}

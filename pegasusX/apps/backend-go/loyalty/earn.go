package loyalty

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/grpc/codes"
)

const (
	defaultEarnBps        int64 = 100
	MinorUnitsPerMajor    int64 = 100
)

// DefaultTiers is STANDARD/SILVER/GOLD on lifetime points.
var DefaultTiers = []Tier{
	{Name: "STANDARD", MinPoints: 0},
	{Name: "SILVER", MinPoints: 50000},
	{Name: "GOLD", MinPoints: 200000},
}

type Tier struct {
	Name      string `json:"name"`
	MinPoints int64  `json:"min_points"`
}

type Program struct {
	SupplierID string `json:"supplier_id"`
	EarnBps    int64  `json:"earn_bps"`
	Tiers      []Tier `json:"tiers"`
	Reason     string `json:"reason,omitempty"`
	Source     string `json:"source,omitempty"`
}

type TierView struct {
	Enrolled        bool   `json:"enrolled"`
	Tier            string `json:"tier,omitempty"`
	LifetimePoints  int64  `json:"lifetime_points"`
	AvailablePoints int64  `json:"available_points"`
	NextTier        string `json:"next_tier,omitempty"`
	PointsToNext    int64  `json:"points_to_next,omitempty"`
	EarnBps         int64  `json:"earn_bps,omitempty"`
	SupplierID      string `json:"supplier_id,omitempty"`
}

type LedgerEntry struct {
	LedgerID    string `json:"ledger_id"`
	OrderID     string `json:"order_id"`
	Points      int64  `json:"points"`
	EarnBps     int64  `json:"earn_bps"`
	AmountMinor int64  `json:"amount_minor"`
	CreatedAt   string `json:"created_at"`
}

func PointsFor(amountMinor, earnBps int64) int64 {
	if amountMinor <= 0 || earnBps <= 0 {
		return 0
	}
	return (amountMinor * earnBps) / (MinorUnitsPerMajor * 10000)
}

func ClawbackBalance(lifetimePoints, availablePoints, clawbackPoints int64) (newLifetime, newAvailable int64) {
	if clawbackPoints <= 0 {
		return lifetimePoints, availablePoints
	}
	newLifetime = lifetimePoints - clawbackPoints
	if newLifetime < 0 {
		newLifetime = 0
	}
	newAvailable = availablePoints - clawbackPoints
	if newAvailable < 0 {
		newAvailable = 0
	}
	return newLifetime, newAvailable
}

func TierFor(points int64, tiers []Tier) (current Tier, next *Tier) {
	if len(tiers) == 0 {
		tiers = DefaultTiers
	}
	current = tiers[0]
	for i, t := range tiers {
		if points >= t.MinPoints {
			current = t
			if i+1 < len(tiers) {
				n := tiers[i+1]
				next = &n
			} else {
				next = nil
			}
		}
	}
	return current, next
}

// EarnInTxn writes an idempotent earn row for a captured order. No program → no-op.
func EarnInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, supplierID, retailerID, orderID string, amountMinor int64) error {
	supplierID = strings.TrimSpace(supplierID)
	retailerID = strings.TrimSpace(retailerID)
	orderID = strings.TrimSpace(orderID)
	if txn == nil || supplierID == "" || retailerID == "" || orderID == "" || amountMinor <= 0 {
		return nil
	}
	prog, err := readProgram(ctx, txn, supplierID)
	if err != nil {
		return err
	}
	if prog == nil {
		return nil
	}
	pts := PointsFor(amountMinor, prog.EarnBps)
	if pts <= 0 {
		return nil
	}
	ledgerID := "earn:" + orderID
	_, err = txn.ReadRow(ctx, "LoyaltyLedger", spanner.Key{supplierID, ledgerID}, []string{"LedgerId"})
	if err == nil {
		return nil
	}
	if spanner.ErrCode(err) != codes.NotFound {
		return err
	}
	muts := []*spanner.Mutation{
		spanner.InsertMap("LoyaltyLedger", map[string]any{
			"SupplierId":  supplierID,
			"LedgerId":    ledgerID,
			"RetailerId":  retailerID,
			"OrderId":     orderID,
			"Points":      pts,
			"EarnBps":     prog.EarnBps,
			"AmountMinor": amountMinor,
			"CreatedAt":   spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("LoyaltyAccounts", map[string]any{
			"SupplierId":      supplierID,
			"RetailerId":      retailerID,
			"LifetimePoints":  spanner.NullInt64{}, // filled below via read
			"AvailablePoints": spanner.NullInt64{},
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	}
	life, avail := int64(0), int64(0)
	acc, accErr := txn.ReadRow(ctx, "LoyaltyAccounts", spanner.Key{supplierID, retailerID}, []string{"LifetimePoints", "AvailablePoints"})
	if accErr == nil {
		_ = acc.Columns(&life, &avail)
	} else if spanner.ErrCode(accErr) != codes.NotFound {
		return accErr
	}
	muts[1] = spanner.InsertOrUpdateMap("LoyaltyAccounts", map[string]any{
		"SupplierId":      supplierID,
		"RetailerId":      retailerID,
		"LifetimePoints":  life + pts,
		"AvailablePoints": avail + pts,
		"UpdatedAt":       spanner.CommitTimestamp,
	})
	if buf != nil {
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, map[string]any{
			"type":         events.EventLoyaltyPointsEarned,
			"supplier_id":  supplierID,
			"retailer_id":  retailerID,
			"order_id":     orderID,
			"points":       pts,
			"amount_minor": amountMinor,
		}); err != nil {
			return err
		}
	}
	return txn.BufferWrite(muts)
}

func deriveClawbackLedgerID(orderID, refundID string, pts, refundAmountMinor int64) (string, string) {
	if refundID != "" {
		return "clawback:" + orderID + ":" + refundID, orderID + "#clawback:" + refundID
	}
	return fmt.Sprintf("clawback:%s:pts%d:amt%d", orderID, pts, refundAmountMinor),
		fmt.Sprintf("%s#clawback:pts%d:amt%d", orderID, pts, refundAmountMinor)
}

// ClawbackInTxn writes an idempotent clawback row for a refunded order/amount. No program → no-op.
func ClawbackInTxn(ctx context.Context, txn *spanner.ReadWriteTransaction, buf outbox.TxnBuffer, supplierID, retailerID, orderID, refundID string, refundAmountMinor int64) error {
	supplierID = strings.TrimSpace(supplierID)
	retailerID = strings.TrimSpace(retailerID)
	orderID = strings.TrimSpace(orderID)
	refundID = strings.TrimSpace(refundID)
	if txn == nil || supplierID == "" || retailerID == "" || orderID == "" || refundAmountMinor <= 0 {
		return nil
	}
	prog, err := readProgram(ctx, txn, supplierID)
	if err != nil {
		return err
	}
	if prog == nil {
		return nil
	}
	pts := PointsFor(refundAmountMinor, prog.EarnBps)
	if pts <= 0 {
		return nil
	}

	ledgerID, clawbackOrderID := deriveClawbackLedgerID(orderID, refundID, pts, refundAmountMinor)

	_, err = txn.ReadRow(ctx, "LoyaltyLedger", spanner.Key{supplierID, ledgerID}, []string{"LedgerId"})
	if err == nil {
		if refundID != "" {
			return nil
		}
		// Subsequent partial refunds on the same order with refundID == "" and identical amount get a unique suffix
		found := false
		for i := 2; i <= 100; i++ {
			candidateID := fmt.Sprintf("%s#%d", ledgerID, i)
			_, cErr := txn.ReadRow(ctx, "LoyaltyLedger", spanner.Key{supplierID, candidateID}, []string{"LedgerId"})
			if spanner.ErrCode(cErr) == codes.NotFound {
				ledgerID = candidateID
				clawbackOrderID = fmt.Sprintf("%s#%d", clawbackOrderID, i)
				found = true
				break
			}
		}
		if !found {
			return nil
		}
	} else if spanner.ErrCode(err) != codes.NotFound {
		return err
	}

	life, avail := int64(0), int64(0)
	acc, accErr := txn.ReadRow(ctx, "LoyaltyAccounts", spanner.Key{supplierID, retailerID}, []string{"LifetimePoints", "AvailablePoints"})
	if accErr == nil {
		_ = acc.Columns(&life, &avail)
	} else if spanner.ErrCode(accErr) != codes.NotFound {
		return accErr
	}

	newLife, newAvail := ClawbackBalance(life, avail, pts)

	muts := []*spanner.Mutation{
		spanner.InsertMap("LoyaltyLedger", map[string]any{
			"SupplierId":  supplierID,
			"LedgerId":    ledgerID,
			"RetailerId":  retailerID,
			"OrderId":     clawbackOrderID,
			"Points":      -pts,
			"EarnBps":     prog.EarnBps,
			"AmountMinor": -refundAmountMinor,
			"CreatedAt":   spanner.CommitTimestamp,
		}),
		spanner.InsertOrUpdateMap("LoyaltyAccounts", map[string]any{
			"SupplierId":      supplierID,
			"RetailerId":      retailerID,
			"LifetimePoints":  newLife,
			"AvailablePoints": newAvail,
			"UpdatedAt":       spanner.CommitTimestamp,
		}),
	}

	if buf != nil {
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, map[string]any{
			"type":         "LOYALTY_POINTS_CLAWED_BACK",
			"supplier_id":  supplierID,
			"retailer_id":  retailerID,
			"order_id":     orderID,
			"refund_id":    refundID,
			"points":       pts,
			"amount_minor": refundAmountMinor,
		}); err != nil {
			return err
		}
	}
	return txn.BufferWrite(muts)
}

func readProgram(ctx context.Context, txn *spanner.ReadWriteTransaction, supplierID string) (*Program, error) {
	row, err := txn.ReadRow(ctx, "LoyaltyPrograms", spanner.Key{supplierID}, []string{"EarnBps", "TiersJson"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var bps int64
	var tiersJSON spanner.NullString
	if err := row.Columns(&bps, &tiersJSON); err != nil {
		return nil, err
	}
	p := &Program{SupplierID: supplierID, EarnBps: bps, Tiers: DefaultTiers, Source: "program"}
	if bps < 0 {
		p.EarnBps = defaultEarnBps
	}
	if tiersJSON.Valid && strings.TrimSpace(tiersJSON.StringVal) != "" {
		var tiers []Tier
		if json.Unmarshal([]byte(tiersJSON.StringVal), &tiers) == nil && len(tiers) > 0 {
			p.Tiers = tiers
		}
	}
	return p, nil
}

func ReadProgramStrong(ctx context.Context, client *spanner.Client, supplierID string) (*Program, error) {
	if client == nil {
		return nil, fmt.Errorf("loyalty_unavailable")
	}
	row, err := client.Single().ReadRow(ctx, "LoyaltyPrograms", spanner.Key{supplierID}, []string{"EarnBps", "TiersJson"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return nil, nil
		}
		return nil, err
	}
	var bps int64
	var tiersJSON spanner.NullString
	if err := row.Columns(&bps, &tiersJSON); err != nil {
		return nil, err
	}
	p := &Program{SupplierID: supplierID, EarnBps: bps, Tiers: DefaultTiers, Source: "program"}
	if bps < 0 {
		p.EarnBps = defaultEarnBps
	}
	if tiersJSON.Valid && strings.TrimSpace(tiersJSON.StringVal) != "" {
		var tiers []Tier
		if json.Unmarshal([]byte(tiersJSON.StringVal), &tiers) == nil && len(tiers) > 0 {
			p.Tiers = tiers
		}
	}
	return p, nil
}

func ReadAccount(ctx context.Context, client *spanner.Client, supplierID, retailerID string) (life, avail int64, err error) {
	if client == nil {
		return 0, 0, nil
	}
	row, err := client.Single().ReadRow(ctx, "LoyaltyAccounts", spanner.Key{supplierID, retailerID}, []string{"LifetimePoints", "AvailablePoints"})
	if err != nil {
		if spanner.ErrCode(err) == codes.NotFound {
			return 0, 0, nil
		}
		return 0, 0, err
	}
	if err := row.Columns(&life, &avail); err != nil {
		return 0, 0, err
	}
	return life, avail, nil
}

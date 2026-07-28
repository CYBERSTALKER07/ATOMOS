package returns

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// TicketLine is one SKU quantity expected back at the warehouse dock.
type TicketLine struct {
	SKU      string
	Quantity int64
	Reason   string
}

// OpenTicketsInput creates reverse-logistics inbound rows for a claim / OS&D.
type OpenTicketsInput struct {
	OrderID     string
	WarehouseID string
	SupplierID  string
	DriverID    string
	ManifestID  string
	ClaimID     string
	Source      string // CLAIM | DRIVER_EXCEPTION
	Note        string
	Lines       []TicketLine
}

// OpenTickets upserts expected inbound SupplierReturns for claim/OS&D reverse logistics.
//
// Dedupes open rows for the same order+sku so driver amend + claim bridge do not double-book.
// Claim-sourced tickets use PhysicalStatus=ARRIVED so they appear on the dock inbound list
// without requiring a truck approach event (goods return from store separately).
func (s *Service) OpenTickets(ctx context.Context, in OpenTicketsInput) (returnIDs []string, err error) {
	if s == nil || s.spanner == nil {
		return nil, nil
	}
	orderID := strings.TrimSpace(in.OrderID)
	if orderID == "" {
		return nil, fmt.Errorf("order_id_required")
	}
	warehouseID := strings.TrimSpace(in.WarehouseID)
	lines := make([]TicketLine, 0, len(in.Lines))
	for _, ln := range in.Lines {
		sku := strings.TrimSpace(ln.SKU)
		if sku == "" || ln.Quantity <= 0 {
			continue
		}
		if !shouldOpenReturnTicket(ln.Reason, in.Source) {
			continue
		}
		reason := normalizeTicketReason(ln.Reason)
		lines = append(lines, TicketLine{SKU: sku, Quantity: ln.Quantity, Reason: reason})
	}
	if len(lines) == 0 {
		return nil, nil
	}

	claimID := strings.TrimSpace(in.ClaimID)
	note := strings.TrimSpace(in.Note)
	source := strings.ToUpper(strings.TrimSpace(in.Source))
	if source == "" {
		source = "CLAIM"
	}

	_, err = s.spanner.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		mutations := make([]*spanner.Mutation, 0, len(lines))
		for _, ln := range lines {
			exists, err := openReturnExists(ctx, txn, orderID, ln.SKU)
			if err != nil {
				return err
			}
			if exists {
				continue
			}
			returnID := s.newID()
			notes := buildTicketNotes(source, claimID, note)
			// Claim reverse logistics: dock-visible without truck telemetry.
			// Driver OS&D usually already created PENDING via amend; when opening here
			// use PENDING so return-complete / approach can still advance the lifecycle.
			physical := PhysicalPending
			if source == "CLAIM" || source == "RETAILER_CLAIM" {
				physical = PhysicalArrived
			}
			mutations = append(mutations, spanner.InsertMap("SupplierReturns", map[string]any{
				"ReturnId":       returnID,
				"OrderId":        orderID,
				"SkuId":          ln.SKU,
				"RejectedQty":    ln.Quantity,
				"Reason":         ln.Reason,
				"DriverNotes":    nullableString(notes),
				"Status":         FinancialPending,
				"ManifestId":     nullableString(in.ManifestID),
				"DriverId":       nullableString(in.DriverID),
				"WarehouseId":    nullableString(warehouseID),
				"ExpectedQty":    ln.Quantity,
				"ReceivedQty":    int64(0),
				"PhysicalStatus": physical,
				"CreatedAt":      spanner.CommitTimestamp,
			}))
			returnIDs = append(returnIDs, returnID)
		}
		if len(mutations) == 0 {
			return nil
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		return nil, err
	}
	if len(returnIDs) > 0 && s.log != nil {
		s.log.InfoContext(ctx, "reverse logistics tickets opened",
			"order_id", orderID,
			"claim_id", claimID,
			"count", len(returnIDs),
			"source", source,
		)
	}
	return returnIDs, nil
}

func openReturnExists(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID, sku string) (bool, error) {
	iter := txn.Query(ctx, spanner.Statement{
		SQL: `SELECT ReturnId FROM SupplierReturns
		      WHERE OrderId = @order_id AND SkuId = @sku
		        AND PhysicalStatus IN UNNEST(@open_phys)
		        AND Status = @fin_pending
		      LIMIT 1`,
		Params: map[string]any{
			"order_id":    orderID,
			"sku":         sku,
			"open_phys":   []string{PhysicalPending, PhysicalOnTruck, PhysicalArrived, PhysicalReceiving},
			"fin_pending": FinancialPending,
		},
	})
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

func shouldOpenReturnTicket(reason, source string) bool {
	r := strings.ToUpper(strings.TrimSpace(reason))
	switch r {
	case "DAMAGED", "WRONG_ITEM", "TAMPER", "TEMPERATURE", "CONCEALED_DAMAGE":
		return true
	case "MISSING", "OTHER", "":
		// Missing units are not expected back; OTHER/empty need an explicit damage reason.
		return false
	default:
		// Unknown non-empty reasons: open ticket so warehouse can inspect.
		_ = source
		return true
	}
}

func normalizeTicketReason(reason string) string {
	r := strings.ToUpper(strings.TrimSpace(reason))
	if r == "" {
		return "DAMAGED"
	}
	return r
}

func buildTicketNotes(source, claimID, note string) string {
	parts := make([]string, 0, 3)
	if source != "" {
		parts = append(parts, "source="+source)
	}
	if claimID != "" {
		parts = append(parts, "claim_id="+claimID)
	}
	if note != "" {
		parts = append(parts, note)
	}
	return strings.Join(parts, " | ")
}

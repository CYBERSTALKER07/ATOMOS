package order

import (
	"context"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// LoadDeliveryExpectations batch-loads delivery expectation projections for order IDs.
func LoadDeliveryExpectations(ctx context.Context, client *spanner.Client, now time.Time, orderIDs []string) map[string]DeliveryExpectation {
	out := make(map[string]DeliveryExpectation, len(orderIDs))
	if client == nil || len(orderIDs) == 0 {
		return out
	}
	ids := dedupeOrderIDs(orderIDs)
	if len(ids) == 0 {
		return out
	}
	stmt := spanner.Statement{
		SQL: `SELECT OrderId, SupplierId, Status, Source, ConfirmationStatus, DeliveryPriority,
		             DeliverBefore, RequestedDeliveryDate, ProposedDeliveryDate,
		             ReceivingWindowOpen, ReceivingWindowClose, Timezone
		      FROM Orders
		      WHERE OrderId IN UNNEST(@ids)`,
		Params: map[string]any{"ids": ids},
	}
	iter := client.Single().Query(ctx, stmt)
	defer iter.Stop()
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return out
		}
		o, orderID, ok := scanOrderForExpectation(row)
		if !ok {
			continue
		}

		loc, locErr := resolveCalendarLocation(ctx, o.SupplierID, o.Timezone)
		if locErr != nil {
			continue
		}
		out[orderID] = ComputeDeliveryExpectation(now, loc, o)
	}
	return out
}

func dedupeOrderIDs(orderIDs []string) []string {
	seen := make(map[string]struct{}, len(orderIDs))
	ids := make([]string, 0, len(orderIDs))
	for _, id := range orderIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func scanOrderForExpectation(row *spanner.Row) (Order, string, bool) {
	var (
		orderID, supplierID, status, source, confirmation, priority string
		deliverBefore, requested, proposed                          spanner.NullTime
		windowOpen, windowClose, timezone                           spanner.NullString
	)
	if err := row.Columns(&orderID, &supplierID, &status, &source, &confirmation, &priority,
		&deliverBefore, &requested, &proposed, &windowOpen, &windowClose, &timezone); err != nil {
		return Order{}, "", false
	}
	o := Order{
		OrderID:              orderID,
		SupplierID:           supplierID,
		Status:               Status(status),
		Source:               OrderSource(source),
		ConfirmationStatus:   ConfirmationStatus(confirmation),
		DeliveryPriority:     DeliveryPriority(priority),
		ReceivingWindowOpen:  windowOpen.StringVal,
		ReceivingWindowClose: windowClose.StringVal,
		Timezone:             timezone.StringVal,
	}
	if deliverBefore.Valid {
		t := deliverBefore.Time
		o.DeliverBefore = &t
	}
	if requested.Valid {
		t := requested.Time
		o.RequestedDeliveryDate = &t
	}
	if proposed.Valid {
		t := proposed.Time
		o.ProposedDeliveryDate = &t
	}
	return o, orderID, true
}

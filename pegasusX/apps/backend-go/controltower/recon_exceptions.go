package controltower

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

func (r *SpannerRepository) appendReconciliationBreakExceptions(
	ctx context.Context,
	supplierID string,
	out []Exception,
) ([]Exception, error) {
	breaks, err := r.listReconciliationBreakExceptions(ctx, supplierID)
	if err != nil {
		return nil, err
	}
	return append(out, breaks...), nil
}

// listReconciliationBreakExceptions surfaces open/acknowledged M0.3 four-way
// reconciliation breaks into the supplier Control Tower queue.
func (r *SpannerRepository) listReconciliationBreakExceptions(ctx context.Context, supplierID string) ([]Exception, error) {
	stmt := spanner.Statement{
		SQL: `SELECT BreakId, RunId, Dimension, ObjectType, ObjectId,
		             ExpectedMinor, ActualMinor, DeltaMinor, Currency,
		             Severity, Status, OwnerId, SlaDueAt, ResolutionNote,
		             ControlTowerExceptionId, CreatedAt
		      FROM ReconciliationBreaks
		      WHERE EntityId = @sid
		        AND Status IN ('OPEN', 'ACKNOWLEDGED')`,
		Params: map[string]any{"sid": supplierID},
	}
	iter := r.client.Single().WithTimestampBound(spanner.StrongRead()).Query(ctx, stmt)
	defer iter.Stop()

	var out []Exception
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("query reconciliation break exceptions: %w", err)
		}
		var breakID, runID, dimension, objectType, objectID, currency, severity, status string
		var expectedMinor, actualMinor, deltaMinor int64
		var ownerID, resolutionNote, ctID spanner.NullString
		var slaDue spanner.NullTime
		var createdAt time.Time
		if err := row.Columns(
			&breakID, &runID, &dimension, &objectType, &objectID,
			&expectedMinor, &actualMinor, &deltaMinor, &currency,
			&severity, &status, &ownerID, &slaDue, &resolutionNote,
			&ctID, &createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan reconciliation break exception: %w", err)
		}

		ticketID := breakID
		if ctID.Valid && strings.TrimSpace(ctID.StringVal) != "" {
			ticketID = strings.TrimSpace(ctID.StringVal)
		}
		orderID := ""
		if strings.EqualFold(strings.TrimSpace(objectType), "ORDER") {
			orderID = objectID
		}
		absDelta := deltaMinor
		if absDelta < 0 {
			absDelta = -absDelta
		}
		payloadMap := map[string]any{
			"break_id":        breakID,
			"run_id":          runID,
			"dimension":       dimension,
			"object_type":     objectType,
			"object_id":       objectID,
			"expected_minor":  expectedMinor,
			"actual_minor":    actualMinor,
			"delta_minor":     deltaMinor,
			"currency":        currency,
			"resolution_note": resolutionNote.StringVal,
		}
		if slaDue.Valid {
			payloadMap["sla_due_at"] = slaDue.Time.UTC().Format(time.RFC3339Nano)
		}
		rawPayload, _ := json.Marshal(payloadMap)

		var warehouseID string
		if strings.EqualFold(strings.TrimSpace(objectType), "STOCK_LOT") {
			if idx := strings.IndexByte(objectID, ':'); idx > 0 {
				warehouseID = objectID[:idx]
			}
		}
		out = append(out, Exception{
			ExceptionID:  ticketID,
			Type:         "RECON_" + strings.ToUpper(strings.TrimSpace(dimension)),
			Severity:     severity,
			EntityType:   objectType,
			EntityID:     objectID,
			OrderID:      orderID,
			SupplierID:   supplierID,
			WarehouseID:  warehouseID,
			Status:       status,
			AmountMinor:  absDelta,
			AssignedRole: ownerID.StringVal,
			Payload:      json.RawMessage(rawPayload),
			CreatedAt:    createdAt,
		})
	}
	return out, nil
}

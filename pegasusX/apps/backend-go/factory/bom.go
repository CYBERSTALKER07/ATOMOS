package factory

import (
	"context"
	"errors"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

var ErrMaterialShortage = errors.New("insufficient_raw_materials")

// ValidateBOMAndStartProduction checks if there are enough raw materials to fulfill the supply request.
// If not, it transitions the request to EXCEPTION_MATERIAL and emits an outbox event to alert the Supplier.
func (s *Service) ValidateBOMAndStartProduction(ctx context.Context, requestID, factoryID string) error {
	if s.spannerClient == nil {
		// Mock bypass for memory repo
		return nil
	}

	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		// 1. Fetch requested items for this Supply Request
		stmt := spanner.Statement{
			SQL: `SELECT r.ProductId, r.RequestedQuantity, b.RawMaterialId, b.QuantityRequired
			      FROM WarehouseSupplyRequestItems r
			      JOIN BillOfMaterials b ON b.FinishedProductId = r.ProductId
			      WHERE r.RequestId = @rid`,
			Params: map[string]any{"rid": requestID},
		}
		
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()
		
		materialsNeeded := make(map[string]float64)
		for {
			row, err := iter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				return err
			}
			var productID, rawMaterialID string
			var requested int64
			var qtyRequired float64
			if err := row.Columns(&productID, &requested, &rawMaterialID, &qtyRequired); err != nil {
				continue
			}
			materialsNeeded[rawMaterialID] += (float64(requested) * qtyRequired)
		}

		// Fetch SupplierId
		var supplierID string
		supRow, err := txn.ReadRow(ctx, "WarehouseSupplyRequests", spanner.Key{requestID, factoryID}, []string{"SupplierId"})
		if err == nil {
			supRow.Columns(&supplierID)
		}

		// 2. Check Raw Material Inventory
		isShortage := false
		var muts []*spanner.Mutation
		for rawID, needed := range materialsNeeded {
			var available float64
			row, err := txn.ReadRow(ctx, "FactoryRawInventory", spanner.Key{factoryID, rawID}, []string{"QuantityOnHand"})
			if err != nil {
				isShortage = true
				break
			}
			if err := row.Columns(&available); err != nil {
				return err
			}
			if available < needed {
				isShortage = true
				break
			}
			// Atomic deduction
			muts = append(muts, spanner.UpdateMap("FactoryRawInventory", map[string]any{
				"FactoryId":      factoryID,
				"RawMaterialId":  rawID,
				"QuantityOnHand": available - needed,
				"UpdatedAt":      spanner.CommitTimestamp,
			}))
		}

		// 3. Handle State Transition
		newState := "IN_PRODUCTION"
		if isShortage {
			newState = "EXCEPTION_MATERIAL"
			muts = nil // discard material deductions
		}

		muts = append(muts, spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
			"RequestId": requestID,
			"FactoryId": factoryID,
			"State":     newState,
			"UpdatedAt": spanner.CommitTimestamp,
		}))

		if err := txn.BufferWrite(muts); err != nil {
			return err
		}

		// 4. Alert Supplier Portal on Shortage via Outbox
		if isShortage {
			buf := &spannerTxnBuffer{}
			payload := map[string]any{
				"type":       "EventMaterialShortage",
				"request_id": requestID,
				"factory_id": factoryID,
			}
			if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, requestID, events.TopicMain, payload); err != nil {
				return err
			}
			return txn.BufferWrite(outboxMutations(buf.events))
		}

		return nil
	})

	if err != nil && !errors.Is(err, ErrMaterialShortage) {
		s.log.Error("Failed to validate BOM for production run", "request_id", requestID, "err", err)
	}
	
	return err
}

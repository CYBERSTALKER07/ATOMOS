package warehouse

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"

	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/inventory"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/stocklots"
)

type receiveLineInput struct {
	ItemID           string
	ReceivedQuantity int64
	LocationID       string
	LotCode          string
	ExpiryDate       string
}

func (s *Service) receiveTransferWithItems(ctx context.Context, ops *auth.WarehouseOps, transferID string, lines []receiveLineInput) error {
	if s.spannerClient == nil {
		return s.receiveTransfer(ctx, ops, transferID, nil)
	}
	_, err := s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "FactoryInternalTransfers", spanner.Key{transferID},
			[]string{"TransferId", "SupplierId", "State", "WarehouseId", "SupplyRequestId"})
		if err != nil {
			return errTransferNotFound
		}
		var id, supplierID, state, warehouseID, supplyRequestID string
		if err := row.Columns(&id, &supplierID, &state, &warehouseID, &supplyRequestID); err != nil {
			return err
		}
		if supplierID != ops.SupplierID {
			return errTransferForbidden
		}
		state = strings.ToUpper(strings.TrimSpace(state))
		if _, ok := receiveableTransferStates[state]; !ok && state != "APPROVED" {
			return fmt.Errorf("%w: state %s", errInvalidTransfer, state)
		}

		receivedByID := map[string]int64{}
		receivedByLocation := map[string]string{}
		receivedByLotCode := map[string]string{}
		receivedByExpiry := map[string]string{}
		for _, line := range lines {
			if id := strings.TrimSpace(line.ItemID); id != "" && line.ReceivedQuantity > 0 {
				receivedByID[id] = line.ReceivedQuantity
				if loc := strings.TrimSpace(line.LocationID); loc != "" {
					receivedByLocation[id] = loc
				}
				if lc := strings.TrimSpace(line.LotCode); lc != "" {
					receivedByLotCode[id] = lc
				}
				if exp := strings.TrimSpace(line.ExpiryDate); exp != "" {
					receivedByExpiry[id] = exp
				}
			}
		}

		if supplyRequestID != "" {
			itemStmt := spanner.Statement{
				SQL: `SELECT ItemId, ProductId, COALESCE(ShippedQuantity, RequestedQuantity), COALESCE(ReceivedQuantity, 0)
				      FROM WarehouseSupplyRequestItems WHERE RequestId = @rid`,
				Params: map[string]any{"rid": supplyRequestID},
			}
			iter := txn.Query(ctx, itemStmt)
			defer iter.Stop()
			for {
				irow, err := iter.Next()
				if err == iterator.Done {
					break
				}
				if err != nil {
					return err
				}
				var itemID, productID string
				var shipped, alreadyReceived int64
				if err := irow.Columns(&itemID, &productID, &shipped, &alreadyReceived); err != nil {
					continue
				}
				received := shipped
				if v, ok := receivedByID[itemID]; ok && v > 0 {
					received = v
				}
				if received <= 0 {
					continue
				}
				if stocklots.LotsEnabled() {
					locID := strings.TrimSpace(receivedByLocation[itemID])
					if locID == "" {
						locID = "recv-default"
						if _, err := stocklots.UpsertBinInTxn(ctx, txn, stocklots.CreateBinRequest{
							WarehouseID: warehouseID, LocationID: locID, Zone: "RECV",
							LocationType: "STAGE", PickSequence: 0,
						}); err != nil {
							return err
						}
					}
					putReq := stocklots.PutawayRequest{
						SupplierID:  supplierID,
						WarehouseID: warehouseID,
						ProductID:   productID,
						LocationID:  locID,
						LotCode:     receivedByLotCode[itemID],
						Quantity:    received,
					}
					if exp := receivedByExpiry[itemID]; exp != "" {
						if t, err := parseReceiveExpiry(exp); err == nil {
							putReq.ExpiryDate = &t
						}
					}
					if _, err := stocklots.PutawayInTxn(ctx, txn, putReq); err != nil {
						return err
					}
				} else if err := inventory.CreditSupplierInventoryV2InTxn(ctx, txn, supplierID, warehouseID, productID, received); err != nil {
					return err
				}
				if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("WarehouseSupplyRequestItems", map[string]any{
					"RequestId":        supplyRequestID,
					"ItemId":           itemID,
					"ReceivedQuantity": received,
				})}); err != nil {
					return err
				}
			}
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("WarehouseSupplyRequests", map[string]any{
				"RequestId": supplyRequestID,
				"State":     "RECEIVED",
				"UpdatedAt": spanner.CommitTimestamp,
			})}); err != nil {
				return err
			}
		}

		if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("FactoryInternalTransfers", map[string]any{
			"TransferId": transferID,
			"State":      "RECEIVED",
			"ReceivedAt": spanner.CommitTimestamp,
			"UpdatedAt":  spanner.CommitTimestamp,
		})}); err != nil {
			return err
		}
		buf := &spannerTxnBuffer{}
		payload := events.WarehouseEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventWarehouseTransferReceived},
			TransferID: transferID,
			SupplierID: supplierID,
		}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateWarehouse, transferID, events.TopicMain, payload); err != nil {
			return err
		}
		muts := []*spanner.Mutation{}
		muts = append(muts, outboxMutations(buf.events)...)
		return txn.BufferWrite(muts)
	})
	if err == nil && s.cache != nil {
		s.cache.Invalidate(ctx, "catalog:products:"+ops.SupplierID)
	}
	return err
}

func parseReceiveItems(body []byte) []receiveLineInput {
	if len(body) == 0 {
		return nil
	}
	var req struct {
		Items []struct {
			ItemID           string `json:"item_id"`
			ReceivedQuantity int64  `json:"received_quantity"`
			LocationID       string `json:"location_id"`
			LotCode          string `json:"lot_code"`
			ExpiryDate       string `json:"expiry_date"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		return nil
	}
	out := make([]receiveLineInput, 0, len(req.Items))
	for _, row := range req.Items {
		if strings.TrimSpace(row.ItemID) == "" {
			continue
		}
		out = append(out, receiveLineInput{
			ItemID:           strings.TrimSpace(row.ItemID),
			ReceivedQuantity: row.ReceivedQuantity,
			LocationID:       strings.TrimSpace(row.LocationID),
			LotCode:          strings.TrimSpace(row.LotCode),
			ExpiryDate:       strings.TrimSpace(row.ExpiryDate),
		})
	}
	return out
}

func parseReceiveExpiry(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

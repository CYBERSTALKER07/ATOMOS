package stocklots

import (
	"context"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"google.golang.org/api/iterator"
)

// InitiateRecallRequest defines the payload to trigger a lot recall campaign.
type InitiateRecallRequest struct {
	SupplierID   string `json:"supplier_id"`
	ProductID    string `json:"product_id"`
	LotCode      string `json:"lot_code,omitempty"`
	LotID        string `json:"lot_id,omitempty"`
	RecallReason string `json:"recall_reason"`
	Severity     string `json:"severity,omitempty"` // CRITICAL | WARNING
	InitiatedBy  string `json:"initiated_by"`
}

// RecallImpactedOrderView is a single order affected by recalled lots.
type RecallImpactedOrderView struct {
	CampaignID       string `json:"campaign_id"`
	OrderID          string `json:"order_id"`
	RetailerID       string `json:"retailer_id"`
	WarehouseID      string `json:"warehouse_id"`
	LotID            string `json:"lot_id"`
	SKU              string `json:"sku"`
	Quantity         int64  `json:"quantity"`
	OrderStatus      string `json:"order_status"`
	CustomerNotified bool   `json:"customer_notified"`
	CreatedAt        string `json:"created_at,omitempty"`
}

// RecallCampaignView is the DTO for a recall campaign.
type RecallCampaignView struct {
	CampaignID         string                    `json:"campaign_id"`
	SupplierID         string                    `json:"supplier_id"`
	ProductID          string                    `json:"product_id"`
	LotCode            string                    `json:"lot_code,omitempty"`
	LotID              string                    `json:"lot_id,omitempty"`
	RecallReason       string                    `json:"recall_reason"`
	Severity           string                    `json:"severity"`
	Status             string                    `json:"status"`
	InitiatedBy        string                    `json:"initiated_by"`
	ImpactedLotCount   int64                     `json:"impacted_lot_count"`
	ImpactedUnitsCount int64                     `json:"impacted_units_count"`
	ImpactedOrderCount int64                     `json:"impacted_order_count"`
	CreatedAt          string                    `json:"created_at"`
	UpdatedAt          string                    `json:"updated_at"`
	ImpactedOrders     []RecallImpactedOrderView `json:"impacted_orders,omitempty"`
}

// LotQuarantineEventView represents an audit record for lot status changes.
type LotQuarantineEventView struct {
	EventID     string `json:"event_id"`
	LotID       string `json:"lot_id"`
	WarehouseID string `json:"warehouse_id"`
	SupplierID  string `json:"supplier_id"`
	ProductID   string `json:"product_id"`
	FromStatus  string `json:"from_status"`
	ToStatus    string `json:"to_status"`
	ReasonCode  string `json:"reason_code"`
	Actor       string `json:"actor"`
	Notes       string `json:"notes,omitempty"`
	CreatedAt   string `json:"created_at"`
}

// LotGenealogyView is the full traceability history for a single lot.
type LotGenealogyView struct {
	Lot              StockLotView             `json:"lot"`
	QuarantineEvents []LotQuarantineEventView `json:"quarantine_events"`
	ImpactedOrders   []RecallImpactedOrderView`json:"impacted_orders"`
}

// InitiateRecallInTxn executes a recall campaign, freezing affected lots and tracing orders.
func InitiateRecallInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	buf outbox.TxnBuffer,
	req InitiateRecallRequest,
) (*RecallCampaignView, error) {
	req.SupplierID = strings.TrimSpace(req.SupplierID)
	req.ProductID = strings.TrimSpace(req.ProductID)
	req.RecallReason = strings.TrimSpace(req.RecallReason)
	req.InitiatedBy = strings.TrimSpace(req.InitiatedBy)
	if req.SupplierID == "" || req.ProductID == "" || req.RecallReason == "" {
		return nil, fmt.Errorf("recall: supplier_id, product_id, and recall_reason required")
	}
	if req.InitiatedBy == "" {
		req.InitiatedBy = "system"
	}
	severity := strings.ToUpper(strings.TrimSpace(req.Severity))
	if severity == "" {
		severity = "CRITICAL"
	}

	campaignID := uuid.NewString()
	now := time.Now().UTC()

	// Find matching lots
	sql := `SELECT LotId, WarehouseId, LocationId, LotCode, QuantityOnHand, QuantityReserved, Status, ExpiryDate
	        FROM StockLots
	        WHERE SupplierId = @sid AND ProductId = @pid`
	params := map[string]any{
		"sid": req.SupplierID,
		"pid": req.ProductID,
	}
	if l := strings.TrimSpace(req.LotID); l != "" {
		sql += ` AND LotId = @lid`
		params["lid"] = l
	}
	if lc := strings.TrimSpace(req.LotCode); lc != "" {
		sql += ` AND LotCode = @lc`
		params["lc"] = lc
	}

	iter := txn.Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	type matchedLot struct {
		lotID       string
		warehouseID string
		status      string
		qoh         int64
	}
	var lots []matchedLot
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var lid, wid, loc, status string
		var qoh, qr int64
		var exp spanner.NullDate
		var codeNull spanner.NullString
		if err := row.Columns(&lid, &wid, &loc, &codeNull, &qoh, &qr, &status, &exp); err != nil {
			return nil, err
		}
		lots = append(lots, matchedLot{
			lotID:       lid,
			warehouseID: wid,
			status:      status,
			qoh:         qoh,
		})
	}

	var impactedLotCount, impactedUnitsCount int64
	warehouseRollups := map[string]struct{}{}

	for _, lot := range lots {
		impactedLotCount++
		impactedUnitsCount += lot.qoh

		if lot.status == "AVAILABLE" {
			// Transition to QUARANTINE / RECALLED
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
				"LotId":     lot.lotID,
				"Status":    "QUARANTINE",
				"UpdatedAt": spanner.CommitTimestamp,
			})}); err != nil {
				return nil, err
			}

			// Record Quarantine Audit Event
			qEventID := uuid.NewString()
			if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("LotQuarantineEvents", map[string]any{
				"EventId":     qEventID,
				"LotId":       lot.lotID,
				"WarehouseId": lot.warehouseID,
				"SupplierId":  req.SupplierID,
				"ProductId":   req.ProductID,
				"FromStatus":  lot.status,
				"ToStatus":    "QUARANTINE",
				"ReasonCode":  "RECALL_CAMPAIGN",
				"Actor":       req.InitiatedBy,
				"Notes":       fmt.Sprintf("Recall campaign %s: %s", campaignID, req.RecallReason),
				"CreatedAt":   spanner.CommitTimestamp,
			})}); err != nil {
				return nil, err
			}

			warehouseRollups[lot.warehouseID] = struct{}{}
		}
	}

	// Re-rollup inventory for touched warehouses so ATP is updated in the same txn
	for wid := range warehouseRollups {
		if err := RollupInventoryV2InTxn(ctx, txn, req.SupplierID, wid, req.ProductID); err != nil {
			return nil, err
		}
	}

	// Trace orders that reserved or picked these lots
	var impactedOrders []RecallImpactedOrderView
	orderSeen := map[string]struct{}{}

	for _, lot := range lots {
		// Trace reservations
		resIter := txn.Query(ctx, spanner.Statement{
			SQL:    `SELECT OrderId, Quantity FROM OrderLotReservations WHERE LotId = @lid`,
			Params: map[string]any{"lid": lot.lotID},
		})
		for {
			rRow, err := resIter.Next()
			if err == iterator.Done {
				break
			}
			if err != nil {
				resIter.Stop()
				return nil, err
			}
			var oid string
			var qty int64
			if err := rRow.Columns(&oid, &qty); err != nil {
				resIter.Stop()
				return nil, err
			}
			if _, exists := orderSeen[oid]; !exists {
				orderSeen[oid] = struct{}{}
				retID, oStatus := resolveOrderDetails(ctx, txn, oid)
				imp := RecallImpactedOrderView{
					CampaignID:       campaignID,
					OrderID:          oid,
					RetailerID:       retID,
					WarehouseID:      lot.warehouseID,
					LotID:            lot.lotID,
					SKU:              req.ProductID,
					Quantity:         qty,
					OrderStatus:      oStatus,
					CustomerNotified: false,
					CreatedAt:        now.Format(time.RFC3339),
				}
				impactedOrders = append(impactedOrders, imp)

				if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("LotRecallImpactedOrders", map[string]any{
					"CampaignId":       campaignID,
					"OrderId":          oid,
					"RetailerId":       retID,
					"WarehouseId":      lot.warehouseID,
					"LotId":            lot.lotID,
					"Sku":              req.ProductID,
					"Quantity":         qty,
					"OrderStatus":      oStatus,
					"CustomerNotified": false,
					"CreatedAt":        spanner.CommitTimestamp,
				})}); err != nil {
					resIter.Stop()
					return nil, err
				}
			}
		}
		resIter.Stop()
	}

	impactedOrderCount := int64(len(impactedOrders))

	// Insert Campaign Header
	campCols := map[string]any{
		"CampaignId":         campaignID,
		"SupplierId":         req.SupplierID,
		"ProductId":          req.ProductID,
		"RecallReason":       req.RecallReason,
		"Severity":           severity,
		"Status":             "INITIATED",
		"InitiatedBy":        req.InitiatedBy,
		"ImpactedLotCount":   impactedLotCount,
		"ImpactedUnitsCount": impactedUnitsCount,
		"ImpactedOrderCount": impactedOrderCount,
		"CreatedAt":          spanner.CommitTimestamp,
		"UpdatedAt":          spanner.CommitTimestamp,
	}
	if req.LotCode != "" {
		campCols["LotCode"] = req.LotCode
	}
	if req.LotID != "" {
		campCols["LotId"] = req.LotID
	}
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("LotRecallCampaigns", campCols)}); err != nil {
		return nil, err
	}

	// Outbox event emission
	if buf != nil {
		_ = outbox.EmitJSON(ctx, buf, events.AggregateLotRecall, campaignID, events.TopicExceptions, events.LotRecallEvent{
			BaseEvent: events.BaseEvent{
				Type: events.EventLotRecallInitiated,
			},
			CampaignID:         campaignID,
			SupplierID:         req.SupplierID,
			ProductID:          req.ProductID,
			LotCode:            req.LotCode,
			LotID:              req.LotID,
			RecallReason:       req.RecallReason,
			Severity:           severity,
			Status:             "INITIATED",
			ImpactedLotCount:   impactedLotCount,
			ImpactedUnitsCount: impactedUnitsCount,
			ImpactedOrderCount: impactedOrderCount,
			Action:             "INITIATE_RECALL",
			Actor:              req.InitiatedBy,
		})
	}

	return &RecallCampaignView{
		CampaignID:         campaignID,
		SupplierID:         req.SupplierID,
		ProductID:          req.ProductID,
		LotCode:            req.LotCode,
		LotID:              req.LotID,
		RecallReason:       req.RecallReason,
		Severity:           severity,
		Status:             "INITIATED",
		InitiatedBy:        req.InitiatedBy,
		ImpactedLotCount:   impactedLotCount,
		ImpactedUnitsCount: impactedUnitsCount,
		ImpactedOrderCount: impactedOrderCount,
		CreatedAt:          now.Format(time.RFC3339),
		UpdatedAt:          now.Format(time.RFC3339),
		ImpactedOrders:     impactedOrders,
	}, nil
}

func resolveOrderDetails(ctx context.Context, txn *spanner.ReadWriteTransaction, orderID string) (retailerID, status string) {
	row, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID}, []string{"RetailerId", "Status"})
	if err != nil {
		return "unknown", "UNKNOWN"
	}
	_ = row.Columns(&retailerID, &status)
	return retailerID, status
}

// QuarantineLotInTxn explicitly freezes a lot into QUARANTINE status.
func QuarantineLotInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	buf outbox.TxnBuffer,
	lotID, warehouseID, reasonCode, actor, notes string,
) (*StockLotView, error) {
	lotID = strings.TrimSpace(lotID)
	if lotID == "" {
		return nil, fmt.Errorf("lot_id required")
	}
	if reasonCode == "" {
		reasonCode = "QUALITY_HOLD"
	}

	lotRow, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lotID},
		[]string{"LotId", "SupplierId", "WarehouseId", "ProductId", "LocationId", "LotCode",
			"ExpiryDate", "ManufacturedDate", "QuantityOnHand", "QuantityReserved", "Status", "ReceivedAt"})
	if err != nil {
		return nil, err
	}

	v, err := scanLot(lotRow)
	if err != nil {
		return nil, err
	}

	if warehouseID != "" && v.WarehouseID != warehouseID {
		return nil, fmt.Errorf("warehouse_scope_mismatch")
	}

	oldStatus := v.Status
	v.Status = "QUARANTINE"

	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
		"LotId":     lotID,
		"Status":    "QUARANTINE",
		"UpdatedAt": spanner.CommitTimestamp,
	})}); err != nil {
		return nil, err
	}

	qEventID := uuid.NewString()
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("LotQuarantineEvents", map[string]any{
		"EventId":     qEventID,
		"LotId":       lotID,
		"WarehouseId": v.WarehouseID,
		"SupplierId":  v.SupplierID,
		"ProductId":   v.ProductID,
		"FromStatus":  oldStatus,
		"ToStatus":    "QUARANTINE",
		"ReasonCode":  reasonCode,
		"Actor":       actor,
		"Notes":       notes,
		"CreatedAt":   spanner.CommitTimestamp,
	})}); err != nil {
		return nil, err
	}

	if oldStatus == "AVAILABLE" {
		if err := RollupInventoryV2InTxn(ctx, txn, v.SupplierID, v.WarehouseID, v.ProductID); err != nil {
			return nil, err
		}
	}

	if buf != nil {
		_ = outbox.EmitJSON(ctx, buf, events.AggregateLotRecall, lotID, events.TopicExceptions, events.LotRecallEvent{
			BaseEvent: events.BaseEvent{
				Type: events.EventLotQuarantined,
			},
			LotID:        lotID,
			SupplierID:   v.SupplierID,
			ProductID:    v.ProductID,
			WarehouseID:  v.WarehouseID,
			RecallReason: reasonCode,
			Status:       "QUARANTINE",
			Action:       "QUARANTINE_LOT",
			Actor:        actor,
		})
	}

	return &v, nil
}

// ReleaseLotInTxn releases a lot from QUARANTINE back to AVAILABLE if valid.
func ReleaseLotInTxn(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	buf outbox.TxnBuffer,
	lotID, warehouseID, reasonCode, actor, notes string,
) (*StockLotView, error) {
	lotID = strings.TrimSpace(lotID)
	if lotID == "" {
		return nil, fmt.Errorf("lot_id required")
	}
	if reasonCode == "" {
		reasonCode = "QUALITY_RELEASE"
	}

	lotRow, err := txn.ReadRow(ctx, "StockLots", spanner.Key{lotID},
		[]string{"LotId", "SupplierId", "WarehouseId", "ProductId", "LocationId", "LotCode",
			"ExpiryDate", "ManufacturedDate", "QuantityOnHand", "QuantityReserved", "Status", "ReceivedAt"})
	if err != nil {
		return nil, err
	}

	v, err := scanLot(lotRow)
	if err != nil {
		return nil, err
	}

	if warehouseID != "" && v.WarehouseID != warehouseID {
		return nil, fmt.Errorf("warehouse_scope_mismatch")
	}

	oldStatus := v.Status
	newStatus := "AVAILABLE"
	if v.QuantityOnHand <= v.QuantityReserved && v.QuantityOnHand > 0 {
		newStatus = "ALLOCATED"
	} else if v.QuantityOnHand == 0 {
		newStatus = "DEPLETED"
	}

	v.Status = newStatus

	if err := txn.BufferWrite([]*spanner.Mutation{spanner.UpdateMap("StockLots", map[string]any{
		"LotId":     lotID,
		"Status":    newStatus,
		"UpdatedAt": spanner.CommitTimestamp,
	})}); err != nil {
		return nil, err
	}

	qEventID := uuid.NewString()
	if err := txn.BufferWrite([]*spanner.Mutation{spanner.InsertMap("LotQuarantineEvents", map[string]any{
		"EventId":     qEventID,
		"LotId":       lotID,
		"WarehouseId": v.WarehouseID,
		"SupplierId":  v.SupplierID,
		"ProductId":   v.ProductID,
		"FromStatus":  oldStatus,
		"ToStatus":    newStatus,
		"ReasonCode":  reasonCode,
		"Actor":       actor,
		"Notes":       notes,
		"CreatedAt":   spanner.CommitTimestamp,
	})}); err != nil {
		return nil, err
	}

	if newStatus == "AVAILABLE" || oldStatus == "QUARANTINE" {
		if err := RollupInventoryV2InTxn(ctx, txn, v.SupplierID, v.WarehouseID, v.ProductID); err != nil {
			return nil, err
		}
	}

	if buf != nil {
		_ = outbox.EmitJSON(ctx, buf, events.AggregateLotRecall, lotID, events.TopicExceptions, events.LotRecallEvent{
			BaseEvent: events.BaseEvent{
				Type: events.EventLotReleased,
			},
			LotID:        lotID,
			SupplierID:   v.SupplierID,
			ProductID:    v.ProductID,
			WarehouseID:  v.WarehouseID,
			RecallReason: reasonCode,
			Status:       newStatus,
			Action:       "RELEASE_LOT",
			Actor:        actor,
		})
	}

	return &v, nil
}

// ListRecallCampaigns retrieves recall campaigns for a supplier or system admin.
func ListRecallCampaigns(ctx context.Context, client *spanner.Client, supplierID, status string, limit int) ([]RecallCampaignView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	sql := `SELECT CampaignId, SupplierId, ProductId, LotCode, LotId, RecallReason,
	               Severity, Status, InitiatedBy, ImpactedLotCount, ImpactedUnitsCount,
	               ImpactedOrderCount, CreatedAt, UpdatedAt
	        FROM LotRecallCampaigns WHERE 1=1`
	params := map[string]any{}
	if sid := strings.TrimSpace(supplierID); sid != "" {
		sql += ` AND SupplierId = @sid`
		params["sid"] = sid
	}
	if st := strings.TrimSpace(status); st != "" {
		sql += ` AND Status = @st`
		params["st"] = strings.ToUpper(st)
	}
	sql += ` ORDER BY CreatedAt DESC LIMIT @lim`
	params["lim"] = int64(limit)

	iter := client.Single().Query(ctx, spanner.Statement{SQL: sql, Params: params})
	defer iter.Stop()

	var out []RecallCampaignView
	for {
		row, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var c RecallCampaignView
		var lc, lid spanner.NullString
		var created, updated time.Time
		if err := row.Columns(
			&c.CampaignID, &c.SupplierID, &c.ProductID, &lc, &lid, &c.RecallReason,
			&c.Severity, &c.Status, &c.InitiatedBy, &c.ImpactedLotCount, &c.ImpactedUnitsCount,
			&c.ImpactedOrderCount, &created, &updated,
		); err != nil {
			return nil, err
		}
		if lc.Valid {
			c.LotCode = lc.StringVal
		}
		if lid.Valid {
			c.LotID = lid.StringVal
		}
		c.CreatedAt = created.UTC().Format(time.RFC3339)
		c.UpdatedAt = updated.UTC().Format(time.RFC3339)
		out = append(out, c)
	}
	return out, nil
}

// GetRecallCampaign loads campaign header and all impacted orders.
func GetRecallCampaign(ctx context.Context, client *spanner.Client, campaignID string) (*RecallCampaignView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	row, err := client.Single().ReadRow(ctx, "LotRecallCampaigns", spanner.Key{campaignID},
		[]string{"CampaignId", "SupplierId", "ProductId", "LotCode", "LotId", "RecallReason",
			"Severity", "Status", "InitiatedBy", "ImpactedLotCount", "ImpactedUnitsCount",
			"ImpactedOrderCount", "CreatedAt", "UpdatedAt"})
	if err != nil {
		return nil, err
	}

	var c RecallCampaignView
	var lc, lid spanner.NullString
	var created, updated time.Time
	if err := row.Columns(
		&c.CampaignID, &c.SupplierID, &c.ProductID, &lc, &lid, &c.RecallReason,
		&c.Severity, &c.Status, &c.InitiatedBy, &c.ImpactedLotCount, &c.ImpactedUnitsCount,
		&c.ImpactedOrderCount, &created, &updated,
	); err != nil {
		return nil, err
	}
	if lc.Valid {
		c.LotCode = lc.StringVal
	}
	if lid.Valid {
		c.LotID = lid.StringVal
	}
	c.CreatedAt = created.UTC().Format(time.RFC3339)
	c.UpdatedAt = updated.UTC().Format(time.RFC3339)

	// Impacted orders query
	oIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT CampaignId, OrderId, RetailerId, WarehouseId, LotId, Sku, Quantity, OrderStatus, CustomerNotified, CreatedAt
		      FROM LotRecallImpactedOrders WHERE CampaignId = @cid ORDER BY CreatedAt DESC`,
		Params: map[string]any{"cid": campaignID},
	})
	defer oIter.Stop()

	for {
		oRow, err := oIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var imp RecallImpactedOrderView
		var oCreated time.Time
		if err := oRow.Columns(
			&imp.CampaignID, &imp.OrderID, &imp.RetailerID, &imp.WarehouseID, &imp.LotID,
			&imp.SKU, &imp.Quantity, &imp.OrderStatus, &imp.CustomerNotified, &oCreated,
		); err != nil {
			return nil, err
		}
		imp.CreatedAt = oCreated.UTC().Format(time.RFC3339)
		c.ImpactedOrders = append(c.ImpactedOrders, imp)
	}

	return &c, nil
}

// TraceLotGenealogy pulls end-to-end audit trail and distribution history for a lot.
func TraceLotGenealogy(ctx context.Context, client *spanner.Client, lotID string) (*LotGenealogyView, error) {
	if client == nil {
		return nil, fmt.Errorf("spanner required")
	}
	lot, err := GetLot(ctx, client, lotID)
	if err != nil {
		return nil, err
	}

	// Quarantine events
	qIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT EventId, LotId, WarehouseId, SupplierId, ProductId, FromStatus, ToStatus, ReasonCode, Actor, Notes, CreatedAt
		      FROM LotQuarantineEvents WHERE LotId = @lid ORDER BY CreatedAt DESC`,
		Params: map[string]any{"lid": lotID},
	})
	defer qIter.Stop()

	var qEvents []LotQuarantineEventView
	for {
		qRow, err := qIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var q LotQuarantineEventView
		var notes spanner.NullString
		var created time.Time
		if err := qRow.Columns(
			&q.EventID, &q.LotID, &q.WarehouseID, &q.SupplierID, &q.ProductID,
			&q.FromStatus, &q.ToStatus, &q.ReasonCode, &q.Actor, &notes, &created,
		); err != nil {
			return nil, err
		}
		if notes.Valid {
			q.Notes = notes.StringVal
		}
		q.CreatedAt = created.UTC().Format(time.RFC3339)
		qEvents = append(qEvents, q)
	}

	// Impacted/recalled orders
	oIter := client.Single().Query(ctx, spanner.Statement{
		SQL: `SELECT CampaignId, OrderId, RetailerId, WarehouseId, LotId, Sku, Quantity, OrderStatus, CustomerNotified, CreatedAt
		      FROM LotRecallImpactedOrders WHERE LotId = @lid ORDER BY CreatedAt DESC`,
		Params: map[string]any{"lid": lotID},
	})
	defer oIter.Stop()

	var impOrders []RecallImpactedOrderView
	for {
		oRow, err := oIter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}
		var imp RecallImpactedOrderView
		var created time.Time
		if err := oRow.Columns(
			&imp.CampaignID, &imp.OrderID, &imp.RetailerID, &imp.WarehouseID, &imp.LotID,
			&imp.SKU, &imp.Quantity, &imp.OrderStatus, &imp.CustomerNotified, &created,
		); err != nil {
			return nil, err
		}
		imp.CreatedAt = created.UTC().Format(time.RFC3339)
		impOrders = append(impOrders, imp)
	}

	return &LotGenealogyView{
		Lot:              *lot,
		QuarantineEvents: qEvents,
		ImpactedOrders:   impOrders,
	}, nil
}

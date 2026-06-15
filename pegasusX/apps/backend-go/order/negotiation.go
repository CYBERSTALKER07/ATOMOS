package order

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/pegasusx/pegasusx/apps/backend-go/auth"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
)

// ProposedNegotiationItem is one line in a driver quantity negotiation.
type ProposedNegotiationItem struct {
	SKUID       string `json:"sku_id"`
	OriginalQty int64  `json:"original_qty"`
	ProposedQty int64  `json:"proposed_qty"`
}

// HandleProposeNegotiation is POST /v1/delivery/negotiate (DRIVER).
func (s *Service) HandleProposeNegotiation(w http.ResponseWriter, r *http.Request) {
	if quantityNegotiationDisabled {
		writeNegotiationDisabled(w)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleDriver {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "negotiation_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		OrderID string                    `json:"order_id"`
		Items   []ProposedNegotiationItem `json:"items"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.OrderID = strings.TrimSpace(req.OrderID)
	if req.OrderID == "" || len(req.Items) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "order_id and items required"})
		return
	}

	driverID := strings.TrimSpace(claims.Subject)
	proposalID := s.newID()
	ctx := r.Context()
	now := s.now()

	var supplierID, retailerID string

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
			[]string{"Status", "DriverId", "SupplierId", "RetailerId"})
		if err != nil {
			return fmt.Errorf("order not found: %w", err)
		}
		var status string
		var driverCol, supplierCol, retailerCol spanner.NullString
		if err := row.Columns(&status, &driverCol, &supplierCol, &retailerCol); err != nil {
			return err
		}
		if status != string(StatusInTransit) && status != string(StatusArrived) {
			return fmt.Errorf("order must be IN_TRANSIT or ARRIVED (current: %s)", status)
		}
		if !driverCol.Valid || strings.TrimSpace(driverCol.StringVal) != driverID {
			return ErrOrderForbidden
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		if retailerCol.Valid {
			retailerID = retailerCol.StringVal
		}

		pendingStmt := spanner.Statement{
			SQL:    `SELECT ProposalId FROM NegotiationProposals WHERE OrderId = @oid AND Status = 'PENDING' LIMIT 1`,
			Params: map[string]interface{}{"oid": req.OrderID},
		}
		iter := txn.Query(ctx, pendingStmt)
		pendingRow, pendingErr := iter.Next()
		iter.Stop()
		if pendingErr == nil && pendingRow != nil {
			return fmt.Errorf("pending negotiation already exists for this order")
		}

		itemsJSON, err := json.Marshal(req.Items)
		if err != nil {
			return err
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, req.OrderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventNegotiationProposed, Timestamp: now.Format(time.RFC3339Nano)},
			ProposalID: proposalID,
			OrderID:    req.OrderID,
			DriverID:   driverID,
			SupplierID: supplierID,
			RetailerID: retailerID,
		}); err != nil {
			return err
		}

		mutations := []*spanner.Mutation{
			spanner.InsertMap("NegotiationProposals", map[string]any{
				"ProposalId":    proposalID,
				"OrderId":       req.OrderID,
				"DriverId":      driverID,
				"Status":        "PENDING",
				"ProposedItems": string(itemsJSON),
				"CreatedAt":     now.UTC(),
			}),
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "order_forbidden"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.broadcastNegotiation(ctx, supplierID, retailerID, driverID, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventNegotiationProposed, Timestamp: now.Format(time.RFC3339Nano)},
		ProposalID: proposalID,
		OrderID:    req.OrderID,
		DriverID:   driverID,
		SupplierID: supplierID,
		RetailerID: retailerID,
	})

	idemCommitted = true
	s.writeIdempotentJSON(w, r, body, http.StatusOK, map[string]string{
		"status":      "PENDING",
		"proposal_id": proposalID,
	})
}

// HandleResolveNegotiation is POST /v1/supplier/negotiate/resolve (SUPPLIER / ADMIN JWT).
func (s *Service) HandleResolveNegotiation(w http.ResponseWriter, r *http.Request) {
	if quantityNegotiationDisabled {
		writeNegotiationDisabled(w)
		return
	}
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method_not_allowed"})
		return
	}
	claims, ok := auth.FromContext(r.Context())
	if !ok || claims.Role != auth.RoleAdmin {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
		return
	}
	if s.spannerClient == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "negotiation_unavailable"})
		return
	}

	body, err := readLimitedBody(r, 64*1024)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "read_body_error"})
		return
	}
	if s.guardIdempotency(w, r, body) {
		return
	}
	idemCommitted := false
	defer func() {
		if !idemCommitted {
			s.releaseIdempotency(r.Context(), r)
		}
	}()

	var req struct {
		ProposalID string `json:"proposal_id"`
		Action     string `json:"action"`
		Resolution string `json:"resolution"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_json"})
		return
	}
	req.ProposalID = strings.TrimSpace(req.ProposalID)
	req.Action = strings.TrimSpace(strings.ToUpper(req.Action))
	req.Resolution = strings.TrimSpace(req.Resolution)
	if req.ProposalID == "" || (req.Action != "APPROVE" && req.Action != "REJECT") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "proposal_id and action (APPROVE|REJECT) required"})
		return
	}

	ctx := r.Context()
	now := s.now()
	resolverID := strings.TrimSpace(claims.Subject)

	var orderID, driverID, supplierID, retailerID string
	var proposedItems []ProposedNegotiationItem

	_, err = s.spannerClient.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(ctx, "NegotiationProposals", spanner.Key{req.ProposalID},
			[]string{"OrderId", "DriverId", "Status", "ProposedItems"})
		if err != nil {
			return fmt.Errorf("proposal not found: %w", err)
		}
		var status, itemsJSON string
		if err := row.Columns(&orderID, &driverID, &status, &itemsJSON); err != nil {
			return err
		}
		if status != "PENDING" {
			return fmt.Errorf("proposal is already %s", status)
		}
		if err := json.Unmarshal([]byte(itemsJSON), &proposedItems); err != nil {
			return fmt.Errorf("parse proposed items: %w", err)
		}

		orderRow, err := txn.ReadRow(ctx, "Orders", spanner.Key{orderID},
			[]string{"SupplierId", "RetailerId", "Version", "LineItemsJson", "Currency"})
		if err != nil {
			return err
		}
		var version int64
		var currency string
		var lineItemsRaw []byte
		var supplierCol, retailerCol spanner.NullString
		if err := orderRow.Columns(&supplierCol, &retailerCol, &version, &lineItemsRaw, &currency); err != nil {
			return err
		}
		if supplierCol.Valid {
			supplierID = supplierCol.StringVal
		}
		if retailerCol.Valid {
			retailerID = retailerCol.StringVal
		}
		if strings.TrimSpace(claims.SupplierID) != "" && supplierID != "" && claims.SupplierID != supplierID {
			return ErrOrderForbidden
		}

		newStatus := "APPROVED"
		if req.Action == "REJECT" {
			newStatus = "REJECTED"
		}

		updateProposal := map[string]any{
			"ProposalId": req.ProposalID,
			"Status":     newStatus,
			"ResolvedBy": resolverID,
			"ResolvedAt": now.UTC(),
		}
		if req.Resolution != "" {
			updateProposal["Resolution"] = req.Resolution
		}

		mutations := []*spanner.Mutation{
			spanner.UpdateMap("NegotiationProposals", updateProposal),
		}

		if req.Action == "APPROVE" && len(proposedItems) > 0 {
			var lineItems []LineItem
			if err := json.Unmarshal(lineItemsRaw, &lineItems); err != nil {
				return fmt.Errorf("parse order line items: %w", err)
			}
			updated, total, err := applyNegotiatedLineItems(lineItems, proposedItems)
			if err != nil {
				return err
			}
			updatedRaw, err := json.Marshal(updated)
			if err != nil {
				return err
			}
			mutations = append(mutations, spanner.UpdateMap("Orders", map[string]any{
				"OrderId":       orderID,
				"LineItemsJson": updatedRaw,
				"TotalMinor":    total,
				"Version":       version + 1,
				"UpdatedAt":     now.UTC(),
			}))
		}

		buf := &spannerTxnBuffer{}
		if err := outbox.EmitJSON(ctx, buf, events.AggregateOrder, orderID, events.TopicMain, events.OrderEvent{
			BaseEvent:  events.BaseEvent{Type: events.EventNegotiationResolved, Timestamp: now.Format(time.RFC3339Nano)},
			ProposalID: req.ProposalID,
			OrderID:    orderID,
			SupplierID: supplierID,
			RetailerID: retailerID,
			DriverID:   driverID,
			Action:     req.Action,
			Resolution: req.Resolution,
		}); err != nil {
			return err
		}
		for _, e := range buf.events {
			mutations = append(mutations, outboxMutation(e))
		}
		return txn.BufferWrite(mutations)
	})
	if err != nil {
		if errors.Is(err, ErrOrderForbidden) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "order_forbidden"})
			return
		}
		writeJSON(w, http.StatusConflict, map[string]string{"error": err.Error()})
		return
	}

	s.broadcastNegotiation(ctx, supplierID, retailerID, driverID, events.OrderEvent{
		BaseEvent:  events.BaseEvent{Type: events.EventNegotiationResolved, Timestamp: now.Format(time.RFC3339Nano)},
		ProposalID: req.ProposalID,
		OrderID:    orderID,
		DriverID:   driverID,
		SupplierID: supplierID,
		RetailerID: retailerID,
		Action:     req.Action,
		Resolution: req.Resolution,
	})
	s.invalidateOrderCache(ctx, orderID)

	resp := map[string]any{
		"status":      req.Action,
		"proposal_id": req.ProposalID,
		"order_id":    orderID,
	}
	respBytes, _ := json.Marshal(resp)
	s.saveIdempotency(ctx, r, body, http.StatusOK, respBytes)
	idemCommitted = true
	writeJSONBytes(w, http.StatusOK, respBytes)
}

func applyNegotiatedLineItems(existing []LineItem, proposed []ProposedNegotiationItem) ([]LineItem, int64, error) {
	bySKU := make(map[string]int64, len(proposed))
	for _, p := range proposed {
		sku := strings.TrimSpace(p.SKUID)
		if sku == "" {
			continue
		}
		if p.ProposedQty < 0 {
			return nil, 0, fmt.Errorf("proposed_qty must be >= 0 for sku %s", sku)
		}
		bySKU[sku] = p.ProposedQty
	}
	if len(bySKU) == 0 {
		return nil, 0, errors.New("no valid proposed items")
	}

	updated := make([]LineItem, len(existing))
	copy(updated, existing)
	var total int64
	matched := 0
	for i, li := range updated {
		if qty, ok := bySKU[strings.TrimSpace(li.SKU)]; ok {
			updated[i].Quantity = qty
			matched++
		}
		if updated[i].Quantity < 0 {
			return nil, 0, fmt.Errorf("invalid quantity for sku %s", li.SKU)
		}
		total += updated[i].Quantity * updated[i].UnitPrice
	}
	if matched == 0 {
		return nil, 0, errors.New("proposed skus did not match order line items")
	}
	return updated, total, nil
}

func (s *Service) broadcastNegotiation(ctx context.Context, supplierID, retailerID, driverID string, payload events.OrderEvent) {
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}
	if s.supplierHub != nil && supplierID != "" {
		s.supplierHub.Broadcast(ctx, "supplier:"+supplierID, raw)
	}
	if s.retailerHub != nil && retailerID != "" {
		s.retailerHub.Broadcast(ctx, "retailer:"+retailerID, raw)
	}
	if s.driverHub != nil && driverID != "" {
		s.driverHub.Broadcast(ctx, "driver:"+driverID, raw)
	}
}

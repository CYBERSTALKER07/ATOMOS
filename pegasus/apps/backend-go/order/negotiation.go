package order

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"backend-go/auth"
	"backend-go/hotspot"
	kafkaEvents "backend-go/kafka"
	"backend-go/outbox"
	"backend-go/telemetry"
	"backend-go/ws"

	"cloud.google.com/go/spanner"
)

// ═══════════════════════════════════════════════════════════════════════════════
// Edge 28: Live Negotiation Mode
// Driver proposes quantity changes → supplier approves/rejects in real-time.
// On approval, AmendOrder executes the quantity change internally.
// ═══════════════════════════════════════════════════════════════════════════════

type NegotiationDeps struct {
	SupplierPush func(supplierID string, payload interface{}) bool
	DriverPush   func(driverID string, payload interface{}) bool
	NotifyUser   func(ctx context.Context, userID, role string, title, body string, data map[string]string)
}

type ProposedItem struct {
	SkuID       string `json:"sku_id"`
	OriginalQty int64  `json:"original_qty"`
	ProposedQty int64  `json:"proposed_qty"`
}

// HandleProposeNegotiation lets a driver propose quantity changes for an order.
// POST /v1/delivery/negotiate (DRIVER role)
func HandleProposeNegotiation(svc *OrderService, deps *NegotiationDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			OrderID string         `json:"order_id"`
			Items   []ProposedItem `json:"items"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.OrderID == "" || len(req.Items) == 0 {
			http.Error(w, `{"error":"order_id and items required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
		defer cancel()

		proposalID := hotspot.NewOpaqueID()
		var supplierID, retailerID string

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Verify order state
			row, err := txn.ReadRow(ctx, "Orders", spanner.Key{req.OrderID},
				[]string{"State", "DriverId", "SupplierId", "RetailerId"})
			if err != nil {
				return fmt.Errorf("order not found: %w", err)
			}
			var state string
			var did, sid, rid spanner.NullString
			if err := row.Columns(&state, &did, &sid, &rid); err != nil {
				return err
			}
			if state != "IN_TRANSIT" && state != "ARRIVED" {
				return fmt.Errorf("order must be IN_TRANSIT or ARRIVED for negotiation (current: %s)", state)
			}
			if !did.Valid || did.StringVal != claims.UserID {
				return fmt.Errorf("driver mismatch")
			}
			if sid.Valid {
				supplierID = sid.StringVal
			}
			if rid.Valid {
				retailerID = rid.StringVal
			}

			// Check no pending proposal exists for this order
			pendingStmt := spanner.Statement{
				SQL:    `SELECT ProposalId FROM NegotiationProposals WHERE OrderId = @oid AND Status = 'PENDING' LIMIT 1`,
				Params: map[string]interface{}{"oid": req.OrderID},
			}
			iter := txn.Query(ctx, pendingStmt)
			pendingRow, pendingErr := iter.Next()
			iter.Stop()
			if pendingErr == nil && pendingRow != nil {
				return fmt.Errorf("a pending negotiation already exists for this order")
			}

			// Insert proposal
			itemsJSON, _ := json.Marshal(req.Items)
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Insert("NegotiationProposals",
					[]string{"ProposalId", "OrderId", "DriverId", "Status", "ProposedItems", "CreatedAt"},
					[]interface{}{proposalID, req.OrderID, claims.UserID, "PENDING", string(itemsJSON), spanner.CommitTimestamp}),
			}); err != nil {
				return fmt.Errorf("insert negotiation proposal: %w", err)
			}

			now := time.Now().UTC()
			if err := outbox.EmitJSON(txn, "NegotiationProposal", proposalID, kafkaEvents.EventNegotiationProposed, topicLogisticsEvents, kafkaEvents.NegotiationProposedEvent{
				ProposalID: proposalID,
				OrderID:    req.OrderID,
				DriverID:   claims.UserID,
				SupplierID: supplierID,
				RetailerID: retailerID,
				Timestamp:  now,
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
				return fmt.Errorf("outbox emit negotiation proposed: %w", err)
			}

			return nil
		})

		if err != nil {
			log.Printf("[NEGOTIATE] Proposal failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// Post-commit: push to supplier
		if deps != nil && deps.SupplierPush != nil {
			deps.SupplierPush(supplierID, map[string]interface{}{
				"type":        ws.EventNegotiationProposed,
				"proposal_id": proposalID,
				"order_id":    req.OrderID,
				"driver_id":   claims.UserID,
				"items":       req.Items,
			})
		}
		if deps != nil && deps.NotifyUser != nil {
			go deps.NotifyUser(context.Background(), supplierID, "SUPPLIER",
				"Negotiation Request",
				fmt.Sprintf("Driver proposes quantity changes on Order %s", req.OrderID[:8]),
				map[string]string{"type": ws.EventNegotiationProposed, "order_id": req.OrderID, "proposal_id": proposalID})
		}

		writeOrderEvent(ctx, svc.Client, req.OrderID, claims.UserID, "DRIVER", "NEGOTIATION_PROPOSED",
			map[string]string{"proposal_id": proposalID}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":      "PENDING",
			"proposal_id": proposalID,
		})
	}
}

// HandleResolveNegotiation lets a supplier approve or reject a negotiation proposal.
// POST /v1/admin/negotiate/resolve (ADMIN/SUPPLIER role)
func HandleResolveNegotiation(svc *OrderService, deps *NegotiationDeps) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}
		claims, ok := r.Context().Value(auth.ClaimsContextKey).(*auth.LabClaims)
		if !ok || claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}

		var req struct {
			ProposalID string `json:"proposal_id"`
			Action     string `json:"action"`     // APPROVE | REJECT
			Resolution string `json:"resolution"` // Optional comment
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ProposalID == "" || (req.Action != "APPROVE" && req.Action != "REJECT") {
			http.Error(w, `{"error":"proposal_id and action (APPROVE|REJECT) required"}`, http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
		defer cancel()

		var orderID, driverID string
		var proposedItems []ProposedItem

		_, err := svc.Client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
			// Read proposal
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
				return fmt.Errorf("failed to parse proposed items: %w", err)
			}

			now := time.Now().UTC()
			newStatus := "APPROVED"
			if req.Action == "REJECT" {
				newStatus = "REJECTED"
			}

			// Update proposal
			cols := []string{"ProposalId", "Status", "ResolvedBy", "ResolvedAt"}
			vals := []interface{}{req.ProposalID, newStatus, claims.UserID, now}
			if req.Resolution != "" {
				cols = append(cols, "Resolution")
				vals = append(vals, req.Resolution)
			}
			if err := txn.BufferWrite([]*spanner.Mutation{
				spanner.Update("NegotiationProposals", cols, vals),
			}); err != nil {
				return fmt.Errorf("update negotiation proposal: %w", err)
			}

			if err := outbox.EmitJSON(txn, "NegotiationProposal", req.ProposalID, kafkaEvents.EventNegotiationResolved, topicLogisticsEvents, kafkaEvents.NegotiationResolvedEvent{
				ProposalID: req.ProposalID,
				OrderID:    orderID,
				SupplierID: claims.ResolveSupplierID(),
				Action:     req.Action,
				Timestamp:  now,
			}, telemetry.TraceIDFromContext(ctx)); err != nil {
				return fmt.Errorf("outbox emit negotiation resolved: %w", err)
			}

			return nil
		})

		if err != nil {
			log.Printf("[NEGOTIATE] Resolve failed: %v", err)
			http.Error(w, fmt.Sprintf(`{"error":"%s"}`, err.Error()), http.StatusConflict)
			return
		}

		// If approved, execute the amendment
		if req.Action == "APPROVE" && len(proposedItems) > 0 {
			amendItems := make([]AmendItemReq, len(proposedItems))
			for i, item := range proposedItems {
				rejected := item.OriginalQty - item.ProposedQty
				if rejected < 0 {
					rejected = 0
				}
				amendItems[i] = AmendItemReq{
					ProductId:   item.SkuID,
					AcceptedQty: item.ProposedQty,
					RejectedQty: rejected,
					Reason:      "NEGOTIATED",
				}
			}
			amendReq := AmendOrderRequest{
				OrderID: orderID,
				Items:   amendItems,
			}
			if _, amendErr := svc.AmendOrder(ctx, amendReq); amendErr != nil {
				log.Printf("[NEGOTIATE] AmendOrder failed after approval for order %s: %v", orderID, amendErr)
			}
		}

		// Push to driver
		if deps != nil && deps.DriverPush != nil {
			deps.DriverPush(driverID, map[string]interface{}{
				"type":        ws.EventNegotiationResolved,
				"proposal_id": req.ProposalID,
				"order_id":    orderID,
				"action":      req.Action,
				"resolution":  req.Resolution,
			})
		}

		writeOrderEvent(ctx, svc.Client, orderID, claims.UserID, claims.Role, "NEGOTIATION_RESOLVED",
			map[string]string{"proposal_id": req.ProposalID, "action": req.Action}, 0, 0)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":      req.Action,
			"proposal_id": req.ProposalID,
			"order_id":    orderID,
		})
	}
}

package proposals

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"cloud.google.com/go/spanner"
	"github.com/google/uuid"
	"github.com/pegasusx/pegasusx/apps/backend-go/events"
	"github.com/pegasusx/pegasusx/apps/backend-go/notifications"
	"github.com/pegasusx/pegasusx/apps/backend-go/outbox"
	"github.com/pegasusx/pegasusx/apps/backend-go/spannerutils"
)

// SpannerDomainExecutor executes approved L1 agent proposals against live Spanner
// tables (ReorderSuggestions, ArInvoices/Notifications, InventoryHolds/Notifications)
// alongside transactional OutboxEvents.
type SpannerDomainExecutor struct {
	Client        *spanner.Client
	Notifications *notifications.Service
}

// ExecuteProposalAction performs the real domain mutation for an approved proposal.
func (e *SpannerDomainExecutor) ExecuteProposalAction(ctx context.Context, p Proposal, executorID string) (map[string]any, error) {
	if e == nil || e.Client == nil {
		return map[string]any{
			"executor": "noop",
			"status":   "completed",
		}, nil
	}
	payload := decodePayloadMap(p.PayloadJSON)
	switch strings.ToUpper(strings.TrimSpace(p.ProposalType)) {
	case TypeReorder, TypeSupplyRequest:
		return e.execReorder(ctx, p, payload, executorID)
	case TypeCollections:
		return e.execCollections(ctx, p, payload, executorID)
	case TypeExpiryRebalance:
		return e.execExpiryRebalance(ctx, p, payload, executorID)
	default:
		return map[string]any{
			"executor":         "governed_proposal",
			"proposal_type":    p.ProposalType,
			"target_object_id": p.TargetObjectID,
			"status":           "completed",
		}, nil
	}
}

func (e *SpannerDomainExecutor) execReorder(ctx context.Context, p Proposal, payload map[string]any, executorID string) (map[string]any, error) {
	sugKey := strField(payload, "suggestion_key")
	if sugKey != "" {
		parts := strings.SplitN(sugKey, ":", 2)
		if len(parts) == 2 {
			err := spannerutils.RunReadWriteTransaction(ctx, e.Client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
				if err := txn.BufferWrite([]*spanner.Mutation{
					spanner.UpdateMap("ReorderSuggestions", map[string]any{
						"RetailerId": parts[0],
						"Sku":        parts[1],
						"Status":     "CONVERTED",
					}),
				}); err != nil {
					return err
				}
				buf := outbox.NewSpannerTxnBuffer(txn)
				stamped := &entityStampedOutboxBuf{inner: buf, entityID: p.TenantID}
				if err := outbox.EmitJSON(txCtx, stamped, "ReorderSuggestion", sugKey, events.TopicMain, map[string]any{
					"type":        "replenishment.suggestion.converted",
					"supplier_id": p.TenantID,
					"retailer_id": parts[0],
					"sku":         parts[1],
					"proposal_id": p.ProposalID,
					"executed_by": executorID,
				}); err != nil {
					return err
				}
				return buf.Flush(txCtx)
			})
			if err != nil {
				return nil, err
			}
			return map[string]any{
				"executor":       "spanner_reorder_suggestion",
				"suggestion_key": sugKey,
				"status":         "CONVERTED",
			}, nil
		}
	}

	whID := strField(payload, "warehouse_id")
	prodID := strField(payload, "product_id")
	qty := int64Field(payload, "suggested_qty")
	if qty <= 0 {
		qty = 1
	}
	notifID := uuid.NewString()
	err := spannerutils.RunReadWriteTransaction(ctx, e.Client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		metaRaw, _ := json.Marshal(map[string]any{
			"proposal_id":  p.ProposalID,
			"warehouse_id": whID,
			"product_id":   prodID,
			"qty":          qty,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("Notifications", map[string]any{
				"NotificationId": notifID,
				"RecipientId":    p.TenantID,
				"RecipientRole":  "ADMIN",
				"EventType":      "PROPOSAL_REORDER_EXECUTED",
				"Title":          p.Title,
				"Body":           fmt.Sprintf("Approved reorder proposal %s for %s (+%d units) executed by %s", p.ProposalID, prodID, qty, executorID),
				"DeepLink":       "/proposals",
				"MetadataJson":   metaRaw,
				"IsRead":         false,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"executor":        "spanner_reorder_dispatch",
		"notification_id": notifID,
		"product_id":      prodID,
		"warehouse_id":    whID,
		"qty":             qty,
		"status":          "completed",
	}, nil
}

func (e *SpannerDomainExecutor) execCollections(ctx context.Context, p Proposal, payload map[string]any, executorID string) (map[string]any, error) {
	invID := strField(payload, "invoice_id")
	if invID == "" {
		invID = p.TargetObjectID
	}
	retID := strField(payload, "retailer_id")
	lang := strings.ToLower(strField(payload, "preferred_lang"))
	draft := strField(payload, "draft_uz")
	if lang == "ru" && strField(payload, "draft_ru") != "" {
		draft = strField(payload, "draft_ru")
	}
	if draft == "" {
		draft = p.Summary
	}
	notifID := uuid.NewString()
	now := time.Now().UTC()

	err := spannerutils.RunReadWriteTransaction(ctx, e.Client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		row, err := txn.ReadRow(txCtx, "ArInvoices", spanner.Key{invID}, []string{"SupplierId", "RetailerId", "OrderId", "BalanceMinor", "DunningStep", "Version", "Status"})
		if err != nil {
			return err
		}
		var sid, rID, orderID, status string
		var bal, step, ver int64
		if err := row.Columns(&sid, &rID, &orderID, &bal, &step, &ver, &status); err != nil {
			return err
		}
		if sid != p.TenantID || (status != "OPEN" && status != "PARTIAL") || bal <= 0 {
			return fmt.Errorf("%w: invoice %s state %s bal=%d", ErrPreconditionFailed, invID, status, bal)
		}
		if retID == "" {
			retID = rID
		}
		nextStep := step + 1
		metaRaw, _ := json.Marshal(map[string]any{
			"proposal_id": p.ProposalID,
			"invoice_id":  invID,
			"draft_uz":    strField(payload, "draft_uz"),
			"draft_ru":    strField(payload, "draft_ru"),
		})
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.UpdateMap("ArInvoices", map[string]any{
				"InvoiceId":    invID,
				"DunningStep":  nextStep,
				"LastDunnedAt": now,
				"Version":      ver + 1,
				"UpdatedAt":    spanner.CommitTimestamp,
			}),
			spanner.InsertMap("Notifications", map[string]any{
				"NotificationId": notifID,
				"RecipientId":    nonEmptyOr(retID, p.TenantID),
				"RecipientRole":  "RETAILER",
				"EventType":      events.EventARInvoiceDunned,
				"Title":          p.Title,
				"Body":           draft,
				"DeepLink":       "/credit/collections",
				"MetadataJson":   metaRaw,
				"IsRead":         false,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		stamped := &entityStampedOutboxBuf{inner: buf, entityID: p.TenantID}
		if err := outbox.EmitJSON(txCtx, stamped, events.AggregateARInvoice, invID, events.TopicMain, events.ARInvoiceEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventARInvoiceDunned,
				Version:   1,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			InvoiceID:    invID,
			SupplierID:   p.TenantID,
			RetailerID:   retID,
			OrderID:      orderID,
			BalanceMinor: bal,
			DunningStep:  nextStep,
			LastDunnedAt: now.Format(time.RFC3339Nano),
		}); err != nil {
			return err
		}
		return buf.Flush(txCtx)
	})
	if err != nil {
		return nil, err
	}
	if e.Notifications != nil && retID != "" {
		_ = e.Notifications.CreateNotification(ctx, retID, "RETAILER", events.EventARInvoiceDunned, p.Title, draft, "/credit/collections")
	}
	return map[string]any{
		"executor":        "spanner_collections_dunning",
		"invoice_id":      invID,
		"retailer_id":     retID,
		"notification_id": notifID,
		"executed_by":     executorID,
		"status":          "completed",
	}, nil
}

func (e *SpannerDomainExecutor) execExpiryRebalance(ctx context.Context, p Proposal, payload map[string]any, executorID string) (map[string]any, error) {
	srcWH := strField(payload, "source_warehouse_id")
	dstWH := strField(payload, "target_warehouse_id")
	prodID := strField(payload, "product_id")
	lotID := strField(payload, "lot_id")
	qty := int64Field(payload, "qty_to_rebalance")
	if qty <= 0 {
		qty = 1
	}
	holdID := uuid.NewString()
	notifID := uuid.NewString()
	now := time.Now().UTC()

	err := spannerutils.RunReadWriteTransaction(ctx, e.Client, func(txCtx context.Context, txn *spanner.ReadWriteTransaction) error {
		holdLinesRaw, _ := json.Marshal([]map[string]any{
			{
				"product_id":          prodID,
				"lot_id":              lotID,
				"qty":                 qty,
				"target_warehouse_id": dstWH,
			},
		})
		metaRaw, _ := json.Marshal(map[string]any{
			"proposal_id":         p.ProposalID,
			"hold_id":             holdID,
			"lot_id":              lotID,
			"source_warehouse_id": srcWH,
			"target_warehouse_id": dstWH,
			"qty":                 qty,
		})
		if err := txn.BufferWrite([]*spanner.Mutation{
			spanner.InsertMap("InventoryHolds", map[string]any{
				"HoldId":        holdID,
				"SupplierId":    p.TenantID,
				"WarehouseId":   nonEmptyOr(srcWH, "default"),
				"ReferenceType": "EXPIRY_REBALANCE",
				"ReferenceId":   p.ProposalID,
				"Status":        "ACTIVE",
				"HoldLinesJson": holdLinesRaw,
				"ExpiresAt":     now.Add(48 * time.Hour),
				"CreatedAt":     spanner.CommitTimestamp,
				"UpdatedAt":     spanner.CommitTimestamp,
			}),
			spanner.InsertMap("Notifications", map[string]any{
				"NotificationId": notifID,
				"RecipientId":    p.TenantID,
				"RecipientRole":  "WAREHOUSE_ADMIN",
				"EventType":      events.EventInventoryRebalanceProposed,
				"Title":          p.Title,
				"Body":           fmt.Sprintf("Approved expiry rebalance %s: reserve %d units (%s -> %s) executed by %s", p.ProposalID, qty, srcWH, dstWH, executorID),
				"DeepLink":       "/proposals",
				"MetadataJson":   metaRaw,
				"IsRead":         false,
				"CreatedAt":      spanner.CommitTimestamp,
			}),
		}); err != nil {
			return err
		}
		buf := outbox.NewSpannerTxnBuffer(txn)
		stamped := &entityStampedOutboxBuf{inner: buf, entityID: p.TenantID}
		if err := outbox.EmitJSON(txCtx, stamped, events.AggregateWarehouse, nonEmptyOr(srcWH, p.TenantID), events.TopicMain, events.WarehouseEvent{
			BaseEvent: events.BaseEvent{
				Type:      events.EventInventoryQuantityUpdated,
				Version:   1,
				Timestamp: now.Format(time.RFC3339Nano),
			},
			WarehouseID: srcWH,
			SupplierID:  p.TenantID,
			LockID:      holdID,
			RequestID:   p.ProposalID,
			Status:      "EXECUTED",
			Units:       qty,
			Action:      "EXPIRY_REBALANCE",
		}); err != nil {
			return err
		}
		return buf.Flush(txCtx)
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"executor":            "spanner_expiry_rebalance",
		"hold_id":             holdID,
		"notification_id":     notifID,
		"source_warehouse_id": srcWH,
		"target_warehouse_id": dstWH,
		"product_id":          prodID,
		"lot_id":              lotID,
		"qty":                 qty,
		"status":              "completed",
	}, nil
}

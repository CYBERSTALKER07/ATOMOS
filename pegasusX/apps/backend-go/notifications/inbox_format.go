package notifications

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pegasusx/pegasusx/apps/backend-go/events"
)

// FormatFromEvent maps a Kafka notification envelope to inbox copy.
func FormatFromEvent(eventType string, payload []byte) FormattedNotification {
	eventType = strings.TrimSpace(eventType)
	if eventType == "" {
		return FormattedNotification{Title: "Update", Body: "You have a new notification", Priority: "normal"}
	}

	switch eventType {
	case events.EventOrderCreated:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatOrderCreated(e.OrderID, "", e.TotalMinor, e.Currency)
		}
	case events.EventOrderStatusChanged, events.EventOrderFinalized:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			status := strings.TrimSpace(e.Status)
			if status == "" {
				status = eventType
			}
			return FormatOrderStatusChanged(e.OrderID, status)
		}
	case events.EventOrderAssigned, events.EventOrderReassigned:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Route Assignment",
				Body:     fmt.Sprintf("Order %s assigned to execution", e.OrderID),
				DeepLink: "/orders/" + e.OrderID,
				Priority: "high",
			}
		}
	case events.EventManifestDraftCreated:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDraftCreated(e.ManifestID)
		}
	case events.EventManifestLoadingStarted:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestLoadingStarted(e.ManifestID, e.DriverID)
		}
	case events.EventManifestSealed:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestSealed(e.ManifestID)
		}
	case events.EventManifestOrderInjected:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestOrderInjected(e.ManifestID, e.OrderID)
		}
	case events.EventManifestOrderException:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestOrderException(e.ManifestID, e.OrderID, e.Reason)
		}
	case events.EventManifestExceptionResolved:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestExceptionResolved(e.ManifestID, e.Reason)
		}
	case events.EventFactoryStaffCreated:
		var e events.FactoryEvent
		if json.Unmarshal(payload, &e) == nil && e.UserID != "" {
			return FormatFactoryStaffCreated(e.UserID, e.FactoryID)
		}
	case events.EventFactoryTransferCreated:
		var e events.WarehouseTransferEvent
		if json.Unmarshal(payload, &e) == nil && e.TransferID != "" {
			return FormatFactoryTransferCreated(e.TransferID)
		}
		var wh events.WarehouseEvent
		if json.Unmarshal(payload, &wh) == nil && wh.TransferID != "" {
			return FormatFactoryTransferCreated(wh.TransferID)
		}
	case events.EventManifestDLQEscalation:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDLQEscalation(e.ManifestID, e.Reason)
		}
	case events.EventManifestRebalanced:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestRebalanced(e.ManifestID, e.FromManifestID, e.ToManifestID)
		}
	case events.EventManifestCancelled:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestCancelled(e.ManifestID, e.Reason)
		}
	case events.EventManifestDispatched:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestDispatched(e.ManifestID, e.DriverID)
		}
	case events.EventManifestCompleted:
		var e events.ManifestEvent
		if json.Unmarshal(payload, &e) == nil && e.ManifestID != "" {
			return FormatManifestCompleted(e.ManifestID, e.OrderCount)
		}
	case events.EventPaymentCleared, events.EventPaymentRequired, "PAYMENT_SETTLED":
		var e events.FinanceEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatPaymentReceived(e.OrderID, 0, "")
		}
	case events.EventShopClosed:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosed(e.OrderID, e.DriverID)
		}
	case events.EventShopClosedEscalated:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedEscalated(e.OrderID)
		}
	case events.EventShopClosedResolved:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedResolved(e.OrderID, e.Resolution)
		}
	case events.EventShopClosedResponse:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedResponse(e.OrderID, e.Response)
		}
	case events.EventShopClosedTimeout:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatShopClosedTimeout(e.OrderID, e.Resolution)
		}
	case events.EventProximityUnlocked:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatProximityUnlocked(e.OrderID, e.Status)
		}
	case events.EventPartialOffload:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatPartialOffload(e.OrderID)
		}
	case events.EventCreditLeave:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormatCreditLeave(e.OrderID)
		}
	case events.EventClaimFiled:
		var e events.LogisticsException
		if json.Unmarshal(payload, &e) == nil && e.ClaimID != "" {
			return FormatClaimFiled(e.ClaimID, e.OrderID, e.ClaimType)
		}
	case events.EventClaimUnderReview:
		var e events.LogisticsException
		if json.Unmarshal(payload, &e) == nil && e.ClaimID != "" {
			return FormatClaimUnderReview(e.ClaimID, e.OrderID)
		}
		// Map payloads use claim_id keys without LogisticsException shape.
		var m map[string]any
		if json.Unmarshal(payload, &m) == nil {
			cid, _ := m["claim_id"].(string)
			oid, _ := m["order_id"].(string)
			if cid != "" {
				return FormatClaimUnderReview(cid, oid)
			}
		}
	case events.EventClaimResolved:
		var e events.LogisticsException
		if json.Unmarshal(payload, &e) == nil && e.ClaimID != "" {
			return FormatClaimResolved(e.ClaimID, e.OrderID, e.Status)
		}
	case events.EventARInvoiceOpened, events.EventARInvoicePayment,
		events.EventARInvoiceSettled, events.EventARInvoiceDunned:
		var e events.ARInvoiceEvent
		if json.Unmarshal(payload, &e) == nil && e.InvoiceID != "" {
			return FormatARInvoice(e.Type, e.InvoiceID, e.OrderID, e.Status, e.BalanceMinor, "")
		}
	case events.EventPayoutBatchGenerated, events.EventPayoutBatchExported,
		events.EventPayoutBatchDispatched, events.EventPayoutBatchPaid:
		var e events.PayoutBatchEvent
		if json.Unmarshal(payload, &e) == nil && e.BatchID != "" {
			return FormatPayoutBatch(e.Type, e.BatchID, e.Status, e.NetPayoutMinor, e.Currency)
		}
	case events.EventLogisticsExceptionReported, events.EventReverseLogisticsRequired:
		var e events.LogisticsException
		if json.Unmarshal(payload, &e) == nil && (e.OrderID != "" || e.ClaimID != "") {
			return FormatLogisticsException(eventType, e.OrderID, e.ClaimID)
		}
	case events.EventDriverCreated:
		var e events.DriverEvent
		if json.Unmarshal(payload, &e) == nil && e.DriverID != "" {
			return FormatDriverCreated(e.DriverID, e.HomeNodeID)
		}
	case events.EventVehicleCreated:
		var e events.VehicleEvent
		if json.Unmarshal(payload, &e) == nil && e.VehicleID != "" {
			return FormatVehicleCreated(e.VehicleID, e.HomeNodeID)
		}
	case events.EventDriverAvailabilityChanged:
		var e events.DriverEvent
		if json.Unmarshal(payload, &e) == nil && e.DriverID != "" {
			return FormatDriverAvailabilityChanged(e.DriverID, e.OnShift, e.Reason, e.Note)
		}
	case events.EventVehicleAvailabilityChanged:
		var e events.VehicleEvent
		if json.Unmarshal(payload, &e) == nil && e.VehicleID != "" {
			return FormatVehicleAvailabilityChanged(e.VehicleID, e.IsActive, e.UnavailableReason, e.UnavailableNote)
		}
	case events.EventRetailerPriceOverride:
		var e events.RetailerPriceOverrideEvent
		if json.Unmarshal(payload, &e) == nil && e.RetailerID != "" {
			return FormatRetailerPriceOverride(e.ProductID, e.PriceMinor, "UZS", strings.EqualFold(e.Action, "CREATED"))
		}
	case events.EventPreOrderDateProposed:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Review delivery date",
				Body:     fmt.Sprintf("Warehouse proposed a new delivery date for order %s. Please review.", e.OrderID),
				DeepLink: "/orders/" + e.OrderID + "?review=1",
				Priority: "high",
			}
		}
	case events.EventPreOrderDateAccepted:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Delivery date accepted",
				Body:     fmt.Sprintf("Retailer accepted the proposed delivery date for order %s.", e.OrderID),
				DeepLink: "/orders/" + e.OrderID,
				Priority: "normal",
			}
		}
	case events.EventPreOrderDateRejected:
		var e events.OrderEvent
		if json.Unmarshal(payload, &e) == nil && e.OrderID != "" {
			return FormattedNotification{
				Title:    "Delivery proposal declined",
				Body:     fmt.Sprintf("Retailer declined the proposed delivery date. Order %s was cancelled.", e.OrderID),
				DeepLink: "/orders/" + e.OrderID,
				Priority: "high",
			}
		}
	case "cash_reconciliation.created", "cash_reconciliation.accepted", "cash_reconciliation.written_off", "cash_reconciliation.escalation":
		var e struct {
			ReconciliationID string `json:"reconciliation_id"`
			DriverID         string `json:"driver_id"`
			DifferenceMinor  int64  `json:"difference_minor"`
		}
		_ = json.Unmarshal(payload, &e)
		title := humanizeEventType(eventType)
		body := title
		if e.DifferenceMinor != 0 {
			body = fmt.Sprintf("%s — difference %d minor", title, e.DifferenceMinor)
		}
		return FormattedNotification{
			Title:    title,
			Body:     body,
			DeepLink: "/treasury/cash-reconciliations",
			Priority: "high",
		}
	case "credit_note.created", "credit_note.issued":
		var e struct {
			CreditNoteID string `json:"credit_note_id"`
			OrderID      string `json:"order_id"`
		}
		_ = json.Unmarshal(payload, &e)
		return FormattedNotification{
			Title:    humanizeEventType(eventType),
			Body:     fmt.Sprintf("Credit note %s for order %s", e.CreditNoteID, e.OrderID),
			DeepLink: "/finance/credit-notes",
			Priority: "normal",
		}
	case "reverse_logistics.task_created", "reverse_logistics.task_received":
		var e struct {
			TaskID  string `json:"task_id"`
			OrderID string `json:"order_id"`
		}
		_ = json.Unmarshal(payload, &e)
		return FormattedNotification{
			Title:    humanizeEventType(eventType),
			Body:     fmt.Sprintf("Reverse logistics task %s — order %s", e.TaskID, e.OrderID),
			DeepLink: "/exceptions",
			Priority: "normal",
		}
	case "credit.score.updated":
		var e struct {
			RetailerID string `json:"retailer_id"`
			Score      int64  `json:"score"`
			RiskTier   string `json:"risk_tier"`
		}
		_ = json.Unmarshal(payload, &e)
		return FormattedNotification{
			Title:    "Credit score updated",
			Body:     fmt.Sprintf("Retailer %s score %d (%s)", e.RetailerID, e.Score, e.RiskTier),
			DeepLink: "/credit/collections",
			Priority: "normal",
		}
	case "reorder.suggestion.updated":
		var e struct {
			RetailerID   string `json:"RetailerId"`
			Sku          string `json:"Sku"`
			SuggestedQty int64  `json:"SuggestedQty"`
		}
		_ = json.Unmarshal(payload, &e)
		if e.RetailerID == "" {
			var alt struct {
				RetailerID   string `json:"retailer_id"`
				Sku          string `json:"sku"`
				SuggestedQty int64  `json:"suggested_qty"`
			}
			_ = json.Unmarshal(payload, &alt)
			e.RetailerID = alt.RetailerID
			e.Sku = alt.Sku
			e.SuggestedQty = alt.SuggestedQty
		}
		return FormattedNotification{
			Title:    "Reorder suggestion",
			Body:     fmt.Sprintf("Suggest %d × %s for retailer %s", e.SuggestedQty, e.Sku, e.RetailerID),
			DeepLink: "/replenishment/suggestions",
			Priority: "normal",
		}
	}

	var generic struct {
		OrderID     string `json:"order_id"`
		ManifestID  string `json:"manifest_id"`
		Status      string `json:"status"`
		WarehouseID string `json:"warehouse_id"`
	}
	_ = json.Unmarshal(payload, &generic)

	title := humanizeEventType(eventType)
	body := title
	deepLink := ""
	if generic.OrderID != "" {
		body = fmt.Sprintf("%s — order %s", title, generic.OrderID)
		deepLink = "/orders/" + generic.OrderID
	} else if generic.ManifestID != "" {
		body = fmt.Sprintf("%s — manifest %s", title, generic.ManifestID)
		deepLink = "/manifests/" + generic.ManifestID
	} else if generic.Status != "" {
		body = fmt.Sprintf("%s (%s)", title, generic.Status)
	}

	return FormattedNotification{
		Title:    title,
		Body:     body,
		DeepLink: deepLink,
		Priority: "normal",
	}
}

// ShouldPersistInboxEvent returns false for high-frequency telemetry envelopes.
func ShouldPersistInboxEvent(eventType string) bool {
	switch strings.TrimSpace(eventType) {
	case events.EventDriverLocationUpdated, "TRACKING_POSITION", "TELEMETRY_PING", "FLEET_LOCATION":
		return false
	default:
		return !strings.HasPrefix(eventType, "SYSTEM")
	}
}

func humanizeEventType(eventType string) string {
	parts := strings.Split(strings.TrimSpace(eventType), "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return strings.Join(parts, " ")
}

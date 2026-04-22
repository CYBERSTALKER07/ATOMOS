// Package ws — Centralized WebSocket event type constants.
//
// Every WS message "type" field used across the backend MUST be defined here.
// This prevents typos, enables grep-auditing, and ensures frontend/mobile
// clients have a single source of truth for expected event names.
package ws

import "time"

// ─── Telemetry Hub → Admin Portal ──────────────────────────────────────────────

const (
	// EventOrderStateChanged is pushed when any order transitions state.
	EventOrderStateChanged = "ORDER_STATE_CHANGED"
	// EventDriverApproaching is pushed when a driver breaches the proximity perimeter.
	EventDriverApproaching = "DRIVER_APPROACHING"
	// EventETAUpdated is pushed when route ETAs are recalculated.
	EventETAUpdated = "ETA_UPDATED"
	// EventDriverAvailabilityChanged is pushed when a driver goes online/offline.
	EventDriverAvailabilityChanged = "DRIVER_AVAILABILITY_CHANGED"
	// EventOrderReassigned is pushed when an order moves between drivers.
	EventOrderReassigned = "ORDER_REASSIGNED"
	// EventTokenRefreshNeeded is sent to connections operating in JWT grace period.
	EventTokenRefreshNeeded = "TOKEN_REFRESH_NEEDED"
)

// ─── Retailer Hub → Retailer Apps ──────────────────────────────────────────────

const (
	// EventPaymentRequired is pushed when offload is confirmed and payment is due.
	EventPaymentRequired = "PAYMENT_REQUIRED"
	// EventPaymentExpired is pushed when a payment session times out.
	EventPaymentExpired = "PAYMENT_EXPIRED"
	// EventPaymentFailed is pushed when a payment attempt fails.
	EventPaymentFailed = "PAYMENT_FAILED"
	// EventPaymentSettled is pushed when payment is confirmed by the gateway.
	EventPaymentSettled = "PAYMENT_SETTLED"
	// EventOrderCompleted is pushed when a delivery is finalized.
	EventOrderCompleted = "ORDER_COMPLETED"
	// EventOrderStatusChanged is pushed for generic order state changes to retailers.
	EventOrderStatusChanged = "ORDER_STATUS_CHANGED"
	// EventOrderAmended is pushed when order line items are amended.
	EventOrderAmended = "ORDER_AMENDED"
	// EventPreOrderConfirmation is pushed for pre-order confirmations.
	EventPreOrderConfirmation = "PRE_ORDER_CONFIRMATION"
	// EventPreOrderNudge is the T-5 soft reminder before the hard T-4 confirmation request.
	EventPreOrderNudge = "PRE_ORDER_NUDGE"
	// EventPreOrderAutoAccepted is pushed when a scheduled order is auto-accepted at T-4.
	EventPreOrderAutoAccepted = "PRE_ORDER_AUTO_ACCEPTED"
	// EventPreOrderConfirmed is pushed when a retailer explicitly confirms a preorder.
	EventPreOrderConfirmed = "PRE_ORDER_CONFIRMED"
	// EventPreOrderEdited is pushed when a retailer edits a scheduled preorder.
	EventPreOrderEdited = "PRE_ORDER_EDITED"
)

// ─── Driver Hub → Driver Apps ──────────────────────────────────────────────────

const (
	// EventCashCollectionRequired is pushed when driver must collect cash.
	EventCashCollectionRequired = "CASH_COLLECTION_REQUIRED"
	// EventEarlyCompleteRequested is pushed to supplier when driver requests early route completion.
	EventEarlyCompleteRequested = "EARLY_COMPLETE_REQUESTED"
	// EventEarlyCompleteApproved is pushed to driver when early completion is approved.
	EventEarlyCompleteApproved = "EARLY_COMPLETE_APPROVED"
	// EventShopClosedAlert is pushed when a shop-closed report is filed.
	EventShopClosedAlert = "SHOP_CLOSED_ALERT"
	// EventShopClosedEscalated is pushed when shop-closed escalates to supplier.
	EventShopClosedEscalated = "SHOP_CLOSED_ESCALATED"
	// EventBypassTokenIssued is pushed when supplier issues a payment bypass token.
	EventBypassTokenIssued = "BYPASS_TOKEN_ISSUED"
)

// ─── Negotiation Events ────────────────────────────────────────────────────────

const (
	// EventNegotiationProposed is pushed when a driver proposes quantity changes.
	EventNegotiationProposed = "NEGOTIATION_PROPOSED"
	// EventNegotiationResolved is pushed when a negotiation proposal is approved/rejected.
	EventNegotiationResolved = "NEGOTIATION_RESOLVED"
)

// ─── Credit & Returns Events ───────────────────────────────────────────────────

const (
	// EventCreditDeliveryMarked is pushed when an order is marked as credit delivery.
	EventCreditDeliveryMarked = "CREDIT_DELIVERY_MARKED"
	// EventCreditDeliveryResolved is pushed when credit delivery is resolved.
	EventCreditDeliveryResolved = "CREDIT_DELIVERY_RESOLVED"
	// EventMissingItemsReported is pushed when driver reports missing items post-seal.
	EventMissingItemsReported = "MISSING_ITEMS_REPORTED"
	// EventReturnResolved is pushed when a return is processed.
	EventReturnResolved = "RETURN_RESOLVED"
	// EventOrderRejectedBySupplier is pushed when supplier rejects an order.
	EventOrderRejectedBySupplier = "ORDER_REJECTED_BY_SUPPLIER"
)

// ─── Warehouse Hub → Warehouse Terminals ───────────────────────────────────────

const (
	// EventSupplyRequestUpdate is pushed when a supply request state changes.
	EventSupplyRequestUpdate = "SUPPLY_REQUEST_UPDATE"
	// EventDispatchLockChange is pushed when a dispatch lock is acquired/released.
	EventDispatchLockChange = "DISPATCH_LOCK_CHANGE"
)

// ─── Notifications / FCM ───────────────────────────────────────────────────────

const (
	// EventAIPrediction is used for AI prediction push notifications.
	EventAIPrediction = "AI_PREDICTION"
	// EventSystemBroadcast is used for system-wide broadcast messages.
	EventSystemBroadcast = "SYSTEM_BROADCAST"
)

// ─── Delta-Sync Protocol ───────────────────────────────────────────────────────
// Compact, role-agnostic envelope for differential real-time updates.
// Instead of sending the full JSON state (~5KB), only the atomic delta (~200B)
// is transmitted — ~90% bandwidth reduction per WebSocket frame.
//
// Short-key convention:
//   Field names are compressed to 1-3 character keys for wire efficiency.
//   The client maintains a local cache and performs shallow-merge on receipt.

// DeltaEvent is the compressed envelope for all real-time differential updates.
// Wire format: {"t":"ORD_UP","i":"uuid","d":{"s":"LOADED"},"ts":1713456000}
type DeltaEvent struct {
	T  string                 `json:"t"`  // Event type (e.g., "ORD_UP", "DRV_MOV")
	I  string                 `json:"i"`  // Entity ID
	D  map[string]interface{} `json:"d"`  // The delta (changed fields only, short-keyed)
	TS int64                  `json:"ts"` // Unix timestamp
}

// NewDelta creates a DeltaEvent with the current Unix timestamp.
func NewDelta(eventType string, entityID string, delta map[string]interface{}) DeltaEvent {
	return DeltaEvent{
		T:  eventType,
		I:  entityID,
		D:  delta,
		TS: time.Now().Unix(),
	}
}

// ── Delta Event Types (Shortened) ──────────────────────────────────────────
// These are the "t" values used on the wire. Kept short for bandwidth.
const (
	DeltaOrderUpdate   = "ORD_UP"  // Order field(s) changed
	DeltaDriverUpdate  = "DRV_UP"  // Driver field(s) changed
	DeltaFleetGPS      = "FLT_GPS" // Batched GPS delta (significant moves only)
	DeltaWarehouseLoad = "WH_LOAD" // Warehouse load threshold crossed
	DeltaWarehouseNew  = "WRH_NEW" // New warehouse node created
	DeltaPaymentUpdate = "PAY_UP"  // Payment state changed
	DeltaRouteUpdate   = "RTE_UP"  // Route/ETA changed
	DeltaNegotiation   = "NEG_UP"  // Negotiation state changed
	DeltaCreditUpdate  = "CRD_UP"  // Credit delivery state changed
)

// ── V.O.I.D. Short-Key Dictionary ───────────────────────────────────────────
// The canonical 8-key dictionary for maximum byte savings on the wire.
// Every field in every delta payload MUST use these keys.
//
// | Short | Original          | Context                    |
// |-------|-------------------|----------------------------|
// | s     | status            | Order/Driver/Warehouse     |
// | l     | location          | [lat, lng] float64 array   |
// | v     | volumetric_units  | Current cargo load         |
// | i     | id                | Primary UUID/ID            |
// | o     | order_id          | References                 |
// | d     | driver_id         | References                 |
// | w     | warehouse_id      | References                 |
// | at    | updated_at        | Unix Timestamp             |

// ShortKeyMap provides long→short key mappings for delta compression.
var ShortKeyMap = map[string]string{
	"status":           "s",
	"state":            "s",
	"location":         "l",
	"volumetric_units": "v",
	"volume_units":     "v",
	"id":               "i",
	"order_id":         "o",
	"driver_id":        "d",
	"warehouse_id":     "w",
	"updated_at":       "at",
}

// CompressDelta converts a map with long keys to short keys using ShortKeyMap.
// Unknown keys are passed through unchanged.
func CompressDelta(fields map[string]interface{}) map[string]interface{} {
	compressed := make(map[string]interface{}, len(fields))
	for k, v := range fields {
		if short, ok := ShortKeyMap[k]; ok {
			compressed[short] = v
		} else {
			compressed[k] = v
		}
	}
	return compressed
}

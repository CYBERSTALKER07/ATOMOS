package kafka

import "time"

// ─── Kafka Event Type Constants ───────────────────────────────────────────────
// Event keys are consumed from both Pegasus and legacy logistics topics during
// transition. The notification dispatcher switches on these keys.

const (
	EventOrderDispatched           = "ORDER_DISPATCHED"
	EventDriverApproaching         = "DRIVER_APPROACHING"
	EventDriverArrived             = "DRIVER_ARRIVED"
	EventOrderStatusChanged        = "ORDER_STATUS_CHANGED"
	EventPayloadReadyToSeal        = "PAYLOAD_READY_TO_SEAL"
	EventPayloadSealed             = "PAYLOAD_SEALED"
	EventPaymentSettled            = "PAYMENT_SETTLED"
	EventPaymentFailed             = "PAYMENT_FAILED"
	EventPaymentIntentCreated      = "PAYMENT_INTENT_CREATED"
	EventOrderCompleted            = "ORDER_COMPLETED"
	EventDriverAvailabilityChanged = "DRIVER_AVAILABILITY_CHANGED"
	EventOrderReassigned           = "ORDER_REASSIGNED"
	EventOrderModified             = "ORDER_MODIFIED"
	EventOrderRerouted             = "ORDER_REROUTED" // Scaffolding: no producer, no consumer — planned for warehouse rebalancing

	// Phase I: Shop-Closed Contact Protocol events
	EventShopClosed          = "SHOP_CLOSED"
	EventShopClosedResponse  = "SHOP_CLOSED_RESPONSE"
	EventShopClosedEscalated = "SHOP_CLOSED_ESCALATED"
	EventShopClosedResolved  = "SHOP_CLOSED_RESOLVED"

	// Phase I: Cancel Request events
	EventCancelRequested = "CANCEL_REQUESTED"
	EventCancelApproved  = "CANCEL_APPROVED"

	// Phase II: Human-Centric Edge Cases (FRIDAY v3.1)
	EventEarlyCompleteRequested = "EARLY_COMPLETE_REQUESTED"
	EventEarlyCompleteApproved  = "EARLY_COMPLETE_APPROVED"
	EventNegotiationProposed    = "NEGOTIATION_PROPOSED"
	EventNegotiationResolved    = "NEGOTIATION_RESOLVED"
	EventCreditDeliveryMarked   = "CREDIT_DELIVERY_MARKED"
	EventCreditDeliveryResolved = "CREDIT_DELIVERY_RESOLVED"
	EventMissingItemsReported   = "MISSING_ITEMS_REPORTED"
	EventPowerOutageReported    = "POWER_OUTAGE_REPORTED"
	EventAiOrderConfirmed       = "AI_ORDER_CONFIRMED"
	EventAiOrderRejected        = "AI_ORDER_REJECTED"
	EventSplitPaymentCreated    = "SPLIT_PAYMENT_CREATED"

	// Phase IV: Warehouse Supply Chain & Pre-Order Policy
	EventSupplyRequestSubmitted    = "SUPPLY_REQUEST_SUBMITTED"
	EventSupplyRequestAcknowledged = "SUPPLY_REQUEST_ACKNOWLEDGED"
	EventSupplyRequestReady        = "SUPPLY_REQUEST_READY"
	EventSupplyRequestFulfilled    = "SUPPLY_REQUEST_FULFILLED"
	EventSupplyRequestCancelled    = "SUPPLY_REQUEST_CANCELLED"
	EventManifestRebalanced        = "MANIFEST_REBALANCED"
	EventTransferUnassigned        = "TRANSFER_UNASSIGNED"
	EventManifestCancelled         = "MANIFEST_CANCELLED"
	EventOrderCancelLocked         = "ORDER_CANCEL_LOCKED"     // T-4 cancel-lock fires, freezes cancellation
	EventPreOrderNotified          = "PRE_ORDER_NOTIFIED"      // T-4 confirmation notification sent
	EventPreOrderAutoAccepted      = "PRE_ORDER_AUTO_ACCEPTED" // SCHEDULED → AUTO_ACCEPTED by midnight guard
	EventPreOrderConfirmed         = "PRE_ORDER_CONFIRMED"     // Retailer explicitly confirmed a scheduled preorder
	EventPreOrderEdited            = "PRE_ORDER_EDITED"        // Retailer edited a scheduled preorder (date or items)
	EventPreOrderCancelled         = "PRE_ORDER_CANCELLED"     // Retailer cancelled a scheduled preorder
	EventDispatchLockAcquired      = "DISPATCH_LOCK_ACQUIRED"
	EventDispatchLockReleased      = "DISPATCH_LOCK_RELEASED"
	EventFreezeLockAcquired        = "FREEZE_LOCK_ACQUIRED"
	EventFreezeLockReleased        = "FREEZE_LOCK_RELEASED"

	// Dispatch Pipeline Events
	EventRouteCreated           = "ROUTE_CREATED"
	EventOrderAssigned          = "ORDER_ASSIGNED"
	EventFactoryManifestCreated = "FACTORY_MANIFEST_CREATED"
	EventDemandForecastReady    = "DEMAND_FORECAST_READY"

	// Outbox Observability
	EventOutboxFailed = "OUTBOX_FAILED"

	// Pegasus topic names.
	TopicFreezeLocks    = "pegasus-freeze-locks"
	TopicMain           = "pegasus-logistics-events"
	TopicDemandForecast = "pegasus-demand-forecast"
	TopicDriverSync     = "pegasus-driver-sync-events"
	TopicMainDLQ        = "pegasus-logistics-events-dlq"

	// Fleet Entity Lifecycle Events
	EventDriverCreated  = "DRIVER_CREATED"
	EventVehicleCreated = "VEHICLE_CREATED"

	// Phase VI: Warehouse Sovereignty Events
	EventWarehouseCreated        = "WAREHOUSE_CREATED"
	EventWarehouseSpatialUpdated = "WAREHOUSE_SPATIAL_UPDATED"
	EventWarehouseStatusChanged  = "WAREHOUSE_STATUS_CHANGED"
	EventOutOfStock              = "OUT_OF_STOCK"
	EventRetailerPriceOverride   = "RETAILER_PRICE_OVERRIDE"

	// Node Sovereignty Events: emitted atomically with Spanner row creation
	// via outbox. Producers: factory/crud.go#createFactory,
	// supplier/retailer_register.go. Consumers: analytics, search index,
	// supplier portal real-time list refresh (external, via WS hub).
	EventFactoryCreated     = "FACTORY_CREATED"
	EventRetailerRegistered = "RETAILER_REGISTERED"

	// Internal Transfer Lifecycle Events
	EventTransferStateChanged = "TRANSFER_STATE_CHANGED"
	EventTransferApproved     = "TRANSFER_APPROVED"
	EventTransferReceived     = "TRANSFER_RECEIVED"

	// Fleet & Delivery Edge Events
	EventFleetDispatched             = "FLEET_DISPATCHED"
	EventOrderCancelled              = "ORDER_CANCELLED"
	EventOrderCancelledByOrigin      = "ORDER_CANCELLED_BY_ORIGIN"
	EventPayloadOverflow             = "PAYLOAD_OVERFLOW"
	EventFulfillmentPaymentCompleted = "FULFILLMENT_PAYMENT_COMPLETED"
	EventFulfillmentPaid             = "FULFILLMENT_PAID"
	EventOffloadConfirmed            = "OFFLOAD_CONFIRMED"
	EventCashCollectionRequired      = "CASH_COLLECTION_REQUIRED"
	EventPaymentBypassIssued         = "PAYMENT_BYPASS_ISSUED"
	EventPaymentBypassCompleted      = "PAYMENT_BYPASS_COMPLETED"
	EventSmsQuickComplete            = "SMS_QUICK_COMPLETE"

	// Checkout Events
	EventStockBackordered         = "STOCK_BACKORDERED"
	EventOrderCreated             = "ORDER_CREATED"
	EventUnifiedCheckoutCompleted = "UNIFIED_CHECKOUT_COMPLETED"

	// Payment Events
	EventPaymentRefunded = "PAYMENT_REFUNDED" // Producer: payment/refund.go via outbox; consumer: mobile push + admin refund log (external).

	// Manifest Alerts
	EventForceSealAlert = "FORCE_SEAL_ALERT"

	// ── Phase V: LEO Event Integrity (Loading Gate) ──────────────────────────
	// EventOrderDelayed — emitted when MarkDelayed transitions an order to DELAYED state
	// (capacity overflow at the loading bay or upstream supply gap).
	EventOrderDelayed = "ORDER_DELAYED"
	// EventManifestOrderReassigned — emitted when a single order is moved between
	// LOADING-state manifests during payloader override. Differs from EventOrderReassigned,
	// which is the route/driver-level reassignment signal.
	EventManifestOrderReassigned = "MANIFEST_ORDER_REASSIGNED"
	// EventPayloadSync — UI refresh signal pushed to PayloaderHub when manifest state
	// changes externally (admin override, kill switch, AI rebalance). Best-effort UX hint.
	EventPayloadSync = "PAYLOAD_SYNC"
	// EventInternalLoadConfirmed — payloader confirmed physical loading of a single order
	// onto the truck. Counterpart to EventOffloadConfirmed (delivery side).
	EventInternalLoadConfirmed = "INTERNAL_LOAD_CONFIRMED"

	// AI / Sync Events (emitter.go — driver-sync + logistics topics)
	EventOrderSync             = "ORDER_SYNC"
	EventAiPredictionCorrected = "AI_PREDICTION_CORRECTED"
	EventAiPlanDateShift       = "AI_PLAN_DATE_SHIFT"
	EventAiPlanSkuModified     = "AI_PLAN_SKU_MODIFIED"
)

// ─── Notification Event Payloads ──────────────────────────────────────────────
// These structs are published to Kafka and consumed by the notification dispatcher.

// OrderDispatchedEvent is emitted when a route is assigned to orders.
// Recipients: Retailer (your order is on the way) + Driver (new dispatch).
type OrderDispatchedEvent struct {
	RouteID     string    `json:"route_id"`
	OrderIDs    []string  `json:"order_ids"`
	DriverID    string    `json:"driver_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DriverArrivedEvent is emitted when a driver marks an order as ARRIVED.
// Recipient: Retailer (driver is at your location).
type DriverArrivedEvent struct {
	OrderID     string    `json:"order_id"`
	RetailerID  string    `json:"retailer_id"`
	DriverID    string    `json:"driver_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderStatusChangedEvent is emitted on every order state transition.
// Recipient: Retailer (order status update).
type OrderStatusChangedEvent struct {
	OrderID     string    `json:"order_id"`
	RetailerID  string    `json:"retailer_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	OldState    string    `json:"old_state"`
	NewState    string    `json:"new_state"`
	Timestamp   time.Time `json:"timestamp"`
}

// PayloadReadyToSealEvent is emitted when orders are dispatched and payloader needs to seal.
// Recipient: Payloader (orders ready for sealing).
type PayloadReadyToSealEvent struct {
	RouteID     string    `json:"route_id"`
	OrderIDs    []string  `json:"order_ids"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// PaymentSettledEvent is emitted after a successful payment settlement.
// Recipients: Retailer (payment confirmed) + Driver (payment received for delivery).
type PaymentSettledEvent struct {
	OrderID     string    `json:"order_id"`
	InvoiceID   string    `json:"invoice_id"`
	RetailerID  string    `json:"retailer_id"`
	DriverID    string    `json:"driver_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Gateway     string    `json:"gateway"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Timestamp   time.Time `json:"timestamp"`
}

// PaymentFailedEvent is emitted after a payment settlement failure.
// Recipient: Retailer (payment failed, retry needed).
type PaymentFailedEvent struct {
	OrderID     string    `json:"order_id"`
	InvoiceID   string    `json:"invoice_id"`
	RetailerID  string    `json:"retailer_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Gateway     string    `json:"gateway"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// DriverAvailabilityChangedEvent is emitted when a driver goes online or offline.
// Recipients: Supplier (admin portal fleet page update).
type DriverAvailabilityChangedEvent struct {
	DriverID    string    `json:"driver_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Available   bool      `json:"available"`
	Reason      string    `json:"reason,omitempty"`
	Note        string    `json:"note,omitempty"`
	TruckID     string    `json:"truck_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderReassignedEvent is emitted when orders are moved between routes/trucks.
// The producer emits route-level data; driver IDs are resolved from the manifests
// so the notification dispatcher can target old/new drivers.
// Recipients: Old Driver (order removed) + New Driver (new order assigned) + Supplier (admin update).
type OrderReassignedEvent struct {
	OrderIDs    []string  `json:"order_ids"`
	OldRouteID  string    `json:"old_route_id"`
	NewRouteID  string    `json:"new_route_id"`
	OldDriverID string    `json:"old_driver_id,omitempty"`
	NewDriverID string    `json:"new_driver_id,omitempty"`
	SupplierID  string    `json:"supplier_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderModifiedEvent is emitted when an order is amended (partial delivery reconciliation).
// Recipients: Supplier (order amended, refund delta) + Retailer (adjusted invoice).
type OrderModifiedEvent struct {
	OrderID     string    `json:"order_id"`
	AmendmentID string    `json:"amendment_id"`
	DriverID    string    `json:"driver_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	RetailerID  string    `json:"retailer_id"`
	NewAmount   int64     `json:"new_amount"`
	Refunded    int64     `json:"refunded"`
	Currency    string    `json:"currency"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderReroutedEvent is emitted when an order is rerouted from an overloaded warehouse
// to a sibling warehouse with lower load. Different semantic from ORDER_REASSIGNED
// (driver change) — this is a warehouse-level rebalancing event.
// Recipients: Supplier (admin portal load heatmap + audit trail).
type OrderReroutedEvent struct {
	OrderID             string    `json:"order_id,omitempty"`
	SupplierID          string    `json:"supplier_id"`
	OriginalWarehouseId string    `json:"original_warehouse_id"`
	NewWarehouseId      string    `json:"new_warehouse_id"`
	OriginalLoadPercent float64   `json:"original_load_percent"`
	NewLoadPercent      float64   `json:"new_load_percent"`
	RetailerLat         float64   `json:"retailer_lat"`
	RetailerLng         float64   `json:"retailer_lng"`
	DistanceKm          float64   `json:"distance_km"`
	Timestamp           time.Time `json:"timestamp"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase I: Shop-Closed Contact Protocol Event Payloads
// ═══════════════════════════════════════════════════════════════════════════════

// OrderCompletedEvent is emitted when an order reaches the COMPLETED state.
// Recipients: Treasurer (ledger split), Reconciler (post-delivery reconciliation).
// Amount + SupplierId are carried inline so the Treasurer can split without extra Spanner reads.
type OrderCompletedEvent struct {
	OrderID     string    `json:"order_id"`
	RetailerID  string    `json:"retailer_id"`
	SupplierId  string    `json:"supplier_id,omitempty"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	Amount      int64     `json:"amount,omitempty"`
	Currency    string    `json:"currency,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// PaymentIntentEvent is emitted by the Treasurer when ledger entries are written
// with Status=PENDING_GATEWAY. The Gateway Worker consumes this event to execute
// the actual charge against the payment provider (GlobalPay, Cash, etc.) with an
// idempotency key that prevents double-charging on replay.
type PaymentIntentEvent struct {
	OrderID            string `json:"order_id"`
	SupplierId         string `json:"supplier_id"`
	Amount             int64  `json:"amount"`
	Currency           string `json:"currency"`
	PaymentGateway     string `json:"payment_gateway"`
	IdempotencyKey     string `json:"idempotency_key"`
	PlatformCommission int64  `json:"platform_commission"`
	SupplierPayout     int64  `json:"supplier_payout"`
	PlatformTxnId      string `json:"platform_txn_id"`
	SupplierTxnId      string `json:"supplier_txn_id"`

	// Global Pay auth-capture fields (omitempty for backward compat with GlobalPay/Cash consumers)
	AuthorizationID  string `json:"authorization_id,omitempty"`
	AuthorizedAmount int64  `json:"authorized_amount,omitempty"`
	FinalAmount      int64  `json:"final_amount,omitempty"`
}

// FleetDispatchedEvent is emitted when a batch of orders is assigned to a truck/route.
// Recipients: Notification dispatcher (driver + retailer notification).
type FleetDispatchedEvent struct {
	RouteID     string    `json:"route_id"`
	ManifestID  string    `json:"manifest_id,omitempty"`
	OrderIDs    []string  `json:"order_ids"`
	DriverID    string    `json:"driver_id,omitempty"`
	SupplierID  string    `json:"supplier_id,omitempty"`
	WarehouseId string    `json:"warehouse_id,omitempty"`
	GeoZone     string    `json:"geo_zone,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// PayloadSealedEvent is emitted when a warehouse worker seals a pallet and dispatches it.
// Recipients: Notification dispatcher (payload terminal + driver notification).
type PayloadSealedEvent struct {
	OrderID       string    `json:"order_id"`
	TerminalID    string    `json:"terminal_id"`
	DeliveryToken string    `json:"delivery_token"`
	Timestamp     time.Time `json:"timestamp"`
}

// OffloadConfirmedEvent is emitted when a driver confirms offload at a retailer.
// Recipients: Payment session, notification dispatcher.
type OffloadConfirmedEvent struct {
	OrderID        string    `json:"order_id"`
	RetailerID     string    `json:"retailer_id"`
	Amount         int64     `json:"amount"`
	OriginalAmount int64     `json:"original_amount"`
	PaymentMethod  string    `json:"payment_method"`
	Timestamp      time.Time `json:"timestamp"`
}

// ShopClosedEvent is emitted when a driver reports a shop as closed.
type ShopClosedEvent struct {
	OrderID    string    `json:"order_id"`
	DriverID   string    `json:"driver_id"`
	RetailerID string    `json:"retailer_id"`
	SupplierID string    `json:"supplier_id"`
	AttemptID  string    `json:"attempt_id"`
	GPSLat     float64   `json:"gps_lat"`
	GPSLng     float64   `json:"gps_lng"`
	Timestamp  time.Time `json:"timestamp"`
}

// ShopClosedResponseEvent is emitted when a retailer responds to a shop-closed alert.
type ShopClosedResponseEvent struct {
	OrderID    string    `json:"order_id"`
	RetailerID string    `json:"retailer_id"`
	AttemptID  string    `json:"attempt_id"`
	Response   string    `json:"response"` // OPEN_NOW | 5_MIN | CALL_ME | CLOSED_TODAY
	Timestamp  time.Time `json:"timestamp"`
}

// ShopClosedEscalatedEvent is emitted when a shop-closed case is escalated to admin.
type ShopClosedEscalatedEvent struct {
	OrderID     string    `json:"order_id"`
	AttemptID   string    `json:"attempt_id"`
	SupplierID  string    `json:"supplier_id"`
	EscalatedTo string    `json:"escalated_to"`
	Timestamp   time.Time `json:"timestamp"`
}

// ShopClosedResolvedEvent is emitted when an admin resolves a shop-closed case.
type ShopClosedResolvedEvent struct {
	OrderID    string    `json:"order_id"`
	AttemptID  string    `json:"attempt_id"`
	Resolution string    `json:"resolution"` // RETAILER_OPENED | BYPASS_ISSUED | RETURN_TO_DEPOT | WAITING
	ResolvedBy string    `json:"resolved_by"`
	Timestamp  time.Time `json:"timestamp"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Consumer Backoff (I-2)
// ═══════════════════════════════════════════════════════════════════════════════

// ConsumerBackoff tracks exponential backoff state for Kafka consumer error loops.
// Reset to 0 on successful read; sleep before retry on consecutive errors.
type ConsumerBackoff struct {
	consecutive int
}

// Sleep returns the backoff duration and increments the error counter.
// Exponential: 100ms, 200ms, 400ms, 800ms, 1.6s, ... capped at 30s.
func (b *ConsumerBackoff) Sleep() time.Duration {
	b.consecutive++
	d := time.Duration(100<<uint(b.consecutive-1)) * time.Millisecond
	if d > 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// Reset clears the error counter on successful read.
func (b *ConsumerBackoff) Reset() {
	b.consecutive = 0
}

// Consecutive returns the current streak of consecutive errors.
func (b *ConsumerBackoff) Consecutive() int {
	return b.consecutive
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase II: Human-Centric Edge Case Event Payloads (FRIDAY v3.1)
// ═══════════════════════════════════════════════════════════════════════════════

// EarlyCompleteRequestedEvent is emitted when a driver requests early route completion.
type EarlyCompleteRequestedEvent struct {
	DriverID   string    `json:"driver_id"`
	SupplierID string    `json:"supplier_id"`
	RouteID    string    `json:"route_id"`
	OrderIDs   []string  `json:"order_ids"`
	Reason     string    `json:"reason"` // FATIGUE | TRAFFIC | VEHICLE_ISSUE | OTHER
	Note       string    `json:"note,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// NegotiationProposedEvent is emitted when a driver proposes quantity changes.
type NegotiationProposedEvent struct {
	ProposalID string    `json:"proposal_id"`
	OrderID    string    `json:"order_id"`
	DriverID   string    `json:"driver_id"`
	SupplierID string    `json:"supplier_id"`
	RetailerID string    `json:"retailer_id"`
	Timestamp  time.Time `json:"timestamp"`
}

// NegotiationResolvedEvent is emitted when a supplier resolves a negotiation.
type NegotiationResolvedEvent struct {
	ProposalID string    `json:"proposal_id"`
	OrderID    string    `json:"order_id"`
	SupplierID string    `json:"supplier_id"`
	Action     string    `json:"action"` // APPROVED | REJECTED
	Timestamp  time.Time `json:"timestamp"`
}

// CreditDeliveryEvent is emitted when an order is delivered on credit or resolved.
type CreditDeliveryEvent struct {
	OrderID    string    `json:"order_id"`
	RetailerID string    `json:"retailer_id"`
	SupplierID string    `json:"supplier_id"`
	DriverID   string    `json:"driver_id"`
	Amount     int64     `json:"amount"`
	Currency   string    `json:"currency"`
	Action     string    `json:"action,omitempty"` // APPROVED | DENIED (for resolved events)
	Timestamp  time.Time `json:"timestamp"`
}

// MissingItemsEvent is emitted when a driver reports items missing after seal.
type MissingItemsEvent struct {
	OrderID    string    `json:"order_id"`
	DriverID   string    `json:"driver_id"`
	SupplierID string    `json:"supplier_id"`
	ItemCount  int       `json:"item_count"`
	Timestamp  time.Time `json:"timestamp"`
}

// SplitPaymentEvent is emitted when a driver creates a split payment.
type SplitPaymentEvent struct {
	OrderID      string    `json:"order_id"`
	DriverID     string    `json:"driver_id"`
	FirstAmount  int64     `json:"first_amount"`
	SecondAmount int64     `json:"second_amount"`
	Timestamp    time.Time `json:"timestamp"`
}

// AiOrderEvent is emitted when a retailer confirms or rejects an AI-suggested order.
type AiOrderEvent struct {
	OrderID    string    `json:"order_id"`
	RetailerID string    `json:"retailer_id"`
	Action     string    `json:"action"` // CONFIRMED | REJECTED
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// ═══════════════════════════════════════════════════════════════════════════════
// Phase IV: Warehouse Supply Chain & Pre-Order Policy Event Payloads
// ═══════════════════════════════════════════════════════════════════════════════

// SupplyRequestEvent is emitted on supply request state transitions.
type SupplyRequestEvent struct {
	RequestID   string    `json:"request_id"`
	WarehouseID string    `json:"warehouse_id"`
	FactoryID   string    `json:"factory_id"`
	SupplierID  string    `json:"supplier_id"`
	State       string    `json:"state"`
	Priority    string    `json:"priority"`
	Timestamp   time.Time `json:"timestamp"`
}

// ManifestRebalancedEvent is emitted when orders are reassigned between manifests during loading.
type ManifestRebalancedEvent struct {
	FactoryID        string    `json:"factory_id"`
	SupplierID       string    `json:"supplier_id"`
	SourceManifestID string    `json:"source_manifest_id"`
	TargetManifestID string    `json:"target_manifest_id"`
	TransferIDs      []string  `json:"transfer_ids"`
	Reason           string    `json:"reason"` // CAPACITY_MISMATCH | MANUAL_OVERRIDE
	RebalancedBy     string    `json:"rebalanced_by"`
	Timestamp        time.Time `json:"timestamp"`
}

// DispatchLockEvent is emitted when a dispatch lock is acquired or released.
type DispatchLockEvent struct {
	LockID      string    `json:"lock_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	FactoryID   string    `json:"factory_id,omitempty"`
	LockType    string    `json:"lock_type"`
	LockedBy    string    `json:"locked_by"`
	Timestamp   time.Time `json:"timestamp"`
}

// PreOrderNotifiedEvent is emitted when the T-4 confirmation notification is sent.
type PreOrderNotifiedEvent struct {
	OrderID      string    `json:"order_id"`
	RetailerID   string    `json:"retailer_id"`
	SupplierID   string    `json:"supplier_id"`
	DeliveryDate time.Time `json:"delivery_date"`
	Timestamp    time.Time `json:"timestamp"`
}

// OrderCancelLockedEvent is emitted when the T-4 auto-lock fires and freezes cancellation.
type OrderCancelLockedEvent struct {
	OrderID    string    `json:"order_id"`
	RetailerID string    `json:"retailer_id"`
	SupplierID string    `json:"supplier_id"`
	Reason     string    `json:"reason"` // AI_POLICY | MANUAL_POLICY
	Timestamp  time.Time `json:"timestamp"`
}

// PreOrderAutoAcceptedEvent is emitted when the midnight guard promotes
// a cancel-locked SCHEDULED order to AUTO_ACCEPTED.
type PreOrderAutoAcceptedEvent struct {
	OrderID      string    `json:"order_id"`
	RetailerID   string    `json:"retailer_id"`
	SupplierID   string    `json:"supplier_id"`
	DeliveryDate string    `json:"delivery_date"`
	Timestamp    time.Time `json:"timestamp"`
}

// PreOrderConfirmedEvent is emitted when a retailer explicitly confirms a preorder.
type PreOrderConfirmedEvent struct {
	OrderID     string    `json:"order_id"`
	ConfirmedBy string    `json:"confirmed_by"`
	Timestamp   time.Time `json:"timestamp"`
}

// PreOrderEditedEvent is emitted when a retailer edits a scheduled preorder.
type PreOrderEditedEvent struct {
	OrderID   string    `json:"order_id"`
	EditedBy  string    `json:"edited_by"`
	NewDate   string    `json:"new_date,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// PreOrderCancelledEvent is emitted when a retailer cancels a scheduled preorder.
type PreOrderCancelledEvent struct {
	OrderID     string    `json:"order_id"`
	CancelledBy string    `json:"cancelled_by"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// ── PHASE V: LEO — LOADING GATE EVENTS ──────────────────────────────────────
// All 10 producers wired. Two consumers wired (MANIFEST_DISPATCHED,
// MANIFEST_COMPLETED in notification_dispatcher.go); the remaining lifecycle
// events still have no consumers — wire them as product surfaces require.

const (
	// MANIFEST_DRAFT_CREATED — auto-dispatcher created a DRAFT manifest (planning phase, not yet loading)
	EventManifestDraftCreated = "MANIFEST_DRAFT_CREATED"
	// MANIFEST_LOADING_STARTED — payloader selected truck, manifest entered LOADING (mutable) phase
	EventManifestLoadingStarted = "MANIFEST_LOADING_STARTED"
	// MANIFEST_SEALED — payloader performed "Slide to Seal", manifest locked. JIT route triggered.
	EventManifestSealed = "MANIFEST_SEALED"
	// MANIFEST_DISPATCHED — driver departed, manifest entered DISPATCHED phase
	EventManifestDispatched = "MANIFEST_DISPATCHED"
	// MANIFEST_COMPLETED — all stops delivered or quarantined, manifest is terminal
	EventManifestCompleted = "MANIFEST_COMPLETED"
	// MANIFEST_ORDER_EXCEPTION — payloader reported overflow/damage/manual exception during LOADING
	EventManifestOrderException = "MANIFEST_ORDER_EXCEPTION"
	// MANIFEST_DLQ_ESCALATION — order hit 3x overflow threshold, frozen for admin resolution
	EventManifestDLQEscalation = "MANIFEST_DLQ_ESCALATION"
	// ROUTE_FINALIZED — JIT route optimization complete, driver route pushed
	EventRouteFinalized = "ROUTE_FINALIZED"
	// MANIFEST_ORDER_INJECTED — mid-load addition: order injected into LOADING manifest
	EventManifestOrderInjected = "MANIFEST_ORDER_INJECTED"
	// MANIFEST_FORCE_SEALED — admin override seal, volumetric validation bypassed
	EventManifestForceSeal = "MANIFEST_FORCE_SEALED"

	// Phase VIII: Replenishment Graph Hardening Events
	EventStockThresholdBreach      = "STOCK_THRESHOLD_BREACH" // Scaffolding: no producer, no consumer — planned for proximity.Engine threshold detection
	EventReplenishmentLockAcquired = "REPLENISHMENT_LOCK_ACQUIRED"
	EventReplenishmentLockReleased = "REPLENISHMENT_LOCK_RELEASED"
	EventPullMatrixCompleted       = "PULL_MATRIX_COMPLETED"
	EventFactorySLABreach          = "FACTORY_SLA_BREACH"
	EventInboundFreightUnannounced = "INBOUND_FREIGHT_UNANNOUNCED"
	EventSupplyLaneTransitUpdated  = "SUPPLY_LANE_TRANSIT_UPDATED"
	EventNetworkModeChanged        = "NETWORK_MODE_CHANGED"

	// Phase V: Pull Matrix Look-Ahead
	EventReplenishmentTransferCreated   = "REPLENISHMENT_TRANSFER_CREATED"
	EventInsightApprovedTransferCreated = "INSIGHT_APPROVED_TRANSFER_CREATED"
	EventLookAheadCompleted             = "LOOK_AHEAD_COMPLETED" // Scaffolding: no producer, no consumer — planned for pull-matrix look-ahead completion signal
)

// ManifestLifecycleEvent covers DRAFT_CREATED, LOADING_STARTED, SEALED, DISPATCHED, COMPLETED.
type ManifestLifecycleEvent struct {
	ManifestID  string    `json:"manifest_id"`
	SupplierId  string    `json:"supplier_id"`
	DriverID    string    `json:"driver_id"`
	TruckID     string    `json:"truck_id"`
	State       string    `json:"state"`
	StopCount   int       `json:"stop_count"`
	VolumeVU    float64   `json:"volume_vu"`
	MaxVolumeVU float64   `json:"max_volume_vu"`
	SealedBy    string    `json:"sealed_by,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ManifestOrderExceptionEvent is emitted when a payloader reports an exception during LOADING.
type ManifestOrderExceptionEvent struct {
	ExceptionID  string    `json:"exception_id"`
	ManifestID   string    `json:"manifest_id"`
	OrderID      string    `json:"order_id"`
	SupplierId   string    `json:"supplier_id"`
	Reason       string    `json:"reason"` // OVERFLOW | DAMAGED | MANUAL
	AttemptCount int64     `json:"attempt_count"`
	Escalated    bool      `json:"escalated"` // true if AttemptCount >= 3
	Metadata     string    `json:"metadata,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// RouteFinalizedEvent is emitted after JIT route optimization on SEALED.
type RouteFinalizedEvent struct {
	ManifestID string    `json:"manifest_id"`
	DriverID   string    `json:"driver_id"`
	StopCount  int       `json:"stop_count"`
	RouteJSON  string    `json:"route_json,omitempty"` // serialized waypoints
	Timestamp  time.Time `json:"timestamp"`
}

// ManifestOrderInjectedEvent is emitted when an order is added to a LOADING manifest mid-load.
type ManifestOrderInjectedEvent struct {
	ManifestID       string    `json:"manifest_id"`
	OrderID          string    `json:"order_id"`
	SupplierId       string    `json:"supplier_id"`
	NewTotalVolumeVU float64   `json:"new_total_volume_vu"`
	InjectedBy       string    `json:"injected_by"`
	Timestamp        time.Time `json:"timestamp"`
}

// ManifestForceSealEvent is emitted when an admin force-seals a manifest, bypassing volumetric validation.
type ManifestForceSealEvent struct {
	ManifestID   string    `json:"manifest_id"`
	SupplierId   string    `json:"supplier_id"`
	SealedBy     string    `json:"sealed_by"`
	Override     bool      `json:"override"`
	VolumeAtSeal float64   `json:"volume_at_seal"`
	MaxVolumeVU  float64   `json:"max_volume_vu"`
	Reason       string    `json:"reason,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// ── Phase VI: Exception Handling — Bifurcated Recovery ─────────────────────────

// OrderCancelledByOriginEvent is emitted when an admin/warehouse-admin kills an order
// before dispatch (hard stop). Treasurer voids pending ledger entries; notification
// dispatcher fans out 3-way (warehouse + supplier + retailer).
type OrderCancelledByOriginEvent struct {
	OrderID     string    `json:"order_id"`
	SupplierId  string    `json:"supplier_id"`
	WarehouseId string    `json:"warehouse_id"`
	RetailerId  string    `json:"retailer_id"`
	ManifestID  string    `json:"manifest_id,omitempty"`
	Reason      string    `json:"reason"`
	CancelledBy string    `json:"cancelled_by"` // user id of the actor
	Amount      int64     `json:"amount"`
	Timestamp   time.Time `json:"timestamp"`
}

// PayloadOverflowEvent is emitted when payload doesn't fit the truck (soft stop).
// Order returns to the unassigned pool for redispatch — no financial void.
type PayloadOverflowEvent struct {
	OrderID      string    `json:"order_id"`
	SupplierId   string    `json:"supplier_id"`
	WarehouseId  string    `json:"warehouse_id"`
	ManifestID   string    `json:"manifest_id"`
	Reason       string    `json:"reason"` // OVERFLOW | VOLUME_EXCEEDED
	AttemptCount int64     `json:"attempt_count"`
	Timestamp    time.Time `json:"timestamp"`
}

// ── Phase VI: Settlement Engine Events ─────────────────────────────────────────

const (
	// MANIFEST_SETTLED — emitted by the Treasurer when manifest-level reconciliation completes.
	// Recipients: notification dispatcher (supplier treasury notification), analytics.
	EventManifestSettled = "MANIFEST_SETTLED"
)

// ManifestSettlementEvent carries the financial summary produced when a manifest completes.
// It reconciles all per-order ledger entries and flags anomalies.
type ManifestSettlementEvent struct {
	ManifestID     string    `json:"manifest_id"`
	SupplierId     string    `json:"supplier_id"`
	TotalOrders    int       `json:"total_orders"`
	SettledOrders  int       `json:"settled_orders"`
	TotalAmount    int64     `json:"total_amount"`
	CashAmount     int64     `json:"cash_amount"`
	DigitalAmount  int64     `json:"digital_amount"`
	PlatformFee    int64     `json:"platform_fee"`
	SupplierPayout int64     `json:"supplier_payout"`
	AnomalyCount   int       `json:"anomaly_count"`
	Currency       string    `json:"currency"`
	Timestamp      time.Time `json:"timestamp"`
}

// ── Phase VI: Warehouse Sovereignty Events ────────────────────────────────────

// WarehouseCreatedEvent is emitted when a new warehouse node is provisioned.
// Used by analytics engines, map layers, and the delta-sync broadcaster.
type WarehouseCreatedEvent struct {
	WarehouseId    string    `json:"warehouse_id"`
	SupplierId     string    `json:"supplier_id"`
	Name           string    `json:"name"`
	Lat            float64   `json:"lat"`
	Lng            float64   `json:"lng"`
	H3Count        int       `json:"h3_count"`
	CoverageRadius float64   `json:"coverage_radius_km"`
	Timestamp      time.Time `json:"timestamp"`
}

// WarehouseSpatialUpdatedEvent is emitted when warehouse coverage (H3Indexes,
// Lat/Lng, CoverageRadiusKm) is modified. Triggers background retailer
// re-assignment by the spatial materialization worker.
type WarehouseSpatialUpdatedEvent struct {
	WarehouseId    string    `json:"warehouse_id"`
	SupplierId     string    `json:"supplier_id"`
	OldH3Count     int       `json:"old_h3_count"`
	NewH3Count     int       `json:"new_h3_count"`
	CoverageRadius float64   `json:"coverage_radius_km"`
	Timestamp      time.Time `json:"timestamp"`
}

// FactoryCreatedEvent is emitted when a new factory node is provisioned by
// a GLOBAL_ADMIN. Consumed by analytics, the supplier-portal node list
// refresh, and the cache-warming relay for warehouse↔factory edges.
type FactoryCreatedEvent struct {
	FactoryId            string    `json:"factory_id"`
	SupplierId           string    `json:"supplier_id"`
	Name                 string    `json:"name"`
	Lat                  float64   `json:"lat"`
	Lng                  float64   `json:"lng"`
	H3Index              string    `json:"h3_index,omitempty"`
	RegionCode           string    `json:"region_code"`
	LeadTimeDays         int64     `json:"lead_time_days"`
	ProductionCapacityVU float64   `json:"production_capacity_vu"`
	ProductTypes         []string  `json:"product_types,omitempty"`
	WarehousesLinked     int       `json:"warehouses_linked"`
	Timestamp            time.Time `json:"timestamp"`
}

// RetailerRegisteredEvent is emitted when a retailer self-registers via the
// public registration endpoint. Consumed by the catalog discovery indexer,
// supplier-portal new-retailer feed, and the AI demand-forecaster cold-start
// bootstrapper.
type RetailerRegisteredEvent struct {
	RetailerId  string    `json:"retailer_id"`
	OwnerName   string    `json:"owner_name"`
	ShopName    string    `json:"shop_name"`
	PhoneNumber string    `json:"phone_number"`
	Lat         float64   `json:"lat"`
	Lng         float64   `json:"lng"`
	H3Cell      string    `json:"h3_cell"`
	RegionCode  string    `json:"region_code"`
	Timestamp   time.Time `json:"timestamp"`
}

// WarehouseStatusChangedEvent is emitted when IsActive or IsOnShift toggles.
// Notification dispatcher uses this to push status changes to affected retailers.
type WarehouseStatusChangedEvent struct {
	WarehouseId string    `json:"warehouse_id"`
	SupplierId  string    `json:"supplier_id"`
	Field       string    `json:"field"` // "is_active" | "is_on_shift"
	OldValue    bool      `json:"old_value"`
	NewValue    bool      `json:"new_value"`
	Reason      string    `json:"reason,omitempty"` // DisabledReason if deactivating
	Timestamp   time.Time `json:"timestamp"`
}

// OutOfStockEvent is emitted when an order fails due to insufficient stock.
// Recipients: Retailer (items unavailable) + Admin (stock depletion alert).
type OutOfStockEvent struct {
	OrderID      string           `json:"order_id,omitempty"`
	RetailerID   string           `json:"retailer_id"`
	SupplierId   string           `json:"supplier_id"`
	WarehouseId  string           `json:"warehouse_id"`
	ShortfallMap map[string]int64 `json:"shortfall_map"` // sku_id → quantity short
	Timestamp    time.Time        `json:"timestamp"`
}

// RetailerPriceOverrideEvent is emitted when a per-retailer price override is
// created, updated, or deactivated. Used for audit and analytics.
type RetailerPriceOverrideEvent struct {
	OverrideId string    `json:"override_id"`
	SupplierId string    `json:"supplier_id"`
	RetailerId string    `json:"retailer_id"`
	SkuId      string    `json:"sku_id"`
	Price      int64     `json:"price"`
	Action     string    `json:"action"` // "CREATED" | "UPDATED" | "DEACTIVATED"
	SetBy      string    `json:"set_by"`
	SetByRole  string    `json:"set_by_role"` // "GLOBAL_ADMIN" | "NODE_ADMIN"
	Timestamp  time.Time `json:"timestamp"`
}

// ── Phase VIII: Replenishment Graph Hardening Events ──────────────────────────

// StockThresholdBreachEvent is emitted when a warehouse SKU hits its safety stock level.
type StockThresholdBreachEvent struct {
	SupplierId   string    `json:"supplier_id"`
	WarehouseId  string    `json:"warehouse_id"`
	ProductId    string    `json:"product_id"`
	CurrentStock int64     `json:"current_stock"`
	SafetyLevel  int64     `json:"safety_level"`
	Timestamp    time.Time `json:"timestamp"`
}

// ReplenishmentLockEvent is emitted when a warehouse acquires or releases a replenishment lock.
type ReplenishmentLockEvent struct {
	LockKey     string    `json:"lock_key"`
	WarehouseId string    `json:"warehouse_id"`
	SupplierId  string    `json:"supplier_id"`
	Priority    float64   `json:"priority"`
	Action      string    `json:"action"` // "ACQUIRED" | "RELEASED" | "PREEMPTED"
	Timestamp   time.Time `json:"timestamp"`
}

// PullMatrixCompletedEvent is emitted after a Pull Matrix aggregation run completes.
type PullMatrixCompletedEvent struct {
	RunId              string    `json:"run_id"`
	SupplierId         string    `json:"supplier_id"`
	TransfersGenerated int64     `json:"transfers_generated"`
	SKUsProcessed      int64     `json:"skus_processed"`
	DurationMs         int64     `json:"duration_ms"`
	Source             string    `json:"source"` // "CRON" | "EVENT_TRIGGERED" | "MANUAL"
	Timestamp          time.Time `json:"timestamp"`
}

// FactorySLABreachEvent is emitted at each escalation level for a stalled transfer.
type FactorySLABreachEvent struct {
	TransferId      string    `json:"transfer_id"`
	FactoryId       string    `json:"factory_id"`
	WarehouseId     string    `json:"warehouse_id"`
	SupplierId      string    `json:"supplier_id"`
	EscalationLevel string    `json:"escalation_level"` // "WARNING" | "CRITICAL" | "AUTO_REROUTE"
	SLABreachMin    int64     `json:"sla_breach_minutes"`
	ReplacementId   string    `json:"replacement_transfer_id,omitempty"`
	Timestamp       time.Time `json:"timestamp"`
}

// InboundFreightUnannouncedEvent is emitted when a warehouse force-receives
// freight that was not tracked in InternalTransferOrders.
type InboundFreightUnannouncedEvent struct {
	TransferId  string    `json:"transfer_id"` // retroactively created transfer
	WarehouseId string    `json:"warehouse_id"`
	SupplierId  string    `json:"supplier_id"`
	ItemsCount  int       `json:"items_count"`
	ReceivedBy  string    `json:"received_by"`
	Timestamp   time.Time `json:"timestamp"`
}

// SupplyLaneTransitUpdatedEvent is emitted when a dampened transit time change
// exceeds the propagation threshold (>15% delta).
type SupplyLaneTransitUpdatedEvent struct {
	LaneId           string    `json:"lane_id"`
	SupplierId       string    `json:"supplier_id"`
	FactoryId        string    `json:"factory_id"`
	WarehouseId      string    `json:"warehouse_id"`
	OldDampenedHours float64   `json:"old_dampened_hours"`
	NewDampenedHours float64   `json:"new_dampened_hours"`
	RawTransitHours  float64   `json:"raw_transit_hours"`
	Timestamp        time.Time `json:"timestamp"`
}

// NetworkModeChangedEvent is emitted when a supplier toggles the optimization mode.
type NetworkModeChangedEvent struct {
	SupplierId string    `json:"supplier_id"`
	OldMode    string    `json:"old_mode"`
	NewMode    string    `json:"new_mode"`
	ChangedBy  string    `json:"changed_by"`
	Reason     string    `json:"reason,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// ── Fleet Entity Lifecycle Events ─────────────────────────────────────────────

// DriverCreatedEvent is emitted when a new driver is registered.
type DriverCreatedEvent struct {
	DriverID     string    `json:"driver_id"`
	SupplierId   string    `json:"supplier_id"`
	Name         string    `json:"name"`
	Phone        string    `json:"phone"`
	DriverType   string    `json:"driver_type"`
	HomeNodeType string    `json:"home_node_type,omitempty"`
	HomeNodeId   string    `json:"home_node_id,omitempty"`
	CreatedBy    string    `json:"created_by"`
	Timestamp    time.Time `json:"timestamp"`
}

// VehicleCreatedEvent is emitted when a new vehicle is registered.
type VehicleCreatedEvent struct {
	VehicleID    string    `json:"vehicle_id"`
	SupplierId   string    `json:"supplier_id"`
	VehicleClass string    `json:"vehicle_class"`
	Label        string    `json:"label"`
	LicensePlate string    `json:"license_plate"`
	MaxVolumeVU  float64   `json:"max_volume_vu"`
	HomeNodeType string    `json:"home_node_type,omitempty"`
	HomeNodeId   string    `json:"home_node_id,omitempty"`
	CreatedBy    string    `json:"created_by"`
	Timestamp    time.Time `json:"timestamp"`
}

// ── Dispatch Pipeline Event Payloads ─────────────────────────────────────────

// RouteCreatedEvent is emitted when auto-dispatch creates a new route manifest.
type RouteCreatedEvent struct {
	RouteID     string    `json:"route_id"`
	ManifestID  string    `json:"manifest_id,omitempty"`
	DriverID    string    `json:"driver_id"`
	TruckID     string    `json:"truck_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	FactoryID   string    `json:"factory_id,omitempty"`
	StopCount   int       `json:"stop_count"`
	VolumeVU    float64   `json:"volume_vu"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderAssignedEvent is emitted per-order when auto-dispatch assigns it to a route.
type OrderAssignedEvent struct {
	OrderID     string    `json:"order_id"`
	RouteID     string    `json:"route_id"`
	DriverID    string    `json:"driver_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DemandForecastReadyEvent is published by the AI worker when a demand
// forecast completes. Backend and portal consume it asynchronously.
type DemandForecastReadyEvent struct {
	RetailerID  string    `json:"retailer_id"`
	WarehouseID string    `json:"warehouse_id"`
	SupplierID  string    `json:"supplier_id"`
	SKUCount    int       `json:"sku_count"`
	Timestamp   time.Time `json:"timestamp"`
}

// ── Phase V: LEO Event Integrity Payloads ────────────────────────────────────

// ManifestCancelledEvent is emitted when a LOADING-state manifest is cancelled
// (factory override, supplier kill switch, or admin force-cancel). Released
// transfers/orders revert to their pre-loading state.
type ManifestCancelledEvent struct {
	ManifestID   string    `json:"manifest_id"`
	SupplierID   string    `json:"supplier_id"`
	FactoryID    string    `json:"factory_id,omitempty"`
	WarehouseID  string    `json:"warehouse_id,omitempty"`
	ReleasedIDs  []string  `json:"released_ids"`  // transfer or order IDs released
	ReleasedKind string    `json:"released_kind"` // "TRANSFER" | "ORDER"
	Reason       string    `json:"reason"`        // MANUAL_OVERRIDE | KILL_SWITCH | CAPACITY_EXCEEDED
	CancelledBy  string    `json:"cancelled_by"`
	Timestamp    time.Time `json:"timestamp"`
}

// TransferUnassignedEvent is emitted when a single transfer is removed from a
// LOADING manifest. Differs from ManifestCancelledEvent (whole-manifest scope).
type TransferUnassignedEvent struct {
	ManifestID   string    `json:"manifest_id"`
	TransferID   string    `json:"transfer_id"`
	FactoryID    string    `json:"factory_id"`
	SupplierID   string    `json:"supplier_id"`
	Reason       string    `json:"reason,omitempty"`
	UnassignedBy string    `json:"unassigned_by"`
	Timestamp    time.Time `json:"timestamp"`
}

// ForceSealAlertEvent is emitted when a supplier crosses the 3/5 force-seal
// override threshold within a 24h window. Recipients: supplier admin + warehouse
// staff (operational alert).
type ForceSealAlertEvent struct {
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	ManifestID  string    `json:"manifest_id"`
	Count24h    int64     `json:"count_24h"`
	Quota       int64     `json:"quota"`
	SealedBy    string    `json:"sealed_by"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderDelayedEvent is emitted when MarkDelayed transitions an order to DELAYED.
// Recipients: retailer (your order is delayed) + supplier admin (operational queue).
type OrderDelayedEvent struct {
	OrderID     string    `json:"order_id"`
	RetailerID  string    `json:"retailer_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	ManifestID  string    `json:"manifest_id,omitempty"`
	Reason      string    `json:"reason"` // CAPACITY_OVERFLOW | UPSTREAM_SHORTAGE | MANUAL
	Timestamp   time.Time `json:"timestamp"`
}

// ManifestOrderReassignedEvent is emitted when a single order is moved between
// LOADING-state manifests. Per-order scope (vs OrderReassignedEvent which is
// route/driver-level). Recipients: old driver, new driver, supplier admin.
type ManifestOrderReassignedEvent struct {
	OrderID          string    `json:"order_id"`
	SourceManifestID string    `json:"source_manifest_id"`
	TargetManifestID string    `json:"target_manifest_id"`
	OldDriverID      string    `json:"old_driver_id,omitempty"`
	NewDriverID      string    `json:"new_driver_id,omitempty"`
	SupplierID       string    `json:"supplier_id"`
	WarehouseID      string    `json:"warehouse_id,omitempty"`
	Reason           string    `json:"reason,omitempty"`
	ReassignedBy     string    `json:"reassigned_by"`
	Timestamp        time.Time `json:"timestamp"`
}

// PayloadSyncEvent is the UI-refresh signal pushed to PayloaderHub when a
// payload-visible manifest mutates. Reason is the upstream mutation source
// (for example MANIFEST_SEALED or MANIFEST_ORDER_EXCEPTION), not a user-facing string.
type PayloadSyncEvent struct {
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	ManifestID  string    `json:"manifest_id"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

// InternalLoadConfirmedEvent is emitted when a payloader confirms physical
// loading of a single order onto the truck. Counterpart to OffloadConfirmedEvent.
type InternalLoadConfirmedEvent struct {
	OrderID     string    `json:"order_id"`
	ManifestID  string    `json:"manifest_id"`
	SupplierID  string    `json:"supplier_id"`
	WarehouseID string    `json:"warehouse_id,omitempty"`
	DriverID    string    `json:"driver_id,omitempty"`
	TruckID     string    `json:"truck_id,omitempty"`
	ConfirmedBy string    `json:"confirmed_by"` // payloader user_id
	VolumeVU    float64   `json:"volume_vu"`
	Timestamp   time.Time `json:"timestamp"`
}

package events

import "github.com/pegasusx/pegasusx/apps/backend-go/orderstage"

// Canonical Event Structs for Kafka Outbox Publishing.
// These structs enforce consistency across the backend and satisfy the
// enterprise kafka-event-contracts rules.

// BaseEvent holds fields universal to almost all events, though Type is omitted from
// some since it's injected by EmitJSON or the relay header. However, providing it
// here allows consumers to parse it from the body for convenience if it's there.
type BaseEvent struct {
	Type      string `json:"type,omitempty"`
	Version   int64  `json:"version"` // For optimistic concurrency and idempotency
	Timestamp string `json:"timestamp,omitempty"`
}

// ARInvoiceEvent is emitted on AR open-item lifecycle changes (open / payment /
// settled / dunned). Fields are omitempty because each event type populates the
// subset relevant to it (e.g. principal_minor only on open, dunning fields only
// on dunned).
type ARInvoiceEvent struct {
	BaseEvent
	InvoiceID      string `json:"invoice_id"`
	SupplierID     string `json:"supplier_id"`
	RetailerID     string `json:"retailer_id"`
	OrderID        string `json:"order_id,omitempty"`
	PrincipalMinor int64  `json:"principal_minor,omitempty"`
	AmountMinor    int64  `json:"amount_minor,omitempty"`
	BalanceMinor   int64  `json:"balance_minor,omitempty"`
	Status         string `json:"status,omitempty"`
	DunningStep    int64  `json:"dunning_step,omitempty"`
	AgingBucket    string `json:"aging_bucket,omitempty"`
	DueAt          string `json:"due_at,omitempty"`
	LastDunnedAt   string `json:"last_dunned_at,omitempty"`
}

// PayoutBatchEvent is emitted on the supplier payout lifecycle (generated /
// exported / dispatched / paid).
type PayoutBatchEvent struct {
	BaseEvent
	BatchID        string `json:"batch_id"`
	SupplierID     string `json:"supplier_id"`
	Status         string `json:"status"`
	NetPayoutMinor int64  `json:"net_payout_minor"`
	Currency       string `json:"currency"`
	RailReference  string `json:"rail_reference,omitempty"`
}

// SupplierServicePromiseEvent handles supplier promise snapshots and breach alerts.
type SupplierServicePromiseEvent struct {
	BaseEvent
	OrderID                string `json:"order_id"`
	SupplierID             string `json:"supplier_id"`
	RetailerID             string `json:"retailer_id"`
	WarehouseID            string `json:"warehouse_id,omitempty"`
	PromiseType            string `json:"promise_type"`
	GuaranteedDeliveryDate string `json:"guaranteed_delivery_date"`
	CutoffAppliedAt        string `json:"cutoff_applied_at,omitempty"`
	FillRateTargetBps      int64  `json:"fill_rate_target_bps"`
	MinOrderMinor          int64  `json:"min_order_minor"`
	Currency               string `json:"currency"`
	SLAHours               int64  `json:"sla_hours"`
	Status                 string `json:"status"`
	BreachedAt             string `json:"breached_at,omitempty"`
	BreachReason           string `json:"breach_reason,omitempty"`
}

// LotRecallEvent handles lot recall campaign lifecycle and quarantine actions.
type LotRecallEvent struct {
	BaseEvent
	CampaignID         string `json:"campaign_id,omitempty"`
	SupplierID         string `json:"supplier_id"`
	ProductID          string `json:"product_id"`
	LotCode            string `json:"lot_code,omitempty"`
	LotID              string `json:"lot_id,omitempty"`
	WarehouseID        string `json:"warehouse_id,omitempty"`
	RecallReason       string `json:"recall_reason,omitempty"`
	Severity           string `json:"severity,omitempty"`
	Status             string `json:"status,omitempty"`
	ImpactedLotCount   int64  `json:"impacted_lot_count,omitempty"`
	ImpactedUnitsCount int64  `json:"impacted_units_count,omitempty"`
	ImpactedOrderCount int64  `json:"impacted_order_count,omitempty"`
	Action             string `json:"action,omitempty"`
	Actor              string `json:"actor,omitempty"`
}

// SupplierEvent handles generic supplier operations.
type SupplierEvent struct {
	BaseEvent
	SupplierID          string   `json:"supplier_id"`
	LegalName           string   `json:"legal_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	Email               string   `json:"email,omitempty"`
	BankName            string   `json:"bank_name,omitempty"`
	AccountHolder       string   `json:"account_holder,omitempty"`
	SelectedGateways    []string `json:"selected_gateways,omitempty"`
	UserID              string   `json:"user_id,omitempty"`
	SupplierRole        string   `json:"supplier_role,omitempty"`
	AssignedWarehouseID string   `json:"assigned_warehouse_id,omitempty"`
	AssignedFactoryID   string   `json:"assigned_factory_id,omitempty"`
	Phone               string   `json:"phone,omitempty"`
	Country             string   `json:"country,omitempty"`
	Categories          []string `json:"categories,omitempty"`
	IsRegistered        bool     `json:"is_registered,omitempty"`
	IsConfigured        bool     `json:"is_configured,omitempty"`
	Action              string   `json:"action,omitempty"`
}

// RetailerEvent handles retailer registration.
type RetailerEvent struct {
	BaseEvent
	RetailerID string `json:"retailer_id"`
	Phone      string `json:"phone,omitempty"`
	Name       string `json:"name,omitempty"`

	SupplierID          string   `json:"supplier_id"`
	LegalName           string   `json:"legal_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	Email               string   `json:"email,omitempty"`
	BankName            string   `json:"bank_name,omitempty"`
	AccountHolder       string   `json:"account_holder,omitempty"`
	SelectedGateways    []string `json:"selected_gateways,omitempty"`
	UserID              string   `json:"user_id,omitempty"`
	SupplierRole        string   `json:"supplier_role,omitempty"`
	AssignedWarehouseID string   `json:"assigned_warehouse_id,omitempty"`
	Lat                 float64  `json:"lat,omitempty"`
	Lng                 float64  `json:"lng,omitempty"`
	H3Cell              string   `json:"h3_cell,omitempty"`
	CountryCode         string   `json:"country_code,omitempty"`
}

// FactoryEvent handles factory creation and operational events.
type FactoryEvent struct {
	BaseEvent
	FactoryID           string   `json:"factory_id"`
	SupplierID          string   `json:"supplier_id"`
	Lat                 float64  `json:"lat,omitempty"`
	Lng                 float64  `json:"lng,omitempty"`
	H3Cell              string   `json:"h3_cell,omitempty"`
	LegalName           string   `json:"legal_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	Email               string   `json:"email,omitempty"`
	BankName            string   `json:"bank_name,omitempty"`
	AccountHolder       string   `json:"account_holder,omitempty"`
	SelectedGateways    []string `json:"selected_gateways,omitempty"`
	UserID              string   `json:"user_id,omitempty"`
	SupplierRole        string   `json:"supplier_role,omitempty"`
	AssignedWarehouseID string   `json:"assigned_warehouse_id,omitempty"`
}

// WarehouseEvent handles warehouse creation and operational events.
type WarehouseEvent struct {
	BaseEvent
	WarehouseID       string `json:"warehouse_id"`
	SupplierID        string `json:"supplier_id"`
	FactoryID         string `json:"factory_id,omitempty"`
	TransferMode      string `json:"transfer_mode,omitempty"`
	LinkedTransferID  string `json:"linked_transfer_id,omitempty"`
	TransferID        string `json:"transfer_id,omitempty"`
	LockID            string `json:"lock_id,omitempty"`
	RequestID         string `json:"request_id,omitempty"`
	Status            string `json:"status,omitempty"`
	Units             int64  `json:"requested_units,omitempty"`
	Projected         int64  `json:"projected_units,omitempty"`
	Committed         int64  `json:"committed_units,omitempty"`
	Pending           int64  `json:"pending_confirmation_units,omitempty"`
	State             string `json:"state,omitempty"`
	RequestedBy       string `json:"requested_by,omitempty"`
	CoverageStartDate string `json:"coverage_start_date,omitempty"`
	CoverageDays      int64  `json:"coverage_days,omitempty"`
	Action            string `json:"action,omitempty"`
}

// DispatchLockEvent is emitted when a manual dispatch freeze lock is acquired or released.
type DispatchLockEvent struct {
	BaseEvent
	LockID      string `json:"lock_id"`
	SupplierID  string `json:"supplier_id"`
	WarehouseID string `json:"warehouse_id,omitempty"`
	FactoryID   string `json:"factory_id,omitempty"`
	LockType    string `json:"lock_type"`
	LockedBy    string `json:"locked_by"`
	EntityType  string `json:"entity_type,omitempty"`
	EntityID    string `json:"entity_id,omitempty"`
	TTLSeconds  int64  `json:"ttl_seconds,omitempty"`
}

// DriverEvent handles driver creation and updates.
type DriverEvent struct {
	BaseEvent
	DriverID     string `json:"driver_id"`
	SupplierID   string `json:"supplier_id"`
	HomeNodeID   string `json:"home_node_id,omitempty"`
	HomeNodeType string `json:"home_node_type,omitempty"`
	Available    bool   `json:"available,omitempty"`
	OnShift      bool   `json:"on_shift,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Note         string `json:"note,omitempty"`
}

// VehicleEvent handles vehicle creation.
type VehicleEvent struct {
	BaseEvent
	VehicleID         string `json:"vehicle_id"`
	SupplierID        string `json:"supplier_id"`
	HomeNodeID        string `json:"home_node_id,omitempty"`
	HomeNodeType      string `json:"home_node_type,omitempty"`
	IsActive          bool   `json:"is_active,omitempty"`
	UnavailableReason string `json:"unavailable_reason,omitempty"`
	UnavailableNote   string `json:"unavailable_note,omitempty"`
}

// OrderEvent handles order creation, status changes, closure, negotiations, and driver edges.
type OrderEvent struct {
	BaseEvent
	OrderID               string  `json:"order_id"`
	SupplierID            string  `json:"supplier_id"`
	RetailerID            string  `json:"retailer_id,omitempty"`
	DriverID              string  `json:"driver_id,omitempty"`
	WarehouseID           string  `json:"warehouse_id,omitempty"`
	VehicleID             string  `json:"vehicle_id,omitempty"`
	LicensePlate          string  `json:"license_plate,omitempty"`
	RouteID               string  `json:"route_id,omitempty"`
	ManifestID            string  `json:"manifest_id,omitempty"`
	Status                string  `json:"status,omitempty"`
	ToDriverID            string  `json:"to_driver_id,omitempty"`
	FromDriverID          string  `json:"from_driver_id,omitempty"`
	OrderSource           string  `json:"order_source,omitempty"`
	ConfirmationStatus    string  `json:"confirmation_status,omitempty"`
	TotalMinor            int64   `json:"total_minor,omitempty"`
	Currency              string  `json:"currency,omitempty"`
	H3Cell                string  `json:"h3_cell,omitempty"`
	Lat                   float64 `json:"lat,omitempty"`
	Lng                   float64 `json:"lng,omitempty"`
	RequestedDeliveryDate string  `json:"requested_delivery_date,omitempty"`
	ReceivingWindowOpen   string  `json:"receiving_window_open,omitempty"`
	ReceivingWindowClose  string  `json:"receiving_window_close,omitempty"`
	LineItems             any     `json:"line_items,omitempty"`
	PreviousStatus        string  `json:"previous_status,omitempty"`
	Reason                string  `json:"reason,omitempty"`
	Action                string  `json:"action,omitempty"`
	ActorRole             string  `json:"actor_role,omitempty"`
	ActorID               string  `json:"actor_id,omitempty"`
	Version               int64   `json:"version,omitempty"`
	ToRouteID             string  `json:"to_route_id,omitempty"`
	FromRouteID           string  `json:"from_route_id,omitempty"`
	PaymentMethod         string  `json:"payment_method,omitempty"`
	ProposedPriceMinor    int64   `json:"proposed_price_minor,omitempty"`
	NegotiationID         string  `json:"negotiation_id,omitempty"`
	ProposalID            string  `json:"proposal_id,omitempty"`
	AttemptID             string  `json:"attempt_id,omitempty"`
	Response              string  `json:"response,omitempty"`
	Resolution            string  `json:"resolution,omitempty"`
	EscalatedTo           string  `json:"escalated_to,omitempty"`
	GPSLat                float64 `json:"gps_lat,omitempty"`
	GPSLng                float64 `json:"gps_lng,omitempty"`
	Message               string  `json:"message,omitempty"`
	// Derived from Status at emit time (see outbox.EmitJSON); never set by callers.
	CustomerStage           string `json:"customer_stage,omitempty"`
	CustomerProgressPercent int    `json:"customer_progress_percent,omitempty"`
}

// WithCustomerStage returns a copy with CustomerStage/CustomerProgressPercent
// derived from Status. Statuses that are not order statuses are left unstamped.
func (e OrderEvent) WithCustomerStage() any {
	if e.CustomerStage != "" {
		return e
	}
	if s, ok := orderstage.Lookup(e.Status); ok {
		e.CustomerStage = string(s)
		e.CustomerProgressPercent = orderstage.ProgressPercent(s)
	}
	return e
}

// OrderConflictReconciledEvent records an offline physical delivery that won
// against an online cancel. The order lands on RECONCILIATION_REQUIRED.
type OrderConflictReconciledEvent struct {
	BaseEvent
	OrderID        string `json:"order_id"`
	SupplierID     string `json:"supplier_id"`
	RetailerID     string `json:"retailer_id,omitempty"`
	DriverID       string `json:"driver_id,omitempty"`
	Resolution     string `json:"resolution,omitempty"`
	PreviousStatus string `json:"previous_status,omitempty"`
	Status         string `json:"status,omitempty"`
	DeliveredAt    string `json:"delivered_at,omitempty"`
	ProofSignature string `json:"proof_signature,omitempty"`
	ProofPhotoURL  string `json:"proof_photo_url,omitempty"`
	DisputeNote    string `json:"dispute_note,omitempty"`
}

// ParentOrderEvent is the multi-supplier ParentOrders rollup lifecycle (B3 M-P0-6 & HARDEN-02).
type ParentOrderEvent struct {
	BaseEvent
	ParentOrderID string   `json:"parent_order_id"`
	RetailerID    string   `json:"retailer_id"`
	Status        string   `json:"status,omitempty"`
	Currency      string   `json:"currency,omitempty"`
	TotalMinor    int64    `json:"total_minor,omitempty"`
	ChildCount    int      `json:"child_count,omitempty"`
	SagaState     string   `json:"saga_state,omitempty"`
	ChildOrderIDs []string `json:"child_order_ids,omitempty"`
}

// SupplierCreditProgramEvent is org-level credit program enable/patch/disable (B4 M-P1-5).
type SupplierCreditProgramEvent struct {
	BaseEvent
	SupplierID     string `json:"supplier_id"`
	ProgramEnabled bool   `json:"program_enabled"`
	Version        int64  `json:"version,omitempty"`
	Action         string `json:"action,omitempty"` // ENABLE | PATCH | DISABLE
	ActorID        string `json:"actor_id,omitempty"`
}

// SupplierCreditTermsEvent is per-retailer payment terms lifecycle (B4 M-P1-5).
type SupplierCreditTermsEvent struct {
	BaseEvent
	SupplierID    string `json:"supplier_id"`
	RetailerID    string `json:"retailer_id"`
	CreditEnabled bool   `json:"credit_enabled"`
	Version       int64  `json:"version,omitempty"`
	Action        string `json:"action,omitempty"` // ENABLE | PATCH | HOLD | UNHOLD | DISABLE
	ActorID       string `json:"actor_id,omitempty"`
}

// ControlTowerEvent is playbook/run lifecycle for supplier control tower (B4 M-P1-4).
type ControlTowerEvent struct {
	BaseEvent
	SupplierID  string `json:"supplier_id"`
	PlaybookID  string `json:"playbook_id,omitempty"`
	RunID       string `json:"run_id,omitempty"`
	ExceptionID string `json:"exception_id,omitempty"`
	Status      string `json:"status,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Action      string `json:"action,omitempty"`
	ActorID     string `json:"actor_id,omitempty"`
}

// Manifest domain plane (G2.D Option B). Dual tables stay intentional:
// FACTORY = FactoryTruckManifests transfer bay; SUPPLIER = SupplierTruckManifests delivery truck.
const (
	ManifestDomainFactory  = "FACTORY"
	ManifestDomainSupplier = "SUPPLIER"
)

// ManifestEvent handles manifest lifecycle events.
// ManifestDomain disambiguates shared event type names across the dual plane.
type ManifestEvent struct {
	BaseEvent
	ManifestID string `json:"manifest_id"`
	// ManifestDomain is FACTORY | SUPPLIER (G2.D). Omitempty keeps older producers valid;
	// new emits always set it.
	ManifestDomain string `json:"manifest_domain,omitempty"`
	SupplierID     string `json:"supplier_id"`
	FactoryID      string `json:"factory_id,omitempty"`
	WarehouseID    string `json:"warehouse_id,omitempty"`
	DriverID       string `json:"driver_id,omitempty"`
	OrderID        string `json:"order_id,omitempty"`
	State          string `json:"state,omitempty"`
	VehicleID      string `json:"vehicle_id,omitempty"`
	TransferCount  int    `json:"transfer_count,omitempty"`
	TotalVolumeVU  int64  `json:"total_volume_vu,omitempty"`
	RouteID        string `json:"route_id,omitempty"`
	Reason         string `json:"reason,omitempty"`
	Action         string `json:"action,omitempty"`
	TransferID     string `json:"transfer_id,omitempty"`
	FromDriverID   string `json:"from_driver_id,omitempty"`
	ToDriverID     string `json:"to_driver_id,omitempty"`
	FromVehicleID  string `json:"from_vehicle_id,omitempty"`
	ToVehicleID    string `json:"to_vehicle_id,omitempty"`
	Depth          int    `json:"depth,omitempty"`
	AttemptCount   int64  `json:"attempt_count,omitempty"`
	Escalated      bool   `json:"escalated,omitempty"`
	StopCount      int64  `json:"stop_count,omitempty"`
	FromManifestID string `json:"from_manifest_id,omitempty"`
	ToManifestID   string `json:"to_manifest_id,omitempty"`
	FromRouteID    string `json:"from_route_id,omitempty"`
	ToRouteID      string `json:"to_route_id,omitempty"`
	OrderCount     int    `json:"order_count,omitempty"`
}

// SplitShipmentEvent is emitted when a warehouse admin approves splitting a
// retailer's oversized consolidated order across multiple trucks. Every
// manifest in the group is linked by SplitGroupID. The driver and retailer
// apps display the same shared route and order list. Payment deduplication:
// the first PAYMENT_COLLECTED event for any order in OrderIDs closes the order
// for all sibling manifests; the system must reject duplicate payment attempts.
type SplitShipmentEvent struct {
	BaseEvent
	SplitGroupID  string   `json:"split_group_id"`
	SupplierID    string   `json:"supplier_id"`
	WarehouseID   string   `json:"warehouse_id,omitempty"`
	RetailerID    string   `json:"retailer_id"`
	SharedRouteID string   `json:"shared_route_id"`
	OrderIDs      []string `json:"order_ids"`
	ManifestIDs   []string `json:"manifest_ids"`
	DriverIDs     []string `json:"driver_ids"`
	TotalVolumeVU float64  `json:"total_volume_vu"`
	// TruckCount is the number of trucks carrying the split shipment.
	TruckCount int `json:"truck_count"`
}

// RouteEvent handles route events.
type RouteEvent struct {
	BaseEvent
	RouteID     string   `json:"route_id"`
	SupplierID  string   `json:"supplier_id"`
	DriverID    string   `json:"driver_id,omitempty"`
	WarehouseID string   `json:"warehouse_id,omitempty"`
	ManifestID  string   `json:"manifest_id,omitempty"`
	VehicleID   string   `json:"vehicle_id,omitempty"`
	OrderIDs    []string `json:"order_ids,omitempty"`
	OrderCount  int      `json:"order_count,omitempty"`
}

// FinanceEvent handles payment/settlement events.
type FinanceEvent struct {
	BaseEvent
	SessionID           string   `json:"session_id,omitempty"`
	OrderID             string   `json:"order_id"`
	SupplierID          string   `json:"supplier_id"`
	LegalName           string   `json:"legal_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	Email               string   `json:"email,omitempty"`
	BankName            string   `json:"bank_name,omitempty"`
	AccountHolder       string   `json:"account_holder,omitempty"`
	SelectedGateways    []string `json:"selected_gateways,omitempty"`
	UserID              string   `json:"user_id,omitempty"`
	SupplierRole        string   `json:"supplier_role,omitempty"`
	AssignedWarehouseID string   `json:"assigned_warehouse_id,omitempty"`

	RetailerID        string `json:"retailer_id,omitempty"`
	AttemptID         string `json:"attempt_id,omitempty"`
	Status            string `json:"status,omitempty"`
	Gateway           string `json:"gateway,omitempty"`
	ExecutionAction   string `json:"execution_action,omitempty"`
	ExecutionMode     string `json:"execution_mode,omitempty"`
	PolicySource      string `json:"policy_source,omitempty"`
	ProviderReference string `json:"provider_reference,omitempty"`
	AmountMinor       int64  `json:"amount_minor,omitempty"`
	CashMinor         int64  `json:"cash_minor,omitempty"`
	CardMinor         int64  `json:"card_minor,omitempty"`
	Currency          string `json:"currency,omitempty"`
	TransactionID     string `json:"transaction_id,omitempty"`
	Source            string `json:"source,omitempty"`
	DriverID          string `json:"driver_id,omitempty"`
}

// AIRecommendationEvent handles AI decisions.
type AIRecommendationEvent struct {
	BaseEvent
	RecommendationID string `json:"recommendation_id"`
	AggregateID      string `json:"aggregate_id,omitempty"`
	AggregateType    string `json:"aggregate_type,omitempty"`
	SupplierID       string `json:"supplier_id"`
	Decision         string `json:"decision,omitempty"`
	Status           string `json:"status,omitempty"`
	DecidedBy        string `json:"decided_by,omitempty"`
	Note             string `json:"note,omitempty"`
}

// DemandSignalEvent is the flywheel broadcast from retailer POS sell-through.
// Suppliers subscribe on TopicDemand (or TopicMain when dual-write is off).
// Deliberately omits POS session/register/tender internals.
type DemandSignalEvent struct {
	BaseEvent
	RetailerID string `json:"retailer_id"`
	LocationID string `json:"location_id,omitempty"`
	SKU        string `json:"sku"`
	Day        string `json:"day"` // YYYY-MM-DD UTC
	// QtyDelta is signed: sale +qty, void −qty.
	QtyDelta int64 `json:"qty_delta"`
	// NetSold is day cumulative net (sold − voided) after this update.
	NetSold    int64  `json:"net_sold"`
	Source     string `json:"source"` // always STORE_POS for this path
	Kind       string `json:"kind"`   // sale | void
	SupplierID string `json:"supplier_id,omitempty"`
}

// PromotionEvent signals supplier promotion create, update, or deactivation.
type PromotionEvent struct {
	BaseEvent
	SupplierID    string   `json:"supplier_id"`
	PromotionID   string   `json:"promotion_id"`
	RetailerScope string   `json:"retailer_scope"`
	RetailerIDs   []string `json:"retailer_ids,omitempty"`
	Action        string   `json:"action"`
}

// RetailerPriceOverrideEvent signals per-retailer absolute price create or deactivate.
type RetailerPriceOverrideEvent struct {
	BaseEvent
	OverrideID string `json:"override_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id"`
	ProductID  string `json:"product_id"`
	PriceMinor int64  `json:"price_minor"`
	Action     string `json:"action"`
	SetBy      string `json:"set_by"`
	SetByRole  string `json:"set_by_role"`
}

// InventoryImportEvent signals supplier bulk-import session lifecycle transitions.
type InventoryImportEvent struct {
	BaseEvent
	SessionID         string `json:"session_id"`
	SupplierID        string `json:"supplier_id"`
	GCSPath           string `json:"gcs_path,omitempty"`
	Status            string `json:"status,omitempty"`
	SuggestedMappings int    `json:"suggested_mappings,omitempty"`
}

// SyncEvent handles catalog and inventory syncing.
type SyncEvent struct {
	BaseEvent
	SupplierID          string   `json:"supplier_id"`
	LegalName           string   `json:"legal_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	Email               string   `json:"email,omitempty"`
	BankName            string   `json:"bank_name,omitempty"`
	AccountHolder       string   `json:"account_holder,omitempty"`
	SelectedGateways    []string `json:"selected_gateways,omitempty"`
	UserID              string   `json:"user_id,omitempty"`
	SupplierRole        string   `json:"supplier_role,omitempty"`
	AssignedWarehouseID string   `json:"assigned_warehouse_id,omitempty"`

	RetailerID string `json:"retailer_id,omitempty"`
}

// CommandEvent handles distributed commands.
type CommandEvent struct {
	BaseEvent
	CommandID           string   `json:"command_id"`
	SupplierID          string   `json:"supplier_id"`
	LegalName           string   `json:"legal_name,omitempty"`
	ContactName         string   `json:"contact_name,omitempty"`
	Email               string   `json:"email,omitempty"`
	BankName            string   `json:"bank_name,omitempty"`
	AccountHolder       string   `json:"account_holder,omitempty"`
	SelectedGateways    []string `json:"selected_gateways,omitempty"`
	UserID              string   `json:"user_id,omitempty"`
	SupplierRole        string   `json:"supplier_role,omitempty"`
	AssignedWarehouseID string   `json:"assigned_warehouse_id,omitempty"`
}

// PlatformEvent handles system-wide events.
type PlatformEvent struct {
	BaseEvent
	SystemID string `json:"system_id,omitempty"`
}

// SupplyRequestEvent handles warehouse supply requests.
type SupplyRequestEvent struct {
	BaseEvent
	RequestID   string `json:"request_id"`
	WarehouseID string `json:"warehouse_id,omitempty"`
	FactoryID   string `json:"factory_id,omitempty"`
	Status      string `json:"status,omitempty"`
	ItemCount   int    `json:"item_count,omitempty"`
}

// SystemEvent handles system-wide locks and updates.
type SystemEvent struct {
	BaseEvent
	SystemID string `json:"system_id,omitempty"`
	Version  string `json:"version,omitempty"`
	LockName string `json:"lock_name,omitempty"`
}

// WarehouseTransferEvent handles inventory transfers.
type WarehouseTransferEvent struct {
	BaseEvent
	TransferID    string `json:"transfer_id"`
	FromWarehouse string `json:"from_warehouse,omitempty"`
	ToWarehouse   string `json:"to_warehouse,omitempty"`
	Status        string `json:"status,omitempty"`
	FactoryID     string `json:"factory_id,omitempty"`
	SupplierID    string `json:"supplier_id,omitempty"`
	State         string `json:"state,omitempty"`
}

// DeliverySessionEvent handles driver delivery sessions.
type DeliverySessionEvent struct {
	BaseEvent
	SessionID string `json:"session_id"`
	DriverID  string `json:"driver_id"`
	Status    string `json:"status,omitempty"`
}

// ShopClosedEvent handles shop closure reporting.
type ShopClosedEvent struct {
	BaseEvent
	ReportID   string `json:"report_id"`
	RetailerID string `json:"retailer_id"`
	DriverID   string `json:"driver_id"`
	Status     string `json:"status,omitempty"`
}

// NegotiationEvent handles order price negotiations.
type NegotiationEvent struct {
	BaseEvent
	NegotiationID string `json:"negotiation_id"`
	OrderID       string `json:"order_id"`
	RetailerID    string `json:"retailer_id"`
	Status        string `json:"status,omitempty"`
}

// ReturnEvent handles product returns.
type ReturnEvent struct {
	BaseEvent
	ReturnID   string `json:"return_id"`
	OrderID    string `json:"order_id"`
	RetailerID string `json:"retailer_id"`
	Status     string `json:"status,omitempty"`
}

// ShopClosedBypassOffloadEvent handles shop-closed bypass offload completion.
type ShopClosedBypassOffloadEvent struct {
	BaseEvent
	OrderID    string `json:"order_id"`
	DriverID   string `json:"driver_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id"`
	Status     string `json:"status,omitempty"`
}

// CreditDeliveryEvent handles credit-delivery marking and resolution.
type CreditDeliveryEvent struct {
	BaseEvent
	OrderID    string `json:"order_id"`
	DriverID   string `json:"driver_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id"`
	Status     string `json:"status,omitempty"`
	// PhotoProofURL is required evidence for credit leave (PoD).
	PhotoProofURL string `json:"photo_proof_url,omitempty"`
	// SignatureURL is optional handwritten acknowledgment captured at handoff.
	SignatureURL string `json:"signature_url,omitempty"`
}

// PreOrderEvent handles pre-order lifecycle.
type PreOrderEvent struct {
	BaseEvent
	PreOrderID string `json:"pre_order_id"`
	RetailerID string `json:"retailer_id"`
	SupplierID string `json:"supplier_id"`
	Status     string `json:"status,omitempty"`
}

// PlanningEvent covers PX90 planning-brain surfaces (MEIO, control tower, demand).
type PlanningEvent struct {
	BaseEvent
	SupplierID     string  `json:"supplier_id"`
	WarehouseID    string  `json:"warehouse_id,omitempty"`
	FactoryID      string  `json:"factory_id,omitempty"`
	InsightID      string  `json:"insight_id,omitempty"`
	ProductID      string  `json:"product_id,omitempty"`
	OverrideID     string  `json:"override_id,omitempty"`
	Action         string  `json:"action,omitempty"`
	Polygon        string  `json:"polygon_geojson,omitempty"`
	TTLSeconds     int64   `json:"ttl_seconds,omitempty"`
	BaselineQty    int64   `json:"baseline_qty,omitempty"`
	LowUnits       int64   `json:"low_units,omitempty"`
	HighUnits      int64   `json:"high_units,omitempty"`
	Confidence     float64 `json:"confidence,omitempty"`
	ConfidencePct  int64   `json:"confidence_pct,omitempty"`
	BaselineSource string  `json:"baseline_source,omitempty"`
	BlockedReason  string  `json:"blocked_reason,omitempty"`
	NetworkNodes   int     `json:"network_nodes,omitempty"`
	Transfers      int     `json:"transfer_recommendations,omitempty"`
	SignalID       string  `json:"signal_id,omitempty"`
	SimulationID   string  `json:"simulation_id,omitempty"`
	ScenarioID     string  `json:"scenario_id,omitempty"`
	Version        int64   `json:"version,omitempty"`
	PublishedBy    string  `json:"published_by,omitempty"`
}

// @Sync(ProductEvent)
// ProductEvent handles product lifecycle and handling classification updates.
type ProductEvent struct {
	BaseEvent
	ProductID          string `json:"product_id"`
	SupplierID         string `json:"supplier_id"`
	HandlingClass      string `json:"handling_class,omitempty"`
	IsHazardous        bool   `json:"is_hazardous,omitempty"`
	IsPerishable       bool   `json:"is_perishable,omitempty"`
	Name               string `json:"name,omitempty"`
	CategoryID         string `json:"category_id,omitempty"`
	PriceMinor         int64  `json:"price_minor,omitempty"`
	PreviousPriceMinor int64  `json:"previous_price_minor,omitempty"`
	Currency           string `json:"currency,omitempty"`
	StockQuantity      int64  `json:"stock_quantity,omitempty"`
	Unit               string `json:"unit,omitempty"`
	Barcode            string `json:"barcode,omitempty"`
	IsActive           bool   `json:"is_active,omitempty"`
	Version            int64  `json:"version,omitempty"`
}

// @Sync(CategoryEvent)
// CategoryEvent handles product category mutations.
type CategoryEvent struct {
	BaseEvent
	CategoryID       string `json:"category_id"`
	SupplierID       string `json:"supplier_id"`
	Name             string `json:"name"`
	ParentCategoryID string `json:"parent_category_id,omitempty"`
	IconKey          string `json:"icon_key,omitempty"`
	SortOrder        int64  `json:"sort_order,omitempty"`
}

// @Sync(ConditionEvent)
// ConditionEvent handles structured order condition reports.
type ConditionEvent struct {
	BaseEvent
	ReportID      string   `json:"report_id"`
	OrderID       string   `json:"order_id"`
	SupplierID    string   `json:"supplier_id,omitempty"`
	RetailerID    string   `json:"retailer_id,omitempty"`
	ReporterID    string   `json:"reporter_id"`
	ReporterRole  string   `json:"reporter_role"`
	ConditionType string   `json:"condition_type"`
	SKU           string   `json:"sku,omitempty"`
	Quantity      int64    `json:"quantity,omitempty"`
	GCSPaths      []string `json:"gcs_paths,omitempty"`
	Notes         string   `json:"notes,omitempty"`
}

// @Sync(LogisticsException)
// LogisticsException is the payload for CLAIM_* / OS&D / reverse-logistics events.
type LogisticsException struct {
	BaseEvent
	ClaimID        string   `json:"claim_id,omitempty"`
	OrderID        string   `json:"order_id,omitempty"`
	SupplierID     string   `json:"supplier_id,omitempty"`
	RetailerID     string   `json:"retailer_id,omitempty"`
	DriverID       string   `json:"driver_id,omitempty"`
	WarehouseID    string   `json:"warehouse_id,omitempty"`
	ClaimType      string   `json:"claim_type,omitempty"`
	Status         string   `json:"status,omitempty"`
	Source         string   `json:"source,omitempty"`
	AmountMinor    int64    `json:"amount_minor,omitempty"`
	Currency       string   `json:"currency,omitempty"`
	Note           string   `json:"note,omitempty"`
	ResolutionNote string   `json:"resolution_note,omitempty"`
	SettlementMode string   `json:"settlement_mode,omitempty"`
	ChargebackID   string   `json:"chargeback_id,omitempty"`
	PhotoURLs      []string `json:"photo_urls,omitempty"`
}

// @Sync(CreditEvent)
// CreditProfileEvent handles retailer credit profile changes.
type CreditProfileEvent struct {
	BaseEvent
	ProfileID        string `json:"profile_id"`
	RetailerID       string `json:"retailer_id"`
	SupplierID       string `json:"supplier_id"`
	CreditLimitMinor int64  `json:"credit_limit_minor"`
	CurrentBalance   int64  `json:"current_balance"`
	RiskTier         string `json:"risk_tier"`
	Delinquent       bool   `json:"delinquent"`
	Reason           string `json:"reason,omitempty"`
}

// @Sync(CreditEvent)
// CreditLimitEvent signals a credit-limit breach at order time.
type CreditLimitEvent struct {
	BaseEvent
	OrderID          string `json:"order_id"`
	RetailerID       string `json:"retailer_id"`
	SupplierID       string `json:"supplier_id"`
	RequestedAmount  int64  `json:"requested_amount"`
	CreditLimitMinor int64  `json:"credit_limit_minor"`
	CurrentBalance   int64  `json:"current_balance"`
}

// RescueEvent handles mid-route rescue lifecycle between driver and warehouse.
type RescueEvent struct {
	BaseEvent
	RescueID       string `json:"rescue_id,omitempty"`
	BrokenDriverID string `json:"broken_driver_id"`
	RescueDriverID string `json:"rescue_driver_id,omitempty"`
	Status         string `json:"status,omitempty"` // REQUESTED, PROPOSED, ACCEPTED, REJECTED
	WarehouseID    string `json:"warehouse_id,omitempty"`
	SupplierID     string `json:"supplier_id,omitempty"`
}

// FiscalReceiptEvent is the ADR-009 OFD attempt lifecycle payload (integer Tiyin).
type FiscalReceiptEvent struct {
	BaseEvent
	OrderID         string `json:"order_id"`
	AttemptID       string `json:"attempt_id"`
	SupplierID      string `json:"supplier_id"`
	RetailerID      string `json:"retailer_id,omitempty"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	PaymentMethod   string `json:"payment_method,omitempty"`
	Provider        string `json:"provider,omitempty"`
	Status          string `json:"status,omitempty"`
	FiscalReceiptID string `json:"fiscal_receipt_id,omitempty"`
	FiscalQR        string `json:"fiscal_qr,omitempty"`
	ErrorCode       string `json:"error_code,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
	TraceID         string `json:"trace_id,omitempty"`
}

// OrderForceCompletedEvent is emitted when ADMIN/WAREHOUSE_ADMIN force-completes past fiscal failure.
type OrderForceCompletedEvent struct {
	BaseEvent
	OrderID    string `json:"order_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id,omitempty"`
	ReasonCode string `json:"reason_code"`
	ActorID    string `json:"actor_id"`
	TraceID    string `json:"trace_id,omitempty"`
}

// BuyerAcceptanceEvent tracks Soliq EHF buyer clearance. ADR-009 still marks the
// order COMPLETED on OFD submit success; this parallel track gates final ledger
// close / reverse-settlement (credit note on REJECT).
type BuyerAcceptanceEvent struct {
	BaseEvent
	OrderID    string `json:"order_id"`
	SupplierID string `json:"supplier_id"`
	RetailerID string `json:"retailer_id,omitempty"`
	EhfID      string `json:"ehf_id,omitempty"`
	Status     string `json:"status"` // PENDING | ACCEPTED | REJECTED | EXPIRED
	Deadline   string `json:"deadline,omitempty"`
}

// CashVarianceEvent records cash shortfall or overage at collection (integer Tiyin).
type CashVarianceEvent struct {
	BaseEvent
	OrderID        string `json:"order_id"`
	SupplierID     string `json:"supplier_id"`
	RetailerID     string `json:"retailer_id,omitempty"`
	DriverID       string `json:"driver_id,omitempty"`
	ExpectedMinor  int64  `json:"expected_minor"`
	ReceivedMinor  int64  `json:"received_minor"`
	ShortfallMinor int64  `json:"shortfall_minor,omitempty"`
	OverageMinor   int64  `json:"overage_minor,omitempty"`
	Currency       string `json:"currency"`
	Note           string `json:"note,omitempty"`
	TraceID        string `json:"trace_id,omitempty"`
}

// MigrationEvent signals universal smart-ingestion project/session lifecycle transitions.
type MigrationEvent struct {
	BaseEvent
	ProjectID       string `json:"project_id,omitempty"`
	BundleID        string `json:"bundle_id,omitempty"`
	SessionID       string `json:"session_id,omitempty"`
	SupplierID      string `json:"supplier_id"`
	Domain          string `json:"domain,omitempty"`
	Status          string `json:"status,omitempty"`
	ReportID        string `json:"report_id,omitempty"`
	ProcessedRows   int64  `json:"processed_rows,omitempty"`
	TotalRows       int64  `json:"total_rows,omitempty"`
	ImportedCount   int64  `json:"imported_count,omitempty"`
	QuarantineCount int64  `json:"quarantine_count,omitempty"`
	ErrorCode       string `json:"error_code,omitempty"`
	ErrorMessage    string `json:"error_message,omitempty"`
}

// JoinRequestEvent signals duplicate-INN team join request lifecycle transitions.
type JoinRequestEvent struct {
	BaseEvent
	RequestID        string `json:"request_id"`
	TargetTenantType string `json:"target_tenant_type"`
	TargetTenantID   string `json:"target_tenant_id"`
	SupplierID       string `json:"supplier_id,omitempty"`
	RetailerID       string `json:"retailer_id,omitempty"`
	NormalizedINN    string `json:"normalized_inn,omitempty"`
	RequesterPhone   string `json:"requester_phone,omitempty"`
	RequesterName    string `json:"requester_name,omitempty"`
	RoleSought       string `json:"role_sought,omitempty"`
	Status           string `json:"status,omitempty"`
	DecidedBy        string `json:"decided_by,omitempty"`
	DecisionReason   string `json:"decision_reason,omitempty"`
}

// ReleasedLotQuantity identifies one stock lot and the quantity released back to available stock.
type ReleasedLotQuantity struct {
	LotID    string `json:"lot_id"`
	Quantity int64  `json:"quantity"`
}

// @Sync(InventoryLotReleasedEvent)
// InventoryLotReleasedEvent is emitted atomically when an order's StockLots reservations are released.
type InventoryLotReleasedEvent struct {
	BaseEvent
	OrderID      string                `json:"order_id"`
	SupplierID   string                `json:"supplier_id"`
	WarehouseID  string                `json:"warehouse_id"`
	ReleasedLots []ReleasedLotQuantity `json:"released_lots"`
}

// @Sync(GLEntryPostedEvent)
// GLEntryPostedEvent is emitted atomically in-txn when a balanced GL journal entry is posted.
type GLEntryPostedEvent struct {
	BaseEvent
	EntryID          string `json:"entry_id"`
	EntityID         string `json:"entity_id"`
	PeriodID         string `json:"period_id"`
	SourceType       string `json:"source_type"`
	SourceID         string `json:"source_id"`
	Currency         string `json:"currency"`
	TotalDebitMinor  int64  `json:"total_debit_minor"`
	TotalCreditMinor int64  `json:"total_credit_minor"`
	Hash             string `json:"hash"`
	ReversesEntryID  string `json:"reverses_entry_id,omitempty"`
}

// @Sync(GLEntryReversedEvent)
// GLEntryReversedEvent is emitted atomically in-txn when a balanced GL reversal entry is posted.
type GLEntryReversedEvent struct {
	BaseEvent
	EntryID          string `json:"entry_id"`
	EntityID         string `json:"entity_id"`
	PeriodID         string `json:"period_id"`
	SourceType       string `json:"source_type"`
	SourceID         string `json:"source_id"`
	Currency         string `json:"currency"`
	TotalDebitMinor  int64  `json:"total_debit_minor"`
	TotalCreditMinor int64  `json:"total_credit_minor"`
	Hash             string `json:"hash"`
	ReversesEntryID  string `json:"reverses_entry_id"`
}

// @Sync(ReconRunCompletedEvent)
// ReconRunCompletedEvent is emitted atomically in-txn when a four-way reconciliation run finishes.
type ReconRunCompletedEvent struct {
	BaseEvent
	RunID         string `json:"run_id"`
	EntityID      string `json:"entity_id"`
	PeriodID      string `json:"period_id"`
	Mode          string `json:"mode"`
	Status        string `json:"status"`
	BreakCount    int64  `json:"break_count"`
	ExposureMinor int64  `json:"exposure_minor"`
	Currency      string `json:"currency,omitempty"`
	TriggeredBy   string `json:"triggered_by"`
	StartedAt     string `json:"started_at"`
	CompletedAt   string `json:"completed_at,omitempty"`
}

// @Sync(ReconBreakOpenedEvent)
// ReconBreakOpenedEvent is emitted atomically in-txn when a reconciliation break is opened.
type ReconBreakOpenedEvent struct {
	BaseEvent
	BreakID                 string `json:"break_id"`
	RunID                   string `json:"run_id"`
	EntityID                string `json:"entity_id"`
	Dimension               string `json:"dimension"`
	ObjectType              string `json:"object_type"`
	ObjectID                string `json:"object_id"`
	ExpectedMinor           int64  `json:"expected_minor"`
	ActualMinor             int64  `json:"actual_minor"`
	DeltaMinor              int64  `json:"delta_minor"`
	Currency                string `json:"currency"`
	Severity                string `json:"severity"`
	Status                  string `json:"status"`
	OwnerID                 string `json:"owner_id,omitempty"`
	SlaDueAt                string `json:"sla_due_at,omitempty"`
	ControlTowerExceptionID string `json:"control_tower_exception_id,omitempty"`
}

// @Sync(ReconBreakResolvedEvent)
// ReconBreakResolvedEvent is emitted atomically in-txn when a reconciliation break is resolved or waived.
type ReconBreakResolvedEvent struct {
	BaseEvent
	BreakID                 string `json:"break_id"`
	RunID                   string `json:"run_id"`
	EntityID                string `json:"entity_id"`
	Dimension               string `json:"dimension"`
	ObjectType              string `json:"object_type"`
	ObjectID                string `json:"object_id"`
	DeltaMinor              int64  `json:"delta_minor"`
	Currency                string `json:"currency"`
	Status                  string `json:"status"`
	ResolvedBy              string `json:"resolved_by"`
	ResolutionNote          string `json:"resolution_note,omitempty"`
	ControlTowerExceptionID string `json:"control_tower_exception_id,omitempty"`
}

// @Sync(AuditAnchorWrittenEvent)
// AuditAnchorWrittenEvent is emitted when a daily audit hash-chain anchor is committed and written to WORM storage.
type AuditAnchorWrittenEvent struct {
	BaseEvent
	TenantID    string `json:"tenant_id"`
	DayBucket   string `json:"day_bucket"`
	HeadEventID string `json:"head_event_id"`
	EventCount  int64  `json:"event_count"`
	RootHash    string `json:"root_hash"`
	WormURI     string `json:"worm_uri,omitempty"`
	SignerKeyID string `json:"signer_key_id,omitempty"`
	AnchoredBy  string `json:"anchored_by"`
	AnchoredAt  string `json:"anchored_at"`
}

// @Sync(IAMRoleChangedEvent)
// IAMRoleChangedEvent is emitted when a supplier user role assignment is granted, updated, or revoked.
type IAMRoleChangedEvent struct {
	BaseEvent
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
	RoleName   string `json:"role_name"`
	Action     string `json:"action"`
	AssignedBy string `json:"assigned_by"`
	TraceID    string `json:"trace_id,omitempty"`
}

// @Sync(SodViolationDetectedEvent)
// SodViolationDetectedEvent is emitted when a Segregation of Duties (four-eyes / role conflict) violation is blocked.
type SodViolationDetectedEvent struct {
	BaseEvent
	TenantID    string `json:"tenant_id"`
	ViolationID string `json:"violation_id"`
	ActorID     string `json:"actor_id"`
	Action      string `json:"action"`
	ObjectType  string `json:"object_type,omitempty"`
	ObjectID    string `json:"object_id,omitempty"`
	MakerID     string `json:"maker_id,omitempty"`
	RuleCode    string `json:"rule_code"`
	RoutePath   string `json:"route_path,omitempty"`
	TraceID     string `json:"trace_id,omitempty"`
}

// @Sync(OntologyObjectChangedEvent)
// OntologyObjectChangedEvent is emitted when an ontology object is created, updated, or rebuilt in Spanner.
type OntologyObjectChangedEvent struct {
	BaseEvent
	CellID           string   `json:"cell_id"`
	ObjectType       string   `json:"object_type"`
	ObjectID         string   `json:"object_id"`
	SupplierID       string   `json:"supplier_id"`
	Version          int64    `json:"version"`
	ChangeKind       string   `json:"change_kind"`
	SourceEventType  string   `json:"source_event_type,omitempty"`
	SourceEventID    string   `json:"source_event_id,omitempty"`
	SecurityMarkings []string `json:"security_markings,omitempty"`
	UpdatedAt        string   `json:"updated_at"`
}

// @Sync(PeriodEvent)
// PeriodEvent is emitted when an accounting period is soft-closed, hard-closed, or reopened.
type PeriodEvent struct {
	BaseEvent
	EntityID     string `json:"entity_id"`
	PeriodID     string `json:"period_id"`
	Status       string `json:"status"`
	Timezone     string `json:"timezone,omitempty"`
	ActorID      string `json:"actor_id"`
	Reason       string `json:"reason,omitempty"`
	SnapshotURI  string `json:"snapshot_uri,omitempty"`
	SnapshotHash string `json:"snapshot_hash,omitempty"`
	ApprovalID   string `json:"approval_id,omitempty"`
	OccurredAt   string `json:"occurred_at,omitempty"`
}

// @Sync(ApprovalRequestedEvent)
// ApprovalRequestedEvent is emitted when a CEL-matched action creates an approval request.
type ApprovalRequestedEvent struct {
	BaseEvent
	TenantID          string `json:"tenant_id"`
	RequestID         string `json:"request_id"`
	PolicyID          string `json:"policy_id,omitempty"`
	PolicyVersion     int64  `json:"policy_version"`
	ObjectType        string `json:"object_type"`
	ObjectID          string `json:"object_id"`
	Action            string `json:"action"`
	AmountMinor       int64  `json:"amount_minor"`
	RequestedBy       string `json:"requested_by"`
	ActorType         string `json:"actor_type"`
	PayloadHash       string `json:"payload_hash"`
	Status            string `json:"status,omitempty"`
	RequiredApprovers int64  `json:"required_approvers,omitempty"`
	BreakGlass        bool   `json:"break_glass"`
	ExpiresAt         string `json:"expires_at,omitempty"`
}

// @Sync(ApprovalDecidedEvent)
// ApprovalDecidedEvent is emitted when an approver approves or rejects a pending approval request.
type ApprovalDecidedEvent struct {
	BaseEvent
	TenantID      string `json:"tenant_id"`
	RequestID     string `json:"request_id"`
	DecisionID    string `json:"decision_id"`
	StepIndex     int64  `json:"step_index"`
	ApproverID    string `json:"approver_id"`
	DelegatedFrom string `json:"delegated_from,omitempty"`
	Decision      string `json:"decision"`
	Status        string `json:"status"`
	MfaVerified   bool   `json:"mfa_verified"`
	BreakGlass    bool   `json:"break_glass"`
	Comment       string `json:"comment,omitempty"`
}

// @Sync(InventoryReceivedEvent)
// InventoryReceivedEvent is emitted when a valued inventory receipt creates a cost layer and GL posting.
type InventoryReceivedEvent struct {
	BaseEvent
	EntityID       string `json:"entity_id"`
	WarehouseID    string `json:"warehouse_id"`
	ProductID      string `json:"product_id"`
	LotID          string `json:"lot_id,omitempty"`
	LayerID        string `json:"layer_id"`
	MovementID     string `json:"movement_id"`
	ReferenceType  string `json:"reference_type,omitempty"`
	ReferenceID    string `json:"reference_id,omitempty"`
	Quantity       int64  `json:"quantity"`
	UnitCostMinor  int64  `json:"unit_cost_minor"`
	TotalCostMinor int64  `json:"total_cost_minor"`
	Currency       string `json:"currency"`
	GlEntryID      string `json:"gl_entry_id,omitempty"`
	ExpiryDate     string `json:"expiry_date,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

// @Sync(InventoryIssuedEvent)
// InventoryIssuedEvent is emitted when inventory is issued or adjusted out using FEFO/moving-average costing.
type InventoryIssuedEvent struct {
	BaseEvent
	EntityID       string `json:"entity_id"`
	WarehouseID    string `json:"warehouse_id"`
	ProductID      string `json:"product_id"`
	LotID          string `json:"lot_id,omitempty"`
	LayerID        string `json:"layer_id,omitempty"`
	MovementID     string `json:"movement_id,omitempty"`
	ReferenceType  string `json:"reference_type"`
	ReferenceID    string `json:"reference_id"`
	OrderID        string `json:"order_id,omitempty"`
	Quantity       int64  `json:"quantity"`
	UnitCostMinor  int64  `json:"unit_cost_minor,omitempty"`
	TotalCostMinor int64  `json:"total_cost_minor"`
	Currency       string `json:"currency"`
	GlEntryID      string `json:"gl_entry_id,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

// @Sync(InventoryExpiryRiskEvent)
// InventoryExpiryRiskEvent is emitted when valued cost layers approach expiration with remaining stock value.
type InventoryExpiryRiskEvent struct {
	BaseEvent
	EntityID         string `json:"entity_id"`
	WarehouseID      string `json:"warehouse_id"`
	ProductID        string `json:"product_id"`
	LotID            string `json:"lot_id,omitempty"`
	LayerID          string `json:"layer_id"`
	QtyRemaining     int64  `json:"qty_remaining"`
	UnitCostMinor    int64  `json:"unit_cost_minor,omitempty"`
	ValueAtRiskMinor int64  `json:"value_at_risk_minor,omitempty"`
	Currency         string `json:"currency,omitempty"`
	ExpiryDate       string `json:"expiry_date"`
	DaysRemaining    int64  `json:"days_remaining,omitempty"`
	DaysToExpiry     int64  `json:"days_to_expiry,omitempty"`
	OccurredAt       string `json:"occurred_at,omitempty"`
}

// @Sync(EsfDocumentEvent)
// EsfDocumentEvent is emitted on Soliq ESF document lifecycle transitions.
type EsfDocumentEvent struct {
	BaseEvent
	EntityID        string `json:"entity_id"`
	EsfID           string `json:"esf_id"`
	Direction       string `json:"direction"`
	OrderID         string `json:"order_id,omitempty"`
	PoID            string `json:"po_id,omitempty"`
	CounterpartyTIN string `json:"counterparty_tin,omitempty"`
	TotalMinor      int64  `json:"total_minor"`
	VatMinor        int64  `json:"vat_minor"`
	VatRateBps      int64  `json:"vat_rate_bps"`
	Status          string `json:"status"`
	Operator        string `json:"operator"`
	OperatorRef     string `json:"operator_ref,omitempty"`
	PeriodID        string `json:"period_id"`
	PayloadHash     string `json:"payload_hash,omitempty"`
	ActorID         string `json:"actor_id,omitempty"`
	Reason          string `json:"reason,omitempty"`
	OccurredAt      string `json:"occurred_at,omitempty"`
}

// @Sync(ForecastRunCompletedEvent)
// ForecastRunCompletedEvent is emitted when a Forecast v2 champion/challenger run finishes.
type ForecastRunCompletedEvent struct {
	BaseEvent
	EntityID           string `json:"entity_id"`
	RunID              string `json:"run_id"`
	SupplierID         string `json:"supplier_id"`
	LocationID         string `json:"location_id,omitempty"`
	HorizonDays        int64  `json:"horizon_days"`
	Status             string `json:"status"`
	ChampionModel      string `json:"champion_model"`
	ChallengerModel    string `json:"challenger_model"`
	WinnerModel        string `json:"winner_model"`
	ChampionWapeBps    int64  `json:"champion_wape_bps"`
	ChallengerWapeBps  int64  `json:"challenger_wape_bps"`
	WapeImprovementBps int64  `json:"wape_improvement_bps"`
	SkuCount           int64  `json:"sku_count"`
	LocationCount      int64  `json:"location_count"`
	TriggeredBy        string `json:"triggered_by,omitempty"`
	CompletedAt        string `json:"completed_at,omitempty"`
}

// @Sync(CreditScoreComputedEvent)
// CreditScoreComputedEvent is emitted when a relationship or network credit score is computed and archived.
type CreditScoreComputedEvent struct {
	BaseEvent
	ScoreID             string `json:"score_id"`
	RetailerID          string `json:"retailer_id"`
	SupplierID          string `json:"supplier_id,omitempty"`
	Scope               string `json:"scope"`
	ConsentVerified     bool   `json:"consent_verified"`
	ModelVersion        string `json:"model_version"`
	ScorerSource        string `json:"scorer_source"`
	Score               int64  `json:"score"`
	PdBps               int64  `json:"pd_bps"`
	RiskTier            string `json:"risk_tier"`
	SuggestedLimitMinor int64  `json:"suggested_limit_minor"`
	ReasonCodesJSON     string `json:"reason_codes_json,omitempty"`
	FeaturesHash        string `json:"features_hash"`
	ComputedAt          string `json:"computed_at"`
}

// @Sync(CreditLimitChangedEvent)
// CreditLimitChangedEvent is emitted when a governed credit limit change is applied.
type CreditLimitChangedEvent struct {
	BaseEvent
	RetailerID         string `json:"retailer_id"`
	SupplierID         string `json:"supplier_id"`
	PreviousLimitMinor int64  `json:"previous_limit_minor"`
	NewLimitMinor      int64  `json:"new_limit_minor"`
	ApprovalID         string `json:"approval_id,omitempty"`
	ActorID            string `json:"actor_id"`
	Reason             string `json:"reason,omitempty"`
	ReasonCodesJSON    string `json:"reason_codes_json,omitempty"`
	OccurredAt         string `json:"occurred_at,omitempty"`
}

// @Sync(CreditHoldEvent)
// CreditHoldEvent is emitted when a credit hold or freeze is applied or released.
type CreditHoldEvent struct {
	BaseEvent
	RetailerID       string `json:"retailer_id"`
	SupplierID       string `json:"supplier_id"`
	Action           string `json:"action"`
	Status           string `json:"status"`
	PrepayOnlyLocked bool   `json:"prepay_only_locked"`
	Score            int64  `json:"score,omitempty"`
	RiskTier         string `json:"risk_tier,omitempty"`
	Reason           string `json:"reason,omitempty"`
	ActorID          string `json:"actor_id,omitempty"`
	OccurredAt       string `json:"occurred_at,omitempty"`
}

// @Sync(ProposalEvent)
// ProposalEvent is emitted on L1 agent proposal creation, approval/rejection decision, and execution.
type ProposalEvent struct {
	BaseEvent
	TenantID            string `json:"tenant_id"`
	ProposalID          string `json:"proposal_id"`
	AgentID             string `json:"agent_id"`
	ProposalType        string `json:"proposal_type"`
	TargetObjectType    string `json:"target_object_type"`
	TargetObjectID      string `json:"target_object_id"`
	Status              string `json:"status"`
	PayloadHash         string `json:"payload_hash"`
	ApprovalRequestID   string `json:"approval_request_id,omitempty"`
	ActorID             string `json:"actor_id,omitempty"`
	ActorType           string `json:"actor_type,omitempty"`
	RoiEventID          string `json:"roi_event_id,omitempty"`
	ExpectedImpactMinor int64  `json:"expected_impact_minor,omitempty"`
	Currency            string `json:"currency,omitempty"`
	ConfidenceBps       int64  `json:"confidence_bps,omitempty"`
	Decision            string `json:"decision,omitempty"`
	Reason              string `json:"reason,omitempty"`
	OccurredAt          string `json:"occurred_at,omitempty"`
}

// @Sync(InventoryRebalanceProposedEvent)
// InventoryRebalanceProposedEvent is emitted when the expiry/rebalance agent proposes stock movement or discounting.
type InventoryRebalanceProposedEvent struct {
	BaseEvent
	TenantID          string `json:"tenant_id"`
	ProposalID        string `json:"proposal_id"`
	SourceWarehouseID string `json:"source_warehouse_id"`
	TargetWarehouseID string `json:"target_warehouse_id,omitempty"`
	ProductID         string `json:"product_id"`
	LotID             string `json:"lot_id,omitempty"`
	LayerID           string `json:"layer_id,omitempty"`
	Qty               int64  `json:"qty"`
	ValueAtRiskMinor  int64  `json:"value_at_risk_minor"`
	Currency          string `json:"currency"`
	ExpiryDate        string `json:"expiry_date,omitempty"`
	ActionMode        string `json:"action_mode,omitempty"`
	ConfidenceBps     int64  `json:"confidence_bps,omitempty"`
	ApprovalRequestID string `json:"approval_request_id,omitempty"`
	OccurredAt        string `json:"occurred_at,omitempty"`
}

// @Sync(RoiEventRecordedEvent)
// RoiEventRecordedEvent is emitted when an attributed ROI savings or holdout event is recorded.
type RoiEventRecordedEvent struct {
	BaseEvent
	TenantID        string `json:"tenant_id"`
	EventID         string `json:"event_id"`
	Category        string `json:"category"`
	SavingsMinor    int64  `json:"savings_minor"`
	Currency        string `json:"currency"`
	Methodology     string `json:"methodology,omitempty"`
	ConfidenceBps   int64  `json:"confidence_bps"`
	IsEstimate      bool   `json:"is_estimate"`
	HoldoutControl  bool   `json:"holdout_control"`
	ProposalID      string `json:"proposal_id,omitempty"`
	EvidenceRefType string `json:"evidence_ref_type,omitempty"`
	EvidenceRefID   string `json:"evidence_ref_id,omitempty"`
	OccurredAt      string `json:"occurred_at,omitempty"`
}

// @Sync(ConnectorSyncEvent)
// ConnectorSyncEvent is emitted when an external connector sync or shadow-compare run completes or fails.
type ConnectorSyncEvent struct {
	BaseEvent
	TenantID             string `json:"tenant_id"`
	RunID                string `json:"run_id"`
	Provider             string `json:"provider"`
	AccountLabel         string `json:"account_label,omitempty"`
	Direction            string `json:"direction"`
	StreamName           string `json:"stream_name"`
	Status               string `json:"status"`
	RecordsProcessed     int64  `json:"records_processed"`
	MismatchCount        int64  `json:"mismatch_count"`
	MissingExternalCount int64  `json:"missing_external_count,omitempty"`
	MissingInternalCount int64  `json:"missing_internal_count,omitempty"`
	StockDeltaCount      int64  `json:"stock_delta_count,omitempty"`
	BalanceDeltaCount    int64  `json:"balance_delta_count,omitempty"`
	ErrorMessage         string `json:"error_message,omitempty"`
	TriggeredBy          string `json:"triggered_by,omitempty"`
	StartedAt            string `json:"started_at,omitempty"`
	CompletedAt          string `json:"completed_at,omitempty"`
	OccurredAt           string `json:"occurred_at,omitempty"`
}

// @Sync(ConnectorShadowMismatchEvent)
// ConnectorShadowMismatchEvent is emitted when shadow comparison detects discrepancies between an external system and Spanner.
type ConnectorShadowMismatchEvent struct {
	BaseEvent
	TenantID             string `json:"tenant_id"`
	RunID                string `json:"run_id"`
	Provider             string `json:"provider"`
	AccountLabel         string `json:"account_label,omitempty"`
	Direction            string `json:"direction,omitempty"`
	StreamName           string `json:"stream_name"`
	Status               string `json:"status,omitempty"`
	ObjectType           string `json:"object_type,omitempty"`
	RecordsProcessed     int64  `json:"records_processed,omitempty"`
	MismatchCount        int64  `json:"mismatch_count"`
	MissingExternalCount int64  `json:"missing_external_count,omitempty"`
	MissingInternalCount int64  `json:"missing_internal_count,omitempty"`
	StockDeltaCount      int64  `json:"stock_delta_count,omitempty"`
	BalanceDeltaCount    int64  `json:"balance_delta_count,omitempty"`
	BalanceDeltaMinor    int64  `json:"balance_delta_minor,omitempty"`
	ErrorMessage         string `json:"error_message,omitempty"`
	SummaryJSON          string `json:"summary_json,omitempty"`
	OccurredAt           string `json:"occurred_at,omitempty"`
}

// @Sync(ExportPackCreatedEvent)
// ExportPackCreatedEvent is emitted atomically when an auditor, bank, or tax export pack is generated.
type ExportPackCreatedEvent struct {
	BaseEvent
	TenantID   string `json:"tenant_id"`
	PackID     string `json:"pack_id"`
	PackType   string `json:"pack_type"`
	EntityID   string `json:"entity_id,omitempty"`
	PeriodID   string `json:"period_id,omitempty"`
	FromDate   string `json:"from_date,omitempty"`
	ToDate     string `json:"to_date,omitempty"`
	Format     string `json:"format"`
	Status     string `json:"status"`
	RowCount   int64  `json:"row_count"`
	Sha256Hex  string `json:"sha256_hex,omitempty"`
	CreatedBy  string `json:"created_by"`
	ExpiresAt  string `json:"expires_at"`
	CreatedAt  string `json:"created_at,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

// @Sync(AuditorAccessGrantedEvent)
// AuditorAccessGrantedEvent is emitted when a time-boxed read-only AUDITOR role grant is issued or updated.
type AuditorAccessGrantedEvent struct {
	BaseEvent
	TenantID   string `json:"tenant_id"`
	UserID     string `json:"user_id"`
	RoleName   string `json:"role_name"`
	Action     string `json:"action,omitempty"`
	GrantedBy  string `json:"granted_by"`
	ExpiresAt  string `json:"expires_at"`
	Reason     string `json:"reason,omitempty"`
	TraceID    string `json:"trace_id,omitempty"`
	OccurredAt string `json:"occurred_at,omitempty"`
}

// @Sync(EntityIntercompanyPostedEvent)
// EntityIntercompanyPostedEvent is emitted atomically when an intercompany transfer posts dual GL entries across legal entities.
type EntityIntercompanyPostedEvent struct {
	BaseEvent
	TenantID        string `json:"tenant_id"`
	TransferID      string `json:"transfer_id"`
	FromEntityID    string `json:"from_entity_id"`
	ToEntityID      string `json:"to_entity_id"`
	TransferType    string `json:"transfer_type"`
	AmountMinor     int64  `json:"amount_minor"`
	Currency        string `json:"currency"`
	PeriodID        string `json:"period_id"`
	ReferenceType   string `json:"reference_type,omitempty"`
	ReferenceID     string `json:"reference_id,omitempty"`
	DebitJournalID  string `json:"debit_journal_id,omitempty"`
	CreditJournalID string `json:"credit_journal_id,omitempty"`
	Status          string `json:"status"`
	CreatedBy       string `json:"created_by,omitempty"`
	OccurredAt      string `json:"occurred_at,omitempty"`
}

// @Sync(FxRevaluationPostedEvent)
// FxRevaluationPostedEvent is emitted atomically when month-end FX revaluation computes unrealized gain/loss and posts a GL entry.
type FxRevaluationPostedEvent struct {
	BaseEvent
	TenantID                string `json:"tenant_id"`
	RunID                   string `json:"run_id"`
	EntityID                string `json:"entity_id,omitempty"`
	PeriodID                string `json:"period_id"`
	BaseCurrency            string `json:"base_currency"`
	UnrealizedGainLossMinor int64  `json:"unrealized_gain_loss_minor"`
	RoundingDeltaMinor      int64  `json:"rounding_delta_minor"`
	JournalEntryID          string `json:"journal_entry_id,omitempty"`
	RatesJSON               string `json:"rates_json,omitempty"`
	CreatedBy               string `json:"created_by,omitempty"`
	OccurredAt              string `json:"occurred_at,omitempty"`
}

// @Sync(P2PEvent)
// P2PEvent is emitted atomically on P2P purchase order, goods receipt, and three-way match lifecycle transitions.
type P2PEvent struct {
	BaseEvent
	TenantID           string `json:"tenant_id"`
	SupplierID         string `json:"supplier_id,omitempty"`
	EntityID           string `json:"entity_id,omitempty"`
	PoID               string `json:"po_id"`
	ReceiptID          string `json:"receipt_id,omitempty"`
	VendorInvoiceID    string `json:"vendor_invoice_id,omitempty"`
	EsfDocID           string `json:"esf_doc_id,omitempty"`
	VendorID           string `json:"vendor_id,omitempty"`
	WarehouseID        string `json:"warehouse_id,omitempty"`
	Status             string `json:"status"`
	MatchStatus        string `json:"match_status,omitempty"`
	SubtotalMinor      int64  `json:"subtotal_minor,omitempty"`
	VatMinor           int64  `json:"vat_minor,omitempty"`
	TotalMinor         int64  `json:"total_minor,omitempty"`
	TotalReceivedMinor int64  `json:"total_received_minor,omitempty"`
	VarianceMinor      int64  `json:"variance_minor,omitempty"`
	VarianceReason     string `json:"variance_reason,omitempty"`
	Currency           string `json:"currency"`
	LineCount          int64  `json:"line_count,omitempty"`
	JournalEntryID     string `json:"journal_entry_id,omitempty"`
	ApprovalRequestID  string `json:"approval_request_id,omitempty"`
	ActorID            string `json:"actor_id,omitempty"`
	OccurredAt         string `json:"occurred_at,omitempty"`
}

// @Sync(ConnectorPushCompletedEvent)
// ConnectorPushCompletedEvent is emitted atomically when an approval-gated external ERP pushback completes.
type ConnectorPushCompletedEvent struct {
	BaseEvent
	TenantID         string `json:"tenant_id"`
	RunID            string `json:"run_id"`
	Provider         string `json:"provider"`
	AccountLabel     string `json:"account_label,omitempty"`
	Direction        string `json:"direction,omitempty"`
	StreamName       string `json:"stream_name"`
	Status           string `json:"status"`
	RecordsProcessed int64  `json:"records_processed"`
	ApprovalID       string `json:"approval_id"`
	ExternalRef      string `json:"external_ref,omitempty"`
	ErrorMessage     string `json:"error_message,omitempty"`
	TriggeredBy      string `json:"triggered_by,omitempty"`
	CompletedAt      string `json:"completed_at,omitempty"`
	OccurredAt       string `json:"occurred_at,omitempty"`
}

// @Sync(AnomalyCaseEvent)
// AnomalyCaseEvent is emitted atomically when an anomaly or fraud rule flags or resolves a case.
type AnomalyCaseEvent struct {
	BaseEvent
	TenantID       string `json:"tenant_id"`
	CaseID         string `json:"case_id"`
	RuleCode       string `json:"rule_code"`
	Category       string `json:"category"`
	Severity       string `json:"severity"`
	Status         string `json:"status"`
	ObjectType     string `json:"object_type"`
	ObjectID       string `json:"object_id"`
	ExposureMinor  int64  `json:"exposure_minor"`
	ScoreBps       int64  `json:"score_bps"`
	PlaybookAction string `json:"playbook_action,omitempty"`
	ProposalID     string `json:"proposal_id,omitempty"`
	AssignedTo     string `json:"assigned_to,omitempty"`
	ResolutionNote string `json:"resolution_note,omitempty"`
	ResolvedBy     string `json:"resolved_by,omitempty"`
	ResolvedAt     string `json:"resolved_at,omitempty"`
	DetectedAt     string `json:"detected_at,omitempty"`
	ActorID        string `json:"actor_id,omitempty"`
	OccurredAt     string `json:"occurred_at,omitempty"`
}

// @Sync(WhatIfSimulatedEvent)
// WhatIfSimulatedEvent is emitted when an E2E What-If simulation finishes computing multi-stage deltas.
type WhatIfSimulatedEvent struct {
	BaseEvent
	TenantID                 string `json:"tenant_id"`
	SimulationID             string `json:"simulation_id"`
	ScenarioID               string `json:"scenario_id,omitempty"`
	Name                     string `json:"name"`
	EntityID                 string `json:"entity_id,omitempty"`
	HorizonDays              int64  `json:"horizon_days"`
	ForecastDeltaMinor       int64  `json:"forecast_delta_minor"`
	ReplenishmentCostMinor   int64  `json:"replenishment_cost_minor"`
	DispatchCostDeltaMinor   int64  `json:"dispatch_cost_delta_minor"`
	CreditExposureDeltaMinor int64  `json:"credit_exposure_delta_minor"`
	NetPnLDeltaMinor         int64  `json:"net_pnl_delta_minor"`
	Status                   string `json:"status"`
	CreatedBy                string `json:"created_by,omitempty"`
	OccurredAt               string `json:"occurred_at,omitempty"`
}

// @Sync(HaccpExcursionReportedEvent)
// HaccpExcursionReportedEvent is emitted when cold-chain HACCP evaluation detects open temperature excursions.
type HaccpExcursionReportedEvent struct {
	BaseEvent
	TenantID          string   `json:"tenant_id"`
	ManifestID        string   `json:"manifest_id,omitempty"`
	ReadingID         string   `json:"reading_id,omitempty"`
	SensorID          string   `json:"sensor_id,omitempty"`
	TempMilliC        int64    `json:"temp_milli_c,omitempty"`
	MinMilliC         int64    `json:"min_milli_c,omitempty"`
	MaxMilliC         int64    `json:"max_milli_c,omitempty"`
	TotalReadings     int64    `json:"total_readings"`
	ExcursionCount    int64    `json:"excursion_count"`
	ComplianceRateBps int64    `json:"compliance_rate_bps"`
	OpenBreaches      int64    `json:"open_breaches"`
	AffectedLotIDs    []string `json:"affected_lot_ids,omitempty"`
	Status            string   `json:"status"`
	ActorID           string   `json:"actor_id,omitempty"`
	OccurredAt        string   `json:"occurred_at,omitempty"`
}

// @Sync(ProposalL2AutoExecutedEvent)
// ProposalL2AutoExecutedEvent is emitted when a soak-eligible low-risk proposal is auto-executed under L2 autonomy.
type ProposalL2AutoExecutedEvent struct {
	BaseEvent
	TenantID             string `json:"tenant_id"`
	ProposalID           string `json:"proposal_id"`
	AgentID              string `json:"agent_id"`
	ProposalType         string `json:"proposal_type"`
	TargetObjectType     string `json:"target_object_type"`
	TargetObjectID       string `json:"target_object_id"`
	AutonomyLevel        string `json:"autonomy_level"`
	Status               string `json:"status"`
	PayloadHash          string `json:"payload_hash"`
	EstimatedImpactMinor int64  `json:"estimated_impact_minor"`
	Currency             string `json:"currency"`
	ConfidenceBps        int64  `json:"confidence_bps"`
	SoakDays             int64  `json:"soak_days"`
	SoakExecutedCount    int64  `json:"soak_executed_count"`
	RollbackRateBps      int64  `json:"rollback_rate_bps"`
	RoiEventID           string `json:"roi_event_id,omitempty"`
	ActorID              string `json:"actor_id,omitempty"`
	OccurredAt           string `json:"occurred_at,omitempty"`
}

// @Sync(ScimIdentitySyncedEvent)
// ScimIdentitySyncedEvent is emitted atomically when SCIM 2.0 provisions, updates, or deprovisions a user or group.
type ScimIdentitySyncedEvent struct {
	BaseEvent
	TenantID     string   `json:"tenant_id"`
	ResourceType string   `json:"resource_type"`
	ResourceID   string   `json:"resource_id"`
	ExternalID   string   `json:"external_id,omitempty"`
	UserName     string   `json:"user_name,omitempty"`
	DisplayName  string   `json:"display_name,omitempty"`
	Email        string   `json:"email,omitempty"`
	Active       bool     `json:"active"`
	Action       string   `json:"action"`
	RolesSynced  []string `json:"roles_synced,omitempty"`
	ActorID      string   `json:"actor_id,omitempty"`
	TraceID      string   `json:"trace_id,omitempty"`
	OccurredAt   string   `json:"occurred_at,omitempty"`
}

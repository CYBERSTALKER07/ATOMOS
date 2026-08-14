package events

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
}

// ParentOrderEvent is the multi-supplier ParentOrders rollup lifecycle (B3 M-P0-6).
type ParentOrderEvent struct {
	BaseEvent
	ParentOrderID string `json:"parent_order_id"`
	RetailerID    string `json:"retailer_id"`
	Status        string `json:"status,omitempty"`
	Currency      string `json:"currency,omitempty"`
	TotalMinor    int64  `json:"total_minor,omitempty"`
	ChildCount    int    `json:"child_count,omitempty"`
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
	Currency          string `json:"currency,omitempty"`
	TransactionID     string `json:"transaction_id,omitempty"`
	Source            string `json:"source,omitempty"`
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
// ProductEvent handles product handling classification updates.
type ProductEvent struct {
	BaseEvent
	ProductID         string   `json:"product_id"`
	SupplierID        string   `json:"supplier_id"`
	HandlingClass     string   `json:"handling_class"`
	RequiresColdChain bool     `json:"requires_cold_chain"`
	IsHazardous       bool     `json:"is_hazardous"`
	IsPerishable      bool     `json:"is_perishable"`
	StorageTempMinC   *float64 `json:"storage_temp_min_c,omitempty"`
	StorageTempMaxC   *float64 `json:"storage_temp_max_c,omitempty"`
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

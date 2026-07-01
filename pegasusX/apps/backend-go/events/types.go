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
	VehicleID          string `json:"vehicle_id"`
	SupplierID         string `json:"supplier_id"`
	HomeNodeID         string `json:"home_node_id,omitempty"`
	HomeNodeType       string `json:"home_node_type,omitempty"`
	IsActive           bool   `json:"is_active,omitempty"`
	UnavailableReason  string `json:"unavailable_reason,omitempty"`
	UnavailableNote    string `json:"unavailable_note,omitempty"`
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

// ManifestEvent handles manifest lifecycle events.
type ManifestEvent struct {
	BaseEvent
	ManifestID     string `json:"manifest_id"`
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
	SupplierID   string  `json:"supplier_id"`
	WarehouseID  string  `json:"warehouse_id,omitempty"`
	FactoryID    string  `json:"factory_id,omitempty"`
	InsightID    string  `json:"insight_id,omitempty"`
	ProductID    string  `json:"product_id,omitempty"`
	OverrideID   string  `json:"override_id,omitempty"`
	Action       string  `json:"action,omitempty"`
	Polygon      string  `json:"polygon_geojson,omitempty"`
	TTLSeconds   int64   `json:"ttl_seconds,omitempty"`
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
}

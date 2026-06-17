// Canonical TypeScript DTO mirrors of the backend contract. Hand-aligned with
// contracts/events.schema.json. Bump `WireVersion` and individual event
// `schema_version` fields in lockstep with the JSON Schema when shapes change.

export const WireVersion = 1 as const;

// ── RFC 7807 Problem Detail ──────────────────────────────────────────────────
export interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  code?: string;
  trace_id?: string;
}

export function isProblemDetail(obj: any): obj is ProblemDetail {
  return (
    typeof obj === "object" &&
    obj !== null &&
    typeof obj.type === "string" &&
    typeof obj.title === "string" &&
    typeof obj.status === "number"
  );
}

export interface PaymentGatewayDegradedPayload {
  gateway: string;
  reason: string;
}

// ── Role + scalar primitives ────────────────────────────────────────────────
export type Role =
  | "ADMIN"
  | "RETAILER"
  | "DRIVER"
  | "PAYLOAD"
  | "FACTORY_ADMIN"
  | "WAREHOUSE_ADMIN";

export type SupplierId = string;
export type RetailerId = string;
export type OrderId = string;
export type DriverId = string;
export type VehicleId = string;
export type WarehouseId = string;
export type FactoryId = string;
export type RouteId = string;
export type ManifestId = string;
export type SessionId = string;
export type CommandId = string;
export type H3Cell = string; // 15-char hex
export type Iso4217 = string; // 3-char currency code

export interface Money {
  /** int64 minor units (e.g. tiyin/cents). Never float. */
  amount: number;
  currency: Iso4217;
}

export type HomeNodeType = "WAREHOUSE" | "FACTORY";
export interface HomeNode {
  home_node_type: HomeNodeType;
  home_node_id: WarehouseId | FactoryId;
}

export type PaymentGateway = "GLOBAL_PAY" | "ADYEN" | "AIRWALLEX" | "CASH";

export type OrderStatus =
  | "PENDING"
  | "SCHEDULED"
  | "AUTO_ACCEPTED"
  | "LOADED"
  | "IN_TRANSIT"
  | "ARRIVED"
  | "ARRIVED_SHOP_CLOSED"
  | "AWAITING_PAYMENT"
  | "PENDING_CASH_COLLECTION"
  | "DELIVERED_ON_CREDIT"
  | "COMPLETED"
  | "CANCELLED"
  | "RECONCILIATION_REQUIRED"
  | "DELAYED"
  | "BACKORDERED"
  // Operational / legacy client aliases still seen on WS payloads and fleet guards.
  | "DISPATCHED"
  | "ARRIVING"
  | "EN_ROUTE";

export type OrderSource = "MANUAL" | "MANUAL_PREORDER" | "AI_PREORDER" | "BACKORDER";

export type DeliveryMode = "STANDARD" | "SCHEDULED";

export type DeliveryPriority = "STANDARD" | "EXPRESS";

export type PreorderPhase = "DRAFT" | "CONFIRMED" | "LOCKED" | "AUTO_ACCEPTED";

export type OutOfStockPolicy = "INHERIT" | "REJECT" | "ACCEPT_BACKORDER";

export interface StockWarning {
  sku: string;
  requested: number;
  available: number;
  backorder_qty: number;
  accepts_backorder: boolean;
}

export interface SupplierOrderCheckoutResult {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  total: number;
  item_count: number;
}

export interface UnifiedCheckoutResponse {
  status: string;
  invoice_id: string;
  total: number;
  supplier_orders: SupplierOrderCheckoutResult[];
  backordered_item_count?: number;
  backorder_order_id?: string;
  stock_warnings?: StockWarning[];
}

export interface RetailerCatalogProductStock {
  available_stock?: number;
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  show_stock_counts?: boolean;
  max_quantity?: number;
}

export interface CheckoutPreviewResponse {
  ok: boolean;
  blocked?: boolean;
  code?: string;
  message?: string;
  rejected_skus?: string[];
  oos_items?: string[];
  stock_warnings?: StockWarning[];
  max_quantities?: Record<string, number>;
  backordered_item_count?: number;
  show_stock_counts?: boolean;
}

export interface WarehouseOpsSettings {
  warehouse_id: string;
  name: string;
  region_id?: string;
  default_out_of_stock_policy: OutOfStockPolicy;
  operating_schedule?: Record<string, unknown>;
  is_on_shift: boolean;
  ops_always_available: boolean;
}

export interface WarehouseInventoryPolicyPatchRequest {
  out_of_stock_policy?: OutOfStockPolicy;
  reorder_threshold?: number;
}

export type OrderConfirmationStatus = "CONFIRMED" | "DRAFT" | "PENDING" | "REJECTED" | "AUTO_CONFIRMED";

// ── Supplier API contracts ─────────────────────────────────────────────────
export interface SupplierRegisterAccount {
  legalName: string;
  contactName: string;
  email: string;
  country: string;
  phone: string;
  password: string;
}

export interface SupplierRegisterAddress {
  name?: string;
  address: string;
  lat: number;
  lng: number;
}

export interface SupplierRegisterLocation {
  warehouse: SupplierRegisterAddress;
  sameAsWarehouse: boolean;
  billing: SupplierRegisterAddress;
}

export interface SupplierRegisterBusiness {
  taxId: string;
  companyRegNumber: string;
  fleetVehicleCount: number;
  fleetMaxVU: number;
  factoryCount: number;
}

export interface SupplierRegisterRequest {
  account: SupplierRegisterAccount;
  location: SupplierRegisterLocation;
  business: SupplierRegisterBusiness;
  categories: string[];
  phone: string;
}

export interface SupplierRegisterResponse {
  supplier_id: SupplierId;
  legal_name: string;
  is_registered: boolean;
  is_configured: boolean;
  next_step: string;
  token?: string;
}

export interface SupplierLoginRequest {
  phone: string;
  password: string;
}

export interface SupplierLoginResponse {
  supplier_id: SupplierId;
  is_registered: boolean;
  is_configured: boolean;
  next_step: string;
  token?: string;
  refresh_token?: string;
}

export interface SupplierDashboardResponse {
  supplier_id: SupplierId;
  is_configured: boolean;
  inventory_skus: number;
  pending_orders: number;
  updated_at: string;
  orders_by_status?: Record<string, number>;
  today_revenue_minor?: number;
  currency?: string;
  active_drivers?: number;
  total_drivers?: number;
  retailers_ordered_today?: number;
  total_retailers?: number;
  delivery_completion_rate_pct?: number;
  fleet_vu_used?: number;
  fleet_vu_total?: number;
  recent_manifests?: SupplierManifestRow[];
  activity_events?: SupplierActivityEvent[];
}

export interface SupplierAnalyticsVelocityPoint {
  date: string;
  orders_created: number;
  orders_completed: number;
}

export interface SupplierAnalyticsVelocityResponse {
  period_days: number;
  points: SupplierAnalyticsVelocityPoint[];
  generated_at: string;
}

export interface SupplierAnalyticsRevenuePoint {
  date: string;
  revenue_minor: number;
}

export interface SupplierAnalyticsRevenueResponse {
  currency: string;
  total_minor: number;
  series: SupplierAnalyticsRevenuePoint[];
  generated_at: string;
}

export interface SupplierDemandSummaryItem {
  sku_id: string;
  product_name: string;
  total_qty: number;
  retailer_count: number;
}

export interface SupplierDemandSummaryResponse {
  total_retailers: number;
  total_pallets: number;
  total_value: number;
  prediction_count: number;
  items: SupplierDemandSummaryItem[];
  generated_at: string;
}

export interface SupplierDemandHistoryPoint {
  date: string;
  predicted: number;
  actual: number;
  predicted_qty: number;
  actual_qty: number;
}

export interface SupplierDemandUpcomingRow {
  date: string;
  retailer_name: string;
  sku_id: string;
  product_name: string;
  predicted_qty: number;
}

export interface SupplierDemandHistoryResponse {
  time_series: SupplierDemandHistoryPoint[];
  upcoming: SupplierDemandUpcomingRow[];
}

export interface SupplierInventoryImportResult {
  session_id?: string;
  applied: number;
  skipped: number;
  errors?: string[];
  updated_at: string;
}

export interface SupplierImportSessionCreateResponse {
  session_id: string;
  status: string;
  file_name: string;
  upload_url?: string;
  gcs_path?: string;
  content_type?: string;
}

export interface SupplierImportSession {
  session_id: string;
  supplier_id?: string;
  status: string;
  file_name: string;
  total_rows?: number;
  error_summary?: unknown;
  created_at?: string;
  updated_at?: string;
}

export interface SupplierImportMappingCandidate {
  source_column: string;
  target_field: string;
  confidence: number;
  reason?: string;
}

export interface SupplierImportMappingResponse {
  session_id: string;
  mapping_json?: {
    mappings?: SupplierImportMappingCandidate[];
    anomalies?: unknown[];
    model?: string;
  };
}

export interface SupplierImportIngestResponse {
  session_id: string;
  status: string;
  rows_staged: number;
  valid_rows?: number;
  invalid_rows?: number;
  suggested_mappings?: number;
}

export interface SupplierImportApplyResponse {
  session_id: string;
  status: string;
  applied_rows: number;
  affected_warehouses: number;
  created_products?: number;
  idempotent?: boolean;
}

export interface SupplierManifestRow {
  manifest_id: string;
  /** Portal queue status; mirrored as `state` for payloader clients. */
  status: "DRAFT" | "LOADING" | "DISPATCHED" | "SEALED" | string;
  state?: string;
  orders_count: number;
  stop_count?: number;
  driver_id?: string;
  driver_name: string;
  vehicle_id?: string;
  truck_id?: string;
  vehicle_plate?: string;
  total_vu: number;
  total_volume_vu?: number;
  max_volume_vu?: number;
  updated_at: string;
}

export interface SupplierManifestsResponse {
  manifests: SupplierManifestRow[];
}

export interface SupplierSupplyLaneRow {
  lane_id: string;
  name: string;
  warehouse_id: string;
  h3_cells: number;
  drivers: number;
  orders_today: number;
  capacity: number;
  utilization_pct: number;
}

export interface SupplierSupplyLanesResponse {
  lanes: SupplierSupplyLaneRow[];
}

export interface SupplierExceptionRow {
  order_id: string;
  kind: string;
  status: string;
  retailer_id?: string;
  note?: string;
  manifest_id?: string;
  updated_at: string;
}

export interface SupplierExceptionsResponse {
  exceptions: SupplierExceptionRow[];
}

export interface SupplierManifestExceptionRow {
  exception_id: string;
  manifest_id: string;
  order_id: string;
  reason: string;
  metadata?: string;
  attempt_count: number;
  escalated: boolean;
  created_at: string;
}

export interface SupplierManifestExceptionsResponse {
  exceptions: SupplierManifestExceptionRow[];
}

export interface SupplierManifestOrderWire {
  order_id: string;
  retailer_id?: string;
  amount: number;
  state: string;
  status: string;
  route_id?: string;
  warehouse_id?: string;
}

export interface SupplierManifestDetail extends SupplierManifestRow {
  orders?: SupplierManifestOrderWire[];
  overflow_count?: number;
  sealed_at?: string;
  dispatched_at?: string;
  created_at?: string;
  region_code?: string;
}

export interface SupplierManifestInjectOrderRequest {
  order_id: string;
  volume_vu?: number;
}

export interface SupplierManifestSealResponse {
  status: string;
  manifest_id: string;
  state: string;
  sealed_at?: string;
  stop_count?: number;
  volume_vu?: number;
  max_vu?: number;
}

export interface ShopClosedAttemptRow {
  attempt_id: string;
  order_id: string;
  original_route_id?: string;
  driver_id: string;
  retailer_id: string;
  resolution: string;
  created_at: string;
  updated_at?: string;
}

export interface ShopClosedActiveResponse {
  data: ShopClosedAttemptRow[];
}

export interface ShopClosedResolveRequest {
  attempt_id: string;
  action: "WAIT" | "BYPASS" | "RETURN_TO_DEPOT";
}

export interface NegotiationProposalItem {
  sku_id: string;
  original_qty: number;
  proposed_qty: number;
}

export interface NegotiationProposalRow {
  proposal_id: string;
  order_id: string;
  driver_id: string;
  items: NegotiationProposalItem[];
  created_at: string;
}

export interface NegotiationPendingResponse {
  data: NegotiationProposalRow[];
}

export interface NegotiationResolveRequest {
  proposal_id: string;
  action: "APPROVE" | "REJECT";
  resolution?: string;
}

export interface NegotiationResolveResponse {
  status: string;
  proposal_id: string;
  order_id: string;
}

export interface PaymentBypassRequest {
  order_id: string;
  reason?: string;
}

export interface PaymentBypassResponse {
  status: string;
  bypass_token: string;
  order_id: string;
}

export interface SupplierEmpathyAdoption {
  total_predictions: number;
  predictions_dormant: number;
  predictions_waiting: number;
  predictions_fired: number;
  predictions_rejected: number;
}

export interface SupplierBroadcastRequest {
  title: string;
  body: string;
  role?: string;
}

export interface SupplierBroadcastResponse {
  status: string;
  supplier_id: string;
}

export interface SupplierReplenishmentTriggerResponse {
  status: string;
  request_id: string;
  warehouse_id: string;
}

export interface SupplierFleetOrderRow {
  id: string;
  order_id: string;
  retailer_id?: string;
  driver_id?: string;
  status: string;
  state?: string;
  route_id?: string;
  total_minor?: number;
  currency?: string;
  updated_at?: string;
  driver_location?: SupplierOrderDriverLocation;
}

export interface SupplierFleetLiveRoute {
  manifest_id: string;
  route_id: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
  route_geometry?: RouteGeometryWire;
  driver_location?: SupplierOrderDriverLocation;
  live_location_available: boolean;
  location_stale?: boolean;
}

export interface SupplierFleetLiveMapResponse {
  routes: SupplierFleetLiveRoute[];
  fetched_at: string;
}

/** Warehouse ops live fleet map — same route wire shape as supplier fleet live map. */
export type WarehouseFleetLiveRoute = SupplierFleetLiveRoute;

export interface WarehouseFleetLiveMapResponse {
  routes: WarehouseFleetLiveRoute[];
  warehouse_id: string;
  fetched_at: string;
}

export interface SupplierBusinessSetupRequest {
  taxId: string;
  registrationNumber?: string;
  headquartersAddress: string;
  city: string;
  postalCode?: string;
}

export interface SupplierBusinessSetupResponse {
  supplier_id: SupplierId;
  is_registered: boolean;
  next_step: string;
}

export interface SupplierBillingSetupRequest {
  bankName: string;
  accountHolder: string;
  accountNumber: string;
  swiftBic: string;
  iban?: string;
  selectedGateways: PaymentGateway[];
}

export interface SupplierBillingSetupResponse {
  supplier_id: SupplierId;
  is_configured: boolean;
  selected_gateways: PaymentGateway[];
}

export interface SupplierConfigureRequest {
  legal_name?: string;
}

export interface SupplierConfigureResponse {
  supplier_id: SupplierId;
  is_registered: boolean;
  is_configured: boolean;
  completed_at: string;
}

export interface SupplierProfile {
  supplier_id: SupplierId;
  legal_name: string;
  contact_name: string;
  email: string;
  phone: string;
  country: string;
  currency: Iso4217;
  categories: string[];
  is_registered: boolean;
  is_configured: boolean;
  selected_gateways: PaymentGateway[];
  updated_at: string;
}

export interface SupplierProfileUpdateRequest {
  legal_name?: string;
  contact_name?: string;
  email?: string;
  phone?: string;
  categories?: string[];
}

export interface SupplierTopologyInventorySeed {
  product_id: string;
  quantity: number;
}

export interface SupplierTopologyWarehouseInput {
  warehouse_id?: WarehouseId;
  name: string;
  lat: number;
  lng: number;
  coverage_radius_km?: number;
  is_active?: boolean;
  is_on_shift?: boolean;
  transfer_mode?: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id?: FactoryId;
  default_out_of_stock_policy?: OutOfStockPolicy;
  operating_schedule?: Record<string, unknown>;
  initial_inventory?: SupplierTopologyInventorySeed[];
}

export interface SupplierTopologyFactoryInput {
  factory_id?: FactoryId;
  name: string;
  lat: number;
  lng: number;
  is_active?: boolean;
}

export interface SupplierTopologyUpdateRequest {
  warehouses: SupplierTopologyWarehouseInput[];
  factories: SupplierTopologyFactoryInput[];
}

export interface SupplierTopologyWarehouse {
  warehouse_id: WarehouseId;
  name: string;
  lat: number;
  lng: number;
  coverage_radius_km: number;
  is_active: boolean;
  is_on_shift: boolean;
  transfer_mode?: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id?: FactoryId;
  primary_factory_id?: FactoryId;
  default_out_of_stock_policy?: OutOfStockPolicy;
  operating_schedule?: Record<string, unknown>;
  initial_inventory?: SupplierTopologyInventorySeed[];
  created_at: string;
  updated_at: string;
}

export interface SupplierInventoryRow {
  sku_id: string;
  product_name: string;
  quantity: number;
  unit_price_minor: number;
  currency: string;
}

export interface SupplierInventoryResponse {
  items: SupplierInventoryRow[];
}

export interface SupplierTopologyFactory {
  factory_id: FactoryId;
  name: string;
  lat: number;
  lng: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface SupplierTopologyResponse {
  supplier_id: SupplierId;
  warehouses: SupplierTopologyWarehouse[];
  factories: SupplierTopologyFactory[];
  updated_at: string;
}

export interface SupplierOrgMemberCreateRequest {
  name: string;
  email?: string;
  phone: string;
  password: string;
  supplier_role: Role;
  assigned_warehouse_id?: WarehouseId;
  assigned_factory_id?: FactoryId;
  is_active?: boolean;
}

export interface SupplierOrgMemberUpdateRequest {
  name?: string;
  supplier_role?: Role;
  assigned_warehouse_id?: WarehouseId;
  assigned_factory_id?: FactoryId;
  is_active?: boolean;
}

export interface SupplierOrgMember {
  user_id: string;
  supplier_id: SupplierId;
  name: string;
  email?: string;
  phone: string;
  supplier_role: Role;
  assigned_warehouse_id?: WarehouseId;
  assigned_factory_id?: FactoryId;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface SupplierOrgMembersResponse {
  supplier_id: SupplierId;
  items: SupplierOrgMember[];
  updated_at: string;
}

export interface SupplierFleetDriverCreateRequest {
  name: string;
  phone: string;
  pin: string;
  home_node_type: HomeNodeType;
  home_node_id: WarehouseId | FactoryId;
  vehicle_id?: VehicleId;
  is_active?: boolean;
}

export interface SupplierFleetDriver {
  driver_id: DriverId;
  supplier_id: SupplierId;
  name: string;
  phone: string;
  home_node_type: HomeNodeType;
  home_node_id: WarehouseId | FactoryId;
  vehicle_id?: VehicleId;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface SupplierFleetDriversResponse {
  supplier_id: SupplierId;
  items: SupplierFleetDriver[];
  updated_at: string;
}

export interface SupplierFleetVehicleCreateRequest {
  label?: string;
  license_plate: string;
  home_node_type: HomeNodeType;
  home_node_id: WarehouseId | FactoryId;
  is_active?: boolean;
}

export interface SupplierFleetVehicle {
  vehicle_id: VehicleId;
  supplier_id: SupplierId;
  label?: string;
  license_plate: string;
  home_node_type: HomeNodeType;
  home_node_id: WarehouseId | FactoryId;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface SupplierFleetVehiclesResponse {
  supplier_id: SupplierId;
  items: SupplierFleetVehicle[];
  updated_at: string;
}

export interface SupplierPricingRule {
  supplier_id: SupplierId;
  base_markup_bps: number;
  retailer_discount_bps: number;
  min_margin_bps: number;
  currency: Iso4217;
  rule_version: number;
  updated_at: string;
}

export interface SupplierPricingRuleUpdateRequest {
  base_markup_bps?: number;
  retailer_discount_bps?: number;
  min_margin_bps?: number;
  currency?: Iso4217;
}

export interface RetailerPriceOverride {
  override_id: string;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  product_id: string;
  price: number;
  set_by: string;
  set_by_role: string;
  is_active: boolean;
  notes?: string;
  expires_at?: string;
  created_at: string;
}

export interface RetailerPriceOverridesResponse {
  overrides: RetailerPriceOverride[];
  total: number;
}

export interface CreateRetailerPriceOverrideRequest {
  retailer_id: RetailerId;
  product_id: string;
  sku_id?: string;
  price: number;
  notes?: string;
  expires_at?: string;
}

export interface CreateRetailerPriceOverrideResponse {
  status: string;
  override_id: string;
  retailer_id: RetailerId;
  product_id: string;
  price: number;
}

export type SupplierPromotionScopeType = "PRODUCT" | "CATEGORY" | "ALL_PRODUCTS";
export type SupplierPromotionRetailerScope = "ALL" | "ALLOWLIST";

export interface SupplierPromotion {
  promotion_id: string;
  supplier_id: SupplierId;
  name: string;
  description?: string;
  discount_bps: number;
  scope_type: SupplierPromotionScopeType;
  scope_product_id?: string;
  scope_category_id?: string;
  retailer_scope: SupplierPromotionRetailerScope;
  retailer_ids?: RetailerId[];
  min_line_quantity?: number;
  min_order_amount_minor?: number;
  starts_at?: string;
  ends_at?: string;
  is_active: boolean;
  priority: number;
  version: number;
  created_at: string;
  updated_at: string;
}

export interface SupplierPromotionUpsertRequest {
  name: string;
  description?: string;
  discount_bps: number;
  scope_type: SupplierPromotionScopeType;
  scope_product_id?: string;
  scope_category_id?: string;
  retailer_scope?: SupplierPromotionRetailerScope;
  retailer_ids?: RetailerId[];
  min_line_quantity?: number;
  min_order_amount_minor?: number;
  starts_at?: string | null;
  ends_at?: string | null;
  priority?: number;
}

export interface SupplierPromotionsResponse {
  promotions: SupplierPromotion[];
}

export interface ProductOffer {
  product_id: string;
  list_price_minor: number;
  sale_price_minor?: number;
  discount_bps?: number;
  promotion_id?: string;
  promotion_name?: string;
  promotion_label?: string;
  promotion_ends_at?: string;
}

export interface CheckoutQuoteLineInput {
  product_id: string;
  category_id?: string;
  quantity: number;
  unit_price_minor: number;
  currency?: Iso4217;
}

export interface CheckoutQuoteRequest {
  supplier_id: SupplierId;
  lines: CheckoutQuoteLineInput[];
}

export interface CheckoutQuotedLine {
  product_id: string;
  quantity: number;
  list_unit_price_minor: number;
  unit_price_minor: number;
  line_total_minor: number;
  currency: Iso4217;
  discount_bps?: number;
  promotion_id?: string;
  promotion_name?: string;
  promotion_label?: string;
}

export interface CheckoutQuoteResponse {
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  lines: CheckoutQuotedLine[];
  subtotal_minor: number;
  discount_minor: number;
  total_minor: number;
  currency: Iso4217;
}

export interface SupplierOrderDriverLocation {
  driver_id: DriverId;
  supplier_id: SupplierId;
  lat: number;
  lng: number;
  latitude: number;
  longitude: number;
  velocity?: number;
  heading?: number;
  reported_at: string;
  received_at: string;
  stale_after_seconds: number;
}

export interface SupplierOrder {
  order_id: OrderId;
  supplier_id?: SupplierId;
  retailer_id: RetailerId;
  warehouse_id?: WarehouseId;
  driver_id?: DriverId;
  vehicle_id?: VehicleId;
  route_id?: RouteId;
  manifest_id?: ManifestId;
  status: string;
  tracking_status?: string;
  decision?: string;
  note?: string;
  total_minor: number;
  currency: Iso4217;
  live_location_available: boolean;
  driver_location?: SupplierOrderDriverLocation;
  created_at: string;
  updated_at: string;
}

export interface SupplierOrdersResponse {
  orders: SupplierOrder[];
  total?: number;
  limit?: number;
  offset?: number;
}

export interface SupplierActivityEvent {
  id: string;
  type: string;
  timestamp: string;
  description: string;
  order_id?: string;
  manifest_id?: string;
}

export interface SupplierActivityResponse {
  events: SupplierActivityEvent[];
}

export interface SupplierDispatchPreview {
  undispatched_orders: Array<{
    order_id: string;
    retailer_id?: string;
    warehouse_id?: string;
    total_minor: number;
    currency: string;
  }>;
  available_drivers: Array<{
    driver_id: string;
    name: string;
    vehicle_id?: string;
    truck_status?: string;
  }>;
  unavailable_drivers?: Array<{
    driver_id: string;
    name: string;
    unavailable_reason?: string;
  }>;
  pending_count?: number;
  available_driver_count?: number;
  proposed_routes?: DispatchProposedRoute[];
  optimizer_source?: string;
  optimizer_warnings?: string[];
  window_constrained_count?: number;
}

export interface SupplierDispatchExecuteRoute {
  manifest_id?: string;
  route_id?: string;
  driver_id: string;
  vehicle_id?: string;
  order_ids: string[];
  volume_vu?: number;
  max_volume_vu?: number;
}

export interface SupplierDispatchExecuteRequest {
  mode?: "MANUAL" | "AUTO" | string;
}

export interface SupplierDispatchExecuteResponse {
  status: string;
  supplier_id: string;
  warehouse_id?: string;
  manifests_created?: number;
  orders_assigned?: number;
  optimizer_source?: string;
  warnings?: string[];
  manifests?: SupplierDispatchExecuteRoute[];
  orphan_order_ids?: string[];
}

export interface RouteGeometryWire {
  route_id?: string;
  encoded_polyline?: string;
  coordinates: Array<{ lat: number; lng: number }>;
  source: string;
  stop_count?: number;
}

export interface RouteStepWire {
  instruction: string;
  distance_m: number;
  duration_s: number;
  maneuver?: string;
  lat: number;
  lng: number;
}

export interface DispatchProposedRoute {
  driver_id?: string;
  driver_name?: string;
  loaded_volume?: number;
  max_volume?: number;
  util_pct?: number;
  stop_count?: number;
  volume_vu?: number;
  max_volume_vu?: number;
  order_ids?: string[];
  stops?: Array<{
    order_id: string;
    retailer_id?: string;
    retailer_name?: string;
    lat?: number;
    lng?: number;
    volume_vu?: number;
  }>;
  route_geometry?: RouteGeometryWire;
}

export type SupplierAIRecommendationStatus =
  | "PENDING"
  | "ACKNOWLEDGED"
  | "OVERRIDDEN"
  | "DISMISSED";

export type SupplierAIRecommendationDecision =
  | "ACKNOWLEDGED"
  | "OVERRIDDEN"
  | "DISMISSED"
  | "REOPENED";

export interface SupplierAIRecommendationEvidence {
  label: string;
  value: string;
  href?: string;
}

export interface SupplierAIRecommendation {
  recommendation_id: string;
  supplier_id: SupplierId;
  aggregate_id: string;
  aggregate_type: string;
  action: string;
  status: SupplierAIRecommendationStatus | string;
  score: number;
  confidence: number;
  source: string;
  explanation: string;
  reason_codes: string[];
  evidence: SupplierAIRecommendationEvidence[];
  decision?: SupplierAIRecommendationDecision | string;
  decision_note?: string;
  decided_by?: string;
  decided_at?: string;
  expires_at?: string;
  generated_at: string;
  updated_at: string;
}

export interface SupplierAIRecommendationsQuery {
  status?: string;
  limit?: number;
}

export interface SupplierAIRecommendationsResponse {
  items: SupplierAIRecommendation[];
  count: number;
  limit: number;
  status?: string;
  updated_at: string;
}

export interface SupplierAIRecommendationDecisionRequest {
  recommendation_id: string;
  decision: SupplierAIRecommendationDecision;
  note?: string;
}

export interface SupplierAIRecommendationDecisionResponse {
  recommendation: SupplierAIRecommendation;
}

export interface RetailerPricingSummary {
  base_markup_bps: number;
  retailer_discount_bps: number;
  min_margin_bps: number;
  currency: Iso4217 | "";
  rule_version: number;
  updated_at?: string;
}

export interface RetailerSupplierPreference {
  supplier_id: SupplierId;
  name: string;
  is_favorite: boolean;
  pricing?: RetailerPricingSummary;
}

export interface RetailerPricingRuleResponse {
  supplier_id: SupplierId;
  configured: boolean;
  pricing: RetailerPricingSummary;
}

export interface PaymentLedgerQuery {
  supplier_id?: SupplierId;
  order_id?: OrderId;
  session_id?: SessionId;
  gateway?: PaymentGateway | string;
  entry_type?: string;
  occurred_from?: string;
  occurred_to?: string;
  limit?: number;
}

export interface PaymentLedgerEntry {
  ledger_entry_id: string;
  session_id?: SessionId;
  order_id?: OrderId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  gateway: string;
  entry_type: string;
  amount_minor: number;
  currency: Iso4217;
  reference_id?: string;
  source: string;
  occurred_at: string;
  created_at: string;
}

export interface PaymentLedgerResponse {
  items: PaymentLedgerEntry[];
  count: number;
  limit: number;
  supplier_id: SupplierId | "";
  filters?: {
    order_id?: OrderId;
    session_id?: SessionId;
    gateway?: string;
    entry_type?: string;
    occurred_from?: string | null;
    occurred_to?: string | null;
  };
}

export interface SettlementAuthorityQuery {
  supplier_id?: SupplierId;
  gateway?: PaymentGateway | string;
  entry_type?: string;
  occurred_from?: string;
  occurred_to?: string;
  group_limit?: number;
}

export interface SettlementAuthorityRow {
  gateway: string;
  entry_type: string;
  currency: Iso4217;
  entry_count: number;
  amount_minor_total: number;
  first_occurred_at: string;
  last_occurred_at: string;
}

export interface SettlementCurrencyTotal {
  currency: Iso4217;
  entry_count: number;
  amount_minor_total: number;
}

export interface SettlementAuthorityResponse {
  items: SettlementAuthorityRow[];
  count: number;
  group_limit: number;
  supplier_id: SupplierId | "";
  entry_count_total: number;
  totals_by_currency: SettlementCurrencyTotal[];
  filters?: {
    gateway?: string;
    entry_type?: string;
    occurred_from?: string | null;
    occurred_to?: string | null;
  };
}

export interface PaymentChargebackRequest {
  order_id: OrderId;
  retailer_id: RetailerId;
  gateway: PaymentGateway | string;
  amount: number;
  currency?: Iso4217;
  amount_uzs?: number;
}

export interface PaymentChargebackResponse {
  status: string;
}

export interface PaymentChargebackReversalRequest {
  session_id: SessionId;
}

export interface PaymentChargebackReversalResponse {
  status: string;
}

export interface ReconciliationMismatchQuery {
  supplier_id?: SupplierId;
  gateway?: PaymentGateway | string;
  occurred_from?: string;
  occurred_to?: string;
  group_limit?: number;
  mismatch_threshold_minor?: number;
}

export interface ReconciliationEntryTypeTotal {
  entry_type: string;
  entry_count: number;
  amount_minor_total: number;
  signed_amount_minor_total: number;
}

export interface ReconciliationMismatchRow {
  gateway: string;
  currency: Iso4217;
  entry_count_total: number;
  group_count: number;
  credit_amount_minor_total: number;
  debit_amount_minor_total: number;
  net_amount_minor: number;
  first_occurred_at: string;
  last_occurred_at: string;
  entry_type_totals: ReconciliationEntryTypeTotal[];
}

export interface ReconciliationMismatchResponse {
  items: ReconciliationMismatchRow[];
  count: number;
  analyzed_group_count: number;
  group_limit: number;
  mismatch_threshold_minor: number;
  supplier_id: SupplierId | "";
  filters?: {
    gateway?: string;
    occurred_from?: string | null;
    occurred_to?: string | null;
  };
}

// ── Envelope + event-type union ─────────────────────────────────────────────
export interface WsEventEnvelope<T extends string = string, P = unknown> {
  type: T;
  trace_id: string;
  timestamp: string; // RFC3339Nano
  v: number;
  schema_version?: number;
  data?: P;
}

export type EventType =
  | "SUPPLIER_CREATED"
  | "RETAILER_REGISTERED"
  | "DRIVER_CREATED"
  | "VEHICLE_CREATED"
  | "WAREHOUSE_CREATED"
  | "WAREHOUSE_SUPPLY_REQUEST_OPENED"
  | "WAREHOUSE_DISPATCH_LOCK_CHANGED"
  | "FACTORY_CREATED"
  | "ORDER_CREATED"
  | "ORDER_STATUS_CHANGED"
  | "ORDER_VALIDATION_FAILED"
  | "ORDER_ASSIGNED"
  | "ORDER_REASSIGNED"
  | "ORDER_FINALIZED"
  | "AI_RECOMMENDATION_CREATED"
  | "AI_RECOMMENDATION_DECIDED"
  | "ROUTE_CREATED"
  | "MANIFEST_DRAFT_CREATED"
  | "MANIFEST_LOADING_STARTED"
  | "MANIFEST_ORDER_INJECTED"
  | "MANIFEST_ORDER_EXCEPTION"
  | "MANIFEST_DLQ_ESCALATION"
  | "MANIFEST_REBALANCED"
  | "MANIFEST_CANCELLED"
  | "MANIFEST_SEALED"
  | "MANIFEST_DISPATCHED"
  | "MANIFEST_COMPLETED"
  | "PAYMENT_CLEARED"
  | "PAYMENT_REQUIRED"
  | "SETTLEMENT_REQUIRED"
  | "DELIVERY_SESSION_UPDATED"
  | "DELIVERY_DISPUTED"
  | "DRIVER_AVAILABILITY_CHANGED"
  | "DRIVER_LOCATION_UPDATED"
  | "SHOP_CLOSED"
  | "SHOP_CLOSED_RESPONSE"
  | "CART_SYNC_UPDATED"
  | "PROMOTION_CHANGED"
  | "RETAILER_PRICE_OVERRIDE"
  | "INVENTORY_SYNC_COMPLETE"
  | "COMMAND_DISPATCHED"
  | "COMMAND_RECEIVED"
  | "COMMAND_SETTLED"
  | "SYSTEM_APP_OUTDATED";

// ── Event payloads ──────────────────────────────────────────────────────────
export interface SupplierCreated {
  supplier_id: SupplierId;
  name: string;
  country: string;
  currency: Iso4217;
}

export interface RetailerRegistered {
  retailer_id: RetailerId;
  phone: string;
  name?: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
}

export interface RetailerProfileResponse {
  retailer_id: RetailerId;
  id: RetailerId;
  supplier_id?: SupplierId;
  name: string;
  company?: string;
  phone?: string;
  location?: string;
  country_code?: string;
  status?: string;
  h3_cell?: H3Cell;
  lat?: number;
  lng?: number;
  receiving_window_open?: string;
  receiving_window_close?: string;
  created_at?: string;
  updated_at?: string;
}

export interface RetailerProfileUpdateRequest {
  name?: string;
  company?: string;
  phone?: string;
  location?: string;
  country_code?: string;
  receiving_window_open?: string;
  receiving_window_close?: string;
}

export interface DriverCreated extends HomeNode {
  driver_id: DriverId;
  supplier_id: SupplierId;
  phone: string;
  name?: string;
}

export interface VehicleCreated extends HomeNode {
  vehicle_id: VehicleId;
  supplier_id: SupplierId;
  plate: string;
  max_vu?: number;
}

export interface WarehouseCreated {
  warehouse_id: WarehouseId;
  supplier_id: SupplierId;
  name?: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
  max_vu?: number;
}

export interface WarehouseSupplyRequestOpened {
  supplier_id: SupplierId;
  warehouse_id?: WarehouseId;
  request_id: string;
  status: string;
  state?: string;
  requested_by?: string;
  coverage_start_date?: string;
  coverage_days?: number;
  projected_units?: number;
  committed_units?: number;
  pending_confirmation_units?: number;
  timestamp: string;
}

export interface WarehouseDispatchLockChanged {
  supplier_id: SupplierId;
  warehouse_id?: WarehouseId;
  lock_id: string;
  entity_type?: string;
  entity_id?: string;
  reason?: string;
  action: "ACQUIRED" | "RELEASED";
  timestamp: string;
}

export interface FactoryCreated {
  factory_id: FactoryId;
  supplier_id: SupplierId;
  name?: string;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
}

export interface OrderCreated {
  order_id: OrderId;
  retailer_id: RetailerId;
  supplier_id: SupplierId;
  warehouse_id?: WarehouseId;
  status: OrderStatus;
  order_source?: OrderSource;
  confirmation_status?: OrderConfirmationStatus;
  requested_delivery_date?: string;
  line_items?: RetailerOrderLineItem[];
  total_minor: number;
  currency: Iso4217;
  h3_cell: H3Cell;
  lat: number;
  lng: number;
  receiving_window_open?: string;
  receiving_window_close?: string;
}

export interface OrderStatusChanged {
  order_id: OrderId;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  previous_status: OrderStatus;
  status: OrderStatus;
  reason?: string;
  actor_role?: Role;
  actor_id?: string;
  order_source?: OrderSource;
  confirmation_status?: OrderConfirmationStatus;
  requested_delivery_date?: string;
  version?: number;
  total_minor?: number;
  currency?: Iso4217;
}

export interface OrderValidationFailed {
  order_id: OrderId;
  reason: string;
}

export interface OrderAssigned {
  order_id: OrderId;
  driver_id: DriverId;
  route_id: RouteId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  warehouse_id?: WarehouseId;
  vehicle_id?: VehicleId;
  manifest_id?: ManifestId;
}

export interface OrderReassigned {
  order_id: OrderId;
  from_driver_id: DriverId;
  to_driver_id: DriverId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  warehouse_id?: WarehouseId;
  from_route_id?: RouteId;
  to_route_id?: RouteId;
  vehicle_id?: VehicleId;
  manifest_id?: ManifestId;
}

export interface AssignOrderRequest {
  driver_id: DriverId;
  route_id: RouteId;
  vehicle_id?: VehicleId;
  manifest_id?: ManifestId;
}

export interface AssignOrderResponse {
  order_id: OrderId;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  driver_id: DriverId;
  route_id: RouteId;
  vehicle_id?: VehicleId;
  manifest_id?: ManifestId;
  event_type: "ORDER_ASSIGNED" | "ORDER_REASSIGNED";
  version: number;
  updated_at: string;
  no_change?: boolean;
}

export interface OrderStatusPatchRequest {
  status: string;
  reason?: string;
}

export interface OrderStatusPatchResponse {
  order_id: OrderId;
  previous_status: string;
  status: string;
  version: number;
  updated_at: string;
  event_type: string;
}

export interface RetailerTrackingLineItem {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  line_total: number;
}

export interface RetailerOrderLineItem {
  sku: string;
  name?: string;
  quantity: number;
  unit_price_minor: number;
}

export interface RetailerTrackingLocation {
  driver_id: DriverId;
  supplier_id: SupplierId;
  lat: number;
  lng: number;
  latitude: number;
  longitude: number;
  velocity?: number;
  heading?: number;
  reported_at: string;
  received_at: string;
  stale_after_seconds: number;
}

export interface RetailerTrackingOrder {
  order_id: OrderId;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  warehouse_id?: WarehouseId;
  driver_id?: DriverId;
  vehicle_id?: VehicleId;
  route_id?: RouteId;
  manifest_id?: ManifestId;
  status: OrderStatus;
  tracking_status: "assigned" | "unassigned";
  total_minor: number;
  currency: Iso4217;
  live_location_available: boolean;
  driver_location?: RetailerTrackingLocation;
  payment_evidence?: RetailerTrackingPaymentEvidence;
  receipt_dossier?: RetailerTrackingReceiptDossier;
  created_at: string;
  updated_at: string;
  items: RetailerTrackingLineItem[];
}

export interface RetailerTrackingPaymentEvidence {
  entry_type: string;
  gateway: string;
  amount_minor: number;
  currency: Iso4217;
  reference_id?: string;
  occurred_at: string;
}

export interface RetailerTrackingReceiptPaymentRecord extends PaymentLedgerEntry {}

export interface RetailerTrackingReceiptGatewayWebhook {
  webhook_id: string;
  session_id?: SessionId;
  gateway: string;
  transaction_id: string;
  status: string;
  amount_minor: number;
  currency: Iso4217;
  signature_valid: boolean;
  received_at: string;
}

export interface RetailerTrackingReceiptDeliveryProof {
  proof_id: string;
  proof_type: string;
  latitude?: number;
  longitude?: number;
  distance_m?: number;
  qr_token_hash_present: boolean;
  scanned_token_hash_present: boolean;
  captured_at: string;
}

export interface RetailerTrackingReceiptChargebackRecord {
  chargeback_id: string;
  gateway: string;
  amount_minor: number;
  currency: Iso4217;
  created_at: string;
}

export interface RetailerTrackingReceiptReversalRecord {
  reversal_id: string;
  session_id?: SessionId;
  gateway: string;
  amount_minor: number;
  currency: Iso4217;
  ledger_entry_id?: string;
  created_at: string;
}

export interface RetailerTrackingReceiptProofStatus {
  payment_timeline_available: boolean;
  gateway_webhooks_available: boolean;
  delivery_proof_available: boolean;
  missing_artifacts: string[];
}

export interface RetailerTrackingReceiptDossier {
  session_id?: SessionId;
  payment_timeline: RetailerTrackingReceiptPaymentRecord[];
  gateway_webhooks: RetailerTrackingReceiptGatewayWebhook[];
  delivery_proofs: RetailerTrackingReceiptDeliveryProof[];
  chargebacks: RetailerTrackingReceiptChargebackRecord[];
  reversals: RetailerTrackingReceiptReversalRecord[];
  proof_status: RetailerTrackingReceiptProofStatus;
}

export type RetailerTrackingEventType = "ORDER_CREATED" | "ORDER_STATUS_SNAPSHOT";

export interface RetailerTrackingEvent {
  event_type: RetailerTrackingEventType;
  order_id: OrderId;
  status?: OrderStatus;
  occurred_at: string;
  derived: boolean;
  source: "ORDER_ROW";
}

export interface RetailerTrackingResponse {
  status: "idle" | "active";
  orders: RetailerTrackingOrder[];
  recent_receipts: RetailerTrackingOrder[];
  events: RetailerTrackingEvent[];
}

export interface RetailerActiveFulfillmentResponse {
  status: "idle" | "active";
  fulfillments: RetailerTrackingOrder[];
}

export interface RetailerPendingPaymentsResponse {
  status: "idle" | "pending";
  count: number;
  pending: RetailerTrackingOrder[];
}

export interface RetailerAIPrediction {
  order_id: OrderId;
  order_source: OrderSource;
  confirmation_status: OrderConfirmationStatus;
  requested_delivery_date?: string;
  auto_confirm_at?: string;
  total_minor: number;
  currency: Iso4217;
  derived_from_order_id?: OrderId;
  updated_at: string;
  line_items: RetailerOrderLineItem[];
}

export interface RetailerAIPredictionsResponse {
  items: RetailerAIPrediction[];
}

export interface RetailerOrderLifecycleResponse {
  order_id: OrderId;
  status: OrderStatus;
  order_source: OrderSource;
  confirmation_status: OrderConfirmationStatus;
  requested_delivery_date?: string;
  auto_confirm_at?: string;
  decision_at?: string;
  decision_by?: string;
  total_minor: number;
  currency: Iso4217;
  version: number;
  updated_at: string;
  created?: boolean;
}

export interface ConfirmAIOrderRequest {
  order_id: OrderId;
  requested_delivery_date?: string;
  line_items?: RetailerOrderLineItem[];
}

export interface RejectAIOrderRequest {
  order_id: OrderId;
  reason?: string;
}

export interface EditPreorderRequest {
  order_id: OrderId;
  requested_delivery_date: string;
  line_items: RetailerOrderLineItem[];
}

export interface ConfirmPreorderRequest {
  order_id: OrderId;
}

export interface WarehouseDemandForecastDay {
  date: string;
  projected_units: number;
  projected_revenue: number;
  committed_units: number;
  pending_confirmation_units: number;
  currency: Iso4217;
}

export interface WarehouseDemandForecastProductSources {
  incoming_orders: number;
  ai_prediction: number;
  pre_orders: number;
  burn_rate: number;
}

export interface WarehouseDemandForecastProduct {
  product_id: string;
  product_name: string;
  current_stock: number;
  recommended_qty: number;
  days_until_stockout: number;
  priority: string;
  unit: string;
  sources: WarehouseDemandForecastProductSources;
}

export interface WarehouseDemandForecastResponse {
  warehouse_id: WarehouseId;
  forecast_days?: number;
  generated_at?: string;
  series: WarehouseDemandForecastDay[];
  /** Portal parity: product-level 4-source breakdown (pegasus warehouse-portal). */
  products?: WarehouseDemandForecastProduct[];
}

export interface WarehouseOpsDashboardResponse {
  supplier_id?: SupplierId;
  inventory_skus: number;
  orders_open: number;
  dispatch_locks: number;
  supply_requests: number;
  updated_at: string;
}

export interface WarehouseInventoryRow {
  sku: string;
  product_name: string;
  quantity: number;
  updated_at: string;
}

export interface WarehouseInventoryResponse {
  items: WarehouseInventoryRow[];
}

export interface WarehouseOrderRow {
  order_id: OrderId;
  retailer_id: RetailerId;
  status: string;
  total_minor: number;
  currency: Iso4217;
  updated_at: string;
}

export interface WarehouseOrdersResponse {
  orders: WarehouseOrderRow[];
}

/** Legacy compact shape; prefer WarehouseDispatchPreview from GET/POST dispatch/preview. */
export interface WarehouseDispatchPreviewResponse {
  status: string;
  recommended_manifests: number;
  estimated_eta_minutes: number;
}

export interface WarehouseSupplyRequest {
  request_id: string;
  supplier_id?: SupplierId;
  warehouse_id?: WarehouseId;
  factory_id?: string;
  status: string;
  state?: string;
  priority?: string;
  requested_delivery_date?: string;
  total_volume_vu?: number;
  notes?: string;
  transfer_order_id?: string;
  created_by?: string;
  requested_by?: string;
  coverage_start_date?: string;
  coverage_days?: number;
  projected_units?: number;
  committed_units?: number;
  pending_confirmation_units?: number;
  item_count?: number;
  items?: WarehouseSupplyRequestItem[];
  created_at: string;
  updated_at?: string;
}

export interface WarehouseSupplyRequestsResponse {
  requests: WarehouseSupplyRequest[];
  supply_requests?: WarehouseSupplyRequest[];
}

export interface WarehouseDispatchLock {
  lock_id: string;
  entity_type?: string;
  entity_id?: string;
  reason?: string;
  created_at?: string;
  supplier_id?: SupplierId;
  warehouse_id?: WarehouseId;
  factory_id?: string;
  lock_type?: string;
  locked_at?: string;
  unlocked_at?: string;
  locked_by?: string;
}

export interface WarehouseDispatchLocksResponse {
  locks: WarehouseDispatchLock[];
}

export interface WarehouseDispatchLockAcquireRequest {
  entity_type: string;
  entity_id: string;
  reason: string;
}

export interface WarehouseDispatchLockReleaseResponse {
  status: string;
  lock_id: string;
}

export interface WarehouseEmergencyTransferRequest {
  total_volume_vu: number;
  notes?: string;
}

export interface WarehouseForceReceiveRequest {
  factory_id?: string;
  total_volume_vu: number;
  notes?: string;
}

export interface WarehouseTransferMutationResponse {
  transfer_id: string;
  state: string;
  notes?: string;
}

export interface WarehouseReplenishmentInsight {
  id: string;
  warehouse_id: WarehouseId;
  warehouse_name: string;
  product_id: string;
  product_name: string;
  urgency: string;
  current_stock: number;
  avg_daily_velocity: number;
  days_until_stockout: number;
  reorder_quantity: number;
  status: string;
  created_at: string;
}

export interface WarehouseReplenishmentInsightsResponse {
  insights: WarehouseReplenishmentInsight[];
  data?: WarehouseReplenishmentInsight[];
}

export interface WarehouseReplenishmentInsightActionResponse {
  insight_id: string;
  status: string;
  transfer_id?: string;
}

export interface WarehouseDispatchSettingsResponse {
  warehouse_id: WarehouseId;
  auto_dispatch_enabled: boolean;
}

export interface WarehouseDispatchSettingsPatchRequest {
  auto_dispatch_enabled: boolean;
}

export interface FactoryAnalyticsOverviewResponse {
  daily_activity: unknown[];
  transfers_total: number;
  manifests_active: number;
  exception_queue: number;
  avg_lead_time_mins: number;
}

export interface WarehouseOpsFinancialsResponse {
  warehouse_id: WarehouseId;
  period: string;
  currency: Iso4217;
  total_revenue: number;
  completed_orders: number;
  avg_order_value: number;
  gateway_breakdown: unknown[];
  daily_revenue: unknown[];
  platform_fee: number;
  net_payout: number;
  cash_pending: number;
  cash_collected: number;
}

export interface WarehouseOrderMutationRequest {
  reason?: string;
}

export interface WarehouseOrderMutationResponse {
  order_id: OrderId;
  status: string;
}

export interface OrderFinalized {
  order_id: OrderId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  total: Money;
  amount_minor?: number;
  currency?: Iso4217;
  status?: OrderStatus;
  fee_amount?: number;
  net_payout_amount?: number;
}

export interface PaymentRequired {
  order_id: OrderId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  amount: Money;
  amount_minor?: number;
  currency?: Iso4217;
  payment_method?: PaymentGateway | string;
  status?: OrderStatus;
}

export interface PaymentCleared {
  order_id: OrderId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  amount: Money;
  amount_minor?: number;
  currency?: Iso4217;
  gateway?: PaymentGateway;
  payment_method?: PaymentGateway | string;
  status?: OrderStatus;
  provider_reference?: string;
}

export interface SettlementRequired {
  order_id: OrderId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  session_id?: SessionId;
  amount: Money;
  amount_minor?: number;
  currency?: Iso4217;
  payment_method?: PaymentGateway | string;
  status?: OrderStatus;
}

export interface DeliverySessionUpdated {
  session_id: SessionId;
  order_id: OrderId;
  amount?: number;
  original_amount?: number;
  adjusted_amount?: number;
  currency?: Iso4217;
}

export interface DriverAvailabilityChanged extends HomeNode {
  driver_id: DriverId;
  supplier_id?: SupplierId;
  available: boolean;
  on_shift?: boolean;
  reason?: string;
}

export interface DriverLocationUpdated {
  driver_id: DriverId;
  supplier_id: SupplierId;
  lat: number;
  lng: number;
  latitude: number;
  longitude: number;
  velocity?: number;
  heading?: number;
  reported_at: string;
  received_at: string;
  stale_after_seconds: number;
}

export type DriverAvailabilityChangedEvent = WsEventEnvelope<"DRIVER_AVAILABILITY_CHANGED"> & DriverAvailabilityChanged;

export type DriverLocationUpdatedEvent = WsEventEnvelope<"DRIVER_LOCATION_UPDATED", DriverLocationUpdated>;

export interface ManifestSealed {
  manifest_id: ManifestId;
  route_id: RouteId;
  driver_id: DriverId;
  vehicle_id?: VehicleId;
  order_count?: number;
}

export interface ManifestOrderInjected {
  manifest_id: ManifestId;
  order_id: OrderId;
  total_volume_vu?: number;
  stop_count?: number;
}

export interface ManifestOrderException {
  manifest_id: ManifestId;
  order_id?: OrderId;
  transfer_id?: string;
  reason: string;
  attempt_count?: number;
  escalated?: boolean;
  exception_id?: string;
  metadata?: string;
}

export interface ManifestRebalanced {
  manifest_id: ManifestId;
  transfer_id?: string;
  order_id?: OrderId;
  from_driver_id?: DriverId;
  to_driver_id?: DriverId;
  from_vehicle_id?: VehicleId;
  to_vehicle_id?: VehicleId;
  from_route_id?: RouteId | string;
  to_route_id?: RouteId | string;
  from_manifest_id?: ManifestId;
  to_manifest_id?: ManifestId;
  depth?: number;
  reason?: string;
}

export interface ManifestCancelled {
  manifest_id: ManifestId;
  reason?: string;
}

export type CommandState = "INITIATED" | "DISPATCHED" | "RECEIVED" | "SETTLED";

export interface CommandLifecycle {
  command_id: CommandId;
  command_state: CommandState;
  target_role?: Role;
  target_id?: string;
}

// ── Discriminated WS event union ────────────────────────────────────────────
export type WsEvent =
  | WsEventEnvelope<"SUPPLIER_CREATED", SupplierCreated>
  | WsEventEnvelope<"RETAILER_REGISTERED", RetailerRegistered>
  | WsEventEnvelope<"DRIVER_CREATED", DriverCreated>
  | WsEventEnvelope<"VEHICLE_CREATED", VehicleCreated>
  | WsEventEnvelope<"WAREHOUSE_CREATED", WarehouseCreated>
  | WsEventEnvelope<"WAREHOUSE_SUPPLY_REQUEST_OPENED", WarehouseSupplyRequestOpened>
  | WsEventEnvelope<"WAREHOUSE_DISPATCH_LOCK_CHANGED", WarehouseDispatchLockChanged>
  | WsEventEnvelope<"FACTORY_CREATED", FactoryCreated>
  | WsEventEnvelope<"ORDER_CREATED", OrderCreated>
  | WsEventEnvelope<"ORDER_STATUS_CHANGED", OrderStatusChanged>
  | WsEventEnvelope<"ORDER_VALIDATION_FAILED", OrderValidationFailed>
  | WsEventEnvelope<"ORDER_ASSIGNED", OrderAssigned>
  | WsEventEnvelope<"ORDER_REASSIGNED", OrderReassigned>
  | WsEventEnvelope<"ORDER_FINALIZED", OrderFinalized>
  | WsEventEnvelope<"PAYMENT_REQUIRED", PaymentRequired>
  | WsEventEnvelope<"PAYMENT_CLEARED", PaymentCleared>
  | WsEventEnvelope<"SETTLEMENT_REQUIRED", SettlementRequired>
  | WsEventEnvelope<"DELIVERY_SESSION_UPDATED", DeliverySessionUpdated>
  | DriverAvailabilityChangedEvent
  | DriverLocationUpdatedEvent
  | WsEventEnvelope<"MANIFEST_ORDER_INJECTED", ManifestOrderInjected>
  | WsEventEnvelope<"MANIFEST_ORDER_EXCEPTION", ManifestOrderException>
  | WsEventEnvelope<"MANIFEST_DLQ_ESCALATION", ManifestOrderException>
  | WsEventEnvelope<"MANIFEST_REBALANCED", ManifestRebalanced>
  | WsEventEnvelope<"MANIFEST_CANCELLED", ManifestCancelled>
  | WsEventEnvelope<"MANIFEST_SEALED", ManifestSealed>
  | WsEventEnvelope<"COMMAND_DISPATCHED", CommandLifecycle>
  | WsEventEnvelope<"COMMAND_RECEIVED", CommandLifecycle>
  | WsEventEnvelope<"COMMAND_SETTLED", CommandLifecycle>
  | WsEventEnvelope<"SYSTEM_APP_OUTDATED", { minimum_version: string }>;

// ── Warehouse portal / desktop (fleet, supply detail, live WS) ──

export type WarehouseStaffRole = "WAREHOUSE_STAFF" | "PAYLOADER";

export interface WarehouseStaffMember {
  worker_id: string;
  name: string;
  phone: string;
  role: WarehouseStaffRole | string;
  is_active: boolean;
  created_at?: string;
}

export interface WarehouseStaffListResponse {
  staff: WarehouseStaffMember[];
}

export interface CreateWarehouseStaffRequest {
  name: string;
  phone: string;
  role: WarehouseStaffRole;
  pin?: string;
}

export interface CreateWarehouseStaffResponse {
  worker_id: string;
  name?: string;
  role?: WarehouseStaffRole | string;
  pin: string;
}

export type WarehouseVehicleUnavailableReason =
  | "MAINTENANCE"
  | "TRUCK_DAMAGED"
  | "REGULATORY_HOLD"
  | "MANUAL_HOLD"
  | "OTHER"
  | string;

export interface WarehouseFleetDriver {
  driver_id: string;
  name: string;
  phone: string;
  driver_type?: string;
  vehicle_type?: string;
  license_plate?: string;
  is_active: boolean;
  on_shift?: boolean;
  truck_status: string;
  unavailable_reason?: string;
  unavailable_note?: string;
  created_at?: string;
  vehicle_id?: string;
  vehicle_class?: string;
  max_volume_vu?: number;
  vehicle_is_active?: boolean;
  vehicle_unavailable_reason?: WarehouseVehicleUnavailableReason;
}

export interface WarehouseFleetDriverListResponse {
  drivers: WarehouseFleetDriver[];
}

export interface WarehouseAssignVehicleRequest {
  vehicle_id?: string;
}

export interface WarehouseAssignVehicleResponse {
  status: "ASSIGNED" | "UNASSIGNED" | string;
  driver_id: string;
  vehicle_id?: string;
  previously_assigned_driver?: string;
}

export interface WarehouseFleetVehicle {
  vehicle_id: string;
  vehicle_class: string;
  class_label?: string;
  label: string;
  license_plate: string;
  max_volume_vu?: number;
  capacity_vu: number;
  is_active: boolean;
  status: string;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
  created_at?: string;
  assigned_driver_id?: string;
  assigned_driver_name?: string;
  driver_truck_status?: string;
}

export interface WarehouseFleetVehicleListResponse {
  vehicles: WarehouseFleetVehicle[];
  total?: number;
}

export interface WarehouseFleetVehicleDetailResponse {
  vehicle: WarehouseFleetVehicle;
}

export interface WarehouseUpdateVehicleRequest {
  label?: string;
  license_plate?: string;
  is_active?: boolean;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
}

export interface WarehouseVehicleMutationResponse {
  status: string;
  vehicle_id: string;
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
}

export interface WarehouseDispatchOrder {
  order_id: string;
  retailer_name: string;
  total_uzs: number;
  item_count: number;
  volume_vu?: number;
  created_at?: string;
}

export interface WarehouseDispatchDriver {
  driver_id: string;
  name: string;
  phone?: string;
  truck_status: string;
  vehicle_id?: string;
  vehicle_class?: string;
  max_volume_vu?: number;
  used_volume_vu?: number;
  free_volume_vu?: number;
  active_manifest_id?: string;
  vehicle_label?: string;
}

export interface WarehouseUnavailableDispatchDriver extends WarehouseDispatchDriver {
  unavailable_reason?: WarehouseVehicleUnavailableReason;
  unavailable_note?: string;
}

export interface WarehouseDispatchProposedStop {
  order_id: string;
  retailer_id?: string;
  retailer_name?: string;
  volume_vu?: number;
  sequence?: number;
}

export interface WarehouseDispatchProposedRoute {
  driver_id?: string;
  driver_name?: string;
  order_ids?: string[];
  stops?: WarehouseDispatchProposedStop[];
  volume_vu?: number;
  loaded_volume?: number;
  max_volume_vu?: number;
  max_volume?: number;
  util_pct?: number;
  stop_count?: number;
  route_geometry?: RouteGeometryWire;
}

export interface WarehouseDispatchPreview {
  orders?: WarehouseDispatchOrder[];
  undispatched_orders: WarehouseDispatchOrder[];
  drivers?: WarehouseDispatchDriver[];
  available_drivers: WarehouseDispatchDriver[];
  unavailable_drivers?: WarehouseUnavailableDispatchDriver[];
  pending_count?: number;
  available_driver_count?: number;
  preview_ready?: boolean;
  proposed_routes?: WarehouseDispatchProposedRoute[];
  optimizer_source?: string;
  optimizer_warnings?: string[];
  window_constrained_count?: number;
  fleet_effective_capacity_vu?: number;
  selected_orders_volume_vu?: number;
  plan_fingerprint?: string;
  plan_computed_at?: string;
  plan_stale?: boolean;
}

export interface WarehouseDispatchCapacityWarning {
  driver_id: string;
  loaded_vu: number;
  max_volume_vu: number;
  effective_max_vu: number;
  excess_vu?: number;
  suggested_unselect_order_ids?: string[];
  suggested_defer_order_ids?: string[];
  fleet_effective_capacity_vu?: number;
  requested_volume_vu?: number;
}

export interface WarehouseDispatchExecuteRoute {
  manifest_id?: string;
  route_id?: string;
  driver_id: string;
  vehicle_id?: string;
  order_ids: string[];
  volume_vu?: number;
  max_volume_vu?: number;
}

export interface WarehouseDispatchExecuteRequest {
  mode: "MANUAL" | "AUTO" | string;
  routes?: WarehouseDispatchExecuteRoute[];
  order_ids?: string[];
  force_capacity?: boolean;
  accept_partial?: boolean;
  plan_fingerprint?: string;
}

export interface WarehouseDispatchExecuteResponse {
  status: string;
  supplier_id: string;
  warehouse_id?: string;
  manifests_created?: number;
  orders_assigned?: number;
  optimizer_source?: string;
  warnings?: string[];
  capacity_warnings?: WarehouseDispatchCapacityWarning[];
  manifests?: WarehouseDispatchExecuteRoute[];
  orphan_order_ids?: string[];
}

export interface CatalogProduct {
  product_id: string;
  supplier_id: string;
  category_id: string;
  name: string;
  description?: string;
  image_url?: string;
  price_minor: number;
  currency: string;
  stock_quantity: number;
  unit: string;
  unit_volume_vu?: number;
  is_active: boolean;
  version: number;
  created_at?: string;
  updated_at?: string;
}

export interface WarehouseSupplyRequestItem {
  item_id: string;
  product_id: string;
  requested_quantity: number;
  shipped_quantity?: number;
  received_quantity?: number;
  recommended_qty: number;
  unit_volume_vu: number;
}

export interface WarehouseSupplyRequestDetail extends WarehouseSupplyRequest {
  demand_breakdown?: unknown;
  items: WarehouseSupplyRequestItem[];
  factory_id?: string;
  priority?: string;
  notes?: string;
  transfer_order_id?: string;
  linked_transfer_id?: string;
  created_by?: string;
}

export interface CreateWarehouseSupplyRequestResponse {
  request_id: string;
  state: string;
  priority: string;
  total_volume_vu: number;
  items_count: number;
}

export interface CreateWarehouseDispatchLockResponse {
  lock_id: string;
  lock_type: string;
  status: "LOCKED" | string;
}

export interface WarehouseCRMRetailer {
  retailer_id: string;
  business_name: string;
  total_orders: number;
  total_revenue: number;
  last_order_date?: string;
}

export interface WarehouseCRMListResponse {
  retailers: WarehouseCRMRetailer[];
}

export interface WarehouseReturnItem {
  line_item_id: string;
  order_id: string;
  product_name: string;
  quantity: number;
  status: string;
  updated_at?: string;
}

export interface WarehouseReturnListResponse {
  items: WarehouseReturnItem[];
}

export interface WarehouseTreasuryOverview {
  total_invoiced: number;
  total_paid: number;
  total_outstanding: number;
}

export interface WarehouseSupplyRequestUpdateEvent {
  type: "SUPPLY_REQUEST_UPDATE";
  warehouse_id: string;
  request_id: string;
  state: string;
  timestamp: string;
}

export interface WarehouseDispatchLockChangeEvent {
  type: "DISPATCH_LOCK_CHANGE";
  warehouse_id: string;
  lock_id: string;
  action: "ACQUIRED" | "RELEASED" | string;
  timestamp: string;
}

export interface WarehouseOutboxFailureEvent {
  type: "OUTBOX_FAILED";
  event_id: string;
  aggregate_id: string;
  topic: string;
  reason: string;
  timestamp: string;
}

export interface WarehouseInventorySyncCompleteEvent {
  type: "INVENTORY_SYNC_COMPLETE";
  supplier_id: string;
  warehouse_id?: string;
  session_id: string;
  rows_affected: number;
  affected_warehouses?: number;
  product_ids?: string[];
  source?: string;
  timestamp: string;
}

export type WarehouseLiveEvent =
  | WarehouseSupplyRequestUpdateEvent
  | WarehouseDispatchLockChangeEvent
  | WarehouseOutboxFailureEvent
  | WarehouseInventorySyncCompleteEvent;

// ── Notification inbox (GET /v1/user/notifications) ───────────────────────────
export interface NotificationInboxItem {
  id: string;
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel: string;
  read_at?: string | null;
  created_at: string;
}

export interface NotificationInboxResponse {
  notifications: NotificationInboxItem[];
  unread_count: number;
  limit?: number;
  offset?: number;
  has_more?: boolean;
}

export interface MarkNotificationsReadRequest {
  notification_ids?: string[];
  mark_all?: boolean;
}

import { DeliveryExpectation, DriverId, FactoryId, FiscalStatus, HomeNodeType, Iso4217, ManifestId, OrderId, OutOfStockPolicy, PaymentGateway, RetailerId, Role, RouteId, SessionId, SupplierId, VehicleId, WarehouseId } from "./primitives";

// ── Compliance / fiscal audit dashboard (Phase 1) ──

export interface ComplianceFiscalOpenRow {
  order_id: string;
  retailer_id: string;
  driver_id?: string;
  status: string;
  fiscal_status?: string;
  total_minor: number;
  currency: string;
  latest_fiscal_attempt_id?: string;
  updated_at: string;
}

export interface ComplianceForceCompleteRow {
  order_id: string;
  retailer_id: string;
  driver_id?: string;
  status: string;
  fiscal_status: string;
  reason_code?: string;
  actor_id?: string;
  attempt_id?: string;
  total_minor: number;
  currency: string;
  completed_at: string;
}

export interface ComplianceClaimMismatchRow {
  claim_id: string;
  order_id: string;
  retailer_id: string;
  claim_status: string;
  claim_amount_minor: number;
  order_total_minor: number;
  order_status: string;
  currency: string;
  mismatch_reason: string;
  created_at: string;
}

export interface ComplianceCreditFreezeRow {
  retailer_id: string;
  status: string;
  risk_tier?: string;
  credit_limit_minor: number;
  current_balance_minor: number;
  available_credit_minor: number;
  updated_at: string;
}

export interface ComplianceSummary {
  open_fiscal_count: number;
  force_complete_count: number;
  claim_mismatch_count: number;
  credit_freeze_count: number;
  generated_at: string;
}

export interface ComplianceDashboardResponse {
  summary: ComplianceSummary;
  open_fiscal: ComplianceFiscalOpenRow[];
  force_completes: ComplianceForceCompleteRow[];
  claim_mismatches: ComplianceClaimMismatchRow[];
  credit_freezes: ComplianceCreditFreezeRow[];
}

export type ComplianceExportBucket =
  | "all"
  | "open_fiscal"
  | "force_completes"
  | "claim_mismatches"
  | "credit_freezes";

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
  yard_manifests?: WarehouseYardManifest[];
  warehouse_id: string;
  fetched_at: string;
}

export interface WarehouseYardManifest {
  manifest_id: string;
  driver_name?: string;
  order_count: number;
  loading_started_at?: string;
  delivery_summary?: string;
  manifest_state: string;
}

/** Factory fleet live map — driver pins (geometry deferred until FactoryTruckManifests polyline columns). */
export type FactoryFleetLiveRoute = Omit<SupplierFleetLiveRoute, "route_geometry"> & {
  order_count?: number;
  loading_started_at?: string;
  delivery_summary?: string;
};

export interface FactoryFleetLiveMapResponse {
  routes: FactoryFleetLiveRoute[];
  yard_manifests?: WarehouseYardManifest[];
  factory_id: string;
  fetched_at: string;
}

export interface WarehouseDispatchRun {
  run_id: string;
  warehouse_id: string;
  supplier_id: string;
  actor_id?: string;
  mode: string;
  status: string;
  manifest_count: number;
  orders_assigned: number;
  warnings?: string[];
  manifest_ids?: string[];
  created_at: string;
}

export interface WarehouseDispatchRunsResponse {
  runs: WarehouseDispatchRun[];
}

export interface WarehouseOpsBoardOrder {
  order_id: string;
  status: string;
  retailer_id?: string;
  total_minor: number;
  delivery_expectation?: DeliveryExpectation;
}

export interface WarehouseOpsBoardResponse {
  date: string;
  warehouse_id: string;
  preorders: WarehouseOpsBoardOrder[];
  deliver_before: WarehouseOpsBoardOrder[];
  stock_commitments: { sku_id: string; committed_qty: number }[];
  draft_manifests: { manifest_id: string; state: string; stop_count: number; driver_name?: string }[];
  loading_manifests: { manifest_id: string; state: string; stop_count: number; driver_name?: string }[];
  fetched_at: string;
}

export interface WarehouseOpsExceptionRow {
  exception_id?: string;
  kind: string;
  order_id?: string;
  manifest_id?: string;
  reason?: string;
  status?: string;
  updated_at?: string;
  delivery_expectation?: DeliveryExpectation;
}

export interface WarehouseOpsExceptionsResponse {
  exceptions: WarehouseOpsExceptionRow[];
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
  /** GS1 Global Location Number (13 digits). */
  gln?: string;
  /** GS1 company prefix (7–10 digits) used to mint SSCC-18 at seal. */
  gs1_company_prefix?: string;
  updated_at: string;
}

export interface SupplierProfileUpdateRequest {
  legal_name?: string;
  contact_name?: string;
  email?: string;
  phone?: string;
  categories?: string[];
  gln?: string;
  gs1_company_prefix?: string;
}

export interface ManifestShipUnit {
  manifest_id: string;
  ship_unit_id: string;
  sscc: string;
  order_id: string;
  sequence: number;
  gtin?: string;
  created_at: string;
}

export interface ManifestShipUnitsResponse {
  manifest_id: string;
  ship_units: ManifestShipUnit[];
}

export interface SupplierTopologyInventorySeed {
  product_id: string;
  quantity: number;
}

export interface SupplierTopologyCoverageCity {
  name: string;
  lat: number;
  lng: number;
}

export interface SupplierTopologyWarehouseInput {
  warehouse_id?: WarehouseId;
  name: string;
  lat: number;
  lng: number;
  address?: string;
  place_id?: string;
  coverage_radius_km?: number;
  is_active?: boolean;
  is_on_shift?: boolean;
  transfer_mode?: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id?: FactoryId;
  primary_factory_id?: FactoryId;
  secondary_factory_id?: FactoryId;
  assigned_factory_ids?: FactoryId[];
  country_code?: string;
  coverage_cities?: SupplierTopologyCoverageCity[];
  default_out_of_stock_policy?: OutOfStockPolicy;
  operating_schedule?: Record<string, unknown>;
  initial_inventory?: SupplierTopologyInventorySeed[];
}

export interface SupplierTopologyFactoryInput {
  factory_id?: FactoryId;
  name: string;
  lat: number;
  lng: number;
  address?: string;
  place_id?: string;
  country_code?: string;
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
  address?: string;
  place_id?: string;
  coverage_radius_km: number;
  is_active: boolean;
  is_on_shift: boolean;
  transfer_mode?: "TRUCK" | "INTERNAL";
  co_locate_with_factory_id?: FactoryId;
  primary_factory_id?: FactoryId;
  secondary_factory_id?: FactoryId;
  assigned_factory_ids?: FactoryId[];
  country_code?: string;
  coverage_cities?: SupplierTopologyCoverageCity[];
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
  address?: string;
  place_id?: string;
  country_code?: string;
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

export type ServicePinTargetType = "LOCATION" | "RETAILER" | "REGION" | "CITY";
export type CoverageMode = "PINNED" | "CITY_CELLS" | "COUNTRY_CLOSEST";

export interface ServicePin {
  warehouse_id?: string;
  target_type: ServicePinTargetType;
  target_id: string;
  priority?: number;
}

export interface WarehouseCoverageResponse {
  warehouse_id: string;
  mode: CoverageMode;
  cities?: SupplierTopologyCoverageCity[];
  pins?: ServicePin[];
  country_code?: string;
}

export interface WarehouseOpsSupplyFactoryResponse {
  warehouse_id: string;
  factory_id?: string;
  transfer_mode?: string;
  source: "engine";
  country_code: string;
}

export interface WarehouseOpsLocationResponse {
  warehouse_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
  gln?: string;
  country_code?: string;
  pack_country_code?: string;
  currency_code?: string;
  timezone?: string;
  updated_at?: string;
}

export interface WarehouseOpsPaymentConfigResponse {
  gateways?: Array<{
    gateway_name: string;
    provider: string;
    is_active: boolean;
    mode: string;
    status?: string;
    selectable?: boolean;
  }>;
  selected_gateways?: string[];
  catalog: PSPListing[];
  currency_code: string;
  market_code?: string;
  payment_acceptor?: string;
  payment_config_id?: string;
}

export interface WarehouseCoverageRequest {
  cities: SupplierTopologyCoverageCity[];
  pins: Array<Pick<ServicePin, "target_type" | "target_id" | "priority">>;
}

export interface WarehousePinsResponse {
  warehouse_id: string;
  mode: CoverageMode;
  pins?: ServicePin[];
}

export interface SupplierRegion {
  region_id: string;
  name: string;
  country_code: string;
}

export interface SupplierRegionsResponse {
  items: SupplierRegion[];
}

export interface SupplierRegionsReplaceRequest {
  items: Array<{ region_id?: string; name: string; country_code?: string }>;
}

export interface PSPListing {
  code: string;
  status: string;
  selectable: boolean;
  national_cards?: boolean;
}

export interface SupplierPaymentCatalogResponse {
  currency_code: string;
  market_code?: string;
  catalog: PSPListing[];
}

/** GET /v1/retailer/payment-catalog — same pack ∩ registry shape as supplier. */
export type RetailerPaymentCatalogResponse = SupplierPaymentCatalogResponse;

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

export interface PromotionTier {
  min_quantity: number;
  discount_bps: number;
}

export interface SupplierPromotion {
  promotion_id: string;
  supplier_id: SupplierId;
  campaign_id?: string;
  name: string;
  description?: string;
  tiers: PromotionTier[];
  discount_bps?: number;
  min_line_quantity?: number;
  scope_type: SupplierPromotionScopeType;
  scope_product_id?: string;
  scope_category_id?: string;
  retailer_scope: SupplierPromotionRetailerScope;
  retailer_ids?: RetailerId[];
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
  tiers?: PromotionTier[];
  discount_bps?: number;
  min_line_quantity?: number;
  scope_type: SupplierPromotionScopeType;
  scope_product_id?: string;
  scope_category_id?: string;
  retailer_scope?: SupplierPromotionRetailerScope;
  retailer_ids?: RetailerId[];
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
  /** ADR-009 denorm rollup when present: NONE | PENDING | SUCCESS | FAILED | FORCE_SKIPPED */
  fiscal_status?: FiscalStatus | string;
  latest_fiscal_receipt_id?: string;
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
    retailer_name?: string;
    warehouse_id?: string;
    total_minor: number;
    currency: string;
    volume_vu?: number;
  }>;
  available_drivers: Array<{
    driver_id: string;
    name: string;
    vehicle_id?: string;
    truck_status?: string;
    max_volume_vu?: number;
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
  plan_fingerprint?: string;
  warehouse_plan_fingerprint?: string;
  plan_fingerprint_mismatch?: boolean;
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
  force_capacity?: boolean;
  routes?: Array<{
    driver_id: string;
    order_ids: string[];
  }>;
}

export interface SupplierDispatchCapacityWarning {
  driver_id: string;
  loaded_vu: number;
  max_volume_vu: number;
  effective_max_vu: number;
  excess_vu?: number;
  suggested_unselect_order_ids?: string[];
}

export interface SupplierDispatchExecuteResponse {
  status: string;
  supplier_id: string;
  warehouse_id?: string;
  manifests_created?: number;
  orders_assigned?: number;
  optimizer_source?: string;
  warnings?: string[];
  capacity_warnings?: SupplierDispatchCapacityWarning[];
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

/** Theatre #13 FX rate row (admin/supplier list). */
export interface FxRateRow {
  rate_id: string;
  base_currency: Iso4217;
  quote_currency: Iso4217;
  rate_scaled: number;
  scale: number;
  effective_at: string;
  source: string;
}

export interface FxRatesListResponse {
  rates: FxRateRow[];
}

export interface FxRateUpsertInput {
  base_currency: Iso4217;
  quote_currency: Iso4217;
  rate_scaled: number;
  scale?: number;
  effective_at?: string;
  source?: string;
}

export interface FxRateUpsertResponse {
  ok: boolean;
  rate: FxRateRow;
}

export interface SettlementAuthorityResponse {
  items: SettlementAuthorityRow[];
  count: number;
  group_limit: number;
  supplier_id: SupplierId | "";
  entry_count_total: number;
  totals_by_currency: SettlementCurrencyTotal[];
  /** Theatre #13 Wave 2: display-only rollup in operating currency. */
  operating_currency?: Iso4217;
  operating_currency_total_minor?: number;
  operating_conversion_partial?: boolean;
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


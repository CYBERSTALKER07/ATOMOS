import { DeliveryExpectation, Iso4217, SupplierId, WarehouseId } from "./primitives";
import { SupplierActivityEvent } from "./compliance";

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
  deliveries_completed_today?: number;
  deliveries_attempted_today?: number;
  manifests_by_state?: Record<string, number>;
  fleet_vu_used?: number;
  fleet_vu_total?: number;
  fleet_vu_available?: boolean;
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
	baseline_source?: "ai_recommendations" | "demand_forecast_baseline" | "mixed" | string;
	granularity?: "macro" | "regional" | "micro" | string;
	confidence?: ForecastConfidence;
}

export interface DemandSignal {
	signalId: string;
	retailerId?: string;
	productId?: string;
	type: string; // HOLIDAY, WEATHER, EVENT, PROMO
	scope: string; // GLOBAL, REGION, CITY, RETAILER, RETAILER_SKU
	startDate: string; // "2006-01-02"
	endDate: string;
	multiplier: number;
	description?: string;
	createdBy: string;
	createdAt: string;
}

/** Kafka DEMAND_SIGNAL flywheel payload (STORE_POS sell-through). Distinct from planning DemandSignal REST rows. */
export interface DemandSignalEvent {
	type: "DEMAND_SIGNAL";
	timestamp?: string;
	retailer_id: string;
	location_id?: string;
	sku: string;
	day: string;
	qty_delta: number;
	net_sold?: number;
	source: "STORE_POS";
	kind?: "sale" | "void";
	supplier_id?: string;
}

/** GET /v1/supplier/analytics/demand/flywheel item (B4.4 supplier feed UI). */
export interface FlywheelDemandItem {
	signal_id: string;
	supplier_id?: string;
	retailer_id: string;
	location_id?: string;
	sku: string;
	day: string;
	qty_delta: number;
	net_sold: number;
	kind: string;
	source: string;
	created_at?: string;
}

export interface FlywheelDemandFeedResponse {
	source: "STORE_POS" | string;
	description?: string;
	items: FlywheelDemandItem[];
	days: number;
	count?: number;
	feed_error?: string;
}

export interface CreateSignalRequest {
	retailerId?: string;
	productId?: string;
	type: string;
	scope: string;
	startDate: string;
	endDate: string;
	multiplier: number;
	description?: string;
}

export interface SupplierMEIONetworkSummary {
  supplier_id: string;
  warehouses_scanned: number;
  skus_analyzed: number;
  insights_generated: number;
  transfer_recommendations: number;
  capital_cap_minor?: number;
  capital_used_minor?: number;
  transfers_skipped_capital?: number;
  warehouse_balances: SupplierMEIOWarehouseNode[];
  generated_at: string;
}

export interface SupplierMEIOWarehouseNode {
  warehouse_id: WarehouseId;
  sku_count: number;
  critical_skus: number;
  warning_skus: number;
  total_stock: number;
  avg_days_cover: number;
}

export interface SupplierReplenishmentPolicy {
  supplier_id: string;
  auto_approve_stable: boolean;
  auto_approve_predictive_push: boolean;
  max_daily_transfer_units: number;
  min_confidence_score: number;
  /** Cycle service level α for safety stock z_α (default 0.98). */
  target_service_level: number;
  /** Policy lead time in days (overridden by observed L when ≥10 samples). */
  lead_time_days: number;
  /** Assumed lead-time σ until observed history exists. */
  lead_time_sigma_days: number;
}

export interface SupplierReplenishmentPolicyPatch {
  auto_approve_stable?: boolean;
  auto_approve_predictive_push?: boolean;
  max_daily_transfer_units?: number;
  min_confidence_score?: number;
  target_service_level?: number;
  lead_time_days?: number;
  lead_time_sigma_days?: number;
}

export interface SupplierReplenishmentTraceRow {
  insight_id: string;
  warehouse_id: string;
  warehouse_name?: string;
  product_id: string;
  product_name?: string;
  status: string;
  reason_code?: string;
  transfer_id?: string;
  transfer_state?: string;
  created_at: string;
  linked_at?: string;
}

export interface SupplierReplenishmentTraceabilityResponse {
  rows: SupplierReplenishmentTraceRow[];
  generated_at: string;
}

export interface ReorderSuggestionRow {
  retailer_id: string;
  retailer_name?: string;
  sku: string;
  sku_name?: string;
  suggested_qty: number;
  adjusted_demand_per_day: number;
  current_stock: number;
  in_flight_qty: number;
  /** §8.2 SS units (legacy demand·0.15 when SAFETY_STOCK_V2_ENABLED off). */
  safety_stock?: number;
  suggested_by_date: string;
  status: string;
  computed_at: string;
  /** L3 sell-through: STORE_POS | WHOLESALE_HISTORY */
  sources?: string[];
  sell_through_velocity?: number;
  base_demand_per_day?: number;
}

export interface ReorderSuggestionsListResponse {
  suggestions: ReorderSuggestionRow[];
}

export interface ReorderSuggestionDismissRequest {
  retailer_id: string;
  sku: string;
}

export interface ReorderSuggestionCreateDraftRequest {
  retailer_id: string;
  sku: string;
}

export interface ReorderSuggestionBulkCreateDraftsRequest {
  items: ReorderSuggestionCreateDraftRequest[];
}

export interface ReorderSuggestionBulkCreateDraftResult {
  retailer_id: string;
  sku: string;
  order_id?: string;
  error?: string;
}

export interface ReorderSuggestionBulkCreateDraftsResponse {
  results: ReorderSuggestionBulkCreateDraftResult[];
}

export interface CashReconciliationRow {
  reconciliation_id: string;
  driver_id: string;
  route_id?: string;
  shift_date: string;
  expected_cash_minor: number;
  declared_cash_minor: number;
  difference_minor: number;
  status: string;
  driver_note?: string;
  finance_note?: string;
  created_at: string;
  resolved_at?: string;
  resolved_by?: string;
}

export interface CashReconciliationsListResponse {
  reconciliations: CashReconciliationRow[];
  supplier_id?: string;
}

export interface CashReconciliationActionRequest {
  note?: string;
}

export interface CreditNoteRow {
  credit_note_id: string;
  order_id: string;
  type: string;
  status: string;
  reason_code: string;
  reason_text?: string;
  total_net_minor: number;
  total_vat_minor: number;
  total_gross_minor: number;
  created_by: string;
  created_at: string;
}

export interface CreditNotesListResponse {
  credit_notes: CreditNoteRow[];
}

export interface CreateCreditNoteRequest {
  order_id: string;
  reason_code: string;
  reason_text?: string;
  lines?: Array<{ order_line_id: string; qty: number }>;
}

export interface TwinStopView {
  RouteID?: string;
  StopID?: string;
  Sequence?: number;
  Status?: string;
  PredictedArrival?: string;
  WindowStart?: string;
  WindowEnd?: string;
  DeliveredGrossMinor?: number;
  RemainingGrossMinor?: number;
}

export interface TwinVehicleInventoryRow {
  RouteID?: string;
  Sku?: string;
  QtyOnVehicle?: number;
}

export interface TemperatureReadingView {
  reading_id: string;
  manifest_id: string;
  sensor_id?: string;
  recorded_at: string;
  temp_c: number;
  lat?: number;
  lng?: number;
  min_c?: number;
  max_c?: number;
  excursion?: boolean;
}

export interface LaborZoneCapacity {
  zoneH3: string;
  date: string;
  totalCapacity: number;
  usedCapacity: number;
  computedAt?: string;
}

export interface LaborDriverScore {
  driverId: string;
  score: number;
  onTimeRate: number;
  completionRate: number;
  damageRate: number;
  shopClosedRate: number;
  feedbackScore: number;
  stopsPerHour: number;
  windowStart?: string;
  windowEnd?: string;
  computedAt?: string;
}

export interface TwinRouteException {
  type: string;
  order_id?: string;
  status?: string;
  detail?: string;
}

export interface TwinOpsRouteView {
  RouteID: string;
  DriverID: string;
  Status: string;
  CurrentLat?: number;
  CurrentLng?: number;
  CurrentH3?: string;
  RemainingStops?: number;
  Stops?: TwinStopView[];
  Inventory?: TwinVehicleInventoryRow[];
  driver_name?: string;
  driver_score?: number;
  lateness: "green" | "amber" | "red";
  has_shop_closed?: boolean;
  exceptions?: TwinRouteException[];
}

export interface PlanningSignalIngestStatus {
  projection_count: number;
  last_ingest_at?: string;
  lag_seconds: number;
  baseline_rows_from_signals: number;
  topic: string;
  healthy: boolean;
}

export interface ControlTowerZoneOverride {
  override_id: string;
  supplier_id: SupplierId;
  warehouse_id?: WarehouseId;
  action: "REROUTE" | "FREEZE_DISPATCH" | "PRIORITY_BOOST" | string;
  polygon_geojson: string | Record<string, unknown>;
  ttl_expires_at: string;
  is_active: boolean;
}

export interface ControlTowerZoneOverridesResponse {
  overrides: ControlTowerZoneOverride[];
}

export interface ControlTowerZoneOverrideRequest {
  warehouse_id?: WarehouseId;
  action: string;
  polygon_geojson: Record<string, unknown>;
  ttl_seconds?: number;
}

export interface PlaybookActionResult {
  index: number;
  type: string;
  status: string;
  message?: string;
}

export interface ControlTowerPlaybookRun {
  run_id: string;
  playbook_id: string;
  exception_id: string;
  supplier_id: string;
  mode: string;
  status: string;
  playbook_name?: string;
  actions_result?: PlaybookActionResult[];
  created_at: string;
  executed_at?: string;
  executed_by?: string;
}

export interface ControlTowerPlaybookRunsResponse {
  runs: ControlTowerPlaybookRun[];
}

export interface ScoredException {
  exception_id: string;
  type: string;
  severity: string;
  order_id?: string;
  retailer_id?: string;
  amount_minor?: number;
  score: number;
  severity_rank: number;
  age_minutes: number;
  retailer_segment?: string;
  recommended_playbook_ids?: string[];
  top_playbook_name?: string;
  created_at: string;
}

export interface ScoredExceptionsResponse {
  exceptions: ScoredException[];
}

export interface ControlTowerPlaybook {
  playbook_id: string;
  supplier_id?: string;
  name: string;
  description?: string;
  is_active: boolean;
  priority: number;
  match_rules: Record<string, unknown>;
  actions: Array<{ type: string; params?: unknown }>;
  auto_execute: boolean;
  created_at: string;
  updated_at: string;
  created_by: string;
}

export interface ControlTowerPlaybooksResponse {
  playbooks: ControlTowerPlaybook[];
}

export interface RetailerSegmentRow {
  retailer_id: string;
  segment: string;
  reason?: string;
  updated_at: string;
}

export interface SkuClassRow {
  sku: string;
  velocity_class: string;
  strategic_flag: boolean;
  updated_at: string;
}

export interface RetailerSegmentsResponse {
  retailers: RetailerSegmentRow[];
}

export interface SkuClassesResponse {
  sku_classes: SkuClassRow[];
}

export interface SegmentationBootstrapResult {
  segments_upserted: number;
  sku_classes_upserted: number;
  policies_seeded: number;
}

export interface SetRetailerSegmentInput {
  segment: string;
  reason?: string;
}

export interface SetSkuClassInput {
  velocity_class: string;
  strategic_flag?: boolean;
}

export interface PlanningScenarioInput {
  factory_downtime_hours?: number;
  demand_delta_pct?: number;
  horizon_days?: number;
  label?: string;
}

export interface PlanningScenarioCloneInput {
  factory_downtime_hours?: number;
  demand_delta_pct?: number;
  horizon_days?: number;
  label?: string;
}

export interface PlanningScenarioResult {
  scenario_id: string;
  supplier_id: SupplierId;
  version?: number;
  status?: "DRAFT" | "PUBLISHED" | "SUPERSEDED" | "REJECTED" | string;
  parent_scenario_id?: string;
  label?: string;
  horizon_days?: number;
  factory_downtime_hours?: number;
  demand_delta_pct?: number;
  sla_risk_pct: number;
  fleet_volume_orders: number;
  stockout_skus: string[];
  capacity_breach: boolean;
  cached_until?: string;
  mode?: string;
  baseline_sla_risk_pct?: number;
  revenue_at_risk_minor?: number;
  unit_value_source?: "products" | "fallback" | "mixed" | string;
  created_by?: string;
  published_by?: string;
  published_at?: string;
  updated_at?: string;
}

export interface PlanningScenarioCompareDeltas {
  sla_risk_pct_delta: number;
  fleet_volume_orders_delta: number;
  revenue_at_risk_minor_delta: number;
  stockout_count_delta: number;
  capacity_breach_changed: boolean;
}

export interface PlanningScenarioCompareResult {
  left: PlanningScenarioResult;
  right: PlanningScenarioResult;
  deltas: PlanningScenarioCompareDeltas;
}

export interface PlanningScenarioListResponse {
  scenarios: PlanningScenarioResult[];
}

export interface PlanningSAndOPSnapshot {
  supplier_id: SupplierId;
  horizon_days: number;
  production_line_count?: number;
  factory_capacity_units: number;
  projected_demand_units?: number;
  warehouse_inbound_cap_units: number;
  warehouse_outbound_cap_units: number;
  utilization_pct: number;
  capacity_alert: boolean;
  capacity_model?: string;
  capacity_source?: "factories_column" | "env_default" | string;
}

export interface KnowledgeGraphNode {
  id: string;
  type: string;
  name?: string;
}

export interface KnowledgeGraphEdge {
  from: string;
  to: string;
  relation: string;
}

export interface SupplierKnowledgeGraph {
  supplier_id: SupplierId;
  nodes: KnowledgeGraphNode[];
  edges: KnowledgeGraphEdge[];
}

export interface GovernedAgentInvocation {
  action: "approve_insight" | "open_supply_request" | "broadcast_template" | string;
  idempotency_key: string;
  supplier_id?: SupplierId;
  target_id: string;
  note?: string;
}

export interface GovernedAgentInvocationResponse {
  status: string;
  action: string;
  idempotency_key: string;
  result_id?: string;
}

export interface ForecastConfidence {
  low_units?: number;
  high_units?: number;
  confidence_pct?: number;
  baseline_source?: "moving_average" | "seasonal_template" | "mixed" | "inventory_hint" | string;
  blocked_reason?: string;
  label?: "insufficient_history" | "early_signal" | "standard" | string;
}

export interface SparsityGateResult {
  allowed: boolean;
  completed_orders: number;
  confidence_cap_pct?: number;
  blocked_reason?: string;
  label: string;
}

export interface SeasonalOverrideInput {
  template_id?: string;
  start_date: string;
  end_date: string;
  name?: string;
  /** Optional; clamped server-side to [0.5, 2.5]. Inherits builtin when template_id matches. */
  multiplier?: number;
}

export interface SeasonalOverrideRow {
  override_id: string;
  supplier_id: SupplierId;
  template_id: string;
  name?: string;
  start_date: string;
  end_date: string;
  is_active: boolean;
  multiplier: number;
}

export interface SeasonalBuiltinTemplate {
  id: string;
  name: string;
  multiplier?: number;
  start_month?: number;
  start_day?: number;
  end_month?: number;
  end_day?: number;
  confidence_floor?: number;
}

export interface SeasonalTemplatesResponse {
  builtin_templates: SeasonalBuiltinTemplate[];
  overrides: SeasonalOverrideRow[];
}

/** One persisted ForecastAccuracyDaily row (admin accuracy surface). */
export interface ForecastAccuracyDailyRow {
  SupplierId: string;
  ForecastDate: string;
  WarehouseId: string;
  ProductId: string;
  ForecastQty: number;
  ActualQty: number;
  AbsError: number;
  SignedError: number;
  Wape7: number;
  Wape28: number;
  Bias7: number;
  Bias28: number;
  TrackingSignal: number;
  SampleDays7: number;
  SampleDays28: number;
  AlertTs: boolean;
  ComputedAt: string;
}

export interface ForecastAccuracyResponse {
  items: ForecastAccuracyDailyRow[];
  days: number;
}

export interface SeasonalEstimateSuggestion {
  template_id: string;
  name: string;
  start_date: string;
  end_date: string;
  multiplier: number;
  basis: string;
  sample_days?: number;
  draft_override_id?: string;
}

export interface SeasonalEstimateResult {
  suggestions: SeasonalEstimateSuggestion[];
  persisted_drafts: number;
}

export interface SupplierCRMRetailer {
  retailer_id: string;
  retailer_name: string;
  phone?: string;
  lifetime: number;
  order_count: number;
  last_order_date?: string;
  status: "ACTIVE" | "INACTIVE" | string;
}

export interface SupplierCRMOrderLine {
  sku?: string;
  product_name?: string;
  qty: number;
  amount_minor?: number;
}

export interface SupplierCRMOrder {
  order_id: string;
  state: string;
  amount: number;
  item_count: number;
  created_at: string;
  lines?: SupplierCRMOrderLine[];
}

export interface SupplierCRMRetailerDetail extends SupplierCRMRetailer {
  orders: SupplierCRMOrder[];
}

export interface SupplierCRMListResponse {
  retailers: SupplierCRMRetailer[];
}

export interface SupplyRequestQCResponse {
  request_id: string;
  result: "" | "PASS" | "FAIL" | string;
  notes?: string;
  inspected_by?: string;
  inspected_at?: string;
}

export interface SupplyRequestQCRequest {
  result: "PASS" | "FAIL";
  notes?: string;
}

export interface NetworkModeResponse {
  mode: string;
  supplier_id: SupplierId;
  planning_enabled?: boolean;
}

export interface NetworkModeUpdateRequest {
  mode: string;
  reason?: string;
}

export interface NetworkModeUpdateResponse {
  old_mode: string;
  new_mode: string;
  status: string;
}

export interface PullMatrixResponse {
  status: string;
  transfers: number;
  skus: number;
  source: string;
}

export interface KillSwitchRequest {
  reason: string;
}

export interface KillSwitchResponse {
  status: string;
  cancelled_transfers: number;
  mode: string;
}

export interface LoyaltyTier {
  name: string;
  min_points: number;
}

export interface LoyaltyProgram {
  supplier_id: string;
  earn_bps: number;
  tiers: LoyaltyTier[];
  reason?: string;
  source?: string;
}

export interface LoyaltyTierView {
  enrolled: boolean;
  tier?: string;
  lifetime_points: number;
  available_points: number;
  next_tier?: string;
  points_to_next?: number;
  earn_bps?: number;
  supplier_id?: string;
}

export interface LoyaltyLedgerEntry {
  ledger_id: string;
  order_id: string;
  points: number;
  earn_bps: number;
  amount_minor: number;
  created_at: string;
}

export interface LoyaltyLedgerResponse {
  entries: LoyaltyLedgerEntry[];
}

export interface EntityResolutionCandidate {
  node_id: string;
  entity_type: string;
  entity_id: string;
  label: string;
  score: number;
  confidence_class: string;
  deterministic: boolean;
  reasons?: string[];
  metadata?: Record<string, string>;
}

export interface EntityResolutionResolveRequest {
  entity_type?: string;
  entity_id?: string;
  query?: string;
  max_candidates?: number;
}

export interface EntityResolutionResolveResponse {
  scope_supplier_id: string;
  requested_type: string;
  query?: string;
  resolved?: EntityResolutionCandidate;
  candidates: EntityResolutionCandidate[];
}

export interface EntityResolutionExplainRequest {
  entity_type: string;
  entity_id: string;
}

export interface EntityResolutionExplainResponse {
  scope_supplier_id: string;
  source: EntityResolutionCandidate;
  projection: {
    nodes: Array<{ node_id: string; entity_type: string; entity_id: string; label: string }>;
    edges: Array<{ from: string; to: string; relation: string; confidence: number }>;
  };
}

export interface PredictivePushResponse {
  transfers: number;
  skus: number;
  source: string;
  grain?: string;
  not_from?: string;
  error?: string;
}

export interface PayoutRailInfo {
  name: string;
  is_live: boolean;
  workflow: string;
  steps?: string[];
  message?: string;
}

export interface PayoutBatch {
  batch_id: string;
  supplier_id: SupplierId;
  period_start: string;
  period_end: string;
  gross_captured_minor: number;
  refunded_minor: number;
  commission_minor: number;
  net_payout_minor: number;
  currency: Iso4217;
  status: string;
  export_file_uri?: string;
  rail_reference?: string;
  idempotency_key?: string;
  created_by?: string;
  created_at?: string;
  updated_at?: string;
}

export interface PayoutBatchListResponse {
  batches: PayoutBatch[];
}

export interface PayoutBatchGenerateRequest {
  period_start: string;
  period_end: string;
  supplier_id?: string;
  idempotency_key?: string;
}

export interface PayoutBatchGenerateResponse {
  batch: PayoutBatch;
  rail: PayoutRailInfo;
}

export interface PayoutMarkPaidResponse {
  status: string;
  batch_id: string;
  rail: PayoutRailInfo;
  message?: string;
}

export interface PayoutDispatchResponse {
  batch?: PayoutBatch;
  rail: PayoutRailInfo;
  error?: string;
  code?: string;
  message?: string;
}

export interface SupplierPayoutPolicy {
  supplier_id: string;
  payout_mode: "HQ_SUPPLIER" | "WAREHOUSE_LOCAL" | string;
  fee_policy_version: string;
  effective_at?: string;
  updated_by?: string;
  updated_by_type?: string;
  reason?: string;
  is_active: boolean;
  source: string;
}

export interface SupplierPayoutPolicyPatch {
  payout_mode: string;
  fee_policy_version?: string;
  reason: string;
}

export interface PlanningSignalIngestInput {
  signal_id?: string;
  source: string;
  warehouse_id?: string;
  retailer_id?: string;
  payload?: Record<string, unknown>;
}

export interface PromoSimulateInput {
  promotion_id?: string;
  discount_pct?: number;
  expected_units?: number;
  avg_unit_margin_minor?: number;
}

export interface PromoSimulateResult {
  simulation_id: string;
  promotion_id?: string;
  projected_volume: number;
  projected_revenue_minor: number;
  projected_margin_minor: number;
  margin_delta_pct: number;
  sandbox_only: boolean;
}

export interface PromoPerformanceResult {
  promotion_id: string;
  predicted_volume: number;
  actual_volume: number;
  volume_accuracy_pct: number;
  predicted_margin_minor: number;
  actual_margin_minor: number;
  closed_loop_score: number;
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

/** §8.4 server-persisted forecast accuracy (WAPE / bias / tracking signal). */
export interface SupplierDemandAccuracySeriesRow {
  date: string;
  warehouse_id: string;
  product_id: string;
  forecast_qty: number;
  actual_qty: number;
  wape_7: number;
  wape_28: number;
  bias_7: number;
  bias_28: number;
  tracking_signal: number;
  alert_ts: boolean;
}

export interface SupplierDemandAccuracyResponse {
  enabled: boolean;
  period_days: number;
  as_of?: string;
  wape_7: number;
  wape_28: number;
  bias_7: number;
  bias_28: number;
  tracking_signal: number;
  sample_days_7: number;
  sample_days_28: number;
  alert_count: number;
  forecast_units: number;
  actual_units: number;
  series: SupplierDemandAccuracySeriesRow[];
  generated_at: string;
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
  split_group_id?: string;
  sibling_manifest_ids?: string[];
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
  score?: number;
  top_playbook_name?: string;
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
  payment_gateway?: string;
  state: string;
  status: string;
  split_group_id?: string;
  route_id?: string;
  warehouse_id?: string;
  delivery_expectation?: DeliveryExpectation;
}

export interface SupplierManifestDetail extends SupplierManifestRow {
  orders?: SupplierManifestOrderWire[];
  overflow_count?: number;
  sealed_at?: string;
  dispatched_at?: string;
  created_at?: string;
  region_code?: string;
  /** Thin inbound map — last known driver GPS for payload seal/handoff. */
  driver_lat?: number;
  driver_lng?: number;
  live_location_available?: boolean;
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
  shop_closed_reason?: string;
  shop_closed_resolution?: string;
  grace_ends_at?: string;
  shop_closed_at?: string;
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

/** Driver: mark shop closed (POST /v1/delivery/shop-closed). */
export type ShopClosedReason = "NO_ANSWER" | "CLOSED" | "REFUSED" | "OTHER";

export interface ShopClosedReportRequest {
  order_id: string;
  latitude: number;
  longitude: number;
  reason?: ShopClosedReason;
  photo_url?: string;
  /** Offline capture time (RFC3339). */
  client_timestamp?: string;
}

export interface ShopClosedReportResponse {
  status: "ARRIVED_SHOP_CLOSED";
  attempt_id: string;
  reason?: string;
  grace_ends_at?: string;
}

/** Retailer response codes for enhanced shop-closed protocol. */
export type ShopClosedRetailerResponse =
  | "OPEN_NOW"
  | "5_MIN"
  | "CALL_ME"
  | "CLOSED_TODAY"
  | "RESCHEDULE"
  | "CREDIT_LEAVE"
  | "CANCEL"
  | "AUTHORIZE_BYPASS";

export interface ShopClosedRetailerResponseRequest {
  order_id: string;
  response: ShopClosedRetailerResponse;
  new_slot?: string;
  photo_url?: string;
}

/** GET/PUT /v1/supplier/return-policy */
export interface SupplierReturnPolicy {
  default_window_hours: number;
  concealed_damage_window_hours?: number | null;
  require_photo?: boolean;
  allow_expired_claims?: boolean;
  policy_source_hint?: string;
}

/** GET/PUT /v1/supplier/service-policy */
export interface SupplierServicePolicy {
  supplier_id: string;
  lead_time_days: number;
  same_day_cutoff_time?: string;
  next_day_cutoff_time?: string;
  min_order_minor: number;
  currency: string;
  fill_rate_guarantee_bps: number;
  allow_scheduled_delivery: boolean;
  max_schedule_advance_days: number;
  assigned_manager_name?: string;
  assigned_manager_phone?: string;
  updated_at?: string;
  updated_by_user_id?: string;
}

/** GET/POST /v1/retailer/service-promise */
export interface PromiseEvaluationRequest {
  supplier_id: string;
  retailer_id?: string;
  warehouse_id?: string;
  total_minor?: number;
  currency?: string;
  requested_delivery_date?: string;
}

export interface PromiseEvaluationResult {
  eligible: boolean;
  promise_type: "SAME_DAY" | "NEXT_DAY" | "SCHEDULED" | string;
  guaranteed_delivery_date: string;
  sla_hours: number;
  fill_rate_target_bps: number;
  min_order_minor: number;
  currency: string;
  cutoff_time?: string;
  reason?: string;
}

export interface OrderServicePromiseSnapshot {
  order_id: string;
  supplier_id: string;
  retailer_id: string;
  warehouse_id: string;
  promise_type: string;
  guaranteed_delivery_date: string;
  cutoff_applied_at?: string;
  fill_rate_target_bps: number;
  min_order_minor: number;
  currency: string;
  sla_hours: number;
  status: "PENDING" | "MET" | "BREACHED" | string;
  breached_at?: string;
  breach_reason?: string;
  created_at?: string;
  updated_at?: string;
}

/** GET/PUT /v1/warehouse/return-policy */
export interface WarehouseReturnPolicy {
  supplier_id: string;
  reverse_dock_sla_hours?: number | null;
  retailer_file_window_hours?: number | null;
  can_override_retailer_window: boolean;
}

/** Proximity unlock before cash/credit/split (POST /v1/delivery/proximity-unlock). */
export type ProximityMethod = "H3" | "GEOFENCE_100M" | "MANUAL" | "FORCE_BYPASS";

export interface ProximityUnlockRequest {
  order_id: string;
  latitude: number;
  longitude: number;
  client_timestamp?: string;
  force_bypass_token?: string;
}

export interface ProximityUnlockResponse {
  order_id: string;
  proximity_unlocked: boolean;
  proximity_method?: ProximityMethod | string;
  distance_m?: number;
  unlocked_at?: string;
  payment_modes_enabled: boolean;
  message?: string;
}

/** Line-level partial offload (POST /v1/delivery/partial-offload). */
export type OffloadStatus = "FULL" | "PARTIAL" | "NONE" | "RETURNED";
export type OffloadReason =
  | "DAMAGED"
  | "MISSING"
  | "SHOP_REFUSED"
  | "CAPACITY"
  | "OTHER";

export interface PartialOffloadLine {
  sku: string;
  delivered_qty: number;
  remaining_qty: number;
  reason?: OffloadReason;
}

export interface PartialOffloadRequest {
  order_id: string;
  lines: PartialOffloadLine[];
  client_timestamp?: string;
  signed_nonce?: string;
  note?: string;
}

export interface PartialOffloadResponse {
  order_id: string;
  partial_delivery: boolean;
  delivered_minor: number;
  remaining_minor: number;
  currency: string;
  status: string;
  message: string;
}

export interface CreditDeliveryRequest {
  order_id: string;
  photo_proof_url?: string;
  force_bypass_token?: string;
}

/** Timeout auto-decision (server-side matrix). */
export type ShopClosedTimeoutResolution =
  | "CREDIT_LEAVE"
  | "RETURN_TO_WAREHOUSE"
  | "FORCE_BYPASS"
  | "RESCHEDULE";


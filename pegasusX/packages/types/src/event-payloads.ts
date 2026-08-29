import { CommandId, DeliveryExpectation, DriverId, FactoryId, H3Cell, HomeNode, Iso4217, ManifestId, Money, OrderConfirmationStatus, OrderId, OrderSource, OrderStatus, PaymentGateway, RetailerId, Role, RouteId, SessionId, SupplierId, VehicleId, WarehouseId } from "./primitives";
import { ForecastConfidence } from "./supplier";
import { PaymentLedgerEntry, RouteGeometryWire } from "./compliance";
import { WsEventEnvelope } from "./envelope";
import { WarehouseDispatchPreview, WarehouseSupplyRequestItem } from "./warehouse";

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

export interface FactoryStaffCreated {
  factory_id: FactoryId;
  supplier_id: SupplierId;
  user_id: string;
  supplier_role?: string;
}

export interface FactoryTransferCreated {
  transfer_id: string;
  factory_id?: FactoryId;
  supplier_id: SupplierId;
  state?: string;
}

export interface ManifestExceptionResolved {
  manifest_id: string;
  exception_id?: string;
  factory_id?: FactoryId;
  supplier_id?: SupplierId;
  reason?: string;
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
  handling_class?: "GENERAL" | "COLD_CHAIN" | "HAZARDOUS" | "PERISHABLE";
  requires_cold_chain?: boolean;
  is_hazardous?: boolean;
  is_perishable?: boolean;
  storage_temp_min_c?: number | null;
  storage_temp_max_c?: number | null;
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
  /** Planned sealed-route overlay from SupplierTruckManifests (Google Routes / OSRM). */
  route_geometry?: RouteGeometryWire;
  payment_evidence?: RetailerTrackingPaymentEvidence;
  receipt_dossier?: RetailerTrackingReceiptDossier;
  created_at: string;
  updated_at: string;
  items: RetailerTrackingLineItem[];
  delivery_expectation?: DeliveryExpectation;
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
  deliver_before?: string;
  preorder_badge?: string;
  proposed_delivery_date?: string;
  delivery_proposal_reason?: string;
  auto_confirm_at?: string;
  decision_at?: string;
  decision_by?: string;
  total_minor: number;
  currency: Iso4217;
  version: number;
  updated_at: string;
  created?: boolean;
}

export interface ProposeDeliveryDateRequest {
  proposed_delivery_date: string;
  reason: string;
}

export interface AcceptDeliveryProposalRequest {
  order_id: OrderId;
}

export interface RejectDeliveryProposalRequest {
  order_id: OrderId;
  reason?: string;
}

export interface RejectPreorderRequest {
  order_id: OrderId;
  reason?: string;
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
  confidence?: ForecastConfidence;
  demand_breakdown?: Record<string, unknown> | null;
}

/** Legacy SKU-forecast card shape. Not returned by GET /v1/retailer/ai/predictions. */
export interface RetailerDemandPrediction {
  id: string;
  product_id?: string;
  product_name?: string;
  predicted_quantity?: number;
  predicted_amount?: number;
  confidence?: number;
  reasoning?: string;
  suggested_order_date?: string;
  status?: string;
  blocked?: boolean;
  blocked_reason?: string;
  label?: "insufficient_history" | "early_signal" | "standard" | string;
}

export interface WarehouseDemandForecastResponse {
  warehouse_id: WarehouseId;
  forecast_days?: number;
  generated_at?: string;
  series: WarehouseDemandForecastDay[];
  /** Portal parity: product-level 4-source breakdown (pegasus warehouse-portal). */
  products?: WarehouseDemandForecastProduct[];
  /** empty | spanner | scaffold — never invent a series when empty. */
  source?: string;
}

export interface WarehouseHoldReason {
  code: string;
  count: number;
}

export interface WarehouseOpsDashboardResponse {
  warehouse_id?: WarehouseId;
  active_orders?: number;
  pending_dispatch?: number;
  drivers_on_route?: number;
  drivers_idle?: number;
  total_drivers?: number;
  total_vehicles?: number;
  low_stock_count?: number;
  total_staff?: number;
  completed_today_available?: boolean;
  today_revenue_available?: boolean;
  history_available?: boolean;
  currency?: string;
  orders_by_status?: Record<string, number>;
  truck_duty?: Record<string, number>;
  hold_reasons?: WarehouseHoldReason[];
  demand_source?: string;
  fleet_status?: Array<{ status: string; count: number }>;
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

/** Portal list row (compact ops shape from GET /v1/warehouse/ops/orders). */
export interface WarehouseOrderListItem {
  order_id: OrderId;
  retailer_name?: string;
  retailer_id?: RetailerId;
  state?: string;
  status?: string;
  total_uzs?: number;
  total_minor?: number;
  created_at?: string;
  updated_at?: string;
}

export interface WarehouseOrderLineItem {
  product_id?: string;
  product_name?: string;
  quantity?: number;
  unit_price?: number;
  handling_class?: "GENERAL" | "COLD_CHAIN" | "HAZARDOUS" | "PERISHABLE";
  requires_cold_chain?: boolean;
  is_hazardous?: boolean;
  is_perishable?: boolean;
  storage_temp_min_c?: number | null;
  storage_temp_max_c?: number | null;
}

export interface WarehouseOrderDetail {
  order_id: OrderId;
  retailer_name?: string;
  state?: string;
  status?: string;
  total_uzs?: number;
  total_minor?: number;
  line_items?: WarehouseOrderLineItem[];
}

export interface OrderConditionReport {
  report_id: string;
  order_id: OrderId;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  line_item_index?: number | null;
  sku?: string;
  condition_type: "DAMAGED" | "EXPIRED" | "TEMPERATURE_BREACH" | "MISSING" | "QUALITY_REJECT" | "OTHER";
  severity: "LOW" | "MEDIUM" | "HIGH";
  description?: string;
  photo_urls?: string[];
  proof_ids?: string[];
  reported_by: string;
  reported_by_role: string;
  resolution_status: "OPEN" | "RESOLVED" | "DISPUTED" | "ESCALATED";
  resolved_by?: string;
  resolved_at?: string;
  resolution_notes?: string;
  created_at: string;
}

export interface RetailerCreditProfile {
  retailer_id: RetailerId;
  supplier_id: SupplierId;
  credit_limit_minor: number;
  current_balance_minor: number;
  reserved_minor?: number;
  available_credit_minor: number;
  risk_score: number;
  risk_tier: "LOW" | "MEDIUM" | "HIGH" | "BLOCK";
  delinquency_count: number;
  status: "INACTIVE" | "ACTIVE" | "FROZEN" | "CLOSED" | "BLACKLISTED";
  last_evaluated_at?: string;
  version: number;
  created_at: string;
  updated_at: string;
}

/** Supplier org-level irreversible credit program (CREDIT_POLICY_V2). */
export interface SupplierCreditProgram {
  supplier_id: SupplierId;
  program_enabled: boolean;
  enabled_at?: string;
  enabled_by_user_id?: string;
  disabled_at?: string;
  disabled_by_actor?: string;
  disable_reason?: string;
  global_terms_days: number;
  global_grace_days: number;
  global_default_limit_minor: number;
  timezone?: string;
  version: number;
  updated_at?: string;
}

/** Per-retailer Net terms / relationship (CREDIT_POLICY_V2). */
export interface RetailerPaymentTerms {
  retailer_id: RetailerId;
  supplier_id: SupplierId;
  credit_enabled: boolean;
  enabled_at?: string;
  terms_days: number;
  grace_period_days: number;
  credit_limit_minor: number;
  use_global_defaults: boolean;
  version: number;
  updated_at?: string;
  profile_status?: string;
  available_credit_minor?: number;
  current_balance_minor?: number;
  on_hold?: boolean;
}

/** AR open item created at credit leave. */
export interface ArInvoice {
  invoice_id: string;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  order_id: OrderId;
  status: "OPEN" | "PARTIAL" | "PAID" | "VOID" | string;
  principal_minor: number;
  balance_minor: number;
  currency: string;
  credit_leave_at: string;
  due_at: string;
  terms_days: number;
  grace_period_days: number;
  aging_bucket?: string;
  dunning_step?: number;
  version: number;
  created_at?: string;
  updated_at?: string;
}

export interface AgingSummaryResponse {
  supplier_id: string;
  currency: string;
  total_open_minor: number;
  total_overdue_minor: number;
  bucket_current_minor: number;
  bucket_1_30_minor: number;
  bucket_31_60_minor: number;
  bucket_61_90_minor: number;
  bucket_90_plus_minor: number;
  total_invoices_count: number;
  delinquent_retailers_count: number;
  high_risk_invoice_count: number;
  computed_at: string;
}

export interface DelinquencyLockStatus {
  retailer_id: string;
  supplier_id?: string;
  is_locked: boolean;
  reason?: string;
  overdue_amount_minor: number;
  overdue_count: number;
  checked_at: string;
}

export interface RetailerPayInvoiceRequest {
  amount_minor: number;
  payment_method: "WALLET" | "CARD" | "BANK_TRANSFER" | string;
  payment_reference?: string;
}

export interface WriteOffRequest {
  reason: string;
}

export interface CashBagSummaryResponse {
  driver_id: string;
  shift_date: string;
  expected_cash_minor: number;
  collected_cash_minor: number;
  declared_cash_minor: number;
  difference_minor: number;
  reconciliation_id?: string;
  reconciliation_status: "PENDING_TURN_IN" | "SUBMITTED" | "BALANCED" | "ACCEPTED" | "DISPUTED" | string;
  driver_note?: string;
  finance_note?: string;
  pending_orders?: { order_id: string; retailer_id?: string; amount: number; state: string }[];
}

export interface CashBagTurnInRequest {
  declared_cash_minor: number;
  driver_note?: string;
  route_id?: string;
}

export interface CashReconciliation {
  reconciliation_id: string;
  driver_id: string;
  route_id?: string;
  shift_date: string;
  expected_cash_minor: number;
  declared_cash_minor: number;
  difference_minor: number;
  status: "PENDING" | "SUBMITTED" | "BALANCED" | "ACCEPTED" | "DISPUTED" | string;
  driver_note?: string;
  finance_note?: string;
  created_at: string;
  resolved_at?: string;
  resolved_by?: string;
}

export interface OrderTimelineEntry {
  transition_id: string;
  order_id: OrderId;
  previous_status?: string;
  new_status: string;
  reason?: string;
  actor_role?: string;
  actor_id?: string;
  event_kind?: string;
  metadata?: Record<string, unknown>;
  created_at: string;
}

export interface OrderTimelineResponse {
  order_id: OrderId;
  items: OrderTimelineEntry[];
}

export interface WarehousePreordersResponse {
  preorders: RetailerOrderLifecycleResponse[];
  items?: RetailerOrderLifecycleResponse[];
}

export interface WarehousePreorderEditRequest {
  line_items?: RetailerOrderLineItem[];
  requested_delivery_date: string;
  reason: string;
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
  reason_code?: string;
  demand_breakdown?: Record<string, unknown> | null;
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

export interface SplitPaymentCreated {
  order_id: OrderId;
  driver_id?: DriverId;
  supplier_id?: SupplierId;
  retailer_id?: RetailerId;
  cash_minor?: number;
  card_minor?: number;
  currency?: string;
  transaction_id?: string;
  source?: string;
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

export interface ARInvoiceEventPayload {
  invoice_id: string;
  supplier_id: SupplierId;
  retailer_id: RetailerId;
  order_id?: OrderId;
  principal_minor?: number;
  amount_minor?: number;
  balance_minor?: number;
  status?: string;
  dunning_step?: number;
  aging_bucket?: string;
  due_at?: string;
  last_dunned_at?: string;
  timestamp: string;
}

export interface PayoutBatchEventPayload {
  batch_id: string;
  supplier_id: SupplierId;
  status: string;
  net_payout_minor: number;
  currency: Iso4217;
  rail_reference: string;
  timestamp: string;
}

export interface BuyerAcceptanceEventPayload {
  order_id: OrderId;
  supplier_id: SupplierId;
  retailer_id?: RetailerId;
  ehf_id?: string;
  status: "PENDING" | "ACCEPTED" | "REJECTED" | "EXPIRED";
  deadline?: string;
  timestamp: string;
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


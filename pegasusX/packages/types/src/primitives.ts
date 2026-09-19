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
export type DeliveryExpectationKind =
  | "STANDARD"
  | "SCHEDULED_PREORDER"
  | "EXPRESS"
  | "PROPOSAL_PENDING";

export type DeliveryExpectationUrgency =
  | "on_track"
  | "due_soon"
  | "overdue"
  | "scheduled_far";

export interface DeliveryExpectation {
  kind: DeliveryExpectationKind;
  target_date?: string;
  target_label: string;
  mode_label?: string;
  receiving_window_open?: string;
  receiving_window_close?: string;
  delayed: boolean;
  delay_reason?: string;
  urgency: DeliveryExpectationUrgency;
  badge_label?: string;
}

export interface StatusExplain {
  code: string;
  title: string;
  summary: string;
  next_steps?: string[];
  deep_link?: string;
  recoverable: boolean;
}

export interface HandoffCardMetadata {
  kind: string;
  title: string;
  subtitle?: string;
  primary_cta?: string;
  primary_link?: string;
  entity_type?: string;
  entity_id?: string;
  fields?: Record<string, string>;
}

export interface PulseEvent {
  id: string;
  kind: string;
  title: string;
  description?: string;
  occurred_at: string;
  deep_link?: string;
  order_id?: string;
  manifest_id?: string;
}

export interface PulseResponse {
  events: PulseEvent[];
  fetched_at: string;
  unread_count?: number;
}

export interface ExceptionMapCell {
  h3_cell: H3Cell;
  lat: number;
  lng: number;
  severity: "low" | "medium" | "high";
  counts: Record<string, number>;
  sample_order_ids?: string[];
  deep_link: string;
}

export interface ExceptionMapResponse {
  cells: ExceptionMapCell[];
  window_hours: number;
}

export interface RetailerOverridePreview {
  retailers_on_sku_count: number;
  active_override_count: number;
  catalog_list_price: number;
  margin_delta_per_unit: number;
  margin_estimate_label: string;
  affected_retailer_ids?: string[];
  read_only?: boolean;
}

export interface SupplyFulfillOptions {
  transfer_mode: "TRUCK" | "INTERNAL";
  warehouse_id: string;
  warehouse_name: string;
  co_located: boolean;
  outcome_internal: string;
  outcome_truck: string;
  linked_driver_eta?: string;
}

export type BroadcastTemplateScope = "supplier" | "warehouse";

export interface BroadcastTemplate {
  id: string;
  category: string;
  title: string;
  body: string;
  default_role: "ALL" | "DRIVER" | "RETAILER" | "PAYLOAD" | "WAREHOUSE" | "FACTORY";
  scope: BroadcastTemplateScope;
  source?: "builtin" | "custom";
  warehouse_id?: string;
  placeholder_keys?: string[];
}

export interface BroadcastTemplatesResponse {
  templates: BroadcastTemplate[];
}

export const SUPPLIER_BROADCAST_TEMPLATES: BroadcastTemplate[] = [
  {
    id: "storm_delay",
    category: "operations",
    scope: "supplier",
    source: "builtin",
    title: "Delivery delay notice",
    body: "Due to weather conditions, deliveries may be delayed on {date}. We will update routes as conditions improve.",
    default_role: "RETAILER",
    placeholder_keys: ["date"],
  },
  {
    id: "holiday_hours",
    category: "operations",
    scope: "supplier",
    source: "builtin",
    title: "Holiday receiving hours",
    body: "Our network will operate on reduced hours on {date}. Please confirm your receiving window in the app.",
    default_role: "RETAILER",
    placeholder_keys: ["date"],
  },
  {
    id: "fee_notice",
    category: "finance",
    scope: "supplier",
    source: "builtin",
    title: "Service fee update",
    body: "A service fee adjustment takes effect on {date}. Review your latest invoices for details.",
    default_role: "RETAILER",
    placeholder_keys: ["date"],
  },
  {
    id: "yard_hold",
    category: "operations",
    scope: "supplier",
    source: "builtin",
    title: "Yard congestion advisory",
    body: "Loading bay congestion reported. Drivers: expect queue delays at warehouse check-in.",
    default_role: "DRIVER",
  },
];

export const WAREHOUSE_BROADCAST_TEMPLATES: BroadcastTemplate[] = [
  {
    id: "wh_yard_hold",
    category: "operations",
    scope: "warehouse",
    source: "builtin",
    title: "Yard congestion advisory",
    body: "Loading bay congestion at this depot. Drivers: expect queue delays at check-in.",
    default_role: "DRIVER",
  },
  {
    id: "wh_gate_delay",
    category: "operations",
    scope: "warehouse",
    source: "builtin",
    title: "Gate delay notice",
    body: "Inbound gate processing is slower than usual. Drivers: allow extra time at arrival.",
    default_role: "DRIVER",
  },
  {
    id: "wh_receiving_hours",
    category: "operations",
    scope: "warehouse",
    source: "builtin",
    title: "Receiving hours update",
    body: "This depot will operate on reduced receiving hours on {date}. Confirm your delivery window.",
    default_role: "RETAILER",
    placeholder_keys: ["date"],
  },
  {
    id: "wh_check_in_slow",
    category: "operations",
    scope: "warehouse",
    source: "builtin",
    title: "Slow check-in advisory",
    body: "Check-in is taking longer than usual at this warehouse. Reason: {reason}",
    default_role: "DRIVER",
    placeholder_keys: ["reason"],
  },
];

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
  | "FISCALIZING"
  | "FISCAL_FAILED"
  | "COMPLETED"
  | "CANCELLED"
  | "RECONCILIATION_REQUIRED"
  | "DELAYED"
  | "BACKORDERED"
  // Operational / legacy client aliases still seen on WS payloads and fleet guards.
  | "DISPATCHED"
  | "ARRIVING"
  | "EN_ROUTE";

/** ADR-009 fiscal attempt / order rollup status. */
export type FiscalStatus =
  | "NONE"
  | "PENDING"
  | "SUCCESS"
  | "FAILED"
  | "FORCE_SKIPPED";

/** Canonical order columns for command boards. Aliases map into these. */
export const ORDER_STATUS_FUNNEL = [
  "PENDING",
  "SCHEDULED",
  "AUTO_ACCEPTED",
  "BACKORDERED",
  "LOADED",
  "IN_TRANSIT",
  "DELAYED",
  "ARRIVED",
  "ARRIVED_SHOP_CLOSED",
  "AWAITING_PAYMENT",
  "PENDING_CASH_COLLECTION",
  "DELIVERED_ON_CREDIT",
  "FISCALIZING",
  "FISCAL_FAILED",
  "RECONCILIATION_REQUIRED",
  "COMPLETED",
  "CANCELLED",
] as const satisfies readonly OrderStatus[];

export type OrderStatusFunnel = (typeof ORDER_STATUS_FUNNEL)[number];

export const ORDER_STATUS_ALIASES: Record<string, OrderStatusFunnel> = {
  DISPATCHED: "LOADED",
  EN_ROUTE: "IN_TRANSIT",
  ARRIVING: "ARRIVED",
  SHOP_CLOSED_PENDING: "ARRIVED_SHOP_CLOSED",
};

export function canonicalizeOrderStatus(status: string): string {
  const key = String(status || "").trim().toUpperCase();
  return ORDER_STATUS_ALIASES[key] ?? key;
}

export function emptyOrderStatusCounts(): Record<OrderStatusFunnel, number> {
  return Object.fromEntries(ORDER_STATUS_FUNNEL.map((key) => [key, 0])) as Record<OrderStatusFunnel, number>;
}

/** Optimistic chip increment. Aliases fold into the funnel; unknown keys are ignored. */
export function incrementOrderStatusCount(
  counts: Partial<Record<string, number>> | null | undefined,
  status: string,
): Record<OrderStatusFunnel, number> {
  const next = { ...emptyOrderStatusCounts() };
  for (const [key, value] of Object.entries(counts ?? {})) {
    const normalized = canonicalizeOrderStatus(key);
    if (normalized in next && Number.isFinite(value)) {
      next[normalized as OrderStatusFunnel] = Number(value);
    }
  }
  const key = canonicalizeOrderStatus(status);
  if (key in next) {
    next[key as OrderStatusFunnel] += 1;
  }
  return next;
}

export type ManifestState = "DRAFT" | "LOADING" | "SEALED" | "DISPATCHED" | "COMPLETED" | "CANCELLED";

export const MANIFEST_STATES = [
  "DRAFT",
  "LOADING",
  "SEALED",
  "DISPATCHED",
  "COMPLETED",
  "CANCELLED",
] as const satisfies readonly ManifestState[];

export function emptyManifestStateCounts(): Record<ManifestState, number> {
  return Object.fromEntries(MANIFEST_STATES.map((key) => [key, 0])) as Record<ManifestState, number>;
}

/** Live factory-transfer dictionary. Last-mile retailer IN_TRANSIT is not a factory truck. */
export const FACTORY_TRANSFER_STATES = [
  "CREATED",
  "APPROVED",
  "PENDING",
  "ASSIGNED",
  "LOADING",
  "DISPATCHED",
  "IN_TRANSIT",
  "ARRIVED",
  "RECEIVED",
  "CANCELLED",
  "REASSIGNED",
] as const;

export type FactoryTransferState = (typeof FACTORY_TRANSFER_STATES)[number];

export function emptyFactoryTransferCounts(): Record<FactoryTransferState, number> {
  return Object.fromEntries(FACTORY_TRANSFER_STATES.map((key) => [key, 0])) as Record<FactoryTransferState, number>;
}

export function canonicalizeFactoryTransfer(status: string): string {
  return String(status || "").trim().toUpperCase();
}

export const FACTORY_VEHICLE_STATES = ["READY", "AVAILABLE", "UNAVAILABLE"] as const;
export type FactoryVehicleState = (typeof FACTORY_VEHICLE_STATES)[number];

export function emptyFactoryVehicleCounts(): Record<FactoryVehicleState, number> {
  return Object.fromEntries(FACTORY_VEHICLE_STATES.map((key) => [key, 0])) as Record<FactoryVehicleState, number>;
}

export function canonicalizeFactoryVehicle(status: string): string {
  const key = String(status || "").trim().toUpperCase();
  if (key === "READY" || key === "AVAILABLE") return key;
  return "UNAVAILABLE";
}

export const FACTORY_DRIVER_DUTY = ["ON_SHIFT", "OFF_SHIFT"] as const;
export type FactoryDriverDuty = (typeof FACTORY_DRIVER_DUTY)[number];

export function emptyFactoryDriverDuty(): Record<FactoryDriverDuty, number> {
  return Object.fromEntries(FACTORY_DRIVER_DUTY.map((key) => [key, 0])) as Record<FactoryDriverDuty, number>;
}

export const FACTORY_SLA_STATUSES = ["ON_TIME", "AT_RISK", "BREACHED", "MET", "N/A"] as const;
export type FactorySLAStatus = (typeof FACTORY_SLA_STATUSES)[number];

export const FACTORY_QC_RESULTS = ["PASS", "FAIL", "MISSING"] as const;
export type FactoryQCResult = (typeof FACTORY_QC_RESULTS)[number];

export interface FactoryDashboardException {
  exception_id?: string;
  manifest_id?: string;
  transfer_id?: string;
  reason?: string;
  escalated?: boolean;
}

export interface FactoryDashboardResponse {
  source?: "spanner" | "memory" | "empty" | string;
  plane?: "factory_trucks" | string;
  pending_transfers?: number;
  loading_transfers?: number;
  active_manifests?: number;
  dispatched_today?: number;
  dispatched_transfers?: number;
  vehicles_total?: number;
  vehicles_available?: number;
  staff_on_shift?: number;
  staff_total?: number;
  critical_insights?: number;
  transfers_by_state?: Record<string, number>;
  manifests_by_state?: Record<string, number>;
  vehicles_by_state?: Record<string, number>;
  driver_duty?: Record<string, number>;
  sla_by_status?: Record<string, number>;
  qc_by_result?: Record<string, number>;
  qc_available?: boolean;
  bay_loading_transfers?: number;
  bay_loading_manifests?: number;
  exceptions?: FactoryDashboardException[];
  supplier_id?: string;
  factory_id?: string;
  updated_at?: string;
}

export type TruckDutyStatus =
  | "AVAILABLE"
  | "IN_TRANSIT"
  | "RETURNING_TO_WAREHOUSE"
  | "OFF_SHIFT"
  | "UNASSIGNED"
  | "VEHICLE_INACTIVE"
  | "UNAVAILABLE"
  | "INACTIVE";

export const TRUCK_DUTY_STATUSES = [
  "AVAILABLE",
  "IN_TRANSIT",
  "RETURNING_TO_WAREHOUSE",
  "OFF_SHIFT",
  "UNASSIGNED",
  "VEHICLE_INACTIVE",
  "UNAVAILABLE",
  "INACTIVE",
] as const satisfies readonly TruckDutyStatus[];

export type TruckDutyFunnel = (typeof TRUCK_DUTY_STATUSES)[number];

export function emptyTruckDutyCounts(): Record<TruckDutyFunnel, number> {
  return Object.fromEntries(TRUCK_DUTY_STATUSES.map((key) => [key, 0])) as Record<TruckDutyFunnel, number>;
}

export function canonicalizeTruckDuty(status: string): string {
  const key = String(status || "").trim().toUpperCase();
  if (key === "RETURNING" || key === "RETURNING_TO_WH") return "RETURNING_TO_WAREHOUSE";
  if (key === "IDLE") return "AVAILABLE";
  if (key === "FULL" || key === "NEEDS_RESCUE") return "UNAVAILABLE";
  return key;
}

export const FISCAL_STATUSES = [
  "NONE",
  "PENDING",
  "SUCCESS",
  "FAILED",
  "FORCE_SKIPPED",
] as const satisfies readonly FiscalStatus[];

export type PlaybookRunStatus = "SUGGESTED" | "APPROVED" | "EXECUTED" | "FAILED" | "SKIPPED";

export const PLAYBOOK_RUN_STATUSES = [
  "SUGGESTED",
  "APPROVED",
  "EXECUTED",
  "FAILED",
  "SKIPPED",
] as const satisfies readonly PlaybookRunStatus[];

export type HistorySeriesSource = "live" | "empty" | "unavailable";

export interface HistorySeries {
  points: number[];
  source: HistorySeriesSource;
  available: boolean;
  generated_at?: string;
}

export type DashboardHistoryRange = "today" | "7d" | "30d";

export function historyRangeDays(range: DashboardHistoryRange): number {
  if (range === "today") return 1;
  if (range === "7d") return 7;
  return 30;
}

export function sliceDatedSeries<T extends { date: string }>(
  rows: T[] | null | undefined,
  range: DashboardHistoryRange,
  now: Date,
): T[] {
  const days = historyRangeDays(range);
  const end = Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate());
  const start = new Date(end);
  start.setUTCDate(start.getUTCDate() - (days - 1));
  const startKey = start.toISOString().slice(0, 10);
  return (rows ?? []).filter((row) => String(row.date ?? "").slice(0, 10) >= startKey);
}

/** Δ vs yesterday only when both today and the previous calendar day exist and yesterday ≠ 0. */
export function yesterdayRevenueDeltaPct(
  series: { date: string; revenue_minor: number }[] | null | undefined,
  todayDate: string,
): number | null {
  if (!series || series.length < 2) return null;
  const todayKey = String(todayDate || "").slice(0, 10);
  if (!/^\d{4}-\d{2}-\d{2}$/.test(todayKey)) return null;
  const today = series.find((point) => String(point.date).slice(0, 10) === todayKey);
  const prevDate = new Date(`${todayKey}T00:00:00.000Z`);
  if (!today || Number.isNaN(prevDate.getTime())) return null;
  prevDate.setUTCDate(prevDate.getUTCDate() - 1);
  const prevKey = prevDate.toISOString().slice(0, 10);
  const prev = series.find((point) => String(point.date).slice(0, 10) === prevKey);
  if (!prev || !Number.isFinite(prev.revenue_minor) || prev.revenue_minor === 0) return null;
  if (!Number.isFinite(today.revenue_minor)) return null;
  return ((today.revenue_minor - prev.revenue_minor) / Math.abs(prev.revenue_minor)) * 100;
}

export function historySeriesFromValues(
  values: number[] | undefined,
  available: boolean,
): HistorySeries {
  const points = (values ?? []).filter((n) => Number.isFinite(n));
  if (!available) {
    return { points: [], source: "unavailable", available: false };
  }
  if (points.length === 0) {
    return { points: [], source: "empty", available: true };
  }
  return { points, source: "live", available: true };
}

/** Draw a spark only when the series is live and has at least two points. */
export function guardHistorySeries(series?: HistorySeries | null): HistorySeries | null {
  if (!series || series.available !== true) return null;
  if (series.source === "unavailable" || series.source === "empty") return null;
  if (!Array.isArray(series.points) || series.points.length < 2) return null;
  if (series.points.some((n) => !Number.isFinite(n))) return null;
  return series;
}

export type StatusStackMode = "empty" | "zero" | "live" | "unavailable";

export type StatusStackRow = {
  key: string;
  count: number | null;
  share: number;
};

export type StatusStackModel = {
  mode: StatusStackMode;
  rows: StatusStackRow[];
  total: number;
};

/** Empty ≠ zero ≠ unavailable. Dictionary keys stay visible on zero/live. */
export function statusStackModel(
  dictionary: readonly string[],
  counts?: Record<string, number> | null,
  available = true,
): StatusStackModel {
  if (!available) {
    return {
      mode: "unavailable",
      total: 0,
      rows: dictionary.map((key) => ({ key, count: null, share: 0 })),
    };
  }
  if (counts == null) {
    return { mode: "empty", total: 0, rows: [] };
  }
  const rows: StatusStackRow[] = dictionary.map((key) => ({
    key,
    count: Number.isFinite(Number(counts[key])) ? Number(counts[key]) : 0,
    share: 0,
  }));
  const total = rows.reduce((sum, row) => sum + (row.count ?? 0), 0);
  if (total > 0) {
    for (const row of rows) {
      row.share = (row.count ?? 0) / total;
    }
  }
  return { mode: total === 0 ? "zero" : "live", rows, total };
}

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
  currency?: string;
  supplier_orders: SupplierOrderCheckoutResult[];
  backordered_item_count?: number;
  backorder_order_id?: string;
  stock_warnings?: StockWarning[];
}

/** GET /v1/order/currencies — flag-gated order currency picker (theatre #13 Wave 2+). */
export interface OrderCurrencyOptions {
  enabled: boolean;
  operating_currency: Iso4217;
  allowlist: Iso4217[];
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
  shortfall?: Record<string, number>;
  stock_warnings?: StockWarning[];
  max_quantities?: Record<string, number>;
  orderable_quantities?: Record<string, number>;
  line_errors?: Record<string, string>;
  backordered_item_count?: number;
  show_stock_counts?: boolean;
  preorder_min_lead_days?: number;
  preorder_max_lead_days?: number;
  order_line_min_quantity?: number;
  order_line_max_quantity?: number;
  delivery_fee_minor?: number;
  delivery_distance_km?: number;
  default_out_of_stock_policy?: OutOfStockPolicy;
  checkout_policy_token?: string;
  checkout_policy_expires_at?: string;
  order_acceptance_open?: boolean;
  order_acceptance_window_label?: string;
  next_order_acceptance_at?: string;
}

export interface DeliveryFeeTier {
  max_km?: number | null;
  fee_minor: number;
}

export interface DeliveryFeeRules {
  currency: string;
  base_fee_minor: number;
  tiers: DeliveryFeeTier[];
}

export interface WarehouseOpsSettings {
  warehouse_id: string;
  name: string;
  region_id?: string;
  default_out_of_stock_policy: OutOfStockPolicy;
  show_stock_counts_to_retailers?: boolean;
  operating_schedule?: Record<string, unknown>;
  is_on_shift: boolean;
  ops_always_available: boolean;
  express_enabled?: boolean;
  express_stock_floor?: number;
  preorder_min_lead_days?: number;
  preorder_max_lead_days?: number;
  order_line_min_quantity?: number | null;
  order_line_max_quantity?: number | null;
  delivery_fee_rules?: DeliveryFeeRules | null;
}

export interface WarehouseOpsSettingsPatchRequest {
  default_out_of_stock_policy?: OutOfStockPolicy;
  show_stock_counts_to_retailers?: boolean;
  operating_schedule?: Record<string, unknown>;
  express_enabled?: boolean;
  express_stock_floor?: number;
  preorder_min_lead_days?: number;
  preorder_max_lead_days?: number;
  order_line_min_quantity?: number | null;
  order_line_max_quantity?: number | null;
  clear_order_line_min_quantity?: boolean;
  clear_order_line_max_quantity?: boolean;
  delivery_fee_rules?: DeliveryFeeRules | null;
  clear_delivery_fee_rules?: boolean;
}

export interface WarehouseInventoryPolicyPatchRequest {
  out_of_stock_policy?: OutOfStockPolicy;
  reorder_threshold?: number;
}

export type OrderConfirmationStatus = "CONFIRMED" | "DRAFT" | "PENDING" | "REJECTED" | "AUTO_CONFIRMED";


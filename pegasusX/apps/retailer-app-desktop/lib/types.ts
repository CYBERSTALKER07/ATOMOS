// lib/types.ts — API response shapes matching Go backend JSON tags

/* ── Orders ── */
export interface LineItem {
  line_item_id: string;
  order_id: string;
  sku_id: string;
  sku_name?: string;
  quantity: number;
  unit_price: number;
  status: string;
}

export interface Order {
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  amount: number;
  payment_gateway: string;
  payment_status: string;
  state: string;
  route_id: string | null;
  order_source: string | null;
  confirmation_status?: string | null;
  preorder_badge?: string | null;
  proposed_delivery_date?: string | null;
  delivery_proposal_reason?: string | null;
  auto_confirm_at: string | null;
  deliver_before: string | null;
  delivery_token: string | null;
  version: number;
  created_at: string;
  items?: LineItem[];
}

export interface ProcurementOrderResponse {
  status: "PROCUREMENT_AUTHORIZED";
  order_id: string;
  retailer_id: string;
  supplier_id?: string;
  state: string;
  amount: number;
  total: number;
  currency: string;
  order_source: "PROCUREMENT";
  created_at: string;
}

/* ── Catalog ── */
export interface Variant {
  id: string;
  size: string;
  pack: string;
  pack_count: number;
  weight_per_unit: string;
  price: number;
}

export interface ProductOffer {
  product_id?: string;
  list_price_minor?: number;
  sale_price_minor?: number;
  discount_bps?: number;
  promotion_label?: string;
}

export interface Product {
  id: string;
  product_id?: string;
  name: string;
  description: string;
  nutrition?: string;
  image_url: string;
  supplier_id: string;
  supplier_name: string;
  supplier_category: string;
  category_id: string;
  category_name: string;
  sell_by_block: boolean;
  units_per_block: number;
  price: number;
  variants?: Variant[];
  available_stock?: number;
  is_out_of_stock?: boolean;
  accepts_backorder?: boolean;
  show_stock_counts?: boolean;
  max_quantity?: number | null;
  offer?: ProductOffer;
}

export function productSalePrice(product: Product): number | null {
  const sale = product.offer?.sale_price_minor;
  return sale != null && sale > 0 ? sale : null;
}

export function productListPrice(product: Product): number {
  return (
    product.offer?.list_price_minor ??
    product.price ??
    0
  );
}

export function productDisplayPrice(product: Product): number {
  return productSalePrice(product) ?? productListPrice(product);
}

export interface Category {
  id: string;
  name: string;
  icon: string;
  product_count: number;
  supplier_count: number;
}

/* ── Suppliers ── */
export interface Supplier {
  id: string;
  name: string;
  logo_url: string;
  category: string;
  primary_category_id?: string;
  operating_category_ids?: string[];
  operating_category_names?: string[];
  order_count: number;
  is_active: boolean;
}

/* ── Analytics ── */
export interface MonthlyExpense {
  month: string;
  total: number;
}
export interface TopSupplier {
  supplier_id: string;
  supplier_name: string;
  total: number;
  order_count: number;
}
export interface TopProduct {
  product_id: string;
  product_name: string;
  total: number;
  quantity: number;
}
export interface RetailerAnalytics {
  monthly_expenses: MonthlyExpense[];
  top_suppliers: TopSupplier[];
  top_products: TopProduct[];
  total_this_month: number;
  total_last_month: number;
}

/* ── AI Predictions (legacy SKU-forecast card — not the live retailer list) ── */
export interface Prediction {
  id: string;
  retailer_id?: string;
  retailerId: string;
  product_id?: string;
  product_name?: string;
  predicted_amount?: number;
  predicted_quantity?: number;
  trigger_date?: string;
  suggested_order_date?: string;
  predictedAmount: number;
  triggerDate: string;
  status: string;
  productName: string;
  predictedQuantity: number;
  confidence: number;
  reasoning: string;
  suggestedOrderDate: string;
  blocked?: boolean;
  blocked_reason?: string;
  label?: "insufficient_history" | "early_signal" | "standard" | string;
}

export function isPredictionBlocked(
  item: Pick<Prediction, "blocked" | "blocked_reason" | "label">,
): boolean {
  return item.blocked === true
    || item.label === "insufficient_history"
    || Boolean(item.blocked_reason);
}

/** Live GET /v1/retailer/ai/predictions item — pending AI preorder, not a SKU forecast. */
export interface RetailerAILineItem {
  sku: string;
  name?: string;
  quantity: number;
  unit_price_minor: number;
}

export interface RetailerAIPrediction {
  order_id: string;
  order_source?: string;
  confirmation_status?: string;
  requested_delivery_date?: string;
  auto_confirm_at?: string;
  total_minor: number;
  currency: string;
  derived_from_order_id?: string;
  updated_at: string;
  line_items?: RetailerAILineItem[];
}

export interface RetailerAIPredictionsResponse {
  items: RetailerAIPrediction[];
}

export function aiPredictionTitle(item: RetailerAIPrediction): string {
  const first = item.line_items?.[0];
  const label = first?.name || first?.sku;
  if (label) return label;
  return item.order_id;
}

export function aiPredictionQty(item: RetailerAIPrediction): number {
  return (item.line_items ?? []).reduce((sum, line) => sum + (line.quantity ?? 0), 0);
}

export function formatMinorAmount(minor: number, currency?: string): string {
  const units = (minor ?? 0) / 100;
  const code = (currency ?? "").trim();
  return code ? `${units.toLocaleString()} ${code}` : units.toLocaleString();
}

/* ── Auto-Order Settings ── */
export interface SupplierOverride {
  supplier_id: string;
  enabled: boolean;
  has_history: boolean;
  analytics_start_date?: string;
}
export interface CategoryOverride {
  category_id: string;
  enabled: boolean;
  has_history: boolean;
  analytics_start_date?: string;
}
export interface ProductOverride {
  product_id: string;
  enabled: boolean;
}
export interface VariantOverride {
  variant_id: string;
  enabled: boolean;
}
export type AutoOrderExecutionMode = "off" | "shadow" | "draft" | "place";

export interface AutoOrderShadowStats {
  proposal_count: number;
  matched_orders: number;
  wape: number;
  unmodified_accept_rate: number;
  window_days: number;
}

export interface AutoOrderShadowProposal {
  proposal_id: string;
  retailer_id: string;
  sku: string;
  supplier_id?: string;
  proposed_qty: number;
  ip: number;
  reorder_point: number;
  order_up_to: number;
  confidence?: number;
  reason?: string;
  bucket_date: string;
  status: string;
  run_id?: string;
  created_at?: string;
}

export interface AutoOrderSettings {
  global_enabled: boolean;
  /** off | shadow | draft | place — place creates real supplier orders when flag on */
  execution_mode?: AutoOrderExecutionMode | string;
  has_any_history: boolean;
  analytics_start_date?: string;
  supplier_overrides: SupplierOverride[];
  category_overrides: CategoryOverride[];
  product_overrides: ProductOverride[];
  variant_overrides: VariantOverride[];
  shadow_stats?: AutoOrderShadowStats;
}

/** POST /v1/retailer/settings/auto-order/run audit row */
export interface AutoOrderSkip {
  sku?: string;
  reason: string;
}

export interface AutoOrderPlacedOrder {
  order_id: string;
  supplier_id?: string;
  line_count: number;
  total_minor: number;
  skus?: string[];
}

export interface AutoOrderRun {
  run_id: string;
  retailer_id: string;
  started_at: string;
  finished_at?: string;
  mode: AutoOrderExecutionMode | string;
  draft_lines: number;
  placed_lines?: number;
  placed_orders?: AutoOrderPlacedOrder[];
  skipped?: AutoOrderSkip[];
  status: string;
  message?: string;
  suggestions_seen?: number;
  schedule_bucket?: string;
  candidate_source?: string;
}

export interface AutoOrderRunsResponse {
  items: AutoOrderRun[];
}

/** Family → Team migrate wire shapes */
export interface FamilyMigrateItem {
  member_id: string;
  user_id: string;
  phone: string;
  name: string;
  retailer_role: string;
  temp_password?: string;
}

export interface FamilyMigrateSkipped {
  member_id: string;
  phone?: string;
  reason: string;
}

export interface FamilyMigrateResult {
  retailer_id: string;
  migrated: FamilyMigrateItem[];
  skipped: FamilyMigrateSkipped[];
  family_remaining: number;
  family_writes: string;
}

export interface FamilyMembersListResponse {
  members: Array<{
    member_id: string;
    name: string;
    phone?: string;
    created_at?: string;
  }>;
  family_writes?: "open" | "gone" | string;
  migrate?: string;
}

/* ── Checkout ── */
export interface SupplierOrderResult {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  total: number;
  item_count: number;
}

export interface StockWarning {
  sku: string;
  requested: number;
  available: number;
  backorder_qty: number;
  accepts_backorder: boolean;
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
  default_out_of_stock_policy?: string;
  checkout_policy_token?: string;
  checkout_policy_expires_at?: string;
  order_acceptance_open?: boolean;
  order_acceptance_window_label?: string;
  next_order_acceptance_at?: string;
}

export interface UnifiedCheckoutResponse {
  status: string;
  invoice_id: string;
  total: number;
  currency?: string;
  supplier_orders: SupplierOrderResult[];
  backordered_item_count?: number;
  backorder_order_id?: string;
  stock_warnings?: StockWarning[];
}

/** GET /v1/order/currencies — flag-gated order currency picker. */
export interface OrderCurrencyOptions {
  enabled: boolean;
  operating_currency: string;
  allowlist: string[];
}

export interface CashCheckoutResponse {
  order_id: string;
  state: string;
  amount: number;
  driver_id?: string;
  retailer_id: string;
  message: string;
}

export interface CardCheckoutResponse {
  order_id: string;
  state: string;
  amount: number;
  gateway: string;
  resolved_gateway?: string;
  policy_source?: string;
  allowed_gateways?: string[];
  policy_reason?: string;
  payment_url: string;
  invoice_id: string;
  session_id?: string;
  attempt_id?: string;
  attempt_no?: number;
  retailer_id: string;
  message: string;
}

export interface ActiveFulfillmentItem {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  state: string;
  adjusted_amount: number;
  item_count: number;
}

export interface ActiveFulfillmentsResponse {
  fulfillments: ActiveFulfillmentItem[];
  count: number;
}

export interface PendingPaymentSession {
  session_id?: string;
  order_id: string;
  retailer_id: string;
  supplier_id: string;
  gateway?: string;
  locked_amount: number;
  currency: string;
  status: string;
  current_attempt_no: number;
  invoice_id?: string;
  redirect_url?: string;
  expires_at?: string;
  created_at: string;
  updated_at?: string;
}

export interface PendingPaymentsResponse {
  pending_payments: PendingPaymentSession[];
  count: number;
}

/* ── Cancel ── */
export interface CancelOrderRequest {
  order_id: string;
  retailer_id: string;
  version: number;
}

/* ── Retailer Profile (from login response) ── */
export interface RetailerProfile {
  id: string;
  name: string;
  company: string;
  email: string;
  avatar_url: string | null;
  // Phase F receiving-window fields. HH:MM canonical form per
  // proximity.ValidateReceivingWindow on the backend. Optional because legacy
  // retailer rows pre-Phase F have NULL columns.
  receiving_window_open?: string | null;
  receiving_window_close?: string | null;
  country_code?: string;
}

/* ── Line Items History ── */
export interface LineItemHistory {
  skuId: string;
  quantity: number;
  unitPrice: number;
  orderDate: string;
  minimumOrderQty: number;
  stepSize: number;
}

/* ── Delivery Tracking ── */
export interface TrackingOrderItem {
  product_id: string;
  product_name: string;
  quantity: number;
  unit_price: number;
  line_total: number;
}

export interface TrackingRouteGeometry {
  route_id?: string;
  encoded_polyline?: string;
  coordinates: Array<{ lat: number; lng: number }>;
  source: string;
  stop_count?: number;
}

export interface TrackingOrder {
  order_id: string;
  supplier_id: string;
  supplier_name: string;
  driver_id: string;
  state: string;
  total_amount: number;
  order_source: string;
  driver_latitude: number | null;
  driver_longitude: number | null;
  is_approaching: boolean;
  delivery_token: string;
  created_at: string;
  fiscal_status?: string;
  fiscal_qr?: string;
  latest_fiscal_receipt_id?: string;
  items: TrackingOrderItem[];
  route_geometry?: TrackingRouteGeometry;
}

export interface TrackingResponse {
  status?: "idle" | "active";
  orders: TrackingOrder[];
  recent_receipts?: TrackingOrder[];
  events?: Array<{
    event_type: string;
    order_id: string;
    status?: string;
    occurred_at: string;
  }>;
}

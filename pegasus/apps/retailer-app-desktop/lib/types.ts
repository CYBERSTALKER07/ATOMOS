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
  auto_confirm_at: string | null;
  deliver_before: string | null;
  delivery_token: string | null;
  version: number;
  created_at: string;
  items?: LineItem[];
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

export interface Product {
  id: string;
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

/* ── AI Predictions ── */
export interface Prediction {
  id: string;
  retailerId: string;
  predictedAmount: number;
  triggerDate: string;
  status: string;
  productName: string;
  predictedQuantity: number;
  confidence: number;
  reasoning: string;
  suggestedOrderDate: string;
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
export interface AutoOrderSettings {
  global_enabled: boolean;
  has_any_history: boolean;
  analytics_start_date?: string;
  supplier_overrides: SupplierOverride[];
  category_overrides: CategoryOverride[];
  product_overrides: ProductOverride[];
  variant_overrides: VariantOverride[];
}

/* ── Checkout ── */
export interface SupplierOrderResult {
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
  supplier_orders: SupplierOrderResult[];
  backordered_item_count?: number;
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
  items: TrackingOrderItem[];
}

export interface TrackingResponse {
  orders: TrackingOrder[];
}

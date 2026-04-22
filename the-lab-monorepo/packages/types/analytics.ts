/**
 * @file packages/types/analytics.ts
 * @description Analytics, predictions, and dashboard metric types.
 * Sync with: apps/backend-go/analytics/, apps/backend-go/order/ai_preorder.go
 */

// ─── Supplier Dashboard Metrics ─────────────────────────────────────────────
export interface SupplierDashboardMetrics {
  active_orders: number;
  completed_today: number;
  revenue_today: number;
  currency: string;
  pending_dispatches: number;
  active_drivers: number;
  active_vehicles: number;
}

// ─── SKU Velocity ───────────────────────────────────────────────────────────
export interface SkuVelocity {
  sku_id: string;
  sku_name: string;
  daily_velocity: number;
  weekly_avg: number;
  monthly_avg: number;
  trend: 'UP' | 'DOWN' | 'STABLE';
}

// ─── Demand Forecast ────────────────────────────────────────────────────────
export interface DemandSummaryItem {
  retailer_id: string;
  retailer_name: string;
  predicted_total: number;
  currency: string;
  confidence: number;
  sku_count: number;
}

export interface DemandHistoryPoint {
  date: string;
  actual: number;
  predicted: number;
}

export interface DemandDetailRow {
  sku_id: string;
  sku_name: string;
  predicted_quantity: number;
  confidence: number;
  last_ordered_at: string | null;
}

// ─── AI Prediction ──────────────────────────────────────────────────────────
export interface AIPredictionItem {
  sku_id: string;
  sku_name: string;
  predicted_quantity: number;
  unit_price: number;
  subtotal: number;
  currency: string;
  confidence: number;            // 0.0 – 1.0
}

export interface AIPredictionResult {
  retailer_id: string;
  items: AIPredictionItem[];
  total: number;
  currency: string;
  generated_at: string;
}

// ─── Retailer Analytics ─────────────────────────────────────────────────────
export interface RetailerAnalyticsResponse {
  total_spent: number;
  order_count: number;
  avg_order: number;
  currency: string;
  monthly_expenses: MonthlyExpense[];
  top_suppliers: TopEntity[];
  top_products: TopEntity[];
}

export interface MonthlyExpense {
  month: string;
  amount: number;
  order_count: number;
}

export interface TopEntity {
  id: string;
  name: string;
  amount: number;
  order_count: number;
}

// ─── Empathy (Auto-Order) Adoption ──────────────────────────────────────────
export interface EmpathyAdoption {
  retailer_id: string;
  retailer_name: string;
  global_enabled: boolean;
  active_sku_count: number;
  total_sku_count: number;
  adoption_rate: number;         // 0.0 – 1.0
}

// ─── Process Metrics (GET /v1/metrics) ──────────────────────────────────────
export interface ProcessMetrics {
  uptime_seconds: number;
  go_version: string;
  goroutines: number;
  num_cpu: number;
  total_requests: number;
  active_requests: number;
  total_errors: number;
  ws_connections: number;
  memory: {
    alloc_mb: number;
    total_alloc_mb: number;
    sys_mb: number;
    heap_objects: number;
    gc_cycles: number;
    gc_pause_total_ms: number;
  };
}

// ─── Health Check ───────────────────────────────────────────────────────────
export interface HealthCheckResponse {
  status: 'ok' | 'degraded';
  spanner: boolean;
  redis: boolean;
  time: string;
}

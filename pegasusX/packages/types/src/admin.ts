// ── Platform Admin DTO Types ──────────────────────────────────────────────────
export interface Tenant {
  TenantType: string;
  TenantID: string;
  Status: string;
  DisplayName: string;
  KybNotes: string;
  market_code?: string;
  home_cell?: string;
  CreatedAt: string;
  UpdatedAt: string;
  ApprovedAt?: string | null;
  SuspendedAt?: string | null;
  OffboardedAt?: string | null;
}

export interface FlagOverride {
  FlagKey: string;
  TenantType: string;
  TenantID: string;
  Enabled: boolean;
  Status: string;
  Reason?: string;
  UpdatedBy?: string;
}

export interface FlagEval {
  flag_key: string;
  enabled: boolean;
  source: string;
  money_affecting: boolean;
}

export interface AccuracyRow {
  supplier_id: string;
  forecast_date: string;
  warehouse_id: string;
  product_id: string;
  mape28: number;
  wape28: number;
  demoted: boolean;
}

export interface AuditRow {
  AuditID: string;
  ActorSubject: string;
  Action: string;
  TenantType: string;
  TenantID: string;
  DetailJSON: string;
  CreatedAt: string;
}

export interface MatchQueueItem {
  queue_id: string;
  supplier_id: string;
  product_id: string;
  candidate_global_product_id?: string;
  match_method: string;
  score: number;
  status: string;
  reason?: string;
}

export interface PartnerKey {
  key_id: string;
  tenant_type: string;
  tenant_id: string;
  key_prefix: string;
  scopes: string[];
  status: string;
}

export interface BillingInvoice {
  invoice_id: string;
  billed_supplier_id: string;
  order_id: string;
  status: string;
  principal_minor: number;
  balance_minor: number;
  currency: string;
  due_at: string;
  created_at: string;
}

export interface BillingFeeSchedule {
  fee_schedule_id: string;
  supplier_id: string;
  tier: string;
  per_order_minor: number;
  gmv_bps: number;
  monthly_subscription_minor: number;
  currency: string;
}


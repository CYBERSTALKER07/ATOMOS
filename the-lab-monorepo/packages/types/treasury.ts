/**
 * @file packages/types/treasury.ts
 * @description Treasury, settlement, and reconciliation types.
 * Sync with: apps/backend-go/treasury/, apps/backend-go/admin/reconciliation.go
 */

// ─── Treasury Report ────────────────────────────────────────────────────────
export interface TreasuryReport {
  total_revenue: number;
  platform_commission: number;
  supplier_payable: number;
  pending_settlements: number;
  cash_in_transit: number;
  currency: string;
  last_reconciled_at: string | null;
}

// ─── Cash Holdings ──────────────────────────────────────────────────────────
export interface CashHolding {
  holding_id: string;
  driver_id: string;
  driver_name: string;
  order_id: string;
  amount: number;
  currency: string;
  status: CashHoldingStatus;
  collected_at: string;
  deposited_at: string | null;
}

export type CashHoldingStatus = 'PENDING' | 'COLLECTED' | 'DEPOSITED';

export interface CashHoldingsReport {
  holdings: CashHolding[];
  summary: {
    total_pending: number;
    total_collected: number;
    total_deposited: number;
    count: number;
  };
}

// ─── Settlement Report ──────────────────────────────────────────────────────
export interface SettlementRow {
  order_id: string;
  invoice_id: string;
  amount: number;
  currency: string;
  payment_mode: string;
  invoice_status: string;
  paid_at: string | null;
}

export interface SettlementSummary {
  total_settled: number;
  total_pending: number;
  settled_count: number;
  pending_count: number;
}

export interface SettlementReportResponse {
  rows: SettlementRow[];
  summary: SettlementSummary;
  from: string;
  to: string;
}

// ─── Reconciliation ─────────────────────────────────────────────────────────
export interface ReconciliationRecord {
  record_id: string;
  order_id: string;
  invoice_id: string;
  expected_amount: number;
  actual_amount: number;
  discrepancy: number;
  currency: string;
  status: ReconciliationStatus;
  resolved_at: string | null;
  created_at: string;
}

export type ReconciliationStatus = 'PENDING' | 'MATCHED' | 'DISCREPANCY' | 'RESOLVED';

// ─── Supplier Earnings ──────────────────────────────────────────────────────
export interface SupplierEarningsResponse {
  net_total: number;
  commission_estimate: number;
  gross_revenue: number;
  currency: string;
  monthly_breakdown: MonthlyRevenue[];
}

export interface MonthlyRevenue {
  month: string;       // "2026-04"
  gross: number;
  net: number;
  order_count: number;
}

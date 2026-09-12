import { Iso4217, OrderId, PaymentGateway, RetailerId, SupplierId } from "./primitives";
import { PaymentLedgerEntry } from "./compliance";

// ── Logistics claims (post-delivery OS&D / chargeback settle) ─────────────────
// Mirrors apps/backend-go/claims types + GET /v1/supplier/claim-chargebacks.

export type ClaimType =
  | "DAMAGED"
  | "MISSING"
  | "CONCEALED_DAMAGE"
  | "TEMPERATURE"
  | "TAMPER"
  | "OTHER"
  | string;

export type ClaimStatus =
  | "OPEN"
  | "UNDER_REVIEW"
  | "APPROVED"
  | "REJECTED"
  | "RESOLVED"
  | string;

export type ClaimSource =
  | "RETAILER_CLAIM"
  | "DRIVER_EXCEPTION"
  | "TELEMETRY"
  | string;

/** Approve settlement modes accepted by POST /v1/claims/{id}/approve. */
export type ClaimSettlementMode = "LEDGER_ONLY" | "STORE_CREDIT" | "GATEWAY_REFUND";

export const CLAIM_SETTLEMENT_MODES: readonly {
  value: ClaimSettlementMode;
  label: string;
  hint: string;
}[] = [
  {
    value: "LEDGER_ONLY",
    label: "Ledger only",
    hint: "Debit supplier settlement only (safe default)",
  },
  {
    value: "STORE_CREDIT",
    label: "Store credit",
    hint: "Ledger + reduce retailer credit balance due",
  },
  {
    value: "GATEWAY_REFUND",
    label: "Card refund (GP)",
    hint: "Ledger + Global Pay partial refund when session is card",
  },
] as const;

export interface ClaimLine {
  sku: string;
  quantity: number;
  reason?: string;
  unit_price_minor?: number;
  amount_minor?: number;
}

export interface ClaimEvidence {
  evidence_id?: string;
  claim_id?: string;
  evidence_type: string;
  uri: string;
  mime_type?: string;
  captured_at?: string;
  captured_by?: string;
  created_at?: string;
}

export interface Claim {
  claim_id: string;
  order_id: OrderId;
  supplier_id?: SupplierId;
  retailer_id: RetailerId;
  filed_by?: string;
  filed_by_role?: string;
  claim_type: ClaimType;
  status: ClaimStatus;
  description?: string;
  amount_minor?: number;
  currency?: Iso4217 | string;
  line_items?: ClaimLine[];
  evidences?: ClaimEvidence[];
  resolution_note?: string;
  resolved_by?: string;
  resolved_at?: string;
  source?: ClaimSource;
  trace_id?: string;
  created_at: string;
  updated_at?: string;
}

export interface SupplierClaimsListResponse {
  claims: Claim[];
}

export interface ApproveClaimRequest {
  resolution_note?: string;
  amount_minor?: number;
  /** When true, force ledger-only even if mode would call gateway. */
  skip_gateway_refund?: boolean;
  settlement_mode?: ClaimSettlementMode | string;
}

export interface RejectClaimRequest {
  resolution_note?: string;
}

/** Result of chargeback adjudication after approve. */
export interface ClaimSettlementResult {
  chargeback_id?: string;
  amount_minor: number;
  currency?: Iso4217 | string;
  gateway?: string;
  gateway_refunded?: boolean;
  provider_ref?: string;
  /** LEDGER_ONLY | LEDGER_AND_GATEWAY_REFUND | LEDGER_AND_STORE_CREDIT | IDEMPOTENT_REPLAY | … */
  mode: string;
  idempotent?: boolean;
}

export interface ApproveClaimResponse {
  claim: Claim;
  settlement?: ClaimSettlementResult;
}

/** GET /v1/supplier/claim-chargebacks — claim-originated ledger rows. */
export interface ClaimChargebacksQuery {
  limit?: number;
  order_id?: OrderId;
}

export interface ClaimChargebacksResponse {
  items: PaymentLedgerEntry[];
  count: number;
  limit: number;
  supplier_id: SupplierId | "";
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


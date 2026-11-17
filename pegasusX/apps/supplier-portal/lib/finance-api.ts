import type { ApiClient } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { formatMinorUnits, parseMajorToMinorBigInt } from "@/lib/money";

function withQuery(path: string, query: Record<string, string | number | undefined>): string {
  const params = new URLSearchParams();
  for (const [k, v] of Object.entries(query)) {
    if (v !== undefined && v !== null && String(v).trim() !== "") {
      params.set(k, String(v).trim());
    }
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

export type GlDirection = "DEBIT" | "CREDIT";

export type GlAccountClass = "ASSET" | "LIABILITY" | "EQUITY" | "REVENUE" | "EXPENSE";

export interface CoaAccountMeta {
  account_code: string;
  class: GlAccountClass;
  name: string;
  is_control: boolean;
}

/**
 * Shipped 27-account Chart of Accounts reference matching backend `gl/coa.go`.
 * Used strictly as UI lookup/selector metadata (never synthetic balance data).
 */
export const CANONICAL_GL_ACCOUNTS: readonly CoaAccountMeta[] = [
  { account_code: "AR:RETAILER", class: "ASSET", name: "Accounts Receivable - Retailers", is_control: true },
  { account_code: "CASH:DRIVER", class: "ASSET", name: "Cash in Transit - Drivers", is_control: true },
  { account_code: "AR:DRIVER:VARIANCE", class: "ASSET", name: "Driver Shortage Receivable", is_control: true },
  { account_code: "CASH:OFFICE", class: "ASSET", name: "Cash Vault - Finance Office", is_control: false },
  { account_code: "PSP:GATEWAY", class: "ASSET", name: "Card / PSP Gateway Transit", is_control: true },
  { account_code: "BANK:OPERATING", class: "ASSET", name: "Operating Bank Account", is_control: false },
  { account_code: "TAX:VAT:INPUT", class: "ASSET", name: "Input VAT Recoverable", is_control: false },
  { account_code: "INVENTORY", class: "ASSET", name: "Merchandise Inventory", is_control: false },
  { account_code: "AR:SUPPLIER:FEES", class: "ASSET", name: "Platform Fees Receivable - Suppliers", is_control: true },
  { account_code: "AP:SUPPLIER", class: "LIABILITY", name: "Accounts Payable - Suppliers", is_control: true },
  { account_code: "ESCROW:ORDER", class: "LIABILITY", name: "Order Fulfillment Escrow Liability", is_control: true },
  { account_code: "WALLET:RETAILER", class: "LIABILITY", name: "Retailer Deposit / Credit Wallet", is_control: true },
  { account_code: "AP:DRIVER:VARIANCE", class: "LIABILITY", name: "Driver Cash Overage Suspense", is_control: true },
  { account_code: "TAX:VAT:OUTPUT", class: "LIABILITY", name: "Output VAT Payable", is_control: false },
  { account_code: "AP:VENDOR", class: "LIABILITY", name: "Accounts Payable - Vendors", is_control: true },
  { account_code: "GRNI", class: "LIABILITY", name: "Goods Received Not Invoiced", is_control: false },
  { account_code: "EQUITY:OPENING_BALANCE", class: "EQUITY", name: "Opening Balance Migration Offset", is_control: false },
  { account_code: "REV:PLATFORM:FEE", class: "REVENUE", name: "Platform Marketplace Fee Revenue", is_control: false },
  { account_code: "REV:CASH:OVERAGE", class: "REVENUE", name: "Cash Overage Surplus Revenue", is_control: false },
  { account_code: "REV:SALES", class: "REVENUE", name: "Merchandise Sales Revenue", is_control: false },
  { account_code: "EXP:PROMO:DISCOUNT", class: "EXPENSE", name: "Promotional Discounts & Allowances", is_control: false },
  { account_code: "EXP:CASH:WRITEOFF", class: "EXPENSE", name: "Cash Variance Write-Off Expense", is_control: false },
  { account_code: "EXP:CLAIMS:SETTLEMENT", class: "EXPENSE", name: "Claims Dispute Settlement Expense", is_control: false },
  { account_code: "ADJ:CREDITNOTE", class: "EXPENSE", name: "Credit Note Sales Adjustment", is_control: false },
  { account_code: "COGS", class: "EXPENSE", name: "Cost of Goods Sold", is_control: false },
  { account_code: "EXP:SHRINK", class: "EXPENSE", name: "Inventory Shrinkage & Spoilage Expense", is_control: false },
  { account_code: "FX:ROUNDING", class: "EXPENSE", name: "FX Conversion & Rounding Difference", is_control: false },
];

export interface TrialBalanceLine {
  account_code: string;
  account_name?: string;
  class?: string;
  currency: string;
  debit_minor: number;
  credit_minor: number;
  net_minor: number;
}

export interface TrialBalanceReport {
  entity_id: string;
  period_id: string;
  currency: string;
  total_debits_minor: number;
  total_credits_minor: number;
  balanced: boolean;
  accounts: TrialBalanceLine[];
}

export interface AccountLedgerRow {
  entry_id: string;
  period_id: string;
  line: number;
  account_code: string;
  direction: GlDirection;
  amount_minor: number;
  currency: string;
  functional_amount_minor: number;
  fx_rate_id?: string;
  party_type?: string;
  party_id?: string;
  source_type: string;
  source_id: string;
  order_id?: string;
  reverses_entry_id?: string;
  memo?: string;
  description?: string;
  entry_hash: string;
  occurred_at: string;
  posted_at: string;
}

export interface AccountLedgerResponse {
  entity_id: string;
  account_code: string;
  period_id: string;
  limit: number;
  count: number;
  items: AccountLedgerRow[];
}

export interface JournalPostingRecord {
  entity_id: string;
  entry_id: string;
  line: number;
  account_code: string;
  direction: GlDirection;
  amount_minor: number;
  currency: string;
  functional_amount_minor: number;
  fx_rate_id?: string;
  party_type?: string;
  party_id?: string;
  warehouse_id?: string;
  product_id?: string;
  lot_id?: string;
  description?: string;
}

export interface JournalEntryRecord {
  entity_id: string;
  entry_id: string;
  period_id: string;
  source_event_id?: string;
  source_type: string;
  source_id: string;
  order_id?: string;
  memo?: string;
  currency: string;
  country_pack: string;
  reverses_entry_id?: string;
  idempotency_key: string;
  prev_hash: string;
  entry_hash: string;
  posted_by: string;
  occurred_at: string;
  posted_at: string;
  postings?: JournalPostingRecord[];
}

export interface PostingInput {
  account_code: string;
  direction: GlDirection;
  amount_minor: number;
  currency?: string;
  functional_amount_minor?: number;
  fx_rate_id?: string;
  party_type?: string;
  party_id?: string;
  warehouse_id?: string;
  product_id?: string;
  lot_id?: string;
  description?: string;
}

export interface ManualJournalRequest {
  entity_id?: string;
  period_id?: string;
  source_type?: string;
  source_id: string;
  order_id?: string;
  memo: string;
  currency: string;
  country_pack?: string;
  idempotency_key?: string;
  occurred_at?: string;
  postings: PostingInput[];
}

export interface JournalEntryResult {
  entry_id: string;
  entity_id: string;
  period_id: string;
  prev_hash: string;
  hash: string;
  already_posted: boolean;
  total_debit_minor: number;
  total_credit_minor: number;
  line_count: number;
}

export interface ReverseJournalRequest {
  entity_id?: string;
  reason?: string;
  idempotency_key?: string;
}

export type ReconDimension = "ORDERS_AR" | "STOCK_GL" | "CASH_PSP" | "TAX_FISCAL";
export type ReconRunStatus = "GREEN" | "BREAKS_FOUND" | "FAILED";
export type ReconCheckStatus = "GREEN" | "BREAK";
export type ReconBreakStatus = "OPEN" | "ACKNOWLEDGED" | "RESOLVED" | "WAIVED";
export type ReconBreakSeverity = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

export interface ReconDimensionCheck {
  dimension: ReconDimension;
  status: ReconCheckStatus;
  expected_minor: number;
  actual_minor: number;
  delta_minor: number;
  break_count: number;
  exposure_minor: number;
  currency: string;
  metrics?: Record<string, number>;
}

export interface ReconRunSummary {
  green_dimensions: number;
  break_dimensions: number;
  total_breaks: number;
  critical_breaks: number;
  high_breaks: number;
  medium_breaks: number;
  low_breaks: number;
}

export interface ReconciliationRun {
  run_id: string;
  entity_id: string;
  period_id: string;
  mode: "MANUAL" | "NIGHTLY" | "TRIGGERED" | string;
  status: ReconRunStatus;
  break_count: number;
  exposure_minor: number;
  currency: string;
  checks: Partial<Record<ReconDimension, ReconDimensionCheck>>;
  summary?: ReconRunSummary;
  triggered_by: string;
  started_at: string;
  completed_at?: string;
}

export interface ReconciliationBreak {
  break_id: string;
  run_id: string;
  entity_id: string;
  dimension: ReconDimension;
  object_type: string;
  object_id: string;
  expected_minor: number;
  actual_minor: number;
  delta_minor: number;
  currency: string;
  severity: ReconBreakSeverity;
  status: ReconBreakStatus;
  owner_id?: string;
  sla_due_at?: string;
  resolution_note?: string;
  control_tower_exception_id?: string;
  created_at: string;
  updated_at: string;
}

export interface TriggerReconRunRequest {
  entity_id?: string;
  period_id?: string;
  mode?: "MANUAL" | "NIGHTLY" | "TRIGGERED";
  dimensions?: ReconDimension[];
}

export interface TriggerReconRunResponse {
  run: ReconciliationRun;
  breaks: ReconciliationBreak[];
}

export interface ListReconRunsResponse {
  entity_id: string;
  runs: ReconciliationRun[];
  count: number;
}

export interface ListReconBreaksResponse {
  entity_id: string;
  breaks: ReconciliationBreak[];
  count: number;
}

export interface UpdateReconBreakRequest {
  action: "ACKNOWLEDGE" | "RESOLVE" | "WAIVE";
  note?: string;
  resolution_note?: string;
  owner_id?: string;
}

export interface UpdateReconBreakResponse {
  break: ReconciliationBreak;
}

export const RECON_DIMENSIONS: readonly ReconDimension[] = [
  "ORDERS_AR",
  "STOCK_GL",
  "CASH_PSP",
  "TAX_FISCAL",
];

export interface FinanceRequestTransport {
  request: ApiClient["request"];
}

export function normalizeMinorBigInt(minor: number | bigint | null | undefined): bigint {
  if (typeof minor === "bigint") return minor;
  if (typeof minor !== "number" || !Number.isFinite(minor)) return BigInt(0);
  return BigInt(Math.trunc(minor));
}

export function formatFinanceMinor(
  minor: number | bigint | null | undefined,
  currency = "UZS",
  options?: { signDisplay?: "auto" | "always" },
): string {
  const safeMinor = normalizeMinorBigInt(minor);
  const absMinor = safeMinor < BigInt(0) ? -safeMinor : safeMinor;
  const minFrac: 0 | 2 = absMinor % BigInt(100) === BigInt(0) ? 0 : 2;
  return formatMinorUnits(safeMinor, {
    currency: (currency || "UZS").trim().toUpperCase(),
    minimumFractionDigits: minFrac,
    maximumFractionDigits: 2,
    signDisplay: options?.signDisplay ?? "auto",
  });
}

export interface ManualDraftLine {
  account_code: string;
  direction: GlDirection;
  amount_major: string;
  description?: string;
}

export interface ManualJournalBalanceSummary {
  debitMinor: bigint;
  creditMinor: bigint;
  validLines: PostingInput[];
  balanced: boolean;
  errorLineIndex: number | null;
}

/**
 * Validates & totals manual journal lines strictly in BigInt minor units (zero float math).
 */
export function evaluateManualJournalLines(
  lines: ManualDraftLine[],
  currency: string,
): ManualJournalBalanceSummary {
  const zero = BigInt(0);
  const maxSafe = BigInt(Number.MAX_SAFE_INTEGER);
  let debitMinor = zero;
  let creditMinor = zero;
  const validLines: PostingInput[] = [];
  let errorLineIndex: number | null = null;
  const ccy = (currency || "UZS").trim().toUpperCase();

  for (let i = 0; i < lines.length; i++) {
    const line = lines[i];
    const code = line.account_code.trim().toUpperCase();
    const parsedMinor = parseMajorToMinorBigInt(line.amount_major, { allowNegative: false });
    if (!code || parsedMinor === null || parsedMinor <= zero || parsedMinor > maxSafe) {
      if (errorLineIndex === null) errorLineIndex = i;
      continue;
    }
    if (line.direction === "DEBIT") {
      debitMinor += parsedMinor;
    } else {
      creditMinor += parsedMinor;
    }
    validLines.push({
      account_code: code,
      direction: line.direction,
      amount_minor: Number(parsedMinor),
      currency: ccy,
      description: line.description?.trim() || undefined,
    });
  }

  const balanced =
    errorLineIndex === null &&
    validLines.length >= 2 &&
    debitMinor > zero &&
    debitMinor === creditMinor;

  return {
    debitMinor,
    creditMinor,
    validLines,
    balanced,
    errorLineIndex,
  };
}

export function currentAccountingPeriod(now = new Date()): string {
  try {
    const parts = new Intl.DateTimeFormat("en-CA", {
      timeZone: "Asia/Tashkent",
      year: "numeric",
      month: "2-digit",
    }).formatToParts(now);
    const year = parts.find((p) => p.type === "year")?.value;
    const month = parts.find((p) => p.type === "month")?.value;
    if (year && month && /^\d{4}$/.test(year) && /^\d{2}$/.test(month)) {
      return `${year}-${month}`;
    }
  } catch {
    // Fallback to UTC YYYY-MM
  }
  return now.toISOString().slice(0, 7);
}

export function createSupplierFinanceApi(client?: FinanceRequestTransport) {
  const transport = client ?? createSupplierApi();

  return {
    getTrialBalance: (query: { entity?: string; period?: string } = {}): Promise<TrialBalanceReport> =>
      transport.request<TrialBalanceReport>(
        withQuery("/v1/finance/gl/trial-balance", query),
        "GET",
      ),

    getAccountLedger: (
      accountCode: string,
      query: { entity?: string; period?: string; limit?: number } = {},
    ): Promise<AccountLedgerResponse> =>
      transport.request<AccountLedgerResponse>(
        withQuery(`/v1/finance/gl/accounts/${encodeURIComponent(accountCode.trim().toUpperCase())}/ledger`, query),
        "GET",
      ),

    getJournalEntry: (
      entryId: string,
      query: { entity?: string } = {},
    ): Promise<JournalEntryRecord> =>
      transport.request<JournalEntryRecord>(
        withQuery(`/v1/finance/journal/${encodeURIComponent(entryId.trim())}`, query),
        "GET",
      ),

    postManualJournal: (
      body: ManualJournalRequest,
      idempotencyKey?: string,
    ): Promise<JournalEntryResult> => {
      const key =
        idempotencyKey?.trim() ||
        body.idempotency_key?.trim() ||
        (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
          ? `manual-${crypto.randomUUID()}`
          : `manual-${Date.now()}`);
      return transport.request<JournalEntryResult>("/v1/finance/journal/manual", "POST", {
        body: { ...body, idempotency_key: key },
        idempotencyKey: key,
      });
    },

    reverseJournalEntry: (
      entryId: string,
      body: ReverseJournalRequest = {},
      idempotencyKey?: string,
    ): Promise<JournalEntryResult> => {
      const cleanId = entryId.trim();
      const key =
        idempotencyKey?.trim() ||
        body.idempotency_key?.trim() ||
        (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
          ? `reverse-${cleanId}-${crypto.randomUUID()}`
          : `reverse-${cleanId}-${Date.now()}`);
      return transport.request<JournalEntryResult>(
        `/v1/finance/journal/${encodeURIComponent(cleanId)}/reverse`,
        "POST",
        {
          body: { ...body, idempotency_key: key },
          idempotencyKey: key,
        },
      );
    },

    listReconRuns: (
      query: { entity?: string; period_id?: string; status?: string; limit?: number } = {},
    ): Promise<ListReconRunsResponse> =>
      transport.request<ListReconRunsResponse>(
        withQuery("/v1/finance/recon/runs", query),
        "GET",
      ),

    triggerReconRun: (
      body: TriggerReconRunRequest = {},
      idempotencyKey?: string,
    ): Promise<TriggerReconRunResponse> => {
      const key =
        idempotencyKey?.trim() ||
        (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
          ? `recon-run-${crypto.randomUUID()}`
          : `recon-run-${Date.now()}`);
      return transport.request<TriggerReconRunResponse>("/v1/finance/recon/run", "POST", {
        body,
        idempotencyKey: key,
      });
    },

    listReconBreaks: (
      query: {
        entity?: string;
        run_id?: string;
        dimension?: string;
        status?: string;
        severity?: string;
        limit?: number;
      } = {},
    ): Promise<ListReconBreaksResponse> =>
      transport.request<ListReconBreaksResponse>(
        withQuery("/v1/finance/recon/breaks", query),
        "GET",
      ),

    updateReconBreak: (
      breakId: string,
      body: UpdateReconBreakRequest,
      query: { entity?: string } = {},
      idempotencyKey?: string,
    ): Promise<UpdateReconBreakResponse> => {
      const cleanId = breakId.trim();
      const key =
        idempotencyKey?.trim() ||
        (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function"
          ? `recon-break-${cleanId}-${crypto.randomUUID()}`
          : `recon-break-${cleanId}-${Date.now()}`);
      return transport.request<UpdateReconBreakResponse>(
        withQuery(`/v1/finance/recon/breaks/${encodeURIComponent(cleanId)}`, query),
        "PATCH",
        {
          body,
          idempotencyKey: key,
        },
      );
    },
  };
}

export type SupplierFinanceApi = ReturnType<typeof createSupplierFinanceApi>;

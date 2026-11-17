import type { ApiClient } from "@pegasusx/api-core";
import { ApiError } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import { formatFinanceMinor, normalizeMinorBigInt, type FinanceRequestTransport } from "@/lib/finance-api";
import { formatBpsPercent, parseMajorToMinorBigInt } from "@/lib/money";

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

function makeIdempotencyKey(prefix: string, explicit?: string): string {
  const clean = explicit?.trim();
  if (clean) return clean;
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    return `${prefix}-${crypto.randomUUID()}`;
  }
  return `${prefix}-${Date.now()}`;
}

export { formatFinanceMinor, normalizeMinorBigInt };

export function parseMajorToSafeMinor(major: string, allowNegative = false): number | null {
  const parsed = parseMajorToMinorBigInt(major, { allowNegative });
  if (parsed === null) return null;
  if (parsed > BigInt(Number.MAX_SAFE_INTEGER) || parsed < BigInt(Number.MIN_SAFE_INTEGER)) {
    return null;
  }
  return Number(parsed);
}

export function formatRateBps(bps: number | bigint | null | undefined): string {
  const safe = normalizeMinorBigInt(bps);
  const abs = safe < BigInt(0) ? -safe : safe;
  const fracDigits: 0 | 2 = abs % BigInt(100) === BigInt(0) ? 0 : 2;
  return `${formatBpsPercent(safe, fracDigits)}%`;
}

export interface ControlErrorDetail {
  message: string;
  code?: string;
  status?: number;
  payload?: Record<string, unknown>;
}

export function extractControlError(err: unknown, fallback: string): ControlErrorDetail {
  if (err instanceof ApiError) {
    const payload =
      err.payload && typeof err.payload === "object"
        ? (err.payload as Record<string, unknown>)
        : undefined;
    const code =
      typeof payload?.code === "string" && payload.code.trim()
        ? payload.code.trim()
        : undefined;
    const parts = [
      code,
      typeof payload?.error === "string" ? payload.error.trim() : "",
      typeof payload?.message === "string" ? payload.message.trim() : "",
      typeof payload?.detail === "string" ? payload.detail.trim() : "",
    ].filter((v): v is string => typeof v === "string" && v.length > 0);
    const unique = Array.from(new Set(parts));
    return {
      message: unique.length > 0 ? unique.join(" · ") : err.message || fallback,
      code,
      status: err.status,
      payload,
    };
  }
  return {
    message: err instanceof Error && err.message.trim() ? err.message : fallback,
  };
}

// ---------------------------------------------------------------------------
// M1.1 Period Close Types
// ---------------------------------------------------------------------------

export type PeriodStatus = "OPEN" | "SOFT_CLOSED" | "CLOSED" | "REOPENED" | string;

export interface AccountingPeriod {
  entity_id: string;
  period_id: string;
  status: PeriodStatus;
  timezone: string;
  period_start_utc?: string;
  period_end_utc?: string;
  cutoff_local?: string;
  soft_closed_by?: string;
  soft_closed_at?: string;
  closed_by?: string;
  closed_at?: string;
  reopened_by?: string;
  reopened_at?: string;
  reopen_reason?: string;
  snapshot_uri?: string;
  snapshot_hash?: string;
  checklist_json?: string;
  created_at?: string;
  updated_at?: string;
}

export interface PeriodsListResponse {
  entity_id: string;
  timezone: string;
  periods: AccountingPeriod[];
}

export interface PeriodChecklistItem {
  key: string;
  label: string;
  passed: boolean;
  blocking: boolean;
  open_count: number;
  detail?: string;
}

export interface PeriodChecklist {
  entity_id: string;
  period_id: string;
  timezone: string;
  period_start_utc?: string;
  period_end_utc?: string;
  ready_to_close: boolean;
  blocking_count: number;
  evaluated_at?: string;
  items: PeriodChecklistItem[];
}

export interface PeriodSnapshotReport {
  entity_id?: string;
  period_id?: string;
  snapshot_hash?: string;
  snapshot_uri?: string;
  generated_at?: string;
  trial_balance?: Record<string, unknown>;
  ar_summary?: Record<string, unknown>;
  valuation_summary?: Record<string, unknown>;
  vat_summary?: Record<string, unknown>;
  [key: string]: unknown;
}

export interface PeriodMutationResponse {
  period: AccountingPeriod;
  checklist?: PeriodChecklist;
  snapshot?: PeriodSnapshotReport;
}

export interface PeriodReopenResponse {
  period?: AccountingPeriod;
  status?: string;
  code?: string;
  approval_request_id?: string;
  required_approvers?: number;
  payload_hash?: string;
}

// ---------------------------------------------------------------------------
// M1.2 Approvals, Delegations & CEL Policy Types
// ---------------------------------------------------------------------------

export interface ApprovalPolicyStep {
  step_index: number;
  name?: string;
  allowed_roles: string[];
  required_approvers: number;
  require_mfa?: boolean;
  mfa_threshold_minor?: number;
  timeout_hours?: number;
}

export interface ApprovalPolicy {
  tenant_id: string;
  policy_id: string;
  name: string;
  object_type: string;
  action: string;
  condition_expr: string;
  steps?: ApprovalPolicyStep[];
  steps_json?: string;
  version: number;
  enabled: boolean;
  created_by?: string;
  created_at?: string;
  updated_at?: string;
}

export interface CreateApprovalPolicyRequest {
  tenant_id?: string;
  name: string;
  object_type: string;
  action: string;
  condition_expr: string;
  steps: ApprovalPolicyStep[];
  enabled?: boolean;
}

export interface UpdateApprovalPolicyRequest {
  tenant_id?: string;
  name?: string;
  object_type?: string;
  action?: string;
  condition_expr?: string;
  steps?: ApprovalPolicyStep[];
  enabled?: boolean;
}

export interface ApprovalPoliciesResponse {
  policies: ApprovalPolicy[];
  count: number;
}

export type ApprovalRequestStatus = "PENDING" | "APPROVED" | "REJECTED" | "EXPIRED" | "CANCELLED" | string;

export interface ApprovalRequest {
  tenant_id: string;
  request_id: string;
  policy_id: string;
  policy_version: number;
  object_type: string;
  object_id: string;
  action: string;
  amount_minor: number;
  currency?: string;
  requested_by: string;
  actor_type: string;
  payload_json?: string;
  payload_hash: string;
  status: ApprovalRequestStatus;
  current_step: number;
  required_approvers: number;
  approvals_received?: number;
  break_glass?: boolean;
  break_glass_reason?: string;
  expires_at?: string;
  idempotency_key?: string;
  resolved_at?: string;
  created_at: string;
  updated_at?: string;
  require_mfa?: boolean;
}

export interface ApprovalDecision {
  tenant_id?: string;
  request_id: string;
  decision_id: string;
  step_index: number;
  approver_id: string;
  delegated_from?: string;
  decision: "APPROVE" | "REJECT" | string;
  comment?: string;
  mfa_verified: boolean;
  created_at: string;
}

export interface ApprovalInboxResponse {
  requests: ApprovalRequest[];
  count: number;
}

export interface ApprovalDetailResponse {
  request: ApprovalRequest;
  decisions: ApprovalDecision[];
  policy?: ApprovalPolicy;
}

export interface DecideApprovalRequest {
  tenant_id?: string;
  comment?: string;
  payload_hash?: string;
  mfa_verified?: boolean;
  totp_code?: string;
  break_glass?: boolean;
  break_glass_reason?: string;
}

export interface DecideApprovalResponse {
  request: ApprovalRequest;
  decision: ApprovalDecision;
}

export interface ApprovalDelegation {
  tenant_id?: string;
  delegation_id: string;
  delegator_id: string;
  delegate_id: string;
  role_scope?: string;
  starts_at?: string;
  ends_at: string;
  reason: string;
  enabled: boolean;
  created_at?: string;
}

export interface ApprovalDelegationsResponse {
  delegations: ApprovalDelegation[];
  count: number;
}

export interface CreateDelegationRequest {
  tenant_id?: string;
  delegator_id?: string;
  delegate_id: string;
  role_scope?: string;
  starts_at?: string;
  ends_at: string;
  reason: string;
}

// ---------------------------------------------------------------------------
// M1.4 ESF & Fiscal Console Types
// ---------------------------------------------------------------------------

export type EsfDirection = "OUTBOUND" | "INBOUND" | string;
export type EsfStatus = "DRAFT" | "SUBMITTED" | "ACCEPTED" | "REJECTED" | "CANCELLED" | "FAILED" | string;

export interface EsfDocument {
  entity_id: string;
  esf_id: string;
  direction: EsfDirection;
  order_id?: string;
  po_id?: string;
  counterparty_tin: string;
  counterparty_name: string;
  total_minor: number;
  vat_minor: number;
  vat_rate_bps: number;
  currency?: string;
  status: EsfStatus;
  operator: string;
  operator_ref?: string;
  signed_payload_uri?: string;
  payload_hash?: string;
  failure_reason?: string;
  period_id?: string;
  issued_at?: string;
  accepted_at?: string;
  created_at: string;
  updated_at?: string;
}

export interface EsfStatusHistoryItem {
  history_id: string;
  from_status?: string;
  to_status: string;
  actor_id: string;
  reason?: string;
  operator_payload_json?: string;
  created_at: string;
}

export interface EsfListResponse {
  entity_id: string;
  documents: EsfDocument[];
  count: number;
}

export interface EsfDetailResponse {
  document: EsfDocument;
  history: EsfStatusHistoryItem[];
  vat_reversed_minor?: number;
  gl_reversal_entry_id?: string;
  credit_note_id?: string;
}

export interface CreateOutboundEsfRequest {
  entity_id?: string;
  order_id: string;
  operator?: string;
  counterparty_tin?: string;
  counterparty_name?: string;
  submit?: boolean;
}

export interface CreateInboundEsfRequest {
  entity_id?: string;
  po_id?: string;
  order_id?: string;
  counterparty_tin: string;
  counterparty_name: string;
  total_minor: number;
  vat_minor: number;
  vat_rate_bps: number;
  operator?: string;
  operator_ref: string;
  period_id?: string;
  status?: string;
}

export interface EsfStatusTransitionRequest {
  entity_id?: string;
  action: "ACCEPT" | "REJECT" | "CANCEL";
  reason?: string;
  operator_ref?: string;
  operator_payload?: string;
}

export interface VatRateBreakdown {
  vat_rate_bps: number;
  outbound_base_minor: number;
  outbound_vat_minor: number;
  inbound_base_minor: number;
  inbound_vat_minor: number;
  doc_count: number;
}

export interface VatSummaryReport {
  entity_id: string;
  period_id: string;
  currency: string;
  esf_output_vat_minor: number;
  esf_input_vat_minor: number;
  esf_net_vat_minor: number;
  gl_output_vat_minor: number;
  gl_input_vat_minor: number;
  gl_net_vat_minor: number;
  output_diff_minor: number;
  input_diff_minor: number;
  diff_minor: number;
  reconciled: boolean;
  outbound_count: number;
  inbound_count: number;
  rejected_count: number;
  by_rate?: VatRateBreakdown[];
}

export interface MissingIkpuItem {
  product_id: string;
  name: string;
  category_id?: string;
  barcode?: string;
  ikpu_code?: string;
  package_code?: string;
  marking_code?: string;
  missing_fields: string[];
}

export interface MissingIkpuResponse {
  supplier_id: string;
  items: MissingIkpuItem[];
  count: number;
}

export function evaluateVatTieOut(summary: VatSummaryReport | null): {
  diffMinor: bigint;
  outputDiffMinor: bigint;
  inputDiffMinor: bigint;
  isZeroDiff: boolean;
} {
  if (!summary) {
    return {
      diffMinor: BigInt(0),
      outputDiffMinor: BigInt(0),
      inputDiffMinor: BigInt(0),
      isZeroDiff: false,
    };
  }
  const diffMinor = normalizeMinorBigInt(summary.diff_minor);
  const outputDiffMinor = normalizeMinorBigInt(summary.output_diff_minor);
  const inputDiffMinor = normalizeMinorBigInt(summary.input_diff_minor);
  return {
    diffMinor,
    outputDiffMinor,
    inputDiffMinor,
    isZeroDiff: summary.reconciled === true && diffMinor === BigInt(0),
  };
}

export function createControlAccountingApi(client?: FinanceRequestTransport) {
  const transport = client ?? createSupplierApi();

  return {
    // M1.1 Periods
    listPeriods: (query: { entity?: string; status?: string } = {}): Promise<PeriodsListResponse> =>
      transport.request<PeriodsListResponse>(withQuery("/v1/finance/periods", query), "GET"),

    createPeriod: (
      body: { entity_id?: string; period_id: string; timezone?: string },
      idempotencyKey?: string,
    ): Promise<{ period: AccountingPeriod }> =>
      transport.request<{ period: AccountingPeriod }>("/v1/finance/periods", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("period-create", idempotencyKey),
      }),

    softClosePeriod: (
      periodId: string,
      body: { entity_id?: string } = {},
      idempotencyKey?: string,
    ): Promise<PeriodMutationResponse> => {
      const cleanId = encodeURIComponent(periodId.trim());
      return transport.request<PeriodMutationResponse>(
        `/v1/finance/periods/${cleanId}/soft-close`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`period-soft-close-${periodId.trim()}`, idempotencyKey),
        },
      );
    },

    closePeriod: (
      periodId: string,
      body: { entity_id?: string } = {},
      idempotencyKey?: string,
    ): Promise<PeriodMutationResponse> => {
      const cleanId = encodeURIComponent(periodId.trim());
      return transport.request<PeriodMutationResponse>(
        `/v1/finance/periods/${cleanId}/close`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`period-close-${periodId.trim()}`, idempotencyKey),
        },
      );
    },

    reopenPeriod: (
      periodId: string,
      body: { entity_id?: string; reason: string; approval_id?: string },
      idempotencyKey?: string,
    ): Promise<PeriodReopenResponse> => {
      const cleanId = encodeURIComponent(periodId.trim());
      return transport.request<PeriodReopenResponse>(
        `/v1/finance/periods/${cleanId}/reopen`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`period-reopen-${periodId.trim()}`, idempotencyKey),
        },
      );
    },

    evaluatePeriodChecklist: (
      periodId: string,
      query: { entity?: string } = {},
    ): Promise<PeriodChecklist> => {
      const cleanId = encodeURIComponent(periodId.trim());
      return transport.request<PeriodChecklist>(
        withQuery(`/v1/finance/periods/${cleanId}/checklist`, query),
        "GET",
      );
    },

    getPeriodSnapshot: (
      periodId: string,
      query: { entity?: string } = {},
    ): Promise<PeriodSnapshotReport> => {
      const cleanId = encodeURIComponent(periodId.trim());
      return transport.request<PeriodSnapshotReport>(
        withQuery(`/v1/finance/periods/${cleanId}/snapshot`, {
          ...query,
          format: "json",
        }),
        "GET",
      );
    },

    downloadPeriodSnapshotCsv: (
      periodId: string,
      query: { entity?: string } = {},
    ): Promise<string> => {
      const cleanId = encodeURIComponent(periodId.trim());
      return transport.request<string>(
        withQuery(`/v1/finance/periods/${cleanId}/snapshot`, {
          ...query,
          format: "csv",
        }),
        "GET",
      );
    },

    // M1.2 Approvals & CEL Policies & Delegations
    listApprovalPolicies: (
      query: { tenant_id?: string; object_type?: string; action?: string } = {},
    ): Promise<ApprovalPoliciesResponse> =>
      transport.request<ApprovalPoliciesResponse>(withQuery("/v1/approvals/policies", query), "GET"),

    createApprovalPolicy: (
      body: CreateApprovalPolicyRequest,
      idempotencyKey?: string,
    ): Promise<{ policy: ApprovalPolicy }> =>
      transport.request<{ policy: ApprovalPolicy }>("/v1/approvals/policies", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("approval-policy-create", idempotencyKey),
      }),

    updateApprovalPolicy: (
      policyId: string,
      body: UpdateApprovalPolicyRequest,
      idempotencyKey?: string,
    ): Promise<{ policy: ApprovalPolicy }> => {
      const cleanId = encodeURIComponent(policyId.trim());
      return transport.request<{ policy: ApprovalPolicy }>(
        `/v1/approvals/policies/${cleanId}`,
        "PUT",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`approval-policy-put-${policyId.trim()}`, idempotencyKey),
        },
      );
    },

    toggleApprovalPolicy: (
      policyId: string,
      body: { tenant_id?: string; enabled: boolean },
      idempotencyKey?: string,
    ): Promise<{ policy: ApprovalPolicy }> => {
      const cleanId = encodeURIComponent(policyId.trim());
      return transport.request<{ policy: ApprovalPolicy }>(
        `/v1/approvals/policies/${cleanId}`,
        "PATCH",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`approval-policy-toggle-${policyId.trim()}`, idempotencyKey),
        },
      );
    },

    listApprovalInbox: (
      query: { tenant_id?: string; status?: string; object_type?: string; limit?: number } = {},
    ): Promise<ApprovalInboxResponse> =>
      transport.request<ApprovalInboxResponse>(withQuery("/v1/approvals/inbox", query), "GET"),

    getApprovalRequest: (
      requestId: string,
      query: { tenant_id?: string } = {},
    ): Promise<ApprovalDetailResponse> => {
      const cleanId = encodeURIComponent(requestId.trim());
      return transport.request<ApprovalDetailResponse>(
        withQuery(`/v1/approvals/${cleanId}`, query),
        "GET",
      );
    },

    approveRequest: (
      requestId: string,
      body: DecideApprovalRequest = {},
      idempotencyKey?: string,
    ): Promise<DecideApprovalResponse> => {
      const cleanId = encodeURIComponent(requestId.trim());
      return transport.request<DecideApprovalResponse>(
        `/v1/approvals/${cleanId}/approve`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`approval-approve-${requestId.trim()}`, idempotencyKey),
        },
      );
    },

    rejectRequest: (
      requestId: string,
      body: DecideApprovalRequest = {},
      idempotencyKey?: string,
    ): Promise<DecideApprovalResponse> => {
      const cleanId = encodeURIComponent(requestId.trim());
      return transport.request<DecideApprovalResponse>(
        `/v1/approvals/${cleanId}/reject`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`approval-reject-${requestId.trim()}`, idempotencyKey),
        },
      );
    },

    listDelegations: (
      query: { tenant_id?: string; actor_id?: string } = {},
    ): Promise<ApprovalDelegationsResponse> =>
      transport.request<ApprovalDelegationsResponse>(withQuery("/v1/approvals/delegations", query), "GET"),

    createDelegation: (
      body: CreateDelegationRequest,
      idempotencyKey?: string,
    ): Promise<{ delegation: ApprovalDelegation }> =>
      transport.request<{ delegation: ApprovalDelegation }>("/v1/approvals/delegations", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("approval-delegation-create", idempotencyKey),
      }),

    revokeDelegation: (
      delegationId: string,
      body: { tenant_id?: string } = {},
      idempotencyKey?: string,
    ): Promise<{ delegation: ApprovalDelegation }> => {
      const cleanId = encodeURIComponent(delegationId.trim());
      return transport.request<{ delegation: ApprovalDelegation }>(
        `/v1/approvals/delegations/${cleanId}/revoke`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`approval-delegation-revoke-${delegationId.trim()}`, idempotencyKey),
        },
      );
    },

    // M1.4 ESF & Fiscal Console
    listEsfDocuments: (
      query: {
        entity_id?: string;
        direction?: string;
        status?: string;
        period_id?: string;
        order_id?: string;
        limit?: number;
      } = {},
    ): Promise<EsfListResponse> =>
      transport.request<EsfListResponse>(withQuery("/v1/fiscal/esf", query), "GET"),

    getEsfDocument: (
      esfId: string,
      query: { entity_id?: string } = {},
    ): Promise<EsfDetailResponse> => {
      const cleanId = encodeURIComponent(esfId.trim());
      return transport.request<EsfDetailResponse>(
        withQuery(`/v1/fiscal/esf/${cleanId}`, query),
        "GET",
      );
    },

    createOutboundEsf: (
      body: CreateOutboundEsfRequest,
      idempotencyKey?: string,
    ): Promise<EsfDocument> =>
      transport.request<EsfDocument>("/v1/fiscal/esf", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("esf-create", idempotencyKey),
      }),

    resendEsfDocument: (
      esfId: string,
      body: { entity_id?: string; operator?: string } = {},
      idempotencyKey?: string,
    ): Promise<EsfDocument> => {
      const cleanId = encodeURIComponent(esfId.trim());
      return transport.request<EsfDocument>(
        `/v1/fiscal/esf/${cleanId}/resend`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`esf-resend-${esfId.trim()}`, idempotencyKey),
        },
      );
    },

    recordInboundEsf: (
      body: CreateInboundEsfRequest,
      idempotencyKey?: string,
    ): Promise<EsfDocument> =>
      transport.request<EsfDocument>("/v1/fiscal/esf/inbound", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("esf-inbound", idempotencyKey),
      }),

    transitionEsfStatus: (
      esfId: string,
      body: EsfStatusTransitionRequest,
      idempotencyKey?: string,
    ): Promise<EsfDetailResponse> => {
      const cleanId = encodeURIComponent(esfId.trim());
      return transport.request<EsfDetailResponse>(
        `/v1/fiscal/esf/${cleanId}/status`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`esf-status-${esfId.trim()}`, idempotencyKey),
        },
      );
    },

    getVatSummary: (
      query: { entity_id?: string; period_id?: string } = {},
    ): Promise<VatSummaryReport> =>
      transport.request<VatSummaryReport>(withQuery("/v1/fiscal/vat-summary", query), "GET"),

    listMissingIkpu: (
      query: { entity_id?: string; supplier_id?: string } = {},
    ): Promise<MissingIkpuResponse> =>
      transport.request<MissingIkpuResponse>(withQuery("/v1/fiscal/missing-ikpu", query), "GET"),
  };
}

export type ControlAccountingApi = ReturnType<typeof createControlAccountingApi>;

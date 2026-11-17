import { ApiError } from "@pegasusx/api-core";
import {
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ControlErrorDetail,
} from "@/lib/control-accounting-api";
import { type FinanceRequestTransport } from "@/lib/finance-api";

function withQuery(
  path: string,
  query: Record<string, string | number | boolean | undefined>,
): string {
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

export {
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ControlErrorDetail,
};

const BIGINT_ZERO = BigInt(0);
const BIGINT_FIFTY = BigInt(50);
const BIGINT_HUNDRED = BigInt(100);
const BIGINT_THOUSAND = BigInt(1000);
const BIGINT_TEN_THOUSAND = BigInt(10000);

function groupIntegerDigits(digits: string): string {
  if (digits.length <= 3) return digits;
  const parts: string[] = [];
  for (let i = digits.length; i > 0; i -= 3) {
    parts.push(digits.slice(Math.max(0, i - 3), i));
  }
  parts.reverse();
  return parts.join(",");
}

/**
 * Formats integer thousandths (`milli`, where 1000 = 1 unit) using pure BigInt math.
 */
export function formatMilliUnits(
  milli: number | bigint | null | undefined,
  unitLabel?: string,
): string {
  const safe = normalizeMinorBigInt(milli);
  const negative = safe < BIGINT_ZERO;
  const abs = negative ? -safe : safe;
  const whole = abs / BIGINT_THOUSAND;
  const frac = abs % BIGINT_THOUSAND;
  const wholeFormatted = groupIntegerDigits(whole.toString());
  const fracTrimmed = frac === BIGINT_ZERO ? "" : frac.toString().padStart(3, "0").replace(/0+$/, "");
  const sign = negative ? "-" : "";
  const numeric = fracTrimmed ? `${sign}${wholeFormatted}.${fracTrimmed}` : `${sign}${wholeFormatted}`;
  return unitLabel ? `${numeric} ${unitLabel}` : numeric;
}

/**
 * Formats ROI multiple in basis points (`10000 bps = 1.00x`) using pure BigInt arithmetic.
 */
export function formatRoiMultipleBps(bps: number | bigint | null | undefined): string {
  const safe = normalizeMinorBigInt(bps);
  const negative = safe < BIGINT_ZERO;
  const abs = negative ? -safe : safe;
  const scaledHundredths = (abs + BIGINT_FIFTY) / BIGINT_HUNDRED;
  const whole = scaledHundredths / BIGINT_HUNDRED;
  const frac = (scaledHundredths % BIGINT_HUNDRED).toString().padStart(2, "0");
  const sign = negative ? "-" : "";
  return `${sign}${groupIntegerDigits(whole.toString())}.${frac}x`;
}

/**
 * Formats signed basis points as a percentage string (e.g. `+12.50%`, `-5%`).
 */
export function formatSignedBps(bps: number | bigint | null | undefined): string {
  const safe = normalizeMinorBigInt(bps);
  if (safe > BIGINT_ZERO) return `+${formatRateBps(safe)}`;
  return formatRateBps(safe);
}

/**
 * Normalizes `milli` quantities into numeric unit values solely for chart coordinate plotting.
 */
export function milliToChartUnits(milli: number | bigint | null | undefined): number {
  const safe = normalizeMinorBigInt(milli);
  const whole = Number(safe / BIGINT_THOUSAND);
  const rem = Number(safe % BIGINT_THOUSAND);
  return whole + rem / 1000;
}

export function isApprovalPendingResponse(
  res: UpdateCreditLimitResponse,
): boolean {
  return (
    res.status === "APPROVAL_REQUIRED" ||
    res.status === "PENDING_APPROVAL" ||
    res.approval_required === true ||
    res.code === "approval_required" ||
    Boolean(res.approval_request_id && res.status !== "APPLIED")
  );
}

// ---------------------------------------------------------------------------
// M1.5 Forecast v2 & Replenishment Types
// ---------------------------------------------------------------------------

export interface ForecastRegressor {
  name: string;
  category: string;
  multiplier_bps: number;
  description: string;
}

export interface ForecastEvidenceRef {
  ref_type: string;
  ref_id: string;
  label: string;
  value_text: string;
}

export interface ForecastRun {
  run_id: string;
  entity_id: string;
  supplier_id?: string;
  location_id?: string;
  horizon_days: number;
  status: string;
  champion_model: string;
  challenger_model: string;
  winner_model: string;
  champion_wape_bps: number;
  challenger_wape_bps: number;
  champion_bias_bps?: number;
  challenger_bias_bps?: number;
  wape_improvement_bps: number;
  sku_count?: number;
  location_count?: number;
  feature_source?: string;
  regressors?: ForecastRegressor[];
  completed_at?: string;
  created_at?: string;
}

export interface ForecastValue {
  run_id: string;
  forecast_date: string;
  location_id: string;
  location_type?: string;
  sku: string;
  product_id?: string;
  model: string;
  is_winner: boolean;
  demand_class: string;
  p10_milli: number;
  p50_milli: number;
  p90_milli: number;
  baseline_p50_milli: number;
  wape_bps: number;
  bias_bps: number;
  confidence_bps: number;
  regressors?: ForecastRegressor[];
}

export interface ForecastExplanation {
  explanation_id: string;
  run_id: string;
  location_id: string;
  sku: string;
  window_start?: string;
  window_end?: string;
  source: string;
  model_name: string;
  summary_text: string;
  evidence_refs?: ForecastEvidenceRef[];
  adjustment_bps: number;
  adjustment_reason?: string;
  status: string;
  baseline_p50_milli: number;
  adjusted_p50_milli: number;
  actual_qty_milli?: number;
  forecast_value_add_bps: number;
  proposed_by?: string;
  approved_by?: string;
  approval_note?: string;
  approval_request_id?: string;
  created_at?: string;
  approved_at?: string;
}

export interface PlanningForecastResponse {
  run?: ForecastRun | null;
  forecasts?: ForecastValue[];
  explanations?: ForecastExplanation[];
}

export interface RunForecastRequest {
  location_id?: string;
  sku?: string;
  horizon_days?: number;
  promo_discount_bps?: number;
  active_promo?: boolean;
  explain?: boolean;
}

export interface RunForecastResponse {
  run: ForecastRun;
  forecasts: ForecastValue[];
  explanations: ForecastExplanation[];
}

export interface ApproveForecastExplanationRequest {
  decision?: "APPROVE" | "REJECT";
  adjustment_bps?: number;
  comment?: string;
}

export interface ApproveForecastExplanationResponse {
  explanation: ForecastExplanation;
}

export interface ApproveReplenishmentSuggestionRequest {
  retailer_id?: string;
  sku?: string;
  qty?: number;
  mode?: "order" | "approve_only";
}

export interface ApproveReplenishmentSuggestionResponse {
  suggestion_id: string;
  retailer_id: string;
  sku: string;
  status: string;
  approved_qty: number;
  order_id?: string;
  approved_by?: string;
}

// ---------------------------------------------------------------------------
// M1.6 Credit v2 Portfolio, Scoring & Limit Adjustment Types
// ---------------------------------------------------------------------------

export interface CreditReasonCode {
  code: string;
  label: string;
  impact_bps: number;
  detail: string;
  source: string;
}

export interface CreditScoreRecord {
  score_id: string;
  retailer_id?: string;
  scope: "RELATIONSHIP" | "NETWORK" | string;
  supplier_id?: string;
  model_version: string;
  scorer_source: string;
  score: number;
  pd_bps: number;
  risk_tier: string;
  suggested_limit_minor: number;
  current_limit_minor: number;
  consent_verified: boolean;
  consent_scope?: string;
  disputes_isolated_count: number;
  reason_codes?: CreditReasonCode[];
  features_hash?: string;
  window_start?: string;
  window_end?: string;
  computed_at?: string;
}

export interface RetailerCreditScoreResponse {
  retailer_id: string;
  relationship_score?: CreditScoreRecord | null;
  network_score?: CreditScoreRecord | null;
  history?: CreditScoreRecord[];
}

export interface CreditPortfolioSummary {
  total_retailers: number;
  total_exposure_minor: number;
  total_limit_minor: number;
  utilization_bps: number;
  frozen_count: number;
  high_risk_count: number;
  avg_score: number;
  consent_verified_count: number;
}

export interface CreditPortfolioItem {
  retailer_id: string;
  retailer_name?: string;
  credit_limit_minor: number;
  current_balance_minor: number;
  reserved_minor: number;
  available_credit_minor: number;
  utilization_bps: number;
  risk_score: number;
  pd_bps: number;
  risk_tier: string;
  status: string;
  prepay_only_locked: boolean;
  consent_verified: boolean;
  suggested_limit_minor: number;
  reason_codes?: CreditReasonCode[];
  last_evaluated_at?: string;
}

export interface CreditPortfolioResponse {
  supplier_id: string;
  summary: CreditPortfolioSummary;
  items: CreditPortfolioItem[];
}

export interface UpdateCreditLimitRequest {
  new_limit_minor?: number;
  credit_limit_minor?: number;
  reason: string;
  supplier_id?: string;
  approval_id?: string;
  consent_verified?: boolean;
  idempotency_key?: string;
}

export interface UpdateCreditLimitResponse {
  status: "APPLIED" | "APPROVAL_REQUIRED" | "PENDING_APPROVAL" | string;
  approval_required?: boolean;
  retailer_id?: string;
  supplier_id?: string;
  old_limit_minor?: number;
  new_limit_minor?: number;
  previous_limit_minor?: number;
  credit_limit_minor?: number;
  approval_request_id?: string;
  code?: string;
  policy_id?: string;
  policy_version?: number;
  payload_hash?: string;
  reason_codes?: CreditReasonCode[];
  score_record?: CreditScoreRecord;
}

// ---------------------------------------------------------------------------
// M1.7 Proposals Inbox & L1 Agents Types
// ---------------------------------------------------------------------------

export type L1AgentId =
  | "REORDER_AGENT"
  | "COLLECTIONS_AGENT"
  | "EXPIRY_REBALANCE_AGENT"
  | "ALL";

export interface ProposalEvidenceRef {
  type: string;
  id: string;
  label?: string;
}

export interface ProposalRecord {
  tenant_id?: string;
  proposal_id: string;
  agent_id: string;
  proposal_type: string;
  target_object_type: string;
  target_object_id: string;
  title: string;
  summary: string;
  rationale: string;
  payload_json?: string;
  payload?: Record<string, unknown>;
  payload_hash: string;
  evidence_refs?: ProposalEvidenceRef[];
  confidence_bps: number;
  estimated_impact_minor: number;
  currency: string;
  roi_category: string;
  status: string;
  approval_request_id?: string;
  created_by?: string;
  actor_type?: string;
  decided_by?: string;
  decision_comment?: string;
  executed_by?: string;
  execution_result?: Record<string, unknown>;
  execution_error?: string;
  roi_event_id?: string;
  expires_at?: string;
  decided_at?: string;
  executed_at?: string;
  created_at?: string;
  updated_at?: string;
}

export interface ProposalsListResponse {
  tenant_id?: string;
  proposals: ProposalRecord[];
  count: number;
}

export interface ProposalDetailResponse {
  proposal: ProposalRecord;
}

export interface RunProposalAgentResponse {
  agent_id: string;
  proposals: ProposalRecord[];
  created_count: number;
}

export interface CreateProposalRequest {
  agent_id?: string;
  proposal_type: string;
  target_object_type: string;
  target_object_id: string;
  title: string;
  summary?: string;
  rationale?: string;
  payload: Record<string, unknown>;
  evidence_refs?: ProposalEvidenceRef[];
  confidence_bps?: number;
  estimated_impact_minor?: number;
  currency?: string;
  roi_category?: string;
  idempotency_key?: string;
}

export interface ApproveProposalRequest {
  comment?: string;
  payload_hash?: string;
  edited_payload?: Record<string, unknown>;
  totp_code?: string;
}

export interface RejectProposalRequest {
  comment?: string;
}

export interface ExecuteProposalRequest {
  payload_hash?: string;
}

export interface ExecuteProposalResponse {
  proposal: ProposalRecord;
  execution_result?: Record<string, unknown>;
  roi_event?: RoiEvent;
}

// ---------------------------------------------------------------------------
// M1.8 ROI v1 Types
// ---------------------------------------------------------------------------

export type RoiCategory =
  | "BAD_DEBT_AVOIDED"
  | "STOCKOUT_AVOIDED"
  | "EXPIRY_SAVED"
  | "LOGISTICS_SAVED"
  | "LABOUR_SAVED"
  | "WORKING_CAPITAL_SAVED"
  | string;

export interface RoiBaseline {
  tenant_id?: string;
  baseline_id: string;
  window_days: number;
  baseline_ar_minor: number;
  baseline_bad_debt_minor: number;
  baseline_expired_minor: number;
  baseline_stockout_minor: number;
  baseline_logistics_minor: number;
  baseline_labour_hours: number;
  survey_hourly_wage_minor: number;
  survey_notes?: string;
  survey?: Record<string, unknown>;
  currency: string;
  captured_by?: string;
  period_start?: string;
  period_end?: string;
  created_at?: string;
}

export interface RoiBaselineResponse {
  tenant_id?: string;
  baseline: RoiBaseline | null;
}

export interface CaptureRoiBaselineRequest {
  window_days?: number;
  baseline_bad_debt_minor?: number;
  baseline_expired_minor?: number;
  baseline_stockout_minor?: number;
  baseline_logistics_minor?: number;
  baseline_labour_hours?: number;
  survey_hourly_wage_minor?: number;
  survey_notes?: string;
  survey?: Record<string, unknown>;
  currency?: string;
}

export interface RoiCategorySummary {
  category: RoiCategory;
  event_count: number;
  total_savings_minor: number;
  realized_minor: number;
  estimated_minor: number;
  holdout_savings_minor: number;
  avg_confidence_bps: number;
  methodology: string;
}

export interface RoiHoldoutStats {
  treatment_event_count: number;
  treatment_savings_minor: number;
  holdout_event_count: number;
  holdout_savings_minor: number;
  incremental_lift_minor: number;
  lift_bps: number;
}

export interface RoiSummaryResponse {
  tenant_id?: string;
  currency: string;
  window_days: number;
  total_savings_minor: number;
  realized_savings_minor: number;
  estimated_savings_minor: number;
  holdout_savings_minor: number;
  platform_fees_minor: number;
  net_savings_minor: number;
  roi_multiple_bps: number;
  categories: RoiCategorySummary[];
  holdout_stats?: RoiHoldoutStats | null;
  baseline?: RoiBaseline | null;
}

export interface RoiEvent {
  tenant_id?: string;
  event_id: string;
  category: RoiCategory;
  savings_minor: number;
  currency: string;
  methodology: string;
  confidence_bps: number;
  is_estimate: boolean;
  holdout_control: boolean;
  proposal_id?: string;
  evidence_ref_type?: string;
  evidence_ref_id?: string;
  notes?: string;
  recorded_by?: string;
  occurred_at?: string;
  created_at?: string;
}

export interface RoiEventsListResponse {
  tenant_id?: string;
  events: RoiEvent[];
  count: number;
}

export interface RecordRoiEventRequest {
  category: RoiCategory;
  savings_minor: number;
  currency?: string;
  methodology: string;
  confidence_bps?: number;
  is_estimate?: boolean;
  holdout_control?: boolean;
  proposal_id?: string;
  evidence_ref_type?: string;
  evidence_ref_id?: string;
  notes?: string;
  idempotency_key?: string;
}

export interface RoiReportResponse {
  tenant_id?: string;
  currency?: string;
  window_days?: number;
  generated_at?: string;
  summary?: RoiSummaryResponse;
  categories?: RoiCategorySummary[];
  holdout_stats?: RoiHoldoutStats | null;
  baseline?: RoiBaseline | null;
  [key: string]: unknown;
}

// ---------------------------------------------------------------------------
// API Client Factory
// ---------------------------------------------------------------------------

export function createIntelRoiApi(transport: FinanceRequestTransport) {
  return {
    // M1.5 Forecast v2
    getPlanningForecast: (
      query: {
        sku?: string;
        location_id?: string;
        horizon_days?: number;
        run_id?: string;
        limit?: number;
      } = {},
    ): Promise<PlanningForecastResponse> =>
      transport.request<PlanningForecastResponse>(withQuery("/v1/planning/forecast", query), "GET"),

    runPlanningForecast: (
      body: RunForecastRequest = {},
      idempotencyKey?: string,
    ): Promise<RunForecastResponse> =>
      transport.request<RunForecastResponse>("/v1/planning/forecast/run", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("forecast-v2-run", idempotencyKey),
      }),

    approveForecastExplanation: (
      explanationId: string,
      body: ApproveForecastExplanationRequest = {},
      idempotencyKey?: string,
    ): Promise<ApproveForecastExplanationResponse> => {
      const cleanId = encodeURIComponent(explanationId.trim());
      return transport.request<ApproveForecastExplanationResponse>(
        `/v1/planning/forecast/explanations/${cleanId}/approve`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`forecast-exp-approve-${explanationId.trim()}`, idempotencyKey),
        },
      );
    },

    approveReplenishmentSuggestion: (
      suggestionId: string,
      body: ApproveReplenishmentSuggestionRequest = {},
      idempotencyKey?: string,
    ): Promise<ApproveReplenishmentSuggestionResponse> => {
      const cleanId = encodeURIComponent(suggestionId.trim());
      return transport.request<ApproveReplenishmentSuggestionResponse>(
        `/v1/replenishment/suggestions/${cleanId}/approve`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`replenishment-approve-${suggestionId.trim()}`, idempotencyKey),
        },
      );
    },

    // M1.6 Credit v2
    getRetailerCreditScore: (
      retailerId: string,
      query: { scope?: "RELATIONSHIP" | "NETWORK" | "BOTH"; supplier_id?: string; recompute?: boolean } = {},
    ): Promise<RetailerCreditScoreResponse> => {
      const cleanId = encodeURIComponent(retailerId.trim());
      return transport.request<RetailerCreditScoreResponse>(
        withQuery(`/v1/credit/retailers/${cleanId}/score`, query),
        "GET",
      );
    },

    getCreditPortfolio: (
      query: { status?: string; risk_tier?: string; limit?: number } = {},
    ): Promise<CreditPortfolioResponse> =>
      transport.request<CreditPortfolioResponse>(withQuery("/v1/credit/portfolio", query), "GET"),

    updateCreditLimit: (
      retailerId: string,
      body: UpdateCreditLimitRequest,
      idempotencyKey?: string,
    ): Promise<UpdateCreditLimitResponse> => {
      const cleanId = encodeURIComponent(retailerId.trim());
      const resolvedIdem = makeIdempotencyKey(
        `credit-limit-${retailerId.trim()}`,
        idempotencyKey ?? body.idempotency_key,
      );
      const path = body.supplier_id
        ? withQuery(`/v1/credit/limits/${cleanId}`, { supplier_id: body.supplier_id })
        : `/v1/credit/limits/${cleanId}`;
      const resolvedLimit = body.new_limit_minor ?? body.credit_limit_minor;
      const payload: Record<string, unknown> = {
        new_limit_minor: resolvedLimit,
        reason: body.reason,
        idempotency_key: resolvedIdem,
      };
      if (body.approval_id) payload.approval_id = body.approval_id;
      if (body.consent_verified !== undefined) payload.consent_verified = body.consent_verified;
      return transport.request<UpdateCreditLimitResponse>(path, "POST", {
        body: payload,
        idempotencyKey: resolvedIdem,
      });
    },

    // M1.7 Proposals Inbox
    listProposals: (
      query: { agent_id?: string; status?: string; limit?: number } = {},
    ): Promise<ProposalsListResponse> =>
      transport.request<ProposalsListResponse>(withQuery("/v1/proposals", query), "GET"),

    createProposal: (
      body: CreateProposalRequest,
      idempotencyKey?: string,
    ): Promise<ProposalDetailResponse> =>
      transport.request<ProposalDetailResponse>("/v1/proposals", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("proposal-create", idempotencyKey),
      }),

    runProposalAgent: (
      agentId: L1AgentId | string,
      body: Record<string, unknown> = {},
      idempotencyKey?: string,
    ): Promise<RunProposalAgentResponse> => {
      const cleanAgent = encodeURIComponent(agentId.trim());
      return transport.request<RunProposalAgentResponse>(
        `/v1/proposals/agents/${cleanAgent}/run`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`proposal-agent-run-${agentId.trim()}`, idempotencyKey),
        },
      );
    },

    getProposal: (proposalId: string): Promise<ProposalDetailResponse> => {
      const cleanId = encodeURIComponent(proposalId.trim());
      return transport.request<ProposalDetailResponse>(`/v1/proposals/${cleanId}`, "GET");
    },

    approveProposal: (
      proposalId: string,
      body: ApproveProposalRequest = {},
      idempotencyKey?: string,
    ): Promise<ProposalDetailResponse> => {
      const cleanId = encodeURIComponent(proposalId.trim());
      return transport.request<ProposalDetailResponse>(`/v1/proposals/${cleanId}/approve`, "POST", {
        body,
        idempotencyKey: makeIdempotencyKey(`proposal-approve-${proposalId.trim()}`, idempotencyKey),
      });
    },

    rejectProposal: (
      proposalId: string,
      body: RejectProposalRequest = {},
      idempotencyKey?: string,
    ): Promise<ProposalDetailResponse> => {
      const cleanId = encodeURIComponent(proposalId.trim());
      return transport.request<ProposalDetailResponse>(`/v1/proposals/${cleanId}/reject`, "POST", {
        body,
        idempotencyKey: makeIdempotencyKey(`proposal-reject-${proposalId.trim()}`, idempotencyKey),
      });
    },

    executeProposal: (
      proposalId: string,
      body: ExecuteProposalRequest = {},
      idempotencyKey?: string,
    ): Promise<ExecuteProposalResponse> => {
      const cleanId = encodeURIComponent(proposalId.trim());
      return transport.request<ExecuteProposalResponse>(`/v1/proposals/${cleanId}/execute`, "POST", {
        body,
        idempotencyKey: makeIdempotencyKey(`proposal-execute-${proposalId.trim()}`, idempotencyKey),
      });
    },

    // M1.8 ROI v1
    getRoiBaseline: (): Promise<RoiBaselineResponse> =>
      transport.request<RoiBaselineResponse>("/v1/roi/baseline", "GET"),

    captureRoiBaseline: (
      body: CaptureRoiBaselineRequest,
      idempotencyKey?: string,
    ): Promise<{ baseline: RoiBaseline }> =>
      transport.request<{ baseline: RoiBaseline }>("/v1/roi/baseline", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("roi-baseline-capture", idempotencyKey),
      }),

    getRoiSummary: (query: { days?: number } = {}): Promise<RoiSummaryResponse> =>
      transport.request<RoiSummaryResponse>(withQuery("/v1/roi/summary", query), "GET"),

    listRoiEvents: (
      query: { category?: string; limit?: number } = {},
    ): Promise<RoiEventsListResponse> =>
      transport.request<RoiEventsListResponse>(withQuery("/v1/roi/events", query), "GET"),

    recordRoiEvent: (
      body: RecordRoiEventRequest,
      idempotencyKey?: string,
    ): Promise<{ event: RoiEvent }> =>
      transport.request<{ event: RoiEvent }>("/v1/roi/events", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("roi-event-record", idempotencyKey),
      }),

    getRoiReportJson: (query: { days?: number } = {}): Promise<RoiReportResponse> =>
      transport.request<RoiReportResponse>(
        withQuery("/v1/roi/report", { ...query, format: "json" }),
        "GET",
      ),

    downloadRoiReportCsv: (query: { days?: number } = {}): Promise<string> =>
      transport.request<string>(
        withQuery("/v1/roi/report", { ...query, format: "csv" }),
        "GET",
      ),
  };
}

export type IntelRoiApi = ReturnType<typeof createIntelRoiApi>;

export function isApiErrorStatus(err: unknown, status: number): boolean {
  return err instanceof ApiError && err.status === status;
}

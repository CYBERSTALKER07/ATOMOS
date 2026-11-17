import { ApiError } from "@pegasusx/api-core";
import { createSupplierApi } from "@/lib/api";
import {
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ControlErrorDetail,
} from "@/lib/control-accounting-api";
import type { FinanceRequestTransport } from "@/lib/finance-api";
import type {
  AnomalyCatalogueResponse,
  AnomalyCase,
  AnomalyCasesListResponse,
  CommitWhatIfProposalsRequest,
  CommitWhatIfProposalsResponse,
  CreateCustomRoleRequest,
  CreateScimGroupRequest,
  CreateScimUserRequest,
  CustomRole,
  CustomRoleResponse,
  CustomRolesListResponse,
  DeleteCustomRoleResponse,
  HaccpComplianceReport,
  L2EvaluateResponse,
  ResolveAnomalyCaseRequest,
  ResolveAnomalyCaseResponse,
  RunAnomalyScanRequest,
  RunAnomalyScanResponse,
  RunL2EvaluateRequest,
  RunWhatIfSimulationRequest,
  ScimGroup,
  ScimListResponse,
  ScimPatchRequest,
  ScimUser,
  UpdateScimGroupRequest,
  UpdateScimUserRequest,
  WhatIfShocks,
  WhatIfSimulation,
  WhatIfSimulationResponse,
  WhatIfSimulationsListResponse,
  WhatIfStageBreakdown,
  WhatIfStructuredBreakdown,
} from "@pegasusx/types";

export {
  extractControlError,
  formatFinanceMinor,
  formatRateBps,
  normalizeMinorBigInt,
  parseMajorToSafeMinor,
  type ControlErrorDetail,
};

const BIGINT_ZERO = BigInt(0);
const BIGINT_THOUSAND = BigInt(1000);
const SCIM_HEADERS = { Accept: "application/scim+json, application/json" };

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
 * Formats integer milli-Celsius values (e.g. -18500 -> "-18.5°C", 2300 -> "2.3°C") using pure BigInt math.
 */
export function formatMilliCelsius(milliC: number | bigint | null | undefined): string {
  if (milliC === null || milliC === undefined) return "—";
  const safe = normalizeMinorBigInt(milliC);
  const negative = safe < BIGINT_ZERO;
  const abs = negative ? -safe : safe;
  const whole = abs / BIGINT_THOUSAND;
  const rem = abs % BIGINT_THOUSAND;
  const padded = rem.toString().padStart(3, "0");
  const trimmed = padded.replace(/0+$/, "");
  const frac = trimmed.length === 0 ? "0" : trimmed;
  const sign = negative && (whole > BIGINT_ZERO || rem > BIGINT_ZERO) ? "-" : "";
  return `${sign}${groupIntegerDigits(whole.toString())}.${frac}°C`;
}

/**
 * Parses integer percentage strings (e.g. "+15.5", "-10", "2.50") into integer basis points (`bps`) without float arithmetic.
 */
export function parsePercentToBps(raw: string): number | null {
  const trimmed = raw.trim().replace(/%$/, "").trim();
  if (!trimmed) return null;
  const match = /^([+-])?(\d+)(?:\.(\d+))?$/.exec(trimmed);
  if (!match) return null;
  const sign = match[1] === "-" ? -1 : 1;
  const intPart = Number(match[2]);
  const fracDigits = (match[3] ?? "").slice(0, 2).padEnd(2, "0");
  const fracPart = Number(fracDigits);
  const total = sign * (intPart * 100 + fracPart);
  if (!Number.isSafeInteger(total)) return null;
  return total;
}

/**
 * Extracts structured error messages, including SCIM `application/scim+json` string error bodies.
 */
export function extractP2bError(err: unknown, fallback: string): ControlErrorDetail {
  if (err instanceof ApiError && typeof err.payload === "string") {
    const raw = err.payload.trim();
    if (raw.startsWith("{")) {
      try {
        const parsed = JSON.parse(raw) as Record<string, unknown>;
        const code =
          typeof parsed.code === "string" && parsed.code.trim()
            ? parsed.code.trim()
            : typeof parsed.scimType === "string" && parsed.scimType.trim()
              ? parsed.scimType.trim()
              : undefined;
        const parts = [
          code,
          typeof parsed.error === "string" ? parsed.error.trim() : "",
          typeof parsed.message === "string" ? parsed.message.trim() : "",
          typeof parsed.detail === "string" ? parsed.detail.trim() : "",
        ].filter((v): v is string => typeof v === "string" && v.length > 0);
        const unique = Array.from(new Set(parts));
        if (unique.length > 0) {
          return {
            message: unique.join(" · "),
            code,
            status: err.status,
            payload: parsed,
          };
        }
      } catch {
        // Fall back to standard extractor below
      }
    }
  }
  return extractControlError(err, fallback);
}

/**
 * Parses hydrated `evidence` or JSON-serialized `evidence_json` on an `AnomalyCase`.
 */
export function parseAnomalyEvidence(
  item: Pick<AnomalyCase, "evidence" | "evidence_json"> | null | undefined,
): Record<string, unknown> | null {
  if (!item) return null;
  if (item.evidence && typeof item.evidence === "object" && !Array.isArray(item.evidence)) {
    return item.evidence;
  }
  if (typeof item.evidence_json === "string" && item.evidence_json.trim() !== "") {
    try {
      const parsed = JSON.parse(item.evidence_json) as unknown;
      if (parsed && typeof parsed === "object" && !Array.isArray(parsed)) {
        return parsed as Record<string, unknown>;
      }
    } catch {
      return null;
    }
  }
  return null;
}

/**
 * Parses hydrated `shocks` or `shocks_json` on a `WhatIfSimulation`.
 */
export function parseWhatIfShocks(
  sim: Pick<WhatIfSimulation, "shocks" | "shocks_json"> | null | undefined,
): WhatIfShocks | null {
  if (!sim) return null;
  if (sim.shocks && typeof sim.shocks === "object") {
    return sim.shocks;
  }
  if (typeof sim.shocks_json === "string" && sim.shocks_json.trim() !== "") {
    try {
      const parsed = JSON.parse(sim.shocks_json) as WhatIfShocks;
      if (parsed && typeof parsed === "object") {
        return parsed;
      }
    } catch {
      return null;
    }
  }
  return null;
}

/**
 * Normalizes hydrated `breakdown` (array or structured WhatIfBreakdown) or `breakdown_json` into ordered stage rows.
 */
export function parseWhatIfBreakdown(
  sim: Pick<WhatIfSimulation, "breakdown" | "breakdown_json"> | null | undefined,
): WhatIfStageBreakdown[] {
  if (!sim) return [];
  let source: unknown = sim.breakdown;
  if (source === undefined && typeof sim.breakdown_json === "string" && sim.breakdown_json.trim()) {
    try {
      source = JSON.parse(sim.breakdown_json) as unknown;
    } catch {
      return [];
    }
  }
  if (Array.isArray(source)) {
    return source.filter((item): item is WhatIfStageBreakdown => Boolean(item && typeof item === "object"));
  }
  if (source && typeof source === "object") {
    const obj = source as WhatIfStructuredBreakdown & Record<string, unknown>;
    if (Array.isArray(obj.stages)) {
      return obj.stages.filter(
        (item): item is WhatIfStageBreakdown => Boolean(item && typeof item === "object"),
      );
    }
    if (obj.demand || obj.replenishment || obj.dispatch || obj.credit || obj.pnl) {
      const rows: WhatIfStageBreakdown[] = [];
      if (obj.demand && typeof obj.demand === "object") {
        rows.push({
          stage: "DEMAND",
          label: "Demand & Forecast",
          delta_minor: obj.demand.forecast_delta_minor,
          cost_minor: obj.demand.shocked_revenue_minor,
          detail: `SKUs: ${obj.demand.sku_count ?? 0}`,
        });
      }
      if (obj.replenishment && typeof obj.replenishment === "object") {
        const skuCount = Array.isArray(obj.replenishment.affected_skus)
          ? obj.replenishment.affected_skus.length
          : 0;
        rows.push({
          stage: "REPLENISHMENT",
          label: "Replenishment & FX",
          delta_minor: obj.replenishment.cost_delta_minor,
          cost_minor: obj.replenishment.shocked_cost_minor,
          detail: `Affected SKUs: ${skuCount}`,
        });
      }
      if (obj.dispatch && typeof obj.dispatch === "object") {
        rows.push({
          stage: "DISPATCH",
          label: "Fleet & Dispatch",
          delta_minor: obj.dispatch.dispatch_cost_delta_minor,
          cost_minor: obj.dispatch.shocked_cost_minor,
          detail: `Routes: ${obj.dispatch.route_count ?? 0} · Orders: ${obj.dispatch.order_count ?? 0}`,
        });
      }
      if (obj.credit && typeof obj.credit === "object") {
        rows.push({
          stage: "CREDIT",
          label: "AR & Credit Risk",
          delta_minor: obj.credit.credit_exposure_delta_minor,
          cost_minor: obj.credit.risk_cost_delta_minor,
          detail: `Over-limit retailers: ${obj.credit.over_limit_retailers ?? 0}`,
        });
      }
      if (obj.pnl && typeof obj.pnl === "object") {
        rows.push({
          stage: "PNL",
          label: "Net PnL Rollup",
          delta_minor: obj.pnl.net_pnl_delta_minor,
          cost_minor: obj.pnl.cogs_delta_minor,
          detail: obj.scenario_applied ? `Scenario patches: ${obj.patch_count ?? 0}` : "Live baseline",
        });
      }
      return rows;
    }
    return Object.entries(obj)
      .filter(([, val]) => val !== null && typeof val === "object" && !Array.isArray(val))
      .map(([stage, val]) => ({
        stage,
        ...(val as Record<string, unknown>),
      }));
  }
  return [];
}

/**
 * Extracts `breakdown.currency` when set on a structured `WhatIfBreakdown`.
 */
export function parseWhatIfBreakdownCurrency(
  sim: Pick<WhatIfSimulation, "breakdown" | "breakdown_json" | "shocks" | "shocks_json"> | null | undefined,
): string | null {
  if (!sim) return null;
  if (sim.breakdown && typeof sim.breakdown === "object" && !Array.isArray(sim.breakdown)) {
    const cur = (sim.breakdown as WhatIfStructuredBreakdown).currency;
    if (typeof cur === "string" && cur.trim()) return cur.trim();
  }
  if (typeof sim.breakdown_json === "string" && sim.breakdown_json.trim()) {
    try {
      const parsed = JSON.parse(sim.breakdown_json) as WhatIfStructuredBreakdown;
      if (typeof parsed?.currency === "string" && parsed.currency.trim()) {
        return parsed.currency.trim();
      }
    } catch {
      // Ignore malformed JSON
    }
  }
  return parseWhatIfShocks(sim)?.currency || null;
}

/**
 * Parses `committed_proposal_ids` or `committed_proposal_ids_json` from a `WhatIfSimulation`.
 */
export function parseCommittedProposalIds(
  sim: Pick<WhatIfSimulation, "committed_proposal_ids" | "committed_proposal_ids_json"> | null | undefined,
): string[] {
  if (!sim) return [];
  if (Array.isArray(sim.committed_proposal_ids)) {
    return sim.committed_proposal_ids.filter((id): id is string => typeof id === "string" && id.trim() !== "");
  }
  if (typeof sim.committed_proposal_ids_json === "string" && sim.committed_proposal_ids_json.trim()) {
    try {
      const parsed = JSON.parse(sim.committed_proposal_ids_json) as unknown;
      if (Array.isArray(parsed)) {
        return parsed.filter((id): id is string => typeof id === "string" && id.trim() !== "");
      }
    } catch {
      return [];
    }
  }
  return [];
}

function parseScimPayload<T>(raw: T | string | undefined): T {
  if (typeof raw === "string") {
    const trimmed = raw.trim();
    if (trimmed.startsWith("{") || trimmed.startsWith("[")) {
      try {
        return JSON.parse(trimmed) as T;
      } catch {
        return raw as unknown as T;
      }
    }
  }
  return raw as T;
}

function unwrapWhatIfSimulation(res: WhatIfSimulationResponse): WhatIfSimulation {
  if (res && typeof res === "object" && "simulation" in res && res.simulation) {
    return res.simulation;
  }
  return res as WhatIfSimulation;
}

function unwrapCustomRole(res: CustomRoleResponse): CustomRole {
  if (res && typeof res === "object" && "custom_role" in res && res.custom_role) {
    return res.custom_role;
  }
  return res as CustomRole;
}

export function createP2bApi(client?: FinanceRequestTransport) {
  const transport = client ?? createSupplierApi();

  return {
    // ── §7.7 Anomaly & Fraud Detector Catalogue, Scanner & Case Resolution ──
    getAnomalyCatalogue: (): Promise<AnomalyCatalogueResponse> =>
      transport.request<AnomalyCatalogueResponse>("/v1/anomalies/catalogue", "GET"),

    scanAnomalies: (
      body: RunAnomalyScanRequest = {},
      idempotencyKey?: string,
    ): Promise<RunAnomalyScanResponse> =>
      transport.request<RunAnomalyScanResponse>("/v1/anomalies/scan", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("anomalies-scan", idempotencyKey),
      }),

    listAnomalyCases: (
      query: {
        status?: string;
        severity?: string;
        category?: string;
        rule_code?: string;
        limit?: number;
      } = {},
    ): Promise<AnomalyCasesListResponse> =>
      transport.request<AnomalyCasesListResponse>(withQuery("/v1/anomalies/cases", query), "GET"),

    resolveAnomalyCase: (
      caseID: string,
      body: ResolveAnomalyCaseRequest,
      idempotencyKey?: string,
    ): Promise<ResolveAnomalyCaseResponse> => {
      const cleanId = encodeURIComponent(caseID.trim());
      return transport.request<ResolveAnomalyCaseResponse>(
        `/v1/anomalies/cases/${cleanId}/resolve`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`anomaly-resolve-${caseID.trim()}`, idempotencyKey),
        },
      );
    },

    // ── §7.9 Full E2E What-If Simulator ──
    runWhatIfSimulation: async (
      body: RunWhatIfSimulationRequest,
      idempotencyKey?: string,
    ): Promise<WhatIfSimulation> => {
      const resolvedKey = makeIdempotencyKey(
        "whatif-simulate",
        idempotencyKey ?? body.idempotency_key,
      );
      const res = await transport.request<WhatIfSimulationResponse>("/v1/whatif/simulate", "POST", {
        body: { ...body, idempotency_key: resolvedKey },
        idempotencyKey: resolvedKey,
      });
      return unwrapWhatIfSimulation(res);
    },

    listWhatIfSimulations: (
      query: { status?: string; limit?: number } = {},
    ): Promise<WhatIfSimulationsListResponse> =>
      transport.request<WhatIfSimulationsListResponse>(
        withQuery("/v1/whatif/simulations", query),
        "GET",
      ),

    getWhatIfSimulation: async (simID: string): Promise<WhatIfSimulation> => {
      const cleanId = encodeURIComponent(simID.trim());
      const res = await transport.request<WhatIfSimulationResponse>(
        `/v1/whatif/simulations/${cleanId}`,
        "GET",
      );
      return unwrapWhatIfSimulation(res);
    },

    commitWhatIfProposals: (
      simID: string,
      body: CommitWhatIfProposalsRequest = {},
      idempotencyKey?: string,
    ): Promise<CommitWhatIfProposalsResponse> => {
      const cleanId = encodeURIComponent(simID.trim());
      return transport.request<CommitWhatIfProposalsResponse>(
        `/v1/whatif/simulations/${cleanId}/commit-proposals`,
        "POST",
        {
          body,
          idempotencyKey: makeIdempotencyKey(`whatif-commit-${simID.trim()}`, idempotencyKey),
        },
      );
    },

    // ── §6.5 Cold-Chain HACCP Excursion & Compliance Report ──
    getHaccpReport: (
      query: { from?: string; to?: string } = {},
    ): Promise<HaccpComplianceReport> =>
      transport.request<HaccpComplianceReport>(
        withQuery("/v1/compliance/haccp", { ...query, format: "json" }),
        "GET",
      ),

    downloadHaccpCsv: (query: { from?: string; to?: string } = {}): Promise<string> =>
      transport.request<string>(
        withQuery("/v1/compliance/haccp", { ...query, format: "csv" }),
        "GET",
      ),

    // ── §7.5 L2 Soak Thresholds & Auto-Execution ──
    evaluateL2Readiness: (
      query: { max_l2_amount_minor?: number } = {},
    ): Promise<L2EvaluateResponse> =>
      transport.request<L2EvaluateResponse>(
        withQuery("/v1/proposals/l2-evaluate", query),
        "GET",
      ),

    runL2AutoExecute: (
      body: RunL2EvaluateRequest,
      idempotencyKey?: string,
    ): Promise<L2EvaluateResponse> =>
      transport.request<L2EvaluateResponse>("/v1/proposals/l2-evaluate", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey("proposals-l2-eval", idempotencyKey),
      }),

    // ── Enterprise Custom Roles & SCIM 2.0 ──
    listCustomRoles: (): Promise<CustomRolesListResponse> =>
      transport.request<CustomRolesListResponse>("/v1/iam/custom-roles", "GET"),

    createCustomRole: async (
      body: CreateCustomRoleRequest,
      idempotencyKey?: string,
    ): Promise<CustomRole> => {
      const res = await transport.request<CustomRoleResponse>("/v1/iam/custom-roles", "POST", {
        body,
        idempotencyKey: makeIdempotencyKey(`iam-custom-role-${body.role_name.trim()}`, idempotencyKey),
      });
      return unwrapCustomRole(res);
    },

    deleteCustomRole: (
      roleName: string,
      idempotencyKey?: string,
    ): Promise<DeleteCustomRoleResponse> => {
      const cleanName = encodeURIComponent(roleName.trim());
      return transport.request<DeleteCustomRoleResponse>(
        `/v1/iam/custom-roles/${cleanName}`,
        "DELETE",
        {
          idempotencyKey: makeIdempotencyKey(`iam-custom-role-del-${roleName.trim()}`, idempotencyKey),
        },
      );
    },

    listScimUsers: async (
      query: { filter?: string; startIndex?: number; count?: number } = {},
    ): Promise<ScimListResponse<ScimUser>> => {
      const res = await transport.request<ScimListResponse<ScimUser> | string>(
        withQuery("/scim/v2/Users", query),
        "GET",
        { headers: SCIM_HEADERS },
      );
      return parseScimPayload<ScimListResponse<ScimUser>>(res);
    },

    createScimUser: async (
      body: CreateScimUserRequest,
      idempotencyKey?: string,
    ): Promise<ScimUser> => {
      const res = await transport.request<ScimUser | string>("/scim/v2/Users", "POST", {
        body,
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-user-create-${body.userName.trim()}`, idempotencyKey),
      });
      return parseScimPayload<ScimUser>(res);
    },

    getScimUser: async (userID: string): Promise<ScimUser> => {
      const cleanId = encodeURIComponent(userID.trim());
      const res = await transport.request<ScimUser | string>(`/scim/v2/Users/${cleanId}`, "GET", {
        headers: SCIM_HEADERS,
      });
      return parseScimPayload<ScimUser>(res);
    },

    replaceScimUser: async (
      userID: string,
      body: UpdateScimUserRequest,
      idempotencyKey?: string,
    ): Promise<ScimUser> => {
      const cleanId = encodeURIComponent(userID.trim());
      const res = await transport.request<ScimUser | string>(`/scim/v2/Users/${cleanId}`, "PUT", {
        body,
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-user-put-${userID.trim()}`, idempotencyKey),
      });
      return parseScimPayload<ScimUser>(res);
    },

    patchScimUser: async (
      userID: string,
      body: ScimPatchRequest,
      idempotencyKey?: string,
    ): Promise<ScimUser> => {
      const cleanId = encodeURIComponent(userID.trim());
      const res = await transport.request<ScimUser | string>(`/scim/v2/Users/${cleanId}`, "PATCH", {
        body,
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-user-patch-${userID.trim()}`, idempotencyKey),
      });
      return parseScimPayload<ScimUser>(res);
    },

    deleteScimUser: async (userID: string, idempotencyKey?: string): Promise<void> => {
      const cleanId = encodeURIComponent(userID.trim());
      await transport.request<void>(`/scim/v2/Users/${cleanId}`, "DELETE", {
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-user-del-${userID.trim()}`, idempotencyKey),
      });
    },

    listScimGroups: async (
      query: { filter?: string; startIndex?: number; count?: number } = {},
    ): Promise<ScimListResponse<ScimGroup>> => {
      const res = await transport.request<ScimListResponse<ScimGroup> | string>(
        withQuery("/scim/v2/Groups", query),
        "GET",
        { headers: SCIM_HEADERS },
      );
      return parseScimPayload<ScimListResponse<ScimGroup>>(res);
    },

    createScimGroup: async (
      body: CreateScimGroupRequest,
      idempotencyKey?: string,
    ): Promise<ScimGroup> => {
      const res = await transport.request<ScimGroup | string>("/scim/v2/Groups", "POST", {
        body,
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-group-create-${body.displayName.trim()}`, idempotencyKey),
      });
      return parseScimPayload<ScimGroup>(res);
    },

    getScimGroup: async (groupID: string): Promise<ScimGroup> => {
      const cleanId = encodeURIComponent(groupID.trim());
      const res = await transport.request<ScimGroup | string>(`/scim/v2/Groups/${cleanId}`, "GET", {
        headers: SCIM_HEADERS,
      });
      return parseScimPayload<ScimGroup>(res);
    },

    replaceScimGroup: async (
      groupID: string,
      body: UpdateScimGroupRequest,
      idempotencyKey?: string,
    ): Promise<ScimGroup> => {
      const cleanId = encodeURIComponent(groupID.trim());
      const res = await transport.request<ScimGroup | string>(`/scim/v2/Groups/${cleanId}`, "PUT", {
        body,
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-group-put-${groupID.trim()}`, idempotencyKey),
      });
      return parseScimPayload<ScimGroup>(res);
    },

    patchScimGroup: async (
      groupID: string,
      body: ScimPatchRequest,
      idempotencyKey?: string,
    ): Promise<ScimGroup> => {
      const cleanId = encodeURIComponent(groupID.trim());
      const res = await transport.request<ScimGroup | string>(`/scim/v2/Groups/${cleanId}`, "PATCH", {
        body,
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-group-patch-${groupID.trim()}`, idempotencyKey),
      });
      return parseScimPayload<ScimGroup>(res);
    },

    deleteScimGroup: async (groupID: string, idempotencyKey?: string): Promise<void> => {
      const cleanId = encodeURIComponent(groupID.trim());
      await transport.request<void>(`/scim/v2/Groups/${cleanId}`, "DELETE", {
        headers: SCIM_HEADERS,
        idempotencyKey: makeIdempotencyKey(`scim-group-del-${groupID.trim()}`, idempotencyKey),
      });
    },
  };
}

export type P2bApi = ReturnType<typeof createP2bApi>;

// API client for the PLATFORM_ADMIN backend surface.
// Bearer token is supplied per-request from the session (see useAdminToken).

import type {
  Tenant,
  FlagOverride,
  AccuracyRow,
  AuditRow,
  FlagEval,
  MatchQueueItem,
  PartnerKey,
  BillingInvoice,
  BillingFeeSchedule,
} from "@pegasusx/types";

export type {
  Tenant,
  FlagOverride,
  AccuracyRow,
  AuditRow,
  FlagEval,
  MatchQueueItem,
  PartnerKey,
  BillingInvoice,
  BillingFeeSchedule,
};

export interface FinanceTrialBalanceLine {
  account_code: string;
  account_name?: string;
  class?: string;
  currency: string;
  debit_minor: number;
  credit_minor: number;
  net_minor: number;
}

export interface FinanceTrialBalanceReport {
  entity_id: string;
  period_id: string;
  currency: string;
  total_debits_minor: number;
  total_credits_minor: number;
  balanced: boolean;
  accounts: FinanceTrialBalanceLine[];
}

export interface FinanceAccountLedgerRow {
  entry_id: string;
  period_id: string;
  line: number;
  account_code: string;
  direction: "DEBIT" | "CREDIT";
  amount_minor: number;
  currency: string;
  functional_amount_minor: number;
  source_type: string;
  source_id: string;
  order_id?: string;
  memo?: string;
  description?: string;
  entry_hash: string;
  occurred_at: string;
  posted_at: string;
}

export type ReconDimension = "ORDERS_AR" | "STOCK_GL" | "CASH_PSP" | "TAX_FISCAL";

export interface FinanceReconDimensionCheck {
  dimension: ReconDimension;
  status: "GREEN" | "BREAK";
  expected_minor: number;
  actual_minor: number;
  delta_minor: number;
  break_count: number;
  exposure_minor: number;
  currency: string;
  metrics?: Record<string, number>;
}

export interface FinanceReconciliationRun {
  run_id: string;
  entity_id: string;
  period_id: string;
  mode: string;
  status: "GREEN" | "BREAKS_FOUND" | "FAILED";
  break_count: number;
  exposure_minor: number;
  currency: string;
  checks: Partial<Record<ReconDimension, FinanceReconDimensionCheck>>;
  triggered_by: string;
  started_at: string;
  completed_at?: string;
}

export interface FinanceReconciliationBreak {
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
  severity: "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";
  status: "OPEN" | "ACKNOWLEDGED" | "RESOLVED" | "WAIVED";
  owner_id?: string;
  sla_due_at?: string;
  resolution_note?: string;
  control_tower_exception_id?: string;
  created_at: string;
  updated_at: string;
}

export type AuditActorType = "human" | "agent" | "connector" | "system";

export interface AuditV2Event {
  tenant_id: string;
  day_bucket: string;
  event_id: string;
  seq_num: number;
  actor_id: string;
  actor_type: AuditActorType;
  role?: string;
  action: string;
  object_type?: string;
  object_id?: string;
  before_json?: string;
  after_json?: string;
  diff_gcs_uri?: string;
  reason_code?: string;
  approval_id?: string;
  trace_id?: string;
  ip?: string;
  device_id?: string;
  prev_hash: string;
  entry_hash: string;
  legal_hold: boolean;
  legal_hold_reason?: string;
  retention_until?: string;
  created_at: string;
  source?: string;
  pii_masked?: boolean;
}

export interface AuditChainVerifyResult {
  tenant_id: string;
  day_bucket: string;
  valid: boolean;
  event_count: number;
  head_event_id?: string;
  root_hash: string;
  anchored_root_hash?: string;
  anchor_matched?: boolean;
  broken_event_id?: string;
  failure_reason?: string;
}

export interface AuditChainAnchor {
  tenant_id: string;
  day_bucket: string;
  head_event_id: string;
  event_count: number;
  root_hash: string;
  worm_uri?: string;
  signer_key_id?: string;
  anchored_by: string;
  anchored_at: string;
}

export interface AuditLegalHold {
  tenant_id: string;
  hold_id: string;
  object_type: string;
  object_id: string;
  reason: string;
  status: "ACTIVE" | "RELEASED" | string;
  created_by: string;
  expires_at?: string;
  created_at: string;
}

export interface IamRoleDefinition {
  role_name: string;
  description: string;
  read_only: boolean;
  agent_role: boolean;
  permissions: string[];
}

export interface IamSodPolicy {
  tenant_id: string;
  policy_id: string;
  name: string;
  permission_a: string;
  permission_b: string;
  enforce_same_object: boolean;
  enabled: boolean;
  created_at: string;
}

export interface IamSodViolation {
  tenant_id: string;
  violation_id: string;
  actor_id: string;
  action: string;
  object_type?: string;
  object_id?: string;
  maker_id?: string;
  rule_code: string;
  route_path?: string;
  trace_id?: string;
  created_at: string;
}

const base = () =>
  (process.env.NEXT_PUBLIC_BACKEND_BASE_URL || "http://localhost:8080").replace(/\/$/, "");

const inFlightAdminGets = new Map<string, Promise<unknown>>();

const ADMIN_COOKIE_SESSION = "__cookie_session__";

function resolveAdminRequestUrl(token: string, path: string): string {
  if (token === ADMIN_COOKIE_SESSION) {
    return `/api/proxy${path.startsWith("/") ? path : `/${path}`}`;
  }
  return base() + path;
}

async function req<T>(token: string, method: string, path: string, body?: unknown): Promise<T> {
  const normalizedMethod = method.toUpperCase();
  const url = resolveAdminRequestUrl(token, path);

  if (normalizedMethod === "GET") {
    const key = `${token}|${url}`;
    const existing = inFlightAdminGets.get(key);
    if (existing) {
      return existing as Promise<T>;
    }
    const task = executeAdminReq<T>(token, normalizedMethod, url, body).finally(() => {
      inFlightAdminGets.delete(key);
    });
    inFlightAdminGets.set(key, task);
    return task;
  }

  inFlightAdminGets.clear();
  return executeAdminReq<T>(token, normalizedMethod, url, body);
}

async function executeAdminReq<T>(token: string, method: string, url: string, body?: unknown): Promise<T> {
  const isCookieSession = token === ADMIN_COOKIE_SESSION;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
  };
  if (!isCookieSession && token) {
    headers.Authorization = `Bearer ${token}`;
  }
  if (typeof crypto !== "undefined" && typeof crypto.randomUUID === "function") {
    headers["X-Trace-Id"] = crypto.randomUUID();
    if (method === "POST" || method === "PUT" || method === "PATCH" || method === "DELETE") {
      headers["Idempotency-Key"] = crypto.randomUUID();
    }
  }
  const res = await fetch(url, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: isCookieSession ? "include" : "same-origin",
    cache: "no-store",
  });
  if (!res.ok) {
    let msg = `HTTP ${res.status}`;
    try {
      const j = await res.json();
      if (j?.error) msg = j.error;
    } catch {
      /* non-json */
    }
    throw new Error(msg);
  }
  return (await res.json()) as T;
}

export const api = {
  listTenants: (token: string, status = "", limit = 100) =>
    req<{ tenants: Tenant[] }>(token, "GET", `/v1/platform-admin/tenants?status=${encodeURIComponent(status)}&limit=${limit}`),
  getTenant: (token: string, type: string, id: string) =>
    req<Tenant>(token, "GET", `/v1/platform-admin/tenants/${type}/${id}`),
  transitionTenant: (token: string, type: string, id: string, status: string, kybNotes = "") =>
    req<Tenant>(token, "POST", `/v1/platform-admin/tenants/${type}/${id}/transition`, { status, kyb_notes: kybNotes }),
  listAudit: (token: string, limit = 100) =>
    req<{ audit: AuditRow[] }>(token, "GET", `/v1/platform-admin/audit?limit=${limit}`),
  listPendingFlags: (token: string) =>
    req<{ items: FlagOverride[]; count: number; available?: boolean }>(token, "GET", "/v1/platform-admin/flags/"),
  evalFlag: (token: string, flagKey: string, tenantType = "", tenantId = "") =>
    req<FlagEval>(token, "GET", `/v1/platform-admin/flags/${encodeURIComponent(flagKey)}?tenant_type=${tenantType}&tenant_id=${tenantId}`),
  setFlag: (token: string, flagKey: string, tenantType: string, tenantId: string, enabled: boolean, reason: string) =>
    req<{ ok: boolean; status: string }>(token, "PUT", `/v1/platform-admin/flags/${encodeURIComponent(flagKey)}`, {
      tenant_type: tenantType,
      tenant_id: tenantId,
      enabled,
      reason,
    }),
  approveFlag: (token: string, flagKey: string, tenantType: string, tenantId: string) =>
    req<{ ok: boolean; status: string }>(token, "POST", `/v1/platform-admin/flags/${encodeURIComponent(flagKey)}/approve`, {
      tenant_type: tenantType,
      tenant_id: tenantId,
    }),
  listMatchQueue: (token: string, status = "PENDING", limit = 100) =>
    req<{ items: MatchQueueItem[] }>(
      token,
      "GET",
      `/v1/admin/product-match-queue?status=${encodeURIComponent(status)}&limit=${limit}`,
    ),
  resolveMatch: (token: string, id: string, decision: string, globalProductId = "") =>
    req<{ status: string }>(token, "POST", `/v1/admin/product-match-queue/${encodeURIComponent(id)}/resolve`, {
      decision,
      global_product_id: globalProductId,
    }),
  listPartnerKeys: (token: string, tenantType: string, tenantId: string) =>
    req<{ keys: PartnerKey[] }>(
      token,
      "GET",
      `/v1/admin/partner-keys?tenant_type=${encodeURIComponent(tenantType)}&tenant_id=${encodeURIComponent(tenantId)}`,
    ),
  // B5 M-P0-10: PLATFORM_ADMIN revoke requires tenant scope (query + body).
  revokePartnerKey: (token: string, keyId: string, tenantType: string, tenantId: string) =>
    req<{ ok?: boolean; status?: string }>(
      token,
      "POST",
      `/v1/admin/partner-keys/${encodeURIComponent(keyId)}/revoke?tenant_type=${encodeURIComponent(tenantType)}&tenant_id=${encodeURIComponent(tenantId)}`,
      { tenant_type: tenantType, tenant_id: tenantId },
    ),
  getPartnerAs2: (token: string, tenantType: string, tenantId: string) =>
    req<Record<string, unknown>>(
      token,
      "GET",
      `/v1/admin/partner-as2?tenant_type=${encodeURIComponent(tenantType)}&tenant_id=${encodeURIComponent(tenantId)}`,
    ),
  getPartnerSftp: (token: string, tenantType: string, tenantId: string) =>
    req<Record<string, unknown>>(
      token,
      "GET",
      `/v1/admin/partner-sftp?tenant_type=${encodeURIComponent(tenantType)}&tenant_id=${encodeURIComponent(tenantId)}`,
    ),
  getPartnerCoa: (token: string, tenantType: string, tenantId: string) =>
    req<Record<string, unknown>>(
      token,
      "GET",
      `/v1/admin/partner-coa?tenant_type=${encodeURIComponent(tenantType)}&tenant_id=${encodeURIComponent(tenantId)}`,
    ),
  runDunningOnce: (token: string) =>
    req<Record<string, unknown>>(token, "POST", `/v1/admin/ar/dunning/run-once`, {}),

  /** G4.B password login (no bearer). */
  platformAdminLogin: async (subject: string, password: string) => {
    const res = await fetch(`${base()}/v1/auth/platform-admin/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ subject, password }),
    });
    if (!res.ok) {
      const body = await res.text();
      throw new Error(body || `login_${res.status}`);
    }
    return res.json() as Promise<{ token: string; subject: string; role: string }>;
  },
  outboxSummary: (token: string) =>
    req<{
      unpublished_count?: number;
      unpublished_available?: boolean;
      dead_letter_count?: number;
      dead_letter_available?: boolean;
      oldest_age_seconds: number;
      lagging?: boolean;
      available?: boolean;
      note?: string;
    }>(token, "GET", "/v1/platform-admin/ops/outbox/summary"),
  outboxEvents: (token: string) =>
    req<{ events: Array<Record<string, string>>; count: number; note?: string }>(
      token,
      "GET",
      "/v1/platform-admin/ops/outbox/events?limit=25",
    ),
  outboxDeadLetters: (token: string) =>
    req<{
      items: Array<Record<string, string | number>>;
      page_count?: number;
      dead_letter_count?: number;
      available?: boolean;
      note?: string;
    }>(token, "GET", "/v1/platform-admin/ops/outbox/dead-letters?limit=25"),
  replayDeadLetter: (token: string, eventId: string) =>
    req<{ ok?: boolean; status?: string; event_id?: string }>(
      token,
      "POST",
      `/v1/platform-admin/ops/outbox/dead-letters/${encodeURIComponent(eventId)}/replay`,
      {},
    ),
  listPlanningAccuracy: (token: string, supplierId: string, days = 28) =>
    req<{ items: AccuracyRow[]; demote_enabled?: boolean }>(
      token,
      "GET",
      `/v1/admin/planning/accuracy?supplier_id=${encodeURIComponent(supplierId)}&days=${days}`,
    ),
  runtimeOps: (token: string) =>
    req<Record<string, unknown>>(token, "GET", "/v1/platform-admin/ops/runtime"),

  mfaStatus: (token: string) =>
    req<{ enrolled: boolean; required: boolean; verified: boolean }>(token, "GET", "/v1/platform-admin/mfa/status"),
  mfaEnroll: (token: string) =>
    req<{ secret: string; otpauth_url: string; subject: string }>(token, "POST", "/v1/platform-admin/mfa/enroll"),
  mfaConfirm: (token: string, code: string) =>
    req<{ ok: boolean; enrolled: boolean; token: string }>(token, "POST", "/v1/platform-admin/mfa/confirm", { code }),
  mfaVerify: (token: string, code: string) =>
    req<{ ok: boolean; verified: boolean; token: string }>(token, "POST", "/v1/platform-admin/mfa/verify", { code }),

  wsSession: (token: string) =>
    req<{ token: string; expires_at: string }>(token, "GET", "/v1/platform-admin/ws-session"),

  listBillingInvoices: (token: string) =>
    req<{ invoices: BillingInvoice[] }>(token, "GET", "/v1/admin/billing/invoices"),
  listBillingFeeSchedules: (token: string) =>
    req<{ fee_schedules: BillingFeeSchedule[] }>(token, "GET", "/v1/admin/billing/fee-schedules"),
  runMonthlyBilling: (token: string, month?: string) =>
    req<{ billed: number; month: string }>(token, "POST", "/v1/admin/billing/run-monthly", month ? { month } : {}),

  getFinanceTrialBalance: (token: string, entityId: string, period?: string) => {
    const params = new URLSearchParams({ entity: entityId.trim() });
    if (period?.trim()) params.set("period", period.trim());
    return req<FinanceTrialBalanceReport>(token, "GET", `/v1/finance/gl/trial-balance?${params.toString()}`);
  },
  getFinanceAccountLedger: (
    token: string,
    entityId: string,
    accountCode: string,
    period?: string,
    limit = 20,
  ) => {
    const params = new URLSearchParams({ entity: entityId.trim(), limit: String(limit) });
    if (period?.trim()) params.set("period", period.trim());
    return req<{
      entity_id: string;
      account_code: string;
      period_id: string;
      limit: number;
      count: number;
      items: FinanceAccountLedgerRow[];
    }>(
      token,
      "GET",
      `/v1/finance/gl/accounts/${encodeURIComponent(accountCode.trim().toUpperCase())}/ledger?${params.toString()}`,
    );
  },
  listFinanceReconRuns: (token: string, entityId: string, limit = 10) => {
    const params = new URLSearchParams({ entity: entityId.trim(), limit: String(limit) });
    return req<{ entity_id: string; runs: FinanceReconciliationRun[]; count: number }>(
      token,
      "GET",
      `/v1/finance/recon/runs?${params.toString()}`,
    );
  },
  triggerFinanceReconRun: (token: string, entityId: string, periodId?: string) =>
    req<{ run: FinanceReconciliationRun; breaks: FinanceReconciliationBreak[] }>(
      token,
      "POST",
      "/v1/finance/recon/run",
      {
        entity_id: entityId.trim(),
        ...(periodId?.trim() ? { period_id: periodId.trim() } : {}),
        mode: "MANUAL",
      },
    ),
  listFinanceReconBreaks: (token: string, entityId: string, status = "", limit = 50) => {
    const params = new URLSearchParams({ entity: entityId.trim(), limit: String(limit) });
    if (status.trim()) params.set("status", status.trim());
    return req<{ entity_id: string; breaks: FinanceReconciliationBreak[]; count: number }>(
      token,
      "GET",
      `/v1/finance/recon/breaks?${params.toString()}`,
    );
  },
  updateFinanceReconBreak: (
    token: string,
    entityId: string,
    breakId: string,
    body: { action: "ACKNOWLEDGE" | "RESOLVE" | "WAIVE"; note?: string; resolution_note?: string },
  ) =>
    req<{ break: FinanceReconciliationBreak }>(
      token,
      "PATCH",
      `/v1/finance/recon/breaks/${encodeURIComponent(breakId.trim())}?entity=${encodeURIComponent(entityId.trim())}`,
      body,
    ),

  listAuditV2Events: (
    token: string,
    params: {
      entity?: string;
      object?: string;
      object_id?: string;
      actor?: string;
      actor_type?: AuditActorType | "";
      date?: string;
      limit?: number;
    } = {},
  ) => {
    const q = new URLSearchParams();
    if (params.entity?.trim()) q.set("entity", params.entity.trim());
    if (params.object?.trim()) q.set("object", params.object.trim());
    if (params.object_id?.trim()) q.set("object_id", params.object_id.trim());
    if (params.actor?.trim()) q.set("actor", params.actor.trim());
    if (params.actor_type) q.set("actor_type", params.actor_type);
    if (params.date?.trim()) q.set("date", params.date.trim());
    if (params.limit != null) q.set("limit", String(params.limit));
    const qs = q.toString();
    return req<{ tenant_id: string; pii_masked: boolean; count: number; events: AuditV2Event[] }>(
      token,
      "GET",
      qs ? `/v1/audit/events?${qs}` : "/v1/audit/events",
    );
  },

  verifyAuditChain: (token: string, entityId: string, date?: string) => {
    const q = new URLSearchParams({ entity: entityId.trim() });
    if (date?.trim()) q.set("date", date.trim());
    return req<AuditChainVerifyResult>(token, "GET", `/v1/audit/verify?${q.toString()}`);
  },

  listAuditAnchors: (
    token: string,
    params: { entity?: string; date?: string; limit?: number } = {},
  ) => {
    const q = new URLSearchParams();
    if (params.entity?.trim()) q.set("entity", params.entity.trim());
    if (params.date?.trim()) q.set("date", params.date.trim());
    if (params.limit != null) q.set("limit", String(params.limit));
    const qs = q.toString();
    return req<{ tenant_id: string; count: number; anchors: AuditChainAnchor[] }>(
      token,
      "GET",
      qs ? `/v1/audit/anchors?${qs}` : "/v1/audit/anchors",
    );
  },

  writeAuditAnchor: (token: string, entityId: string, date?: string) =>
    req<AuditChainAnchor>(token, "POST", "/v1/audit/anchors", {
      entity_id: entityId.trim(),
      ...(date?.trim() ? { date: date.trim() } : {}),
    }),

  listAuditLegalHolds: (
    token: string,
    params: { entity?: string; object?: string; object_id?: string; status?: string; limit?: number } = {},
  ) => {
    const q = new URLSearchParams();
    if (params.entity?.trim()) q.set("entity", params.entity.trim());
    if (params.object?.trim()) q.set("object", params.object.trim());
    if (params.object_id?.trim()) q.set("object_id", params.object_id.trim());
    if (params.status?.trim()) q.set("status", params.status.trim());
    if (params.limit != null) q.set("limit", String(params.limit));
    const qs = q.toString();
    return req<{ tenant_id: string; count: number; holds: AuditLegalHold[] }>(
      token,
      "GET",
      qs ? `/v1/audit/legal-holds?${qs}` : "/v1/audit/legal-holds",
    );
  },

  createAuditLegalHold: (
    token: string,
    body: {
      entity_id: string;
      object_type: string;
      object_id: string;
      reason: string;
      expires_at?: string;
    },
  ) => req<AuditLegalHold>(token, "POST", "/v1/audit/legal-hold", body),

  listIamRoles: (token: string, entityId?: string) => {
    const q = new URLSearchParams();
    if (entityId?.trim()) q.set("entity", entityId.trim());
    const qs = q.toString();
    return req<{
      tenant_id: string;
      roles: IamRoleDefinition[];
      user_roles: Array<{
        tenant_id: string;
        user_id: string;
        role_name: string;
        assigned_by: string;
        created_at: string;
        updated_at: string;
      }>;
    }>(token, "GET", qs ? `/v1/iam/roles?${qs}` : "/v1/iam/roles");
  },

  listSodPolicies: (token: string, entityId?: string) => {
    const q = new URLSearchParams();
    if (entityId?.trim()) q.set("entity", entityId.trim());
    const qs = q.toString();
    return req<{ tenant_id: string; policies: IamSodPolicy[]; count: number }>(
      token,
      "GET",
      qs ? `/v1/iam/sod-policies?${qs}` : "/v1/iam/sod-policies",
    );
  },

  updateSodPolicy: (
    token: string,
    policyId: string,
    body: {
      entity_id: string;
      name?: string;
      permission_a?: string;
      permission_b?: string;
      enforce_same_object?: boolean;
      enabled?: boolean;
    },
  ) =>
    req<{ policy: IamSodPolicy }>(
      token,
      "PUT",
      `/v1/iam/sod-policies/${encodeURIComponent(policyId.trim())}`,
      body,
    ),

  listSodViolations: (
    token: string,
    params: { entity?: string; actor?: string; limit?: number } = {},
  ) => {
    const q = new URLSearchParams();
    if (params.entity?.trim()) q.set("entity", params.entity.trim());
    if (params.actor?.trim()) q.set("actor", params.actor.trim());
    if (params.limit != null) q.set("limit", String(params.limit));
    const qs = q.toString();
    return req<{ tenant_id: string; violations: IamSodViolation[]; count: number }>(
      token,
      "GET",
      qs ? `/v1/iam/sod-violations?${qs}` : "/v1/iam/sod-violations",
    );
  },
};


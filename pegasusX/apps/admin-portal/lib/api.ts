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

const base = () =>
  (process.env.NEXT_PUBLIC_BACKEND_BASE_URL || "http://localhost:8080").replace(/\/$/, "");

async function req<T>(token: string, method: string, path: string, body?: unknown): Promise<T> {
  const res = await fetch(base() + path, {
    method,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: body === undefined ? undefined : JSON.stringify(body),
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
};


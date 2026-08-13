// API client for the PLATFORM_ADMIN backend surface.
// Bearer token is supplied per-request from the session (see useAdminToken).

export interface Tenant {
  TenantType: string;
  TenantID: string;
  Status: string;
  DisplayName: string;
  KybNotes: string;
  CreatedAt: string;
  UpdatedAt: string;
  ApprovedAt?: string | null;
  SuspendedAt?: string | null;
  OffboardedAt?: string | null;
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

export interface FlagEval {
  flag_key: string;
  enabled: boolean;
  source: string;
  money_affecting: boolean;
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
};

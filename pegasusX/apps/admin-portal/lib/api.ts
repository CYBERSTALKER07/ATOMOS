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
};

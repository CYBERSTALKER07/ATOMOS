/**
 * Wave C1.3 multi-org auth client helpers (select-org / switch-org / memberships).
 * Flag-off backends never return pending_org_select — callers still handle full path.
 */
import { storeToken, isTauri } from "./bridge";
import { clearOrgScopedState } from "./clear-org-scoped-state";
import { setRetailerProfile } from "./retailer-profile";
import type {
  RetailerLoginResponse,
  RetailerMembershipDTO,
  RetailerMembershipsResponse,
} from "@pegasusx/types";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8180";

const PENDING_TOKEN_COOKIE = "pegasus_retailer_pending_jwt";

export function isPendingOrgSelectResponse(
  data: RetailerLoginResponse | null | undefined,
): boolean {
  if (!data?.token) return false;
  return data.token_type === "pending_org_select";
}

export function persistFullAuthTokens(data: {
  token: string;
  refresh_token?: string;
}): void {
  if (typeof document === "undefined") return;
  document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
  if (data.refresh_token) {
    document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
  }
  // Drop intermediate cookie if any
  document.cookie = `${PENDING_TOKEN_COOKIE}=; Max-Age=0; path=/`;
  if (isTauri()) {
    void storeToken(data.token, data.refresh_token || "");
  }
}

export function persistPendingOrgToken(token: string, expiresInSec = 420): void {
  if (typeof document === "undefined") return;
  const maxAge = Math.max(60, Math.min(600, expiresInSec));
  document.cookie = `${PENDING_TOKEN_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=${maxAge}; SameSite=Lax`;
  // Do not set full session cookie until select-org.
}

export function readPendingOrgToken(): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(
    new RegExp(`(?:^|; )${PENDING_TOKEN_COOKIE}=([^;]*)`),
  );
  return match ? decodeURIComponent(match[1]) : "";
}

export function stashPendingMemberships(memberships: RetailerMembershipDTO[]): void {
  if (typeof sessionStorage === "undefined") return;
  try {
    sessionStorage.setItem(
      "retailer_pending_memberships_v1",
      JSON.stringify(memberships),
    );
  } catch {
    /* ignore */
  }
}

export function loadStashedMemberships(): RetailerMembershipDTO[] {
  if (typeof sessionStorage === "undefined") return [];
  try {
    const raw = sessionStorage.getItem("retailer_pending_memberships_v1");
    if (!raw) return [];
    const parsed = JSON.parse(raw) as RetailerMembershipDTO[];
    return Array.isArray(parsed) ? parsed : [];
  } catch {
    return [];
  }
}

export function clearStashedMemberships(): void {
  if (typeof sessionStorage === "undefined") return;
  try {
    sessionStorage.removeItem("retailer_pending_memberships_v1");
  } catch {
    /* ignore */
  }
}

/** Apply full auth login/select/switch response: clear org state, tokens, profile. */
export async function applyFullAuthResponse(
  data: RetailerLoginResponse,
  opts?: { clearScoped?: boolean },
): Promise<void> {
  if (!data.token) throw new Error("missing_token");
  if (opts?.clearScoped !== false) {
    await clearOrgScopedState();
  }
  persistFullAuthTokens({
    token: data.token,
    refresh_token: data.refresh_token,
  });
  clearStashedMemberships();
  if (data.user) {
    const orgId = data.user.retailer_id || data.retailer_id || data.retailer_org_id || "";
    await setRetailerProfile({
      id: orgId || data.user.id,
      name: data.user.name,
      company: data.user.company || data.user.name,
      email: data.user.email || "",
      avatar_url: data.user.avatar_url ?? null,
      receiving_window_open: null,
      receiving_window_close: null,
    });
    // Stash org id separately for switcher highlight
    if (typeof sessionStorage !== "undefined" && orgId) {
      try {
        sessionStorage.setItem("retailer_active_org_id_v1", orgId);
      } catch {
        /* ignore */
      }
    }
  }
}

export function readActiveOrgId(): string {
  if (typeof sessionStorage === "undefined") return "";
  try {
    return sessionStorage.getItem("retailer_active_org_id_v1") || "";
  } catch {
    return "";
  }
}

export async function selectOrg(retailerId: string): Promise<RetailerLoginResponse> {
  const pending = readPendingOrgToken();
  if (!pending) throw new Error("pending_org_token_missing");
  const res = await fetch(`${API}/v1/auth/retailer/select-org`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${pending}`,
    },
    body: JSON.stringify({ retailer_id: retailerId }),
  });
  const data = (await res.json().catch(() => ({}))) as RetailerLoginResponse & {
    error?: string;
    code?: string;
  };
  if (!res.ok) {
    throw new Error(data.code || data.error || `select_org_failed_${res.status}`);
  }
  await applyFullAuthResponse(data, { clearScoped: true });
  return data;
}

export async function switchOrg(retailerId: string): Promise<RetailerLoginResponse> {
  // Use full session token via cookie-backed get path
  const { getRetailerToken } = await import("./auth");
  const token = await getRetailerToken();
  const res = await fetch(`${API}/v1/auth/retailer/switch-org`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${token}`,
    },
    body: JSON.stringify({ retailer_id: retailerId }),
  });
  const data = (await res.json().catch(() => ({}))) as RetailerLoginResponse & {
    error?: string;
    code?: string;
  };
  if (!res.ok) {
    throw new Error(data.code || data.error || `switch_org_failed_${res.status}`);
  }
  await applyFullAuthResponse(data, { clearScoped: true });
  return data;
}

export async function listMemberships(): Promise<RetailerMembershipDTO[]> {
  const pending = readPendingOrgToken();
  let token = pending;
  if (!token) {
    const { getRetailerToken } = await import("./auth");
    token = await getRetailerToken();
  }
  const res = await fetch(`${API}/v1/auth/retailer/memberships`, {
    headers: { Authorization: `Bearer ${token}` },
  });
  if (!res.ok) {
    throw new Error(`list_memberships_failed_${res.status}`);
  }
  const data = (await res.json()) as RetailerMembershipsResponse;
  return data.memberships ?? [];
}

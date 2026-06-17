import { clearStoredToken, getStoredToken, isTauri, storeToken } from "@/lib/bridge";

const SUPPLIER_JWT_COOKIE = "supplier_jwt";
const SUPPLIER_REFRESH_COOKIE = "pegasus_supplier_refresh";
const SUPPLIER_JWT_STORAGE = "supplier_jwt";

export function supplierApiBaseUrl(): string {
  const envUrl = (
    process.env.NEXT_PUBLIC_API_URL ||
    process.env.NEXT_PUBLIC_SUPPLIER_BACKEND_BASE_URL ||
    process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
    ""
  ).replace(/\/$/, "");
  if (envUrl) return envUrl;
  if (typeof window !== "undefined" && isTauri()) {
    return "http://10.0.2.2:8180";
  }
  if (typeof window !== "undefined") {
    return "/api";
  }
  return "http://localhost:8180";
}

export function readTokenFromCookie(): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(new RegExp(`(?:^|; )${SUPPLIER_JWT_COOKIE}=([^;]*)`));
  if (match) return decodeURIComponent(match[1]);
  return "";
}

function readRefreshFromCookie(): string {
  if (typeof document === "undefined") return "";
  const match = document.cookie.match(new RegExp(`(?:^|; )${SUPPLIER_REFRESH_COOKIE}=([^;]*)`));
  if (match) return decodeURIComponent(match[1]);
  return "";
}

export function persistSession(token: string, refreshToken?: string) {
  if (typeof document !== "undefined") {
    document.cookie = `${SUPPLIER_JWT_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=86400; SameSite=Lax`;
    if (refreshToken) {
      document.cookie = `${SUPPLIER_REFRESH_COOKIE}=${encodeURIComponent(refreshToken)}; path=/; max-age=604800; SameSite=Lax`;
    }
    try {
      localStorage.setItem(SUPPLIER_JWT_STORAGE, token);
    } catch {
      // ignore quota errors on constrained devices
    }
  }
  if (isTauri()) {
    void storeToken(token);
  }
}

export function clearSession() {
  if (typeof document !== "undefined") {
    document.cookie = `${SUPPLIER_JWT_COOKIE}=; Max-Age=0; path=/`;
    document.cookie = `${SUPPLIER_REFRESH_COOKIE}=; Max-Age=0; path=/`;
    try {
      localStorage.removeItem(SUPPLIER_JWT_STORAGE);
    } catch {
      // ignore
    }
  }
  if (isTauri()) {
    void clearStoredToken();
  }
}

export function getSupplierToken(): string {
  const cookie = readTokenFromCookie();
  if (cookie) return cookie;
  if (typeof window !== "undefined") {
    try {
      const stored = localStorage.getItem(SUPPLIER_JWT_STORAGE);
      if (stored) return stored;
    } catch {
      // ignore
    }
  }
  return "";
}

export async function resolveSupplierToken(): Promise<string> {
  const desktop = await getStoredToken();
  if (desktop) return desktop;
  return getSupplierToken();
}

export function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split(".");
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, "+").replace(/_/g, "/"));
    return JSON.parse(payload) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function readIsConfigured(token: string): boolean {
  const claims = decodeJwtPayload(token);
  return claims?.is_configured === true;
}

let refreshInFlight: Promise<string | null> | null = null;

async function tryRefreshToken(): Promise<string | null> {
  const refresh = readRefreshFromCookie();
  if (!refresh) return null;
  const base = supplierApiBaseUrl();
  try {
    const res = await fetch(`${base}/v1/auth/supplier/refresh`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ refresh_token: refresh }),
      credentials: base === "/api" ? "include" : undefined,
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { token?: string; refresh_token?: string };
    if (!data.token) return null;
    persistSession(data.token, data.refresh_token);
    return data.token;
  } catch {
    return null;
  }
}

export async function supplierFetch(path: string, init?: RequestInit): Promise<Response> {
  const token = await resolveSupplierToken();
  const base = supplierApiBaseUrl();
  const url = path.startsWith("http") ? path : `${base}${path.startsWith("/") ? path : `/${path}`}`;
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(init?.headers as Record<string, string>),
  };
  if (token) {
    headers.Authorization = `Bearer ${token}`;
  }
  const res = await fetch(url, {
    ...init,
    headers,
    credentials: base === "/api" ? "include" : init?.credentials,
  });

  if (res.status === 401) {
    if (!refreshInFlight) {
      refreshInFlight = tryRefreshToken().finally(() => {
        refreshInFlight = null;
      });
    }
    const newToken = await refreshInFlight;
    if (newToken) {
      const retryHeaders: Record<string, string> = {
        ...headers,
        Authorization: `Bearer ${newToken}`,
      };
      return fetch(url, {
        ...init,
        headers: retryHeaders,
        credentials: base === "/api" ? "include" : init?.credentials,
      });
    }
    clearSession();
    if (typeof window !== "undefined") {
      window.location.href = "/auth/login";
    }
    throw new Error("Session expired");
  }

  return res;
}

function toApiPath(input: RequestInfo | URL): string {
  const raw = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
  const base = supplierApiBaseUrl();
  if (raw.startsWith(base)) {
    return raw.slice(base.length);
  }
  try {
    const url = new URL(raw, base);
    return `${url.pathname}${url.search}`;
  } catch {
    return raw.startsWith("/") ? raw : `/${raw}`;
  }
}

/** fetch-compatible wrapper for session reconciliation (full URL in, path-based supplierFetch). */
export const supplierSessionFetch: typeof fetch = (input, init) =>
  supplierFetch(toApiPath(input), init);

import { homeCellFromJwt, pinApiBaseUrl, readCachedAuthSession } from '@pegasusx/api-core';
import { clearStoredToken, getStoredToken, isTauri, storeToken } from "@/lib/bridge";

const SUPPLIER_JWT_COOKIE = "supplier_jwt";
const SUPPLIER_REFRESH_COOKIE = "pegasus_supplier_refresh";
const SUPPLIER_JWT_STORAGE = "supplier_jwt";

function tauriBackendBaseUrl(): string {
  const envUrl = (
    process.env.NEXT_PUBLIC_API_URL ||
    process.env.NEXT_PUBLIC_SUPPLIER_BACKEND_BASE_URL ||
    process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
    ""
  ).replace(/\/$/, "");
  if (envUrl) return envUrl;
  // 10.0.2.2 is the Android emulator host alias; desktop/macOS uses localhost.
  if (typeof navigator !== "undefined" && /android/i.test(navigator.userAgent)) {
    return "http://10.0.2.2:8180";
  }
  return "http://localhost:8180";
}


function usesApiProxy(base: string): boolean {
  return base === "/api" || base.endsWith("/api");
}

function supplierBootstrapUrl(): string {
  if (typeof window !== "undefined") {
    if (isTauri()) {
      const { hostname, port, origin } = window.location;
      // Tauri dev loads from Next.js (:3000). Proxy /api → :8180 avoids WKWebView direct fetch issues.
      if ((hostname === "localhost" || hostname === "127.0.0.1") && port === "3000") {
        return `${origin}/api`;
      }
      return tauriBackendBaseUrl();
    }
    return "/api";
  }
  return "http://localhost:8180";
}

/** GS-C5: pin to JWT home_cell API URL. Local /api and localhost stay on bootstrap. */
export function supplierApiBaseUrl(): string {
  const token = typeof window !== "undefined" ? getSupplierToken() : "";
  return pinApiBaseUrl({
    bootstrap: supplierBootstrapUrl(),
    homeCell: token ? homeCellFromJwt(token) : "",
    sessionApiUrl: readCachedAuthSession()?.api_url,
  });
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
    const secureFlag = typeof location !== "undefined" && location.protocol === "https:" ? "; Secure" : "";
    document.cookie = `${SUPPLIER_JWT_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=86400; SameSite=Strict${secureFlag}`;
    if (refreshToken) {
      document.cookie = `${SUPPLIER_REFRESH_COOKIE}=${encodeURIComponent(refreshToken)}; path=/; max-age=604800; SameSite=Strict${secureFlag}`;
    }
  }
  if (isTauri()) {
    void storeToken(token);
  }
}

export function clearSession() {
  if (typeof document !== "undefined") {
    const secureFlag = typeof location !== "undefined" && location.protocol === "https:" ? "; Secure" : "";
    document.cookie = `${SUPPLIER_JWT_COOKIE}=; Max-Age=0; path=/${secureFlag}`;
    document.cookie = `${SUPPLIER_REFRESH_COOKIE}=; Max-Age=0; path=/${secureFlag}`;
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
  return readTokenFromCookie();
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
      credentials: usesApiProxy(base) ? "include" : undefined,
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
  const hdrs = new Headers(init?.headers);
  const isFormData = typeof FormData !== "undefined" && init?.body instanceof FormData;
  if (!hdrs.has("Content-Type") && !isFormData) {
    hdrs.set("Content-Type", "application/json");
  }
  const method = (init?.method || "GET").toUpperCase();
  if (["POST", "PUT", "PATCH", "DELETE"].includes(method)) {
    if (!hdrs.has("Idempotency-Key") && !hdrs.has("X-Idempotency-Key")) {
      hdrs.set(
        "Idempotency-Key",
        typeof crypto !== "undefined" && crypto.randomUUID
          ? crypto.randomUUID()
          : Math.random().toString(36).substring(2),
      );
    }
  }
  if (token && !hdrs.has("Authorization")) {
    hdrs.set("Authorization", `Bearer ${token}`);
  }
  const headers = Object.fromEntries(hdrs.entries());
  const controller = new AbortController();
  const externalSignal = init?.signal;
  const onExternalAbort = () => controller.abort();
  if (externalSignal) {
    if (externalSignal.aborted) {
      controller.abort();
    } else {
      externalSignal.addEventListener("abort", onExternalAbort, { once: true });
    }
  }
  const timeoutMs = /\/(?:migrations|imports)\//.test(url) ? 300_000 : 30_000;
  const timeout = window.setTimeout(() => controller.abort(), timeoutMs);
  let res: Response;
  try {
    res = await fetch(url, {
      ...init,
      headers,
      credentials: usesApiProxy(base) ? "include" : init?.credentials,
      signal: controller.signal,
    });
  } catch (err) {
    if (err instanceof DOMException && err.name === "AbortError") {
      throw new Error(`Request timed out contacting ${base}`);
    }
    throw new Error(`Cannot reach API at ${base}. Is the backend running on port 8180?`);
  } finally {
    window.clearTimeout(timeout);
    externalSignal?.removeEventListener("abort", onExternalAbort);
  }

  if (res.status === 401) {
    if (!token) {
      return res;
    }
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
        credentials: usesApiProxy(base) ? "include" : init?.credentials,
      });
    }
    clearSession();
    if (typeof window !== "undefined") {
      const path = window.location.pathname;
      const onPublic =
        path.startsWith("/auth/") ||
        path.startsWith("/setup/") ||
        path === "/";
      if (!onPublic) {
        window.location.href = "/auth/login";
      }
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
  if (raw.startsWith("/api/")) {
    return raw.slice("/api".length);
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

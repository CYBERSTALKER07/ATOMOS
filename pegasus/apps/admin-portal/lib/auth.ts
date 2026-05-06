import { useState, useEffect } from "react";
import {
  isTauri,
  getStoredToken,
  storeToken,
  clearStoredToken,
} from "./bridge";
import { getFirebaseIdToken, firebaseSignOut } from "./firebase";
import { OfflineManager } from "./api/offlineQueue";
import {
  createTranslator,
  detectBrowserLocale,
  translateProblemDetail,
} from "@pegasus/i18n";
import type { ProblemDetail } from "@pegasus/types";
import { isProblemDetail } from "@pegasus/types";

const API = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";
const getTranslator = () => createTranslator(detectBrowserLocale());

/** Methods that mutate server state — eligible for offline queueing */
const MUTABLE_METHODS = new Set(["POST", "PUT", "PATCH", "DELETE"]);

/**
 * Global RequestGate blocks outbound requests during cooldown/jail
 * and applies backpressure to non-critical polling or reads.
 */
class RequestGateManager {
  private jailUntil: number = 0;

  constructor() {
    if (typeof window !== "undefined") {
      window.addEventListener("cooldown", (ev: Event) => {
        const detail = (ev as CustomEvent).detail;
        if (detail?.jailUntil) {
          this.jailUntil = Math.max(this.jailUntil, detail.jailUntil);
        }
      });
    }
  }

  async checkGate(
    isMutable: boolean,
    isCritical: boolean = false,
  ): Promise<void> {
    const now = Math.floor(Date.now() / 1000);
    if (this.jailUntil > now && !isCritical) {
      const msg = `[RequestGate] Outbound request dropped. Jailed for ${this.jailUntil - now}s`;
      console.warn(msg);
      // Throw a synthetic error to halt execution without hitting the network
      throw new Error(msg);
    }
  }
}
export const RequestGate = new RequestGateManager();

/**
 * Read the admin JWT from cookies. Returns empty string on server-side or when not logged in.
 */
export function readTokenFromCookie(): string {
  if (typeof document === "undefined") return "";
  // Admin pages use pegasus_admin_jwt; supplier pages use pegasus_supplier_jwt
  const adminMatch = document.cookie.match(/(?:^|; )pegasus_admin_jwt=([^;]*)/);
  if (adminMatch) return decodeURIComponent(adminMatch[1]);
  const supplierMatch = document.cookie.match(
    /(?:^|; )pegasus_supplier_jwt=([^;]*)/,
  );
  if (supplierMatch) return decodeURIComponent(supplierMatch[1]);
  return "";
}

/**
 * Hydration-safe token hook. Returns '' on the first (SSR-matching) render,
 * then reads the cookie after mount so server and client HTML always agree.
 */
export function useToken(): string {
  const [token, setToken] = useState("");
  useEffect(() => {
    const syncToken = () => {
      setToken(readTokenFromCookie());
    };

    syncToken();

    const handleVisibilityChange = () => {
      if (document.visibilityState === "visible") {
        syncToken();
      }
    };

    window.addEventListener("focus", syncToken);
    document.addEventListener("visibilitychange", handleVisibilityChange);

    const interval = window.setInterval(syncToken, 30_000);

    return () => {
      window.removeEventListener("focus", syncToken);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
      window.clearInterval(interval);
    };
  }, []);
  return token;
}

/**
 * Get a valid admin token. On desktop, checks OS keyring first.
 * Then reads from cookie. In dev mode only, falls back to debug mint.
 * Throws if no token is available.
 */
export async function getAdminToken(): Promise<string> {
  const t = getTranslator();
  const cookie = readTokenFromCookie();
  if (cookie) return cookie;

  // Desktop: try OS keyring first
  if (isTauri()) {
    try {
      const stored = await getStoredToken();
      if (stored) return stored;
    } catch {
      /* fall through to cookie */
    }
  }

  // Dev-only fallback — this endpoint is disabled in production
  if (process.env.NODE_ENV === "development") {
    const res = await fetch(`${API}/debug/mint-token?role=ADMIN`);
    if (res.ok) return (await res.text()).trim();
  }

  throw new Error(t("auth.error.no_auth_token"));
}

/**
 * Centralized fetch wrapper with automatic 401 handling.
 * On 401, attempts a silent token refresh before redirecting to login.
 */
let refreshInFlight: Promise<string | null> | null = null;

async function tryRefreshToken(): Promise<string | null> {
  const oldToken = readTokenFromCookie();
  if (!oldToken) return null;
  try {
    const res = await fetch(`${API}/v1/auth/refresh`, {
      method: "POST",
      headers: { Authorization: `Bearer ${oldToken}` },
    });
    if (!res.ok) return null;
    const data = await res.json();
    if (data.token) {
      document.cookie = `pegasus_admin_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      // Desktop: persist refreshed token in OS keyring
      if (isTauri()) {
        storeToken(data.token, data.refresh_token || "").catch(() => {});
      }
      return data.token;
    }
    return null;
  } catch {
    return null;
  }
}

type ApiFetchOptions = {
  queueMutableOnNetworkError?: boolean;
  isCritical?: boolean;
};

async function performApiFetch(
  path: string,
  init?: RequestInit,
  options?: ApiFetchOptions,
): Promise<Response> {
  // Prefer Firebase ID token if a Firebase session exists, fall back to legacy JWT
  let token = await getFirebaseIdToken();
  if (!token) {
    token = await getAdminToken();
  }
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    "Content-Type": "application/json",
    "X-Trace-Id": crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  const method = (init?.method || "GET").toUpperCase();
  const isMutable = MUTABLE_METHODS.has(method);

  // Centralized Idempotency for mutating methods
  if (isMutable && !headers['Idempotency-Key']) {
    headers['Idempotency-Key'] = crypto.randomUUID();
  }

  // RequestGate pre-flight check
  await RequestGate.checkGate(isMutable, options?.isCritical);

  let res: Response;
  try {
    res = await fetch(`${API}${path}`, { ...init, headers });
  } catch (err) {
    // Network failure — queue mutable requests for offline replay
    if (
      (options?.queueMutableOnNetworkError ?? true) &&
      isMutable &&
      typeof window !== "undefined"
    ) {
      OfflineManager.enqueue({
        url: path,
        method,
        body: typeof init?.body === "string" ? init.body : null,
        headers,
      });
      // Return a synthetic "queued" response so callers don't crash
      return new Response(JSON.stringify({ queued: true }), {
        status: 202,
        headers: { "Content-Type": "application/json" },
      });
    }
    throw err;
  }

  // ── Backpressure detection ──
  // Backend priority shedder may attach X-Backpressure-Interval to signal
  // the client should slow down polling or mutations. The value is in seconds.
  const backpressureSec = res.headers.get("X-Backpressure-Interval");
  if (backpressureSec && typeof window !== "undefined") {
    window.dispatchEvent(
      new CustomEvent("backpressure", {
        detail: parseInt(backpressureSec, 10) * 1000,
      }),
    );
  }

  // ── Cooldown / jail detection ──
  // X-Jail-Until carries the Unix epoch (seconds) when the rate-limit jail
  // expires for this actor+priority. Surfaces listen on the `cooldown` event
  // to render a precise countdown UI rather than a generic error toast.
  const jailUntil = res.headers.get("X-Jail-Until");
  if (res.status === 429 && jailUntil && typeof window !== "undefined") {
    const until = parseInt(jailUntil, 10);
    if (!Number.isNaN(until)) {
      window.dispatchEvent(
        new CustomEvent("cooldown", {
          detail: {
            jailUntil: until,
            priority: res.headers.get("X-Priority") || "UNKNOWN",
            path,
          },
        }),
      );
    }
  }

  // ── RFC 7807 Problem Detail detection ──
  // When the backend returns application/problem+json, parse the structured
  // error and attach it to the response for callers to inspect.
  const contentType = res.headers.get("Content-Type") || "";
  if (contentType.includes("application/problem+json") && !res.ok) {
    const cloned = res.clone();
    try {
      const body = await cloned.json();
      if (isProblemDetail(body)) {
        (res as Response & { problem?: ProblemDetail }).problem = body;
        const problemMessage = translateProblemDetail(
          body,
          detectBrowserLocale(),
        );
        console.error(
          `[API] ${body.status} ${body.code || body.type} trace=${body.trace_id} detail=${problemMessage}`,
        );
      }
    } catch {
      /* body parse failed — fall through to existing error handling */
    }
  }

  if (res.status === 401) {
    // Deduplicate concurrent refresh attempts
    if (!refreshInFlight) {
      refreshInFlight = tryRefreshToken().finally(() => {
        refreshInFlight = null;
      });
    }
    const newToken = await refreshInFlight;
    if (newToken) {
      // Retry the original request with the fresh token
      const retryHeaders: Record<string, string> = {
        ...headers,
        Authorization: `Bearer ${newToken}`,
      };
      return fetch(`${API}${path}`, { ...init, headers: retryHeaders });
    }
    // Refresh failed — clear cookies (and keyring on desktop), redirect
    document.cookie = "pegasus_admin_jwt=; Max-Age=0; path=/";
    document.cookie = "pegasus_supplier_jwt=; Max-Age=0; path=/";
    firebaseSignOut().catch(() => {});
    if (isTauri()) {
      clearStoredToken().catch(() => {});
    }
    window.location.href = "/auth/login";
    throw new Error(getTranslator()("auth.error.session_expired"));
  }

  return res;
}

export async function apiFetch(
  path: string,
  init?: RequestInit,
): Promise<Response> {
  return performApiFetch(path, init, { queueMutableOnNetworkError: true });
}

export async function apiFetchNoQueue(
  path: string,
  init?: RequestInit,
): Promise<Response> {
  return performApiFetch(path, init, { queueMutableOnNetworkError: false });
}

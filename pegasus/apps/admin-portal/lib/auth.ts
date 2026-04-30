import { useState, useEffect } from 'react';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { getFirebaseIdToken, firebaseSignOut } from './firebase';
import { OfflineManager } from './api/offlineQueue';
import type { ProblemDetail } from '@lab/types';
import { isProblemDetail } from '@lab/types';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/** Methods that mutate server state — eligible for offline queueing */
const MUTABLE_METHODS = new Set(['POST', 'PUT', 'PATCH', 'DELETE']);

/**
 * Read the admin JWT from cookies. Returns empty string on server-side or when not logged in.
 */
export function readTokenFromCookie(): string {
  if (typeof document === 'undefined') return '';
  // Admin pages use admin_jwt; supplier pages use supplier_jwt
  const adminMatch = document.cookie.match(/(?:^|; )admin_jwt=([^;]*)/);
  if (adminMatch) return decodeURIComponent(adminMatch[1]);
  const supplierMatch = document.cookie.match(/(?:^|; )supplier_jwt=([^;]*)/);
  if (supplierMatch) return decodeURIComponent(supplierMatch[1]);
  return '';
}

/**
 * Hydration-safe token hook. Returns '' on the first (SSR-matching) render,
 * then reads the cookie after mount so server and client HTML always agree.
 */
export function useToken(): string {
  const [token, setToken] = useState('');
  useEffect(() => {
    setToken(readTokenFromCookie());
  }, []);
  return token;
}

/**
 * Get a valid admin token. On desktop, checks OS keyring first.
 * Then reads from cookie. In dev mode only, falls back to debug mint.
 * Throws if no token is available.
 */
export async function getAdminToken(): Promise<string> {
  // Desktop: try OS keyring first
  if (isTauri()) {
    try {
      const stored = await getStoredToken();
      if (stored) return stored;
    } catch { /* fall through to cookie */ }
  }

  const cookie = readTokenFromCookie();
  if (cookie) return cookie;

  // Dev-only fallback — this endpoint is disabled in production
  if (process.env.NODE_ENV === 'development') {
    const res = await fetch(`${API}/debug/mint-token?role=ADMIN`);
    if (res.ok) return (await res.text()).trim();
  }

  throw new Error('No auth token available. Please log in.');
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
      method: 'POST',
      headers: { Authorization: `Bearer ${oldToken}` },
    });
    if (!res.ok) return null;
    const data = await res.json();
    if (data.token) {
      document.cookie = `admin_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      // Desktop: persist refreshed token in OS keyring
      if (isTauri()) {
        storeToken(data.token, data.refresh_token || '').catch(() => {});
      }
      return data.token;
    }
    return null;
  } catch {
    return null;
  }
}

export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  // Prefer Firebase ID token if a Firebase session exists, fall back to legacy JWT
  let token = await getFirebaseIdToken();
  if (!token) {
    token = await getAdminToken();
  }
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Trace-Id': crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  let res: Response;
  try {
    res = await fetch(`${API}${path}`, { ...init, headers });
  } catch (err) {
    // Network failure — queue mutable requests for offline replay
    const method = (init?.method || 'GET').toUpperCase();
    if (MUTABLE_METHODS.has(method) && typeof window !== 'undefined') {
      OfflineManager.enqueue({
        url: path,
        method,
        body: typeof init?.body === 'string' ? init.body : null,
        headers,
      });
      // Return a synthetic "queued" response so callers don't crash
      return new Response(JSON.stringify({ queued: true }), {
        status: 202,
        headers: { 'Content-Type': 'application/json' },
      });
    }
    throw err;
  }

  // ── Backpressure detection ──
  // Backend priority shedder may attach X-Backpressure-Interval to signal
  // the client should slow down polling or mutations.
  const backpressureMs = res.headers.get('X-Backpressure-Interval');
  if (backpressureMs && typeof window !== 'undefined') {
    window.dispatchEvent(
      new CustomEvent('backpressure', { detail: parseInt(backpressureMs, 10) }),
    );
  }

  // ── RFC 7807 Problem Detail detection ──
  // When the backend returns application/problem+json, parse the structured
  // error and attach it to the response for callers to inspect.
  const contentType = res.headers.get('Content-Type') || '';
  if (contentType.includes('application/problem+json') && !res.ok) {
    const cloned = res.clone();
    try {
      const body = await cloned.json();
      if (isProblemDetail(body)) {
        (res as Response & { problem?: ProblemDetail }).problem = body;
        console.error(
          `[API] ${body.status} ${body.code || body.type} trace=${body.trace_id} detail=${body.detail}`,
        );
      }
    } catch { /* body parse failed — fall through to existing error handling */ }
  }

  if (res.status === 401) {
    // Deduplicate concurrent refresh attempts
    if (!refreshInFlight) {
      refreshInFlight = tryRefreshToken().finally(() => { refreshInFlight = null; });
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
    document.cookie = 'admin_jwt=; Max-Age=0; path=/';
    document.cookie = 'supplier_jwt=; Max-Age=0; path=/';
    firebaseSignOut().catch(() => {});
    if (isTauri()) {
      clearStoredToken().catch(() => {});
    }
    window.location.href = '/auth/login';
    throw new Error('Session expired');
  }

  return res;
}

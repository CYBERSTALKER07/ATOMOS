// lib/auth.ts
import { useState, useEffect } from 'react';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { createTranslator, detectBrowserLocale, translateProblemDetail } from '@pegasusx/i18n';
import type { ProblemDetail } from '@pegasusx/types';
import { isProblemDetail } from '@pegasusx/types';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8180';
const getTranslator = () => createTranslator(detectBrowserLocale());

/**
 * Read the retailer JWT from cookies only. Returns empty string on server-side or when not logged in.
 * Note: localStorage is NOT used for auth tokens (XSS risk). Desktop uses OS keyring via Tauri.
 */
export function readToken(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(/(?:^|; )pegasus_retailer_jwt=([^;]*)/);
  if (match) return decodeURIComponent(match[1]);
  return '';
}

/**
 * Hydration-safe token hook. Returns '' on the first (SSR-matching) render,
 * then reads the cookie after mount so server and client HTML always agree.
 */
export function useToken(): string {
  const [token, setToken] = useState('');
  useEffect(() => {
    setToken(readToken());
  }, []);
  return token;
}

/**
 * Get a valid retailer token. On desktop, checks OS keyring first.
 * Then reads from cookie / local storage.
 * Throws if no token is available.
 */
export async function getRetailerToken(): Promise<string> {
  const t = getTranslator();
  // Desktop: try OS keyring first
  if (isTauri()) {
    try {
      const stored = await getStoredToken();
      if (stored) return stored;
    } catch { /* fall through */ }
  }

  const token = readToken();
  if (token) return token;

  throw new Error(t('auth.error.no_auth_token'));
}

let refreshInFlight: Promise<string | null> | null = null;

function readRefreshFromCookie(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(/(?:^|; )pegasus_retailer_refresh=([^;]*)/);
  if (match) return decodeURIComponent(match[1]);
  return '';
}

async function tryRefreshToken(): Promise<string | null> {
  const refresh = readRefreshFromCookie();
  if (!refresh) return null;
  try {
    const res = await fetch(`${API}/v1/auth/retailer/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    if (!res.ok) return null;
    const data = (await res.json()) as { token?: string; refresh_token?: string };
    if (!data.token) return null;
    document.cookie = `pegasus_retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
    if (data.refresh_token) {
      document.cookie = `pegasus_retailer_refresh=${encodeURIComponent(data.refresh_token)}; path=/; max-age=604800; SameSite=Lax`;
    }
    if (isTauri()) {
      storeToken(data.token, data.refresh_token || refresh).catch(() => {});
    }
    return data.token;
  } catch {
    return null;
  }
}

export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const token = await getRetailerToken();
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Trace-Id': crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  const res = await fetch(`${API}${path}`, { ...init, headers });

  // ── RFC 7807 Problem Detail detection ──
  const contentType = res.headers.get('Content-Type') || '';
  if (contentType.includes('application/problem+json') && !res.ok) {
    const cloned = res.clone();
    try {
      const body = await cloned.json();
      if (isProblemDetail(body)) {
        (res as Response & { problem?: ProblemDetail }).problem = body;
        const problemMessage = translateProblemDetail(body, detectBrowserLocale());
        console.error(
          `[API] ${body.status} ${body.code || body.type} trace=${body.trace_id} detail=${problemMessage}`,
        );
      }
    } catch { /* body parse failed — fall through */ }
  }

  if (res.status === 401) {
    if (!refreshInFlight) {
      refreshInFlight = tryRefreshToken().finally(() => { refreshInFlight = null; });
    }
    const newToken = await refreshInFlight;
    if (newToken) {
      // Retry with fresh token
      const retryHeaders: Record<string, string> = {
        ...headers,
        Authorization: `Bearer ${newToken}`,
      };
      return fetch(`${API}${path}`, { ...init, headers: retryHeaders });
    }
    // Refresh failed
    document.cookie = 'pegasus_retailer_jwt=; Max-Age=0; path=/';
    if (isTauri() || typeof localStorage !== 'undefined') {
      clearStoredToken().catch(() => {});
    }
    window.location.href = '/';
    throw new Error(getTranslator()('auth.error.session_expired'));
  }

  return res;
}

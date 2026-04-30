// lib/auth.ts
import { useState, useEffect } from 'react';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import type { ProblemDetail } from '@pegasus/types';
import { isProblemDetail } from '@pegasus/types';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/**
 * Read the retailer JWT from cookies only. Returns empty string on server-side or when not logged in.
 * Note: localStorage is NOT used for auth tokens (XSS risk). Desktop uses OS keyring via Tauri.
 */
export function readToken(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(/(?:^|; )retailer_jwt=([^;]*)/);
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
  // Desktop: try OS keyring first
  if (isTauri()) {
    try {
      const stored = await getStoredToken();
      if (stored) return stored;
    } catch { /* fall through */ }
  }

  const token = readToken();
  if (token) return token;

  throw new Error('No auth token available. Please log in.');
}

let refreshInFlight: Promise<string | null> | null = null;

async function tryRefreshToken(): Promise<string | null> {
  const oldToken = readToken();
  if (!oldToken) return null;
  try {
    const res = await fetch(`${API}/v1/auth/refresh`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${oldToken}` },
    });
    if (!res.ok) return null;
    const data = await res.json();
    if (data.token) {
      document.cookie = `retailer_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      if (isTauri() || typeof localStorage !== 'undefined') {
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
        console.error(
          `[API] ${body.status} ${body.code || body.type} trace=${body.trace_id} detail=${body.detail}`,
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
    document.cookie = 'retailer_jwt=; Max-Age=0; path=/';
    if (isTauri() || typeof localStorage !== 'undefined') {
      clearStoredToken().catch(() => {});
    }
    window.location.href = '/';
    throw new Error('Session expired');
  }

  return res;
}
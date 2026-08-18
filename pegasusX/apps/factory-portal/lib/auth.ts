import { useState, useEffect } from 'react';
import { homeCellFromJwt, pinApiBaseUrl, readCachedAuthSession } from '@pegasusx/api-client';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { runFactorySessionReconcile } from './session-reconcile';

const BOOTSTRAP = (
  process.env.NEXT_PUBLIC_API_URL ||
  process.env.NEXT_PUBLIC_FACTORY_BACKEND_BASE_URL ||
  process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
  'http://localhost:8180'
).replace(/\/$/, '');

/** GS-C5: pin to JWT home_cell. Login (no token) stays on bootstrap. */
export function factoryApiBaseUrl(): string {
  return pinApiBaseUrl({
    bootstrap: BOOTSTRAP,
    homeCell: homeCellFromJwt(readTokenFromCookie()),
    sessionApiUrl: readCachedAuthSession()?.api_url,
  });
}

const FACTORY_JWT_COOKIE = 'pegasus_factory_jwt';
const FACTORY_REFRESH_COOKIE = 'pegasus_factory_refresh';

const FACTORY_LIVE_EVENT_TYPES = [
  'FACTORY_SUPPLY_REQUEST_UPDATE',
  'FACTORY_TRANSFER_UPDATE',
  'FACTORY_MANIFEST_UPDATE',
  'FACTORY_OUTBOX_FAILED',
] as const;

type FactoryLiveEventType = (typeof FACTORY_LIVE_EVENT_TYPES)[number];

export interface FactoryLiveEvent {
  type: FactoryLiveEventType;
  [key: string]: unknown;
}

export type FactorySocketStatus = 'idle' | 'connecting' | 'live' | 'reconnecting' | 'offline';

export function readTokenFromCookie(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(new RegExp(`(?:^|; )${FACTORY_JWT_COOKIE}=([^;]*)`));
  if (match) return decodeURIComponent(match[1]);
  return '';
}

function readRefreshFromCookie(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(new RegExp(`(?:^|; )${FACTORY_REFRESH_COOKIE}=([^;]*)`));
  if (match) return decodeURIComponent(match[1]);
  return '';
}

export function persistSession(token: string, refreshToken?: string) {
  if (typeof document !== 'undefined') {
    document.cookie = `${FACTORY_JWT_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=86400; SameSite=Lax`;
    if (refreshToken) {
      document.cookie = `${FACTORY_REFRESH_COOKIE}=${encodeURIComponent(refreshToken)}; path=/; max-age=604800; SameSite=Lax`;
    }
  }
  if (isTauri()) {
    void storeToken(token, refreshToken);
  }
}

export function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload) as Record<string, unknown>;
  } catch {
    return null;
  }
}

export function useToken(): string {
  const [token, setToken] = useState('');
  useEffect(() => {
    setToken(readTokenFromCookie());
  }, []);
  return token;
}

export async function refreshFactorySession(): Promise<{ ok: boolean; isConfigured?: boolean }> {
  const refresh = readRefreshFromCookie();
  if (!refresh) return { ok: false };
  try {
    const res = await fetch(`${factoryApiBaseUrl()}/v1/auth/factory/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
    });
    if (!res.ok) return { ok: false };
    const data = (await res.json()) as { token?: string; refresh_token?: string; is_configured?: boolean };
    if (!data.token) return { ok: false };
    persistSession(data.token, data.refresh_token);
    return { ok: true, isConfigured: data.is_configured === true };
  } catch {
    return { ok: false };
  }
}

export async function getFactoryToken(): Promise<string> {
  if (isTauri()) {
    try {
      const stored = await getStoredToken();
      if (stored) return stored;
    } catch { /* fall through to cookie */ }
  }

  const cookie = readTokenFromCookie();
  if (cookie) return cookie;

  throw new Error('No auth token available. Please log in.');
}

/** Session JWT for HTTP Bearer. Firebase ID is OTP `id_token` body only — never Authorization. */
export function httpAuthorizationToken(sessionJwt: string, firebaseIdToken?: string): string {
  const jwt = sessionJwt.trim();
  if (jwt) return jwt;
  if (firebaseIdToken) {
    return '';
  }
  return '';
}

let refreshInFlight: Promise<string | null> | null = null;

async function tryRefreshToken(): Promise<string | null> {
  const refresh = readRefreshFromCookie();
  if (!refresh) return null;
  try {
    const res = await fetch(`${factoryApiBaseUrl()}/v1/auth/factory/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refresh }),
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

export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const token = httpAuthorizationToken(await getFactoryToken());
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Trace-Id': crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  const res = await fetch(`${factoryApiBaseUrl()}${path}`, { ...init, headers });

  if (res.status === 401) {
    if (!refreshInFlight) {
      refreshInFlight = tryRefreshToken().finally(() => { refreshInFlight = null; });
    }
    const newToken = await refreshInFlight;
    if (newToken) {
      const retryHeaders: Record<string, string> = {
        ...headers,
        Authorization: `Bearer ${newToken}`,
      };
      return fetch(`${factoryApiBaseUrl()}${path}`, { ...init, headers: retryHeaders });
    }
    document.cookie = `${FACTORY_JWT_COOKIE}=; Max-Age=0; path=/`;
    document.cookie = `${FACTORY_REFRESH_COOKIE}=; Max-Age=0; path=/`;
    import('./firebase').then(({ firebaseSignOut }) => firebaseSignOut()).catch(() => {});
    if (isTauri()) {
      clearStoredToken().catch(() => {});
    }
    window.location.href = '/auth/login';
    throw new Error('Session expired');
  }

  return res;
}

function toWSBaseUrl(baseUrl: string): string {
  return baseUrl.replace(/^http/, 'ws');
}

async function readFactorySocketToken(): Promise<string> {
  const res = await apiFetch('/v1/factory/ws-session', { method: 'GET', cache: 'no-store' });
  const payload = (await res.json().catch(() => null)) as { token?: string } | null;
  if (!res.ok || typeof payload?.token !== 'string' || !payload.token) {
    throw new Error('Failed to mint factory WebSocket session');
  }
  return payload.token;
}

function isFactoryEventType(value: string): value is FactoryLiveEventType {
  return (FACTORY_LIVE_EVENT_TYPES as readonly string[]).includes(value);
}

export function parseFactoryLiveEvent(rawPayload: string): FactoryLiveEvent | null {
  try {
    const parsed = JSON.parse(rawPayload) as { type?: unknown };
    if (!parsed || typeof parsed !== 'object') {
      return null;
    }
    if (typeof parsed.type !== 'string' || !isFactoryEventType(parsed.type)) {
      return null;
    }
    return parsed as FactoryLiveEvent;
  } catch {
    return null;
  }
}

/** Subscribe to the factory websocket and auto-reconnect on transient drops. */
export function subscribeFactoryWS(options: {
  onMessage: (payload: string) => void;
  onStatusChange?: (status: FactorySocketStatus) => void;
}): () => void {
  let socket: WebSocket | null = null;
  let reconnectTimer: number | null = null;
  let reconnectAttempt = 0;
  let hasConnectedOnce = false;
  let disposed = false;

  const clearReconnect = () => {
    if (reconnectTimer !== null) {
      window.clearTimeout(reconnectTimer);
      reconnectTimer = null;
    }
  };

  const openSocket = async (isReconnect: boolean) => {
    if (disposed) return;
    if (typeof navigator !== 'undefined' && !navigator.onLine) {
      options.onStatusChange?.('offline');
      return;
    }

    options.onStatusChange?.(isReconnect ? 'reconnecting' : 'connecting');

    let token = '';
    try {
      token = await readFactorySocketToken();
    } catch {
      token = '';
    }
    if (disposed) return;
    if (!token) {
      reconnectAttempt += 1;
      options.onStatusChange?.('reconnecting');
      reconnectTimer = window.setTimeout(() => {
        void openSocket(true);
      }, Math.min(30_000, 1_000 * 2 ** (reconnectAttempt - 1)));
      return;
    }

    const wsBase = toWSBaseUrl(factoryApiBaseUrl());
    socket = new WebSocket(`${wsBase}/v1/ws?token=${encodeURIComponent(token)}`);

    socket.onopen = () => {
      const wasReconnect = hasConnectedOnce;
      hasConnectedOnce = true;
      reconnectAttempt = 0;
      options.onStatusChange?.('live');
      if (wasReconnect) {
        void runFactorySessionReconcile();
      }
    };

    socket.onmessage = event => {
      options.onMessage(String(event.data));
    };

    socket.onerror = () => {
      socket?.close();
    };

    socket.onclose = () => {
      socket = null;
      if (disposed) {
        options.onStatusChange?.('idle');
        return;
      }
      if (typeof navigator !== 'undefined' && !navigator.onLine) {
        options.onStatusChange?.('offline');
        return;
      }
      reconnectAttempt += 1;
      options.onStatusChange?.('reconnecting');
      reconnectTimer = window.setTimeout(() => {
        void openSocket(true);
      }, Math.min(30_000, 1_000 * 2 ** (reconnectAttempt - 1)));
    };
  };

  const handleOnline = () => {
    if (disposed || socket) return;
    void openSocket(reconnectAttempt > 0);
  };

  const handleOffline = () => {
    clearReconnect();
    options.onStatusChange?.('offline');
    socket?.close();
    socket = null;
  };

  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);
  void openSocket(false);

  return () => {
    disposed = true;
    clearReconnect();
    window.removeEventListener('online', handleOnline);
    window.removeEventListener('offline', handleOffline);
    socket?.close();
    socket = null;
    options.onStatusChange?.('idle');
  };
}

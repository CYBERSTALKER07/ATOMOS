import { useState, useEffect } from 'react';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { getFirebaseIdToken, firebaseSignOut } from './firebase';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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
  const match = document.cookie.match(/(?:^|; )pegasus_factory_jwt=([^;]*)/);
  if (match) return decodeURIComponent(match[1]);
  return '';
}

export function useToken(): string {
  const [token, setToken] = useState('');
  useEffect(() => {
    setToken(readTokenFromCookie());
  }, []);
  return token;
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

  if (process.env.NODE_ENV === 'development') {
    const res = await fetch(`${API}/debug/mint-token?role=FACTORY`);
    if (res.ok) return (await res.text()).trim();
  }

  throw new Error('No auth token available. Please log in.');
}

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
      document.cookie = `pegasus_factory_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
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
  let token = await getFirebaseIdToken();
  if (!token) {
    token = await getFactoryToken();
  }
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Trace-Id': crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  const res = await fetch(`${API}${path}`, { ...init, headers });

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
      return fetch(`${API}${path}`, { ...init, headers: retryHeaders });
    }
    document.cookie = 'pegasus_factory_jwt=; Max-Age=0; path=/';
    firebaseSignOut().catch(() => {});
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
  try {
    return await getFactoryToken();
  } catch {
    return '';
  }
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

    const token = await readFactorySocketToken();
    if (disposed) return;

    const wsBase = toWSBaseUrl(API);
    socket = new WebSocket(`${wsBase}/v1/ws/factory?token=${encodeURIComponent(token)}`);

    socket.onopen = () => {
      reconnectAttempt = 0;
      options.onStatusChange?.('live');
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

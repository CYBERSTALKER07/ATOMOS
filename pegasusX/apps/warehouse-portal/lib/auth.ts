import { useEffect, useState } from 'react';
import { clearStoredToken, getStoredToken, isTauri, storeToken } from '@/lib/bridge';

const API = (
  process.env.NEXT_PUBLIC_API_URL ||
  process.env.NEXT_PUBLIC_WAREHOUSE_BACKEND_BASE_URL ||
  process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
  'http://localhost:8180'
).replace(/\/$/, '');

const WAREHOUSE_JWT_COOKIE = 'pegasus_warehouse_jwt';
const WAREHOUSE_REFRESH_COOKIE = 'pegasus_warehouse_refresh';

export function readTokenFromCookie(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(new RegExp(`(?:^|; )${WAREHOUSE_JWT_COOKIE}=([^;]*)`));
  if (match) return decodeURIComponent(match[1]);
  return '';
}

function readRefreshFromCookie(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(new RegExp(`(?:^|; )${WAREHOUSE_REFRESH_COOKIE}=([^;]*)`));
  if (match) return decodeURIComponent(match[1]);
  return '';
}

export function persistSession(token: string, refreshToken?: string) {
  if (typeof document !== 'undefined') {
    document.cookie = `${WAREHOUSE_JWT_COOKIE}=${encodeURIComponent(token)}; path=/; max-age=86400; SameSite=Lax`;
    if (refreshToken) {
      document.cookie = `${WAREHOUSE_REFRESH_COOKIE}=${encodeURIComponent(refreshToken)}; path=/; max-age=604800; SameSite=Lax`;
    }
  }
  if (isTauri()) {
    void storeToken(token, refreshToken);
  }
}

export function clearSession() {
  if (typeof document !== 'undefined') {
    document.cookie = `${WAREHOUSE_JWT_COOKIE}=; Max-Age=0; path=/`;
    document.cookie = `${WAREHOUSE_REFRESH_COOKIE}=; Max-Age=0; path=/`;
  }
  if (isTauri()) {
    void clearStoredToken();
  }
}

export function useToken(): string {
  const [token, setToken] = useState('');
  useEffect(() => {
    void (async () => {
      const desktop = await getStoredToken();
      setToken(desktop || readTokenFromCookie());
    })();
  }, []);
  return token;
}

export function decodeJwtPayload(token: string): Record<string, unknown> | null {
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return null;
    const payload = atob(parts[1].replace(/-/g, '+').replace(/_/g, '/'));
    return JSON.parse(payload);
  } catch {
    return null;
  }
}

export async function getWarehouseToken(): Promise<string> {
  const desktop = await getStoredToken();
  if (desktop) return desktop;

  const cookie = readTokenFromCookie();
  if (cookie) return cookie;

  throw new Error('No auth token available. Please log in.');
}

let refreshInFlight: Promise<string | null> | null = null;

async function tryRefreshToken(): Promise<string | null> {
  const refresh = readRefreshFromCookie();
  if (!refresh) return null;
  try {
    const res = await fetch(`${API}/v1/auth/warehouse/refresh`, {
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
  const token = await getWarehouseToken();
  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
    'X-Trace-Id': crypto.randomUUID(),
    ...(init?.headers as Record<string, string>),
  };

  const res = await fetch(`${API}${path}`, { ...init, headers });

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
      return fetch(`${API}${path}`, { ...init, headers: retryHeaders });
    }
    clearSession();
    if (typeof window !== 'undefined') {
      window.location.href = '/auth/login';
    }
    throw new Error('Session expired');
  }

  return res;
}

/** Open a WebSocket to the unified pegasusX hub (warehouse rooms). */
export async function connectWarehouseWS(): Promise<WebSocket> {
  const wsBase = API.replace(/^http/, 'ws');
  const token = await getWarehouseToken();
  return new WebSocket(`${wsBase}/v1/ws?token=${encodeURIComponent(token)}`);
}

export type WarehouseSocketStatus = 'idle' | 'connecting' | 'live' | 'reconnecting' | 'offline';

export function subscribeWarehouseWS(options: {
  onMessage: (payload: string) => void;
  onStatusChange?: (status: WarehouseSocketStatus) => void;
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

  const openSocket = (isReconnect: boolean) => {
    if (disposed) return;
    if (typeof navigator !== 'undefined' && !navigator.onLine) {
      options.onStatusChange?.('offline');
      return;
    }

    options.onStatusChange?.(isReconnect ? 'reconnecting' : 'connecting');
    void connectWarehouseWS().then((ws) => {
      if (disposed) {
        ws.close();
        return;
      }
      socket = ws;

      socket.onopen = () => {
        reconnectAttempt = 0;
        options.onStatusChange?.('live');
      };

      socket.onmessage = (event) => {
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
          openSocket(true);
        }, Math.min(30_000, 1_000 * 2 ** (reconnectAttempt - 1)));
      };
    });
  };

  const handleOnline = () => {
    if (disposed || socket) return;
    openSocket(reconnectAttempt > 0);
  };

  const handleOffline = () => {
    clearReconnect();
    options.onStatusChange?.('offline');
    socket?.close();
    socket = null;
  };

  window.addEventListener('online', handleOnline);
  window.addEventListener('offline', handleOffline);
  openSocket(false);

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

export { API as warehouseApiBaseUrl };

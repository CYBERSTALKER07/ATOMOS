import { useState, useEffect } from 'react';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

export function readTokenFromCookie(): string {
  if (typeof document === 'undefined') return '';
  const match = document.cookie.match(/(?:^|; )pegasus_warehouse_jwt=([^;]*)/);
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
  const cookie = readTokenFromCookie();
  if (cookie) return cookie;

  if (process.env.NODE_ENV === 'development') {
    const res = await fetch(`${API}/debug/mint-token?role=WAREHOUSE`);
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
      document.cookie = `pegasus_warehouse_jwt=${encodeURIComponent(data.token)}; path=/; max-age=86400; SameSite=Lax`;
      return data.token;
    }
    return null;
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
    document.cookie = 'pegasus_warehouse_jwt=; Max-Age=0; path=/';
    window.location.href = '/auth/login';
    throw new Error('Session expired');
  }

  return res;
}

/** Open a WebSocket to the warehouse hub */
export function connectWarehouseWS(): WebSocket {
  const wsBase = (API.replace(/^http/, 'ws'));
  const token = readTokenFromCookie();
  return new WebSocket(`${wsBase}/ws/warehouse?token=${encodeURIComponent(token)}`);
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
    socket = connectWarehouseWS();

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
        openSocket(true);
      }, Math.min(30_000, 1_000 * 2 ** (reconnectAttempt - 1)));
    };
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

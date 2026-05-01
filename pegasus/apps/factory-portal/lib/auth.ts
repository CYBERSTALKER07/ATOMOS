import { useState, useEffect } from 'react';
import { isTauri, getStoredToken, storeToken, clearStoredToken } from './bridge';
import { getFirebaseIdToken, firebaseSignOut } from './firebase';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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

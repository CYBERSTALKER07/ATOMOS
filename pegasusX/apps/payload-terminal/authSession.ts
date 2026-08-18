import * as SecureStore from 'expo-secure-store';
import { payloadApiBaseUrl } from './marketPack';

type SessionPayload = {
  token: string;
  refresh_token?: string;
  name?: string;
  supplier_id?: string;
  warehouse_id?: string;
  warehouse_name?: string;
};

let tokenRefreshListener: ((token: string) => void) | null = null;

export function setTokenRefreshListener(listener: ((token: string) => void) | null) {
  tokenRefreshListener = listener;
}

export async function savePayloaderSession(data: SessionPayload): Promise<void> {
  await SecureStore.setItemAsync('payloader_token', data.token);
  if (data.refresh_token) {
    await SecureStore.setItemAsync('payloader_refresh_token', data.refresh_token);
  }
  if (data.name) await SecureStore.setItemAsync('payloader_name', data.name);
  if (data.supplier_id) await SecureStore.setItemAsync('payloader_supplier_id', data.supplier_id);
  if (data.warehouse_id) await SecureStore.setItemAsync('payloader_warehouse_id', data.warehouse_id);
  if (data.warehouse_name) await SecureStore.setItemAsync('payloader_warehouse_name', data.warehouse_name);
}

export async function clearPayloaderSession(): Promise<void> {
  await SecureStore.deleteItemAsync('payloader_token');
  await SecureStore.deleteItemAsync('payloader_refresh_token');
  await SecureStore.deleteItemAsync('payloader_name');
  await SecureStore.deleteItemAsync('payloader_supplier_id');
  await SecureStore.deleteItemAsync('payloader_warehouse_id');
  await SecureStore.deleteItemAsync('payloader_warehouse_name');
}

async function refreshPayloaderSession(): Promise<string | null> {
  const refresh = await SecureStore.getItemAsync('payloader_refresh_token');
  if (!refresh) return null;
  const res = await fetch(`${payloadApiBaseUrl()}/v1/auth/payloader/refresh`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ refresh_token: refresh }),
  });
  if (!res.ok) return null;
  const data = await res.json();
  if (!data?.token) return null;
  await SecureStore.setItemAsync('payloader_token', data.token);
  if (data.refresh_token) {
    await SecureStore.setItemAsync('payloader_refresh_token', data.refresh_token);
  }
  tokenRefreshListener?.(data.token);
  return data.token as string;
}

export async function authFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const token = await SecureStore.getItemAsync('payloader_token');
  const base = payloadApiBaseUrl(token || undefined);
  const url = path.startsWith('http') ? path : `${base}${path.startsWith('/') ? path : `/${path}`}`;
  const headers = new Headers(init.headers);
  if (token) headers.set('Authorization', `Bearer ${token}`);
  let res = await fetch(url, { ...init, headers });
  if (res.status === 401 && !headers.get('X-Refresh-Attempted')) {
    const nextToken = await refreshPayloaderSession();
    if (nextToken) {
      headers.set('Authorization', `Bearer ${nextToken}`);
      headers.set('X-Refresh-Attempted', 'true');
      res = await fetch(url, { ...init, headers });
    }
  }
  return res;
}

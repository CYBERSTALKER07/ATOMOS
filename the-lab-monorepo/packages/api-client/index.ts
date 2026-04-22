/**
 * @file packages/api-client/index.ts
 * @description Typed API client for the Lab Industries backend.
 * Wraps fetch with JWT auth, refresh, and typed responses.
 *
 * Usage:
 *   import { createApiClient } from '@lab/api-client';
 *   const api = createApiClient({ baseUrl: 'http://localhost:8080', getToken: () => token });
 *   const orders = await api.get<Order[]>('/v1/supplier/orders');
 */

// ─── Client Configuration ───────────────────────────────────────────────────
export interface ApiClientConfig {
  baseUrl: string;
  getToken: () => Promise<string | null> | string | null;
  onUnauthorized?: () => void;
  refreshToken?: () => Promise<string | null>;
}

// ─── Typed Response ─────────────────────────────────────────────────────────
export interface ApiResponse<T> {
  data: T;
  status: number;
  ok: boolean;
}

export class ApiError extends Error {
  status: number;
  body: unknown;
  constructor(status: number, body: unknown) {
    super(`API Error ${status}`);
    this.name = 'ApiError';
    this.status = status;
    this.body = body;
  }
}

// ─── Client Factory ─────────────────────────────────────────────────────────
export function createApiClient(config: ApiClientConfig) {
  let refreshInFlight: Promise<string | null> | null = null;

  async function request<T>(
    method: string,
    path: string,
    body?: unknown,
    init?: RequestInit,
  ): Promise<ApiResponse<T>> {
    const token = await config.getToken();
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...(init?.headers as Record<string, string>),
    };

    const fetchInit: RequestInit = {
      method,
      headers,
      ...init,
      ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
    };

    let res = await fetch(`${config.baseUrl}${path}`, fetchInit);

    // Auto-refresh on 401
    if (res.status === 401 && config.refreshToken) {
      if (!refreshInFlight) {
        refreshInFlight = config.refreshToken().finally(() => {
          refreshInFlight = null;
        });
      }
      const newToken = await refreshInFlight;
      if (newToken) {
        headers.Authorization = `Bearer ${newToken}`;
        res = await fetch(`${config.baseUrl}${path}`, { ...fetchInit, headers });
      } else {
        config.onUnauthorized?.();
        throw new ApiError(401, { error: 'session_expired' });
      }
    }

    if (!res.ok) {
      let errorBody: unknown = null;
      try {
        errorBody = await res.json();
      } catch {
        errorBody = await res.text().catch(() => null);
      }
      throw new ApiError(res.status, errorBody);
    }

    const data = (await res.json()) as T;
    return { data, status: res.status, ok: true };
  }

  return {
    get<T>(path: string, init?: RequestInit): Promise<ApiResponse<T>> {
      return request<T>('GET', path, undefined, init);
    },
    post<T>(path: string, body?: unknown, init?: RequestInit): Promise<ApiResponse<T>> {
      return request<T>('POST', path, body, init);
    },
    patch<T>(path: string, body?: unknown, init?: RequestInit): Promise<ApiResponse<T>> {
      return request<T>('PATCH', path, body, init);
    },
    put<T>(path: string, body?: unknown, init?: RequestInit): Promise<ApiResponse<T>> {
      return request<T>('PUT', path, body, init);
    },
    delete<T>(path: string, init?: RequestInit): Promise<ApiResponse<T>> {
      return request<T>('DELETE', path, undefined, init);
    },
  };
}

// Re-export all types for convenience
export type * from '@lab/types';

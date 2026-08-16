import { reconcileSession } from '@pegasusx/api-client';
import { apiFetch, factoryApiBaseUrl, readTokenFromCookie } from './auth';
import { notifyFactorySessionReconciled } from './factory-reconnect';

/** Refetch server-authoritative factory snapshots after transport reconnect. */
export async function reconcileFactorySession(): Promise<void> {
  await reconcileSession({
    role: 'factory',
    baseUrl: factoryApiBaseUrl(),
    getAuthToken: () => readTokenFromCookie() || null,
    fetchImpl: (input, init) => {
      const href = typeof input === 'string' ? input : input instanceof URL ? input.href : input.url;
      const { pathname, search } = new URL(href);
      return apiFetch(`${pathname}${search}`, init);
    },
  });
}

export async function runFactorySessionReconcile(): Promise<void> {
  await reconcileFactorySession();
  notifyFactorySessionReconciled();
}

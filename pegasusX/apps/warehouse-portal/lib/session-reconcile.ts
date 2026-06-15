import { reconcileSession } from '@pegasusx/api-client';
import { apiFetch, readTokenFromCookie, warehouseApiBaseUrl } from './auth';
import { notifyWarehouseSessionReconciled } from './warehouse-reconnect';

/** Refetch server-authoritative warehouse snapshots after transport reconnect. */
export async function reconcileWarehouseSession(
  query?: Record<string, string>,
): Promise<void> {
  await reconcileSession({
    role: 'warehouse',
    baseUrl: warehouseApiBaseUrl(),
    getAuthToken: () => readTokenFromCookie() || null,
    fetchImpl: apiFetch,
    query,
  });
}

export async function runWarehouseSessionReconcile(
  query?: Record<string, string>,
): Promise<void> {
  await reconcileWarehouseSession(query);
  notifyWarehouseSessionReconciled();
}

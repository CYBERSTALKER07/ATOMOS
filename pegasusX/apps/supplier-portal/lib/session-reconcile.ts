import { reconcileSession } from '@pegasusx/api-core';
import { readTokenFromCookie, supplierApiBaseUrl, supplierSessionFetch } from "@/lib/auth";
import { getSupplierReconcileScope, notifySupplierSessionReconciled } from "@/lib/supplier-reconnect";

/** Refetch server-authoritative supplier snapshots after transport reconnect. */
export async function reconcileSupplierSession(
  query?: Record<string, string>,
): Promise<void> {
  await reconcileSession({
    role: "supplier",
    baseUrl: supplierApiBaseUrl(),
    getAuthToken: () => readTokenFromCookie() || null,
    fetchImpl: supplierSessionFetch,
    query,
  });
}

export async function runSupplierSessionReconcile(
  query?: Record<string, string>,
): Promise<void> {
  const scope = query ?? getSupplierReconcileScope();
  const hasScope = Object.keys(scope).some((key) => scope[key]);
  await reconcileSupplierSession(hasScope ? scope : undefined);
  notifySupplierSessionReconciled();
}

import { reconcileSession } from "@pegasusx/api-client";
import { readTokenFromCookie, supplierApiBaseUrl, supplierFetch } from "@/lib/auth";
import { getSupplierReconcileScope, notifySupplierSessionReconciled } from "@/lib/supplier-reconnect";

/** Refetch server-authoritative supplier snapshots after transport reconnect. */
export async function reconcileSupplierSession(
  query?: Record<string, string>,
): Promise<void> {
  await reconcileSession({
    role: "supplier",
    baseUrl: supplierApiBaseUrl(),
    getAuthToken: () => readTokenFromCookie() || null,
    fetchImpl: supplierFetch,
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

import { supplierScopeId } from "@/lib/supplier-scope";

/** Cache keys aligned with supplier session-reconcile API paths. */
export function supplierDashboardCacheKey(): string {
  return `/v1/supplier/dashboard:${supplierScopeId()}`;
}

export function supplierOrdersCacheKey(
  query: Record<string, string | number | undefined>,
): string {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value !== undefined && value !== null) {
      params.set(key, String(value));
    }
  }
  const qs = params.toString();
  return `/v1/supplier/orders:${supplierScopeId()}${qs ? `?${qs}` : ""}`;
}

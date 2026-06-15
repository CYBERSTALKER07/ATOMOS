/**
 * Session reconciliation: after transport reconnect, refetch server-authoritative
 * snapshots so UI resumes the correct in-flight operation (payment, dispatch, seal).
 * Clients should call these before retrying a queued mutation.
 */
export type SessionReconcileRole =
  | "retailer"
  | "driver"
  | "supplier"
  | "warehouse"
  | "factory"
  | "payload";

export type SessionReconcileEndpoint = {
  path: string;
  /** Human label for logging/metrics */
  label: string;
};

export const SESSION_RECONCILE_ENDPOINTS: Record<SessionReconcileRole, SessionReconcileEndpoint[]> = {
  retailer: [
    { path: "/v1/retailer/active-fulfillment", label: "active_fulfillment" },
    { path: "/v1/retailer/pending-payments", label: "pending_payments" },
    { path: "/v1/retailer/tracking", label: "tracking" },
  ],
  driver: [
    { path: "/v1/fleet/orders", label: "fleet_orders" },
    { path: "/v1/driver/manifest", label: "active_manifest" },
  ],
  supplier: [
    { path: "/v1/supplier/dispatch/preview", label: "dispatch_preview" },
    { path: "/v1/supplier/manifests", label: "manifests" },
  ],
  warehouse: [
    { path: "/v1/warehouse/ops/dispatch/preview", label: "dispatch_preview" },
    { path: "/v1/warehouse/dispatch-locks", label: "dispatch_locks" },
  ],
  factory: [
    { path: "/v1/factory/manifests", label: "manifests" },
  ],
  payload: [
    { path: "/v1/payloader/manifests", label: "manifests" },
    { path: "/v1/payloader/trucks", label: "trucks" },
  ],
};

export type SessionReconcileOptions = {
  role: SessionReconcileRole;
  fetchImpl?: typeof fetch;
  getAuthToken?: () => string | null;
  baseUrl: string;
  /** Optional query string appended to each path (e.g. warehouse_id=...) */
  query?: Record<string, string>;
};

export type SessionReconcileResult = {
  label: string;
  path: string;
  ok: boolean;
  status: number;
};

/** Parallel refetch of authoritative snapshots for a role after reconnect. */
export async function reconcileSession(
  options: SessionReconcileOptions,
): Promise<SessionReconcileResult[]> {
  const fetchImpl = options.fetchImpl ?? fetch;
  const endpoints = SESSION_RECONCILE_ENDPOINTS[options.role] ?? [];
  const base = options.baseUrl.endsWith("/") ? options.baseUrl : `${options.baseUrl}/`;

  return Promise.all(
    endpoints.map(async (endpoint) => {
      const url = new URL(endpoint.path.replace(/^\//, ""), base);
      if (options.query) {
        for (const [key, value] of Object.entries(options.query)) {
          if (value) url.searchParams.set(key, value);
        }
      }
      const headers = new Headers();
      const token = options.getAuthToken?.();
      if (token) headers.set("Authorization", `Bearer ${token}`);
      try {
        const res = await fetchImpl(url.toString(), { headers, credentials: "include" });
        return { label: endpoint.label, path: endpoint.path, ok: res.ok, status: res.status };
      } catch {
        return { label: endpoint.label, path: endpoint.path, ok: false, status: 0 };
      }
    }),
  );
}

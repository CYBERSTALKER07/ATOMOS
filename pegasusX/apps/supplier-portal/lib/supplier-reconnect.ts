export const SUPPLIER_SESSION_RECONCILED = "supplier:session-reconciled";

let reconnectEpoch = 0;
let reconcileScope: Record<string, string> = {};

export function getSupplierReconnectEpoch(): number {
  return reconnectEpoch;
}

/** Optional warehouse_id (or other) query params for dispatch-scoped session reconcile. */
export function setSupplierReconcileScope(query: Record<string, string>): void {
  reconcileScope = query;
}

export function getSupplierReconcileScope(): Record<string, string> {
  return reconcileScope;
}

/** Notify portal surfaces that transport reconnected and server snapshots were refetched. */
export function notifySupplierSessionReconciled(): void {
  if (typeof window === "undefined") return;
  reconnectEpoch += 1;
  window.dispatchEvent(
    new CustomEvent(SUPPLIER_SESSION_RECONCILED, { detail: { epoch: reconnectEpoch } }),
  );
}

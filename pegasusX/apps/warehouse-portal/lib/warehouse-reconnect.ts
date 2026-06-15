export const WAREHOUSE_SESSION_RECONCILED = 'warehouse:session-reconciled';

let reconnectEpoch = 0;

export function getWarehouseReconnectEpoch(): number {
  return reconnectEpoch;
}

export function notifyWarehouseSessionReconciled(): void {
  if (typeof window === 'undefined') return;
  reconnectEpoch += 1;
  window.dispatchEvent(
    new CustomEvent(WAREHOUSE_SESSION_RECONCILED, { detail: { epoch: reconnectEpoch } }),
  );
}

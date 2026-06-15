export const FACTORY_SESSION_RECONCILED = 'factory:session-reconciled';

let reconnectEpoch = 0;

export function getFactoryReconnectEpoch(): number {
  return reconnectEpoch;
}

/** Notify portal surfaces that transport reconnected and server snapshots were refetched. */
export function notifyFactorySessionReconciled(): void {
  if (typeof window === 'undefined') return;
  reconnectEpoch += 1;
  window.dispatchEvent(
    new CustomEvent(FACTORY_SESSION_RECONCILED, { detail: { epoch: reconnectEpoch } }),
  );
}

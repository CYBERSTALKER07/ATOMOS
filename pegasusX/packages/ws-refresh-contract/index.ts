/** Shared websocket refresh contracts across PegasusX web and native clients. */

export const ORDER_STATUS_REFRESH_EVENTS = new Set([
  "ORDER_STATUS_CHANGED",
  "ORDER_ASSIGNED",
  "ORDER_REASSIGNED",
  "ORDER_AMENDED",
  "ORDER_FINALIZED",
  "ORDER_COMPLETED",
  "ORDER_DISPATCHED",
  "ORDER_DELAYED",
  "ORDER_REROUTED",
  "CANCEL_REQUESTED",
  "CANCEL_APPROVED",
]);

export const DISPATCH_REFRESH_EVENTS = new Set([
  "DISPATCH_COMMITTED",
  "DISPATCH_LOCK_CHANGE",
  "WAREHOUSE_DISPATCH_LOCK_CHANGED",
  "FREEZE_LOCK_ACQUIRED",
  "FREEZE_LOCK_RELEASED",
  "MANIFEST_CREATED",
  "MANIFEST_SEALED",
  "MANIFEST_DISPATCHED",
  "MANIFEST_COMPLETED",
  "ORDER_ASSIGNED",
  ...ORDER_STATUS_REFRESH_EVENTS,
]);

export const PREORDER_REFRESH_EVENTS = new Set([
  "PRE_ORDER_NOTIFIED",
  "PRE_ORDER_NUDGE",
  "PRE_ORDER_CONFIRMATION",
  "PRE_ORDER_CONFIRMED",
  "PRE_ORDER_EDITED",
  "PRE_ORDER_CANCELLED",
  "PRE_ORDER_AUTO_ACCEPTED",
  "PRE_ORDER_DATE_PROPOSED",
  "PRE_ORDER_DATE_ACCEPTED",
  "PRE_ORDER_DATE_REJECTED",
]);

export const RETAILER_ORDER_REFRESH_EVENTS = new Set([
  ...ORDER_STATUS_REFRESH_EVENTS,
  ...PREORDER_REFRESH_EVENTS,
  "DRIVER_APPROACHING",
  "DRIVER_ARRIVED",
  "SETTLEMENT_REQUIRED",
  "PAYMENT_REQUIRED",
  "PAYMENT_CLEARED",
  "PAYMENT_SETTLED",
  "PAYMENT_FAILED",
  "PAYMENT_EXPIRED",
  "DELIVERY_SESSION_UPDATED",
]);

export function shouldRefreshOnEvent(eventType: string, allowed: ReadonlySet<string>): boolean {
  return allowed.has(eventType);
}

export function parseWsEventType(raw: unknown): string | null {
  if (typeof raw !== "string" || raw.trim() === "") {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as { type?: string };
    return typeof parsed.type === "string" ? parsed.type : null;
  } catch {
    return null;
  }
}

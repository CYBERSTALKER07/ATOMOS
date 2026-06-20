/** Warehouse hub events that should refresh the live fleet map surface. */
export const WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS = new Set([
  'MANIFEST_SEALED',
  'MANIFEST_DISPATCHED',
  'MANIFEST_COMPLETED',
  'DISPATCH_COMMITTED',
  'DRIVER_LOCATION_UPDATED',
  'ORDER_ASSIGNED',
  'DRIVER_AVAILABILITY_CHANGED',
  'VEHICLE_AVAILABILITY_CHANGED',
]);

/** Dispatch board + lock surfaces. */
export const WAREHOUSE_DISPATCH_REFRESH_EVENTS = new Set([
  'DISPATCH_COMMITTED',
  'DISPATCH_LOCK_CHANGE',
  'MANIFEST_CREATED',
  'MANIFEST_SEALED',
  'MANIFEST_DISPATCHED',
  'ORDER_ASSIGNED',
  'ORDER_STATUS_CHANGED',
  'DRIVER_AVAILABILITY_CHANGED',
  'VEHICLE_AVAILABILITY_CHANGED',
]);

/** Orders + pre-orders queue surfaces. */
export const WAREHOUSE_ORDERS_REFRESH_EVENTS = new Set([
  'ORDER_CREATED',
  'ORDER_STATUS_CHANGED',
  'ORDER_ASSIGNED',
  'ORDER_REASSIGNED',
  'ORDER_AMENDED',
  'ORDER_FINALIZED',
  'PRE_ORDER_NOTIFIED',
  'PRE_ORDER_NUDGE',
  'PRE_ORDER_CONFIRMATION',
  'PRE_ORDER_CONFIRMED',
  'PRE_ORDER_EDITED',
  'PRE_ORDER_CANCELLED',
  'PRE_ORDER_AUTO_ACCEPTED',
  'PRE_ORDER_DATE_PROPOSED',
  'PRE_ORDER_DATE_ACCEPTED',
  'PRE_ORDER_DATE_REJECTED',
]);

export function parseWarehouseWsEventType(raw: unknown): string | null {
  if (typeof raw !== 'string' || raw.trim() === '') {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as { type?: string };
    return typeof parsed.type === 'string' ? parsed.type : null;
  } catch {
    return null;
  }
}

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

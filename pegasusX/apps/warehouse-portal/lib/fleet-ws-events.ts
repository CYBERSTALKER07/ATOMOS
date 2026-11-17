'use client';

import { DISPATCH_REFRESH_EVENTS, ORDER_STATUS_REFRESH_EVENTS, parseWsEventType } from '@pegasusx/ws-refresh-contract';

/** Warehouse hub events that should refresh the live fleet map surface. */
export const WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS = new Set([
  'MANIFEST_SEALED',
  'MANIFEST_DISPATCHED',
  'MANIFEST_COMPLETED',
  'DRIVER_AVAILABILITY_CHANGED',
  'VEHICLE_AVAILABILITY_CHANGED',
  ...DISPATCH_REFRESH_EVENTS,
]);

export const WAREHOUSE_LOCATION_PATCH_EVENTS = new Set(['DRIVER_LOCATION_UPDATED']);

/** Dispatch board + lock surfaces. */
export const WAREHOUSE_DISPATCH_REFRESH_EVENTS = DISPATCH_REFRESH_EVENTS;

/** Orders + pre-orders queue surfaces. */
export const WAREHOUSE_ORDERS_REFRESH_EVENTS = new Set([
  'ORDER_CREATED',
  ...ORDER_STATUS_REFRESH_EVENTS,
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
  return parseWsEventType(raw);
}

/** Shared websocket and SSE refresh contracts across PegasusX web and native clients. */

export const SSE_RETRY_TIMEOUT_MS = 3000;
export const SSE_PING_INTERVAL_MS = 15000;
export const SSE_SUPPLIER_ENDPOINT = "/v1/supplier/events";
export const SSE_EVENTS_ENDPOINT = "/v1/events";

export interface SSEEvent<T = unknown> {
  id?: string;
  event?: string;
  type?: string;
  data: T;
}

export function parseSSEEventData<T = unknown>(raw: unknown): T | null {
  if (typeof raw !== "string" || raw.trim() === "") {
    return null;
  }
  try {
    return JSON.parse(raw) as T;
  } catch {
    return null;
  }
}

export function parseSSEEventType(event: MessageEvent | { type?: string; data?: unknown }): string | null {
  if (event && "type" in event && typeof event.type === "string" && event.type !== "message" && event.type !== "") {
    return event.type;
  }
  if (event && "data" in event) {
    return parseWsEventType(event.data);
  }
  return null;
}

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
  "ROUTE_REORDERED",
  "ORDER_CONDITION_REPORTED",
]);

export const RETURN_REFRESH_EVENTS = new Set([
  "SUPPLIER_RETURN_CREATED",
  "SUPPLIER_RETURN_RESOLVED",
  "RETURN_RECEIVED_AT_WAREHOUSE",
  "DRIVER_RETURN_APPROACHING",
]);

export const FAILED_DELIVERY_REFRESH_EVENTS = new Set([
  "SHOP_CLOSED",
  "SHOP_CLOSED_RESPONSE",
  "SHOP_CLOSED_ESCALATED",
  "SHOP_CLOSED_RESOLVED",
  "SHOP_CLOSED_BYPASS_OFFLOAD",
  "CREDIT_DELIVERY_MARKED",
  "CREDIT_DELIVERY_RESOLVED",
  "DELIVERY_DISPUTED",
]);

export const DISPATCH_REFRESH_EVENTS = new Set([
  "DISPATCH_COMMITTED",
  "DISPATCH_PLAN_UPDATED",
  "DISPATCH_LOCK_CHANGE",
  "WAREHOUSE_DISPATCH_LOCK_CHANGED",
  "FREEZE_LOCK_ACQUIRED",
  "FREEZE_LOCK_RELEASED",
  "MANIFEST_DRAFT_CREATED",
  "MANIFEST_LOADING_STARTED",
  "MANIFEST_ORDER_INJECTED",
  "MANIFEST_ORDER_EXCEPTION",
  "MANIFEST_DLQ_ESCALATION",
  "MANIFEST_REBALANCED",
  "MANIFEST_CANCELLED",
  "MANIFEST_SEALED",
  "MANIFEST_DISPATCHED",
  "MANIFEST_COMPLETED",
  "ORDER_ASSIGNED",
  ...ORDER_STATUS_REFRESH_EVENTS,
  ...RETURN_REFRESH_EVENTS,
  ...FAILED_DELIVERY_REFRESH_EVENTS,
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
  "SPLIT_PAYMENT_CREATED",
  "PAYMENT_CLEARED",
  "PAYMENT_SETTLED",
  "PAYMENT_FAILED",
  "PAYMENT_EXPIRED",
  "DELIVERY_SESSION_UPDATED",
  "DELIVERY_DISPUTED",
  "RETAILER_CREDIT_PROFILE_CHANGED",
  "RETAILER_CREDIT_LIMIT_BREACHED",
]);

export function shouldRefreshOnEvent(eventType: string, allowed: ReadonlySet<string>): boolean {
  return allowed.has(eventType);
}

export const PULSE_REFRESH_EVENTS = new Set([
  ...ORDER_STATUS_REFRESH_EVENTS,
  ...PREORDER_REFRESH_EVENTS,
  ...RETURN_REFRESH_EVENTS,
  ...FAILED_DELIVERY_REFRESH_EVENTS,
  "DISPATCH_COMMITTED",
  "MANIFEST_DRAFT_CREATED",
  "MANIFEST_SEALED",
  "MANIFEST_DISPATCHED",
  "MANIFEST_COMPLETED",
  "MANIFEST_LOADING_STARTED",
  "ORDER_CREATED",
  "ORDER_ASSIGNED",
  "PAYMENT_REQUIRED",
  "SPLIT_PAYMENT_CREATED",
  "PAYMENT_CLEARED",
  "PAYMENT_SETTLED",
]);

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

/** Command-dashboard dirty slice. Location never refetches the rollup GET. */
export type DashboardDirtySlice =
  | "orders"
  | "manifests"
  | "money"
  | "shop_closed"
  | "pulse"
  | "map"
  | "plan"
  | null;

export const DASHBOARD_MONEY_EVENTS = new Set([
  "PAYMENT_REQUIRED",
  "SPLIT_PAYMENT_CREATED",
  "PAYMENT_CLEARED",
  "PAYMENT_SETTLED",
  "PAYMENT_FAILED",
  "PAYMENT_EXPIRED",
  "FISCAL_FAILED",
  "RECONCILIATION_REQUIRED",
]);

export const DASHBOARD_PLAN_EVENTS = new Set([
  "SCENARIO_PUBLISHED",
  "PLANNING_SCENARIO_PUBLISHED",
]);

export const DASHBOARD_ROLLUP_REFRESH_EVENTS = new Set([
  ...DISPATCH_REFRESH_EVENTS,
  ...DASHBOARD_MONEY_EVENTS,
  "ORDER_CREATED",
]);

export function dashboardDirtySlice(eventType: string): DashboardDirtySlice {
  const type = eventType.trim();
  if (!type || type.startsWith("SYSTEM")) {
    return null;
  }
  if (type === "DRIVER_LOCATION_UPDATED") {
    return "map";
  }
  if (DASHBOARD_PLAN_EVENTS.has(type)) {
    return "plan";
  }
  if (type.startsWith("PULSE_")) {
    return "pulse";
  }
  if (type.startsWith("SHOP_CLOSED") || FAILED_DELIVERY_REFRESH_EVENTS.has(type)) {
    return "shop_closed";
  }
  if (DASHBOARD_MONEY_EVENTS.has(type)) {
    return "money";
  }
  if (
    type.startsWith("MANIFEST_") ||
    type.startsWith("DISPATCH_") ||
    type.includes("DISPATCH_LOCK") ||
    type.startsWith("FREEZE_LOCK") ||
    type.startsWith("FACTORY_MANIFEST") ||
    type.startsWith("FACTORY_TRANSFER") ||
    type === "FACTORY_SUPPLY_REQUEST_UPDATE" ||
    type === "FACTORY_OUTBOX_FAILED"
  ) {
    return "manifests";
  }
  if (
    ORDER_STATUS_REFRESH_EVENTS.has(type) ||
    type === "ORDER_CREATED" ||
    type === "ORDER_ASSIGNED" ||
    RETURN_REFRESH_EVENTS.has(type)
  ) {
    return "orders";
  }
  return null;
}

export function shouldRefetchDashboardRollup(eventType: string): boolean {
  const slice = dashboardDirtySlice(eventType);
  return slice === "orders" || slice === "manifests" || slice === "money" || slice === "shop_closed";
}

export type DriverLocationPatch = {
  driver_id?: string;
  route_id?: string;
  lat: number;
  lng: number;
};

function finiteNumber(value: unknown): number | null {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim() !== "") {
    const parsed = Number(value);
    if (Number.isFinite(parsed)) {
      return parsed;
    }
  }
  return null;
}

function asRecord(raw: unknown): Record<string, unknown> | null {
  if (typeof raw === "string") {
    try {
      const parsed = JSON.parse(raw) as unknown;
      return parsed && typeof parsed === "object" ? (parsed as Record<string, unknown>) : null;
    } catch {
      return null;
    }
  }
  if (raw && typeof raw === "object") {
    return raw as Record<string, unknown>;
  }
  return null;
}

export function parseDriverLocationPatch(raw: unknown): DriverLocationPatch | null {
  const obj = asRecord(raw);
  if (!obj) {
    return null;
  }
  if (obj.type !== "DRIVER_LOCATION_UPDATED") {
    return null;
  }
  const data =
    obj.data && typeof obj.data === "object" ? (obj.data as Record<string, unknown>) : obj;
  const lat = finiteNumber(data.lat) ?? finiteNumber(data.latitude) ?? finiteNumber(data.gps_lat);
  const lng = finiteNumber(data.lng) ?? finiteNumber(data.longitude) ?? finiteNumber(data.gps_lng);
  if (lat == null || lng == null) {
    return null;
  }
  return {
    driver_id: typeof data.driver_id === "string" ? data.driver_id : undefined,
    route_id: typeof data.route_id === "string" ? data.route_id : undefined,
    lat,
    lng,
  };
}

export function applyDriverLocationPatch<
  T extends {
    driver_id?: string;
    route_id?: string;
    live_location_available?: boolean;
    driver_location?: {
      lat?: number;
      lng?: number;
      latitude?: number;
      longitude?: number;
    };
  },
>(routes: T[], patch: DriverLocationPatch): T[] {
  if (!patch.driver_id && !patch.route_id) {
    return routes;
  }
  return routes.map((route) => {
    const driverMatch = !patch.driver_id || route.driver_id === patch.driver_id;
    const routeMatch = !patch.route_id || !route.route_id || route.route_id === patch.route_id;
    if (!driverMatch || !routeMatch) {
      return route;
    }
    return {
      ...route,
      live_location_available: true,
      driver_location: {
        ...route.driver_location,
        lat: patch.lat,
        lng: patch.lng,
        latitude: patch.lat,
        longitude: patch.lng,
      },
    };
  });
}

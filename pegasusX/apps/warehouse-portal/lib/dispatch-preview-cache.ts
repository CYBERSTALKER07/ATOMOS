import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";
import type {
  WarehouseDispatchDriver,
  WarehouseDispatchOrder,
  WarehouseDispatchProposedRoute,
  WarehouseUnavailableDispatchDriver,
} from "@pegasusx/types";

export type DispatchPreviewSnapshot = {
  orders: WarehouseDispatchOrder[];
  drivers: WarehouseDispatchDriver[];
  unavailableDrivers: WarehouseUnavailableDispatchDriver[];
  proposedRoutes: WarehouseDispatchProposedRoute[];
  optimizerSource: string | null;
  optimizerWarnings: string[];
  windowConstrainedCount: number;
  fleetEffectiveCapacityVU: number;
  planFingerprint: string | null;
  cachedAt: string;
};

const CACHE_PREFIX = "warehouse_dispatch_preview";

export function dispatchPreviewCacheKey(
  warehouseId: string,
  orderIds: string[] = [],
): string {
  const scope = warehouseId || "warehouse";
  if (orderIds.length === 0) {
    return `${CACHE_PREFIX}:${scope}:all`;
  }
  const filter = [...orderIds].sort().join(",");
  return `${CACHE_PREFIX}:${scope}:${filter}`;
}

export async function getDispatchPreviewCache(
  key: string,
): Promise<DispatchPreviewSnapshot | null> {
  return cacheGet<DispatchPreviewSnapshot>(key, {
    maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS,
  });
}

export async function setDispatchPreviewCache(
  key: string,
  snapshot: DispatchPreviewSnapshot,
): Promise<void> {
  await cacheSet(key, snapshot);
}

export function snapshotFromPreviewResponse(data: {
  undispatched_orders?: WarehouseDispatchOrder[];
  orders?: WarehouseDispatchOrder[];
  available_drivers?: WarehouseDispatchDriver[];
  drivers?: WarehouseDispatchDriver[];
  unavailable_drivers?: WarehouseUnavailableDispatchDriver[];
  proposed_routes?: WarehouseDispatchProposedRoute[];
  optimizer_source?: string | null;
  optimizer_warnings?: string[];
  window_constrained_count?: number;
  fleet_effective_capacity_vu?: number;
  plan_fingerprint?: string | null;
}): DispatchPreviewSnapshot {
  return {
    orders: data.undispatched_orders || data.orders || [],
    drivers: data.available_drivers || data.drivers || [],
    unavailableDrivers: data.unavailable_drivers || [],
    proposedRoutes: data.proposed_routes || [],
    optimizerSource: data.optimizer_source ?? null,
    optimizerWarnings: data.optimizer_warnings || [],
    windowConstrainedCount: data.window_constrained_count || 0,
    fleetEffectiveCapacityVU: data.fleet_effective_capacity_vu ?? 0,
    planFingerprint: data.plan_fingerprint ?? null,
    cachedAt: new Date().toISOString(),
  };
}

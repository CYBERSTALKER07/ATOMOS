import { DEFAULT_CACHE_MAX_AGE_MS, cacheGet, cacheSet } from "@pegasusx/desktop-cache";

export type DispatchRunRow = {
  run_id: string;
  status: string;
  manifest_count: number;
  orders_assigned: number;
  created_at: string;
};

const CACHE_PREFIX = "warehouse_dispatch_runs";

export function dispatchRunsCacheKey(warehouseId: string): string {
  return `${CACHE_PREFIX}:${warehouseId || "warehouse"}`;
}

export async function getDispatchRunsCache(
  key: string,
): Promise<DispatchRunRow[] | null> {
  return cacheGet<DispatchRunRow[]>(key, { maxAgeMs: DEFAULT_CACHE_MAX_AGE_MS });
}

export async function setDispatchRunsCache(
  key: string,
  runs: DispatchRunRow[],
): Promise<void> {
  await cacheSet(key, runs);
}

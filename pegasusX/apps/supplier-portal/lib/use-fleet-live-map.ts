import { useCallback, useState } from "react";
import { usePolling } from "@pegasusx/api-client";
import type { SupplierFleetLiveRoute } from "@pegasusx/types";
import { applyDriverLocationPatch, parseDriverLocationPatch } from "@pegasusx/ws-refresh-contract";
import { createSupplierApi } from "@/lib/api";
import {
  SUPPLIER_FLEET_LIVE_REFRESH_EVENTS,
  SUPPLIER_LOCATION_PATCH_EVENTS,
} from "@/lib/supplier-ws-events";
import { useSupplierWsRefresh } from "@/lib/use-supplier-ws-refresh";

const api = createSupplierApi();

export function useFleetLiveMap(pollMs = 15_000) {
  const [routes, setRoutes] = useState<SupplierFleetLiveRoute[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fetchedAt, setFetchedAt] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      const response = await api.getSupplierFleetLiveMap();
      setRoutes(response.routes ?? []);
      setFetchedAt(response.fetched_at);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : "fleet_live_map_failed");
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, []);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await refresh(routes.length > 0);
    },
    pollMs,
    [pollMs, refresh, routes.length],
    { hiddenIntervalMs: 60_000 },
  );

  useSupplierWsRefresh(
    () => {
      void refresh(true);
    },
    {
      eventTypes: SUPPLIER_FLEET_LIVE_REFRESH_EVENTS,
      debounceMs: 750,
    },
  );

  useSupplierWsRefresh(
    (_eventType, raw) => {
      const patch = parseDriverLocationPatch(raw);
      if (!patch) {
        return;
      }
      setRoutes((current) => applyDriverLocationPatch(current, patch));
    },
    {
      eventTypes: SUPPLIER_LOCATION_PATCH_EVENTS,
      debounceMs: 400,
    },
  );

  return { routes, loading, error, fetchedAt, refresh };
}

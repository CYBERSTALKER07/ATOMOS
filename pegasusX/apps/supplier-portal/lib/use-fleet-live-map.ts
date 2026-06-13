import { useCallback, useEffect, useState } from "react";
import type { SupplierFleetLiveRoute } from "@pegasusx/types";
import { createSupplierApi } from "@/lib/api";
import { SUPPLIER_FLEET_LIVE_REFRESH_EVENTS } from "@/lib/supplier-ws-events";
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
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => {
      void refresh(true);
    }, pollMs);
    return () => window.clearInterval(timer);
  }, [pollMs, refresh]);

  useSupplierWsRefresh(
    () => {
      void refresh(true);
    },
    {
      eventTypes: SUPPLIER_FLEET_LIVE_REFRESH_EVENTS,
      debounceMs: 750,
    },
  );

  return { routes, loading, error, fetchedAt, refresh };
}

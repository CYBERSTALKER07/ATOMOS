'use client';

import { useCallback, useEffect, useState } from 'react';
import { usePolling } from '@pegasusx/api-client';
import { applyDriverLocationPatch, parseDriverLocationPatch } from '@pegasusx/ws-refresh-contract';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';

export type FactoryFleetLiveRoute = {
  manifest_id: string;
  route_id?: string;
  driver_id: string;
  driver_name?: string;
  manifest_state: string;
  live_location_available: boolean;
  location_stale?: boolean;
  driver_location?: {
    lat: number;
    lng: number;
    latitude?: number;
    longitude?: number;
  };
};

export function useFactoryFleetLiveMap(pollMs = 15_000) {
  const [routes, setRoutes] = useState<FactoryFleetLiveRoute[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fetchedAt, setFetchedAt] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    if (!silent) setLoading(true);
    try {
      const res = await apiFetch('/v1/factory/fleet/live-map');
      if (!res.ok) {
        throw new Error(`fleet_live_map_${res.status}`);
      }
      const data = (await res.json()) as {
        routes?: FactoryFleetLiveRoute[];
        fetched_at?: string;
      };
      setRoutes(data.routes ?? []);
      setFetchedAt(data.fetched_at ?? null);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'fleet_live_map_failed');
    } finally {
      if (!silent) setLoading(false);
    }
  }, []);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await refresh(true);
    },
    pollMs,
    [refresh],
    { hiddenIntervalMs: 60_000 },
  );

  useFactorySessionReconcile(() => {
    void refresh(true);
  });

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: (payload) => {
        const patch = parseDriverLocationPatch(payload);
        if (patch) {
          setRoutes((current) => applyDriverLocationPatch(current, patch));
          return;
        }
        const event = parseFactoryLiveEvent(payload);
        if (!event) return;
        if (event.type === 'FACTORY_TRANSFER_UPDATE' || event.type === 'FACTORY_MANIFEST_UPDATE') {
          void refresh(true);
        }
      },
    });
    return () => unsubscribe();
  }, [refresh]);

  return { routes, loading, error, fetchedAt, refresh };
}

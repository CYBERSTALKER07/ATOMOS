'use client';

import { useCallback, useEffect, useState } from 'react';
import { usePolling } from '@pegasusx/api-client';
import { createWarehouseApi } from '@/lib/api';
import { subscribeWarehouseWS } from '@/lib/auth';
import { applyDriverLocationPatch, parseDriverLocationPatch } from '@pegasusx/ws-refresh-contract';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS, WAREHOUSE_LOCATION_PATCH_EVENTS } from '@/lib/fleet-ws-events';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';

const api = createWarehouseApi();

export function useWarehouseFleetLiveMap(pollMs = 15_000) {
  const [routes, setRoutes] = useState<Awaited<ReturnType<typeof api.getWarehouseFleetLiveMap>>['routes']>([]);
  const [yardManifests, setYardManifests] = useState<Awaited<ReturnType<typeof api.getWarehouseFleetLiveMap>>['yard_manifests']>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [fetchedAt, setFetchedAt] = useState<string | null>(null);

  const refresh = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      const response = await api.getWarehouseFleetLiveMap();
      setRoutes(response.routes ?? []);
      setYardManifests(response.yard_manifests ?? []);
      setFetchedAt(response.fetched_at);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'fleet_live_map_failed');
    } finally {
      if (!silent) {
        setLoading(false);
      }
    }
  }, []);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await refresh((routes?.length ?? 0) > 0);
    },
    pollMs,
    [pollMs, refresh],
    { hiddenIntervalMs: 60_000 },
  );

  useEffect(() => {
    let signalTimer: number | undefined;
    const unsubscribe = subscribeWarehouseWS({
      onMessage: (payload) => {
        const eventType = parseWarehouseWsEventType(payload);
        if (eventType && WAREHOUSE_LOCATION_PATCH_EVENTS.has(eventType)) {
          const patch = parseDriverLocationPatch(payload);
          if (patch) {
            setRoutes((current) => applyDriverLocationPatch(current ?? [], patch));
          }
          return;
        }
        if (!eventType || !WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS.has(eventType) || eventType.startsWith('SYSTEM')) {
          return;
        }
        if (signalTimer !== undefined) {
          window.clearTimeout(signalTimer);
        }
        signalTimer = window.setTimeout(() => {
          void refresh(true);
        }, 750);
      },
    });
    return () => {
      if (signalTimer !== undefined) {
        window.clearTimeout(signalTimer);
      }
      unsubscribe();
    };
  }, [refresh]);

  useWarehouseSessionReconcile(() => {
    void refresh(true);
  });

  return { routes, yardManifests, loading, error, fetchedAt, refresh };
}

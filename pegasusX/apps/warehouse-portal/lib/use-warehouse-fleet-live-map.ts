'use client';

import { useCallback, useEffect, useState } from 'react';
import { createWarehouseApi } from '@/lib/api';
import { subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS } from '@/lib/fleet-ws-events';

const api = createWarehouseApi();

export function useWarehouseFleetLiveMap(pollMs = 15_000) {
  const [routes, setRoutes] = useState<Awaited<ReturnType<typeof api.getWarehouseFleetLiveMap>>['routes']>([]);
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
      setFetchedAt(response.fetched_at);
      setError(null);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'fleet_live_map_failed');
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

  useEffect(() => {
    let signalTimer: number | undefined;
    const unsubscribe = subscribeWarehouseWS({
      onMessage: (payload) => {
        const eventType = parseWarehouseWsEventType(payload);
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

  return { routes, loading, error, fetchedAt, refresh };
}

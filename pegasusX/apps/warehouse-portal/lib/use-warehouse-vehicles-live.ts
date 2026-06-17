'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import type { WarehouseFleetVehicle, WarehouseFleetVehicleListResponse } from '@pegasusx/types';
import { apiFetch, subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS } from '@/lib/fleet-ws-events';

const VEHICLE_REFRESH_EVENTS = new Set([
  ...WAREHOUSE_FLEET_LIVE_REFRESH_EVENTS,
  'DRIVER_CREATED',
  'VEHICLE_CREATED',
]);

export function useWarehouseVehiclesLive(options: { enabled?: boolean } = {}) {
  const enabled = options.enabled !== false;
  const [vehicles, setVehicles] = useState<WarehouseFleetVehicle[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [liveMessage, setLiveMessage] = useState<string | null>(null);

  const load = useCallback(async () => {
    if (!enabled) {
      return;
    }
    setError(null);
    try {
      const res = await apiFetch('/v1/warehouse/ops/vehicles');
      if (!res.ok) {
        throw new Error('Unable to load trucks');
      }
      const data = await res.json() as WarehouseFleetVehicleListResponse;
      setVehicles(data.vehicles || []);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : 'Unable to load trucks');
    } finally {
      setLoading(false);
    }
  }, [enabled]);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    void load();
  }, [enabled, load]);

  const loadRef = useRef(load);
  loadRef.current = load;

  useEffect(() => {
    if (!enabled) {
      return;
    }
    let timer: number | undefined;
    const unsubscribe = subscribeWarehouseWS({
      onMessage: (payload) => {
        const eventType = parseWarehouseWsEventType(payload);
        if (!eventType || !VEHICLE_REFRESH_EVENTS.has(eventType)) {
          return;
        }
        if (eventType === 'VEHICLE_AVAILABILITY_CHANGED' || eventType === 'DRIVER_AVAILABILITY_CHANGED') {
          try {
            const parsed = JSON.parse(payload) as { body?: string; title?: string };
            if (parsed.body || parsed.title) {
              setLiveMessage(parsed.body || parsed.title || 'Fleet updated');
            }
          } catch {
            setLiveMessage('Fleet updated');
          }
        }
        if (timer !== undefined) {
          window.clearTimeout(timer);
        }
        const delay = eventType === 'VEHICLE_AVAILABILITY_CHANGED' ? 0 : 200;
        timer = window.setTimeout(() => {
          void loadRef.current();
        }, delay);
      },
    });
    return () => {
      if (timer !== undefined) {
        window.clearTimeout(timer);
      }
      unsubscribe();
    };
  }, [enabled]);

  return {
    vehicles,
    loading,
    error,
    liveMessage,
    setLiveMessage,
    reload: load,
  };
}

export async function fetchWarehouseVehicle(vehicleId: string): Promise<WarehouseFleetVehicle | null> {
  const res = await apiFetch(`/v1/warehouse/ops/vehicles/${vehicleId}`);
  if (res.status === 404) {
    return null;
  }
  if (!res.ok) {
    throw new Error('Unable to load truck');
  }
  const data = await res.json() as { vehicle?: WarehouseFleetVehicle };
  return data.vehicle ?? null;
}

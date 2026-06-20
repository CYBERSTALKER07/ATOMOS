'use client';

import { useEffect, useRef } from 'react';
import { subscribeWarehouseWS } from '@/lib/auth';
import { parseWarehouseWsEventType, WAREHOUSE_ORDERS_REFRESH_EVENTS } from '@/lib/fleet-ws-events';

type UseWarehouseWsRefreshOptions = {
  eventTypes?: ReadonlySet<string>;
  debounceMs?: number;
  enabled?: boolean;
};

/**
 * Invokes onSignal when matching warehouse WS events arrive.
 * Polling / manual refresh should remain as fallback.
 */
export function useWarehouseWsRefresh(
  onSignal: (eventType: string) => void,
  {
    eventTypes = WAREHOUSE_ORDERS_REFRESH_EVENTS,
    debounceMs = 500,
    enabled = true,
  }: UseWarehouseWsRefreshOptions = {},
) {
  const onSignalRef = useRef(onSignal);
  onSignalRef.current = onSignal;

  useEffect(() => {
    if (!enabled) return;

    let signalTimer: number | undefined;

    const unsubscribe = subscribeWarehouseWS({
      onMessage: (raw) => {
        const eventType = parseWarehouseWsEventType(raw);
        if (!eventType || !eventTypes.has(eventType) || eventType.startsWith('SYSTEM')) {
          return;
        }
        if (signalTimer !== undefined) {
          window.clearTimeout(signalTimer);
        }
        signalTimer = window.setTimeout(() => {
          onSignalRef.current(eventType);
        }, debounceMs);
      },
    });

    return () => {
      if (signalTimer !== undefined) {
        window.clearTimeout(signalTimer);
      }
      unsubscribe();
    };
  }, [debounceMs, enabled, eventTypes]);
}

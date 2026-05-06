'use client';

/**
 * useSync — Desert Protocol catchup orchestrator.
 *
 * Tracks last-seen server timestamp in localStorage. On reconnect (online,
 * focus, visibilitychange) calls GET /v1/sync/catchup?since=<lastSeen> and
 * publishes the delta to subscribers so each surface can reconcile its state
 * without replaying every missed real-time event.
 *
 * This composes alongside useTelemetry / useNotifications — it does NOT
 * replace them. Sockets handle the live stream; this handles the gap.
 */

import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';

const STORAGE_KEY = 'pegasus.sync.lastSeen';

export interface SyncOrderDelta {
  order_id: string;
  state: string;
  updated_at: string;
}

export interface SyncFleetDelta {
  driver_id: string;
  status: string;
  is_offline: boolean;
}

export interface SyncCatchup {
  orders: SyncOrderDelta[];
  fleet?: SyncFleetDelta[];
  unread_notifications: number;
  server_time: string;
}

type CatchupListener = (delta: SyncCatchup) => void;

const listeners = new Set<CatchupListener>();
let inFlight: Promise<SyncCatchup | null> | null = null;
let lifecycleBound = false;
let consumerCount = 0;

function readLastSeen(): string {
  if (typeof window === 'undefined') return new Date(Date.now() - 60_000).toISOString();
  return localStorage.getItem(STORAGE_KEY) || new Date(Date.now() - 60_000).toISOString();
}

function writeLastSeen(ts: string): void {
  if (typeof window === 'undefined') return;
  try { localStorage.setItem(STORAGE_KEY, ts); } catch { /* quota exceeded — non-fatal */ }
}

async function runCatchup(): Promise<SyncCatchup | null> {
  if (inFlight) return inFlight;

  inFlight = (async () => {
    try {
      const since = readLastSeen();
      const res = await apiFetch(`/v1/sync/catchup?since=${encodeURIComponent(since)}`);
      if (!res.ok) return null;
      const delta = (await res.json()) as SyncCatchup;
      writeLastSeen(delta.server_time);
      listeners.forEach((l) => {
        try { l(delta); } catch { /* listener errors are isolated */ }
      });
      return delta;
    } catch {
      return null;
    } finally {
      inFlight = null;
    }
  })();

  return inFlight;
}

function bindLifecycle(): void {
  if (lifecycleBound || typeof window === 'undefined') return;
  const onWake = () => { void runCatchup(); };
  const onVisible = () => { if (document.visibilityState === 'visible') onWake(); };
  window.addEventListener('online', onWake);
  window.addEventListener('focus', onWake);
  document.addEventListener('visibilitychange', onVisible);
  lifecycleBound = true;
}

/**
 * useSync subscribes a surface to catchup deltas. Pass an `onDelta` callback
 * to react to incoming reconciliation payloads. Returns `triggerCatchup` for
 * manual invocation (e.g., after a known-stale period).
 */
export function useSync(onDelta?: CatchupListener) {
  const [lastDelta, setLastDelta] = useState<SyncCatchup | null>(null);

  useEffect(() => {
    const handler: CatchupListener = (delta) => {
      setLastDelta(delta);
      onDelta?.(delta);
    };
    listeners.add(handler);
    consumerCount += 1;
    bindLifecycle();
    // Initial catchup on mount so a fresh tab reconciles immediately.
    void runCatchup();
    return () => {
      listeners.delete(handler);
      consumerCount = Math.max(0, consumerCount - 1);
    };
  }, [onDelta]);

  const triggerCatchup = useCallback(() => runCatchup(), []);

  return { lastDelta, triggerCatchup };
}

/** Mark the live stream's last good frame so the next catchup is precise. */
export function markSyncFrame(serverTime?: string): void {
  writeLastSeen(serverTime || new Date().toISOString());
}

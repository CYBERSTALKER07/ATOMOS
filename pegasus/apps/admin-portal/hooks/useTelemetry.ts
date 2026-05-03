'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { readTokenFromCookie } from '@/lib/auth';
import {
  isDeltaEvent,
  applyDelta,
  seedCache,
  DeltaType,
  type DeltaEventType,
} from '@/lib/delta-sync';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const WS_BASE = API.replace(/^http/, 'ws');

const MAX_BACKOFF_MS = 16_000;
const INITIAL_BACKOFF_MS = 1_000;

export interface TelemetryState {
  connected: boolean;
  lastSyncTs: string | null;
  reconnectCount: number;
}

type DeltaListener = (type: DeltaEventType, id: string, state: Record<string, unknown>) => void;

/**
 * useTelemetry: Real-time delta-sync WebSocket hook.
 *
 * - Connects to /ws/telemetry with JWT auth
 * - Routes incoming DeltaEvents through the entity cache (delta-sync.ts)
 * - On reconnect, fetches catch-up deltas from /v1/sync/catchup
 * - Exponential backoff: 1s → 2s → 4s → 8s → 16s max (with jitter)
 * - Persists lastSyncTs to sessionStorage for tab-close recovery
 */
export function useTelemetry(onDelta?: DeltaListener): TelemetryState {
  const [state, setState] = useState<TelemetryState>({
    connected: false,
    lastSyncTs: null,
    reconnectCount: 0,
  });

  const wsRef = useRef<WebSocket | null>(null);
  const backoffRef = useRef(INITIAL_BACKOFF_MS);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);
  const lastSyncRef = useRef<string | null>(null);
  const isFirstConnect = useRef(true);
  const disposedRef = useRef(false);

  // Restore lastSyncTs from sessionStorage on mount
  useEffect(() => {
    const stored = sessionStorage.getItem('void_last_sync');
    if (stored) {
      lastSyncRef.current = stored;
      setState((s) => ({ ...s, lastSyncTs: stored }));
    }
  }, []);

  const updateLastSync = useCallback((ts: string) => {
    lastSyncRef.current = ts;
    sessionStorage.setItem('void_last_sync', ts);
    setState((s) => ({ ...s, lastSyncTs: ts }));
  }, []);

  /**
   * Fetch catch-up deltas from backend after reconnection.
   * Seeds the entity cache with any changes that occurred while offline.
   */
  const runCatchUp = useCallback(async () => {
    const since = lastSyncRef.current;
    if (!since) return; // First connect — no catch-up needed

    const token = readTokenFromCookie();
    if (!token) return;

    try {
      const res = await fetch(`${API}/v1/sync/catchup?since=${encodeURIComponent(since)}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) return;

      const data = await res.json();
      if (disposedRef.current) return;

      // Seed cache with catch-up data
      if (data.orders?.length) {
        seedCache(DeltaType.ORDER_UPDATE, data.orders);
      }
      if (data.fleet?.length) {
        seedCache(DeltaType.FLEET_GPS, data.fleet);
      }

      // Update sync timestamp to server time
      if (data.server_time) {
        updateLastSync(data.server_time);
      }
    } catch (err) {
      console.error('[TELEMETRY] Catch-up failed:', err);
    }
  }, [updateLastSync]);

  const connect = useCallback(() => {
    if (disposedRef.current) return;
    const token = readTokenFromCookie();
    if (!token) return;

    clearTimeout(reconnectTimer.current);
    const ws = new WebSocket(
      `${WS_BASE}/ws/telemetry?token=${encodeURIComponent(token)}`,
    );
    wsRef.current = ws;

    ws.onopen = () => {
      if (disposedRef.current) {
        ws.close();
        return;
      }
      backoffRef.current = INITIAL_BACKOFF_MS;
      setState((s) => ({ ...s, connected: true }));

      if (!isFirstConnect.current) {
        // Reconnect — trigger catch-up
        runCatchUp();
      }
      isFirstConnect.current = false;

      // Record sync time
      updateLastSync(new Date().toISOString());
    };

    ws.onmessage = (event) => {
      if (disposedRef.current) return;
      try {
        const msg = JSON.parse(event.data);
        if (isDeltaEvent(msg)) {
          // Apply to entity cache (snake_case)
          const merged = applyDelta(msg);

          // Notify listener with full merged state
          onDelta?.(msg.t, msg.i, merged);

          // Update last sync timestamp
          updateLastSync(new Date().toISOString());
        }
      } catch {
        /* ignore malformed frames */
      }
    };

    ws.onclose = () => {
      if (wsRef.current === ws) {
        wsRef.current = null;
      }
      if (disposedRef.current) return;
      setState((s) => ({
        ...s,
        connected: false,
        reconnectCount: s.reconnectCount + 1,
      }));

      // Exponential backoff with jitter
      const jitter = Math.random() * 500;
      const delay = Math.min(backoffRef.current + jitter, MAX_BACKOFF_MS);
      backoffRef.current = Math.min(backoffRef.current * 2, MAX_BACKOFF_MS);

      reconnectTimer.current = setTimeout(connect, delay);
    };

    ws.onerror = () => ws.close();
  }, [onDelta, runCatchUp, updateLastSync]);

  useEffect(() => {
    disposedRef.current = false;
    connect();
    return () => {
      disposedRef.current = true;
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [connect]);

  return state;
}

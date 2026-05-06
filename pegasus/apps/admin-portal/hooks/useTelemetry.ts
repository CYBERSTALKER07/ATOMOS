'use client';

import { useEffect, useRef, useState } from 'react';
import { getAdminToken, readTokenFromCookie } from '@/lib/auth';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const WS_BASE = API.replace(/^http/, 'ws');
const MAX_BACKOFF_MS = 16_000;
const INITIAL_BACKOFF_MS = 1_000;

export type TelemetryConnectionStatus = 'CONNECTING' | 'LIVE' | 'OFFLINE';
export type TelemetryMessage = Record<string, unknown>;

export interface TelemetryDriverPosition {
  driver_id: string;
  latitude: number;
  longitude: number;
  timestamp?: number;
}

export interface TelemetryState {
  connected: boolean;
  status: TelemetryConnectionStatus;
  lastSyncTs: string | null;
  reconnectCount: number;
}

interface UseTelemetryOptions {
  enabled?: boolean;
}

type TelemetryListener = (message: TelemetryMessage) => void;

const initialState: TelemetryState = {
  connected: false,
  status: 'OFFLINE',
  lastSyncTs: null,
  reconnectCount: 0,
};

let telemetryState = initialState;
let socket: WebSocket | null = null;
let reconnectTimer: number | null = null;
let reconnectDelayMs = INITIAL_BACKOFF_MS;
let consumerCount = 0;
let lifecycleBound = false;
let lifecycleCleanup: (() => void) | null = null;

const messageListeners = new Set<TelemetryListener>();
const stateListeners = new Set<(state: TelemetryState) => void>();

function normalizeSnapshotDriver(candidate: unknown): TelemetryDriverPosition | null {
  if (!candidate || typeof candidate !== 'object' || Array.isArray(candidate)) {
    return null;
  }

  const record = candidate as Record<string, unknown>;
  if (
    typeof record.driver_id !== 'string' ||
    typeof record.latitude !== 'number' ||
    typeof record.longitude !== 'number'
  ) {
    return null;
  }

  return {
    driver_id: record.driver_id,
    latitude: record.latitude,
    longitude: record.longitude,
    timestamp: typeof record.timestamp === 'number' ? record.timestamp : undefined,
  };
}

function normalizeDeltaDriver(candidate: unknown): TelemetryDriverPosition | null {
  if (!candidate || typeof candidate !== 'object' || Array.isArray(candidate)) {
    return null;
  }

  const record = candidate as Record<string, unknown>;
  const location = record.l;
  if (
    typeof record.d !== 'string' ||
    !Array.isArray(location) ||
    location.length < 2 ||
    typeof location[0] !== 'number' ||
    typeof location[1] !== 'number'
  ) {
    return null;
  }

  return {
    driver_id: record.d,
    latitude: location[0],
    longitude: location[1],
  };
}

export function extractDriverPositions(message: TelemetryMessage): TelemetryDriverPosition[] {
  if (
    typeof message.driver_id === 'string' &&
    typeof message.latitude === 'number' &&
    typeof message.longitude === 'number'
  ) {
    return [{
      driver_id: message.driver_id,
      latitude: message.latitude,
      longitude: message.longitude,
      timestamp: typeof message.timestamp === 'number' ? message.timestamp : undefined,
    }];
  }

  if (message.type === 'FLEET_SNAPSHOT' && Array.isArray(message.drivers)) {
    return message.drivers
      .map((driver) => normalizeSnapshotDriver(driver))
      .filter((driver): driver is TelemetryDriverPosition => driver !== null);
  }

  if (message.t === 'FLT_GPS' && message.d && typeof message.d === 'object' && !Array.isArray(message.d)) {
    const payload = message.d as Record<string, unknown>;
    if (!Array.isArray(payload.drivers)) {
      return [];
    }
    return payload.drivers
      .map((driver) => normalizeDeltaDriver(driver))
      .filter((driver): driver is TelemetryDriverPosition => driver !== null);
  }

  return [];
}

function publishState(next: Partial<TelemetryState>): void {
  telemetryState = { ...telemetryState, ...next };
  stateListeners.forEach((listener) => listener(telemetryState));
}

function clearReconnectTimer(): void {
  if (!reconnectTimer) {
    return;
  }

  window.clearTimeout(reconnectTimer);
  reconnectTimer = null;
}

function scheduleReconnect(): void {
  if (typeof window === 'undefined' || consumerCount === 0 || reconnectTimer) {
    return;
  }

  const jitterMs = Math.floor(Math.random() * 500);
  const delayMs = Math.min(reconnectDelayMs + jitterMs, MAX_BACKOFF_MS);
  reconnectDelayMs = Math.min(reconnectDelayMs * 2, MAX_BACKOFF_MS);

  reconnectTimer = window.setTimeout(() => {
    reconnectTimer = null;
    void connectTelemetry();
  }, delayMs);
}

function broadcastMessage(message: TelemetryMessage): void {
  messageListeners.forEach((listener) => listener(message));
}

async function resolveTelemetryToken(): Promise<string> {
  const cookieToken = readTokenFromCookie();
  if (cookieToken) {
    return cookieToken;
  }

  try {
    return await getAdminToken();
  } catch {
    return '';
  }
}

function closeTelemetrySocket(): void {
  if (!socket) {
    return;
  }

  const activeSocket = socket;
  socket = null;
  activeSocket.close();
}

function reconnectTelemetryNow(): void {
  if (consumerCount === 0) {
    return;
  }
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
    return;
  }

  clearReconnectTimer();
  reconnectDelayMs = INITIAL_BACKOFF_MS;
  void connectTelemetry();
}

function bindLifecycleEvents(): void {
  if (lifecycleBound || typeof window === 'undefined') {
    return;
  }

  const handleVisibilityChange = () => {
    if (document.visibilityState === 'visible') {
      reconnectTelemetryNow();
    }
  };

  window.addEventListener('online', reconnectTelemetryNow);
  window.addEventListener('focus', reconnectTelemetryNow);
  document.addEventListener('visibilitychange', handleVisibilityChange);

  lifecycleCleanup = () => {
    window.removeEventListener('online', reconnectTelemetryNow);
    window.removeEventListener('focus', reconnectTelemetryNow);
    document.removeEventListener('visibilitychange', handleVisibilityChange);
  };
  lifecycleBound = true;
}

function unbindLifecycleEvents(): void {
  if (!lifecycleBound) {
    return;
  }

  lifecycleCleanup?.();
  lifecycleCleanup = null;
  lifecycleBound = false;
}

async function connectTelemetry(): Promise<void> {
  if (typeof window === 'undefined' || consumerCount === 0) {
    return;
  }
  if (socket && (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)) {
    return;
  }

  publishState({ connected: false, status: 'CONNECTING' });

  const token = await resolveTelemetryToken();
  if (!token) {
    publishState({ connected: false, status: 'OFFLINE' });
    scheduleReconnect();
    return;
  }

  const url = new URL('/ws/telemetry', WS_BASE);
  url.searchParams.set('token', token);

  const activeSocket = new WebSocket(url.toString());
  socket = activeSocket;

  activeSocket.onopen = () => {
    if (socket !== activeSocket) {
      activeSocket.close();
      return;
    }

    reconnectDelayMs = INITIAL_BACKOFF_MS;
    publishState({ connected: true, status: 'LIVE' });
  };

  activeSocket.onmessage = (event) => {
    if (socket !== activeSocket) {
      return;
    }

    try {
      const parsed = JSON.parse(event.data);
      if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
        return;
      }
      
      // Dispatch hybrid sync event
      if (typeof window !== "undefined" && typeof parsed.type === "string") {
        window.dispatchEvent(new CustomEvent("sync-invalidate", { detail: parsed.type }));
      }

      publishState({ lastSyncTs: new Date().toISOString() });
      broadcastMessage(parsed as TelemetryMessage);
    } catch {
      // Ignore malformed or non-JSON telemetry frames.
    }
  };

  activeSocket.onclose = () => {
    if (socket === activeSocket) {
      socket = null;
    }
    if (consumerCount === 0) {
      publishState({ connected: false, status: 'OFFLINE' });
      return;
    }

    publishState({
      connected: false,
      status: 'OFFLINE',
      reconnectCount: telemetryState.reconnectCount + 1,
    });
    scheduleReconnect();
  };

  activeSocket.onerror = () => {
    activeSocket.close();
  };
}

/**
 * useTelemetry subscribes a surface to the shared supplier telemetry socket so
 * one browser tab reuses one live connection across portal screens.
 */
export function useTelemetry(
  onMessage?: TelemetryListener,
  options?: UseTelemetryOptions,
): TelemetryState {
  const enabled = options?.enabled ?? true;
  const [state, setState] = useState<TelemetryState>(telemetryState);
  const messageRef = useRef<TelemetryListener | undefined>(onMessage);

  useEffect(() => {
    messageRef.current = onMessage;
  }, [onMessage]);

  useEffect(() => {
    if (!enabled) {
      setState((prev) => ({ ...prev, connected: false, status: 'OFFLINE' }));
      return;
    }

    const handleState = (next: TelemetryState) => {
      setState(next);
    };
    const handleMessage = (message: TelemetryMessage) => {
      messageRef.current?.(message);
    };

    stateListeners.add(handleState);
    messageListeners.add(handleMessage);
    consumerCount += 1;
    bindLifecycleEvents();
    setState(telemetryState);
    void connectTelemetry();

    return () => {
      stateListeners.delete(handleState);
      messageListeners.delete(handleMessage);
      consumerCount = Math.max(0, consumerCount - 1);

      if (consumerCount === 0) {
        clearReconnectTimer();
        closeTelemetrySocket();
        unbindLifecycleEvents();
      }
    };
  }, [enabled]);

  if (!enabled) {
    return { ...state, connected: false, status: 'OFFLINE' };
  }

  return state;
}

'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { reconnectDelayMs } from '@pegasusx/api-client';
import { readTokenFromCookie, apiFetch } from './auth';
import { runWarehouseSessionReconcile } from './session-reconcile';
import type { HandoffCardMetadata } from '@pegasusx/types';

const API = (
  process.env.NEXT_PUBLIC_API_URL ||
  process.env.NEXT_PUBLIC_WAREHOUSE_BACKEND_BASE_URL ||
  process.env.NEXT_PUBLIC_BACKEND_BASE_URL ||
  'http://localhost:8180'
).replace(/\/$/, '');

const WAREHOUSE_NOTIFICATIONS_WS_PATH = '/v1/ws';

interface BackendNotification {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}

interface RealtimeNotificationFrame {
  id?: string;
  type?: string;
  title?: string;
  body?: string;
  payload?: string;
  channel?: string;
  created_at?: string;
}

function parseRealtimePayload(raw?: string): Record<string, unknown> {
  const trimmed = (raw || '').trim();
  if (!trimmed) {
    return {};
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Ignore malformed payload JSON and keep a safe empty payload object.
  }
  return {};
}

function normalizeNotification(item: BackendNotification): Notification {
  return {
    id: item.notification_id,
    type: item.type,
    title: item.title,
    body: item.body,
    payload: item.payload || '',
    channel: item.channel || 'PUSH',
    read_at: item.read_at,
    created_at: item.created_at,
    handoff_metadata: item.handoff_metadata,
  };
}

export interface Notification {
  id: string;
  type: string;
  title: string;
  body: string;
  payload: string;
  channel: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}

interface NotificationsState {
  items: Notification[];
  unreadCount: number;
  loading: boolean;
}

export type WarehouseWsState = 'connected' | 'connecting' | 'reconnecting' | 'offline';

export function useNotifications(options?: { enabled?: boolean }) {
  const enabled = options?.enabled !== false;
  const [state, setState] = useState<NotificationsState>({
    items: [],
    unreadCount: 0,
    loading: true,
  });
  const [wsState, setWsState] = useState<WarehouseWsState>('offline');
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const disposedRef = useRef(false);
  const reconnectAttemptRef = useRef(0);
  const hasConnectedOnceRef = useRef(false);

  const fetchAllInboxPages = useCallback(async (signal?: AbortSignal) => {
    const pageSize = 100;
    let offset = 0;
    const items: Notification[] = [];
    let unreadCount = 0;
    let hasMore = true;
    while (hasMore && offset < 2500) {
      const res = await apiFetch(`/v1/user/notifications?limit=${pageSize}&offset=${offset}`, { signal });
      if (!res.ok) {
        return null;
      }
      const data = await res.json();
      const page = Array.isArray(data.notifications)
        ? data.notifications.map((item: BackendNotification) => normalizeNotification(item))
        : [];
      items.push(...page);
      unreadCount = data.unread_count ?? unreadCount;
      hasMore = Boolean(data.has_more);
      offset += pageSize;
    }
    return { items, unreadCount };
  }, []);

  // ── Fetch inbox ──
  const fetchInbox = useCallback(async (signal?: AbortSignal) => {
    if (!enabled) return;
    try {
      const data = await fetchAllInboxPages(signal);
      if (!data) return;
      if (disposedRef.current) return;
      setState({
        items: data.items,
        unreadCount: data.unreadCount,
        loading: false,
      });
    } catch {
      if (disposedRef.current) return;
      setState(s => ({ ...s, loading: false }));
    }
  }, [fetchAllInboxPages, enabled]);

  // ── Mark single notification read ──
  const markRead = useCallback(async (notificationId: string) => {
    try {
      const res = await apiFetch('/v1/user/notifications/read', {
        method: 'POST',
        body: JSON.stringify({ notification_ids: [notificationId] }),
      });
      if (!res.ok) return;

      setState(s => ({
        ...s,
        items: s.items.map(n => n.id === notificationId ? { ...n, read_at: new Date().toISOString() } : n),
        unreadCount: Math.max(0, s.unreadCount - 1),
      }));
    } catch {
      // Preserve the current unread state when the backend write fails.
    }
  }, []);

  // ── Mark all read ──
  const markAllRead = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/user/notifications/read', {
        method: 'POST',
        body: JSON.stringify({ mark_all: true }),
      });
      if (!res.ok) return;

      setState(s => ({
        ...s,
        items: s.items.map(n => ({ ...n, read_at: n.read_at || new Date().toISOString() })),
        unreadCount: 0,
      }));
    } catch {
      // Preserve the current unread state when the backend write fails.
    }
  }, []);

  // ── WebSocket for real-time notifications ──
  const connectWS = useCallback(() => {
    if (!enabled || disposedRef.current) return;
    const token = readTokenFromCookie();
    if (!token) {
      setWsState('offline');
      return;
    }

    clearTimeout(reconnectTimer.current);
    const isReconnect = reconnectAttemptRef.current > 0;
    setWsState(isReconnect ? 'reconnecting' : 'connecting');
    const wsBase = API.replace(/^http/, 'ws');
    const url = new URL(WAREHOUSE_NOTIFICATIONS_WS_PATH, wsBase);
    url.searchParams.set('token', token);
    const ws = new WebSocket(url.toString());
    wsRef.current = ws;

    ws.onopen = () => {
      if (disposedRef.current) return;
      const wasReconnect = hasConnectedOnceRef.current;
      hasConnectedOnceRef.current = true;
      reconnectAttemptRef.current = 0;
      setWsState('connected');
      void fetchInbox();
      if (wasReconnect) {
        void runWarehouseSessionReconcile();
      }
    };

    ws.onmessage = (event) => {
      if (disposedRef.current) return;
      try {
        const msg = JSON.parse(event.data) as RealtimeNotificationFrame;
        
        // Dispatch hybrid sync event globally
        if (msg.type && typeof window !== "undefined") {
           window.dispatchEvent(new CustomEvent("sync-invalidate", { detail: msg.type }));

          const payload = parseRealtimePayload(msg.payload);
          window.dispatchEvent(new CustomEvent("warehouse-live-event", {
            detail: {
              ...payload,
              ...msg,
              type: msg.type,
            },
          }));
        }

        if (msg.type && msg.title) {
          const notif: Notification = {
            id: msg.id || crypto.randomUUID(),
            type: msg.type,
            title: msg.title,
            body: msg.body || '',
            payload: msg.payload || '',
            channel: msg.channel || 'WS',
            read_at: null,
            created_at: msg.created_at || new Date().toISOString(),
          };
          setState(s => ({
            ...s,
            items: s.items.some(existing => existing.id === notif.id)
              ? s.items
              : [notif, ...s.items].slice(0, 100),
            unreadCount: s.items.some(existing => existing.id === notif.id) ? s.unreadCount : s.unreadCount + 1,
            loading: false,
          }));
        }
      } catch { /* ignore malformed */ }
    };

    ws.onclose = () => {
      if (wsRef.current === ws) {
        wsRef.current = null;
      }
      if (disposedRef.current) return;
      reconnectAttemptRef.current += 1;
      setWsState('reconnecting');
      const delay = reconnectDelayMs(reconnectAttemptRef.current - 1, { baseMs: 5_000, maxMs: 60_000 });
      reconnectTimer.current = setTimeout(connectWS, delay);
    };

    ws.onerror = () => ws.close();
  }, [fetchInbox, enabled]);

  // ── Lifecycle ──
  useEffect(() => {
    if (!enabled) {
      disposedRef.current = true;
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
      reconnectAttemptRef.current = 0;
      setWsState('offline');
      return;
    }
    disposedRef.current = false;
    const ac = new AbortController();
    fetchInbox(ac.signal);
    connectWS();

    const reconnectIfNeeded = () => {
      if (wsRef.current && (wsRef.current.readyState === WebSocket.OPEN || wsRef.current.readyState === WebSocket.CONNECTING)) {
        return;
      }
      connectWS();
    };

    const handleWake = () => {
      void fetchInbox();
      if (document.visibilityState !== 'hidden') {
        reconnectIfNeeded();
      }
    };

    const handleOffline = () => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) {
        setWsState('offline');
      }
    };

    const handleVisible = () => {
      if (document.visibilityState === 'visible') {
        handleWake();
      }
    };

    window.addEventListener('online', handleWake);
    window.addEventListener('offline', handleOffline);
    window.addEventListener('focus', handleWake);
    window.addEventListener('pageshow', handleWake);
    document.addEventListener('visibilitychange', handleVisible);
    return () => {
      disposedRef.current = true;
      ac.abort();
      window.removeEventListener('online', handleWake);
      window.removeEventListener('offline', handleOffline);
      window.removeEventListener('focus', handleWake);
      window.removeEventListener('pageshow', handleWake);
      document.removeEventListener('visibilitychange', handleVisible);
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
      reconnectAttemptRef.current = 0;
      setWsState('offline');
    };
  }, [fetchInbox, connectWS, enabled]);

  return { ...state, wsState, fetchInbox, markRead, markAllRead };
}

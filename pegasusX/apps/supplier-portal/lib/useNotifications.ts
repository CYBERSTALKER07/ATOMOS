'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { reconnectDelayMs } from '@pegasusx/api-client';
import { readTokenFromCookie, supplierFetch } from './auth';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const SUPPLIER_NOTIFICATIONS_WS_PATH = '/v1/ws/supplier';

interface BackendNotification {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at: string | null;
  created_at: string;
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
}

interface NotificationsState {
  items: Notification[];
  unreadCount: number;
  loading: boolean;
}

export function useNotifications() {
  const [state, setState] = useState<NotificationsState>({
    items: [],
    unreadCount: 0,
    loading: true,
  });
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const reconnectAttempt = useRef(0);
  const disposedRef = useRef(false);

  const fetchAllInboxPages = useCallback(async (signal?: AbortSignal) => {
    const pageSize = 100;
    let offset = 0;
    const items: Notification[] = [];
    let unreadCount = 0;
    let hasMore = true;
    while (hasMore && offset < 2500) {
      const res = await supplierFetch(`/v1/user/notifications?limit=${pageSize}&offset=${offset}`, { signal });
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
  }, [fetchAllInboxPages]);

  // ── Mark single notification read ──
  const markRead = useCallback(async (notificationId: string) => {
    try {
      const res = await supplierFetch('/v1/user/notifications/read', {
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
      const res = await supplierFetch('/v1/user/notifications/read', {
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
    if (disposedRef.current) return;
    const token = readTokenFromCookie();
    if (!token) return;

    clearTimeout(reconnectTimer.current);
    const wsBase = API.replace(/^http/, 'ws');
    const url = new URL(SUPPLIER_NOTIFICATIONS_WS_PATH, wsBase);
    url.searchParams.set('token', token);
    const ws = new WebSocket(url.toString());
    wsRef.current = ws;

    ws.onopen = () => {
      if (disposedRef.current) return;
      reconnectAttempt.current = 0;
      void fetchInbox();
    };

    ws.onmessage = (event) => {
      if (disposedRef.current) return;
      try {
        const msg = JSON.parse(event.data) as RealtimeNotificationFrame;

        // Dispatch hybrid sync event globally
        if (msg.type && typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("sync-invalidate", { detail: msg.type }));

          const payload = parseRealtimePayload(msg.payload);
          window.dispatchEvent(new CustomEvent("supplier-live-event", {
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
      const delay = reconnectDelayMs(reconnectAttempt.current, { baseMs: 5_000, maxMs: 60_000 });
      reconnectAttempt.current += 1;
      reconnectTimer.current = setTimeout(connectWS, delay);
    };

    ws.onerror = () => ws.close();
  }, [fetchInbox]);

  // ── Lifecycle ──
  useEffect(() => {
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

    const handleVisible = () => {
      if (document.visibilityState === 'visible') {
        handleWake();
      }
    };

    window.addEventListener('online', handleWake);
    window.addEventListener('focus', handleWake);
    window.addEventListener('pageshow', handleWake);
    document.addEventListener('visibilitychange', handleVisible);
    return () => {
      disposedRef.current = true;
      ac.abort();
      window.removeEventListener('online', handleWake);
      window.removeEventListener('focus', handleWake);
      window.removeEventListener('pageshow', handleWake);
      document.removeEventListener('visibilitychange', handleVisible);
      clearTimeout(reconnectTimer.current);
      wsRef.current?.close();
      wsRef.current = null;
    };
  }, [fetchInbox, connectWS]);

  return { ...state, fetchInbox, markRead, markAllRead };
}

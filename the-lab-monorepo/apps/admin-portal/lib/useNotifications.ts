'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { apiFetch, readTokenFromCookie } from './auth';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

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

/**
 * Hook for managing notifications:
 * - Fetches persistent inbox from GET /v1/user/notifications
 * - Connects to /v1/ws/payloader for real-time push
 * - Exposes markRead + markAllRead
 */
export function useNotifications() {
  const [state, setState] = useState<NotificationsState>({
    items: [],
    unreadCount: 0,
    loading: true,
  });
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);

  // ── Fetch inbox ──
  const fetchInbox = useCallback(async (signal?: AbortSignal) => {
    try {
      const res = await apiFetch('/v1/user/notifications?limit=50', { signal });
      if (!res.ok) return;
      const data = await res.json();
      const items = Array.isArray(data.notifications)
        ? data.notifications.map((item: BackendNotification) => normalizeNotification(item))
        : [];
      setState({
        items,
        unreadCount: data.unread_count ?? 0,
        loading: false,
      });
    } catch {
      setState(s => ({ ...s, loading: false }));
    }
  }, []);

  // ── Mark single notification read ──
  const markRead = useCallback(async (notificationId: string) => {
    await apiFetch('/v1/user/notifications/read', {
      method: 'POST',
      body: JSON.stringify({ notification_ids: [notificationId] }),
    });
    setState(s => ({
      ...s,
      items: s.items.map(n => n.id === notificationId ? { ...n, read_at: new Date().toISOString() } : n),
      unreadCount: Math.max(0, s.unreadCount - 1),
    }));
  }, []);

  // ── Mark all read ──
  const markAllRead = useCallback(async () => {
    await apiFetch('/v1/user/notifications/read', {
      method: 'POST',
      body: JSON.stringify({ mark_all: true }),
    });
    setState(s => ({
      ...s,
      items: s.items.map(n => ({ ...n, read_at: n.read_at || new Date().toISOString() })),
      unreadCount: 0,
    }));
  }, []);

  // ── WebSocket for real-time notifications ──
  const connectWS = useCallback(() => {
    const token = readTokenFromCookie();
    if (!token) return;

    const wsBase = API.replace(/^http/, 'ws');
    const ws = new WebSocket(`${wsBase}/v1/ws/payloader?token=${encodeURIComponent(token)}`);
    wsRef.current = ws;

    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type && msg.title) {
          const notif: Notification = {
            id: msg.id || crypto.randomUUID(),
            type: msg.type,
            title: msg.title,
            body: msg.body || '',
            payload: msg.payload || '',
            channel: 'WS',
            read_at: null,
            created_at: new Date().toISOString(),
          };
          setState(s => ({
            items: [notif, ...s.items].slice(0, 100),
            unreadCount: s.unreadCount + 1,
            loading: false,
          }));
        }
      } catch { /* ignore malformed */ }
    };

    ws.onclose = () => {
      wsRef.current = null;
      reconnectTimer.current = setTimeout(connectWS, 5000);
    };

    ws.onerror = () => ws.close();
  }, []);

  // ── Lifecycle ──
  useEffect(() => {
    const ac = new AbortController();
    fetchInbox(ac.signal);
    connectWS();
    return () => {
      ac.abort();
      wsRef.current?.close();
      clearTimeout(reconnectTimer.current);
    };
  }, [fetchInbox, connectWS]);

  return { ...state, fetchInbox, markRead, markAllRead };
}

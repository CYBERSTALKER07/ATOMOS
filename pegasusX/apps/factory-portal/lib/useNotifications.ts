'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { apiFetch, readTokenFromCookie } from './auth';

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

export function useNotifications(options?: { enabled?: boolean }) {
  const enabled = options?.enabled !== false;
  const [state, setState] = useState<NotificationsState>({
    items: [],
    unreadCount: 0,
    loading: true,
  });
  const disposedRef = useRef(false);

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

  const fetchInbox = useCallback(async (signal?: AbortSignal) => {
    if (!enabled || !readTokenFromCookie()) {
      if (!disposedRef.current) {
        setState({ items: [], unreadCount: 0, loading: false });
      }
      return;
    }
    try {
      const data = await fetchAllInboxPages(signal);
      if (!data || disposedRef.current) return;
      setState({
        items: data.items,
        unreadCount: data.unreadCount,
        loading: false,
      });
    } catch {
      if (!disposedRef.current) {
        setState(s => ({ ...s, loading: false }));
      }
    }
  }, [fetchAllInboxPages, enabled]);

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
      // Preserve unread state when the backend write fails.
    }
  }, []);

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
      // Preserve unread state when the backend write fails.
    }
  }, []);

  useEffect(() => {
    if (!enabled) {
      disposedRef.current = true;
      setState({ items: [], unreadCount: 0, loading: false });
      return;
    }
    disposedRef.current = false;
    const ac = new AbortController();
    void fetchInbox(ac.signal);

    const handleWake = () => {
      if (document.visibilityState === 'hidden') return;
      void fetchInbox();
    };

    window.addEventListener('online', handleWake);
    window.addEventListener('focus', handleWake);
    window.addEventListener('pageshow', handleWake);
    document.addEventListener('visibilitychange', handleWake);

    return () => {
      disposedRef.current = true;
      ac.abort();
      window.removeEventListener('online', handleWake);
      window.removeEventListener('focus', handleWake);
      window.removeEventListener('pageshow', handleWake);
      document.removeEventListener('visibilitychange', handleWake);
    };
  }, [fetchInbox, enabled]);

  return { ...state, fetchInbox, markRead, markAllRead };
}

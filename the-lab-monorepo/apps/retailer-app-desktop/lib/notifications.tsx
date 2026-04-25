'use client';

import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { apiFetch } from './auth';
import { useWebSocket } from './ws';
import {
  normalizeNotification,
  shouldRefreshNotificationInbox,
  type BackendNotificationItem,
  type RetailerNotificationItem,
} from './notifications-core';

type BackendNotificationsResponse = {
  notifications?: BackendNotificationItem[];
  unread_count?: number;
};

type NotificationsContextType = {
  items: RetailerNotificationItem[];
  unreadCount: number;
  loading: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  markRead: (notificationId: string) => Promise<void>;
  markAllRead: () => Promise<void>;
};

const NotificationsContext = createContext<NotificationsContextType | undefined>(undefined);

export function NotificationsProvider({ children }: { children: React.ReactNode }) {
  const { lastMessage } = useWebSocket();
  const [items, setItems] = useState<RetailerNotificationItem[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const refresh = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/user/notifications?limit=50');
      if (!res.ok) {
        throw new Error(`Notifications fetch failed with ${res.status}`);
      }

      const data = (await res.json()) as BackendNotificationsResponse;
      setItems((data.notifications ?? []).map(normalizeNotification));
      setUnreadCount(data.unread_count ?? 0);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load notifications');
    } finally {
      setLoading(false);
    }
  }, []);

  const markRead = useCallback(async (notificationId: string) => {
    const target = items.find((item) => item.id === notificationId);
    if (!target || target.readAt) {
      return;
    }

    const res = await apiFetch('/v1/user/notifications/read', {
      method: 'POST',
      body: JSON.stringify({ notification_ids: [notificationId] }),
    });
    if (!res.ok) {
      throw new Error(`Mark read failed with ${res.status}`);
    }

    setItems((current) => current.map((item) => (
      item.id === notificationId ? { ...item, readAt: item.readAt ?? new Date().toISOString() } : item
    )));
    setUnreadCount((current) => Math.max(0, current - 1));
  }, [items]);

  const markAllRead = useCallback(async () => {
    if (unreadCount === 0) {
      return;
    }

    const res = await apiFetch('/v1/user/notifications/read', {
      method: 'POST',
      body: JSON.stringify({ mark_all: true }),
    });
    if (!res.ok) {
      throw new Error(`Mark all read failed with ${res.status}`);
    }

    const now = new Date().toISOString();
    setItems((current) => current.map((item) => ({ ...item, readAt: item.readAt ?? now })));
    setUnreadCount(0);
  }, [unreadCount]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  useEffect(() => {
    if (!shouldRefreshNotificationInbox(lastMessage)) {
      return;
    }
    void refresh();
  }, [lastMessage, refresh]);

  return (
    <NotificationsContext.Provider value={{ items, unreadCount, loading, error, refresh, markRead, markAllRead }}>
      {children}
    </NotificationsContext.Provider>
  );
}

export function useRetailerNotifications() {
  const ctx = useContext(NotificationsContext);
  if (!ctx) {
    throw new Error('useRetailerNotifications must be used within a NotificationsProvider');
  }
  return ctx;
}
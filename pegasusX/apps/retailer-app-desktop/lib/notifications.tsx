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

const INBOX_PAGE_SIZE = 50;

type BackendNotificationsResponse = {
  notifications?: BackendNotificationItem[];
  unread_count?: number;
  has_more?: boolean;
};

async function fetchNotificationPage(offset: number): Promise<BackendNotificationsResponse> {
  const res = await apiFetch(
    `/v1/user/notifications?limit=${INBOX_PAGE_SIZE}&offset=${offset}`,
  );
  if (!res.ok) {
    throw new Error(`Notifications fetch failed with ${res.status}`);
  }
  return (await res.json()) as BackendNotificationsResponse;
}

type NotificationsContextType = {
  items: RetailerNotificationItem[];
  unreadCount: number;
  loading: boolean;
  isLoadingMore: boolean;
  hasMore: boolean;
  error: string | null;
  refresh: () => Promise<void>;
  loadMore: () => Promise<void>;
  markRead: (notificationId: string) => Promise<void>;
  markAllRead: () => Promise<void>;
};

const NotificationsContext = createContext<NotificationsContextType | undefined>(undefined);

export function NotificationsProvider({ children }: { children: React.ReactNode }) {
  const { lastMessage } = useWebSocket();
  const [items, setItems] = useState<RetailerNotificationItem[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [loading, setLoading] = useState(true);
  const [isLoadingMore, setIsLoadingMore] = useState(false);
  const [hasMore, setHasMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [nextOffset, setNextOffset] = useState(0);

  const refresh = useCallback(async () => {
    setLoading(true);
    try {
      const data = await fetchNotificationPage(0);
      const pageItems = (data.notifications ?? []).map(normalizeNotification);
      setItems(pageItems);
      setUnreadCount(data.unread_count ?? 0);
      setHasMore(Boolean(data.has_more));
      setNextOffset(pageItems.length);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load notifications');
    } finally {
      setLoading(false);
    }
  }, []);

  const loadMore = useCallback(async () => {
    if (loading || isLoadingMore || !hasMore) {
      return;
    }
    setIsLoadingMore(true);
    try {
      const data = await fetchNotificationPage(nextOffset);
      const pageItems = (data.notifications ?? []).map(normalizeNotification);
      setItems((current) => [...current, ...pageItems]);
      setUnreadCount(data.unread_count ?? unreadCount);
      setHasMore(Boolean(data.has_more));
      setNextOffset((current) => current + pageItems.length);
      setError(null);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : 'Failed to load more notifications');
    } finally {
      setIsLoadingMore(false);
    }
  }, [hasMore, isLoadingMore, loading, nextOffset, unreadCount]);

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
    <NotificationsContext.Provider
      value={{
        items,
        unreadCount,
        loading,
        isLoadingMore,
        hasMore,
        error,
        refresh,
        loadMore,
        markRead,
        markAllRead,
      }}
    >
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

import { useCallback, useState } from 'react';

import { authFetch } from '../authSession';
import type { NotifItem } from '../components/NotificationsSheet';

// ─── Types ────────────────────────────────────────────────────────────────────

export type BackendNotifItem = {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  read_at: string | null;
  created_at: string;
};

export type LiveNotifFrame = {
  type?: string;
  title?: string;
  body?: string;
  channel?: string;
  manifest_id?: string;
  warehouse_id?: string;
  reason?: string;
  timestamp?: string;
};

const normalizeNotification = (item: BackendNotifItem): NotifItem => ({
  id: item.notification_id,
  type: item.type,
  title: item.title,
  body: item.body,
  read_at: item.read_at,
  created_at: item.created_at,
});

// ─── Hook ─────────────────────────────────────────────────────────────────────

export function useNotifications({ token }: { token: string | null }) {
  // Notification state
  const [notifications, setNotifications] = useState<NotifItem[]>([]);
  const [unreadCount, setUnreadCount] = useState(0);
  const [showNotifPanel, setShowNotifPanel] = useState(false);

  // ── Notifications: fetch ───────────────────────────────────────────────
  const fetchNotifications = useCallback(async () => {
    if (!token) return;
    try {
      const pageSize = 100;
      let offset = 0;
      const items: ReturnType<typeof normalizeNotification>[] = [];
      let unreadCount = 0;
      let hasMore = true;
      while (hasMore && offset < 2500) {
        const res = await authFetch(`/v1/user/notifications?limit=${pageSize}&offset=${offset}`);
        if (!res.ok) return;
        const data = await res.json();
        const page = Array.isArray(data.notifications)
          ? data.notifications.map((item: BackendNotifItem) => normalizeNotification(item))
          : [];
        items.push(...page);
        unreadCount = data.unread_count ?? unreadCount;
        hasMore = Boolean(data.has_more);
        offset += pageSize;
      }
      setNotifications(items);
      setUnreadCount(unreadCount);
    } catch {}
  }, [token]);

  const markNotifRead = useCallback(async (id: string) => {
    if (!token) return;
    setNotifications(prev => prev.map(n => n.id === id ? { ...n, read_at: new Date().toISOString() } : n));
    setUnreadCount(prev => Math.max(0, prev - 1));
    try {
      await authFetch('/v1/user/notifications/read', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ notification_ids: [id] }),
      });
    } catch {}
  }, [token]);

  const markAllNotifsRead = useCallback(async () => {
    if (!token) return;
    setNotifications(prev => prev.map(n => ({ ...n, read_at: n.read_at || new Date().toISOString() })));
    setUnreadCount(0);
    try {
      await authFetch('/v1/user/notifications/read', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ mark_all: true }),
      });
    } catch {}
  }, [token]);

  return {
    notifications,
    setNotifications,
    unreadCount,
    setUnreadCount,
    showNotifPanel,
    setShowNotifPanel,
    fetchNotifications,
    markNotifRead,
    markAllNotifsRead,
  };
}

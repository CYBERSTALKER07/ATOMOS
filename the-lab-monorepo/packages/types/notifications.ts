/**
 * @file packages/types/notifications.ts
 * @description Notification inbox and broadcast types.
 * Sync with: apps/backend-go/notifications/
 */

// ─── Notification Inbox ─────────────────────────────────────────────────────
export interface NotificationItem {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload: string | null;
  channel: string;
  read_at: string | null;
  created_at: string;
}

export interface NotificationInboxResponse {
  notifications: NotificationItem[];
  unread_count: number;
}

// ─── Broadcast Request ──────────────────────────────────────────────────────
export interface BroadcastRequest {
  title: string;
  body: string;
  role: 'RETAILER' | 'DRIVER' | 'ALL';
  data?: Record<string, string>;
}

export interface BroadcastResponse {
  status: 'broadcast_complete';
  recipients: number;
  fcm_sent: number;
  fcm_failed: number;
}

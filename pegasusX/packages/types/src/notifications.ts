import { HandoffCardMetadata } from "./primitives";

// ── Notification inbox (GET /v1/user/notifications) ───────────────────────────
export interface NotificationInboxItem {
  id: string;
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel: string;
  read_at?: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}

export interface NotificationInboxResponse {
  notifications: NotificationInboxItem[];
  unread_count: number;
  limit?: number;
  offset?: number;
  has_more?: boolean;
}

export interface MarkNotificationsReadRequest {
  notification_ids?: string[];
  mark_all?: boolean;
}


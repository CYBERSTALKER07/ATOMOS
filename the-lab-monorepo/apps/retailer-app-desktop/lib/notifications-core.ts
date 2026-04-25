import type { WsMessage } from './ws';

export type BackendNotificationItem = {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at?: string | null;
  created_at: string;
};

export type RetailerNotificationItem = {
  id: string;
  type: string;
  title: string;
  body: string;
  payload: string;
  channel: string;
  readAt: string | null;
  createdAt: string;
};

const retailerNotificationEventTypes = new Set([
  'ORDER_DISPATCHED',
  'DRIVER_ARRIVED',
  'ORDER_STATUS_CHANGED',
  'PAYMENT_SETTLED',
  'PAYMENT_FAILED',
  'ORDER_MODIFIED',
  'OUT_OF_STOCK',
  'RETAILER_PRICE_OVERRIDE',
  'CANCEL_APPROVED',
  'ORDER_COMPLETED',
  'NEGOTIATION_PROPOSED',
  'PRE_ORDER_AUTO_ACCEPTED',
  'PRE_ORDER_CONFIRMED',
  'PRE_ORDER_EDITED',
]);

export function normalizeNotification(item: BackendNotificationItem): RetailerNotificationItem {
  return {
    id: item.notification_id,
    type: item.type,
    title: item.title,
    body: item.body,
    payload: item.payload ?? '',
    channel: item.channel ?? 'PUSH',
    readAt: item.read_at ?? null,
    createdAt: item.created_at,
  };
}

export function shouldRefreshNotificationInbox(message: WsMessage | null): boolean {
  if (!message || typeof message.type !== 'string') {
    return false;
  }
  return retailerNotificationEventTypes.has(message.type);
}
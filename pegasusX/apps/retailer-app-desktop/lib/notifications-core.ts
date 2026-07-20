import type { WsMessage } from './ws';

export type BackendNotificationItem = {
  id?: string;
  notification_id?: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at?: string | null;
  created_at: string;
  handoff_metadata?: import('@pegasusx/types').HandoffCardMetadata;
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
  handoffMetadata?: import('@pegasusx/types').HandoffCardMetadata;
};

const retailerNotificationEventTypes = new Set([
  'ORDER_DISPATCHED',
  'DRIVER_ARRIVED',
  'ORDER_STATUS_CHANGED',
  'SETTLEMENT_REQUIRED',
  'DELIVERY_SESSION_UPDATED',
  'PAYMENT_SETTLED',
  'PAYMENT_FAILED',
  'PAYMENT_GATEWAY_DEGRADED',
  'ORDER_MODIFIED',
  'ORDER_REASSIGNED',
  'OUT_OF_STOCK',
  'RETAILER_PRICE_OVERRIDE',
  'CANCEL_APPROVED',
  'ORDER_COMPLETED',
  'FISCAL_RECEIPT_SUCCEEDED',
  'FISCAL_RECEIPT_FAILED',
  'FISCAL_RECEIPT_REQUESTED',
  // 'NEGOTIATION_PROPOSED', // quantity negotiation disabled
  'PRE_ORDER_AUTO_ACCEPTED',
  'PRE_ORDER_CONFIRMED',
  'PRE_ORDER_EDITED',
  'PRE_ORDER_DATE_PROPOSED',
  'PRE_ORDER_DATE_ACCEPTED',
  'PRE_ORDER_DATE_REJECTED',
  'PRE_ORDER_CANCELLED',
  'PRE_ORDER_NUDGE',
  'PRE_ORDER_CONFIRMATION',
]);

export function normalizeNotification(item: BackendNotificationItem): RetailerNotificationItem {
  const id = item.id ?? item.notification_id ?? "";
  return {
    id,
    type: item.type,
    title: item.title,
    body: item.body,
    payload: item.payload ?? '',
    channel: item.channel ?? 'PUSH',
    readAt: item.read_at ?? null,
    createdAt: item.created_at,
    handoffMetadata: item.handoff_metadata,
  };
}

export function shouldRefreshNotificationInbox(message: WsMessage | null): boolean {
  if (!message || typeof message.type !== 'string') {
    return false;
  }
  return retailerNotificationEventTypes.has(message.type);
}
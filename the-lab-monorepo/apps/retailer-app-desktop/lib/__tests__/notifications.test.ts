import { describe, expect, it } from 'vitest';
import { normalizeNotification, shouldRefreshNotificationInbox } from '../../lib/notifications-core';

describe('normalizeNotification', () => {
  it('maps backend inbox payload into desktop notification shape', () => {
    expect(normalizeNotification({
      notification_id: 'notif-1',
      type: 'ORDER_COMPLETED',
      title: 'Delivery Complete',
      body: 'Order ABC has been delivered.',
      created_at: '2026-04-24T10:00:00Z',
      read_at: null,
    })).toEqual({
      id: 'notif-1',
      type: 'ORDER_COMPLETED',
      title: 'Delivery Complete',
      body: 'Order ABC has been delivered.',
      payload: '',
      channel: 'PUSH',
      readAt: null,
      createdAt: '2026-04-24T10:00:00Z',
    });
  });
});

describe('shouldRefreshNotificationInbox', () => {
  it('refreshes on persisted retailer notification events', () => {
    expect(shouldRefreshNotificationInbox({ type: 'ORDER_STATUS_CHANGED' })).toBe(true);
  });

  it('ignores unrelated retailer websocket payloads', () => {
    expect(shouldRefreshNotificationInbox({ type: 'TRACKING_POSITION' })).toBe(false);
    expect(shouldRefreshNotificationInbox(null)).toBe(false);
  });
});
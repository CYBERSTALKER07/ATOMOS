'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { reconnectDelayMs } from '@pegasusx/api-core';
import { readTokenFromCookie, resolveSupplierToken, supplierFetch } from './auth';
import { runSupplierSessionReconcile } from './session-reconcile';
import type { HandoffCardMetadata } from '@pegasusx/types';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';
const SUPPLIER_NOTIFICATIONS_SSE_PATH = '/v1/supplier/events';

export const SUPPLIER_SSE_EVENT_TYPES: string[] = [
  'notification',
  'ORDER_CREATED',
  'ORDER_UPDATED',
  'ORDER_STATUS_CHANGED',
  'ORDER_CANCELLED',
  'ORDER_DELIVERED',
  'CLAIM_CREATED',
  'CLAIM_UPDATED',
  'PAYMENT_CLEARED',
  'PAYMENT_FAILED',
  'INVENTORY_UPDATED',
  'LOW_STOCK_ALERT',
  'EXCEPTION_CREATED',
  'EXCEPTION_RESOLVED',
  'PLAYBOOK_RUN_UPDATED',
  'FISCAL_RECEIPT_ISSUED',
  'FISCAL_RECEIPT_FAILED',
  'SUPPLY_REQUEST_UPDATED',
  'CASHIER_SESSION_STARTED',
  'CASHIER_SESSION_TRANSITION',
  'HANDOFF_CARD',
  'order.created',
  'order.updated',
  'order.status_changed',
  'SUPPLIER_CREATED',
  'SUPPLIER_UPDATED',
  'SUPPLIER_PROFILE_UPDATED',
  'SUPPLIER_BILLING_UPDATED',
  'SUPPLIER_BILLING_CONFIGURED',
  'SUPPLIER_MEMBER_ADDED',
  'SUPPLIER_BROADCAST',
  'WAREHOUSE_SUPPLY_REQUEST_OPENED',
  'SUPPLY_REQUEST_ACCEPTED',
  'SUPPLY_REQUEST_UPDATE',
  'INVENTORY_QUANTITY_UPDATED',
  'INVENTORY_POLICY_UPDATED',
  'CLAIM_FILED',
  'CLAIM_UNDER_REVIEW',
  'CLAIM_RESOLVED',
  'FISCAL_RECEIPT_REQUESTED',
  'FISCAL_RECEIPT_SUCCEEDED',
  'CONTROL_TOWER_RUN_CREATED',
  'CONTROL_TOWER_RUN_UPDATED',
];

interface BackendNotification {
  notification_id: string;
  type: string;
  title: string;
  body: string;
  payload?: string;
  channel?: string;
  read_at: string | null;
  created_at: string;
  handoff_metadata?: HandoffCardMetadata;
}

interface RealtimeNotificationFrame {
  id?: string;
  type?: string;
  title?: string;
  body?: string;
  payload?: string;
  channel?: string;
  created_at?: string;
}

function parseRealtimePayload(raw?: string): Record<string, unknown> {
  const trimmed = (raw || '').trim();
  if (!trimmed) {
    return {};
  }
  try {
    const parsed = JSON.parse(trimmed) as unknown;
    if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
      return parsed as Record<string, unknown>;
    }
  } catch {
    // Ignore malformed payload JSON and keep a safe empty payload object.
  }
  return {};
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
    handoff_metadata: item.handoff_metadata,
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
  handoff_metadata?: HandoffCardMetadata;
}

interface NotificationsState {
  items: Notification[];
  unreadCount: number;
  loading: boolean;
}

export function useNotifications() {
  const [state, setState] = useState<NotificationsState>({
    items: [],
    unreadCount: 0,
    loading: true,
  });
  const esRef = useRef<EventSource | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout>>(undefined);
  const reconnectAttempt = useRef(0);
  const hasConnectedOnce = useRef(false);
  const disposedRef = useRef(false);
  const processedEventIds = useRef<Set<string>>(new Set());

  const fetchAllInboxPages = useCallback(async (signal?: AbortSignal) => {
    const pageSize = 100;
    let offset = 0;
    const items: Notification[] = [];
    let unreadCount = 0;
    let hasMore = true;
    while (hasMore && offset < 2500) {
      const res = await supplierFetch(`/v1/user/notifications?limit=${pageSize}&offset=${offset}`, { signal });
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

  // ── Fetch inbox ──
  const fetchInbox = useCallback(async (signal?: AbortSignal) => {
    try {
      const data = await fetchAllInboxPages(signal);
      if (!data) return;
      if (disposedRef.current) return;
      setState({
        items: data.items,
        unreadCount: data.unreadCount,
        loading: false,
      });
    } catch {
      if (disposedRef.current) return;
      setState(s => ({ ...s, loading: false }));
    }
  }, [fetchAllInboxPages]);

  // ── Mark single notification read ──
  const markRead = useCallback(async (notificationId: string) => {
    try {
      const res = await supplierFetch('/v1/user/notifications/read', {
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
      // Preserve the current unread state when the backend write fails.
    }
  }, []);

  // ── Mark all read ──
  const markAllRead = useCallback(async () => {
    try {
      const res = await supplierFetch('/v1/user/notifications/read', {
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
      // Preserve the current unread state when the backend write fails.
    }
  }, []);

  // ── SSE for real-time notifications ──
  const connectSSE = useCallback((tokenOverride?: string) => {
    void (async () => {
      if (disposedRef.current) return;
      const authToken = (tokenOverride || readTokenFromCookie() || (await resolveSupplierToken())).trim();
      if (disposedRef.current || !authToken) return;

      let wsTicket = '';
      try {
        const wsRes = await supplierFetch('/v1/supplier/ws-session', {
          method: 'GET',
          headers: tokenOverride ? { Authorization: `Bearer ${authToken}` } : undefined,
          cache: 'no-store',
        });
        const payload = (await wsRes.json().catch(() => null)) as { token?: string } | null;
        if (wsRes.ok && typeof payload?.token === 'string' && payload.token.trim()) {
          wsTicket = payload.token.trim();
        }
      } catch {
        wsTicket = '';
      }
      if (disposedRef.current || !wsTicket) return;

      clearTimeout(reconnectTimer.current);
      if (esRef.current) {
        esRef.current.close();
        esRef.current = null;
      }

      const sseUrl = new URL(SUPPLIER_NOTIFICATIONS_SSE_PATH, API);
      sseUrl.searchParams.set('token', wsTicket);
      const es = new EventSource(sseUrl.toString(), { withCredentials: true });
      if (disposedRef.current) {
        es.close();
        return;
      }
      esRef.current = es;

    es.onopen = () => {
      if (disposedRef.current) return;
      const wasReconnect = hasConnectedOnce.current;
      hasConnectedOnce.current = true;
      reconnectAttempt.current = 0;
      void fetchInbox();
      if (wasReconnect) {
        void runSupplierSessionReconcile();
      }
    };

    const handleSSEMessage = (event: MessageEvent) => {
      if (disposedRef.current) return;
      try {
        const msg = JSON.parse(event.data) as RealtimeNotificationFrame;
        const rawId = (event.lastEventId || msg.id || '').trim();
        const eventType = msg.type || (event.type && event.type !== 'message' ? event.type : undefined);
        const dedupeKey = rawId || (eventType ? `${eventType}:${event.data}` : '');

        if (dedupeKey) {
          if (processedEventIds.current.has(dedupeKey)) {
            return;
          }
          processedEventIds.current.add(dedupeKey);
          if (processedEventIds.current.size > 2000) {
            const first = processedEventIds.current.values().next().value;
            if (first) processedEventIds.current.delete(first);
          }
        }

        // Dispatch hybrid sync event globally
        if (eventType && typeof window !== "undefined") {
          window.dispatchEvent(new CustomEvent("sync-invalidate", { detail: eventType }));

          const payload = parseRealtimePayload(msg.payload);
          window.dispatchEvent(new CustomEvent("supplier-live-event", {
            detail: {
              ...payload,
              ...msg,
              type: eventType,
            },
          }));
        }

        if (eventType && msg.title) {
          const notifId = rawId || crypto.randomUUID();
          const notif: Notification = {
            id: notifId,
            type: eventType,
            title: msg.title,
            body: msg.body || '',
            payload: msg.payload || '',
            channel: msg.channel || 'SSE',
            read_at: null,
            created_at: msg.created_at || new Date().toISOString(),
          };
          setState(s => ({
            ...s,
            items: s.items.some(existing => existing.id === notif.id)
              ? s.items
              : [notif, ...s.items].slice(0, 100),
            unreadCount: s.items.some(existing => existing.id === notif.id) ? s.unreadCount : s.unreadCount + 1,
            loading: false,
          }));
        }
      } catch { /* ignore malformed */ }
    };

    es.onmessage = handleSSEMessage;
    for (const evtName of SUPPLIER_SSE_EVENT_TYPES) {
      es.addEventListener(evtName, handleSSEMessage as EventListener);
    }

    es.onerror = () => {
      // EventSource reconnects natively. If closed, retry.
      if (es.readyState === EventSource.CLOSED && !disposedRef.current) {
        const delay = reconnectDelayMs(reconnectAttempt.current, { baseMs: 3_000, maxMs: 30_000 });
        reconnectAttempt.current += 1;
        reconnectTimer.current = setTimeout(() => { connectSSE(); }, delay);
      }
    };
    })();
  }, [fetchInbox]);

  // ── Lifecycle ──
  useEffect(() => {
    disposedRef.current = false;
    const ac = new AbortController();
    let cancelled = false;

    void (async () => {
      const token = await resolveSupplierToken();
      if (cancelled || !token) {
        setState((s) => ({ ...s, loading: false }));
        return;
      }
      void fetchInbox(ac.signal);
      connectSSE(token);
    })();

    const reconnectIfNeeded = () => {
      if (!hasConnectedOnce.current && !readTokenFromCookie()) {
        return;
      }
      if (esRef.current && (esRef.current.readyState === EventSource.OPEN || esRef.current.readyState === EventSource.CONNECTING)) {
        return;
      }
      connectSSE();
    };

    const handleWake = () => {
      void fetchInbox();
      if (document.visibilityState !== 'hidden') {
        reconnectIfNeeded();
      }
    };

    const handleVisible = () => {
      if (document.visibilityState === 'visible') {
        handleWake();
      }
    };

    window.addEventListener('online', handleWake);
    window.addEventListener('focus', handleWake);
    window.addEventListener('pageshow', handleWake);
    document.addEventListener('visibilitychange', handleVisible);
    return () => {
      cancelled = true;
      disposedRef.current = true;
      ac.abort();
      window.removeEventListener('online', handleWake);
      window.removeEventListener('focus', handleWake);
      window.removeEventListener('pageshow', handleWake);
      document.removeEventListener('visibilitychange', handleVisible);
      clearTimeout(reconnectTimer.current);
      esRef.current?.close();
      esRef.current = null;
    };
  }, [fetchInbox, connectSSE]);

  return { ...state, fetchInbox, markRead, markAllRead };
}

'use client';

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { useToken } from '@/lib/auth';
import { readTokenFromCookie } from '@/lib/auth';
import { usePolling } from '@/lib/usePolling';
import { isTauri } from '@/lib/bridge';
import type { PaginationState } from '@/lib/usePagination';
import PaginationControls from '@/components/PaginationControls';
import StatusBadge from '@/components/StatusBadge';
import EmptyState from '@/components/EmptyState';
import { Skeleton } from '@/components/Skeleton';
import Dialog from '@/components/Dialog';
import Drawer from '@/components/Drawer';
import Icon from '@/components/Icon';
import { useToast } from '@/components/Toast';
import ShopClosedBanner from '@/components/ShopClosedBanner';
import EarlyCompleteBanner from '@/components/EarlyCompleteBanner';
import NegotiationBanner from '@/components/NegotiationBanner';
import { Button } from '@heroui/react';
import {
  buildSupplierAutoDispatchIdempotencyKey,
  buildSupplierApproveCancelIdempotencyKey,
  buildSupplierFleetReassignIdempotencyKey,
  buildSupplierResolveCreditIdempotencyKey,
} from '../_shared/idempotency';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

/* ─── Types ───────────────────────────────────────────────── */

interface Order {
  order_id: string;
  retailer_id: string;
  retailer_name: string;
  supplier_id: string;
  amount: number;
  item_count: number;
  state: string;
  order_source: string;
  created_at: string;
  route_id?: string;
  requested_delivery_date?: string;
  payment_gateway?: string;
  payment_status?: string;
}

interface CapacityInfo {
  route_id: string;
  max_volume_vu: number;
  used_volume_vu: number;
  free_volume_vu: number;
}

interface OrderEvent {
  event_id: string;
  event_type: string;
  actor_role: string;
  actor_id: string;
  metadata?: string;
  created_at: string;
}

interface TruckOption {
  driver_id: string;
  driver_name: string;
  vehicle_class: string;
  plate_number: string;
}

interface SupplierDriverOption {
  driver_id: string;
  name: string;
  vehicle_class?: string;
  vehicle_type?: string;
  license_plate?: string;
  vehicle_id?: string;
  is_active: boolean;
}

type Tab = 'active' | 'scheduled';

const TAB_META: { key: Tab; label: string; icon: string }[] = [
  { key: 'active', label: 'Active', icon: 'orders' },
  { key: 'scheduled', label: 'Scheduled', icon: 'schedule' },
];

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('en-US').format(amount);
}

function shortId(id: string): string {
  return id.slice(0, 12) + '…';
}

function buildOrderVettingIdempotencyKey(orderId: string, decision: 'APPROVED' | 'REJECTED', reason?: string): string {
  return ['supplier-order-vet', orderId.trim(), decision.trim().toUpperCase(), (reason || '').trim()].join(':');
}

function normalizeOrderListResponse(payload: unknown, pageSize: number): { items: Order[]; hasMore: boolean } {
  if (Array.isArray(payload)) {
    return {
      items: payload as Order[],
      hasMore: payload.length === pageSize,
    };
  }

  if (payload && typeof payload === 'object') {
    const record = payload as { data?: Order[]; has_more?: boolean };
    const items = Array.isArray(record.data) ? record.data : [];
    return {
      items,
      hasMore: typeof record.has_more === 'boolean' ? record.has_more : items.length === pageSize,
    };
  }

  return { items: [], hasMore: false };
}

/* ─── Main Page ───────────────────────────────────────────── */

export default function OrdersPage() {
  const token = useToken();
  const { toast } = useToast();

  const [tab, setTab] = useState<Tab>('active');
  const [showHistory, setShowHistory] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [stateFilter, setStateFilter] = useState('');
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [page, setPage] = useState(1);
  const [pageSize, setPageSize] = useState(30);
  const [hasMore, setHasMore] = useState(false);

  // Action state
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [rejectingId, setRejectingId] = useState<string | null>(null);
  const [rejectReason, setRejectReason] = useState('');

  // Selection state for bulk ops
  const [selected, setSelected] = useState<Set<string>>(new Set());

  // Bulk dispatch
  const [dispatching, setDispatching] = useState(false);

  // Reassignment modal
  const [reassignOpen, setReassignOpen] = useState(false);
  const [reassignOrderIds, setReassignOrderIds] = useState<string[]>([]);
  const [trucks, setTrucks] = useState<TruckOption[]>([]);
  const [targetTruck, setTargetTruck] = useState('');
  const [targetCapacity, setTargetCapacity] = useState<CapacityInfo | null>(null);
  const [reassigning, setReassigning] = useState(false);

  // Order detail drawer
  const [detailOrder, setDetailOrder] = useState<Order | null>(null);

  const serverOffset = (page - 1) * pageSize;

  /* ─── Data Fetching ─────────────────────────────────────── */

  const fetchOrders = useCallback(async () => {
    if (!token) return;
    setLoading(true);
    try {
      const params = new URLSearchParams({
        limit: String(pageSize),
        offset: String(serverOffset),
      });
      if (showHistory) {
        params.set('bucket', 'history');
      } else if (tab === 'active') {
        params.set('states', 'PENDING,LOADED,DISPATCHED,IN_TRANSIT,ARRIVED,ARRIVED_SHOP_CLOSED,AWAITING_PAYMENT,PENDING_CASH_COLLECTION,CANCEL_REQUESTED,NO_CAPACITY');
      } else {
        params.set('bucket', 'scheduled');
      }
      const res = await fetch(`${API}/v1/supplier/orders?${params}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error('Failed to load orders');
      const json = await res.json();
      const normalized = normalizeOrderListResponse(json, pageSize);
      setOrders(normalized.items);
      setHasMore(normalized.hasMore);
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setLoading(false);
    }
  }, [token, tab, showHistory, toast, pageSize, serverOffset]);

  useEffect(() => {
    fetchOrders();
    setSelected(new Set());
  }, [fetchOrders]);

  // Auto-refresh every 30s for active tabs
  usePolling(async (signal) => {
    if (showHistory || !token) return;
    try {
      const params = new URLSearchParams({
        limit: String(pageSize),
        offset: String(serverOffset),
      });
      if (tab === 'active') {
        params.set('states', 'PENDING,LOADED,DISPATCHED,IN_TRANSIT,ARRIVED,ARRIVED_SHOP_CLOSED,AWAITING_PAYMENT,PENDING_CASH_COLLECTION,CANCEL_REQUESTED,NO_CAPACITY,QUARANTINE,DELIVERED_ON_CREDIT');
      } else {
        params.set('bucket', 'scheduled');
      }
      const res = await fetch(`${API}/v1/supplier/orders?${params}`, {
        headers: { Authorization: `Bearer ${token}` }, signal,
      });
      if (!res.ok) return;
      const json = await res.json();
      const normalized = normalizeOrderListResponse(json, pageSize);
      setOrders(normalized.items);
      setHasMore(normalized.hasMore);
    } catch (err) {
      if ((err as Error).name === 'AbortError') return;
    }
  }, 30_000, [tab, showHistory, token, pageSize, serverOffset]);

  // ── WebSocket: instant order state change notifications ──────────────────
  const fetchOrdersRef = useRef(fetchOrders);
  fetchOrdersRef.current = fetchOrders;

  useEffect(() => {
    if (isTauri()) return; // Fleet page handles it via Tauri bridge

    let disposed = false;
    let reconnectTimer: ReturnType<typeof setTimeout> | null = null;
    let backoff = 1000;

    const connect = () => {
      if (disposed || !token) return;
      const apiBase = API;
      const wsBase = apiBase.replace(/^http/, 'ws');
      const url = new URL('/ws/telemetry', wsBase);
      const wsToken = readTokenFromCookie() || token;
      if (wsToken) url.searchParams.set('token', wsToken);

      const ws = new WebSocket(url.toString());

      ws.onopen = () => { backoff = 1000; };
      ws.onmessage = (event) => {
        if (disposed) return;
        try {
          const data = JSON.parse(event.data);
          if (data.type === 'ORDER_STATE_CHANGED') {
            fetchOrdersRef.current();
          }
        } catch { /* ignore GPS pings */ }
      };
      ws.onclose = () => {
        if (disposed) return;
        reconnectTimer = setTimeout(() => connect(), backoff);
        backoff = Math.min(backoff * 2, 30_000);
      };
      ws.onerror = () => {};

      return ws;
    };

    const ws = connect();
    return () => {
      disposed = true;
      if (reconnectTimer) clearTimeout(reconnectTimer);
      ws?.close();
    };
  }, [token]);

  /* ─── Available Trucks ──────────────────────────────────── */

  const fetchTrucks = useCallback(async () => {
    if (!token) return;
    try {
      const res = await fetch(`${API}/v1/supplier/fleet/drivers`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) return;
      const drivers = await res.json();
      const list: SupplierDriverOption[] = Array.isArray(drivers) ? drivers : [];
      setTrucks(list
        .filter((driver) => driver.is_active && Boolean(driver.driver_id))
        .filter((driver) => Boolean(driver.vehicle_id || driver.vehicle_class || driver.vehicle_type || driver.license_plate))
        .map((driver) => ({
          driver_id: driver.driver_id,
          driver_name: driver.name || driver.driver_id,
          vehicle_class: driver.vehicle_class || driver.vehicle_type || '',
          plate_number: driver.license_plate || '',
        }))
        .filter((driver): driver is TruckOption => driver.driver_id.length > 0));
    } catch { /* noop */ }
  }, [token]);

  const fetchCapacity = useCallback(async (routeId: string) => {
    if (!token || !routeId) { setTargetCapacity(null); return; }
    try {
      const res = await fetch(`${API}/v1/fleet/capacity?route_id=${encodeURIComponent(routeId)}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) { setTargetCapacity(null); return; }
      setTargetCapacity(await res.json());
    } catch { setTargetCapacity(null); }
  }, [token]);

  /* ─── Vetting Actions ───────────────────────────────────── */

  async function vetOrder(orderId: string, decision: 'APPROVED' | 'REJECTED', reason?: string) {
    setActionLoading(orderId);
    try {
      const res = await fetch(`${API}/v1/supplier/orders/vet`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildOrderVettingIdempotencyKey(orderId, decision, reason),
        },
        body: JSON.stringify({ order_id: orderId, decision, reason: reason || '' }),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({ error: 'Action failed' }));
        toast(errJson.error || 'Action failed', 'error');
        return;
      }
      const result = await res.json();
      toast(`Order ${shortId(orderId)} ${result.status}`, 'success');
      setRejectingId(null);
      setRejectReason('');
      fetchOrders();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setActionLoading(null);
    }
  }

  /* ─── Approve Cancel ─────────────────────────────────────── */

  async function approveCancel(orderId: string) {
    setActionLoading(orderId);
    try {
      const res = await fetch(`${API}/v1/admin/orders/approve-cancel`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierApproveCancelIdempotencyKey(orderId),
        },
        body: JSON.stringify({ order_id: orderId }),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({ error: 'Action failed' }));
        toast(errJson.error || 'Failed to approve cancel', 'error');
        return;
      }
      toast(`Order ${shortId(orderId)} cancelled`, 'success');
      fetchOrders();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setActionLoading(null);
    }
  }

  /* ─── Resolve Credit Delivery ────────────────────────────── */

  async function resolveCreditDelivery(orderId: string, decision: 'APPROVE' | 'DENY') {
    setActionLoading(orderId);
    try {
      const res = await fetch(`${API}/v1/admin/orders/resolve-credit`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierResolveCreditIdempotencyKey(orderId, decision),
        },
        body: JSON.stringify({ order_id: orderId, decision }),
      });
      if (!res.ok) {
        const errJson = await res.json().catch(() => ({ error: 'Action failed' }));
        toast(errJson.error || 'Failed to resolve credit', 'error');
        return;
      }
      toast(`Credit delivery ${decision === 'APPROVE' ? 'approved' : 'denied'} — ${shortId(orderId)}`, 'success');
      fetchOrders();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setActionLoading(null);
    }
  }

  /* ─── Reassignment ──────────────────────────────────────── */

  function openReassign(orderIds: string[]) {
    setReassignOrderIds(orderIds);
    setTargetTruck('');
    setTargetCapacity(null);
    setReassignOpen(true);
    fetchTrucks();
  }

  async function executeReassign() {
    if (!targetTruck || reassignOrderIds.length === 0) return;
    setReassigning(true);
    try {
      const res = await fetch(`${API}/v1/fleet/reassign`, {
        method: 'POST',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierFleetReassignIdempotencyKey(targetTruck, reassignOrderIds),
        },
        body: JSON.stringify({ order_ids: reassignOrderIds, new_route_id: targetTruck }),
      });
      const json = await res.json();
      if (json.conflicts?.length > 0) {
        const conflictMsgs = json.conflicts.map((c: { order_id: string; reason: string }) => `${shortId(c.order_id)}: ${c.reason}`);
        toast(`Reassigned ${json.reassigned}/${json.total}. Conflicts: ${conflictMsgs.join('; ')}`, json.reassigned > 0 ? 'success' : 'error');
      } else {
        toast(`Reassigned ${json.reassigned} order(s) to ${targetTruck}`, 'success');
      }
      setReassignOpen(false);
      setSelected(new Set());
      fetchOrders();
    } catch (e) {
      toast((e as Error).message, 'error');
    } finally {
      setReassigning(false);
    }
  }

  /* ─── Selection helpers ─────────────────────────────────── */

  function toggleSelect(id: string) {
    setSelected(prev => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id); else next.add(id);
      return next;
    });
  }

  function toggleSelectAll() {
    if (selected.size === filteredOrders.length) {
      setSelected(new Set());
    } else {
      setSelected(new Set(filteredOrders.map(o => o.order_id)));
    }
  }

  const selectedOrders = orders.filter(o => selected.has(o.order_id));
  const canReassign = selectedOrders.length > 0 && selectedOrders.every(o => ['PENDING', 'LOADED', 'DISPATCHED'].includes(o.state) && o.route_id);
  const canDispatch = selectedOrders.length > 0 && selectedOrders.every(o => o.state === 'PENDING');

  async function dispatchSelected() {
    setDispatching(true);
    try {
      const token = readTokenFromCookie();
      const res = await fetch(`${API}/v1/supplier/manifests/auto-dispatch`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
          'Idempotency-Key': buildSupplierAutoDispatchIdempotencyKey(Array.from(selected)),
        },
        body: JSON.stringify({ order_ids: Array.from(selected) }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      toast(`Dispatched ${data.manifests?.length ?? 0} manifest(s)`, 'success');
      setSelected(new Set());
      fetchOrders();
    } catch (err) {
      toast((err as Error).message || 'Dispatch failed', 'error');
    } finally {
      setDispatching(false);
    }
  }

  // Client-side filtering
  const filteredOrders = useMemo(() => {
    let result = orders;
    if (searchQuery.trim()) {
      const q = searchQuery.trim().toLowerCase();
      result = result.filter(o => o.order_id.toLowerCase().includes(q) || (o.retailer_name || '').toLowerCase().includes(q));
    }
    if (stateFilter) {
      result = result.filter(o => o.state === stateFilter);
    }
    return result;
  }, [orders, searchQuery, stateFilter]);

  const totalItems = hasMore ? serverOffset + orders.length + 1 : serverOffset + orders.length;
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const pagination: PaginationState<Order> = {
    page,
    pageSize,
    totalPages,
    totalItems,
    pageItems: filteredOrders,
    setPage: (p) => setPage(Math.max(1, p)),
    nextPage: () => {
      if (hasMore) setPage((p) => p + 1);
    },
    prevPage: () => setPage((p) => Math.max(1, p - 1)),
    setPageSize: (size) => {
      setPageSize(size);
      setPage(1);
      setSelected(new Set());
    },
    canNext: hasMore,
    canPrev: page > 1,
  };

  /* ─── Render guards ─────────────────────────────────────── */

  if (!token) {
    return (
      <div className="min-h-full flex items-center justify-center bg-background">
        <div className="rounded-xl p-6 bg-danger text-danger-foreground">
          Unauthorized — supplier credentials required
        </div>
      </div>
    );
  }

  /* ─── Tab content emptys ────────────────────────────────── */

  const emptyMeta: Record<Tab | 'history', { headline: string; body: string }> = {
    active: { headline: 'No active orders', body: 'Pending, in-transit, and arrived orders appear here.' },
    scheduled: { headline: 'No scheduled orders', body: 'Future-dated orders will appear in this tab.' },
    history: { headline: 'No order history', body: 'Completed and cancelled orders will show up here.' },
  };
  const currentEmptyKey: Tab | 'history' = showHistory ? 'history' : tab;

  return (
    <div className="min-h-full p-6 md:p-8 bg-background text-foreground">
      {/* ── Page Header ── */}
      <header className="mb-6 flex items-start justify-between gap-4">
        <div>
          <h1 className="md-typescale-headline-medium">Orders</h1>
          <p className="md-typescale-body-small mt-1 text-muted">
            Manage the full order lifecycle — approval, dispatch, tracking, and history
          </p>
        </div>
        <Button
          variant="secondary"
          onPress={fetchOrders}
          isDisabled={loading}
          aria-label="Refresh orders"
          className="shrink-0"
        >
          <Icon name="returns" size={18} />
          Refresh
        </Button>
      </header>

      <div className="md-divider mb-6" />

      {/* ── Shop-Closed Escalation Banner ── */}
      <ShopClosedBanner />

      {/* ── Early Complete Request Banner ── */}
      <EarlyCompleteBanner />

      {/* ── Live Negotiation Banner ── */}
      <NegotiationBanner />

      {/* ── Tab Bar + Filter Chips ── */}
      <div className="flex items-center justify-between mb-3">
        <div className="md-tab-bar">
          {TAB_META.map(t => (
            <button
              key={t.key}
              onClick={() => { setTab(t.key); setShowHistory(false); setSelected(new Set()); setPage(1); }}
              className={`md-tab ${!showHistory && tab === t.key ? 'md-tab-active' : ''}`}
              data-active={!showHistory && tab === t.key}
            >
              <Icon name={t.icon} size={18} />
              <span>{t.label}</span>
              {!loading && !showHistory && tab === t.key && (
                <span
                  className="ml-1.5 inline-flex items-center justify-center h-5 min-w-5 px-1.5 md-typescale-label-small md-shape-full bg-accent text-accent-foreground"
                >
                  {orders.length}
                </span>
              )}
            </button>
          ))}
        </div>
      </div>

      {/* ── Filter Chips Row ── */}
      <div className="flex flex-wrap items-center gap-2 mb-4">
        <input
          type="text"
          value={searchQuery}
          onChange={e => setSearchQuery(e.target.value)}
          placeholder="Search order ID or retailer…"
          className="md-input-outlined text-[12px] h-8 px-3"
          style={{ minWidth: 220, maxWidth: 300 }}
        />
        <select
          value={stateFilter}
          onChange={e => setStateFilter(e.target.value)}
          className="md-input-outlined text-[12px] h-8 px-2"
          style={{ minWidth: 150 }}
        >
          <option value="">All States</option>
          {['PENDING','LOADED','DISPATCHED','IN_TRANSIT','ARRIVED','AWAITING_PAYMENT','PENDING_CASH_COLLECTION','COMPLETED','SCHEDULED','QUARANTINE','DELIVERED_ON_CREDIT'].map(s => (
            <option key={s} value={s}>{s.replace(/_/g, ' ')}</option>
          ))}
        </select>
        <button
          type="button"
          onClick={() => { setShowHistory(h => !h); setSelected(new Set()); setPage(1); }}
          className="flex items-center gap-1.5 px-3 h-8 rounded-full text-sm"
          style={showHistory
            ? { background: 'var(--accent)', color: 'var(--accent-foreground)' }
            : { border: '1px solid var(--border)', color: 'var(--muted)' }}
        >
          <Icon name="ledger" size={14} />
          History
        </button>
      </div>

      {/* ── Bulk Action Bar ── */}
      {selected.size > 0 && (
        <div
          className="mb-4 px-4 py-3 md-shape-md flex items-center justify-between bg-accent-soft text-accent-soft-foreground"
        >
          <span className="md-typescale-label-large">{selected.size} selected</span>
          <div className="flex items-center gap-2">
            {canDispatch && (
              <Button variant="primary" isDisabled={dispatching} onPress={dispatchSelected}>
                {dispatching ? 'Dispatching…' : 'Dispatch Selected'}
              </Button>
            )}
            {canReassign && (
              <Button variant="primary" onPress={() => openReassign(Array.from(selected))}>
                Reassign Truck
              </Button>
            )}
            <Button variant="ghost" onPress={() => setSelected(new Set())}>
              Clear
            </Button>
          </div>
        </div>
      )}

      {/* ── Data Table ── */}
      <div className="rounded-xl border border-border p-0 overflow-hidden">
        {loading ? (
          <div className="p-6 space-y-1">
            {Array.from({ length: 8 }).map((_, i) => (
              <Skeleton key={i} className="md-skeleton-row" />
            ))}
          </div>
        ) : orders.length === 0 ? (
          <EmptyState
            icon="orders"
            headline={emptyMeta[currentEmptyKey].headline}
            body={emptyMeta[currentEmptyKey].body}
            action="Refresh"
            onAction={fetchOrders}
          />
        ) : (
          <>
            <div className="overflow-x-auto">
              <table className="md-table w-full">
                <thead>
                  <tr>
                    <th className="w-10">
                      <input
                        type="checkbox"
                        checked={selected.size === filteredOrders.length && filteredOrders.length > 0}
                        onChange={toggleSelectAll}
                        aria-label="Select all"
                      />
                    </th>
                    <th>Order</th>
                    <th>Retailer</th>
                    <th>State</th>
                    <th>Truck / Route</th>
                    <th>Delivery Date</th>
                    <th className="text-right">Items</th>
                    <th className="text-right">Amount</th>
                    <th>Payment</th>
                    <th>Created</th>
                    <th className="text-right">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {filteredOrders.map(order => (
                    <tr
                      key={order.order_id}
                      className="cursor-pointer"
                      style={selected.has(order.order_id) ? { background: 'var(--accent-soft)', opacity: 0.85 } : undefined}
                      onClick={() => setDetailOrder(order)}
                    >
                      <td onClick={e => e.stopPropagation()}>
                        <input
                          type="checkbox"
                          checked={selected.has(order.order_id)}
                          onChange={() => toggleSelect(order.order_id)}
                          aria-label={`Select ${order.order_id}`}
                        />
                      </td>
                      <td className="font-mono md-typescale-body-small">{shortId(order.order_id)}</td>
                      <td>{order.retailer_name || shortId(order.retailer_id)}</td>
                      <td><StatusBadge state={order.state} /></td>
                      <td className="font-mono md-typescale-label-small">{order.route_id || '—'}</td>
                      <td className="md-typescale-label-small">
                        {order.requested_delivery_date ? (() => {
                          const daysOut = Math.ceil((new Date(order.requested_delivery_date).getTime() - Date.now()) / 86400000);
                          return (
                            <span style={daysOut > 1 ? { color: 'var(--warning)' } : undefined}>
                              {order.requested_delivery_date}
                              {daysOut > 1 && <span title="Scheduled far out" style={{ marginLeft: 4 }}>&#9888;</span>}
                            </span>
                          );
                        })() : '—'}
                      </td>
                      <td className="text-right font-mono">{order.item_count}</td>
                      <td className="text-right font-mono" style={{ fontVariantNumeric: 'tabular-nums' }}>{formatAmount(order.amount)}</td>
                      <td>
                        {order.payment_gateway ? (
                          <span className="md-typescale-label-small">
                            {order.payment_gateway}
                            {order.payment_status && (
                              <span className="ml-1 text-muted">· {order.payment_status}</span>
                            )}
                          </span>
                        ) : (
                          <span className="text-muted">—</span>
                        )}
                      </td>
                      <td className="md-typescale-label-small text-muted">
                        {order.created_at ? new Date(order.created_at).toLocaleString() : '—'}
                      </td>
                      <td className="text-right" onClick={e => e.stopPropagation()}>
                        {renderActions(order)}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <PaginationControls pagination={pagination} />
          </>
        )}
      </div>

      {/* ── Order Detail Drawer ── */}
      {detailOrder && (
        <OrderDetailDrawer
          order={detailOrder}
          onClose={() => setDetailOrder(null)}
          onReassign={(id) => { setDetailOrder(null); openReassign([id]); }}
        />
      )}

      {/* ── Reassignment Modal ── */}
      <Dialog
        open={reassignOpen}
        onClose={() => setReassignOpen(false)}
        title="Reassign Orders"
        actions={
          <>
            <Button variant="ghost" onPress={() => setReassignOpen(false)}>Cancel</Button>
            <Button
              variant="primary"
              onPress={executeReassign}
              isDisabled={!targetTruck || reassigning}
            >
              {reassigning ? 'Reassigning…' : `Reassign ${reassignOrderIds.length} Order(s)`}
            </Button>
          </>
        }
      >
        <div className="space-y-4">
          <p className="md-typescale-body-small text-muted">
            Moving {reassignOrderIds.length} order(s) to a new truck. Capacity will be validated before execution.
          </p>

          <div>
            <label className="md-typescale-label-medium block mb-1">Target Truck</label>
            <select
              className="md-input-outlined w-full"
              value={targetTruck}
              onChange={(e) => {
                setTargetTruck(e.target.value);
                fetchCapacity(e.target.value);
              }}
            >
              <option value="">Select a truck…</option>
              {trucks.map(t => (
                <option key={t.driver_id} value={t.driver_id}>
                  {t.driver_name} — {t.vehicle_class} {t.plate_number && `(${t.plate_number})`}
                </option>
              ))}
            </select>
          </div>

          {targetCapacity && (
            <div
              className="p-3 md-shape-sm bg-surface text-foreground"
            >
              <p className="md-typescale-label-medium mb-2">Truck Capacity</p>
              <div className="flex gap-6 md-typescale-body-small">
                <div>
                  <span className="text-muted">Max: </span>
                  <span className="font-mono font-medium">{targetCapacity.max_volume_vu.toFixed(0)} VU</span>
                </div>
                <div>
                  <span className="text-muted">Used: </span>
                  <span className="font-mono font-medium">{targetCapacity.used_volume_vu.toFixed(0)} VU</span>
                </div>
                <div>
                  <span className="text-muted">Free: </span>
                  <span className="font-mono font-medium" style={{ color: targetCapacity.free_volume_vu > 0 ? 'var(--success)' : 'var(--danger)' }}>
                    {targetCapacity.free_volume_vu.toFixed(0)} VU
                  </span>
                </div>
              </div>
              {/* Capacity bar */}
              <div className="mt-2 h-2 md-shape-full overflow-hidden" style={{ background: 'var(--border)' }}>
                <div
                  className="h-full md-shape-full"
                  style={{
                    width: `${Math.min((targetCapacity.used_volume_vu / targetCapacity.max_volume_vu) * 100, 100)}%`,
                    background: targetCapacity.free_volume_vu > 0 ? 'var(--accent)' : 'var(--danger)',
                  }}
                />
              </div>
            </div>
          )}
        </div>
      </Dialog>
    </div>
  );

  /* ─── Actions column renderer ───────────────────────────── */

  function renderActions(order: Order) {
    const isActing = actionLoading === order.order_id;

    // Active tab: vetting actions for pending orders
    if (tab === 'active' && !showHistory && order.state === 'PENDING') {
      if (rejectingId === order.order_id) {
        return (
          <div className="flex items-center gap-2">
            <input
              type="text"
              value={rejectReason}
              onChange={e => setRejectReason(e.target.value)}
              placeholder="Reason…"
              className="w-36 md-input-outlined text-xs"
            />
            <Button
              variant="primary"
              size="sm"
              onPress={() => vetOrder(order.order_id, 'REJECTED', rejectReason)}
              isDisabled={!rejectReason || isActing}
            >
              Reject
            </Button>
            <button onClick={() => { setRejectingId(null); setRejectReason(''); }} className="md-typescale-label-small px-1 text-muted">×</button>
          </div>
        );
      }
      return (
        <div className="flex justify-end gap-1">
          <Button
            variant="primary"
            size="sm"
            onPress={() => vetOrder(order.order_id, 'APPROVED')}
            isDisabled={isActing}
          >
            {isActing ? '…' : 'Approve'}
          </Button>
          <Button
            variant="outline"
            size="sm"
            onPress={() => setRejectingId(order.order_id)}
            className="text-danger"
          >
            Reject
          </Button>
        </div>
      );
    }

    // Active tab: reassign dispatched orders
    if (tab === 'active' && !showHistory && ['PENDING', 'LOADED', 'DISPATCHED'].includes(order.state) && order.route_id) {
      return (
        <Button
          variant="secondary"
          size="sm"
          onPress={() => openReassign([order.order_id])}
        >
          Reassign
        </Button>
      );
    }

    // Cancel requested: approve cancel
    if (order.state === 'CANCEL_REQUESTED') {
      return (
        <Button
          variant="danger"
          size="sm"
          isDisabled={actionLoading === order.order_id}
          onPress={() => approveCancel(order.order_id)}
        >
          Approve Cancel
        </Button>
      );
    }

    // Credit delivery: approve or deny
    if (order.state === 'DELIVERED_ON_CREDIT') {
      return (
        <div className="flex justify-end gap-1">
          <Button
            variant="primary"
            size="sm"
            isDisabled={isActing}
            onPress={() => resolveCreditDelivery(order.order_id, 'APPROVE')}
          >
            {isActing ? '…' : 'Confirm Payment'}
          </Button>
          <Button
            variant="danger"
            size="sm"
            isDisabled={isActing}
            onPress={() => resolveCreditDelivery(order.order_id, 'DENY')}
          >
            Deny
          </Button>
        </div>
      );
    }

    return (
      <span className="md-typescale-label-small text-muted" style={{ lineHeight: '32px' }}>—</span>
    );
  }
}

/* ─── Order Detail Drawer ─────────────────────────────────── */

function OrderDetailDrawer({
  order,
  onClose,
  onReassign,
}: {
  order: Order;
  onClose: () => void;
  onReassign: (id: string) => void;
}) {
  const token = useToken();
  const [events, setEvents] = useState<OrderEvent[]>([]);
  const [eventsLoading, setEventsLoading] = useState(false);

  useEffect(() => {
    if (!token || !order.order_id) return;
    setEventsLoading(true);
    fetch(`${API}/v1/orders/${encodeURIComponent(order.order_id)}/events`, {
      headers: { Authorization: `Bearer ${token}` },
    })
      .then(res => res.ok ? res.json() : { events: [] })
      .then(json => setEvents(json.events || json.data || []))
      .catch(() => {})
      .finally(() => setEventsLoading(false));
  }, [token, order.order_id]);
  return (
    <Drawer open={true} onClose={onClose} title="Order Details">
        {/* Content */}
        <div className="px-6 py-5 space-y-6">
          {/* ID + Status */}
          <div>
            <p className="md-typescale-label-small mb-1 text-muted">Order ID</p>
            <p className="font-mono md-typescale-body-medium break-all">{order.order_id}</p>
            <div className="mt-2">
              <StatusBadge state={order.state} />
            </div>
          </div>

          {/* ── Order Progress Tracker ── */}
          {(() => {
            const stages = ["PENDING", "LOADED", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "COMPLETED"] as const;
            const labels: Record<string, string> = {
              PENDING: "Pending", LOADED: "Loaded", DISPATCHED: "Dispatched",
              IN_TRANSIT: "In Transit", ARRIVED: "Arrived", COMPLETED: "Completed",
            };
            const idx = stages.indexOf(order.state as typeof stages[number]);
            const cancelled = order.state === "CANCELLED" || order.state === "CANCEL_REQUESTED";
            const specialState = ["ARRIVED_SHOP_CLOSED", "NO_CAPACITY", "QUARANTINE", "DELIVERED_ON_CREDIT"].includes(order.state);
            return (
              <div>
                <p className="md-typescale-label-small mb-2 text-muted">Progress</p>
                {cancelled ? (
                  <div className="flex items-center gap-2 p-2 rounded-lg" style={{ background: 'var(--color-md-error-container, rgba(220,38,38,0.08))' }}>
                    <span className="md-typescale-label-medium font-semibold" style={{ color: 'var(--color-md-error, #dc2626)' }}>
                      {order.state === 'CANCEL_REQUESTED' ? 'Cancel Requested' : 'Cancelled'}
                    </span>
                  </div>
                ) : specialState ? (
                  <div className="flex items-center gap-2 p-2 rounded-lg" style={{ background: ['NO_CAPACITY', 'QUARANTINE'].includes(order.state) ? 'var(--color-md-error-container, rgba(220,38,38,0.08))' : order.state === 'DELIVERED_ON_CREDIT' ? 'var(--color-md-warning-container, rgba(234,179,8,0.12))' : 'var(--color-md-warning-container, rgba(234,179,8,0.12))' }}>
                    <span className="md-typescale-label-medium font-semibold" style={{ color: ['NO_CAPACITY', 'QUARANTINE'].includes(order.state) ? 'var(--color-md-error, #dc2626)' : order.state === 'DELIVERED_ON_CREDIT' ? 'var(--color-md-warning, #d97706)' : 'var(--color-md-warning, #d97706)' }}>
                      {order.state === 'NO_CAPACITY' ? 'No Capacity' : order.state === 'QUARANTINE' ? 'Quarantined' : order.state === 'DELIVERED_ON_CREDIT' ? 'Delivered on Credit' : 'Shop Closed'}
                    </span>
                  </div>
                ) : (
                  <div className="flex items-center gap-0.5">
                    {stages.map((s, i) => {
                      const done = i <= idx;
                      const current = i === idx;
                      return (
                        <div key={s} className="flex items-center gap-0.5 flex-1">
                          <div className="flex flex-col items-center gap-1 flex-1">
                            <div
                              className="w-6 h-6 rounded-full flex items-center justify-center text-[10px] font-bold shrink-0"
                              style={{
                                background: done ? 'var(--color-md-primary, #1a1a1a)' : 'var(--color-md-surface-container, #f2f2f2)',
                                color: done ? 'var(--color-md-on-primary, #fff)' : 'var(--color-md-outline, #999)',
                                boxShadow: current ? '0 0 0 2px var(--color-md-primary, #1a1a1a), 0 0 0 4px var(--color-md-surface, #fff)' : 'none',
                              }}
                            >
                              {done && i < idx ? '✓' : i + 1}
                            </div>
                            <span className="md-typescale-label-small text-center" style={{ color: current ? 'var(--color-md-primary, #1a1a1a)' : done ? 'var(--color-md-on-surface, #1a1a1a)' : 'var(--color-md-outline, #999)', fontSize: '9px' }}>
                              {labels[s]}
                            </span>
                          </div>
                          {i < stages.length - 1 && (
                            <div className="h-px flex-1 mt-[-14px]" style={{ background: i < idx ? 'var(--color-md-primary, #1a1a1a)' : 'var(--color-md-surface-container, #f2f2f2)' }} />
                          )}
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>
            );
          })()}

          {/* Retailer */}
          <div>
            <p className="md-typescale-label-small mb-1 text-muted">Retailer</p>
            <p className="md-typescale-body-medium">{order.retailer_name || 'Unknown'}</p>
            <p className="font-mono md-typescale-label-small text-muted">{order.retailer_id}</p>
          </div>

          {/* Financial */}
          <div className="grid grid-cols-2 gap-4">
            <div>
              <p className="md-typescale-label-small mb-1 text-muted">Amount</p>
              <p className="font-mono md-typescale-body-medium" style={{ fontVariantNumeric: 'tabular-nums' }}>{formatAmount(order.amount)}</p>
            </div>
            <div>
              <p className="md-typescale-label-small mb-1 text-muted">Items</p>
              <p className="font-mono md-typescale-body-medium">{order.item_count}</p>
            </div>
          </div>

          {/* Payment */}
          {(order.payment_gateway || order.payment_status) && (
            <div className="grid grid-cols-2 gap-4">
              {order.payment_gateway && (
                <div>
                  <p className="md-typescale-label-small mb-1 text-muted">Payment Gateway</p>
                  <p className="md-typescale-body-medium">{order.payment_gateway}</p>
                </div>
              )}
              {order.payment_status && (
                <div>
                  <p className="md-typescale-label-small mb-1 text-muted">Payment Status</p>
                  <StatusBadge state={order.payment_status} />
                </div>
              )}
            </div>
          )}

          {/* Assignment */}
          {order.route_id && (
            <div>
              <p className="md-typescale-label-small mb-1 text-muted">Assigned Truck</p>
              <p className="font-mono md-typescale-body-medium">{order.route_id}</p>
            </div>
          )}

          {/* Delivery */}
          {order.requested_delivery_date && (
            <div>
              <p className="md-typescale-label-small mb-1 text-muted">Requested Delivery</p>
              <p className="md-typescale-body-medium">{order.requested_delivery_date}</p>
            </div>
          )}

          {/* Timestamps */}
          <div>
            <p className="md-typescale-label-small mb-1 text-muted">Created</p>
            <p className="md-typescale-body-small">{order.created_at ? new Date(order.created_at).toLocaleString() : '—'}</p>
          </div>

          <div>
            <p className="md-typescale-label-small mb-1 text-muted">Source</p>
            <p className="md-typescale-body-small">{order.order_source}</p>
          </div>

          {/* ── Order Events Timeline ── */}
          <div>
            <p className="md-typescale-label-small mb-2 text-muted">Activity Timeline</p>
            {eventsLoading ? (
              <div className="space-y-2">
                {[1, 2, 3].map(i => (
                  <div key={i} className="h-10 rounded-lg animate-pulse" style={{ background: 'var(--color-md-surface-container, #f5f5f5)' }} />
                ))}
              </div>
            ) : events.length === 0 ? (
              <p className="md-typescale-body-small text-muted">No events recorded yet.</p>
            ) : (
              <div className="relative pl-5">
                {/* Timeline line */}
                <div
                  className="absolute left-[7px] top-1 bottom-1 w-px"
                  style={{ background: 'var(--color-md-outline-variant, #e0e0e0)' }}
                />
                <div className="space-y-3">
                  {events.map(evt => (
                    <div key={evt.event_id} className="relative flex gap-3">
                      {/* Dot */}
                      <div
                        className="absolute -left-5 top-1 w-3 h-3 rounded-full shrink-0"
                        style={{ background: 'var(--color-md-primary, #1a1a1a)', border: '2px solid var(--color-md-surface, #fff)' }}
                      />
                      <div className="flex-1 min-w-0">
                        <p className="md-typescale-label-small font-semibold">
                          {evt.event_type.replace(/_/g, ' ')}
                        </p>
                        <p className="md-typescale-body-small text-muted">
                          {evt.actor_role} • {new Date(evt.created_at).toLocaleString()}
                        </p>
                        {evt.metadata && (() => {
                          try {
                            const meta = JSON.parse(evt.metadata);
                            const entries = Object.entries(meta).slice(0, 4);
                            if (entries.length === 0) return null;
                            return (
                              <div className="mt-1 flex flex-wrap gap-1">
                                {entries.map(([k, v]) => (
                                  <span
                                    key={k}
                                    className="md-typescale-label-small px-1.5 py-0.5 md-shape-xs font-mono"
                                    style={{ background: 'var(--color-md-surface-container, #f5f5f5)' }}
                                  >
                                    {k}: {String(v)}
                                  </span>
                                ))}
                              </div>
                            );
                          } catch { return null; }
                        })()}
                      </div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Footer actions */}
        {order.route_id && ['PENDING', 'LOADED', 'DISPATCHED'].includes(order.state) && (
          <div className="px-6 py-4" style={{ borderTop: '1px solid var(--border)' }}>
            <Button
              variant="secondary"
              fullWidth
              onPress={() => onReassign(order.order_id)}
            >
              Reassign to Different Truck
            </Button>
          </div>
        )}
    </Drawer>
  );
}

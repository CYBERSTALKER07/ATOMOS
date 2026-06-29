'use client';

import { useState, useEffect, useCallback, useMemo, useRef, Fragment } from 'react';
import { usePolling, factorySupplyRequestTransitionKey } from '@pegasusx/api-client';
import type { SupplyFulfillOptions } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { downloadCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { ListToolbar } from '@/components/ListToolbar';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import FactoryRuntimeBanner from '@/components/FactoryRuntimeBanner';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';

interface SupplyRequestItem {
  item_id: string;
  product_id: string;
  requested_quantity: number;
  shipped_quantity?: number;
  received_quantity?: number;
  recommended_qty?: number;
  unit_volume_vu?: number;
}

interface SupplyRequest {
  request_id: string;
  warehouse_id: string;
  warehouse_name?: string;
  supplier_id: string;
  state: string;
  priority: string;
  requested_delivery_date: string;
  total_volume_vu: number;
  notes: string;
  item_count?: number;
  items?: SupplyRequestItem[];
  created_at: string;
}

const LIVE_REFRESH_MS = 30_000;

const STATE_COLORS: Record<string, string> = {
  DRAFT: 'var(--color-md-outline)',
  SUBMITTED: 'var(--color-md-info)',
  ACKNOWLEDGED: 'var(--color-md-primary)',
  IN_PRODUCTION: 'var(--color-md-warning)',
  READY: 'var(--color-md-success)',
  FULFILLED: 'var(--color-md-on-surface-variant)',
  CANCELLED: 'var(--color-md-error)',
};

const PRIORITY_COLORS: Record<string, string> = {
  CRITICAL: 'var(--color-md-error)',
  URGENT: 'var(--color-md-warning)',
  NORMAL: 'var(--color-md-on-surface-variant)',
};

const ACTIONS: Record<string, { label: string; action: string; color: string }[]> = {
  SUBMITTED: [
    { label: 'Acknowledge', action: 'ACKNOWLEDGE', color: 'var(--color-md-primary)' },
    { label: 'Cancel', action: 'CANCEL', color: 'var(--color-md-error)' },
  ],
  ACKNOWLEDGED: [
    { label: 'Start Production', action: 'START_PRODUCTION', color: 'var(--color-md-warning)' },
    { label: 'Cancel', action: 'CANCEL', color: 'var(--color-md-error)' },
  ],
  IN_PRODUCTION: [
    { label: 'Mark Ready', action: 'MARK_READY', color: 'var(--color-md-success)' },
  ],
  READY: [
    { label: 'Fulfill', action: 'FULFILL', color: 'var(--color-md-success)' },
  ],
};

type FilterState = 'ALL' | 'SUBMITTED' | 'ACKNOWLEDGED' | 'IN_PRODUCTION' | 'READY' | 'FULFILLED' | 'CANCELLED';

function requestSignature(items: SupplyRequest[]) {
  return items
    .map((request) => `${request.request_id}:${request.state}:${request.total_volume_vu}`)
    .join('|');
}

function formatSyncTime(value: number | null) {
  if (!value) return 'Waiting for first sync';
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}

export default function SupplyRequestsPage() {
  const { toast } = useToast();
  const [requests, setRequests] = useState<SupplyRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<FilterState>('ALL');
  const [expandedRequestId, setExpandedRequestId] = useState<string | null>(null);
  const [transitioning, setTransitioning] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [viewMode, setViewMode] = useState<'table' | 'board'>('table');
  const [fulfillModal, setFulfillModal] = useState<{ request: SupplyRequest; options: SupplyFulfillOptions } | null>(null);
  const [lastSyncedAt, setLastSyncedAt] = useState<number | null>(null);
  const [isOffline, setIsOffline] = useState(() => (typeof navigator === 'undefined' ? false : !navigator.onLine));
  const previousSignatureRef = useRef('');

  const fetchRequests = useCallback(async (options?: { background?: boolean; silent?: boolean }) => {
    const background = options?.background ?? false;
    const silent = options?.silent ?? false;

    if (background) {
      setRefreshing(true);
    } else if (requests.length === 0) {
      setLoading(true);
    }

    try {
      const res = await apiFetch('/v1/factory/supply-requests');
      if (!res.ok) {
        throw new Error(`Factory API responded with ${res.status}`);
      }

      const data = await res.json();
      const next = Array.isArray(data) ? data : data.requests || data.data || [];
      const nextSignature = requestSignature(next);

      if (background && previousSignatureRef.current && previousSignatureRef.current !== nextSignature && !silent) {
        toast('Supply queue updated', 'info');
      }

      previousSignatureRef.current = nextSignature;
      setRequests(next);
      setLastSyncedAt(Date.now());
      setError(null);
      setIsOffline(false);
    } catch {
      const message = isOffline || (typeof navigator !== 'undefined' && !navigator.onLine)
        ? 'Offline. Showing the last synced supply queue.'
        : 'Live refresh failed. Showing the last synced supply queue.';

      if (requests.length === 0) {
        setError(message);
      } else {
        setError(message);
        if (!silent) {
          toast(message, 'warning');
        }
      }

      if (typeof navigator !== 'undefined') {
        setIsOffline(!navigator.onLine);
      }
    } finally {
      setLoading(false);
      setRefreshing(false);
    }
  }, [isOffline, requests.length, toast]);

  useEffect(() => {
    void fetchRequests();
  }, [fetchRequests]);

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event || event.type !== 'FACTORY_SUPPLY_REQUEST_UPDATE') {
          return;
        }
        void fetchRequests({ background: true, silent: true });
      },
    });

    return () => {
      unsubscribe();
    };
  }, [fetchRequests]);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await fetchRequests({ background: true, silent: true });
    },
    LIVE_REFRESH_MS,
    [fetchRequests],
  );

  useEffect(() => {
    const handleOnline = () => {
      setIsOffline(false);
      toast('Connection restored. Refreshing supply queue.', 'info');
      void fetchRequests({ background: true, silent: true });
    };

    const handleOffline = () => {
      setIsOffline(true);
      toast('Offline. Showing the last synced supply queue.', 'warning');
    };

    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);

    return () => {
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
    };
  }, [fetchRequests, toast]);

  const filtered = useMemo(
    () => (filter === 'ALL' ? requests : requests.filter((request) => request.state === filter)),
    [filter, requests],
  );

  const { page, pageCount, pageItems, next, prev, reset } = usePagination(filtered, 25);

  useEffect(() => {
    reset();
  }, [filter, reset]);

  const exportCsv = () => {
    downloadCsv(
      `factory-supply-requests${filter !== 'ALL' ? `-${filter.toLowerCase()}` : ''}.csv`,
      ['request_id', 'warehouse_id', 'state', 'priority', 'total_volume_vu', 'created_at'],
      filtered.map((request) => [
        request.request_id,
        request.warehouse_id,
        request.state,
        request.priority,
        String(request.total_volume_vu),
        request.created_at,
      ]),
    );
  };

  const runtimeMessage = isOffline
    ? `Offline — showing last sync from ${formatSyncTime(lastSyncedAt)}`
    : error && requests.length > 0
      ? `${error} Last sync ${formatSyncTime(lastSyncedAt)}`
      : refreshing
        ? `Refreshing live queue — last sync ${formatSyncTime(lastSyncedAt)}`
        : `Live sync active — last sync ${formatSyncTime(lastSyncedAt)}`;

  const runtimeTone = isOffline
    ? 'offline'
    : error && requests.length > 0
      ? 'warning'
      : refreshing
        ? 'refreshing'
        : 'live';

  const handleTransition = async (request: SupplyRequest, action: string) => {
    if (action === 'FULFILL') {
      try {
        const res = await apiFetch(`/v1/factory/supply-requests/${request.request_id}/fulfill-options`);
        if (!res.ok) {
          toast('Could not load fulfill options', 'error');
          return;
        }
        const options = (await res.json()) as SupplyFulfillOptions;
        setFulfillModal({ request, options });
      } catch {
        toast('Could not load fulfill options', 'error');
      }
      return;
    }
    await runTransition(request, action);
  };

  const runTransition = async (request: SupplyRequest, action: string) => {
    setTransitioning(request.request_id);
    try {
      const body: Record<string, unknown> = { action };
      if (action === 'FULFILL' && request.items?.length) {
        body.items = request.items.map((item) => ({
          item_id: item.item_id,
          shipped_quantity: item.shipped_quantity ?? item.requested_quantity,
        }));
      }
      const res = await apiFetch(`/v1/factory/supply-requests/${request.request_id}`, {
        method: 'PATCH',
        headers: {
          'Idempotency-Key': factorySupplyRequestTransitionKey(request.request_id, action),
        },
        body: JSON.stringify(body),
      });

      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Transition failed', 'error');
        return;
      }

      toast('Supply request updated', 'success');
      await fetchRequests({ background: true, silent: true });
    } catch {
      toast('Transition failed', 'error');
    } finally {
      setTransitioning(null);
    }
  };

  const submittedCount = useMemo(() => requests.filter((r) => r.state === 'SUBMITTED').length, [requests]);
  const inProductionCount = useMemo(() => requests.filter((r) => r.state === 'IN_PRODUCTION').length, [requests]);
  const readyCount = useMemo(() => requests.filter((r) => r.state === 'READY').length, [requests]);
  const totalVolume = useMemo(
    () => requests.reduce((sum, r) => sum + r.total_volume_vu, 0),
    [requests],
  );

  const fatalError =
    error && requests.length === 0
      ? isOffline
        ? 'Reconnect to fetch the first live sync for this factory queue.'
        : error
      : null;

  return (
    <PageTransition>
      <PageChrome
        icon="supplyRequests"
        title="Supply requests"
        description="Factory operators review and advance warehouse demand from this queue."
        loading={loading}
        skeletonVariant="table"
        error={fatalError}
        actions={
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => void fetchRequests({ background: requests.length > 0 })}
            className="portal-btn portal-btn--ghost inline-flex h-10 items-center gap-2"
          >
            <Icon name="refresh" size={16} /> Refresh
          </motion.button>
        }
      >
        <FactoryRuntimeBanner tone={runtimeTone} message={runtimeMessage} />

        <KpiStatGrid columns={4}>
          <KpiStatCard label="Submitted" value={submittedCount} sub="Awaiting factory ACK" />
          <KpiStatCard label="In production" value={inProductionCount} sub="Active factory work" />
          <KpiStatCard label="Ready to fulfill" value={readyCount} sub="Outbound handoff queue" />
          <KpiStatCard label="Total volume (VU)" value={totalVolume.toLocaleString()} sub={`${requests.length} requests total`} />
        </KpiStatGrid>

        <PageSection title="Queue filter" description={`${filtered.length} request${filtered.length !== 1 ? 's' : ''} in view`} className="mt-6">
          <div className="flex gap-2 flex-wrap items-center justify-between">
          <div className="flex gap-2 flex-wrap">
            {(['ALL', 'SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'] as FilterState[]).map((value) => (
              <button
                key={value}
                type="button"
                onClick={() => setFilter(value)}
                className="px-3 py-1.5 rounded-full text-xs font-medium transition-colors border"
                style={{
                  background: filter === value ? 'var(--color-md-primary)' : 'transparent',
                  color: filter === value ? 'var(--color-md-on-primary)' : 'var(--color-md-on-surface-variant)',
                  borderColor: filter === value ? 'var(--color-md-primary)' : 'var(--color-md-outline-variant)',
                }}
              >
                {value.replace(/_/g, ' ')}
              </button>
            ))}
          </div>
          <div className="flex gap-2">
            <button type="button" className={`portal-btn ${viewMode === 'table' ? 'portal-btn--primary' : 'portal-btn--ghost'} text-xs`} onClick={() => setViewMode('table')}>Table</button>
            <button type="button" className={`portal-btn ${viewMode === 'board' ? 'portal-btn--primary' : 'portal-btn--ghost'} text-xs`} onClick={() => setViewMode('board')}>Board</button>
          </div>
          </div>
        </PageSection>

        {filtered.length === 0 ? (
          <EmptyState
            variant={filter === 'ALL' ? 'no-data' : 'no-results'}
            imageUrl="/images/empty-orders.png"
            headline={filter === 'ALL' ? 'No supply requests found' : 'No requests match this filter'}
            body={
              filter === 'ALL'
                ? 'Warehouse demand will appear here as soon as requests reach this factory queue.'
                : `There are no ${filter.replace(/_/g, ' ').toLowerCase()} requests in the current view.`
            }
            action={filter === 'ALL' ? undefined : 'Clear filter'}
            onAction={filter === 'ALL' ? undefined : () => setFilter('ALL')}
          />
        ) : viewMode === 'board' ? (
          <PageSection title="Production lane board" description="Kanban by supply-request lifecycle state.">
            <div className="grid gap-4 lg:grid-cols-4 overflow-x-auto">
              {(['SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY'] as const).map((lane) => (
                <div key={lane} className="min-w-[220px] rounded-xl border p-3" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                  <div className="text-xs font-bold uppercase tracking-wider mb-3">{lane.replace(/_/g, ' ')}</div>
                  <div className="space-y-2">
                    {filtered.filter((r) => r.state === lane).map((request) => (
                      <div key={request.request_id} className="rounded-lg border p-3 text-sm" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                        <div className="font-medium">{request.warehouse_name || request.warehouse_id.slice(0, 8)}</div>
                        <div className="text-xs opacity-70 mt-1">{request.priority} · {request.item_count ?? request.items?.length ?? 0} items</div>
                        <div className="text-xs font-mono opacity-60 mt-1">
                          {request.requested_delivery_date ? new Date(request.requested_delivery_date).toLocaleDateString() : 'No delivery date'}
                        </div>
                        <div className="flex flex-wrap gap-1 mt-2">
                          {(ACTIONS[request.state] || []).map((action) => (
                            <button
                              key={action.action}
                              type="button"
                              className="px-2 py-1 rounded text-[10px] font-medium text-white"
                              style={{ background: action.color }}
                              disabled={transitioning === request.request_id}
                              onClick={() => void handleTransition(request, action.action)}
                            >
                              {action.label}
                            </button>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </PageSection>
        ) : (
          <>
            <ListToolbar
              page={page}
              pageCount={pageCount}
              totalLabel={`${filtered.length} supply requests`}
              onPrev={prev}
              onNext={next}
              onExport={exportCsv}
            />
            <PageSection title="Demand queue" description="Advance requests through ACK → production → ready → fulfill.">
              <motion.div
                initial={{ opacity: 0 }}
                animate={{ opacity: 1 }}
                className="overflow-x-auto -mx-5 px-5"
              >
                <table className="desk-table w-full text-sm">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container)' }}>
                  <th className="text-left px-4 py-3 font-medium">Warehouse</th>
                  <th className="text-left px-4 py-3 font-medium">Priority</th>
                  <th className="text-left px-4 py-3 font-medium">State</th>
                  <th className="text-left px-4 py-3 font-medium">Items</th>
                  <th className="text-left px-4 py-3 font-medium">Notes</th>
                  <th className="text-left px-4 py-3 font-medium">Volume (VU)</th>
                  <th className="text-left px-4 py-3 font-medium">Delivery Date</th>
                  <th className="text-left px-4 py-3 font-medium">Created</th>
                  <th className="text-right px-4 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {pageItems.map((request, index) => (
                  <Fragment key={request.request_id}>
                  <motion.tr 
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.05 }}
                    className="border-t hover:bg-[var(--default)]/50 transition-colors cursor-pointer" 
                    style={{ borderColor: 'var(--color-md-outline-variant)' }}
                    onClick={() => setExpandedRequestId(expandedRequestId === request.request_id ? null : request.request_id)}
                  >
                    <td className="px-4 py-3">
                      <div className="font-medium">{request.warehouse_name || request.warehouse_id.slice(0, 8)}</div>
                      <div className="text-xs font-mono" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {request.request_id.slice(0, 8)}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-bold uppercase tracking-wider" style={{ border: `1px solid ${PRIORITY_COLORS[request.priority]}`, color: PRIORITY_COLORS[request.priority] || 'inherit' }}>
                        {request.priority}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded text-[10px] font-bold uppercase tracking-wider" style={{ background: STATE_COLORS[request.state], color: 'white' }}>
                        {request.state.replace(/_/g, ' ')}
                      </span>
                    </td>
                    <td className="px-4 py-3 tabular-nums font-mono">
                      {request.item_count ?? request.items?.length ?? 0}
                    </td>
                    <td className="px-4 py-3 text-xs max-w-[180px] truncate" title={request.notes || undefined}>
                      {request.notes || '—'}
                    </td>
                    <td className="px-4 py-3 tabular-nums font-mono">{request.total_volume_vu.toLocaleString()}</td>
                    <td className="px-4 py-3 tabular-nums font-mono text-xs">
                      {request.requested_delivery_date ? new Date(request.requested_delivery_date).toLocaleDateString() : '—'}
                    </td>
                    <td className="px-4 py-3 text-xs tabular-nums font-mono" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                      {new Date(request.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 text-right" onClick={(e) => e.stopPropagation()}>
                      <div className="flex gap-2 justify-end">
                        {(ACTIONS[request.state] || []).map((action) => (
                          <motion.button
                            whileHover={{ scale: 1.05 }}
                            whileTap={{ scale: 0.95 }}
                            key={action.action}
                            onClick={() => void handleTransition(request, action.action)}
                            disabled={transitioning === request.request_id}
                            className="px-3 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50 hover-lift active-press"
                            style={{ background: action.color, color: 'white' }}
                          >
                            {transitioning === request.request_id ? '...' : action.label}
                          </motion.button>
                        ))}
                      </div>
                    </td>
                  </motion.tr>
                  {expandedRequestId === request.request_id && (request.items?.length ?? 0) > 0 && (
                    <tr key={`${request.request_id}-items`} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                      <td colSpan={9} className="px-4 py-3 bg-[var(--color-md-surface-container-low)]">
                        <div className="text-xs font-bold uppercase tracking-wider mb-2" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                          Requested SKUs
                        </div>
                        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
                          {request.items?.map((item) => (
                            <div key={item.item_id} className="rounded-lg border px-3 py-2 text-sm" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                              <div className="font-mono text-xs">{item.product_id}</div>
                              <div className="tabular-nums">Qty {item.requested_quantity.toLocaleString()}</div>
                            </div>
                          ))}
                        </div>
                      </td>
                    </tr>
                  )}
                  </Fragment>
                ))}
              </tbody>
            </table>
              </motion.div>
            </PageSection>
          </>
        )}
        {fulfillModal ? (
          <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
            <div className="w-full max-w-lg rounded-2xl border bg-[var(--background)] p-6 space-y-4">
              <h3 className="text-lg font-semibold">Confirm fulfill</h3>
              <p className="text-sm opacity-80">
                Warehouse: {fulfillModal.options.warehouse_name} · Mode: {fulfillModal.options.transfer_mode}
                {fulfillModal.options.co_located ? ' · Co-located site' : ''}
              </p>
              <div className="rounded-lg border p-3 text-sm space-y-2" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                <p><strong>INTERNAL:</strong> {fulfillModal.options.outcome_internal}</p>
                <p><strong>TRUCK:</strong> {fulfillModal.options.outcome_truck}</p>
                {fulfillModal.options.linked_driver_eta ? (
                  <p className="text-xs opacity-70">Linked transfer updated: {new Date(fulfillModal.options.linked_driver_eta).toLocaleString()}</p>
                ) : null}
              </div>
              <div className="flex gap-2 justify-end">
                <button type="button" className="portal-btn portal-btn--ghost" onClick={() => setFulfillModal(null)}>Cancel</button>
                <button
                  type="button"
                  className="portal-btn portal-btn--primary"
                  disabled={transitioning === fulfillModal.request.request_id}
                  onClick={() => {
                    const req = fulfillModal.request;
                    setFulfillModal(null);
                    void runTransition(req, 'FULFILL');
                  }}
                >
                  Confirm fulfill
                </button>
              </div>
            </div>
          </div>
        ) : null}
      </PageChrome>
    </PageTransition>
  );
}

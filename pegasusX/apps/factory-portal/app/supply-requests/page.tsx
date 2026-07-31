'use client';

import { useState, useEffect, useCallback, useMemo, useRef, Fragment } from 'react';
import { usePolling, factorySupplyRequestTransitionKey } from '@pegasusx/api-client';
import type { SupplyFulfillOptions } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { exportCsv } from '@/lib/csv';
import { usePagination } from '@/lib/use-pagination';
import { ListToolbar } from '@/components/ListToolbar';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { PageSection } from '@/components/PageSection';
import FactoryRuntimeBanner from '@/components/FactoryRuntimeBanner';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';

import type { SupplyRequest } from '@/components/supply-requests/types';
import { ACTIONS } from '@/components/supply-requests/constants';
import { SupplyRequestHeader } from '@/components/supply-requests/SupplyRequestHeader';
import { SupplyRequestList } from '@/components/supply-requests/SupplyRequestList';
import { SupplyRequestBoard } from '@/components/supply-requests/SupplyRequestBoard';

type FilterState = 'ALL' | 'SUBMITTED' | 'ACKNOWLEDGED' | 'IN_PRODUCTION' | 'READY' | 'FULFILLED' | 'CANCELLED';

const LIVE_REFRESH_MS = 30_000;

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
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  const [batchTransitioning, setBatchTransitioning] = useState<string | null>(null);
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

  useFactorySessionReconcile(() => {
    void fetchRequests({ background: true, silent: true });
  });

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
    setSelectedIds(new Set());
  }, [filter, reset]);

  const exportCsvHandler = async () => {
    const result = await exportCsv(
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
    if (!result.saved && !result.cancelled && result.reason) {
      toast(`Export failed: ${result.reason}`, 'error');
    }
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

  const handleToggleAll = () => {
    if (selectedIds.size === pageItems.length) {
      setSelectedIds(new Set());
    } else {
      setSelectedIds(new Set(pageItems.map(r => r.request_id)));
    }
  };

  const handleToggleOne = (id: string, e: React.ChangeEvent<HTMLInputElement>) => {
    e.stopPropagation();
    const next = new Set(selectedIds);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelectedIds(next);
  };

  const runBatchTransition = async (action: string) => {
    setBatchTransitioning(action);
    const selectedRequests = requests.filter(r => selectedIds.has(r.request_id));
    const validRequests = selectedRequests.filter(r => ACTIONS[r.state]?.some(a => a.action === action));
    
    if (validRequests.length === 0) {
      toast(`No selected requests are eligible for ${action.replace(/_/g, ' ').toLowerCase()}`, 'warning');
      setBatchTransitioning(null);
      return;
    }

    let successCount = 0;
    for (const request of validRequests) {
      try {
        const body: Record<string, unknown> = { action };
        const res = await apiFetch(`/v1/factory/supply-requests/${request.request_id}`, {
          method: 'PATCH',
          headers: {
            'Idempotency-Key': factorySupplyRequestTransitionKey(request.request_id, action),
          },
          body: JSON.stringify(body),
        });
        if (res.ok) successCount++;
      } catch {}
    }
    
    toast(`Batch completed: ${successCount}/${validRequests.length} succeeded`, successCount > 0 ? 'success' : 'error');
    setSelectedIds(new Set());
    setBatchTransitioning(null);
    await fetchRequests({ background: true, silent: true });
  };

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

        <SupplyRequestHeader requests={requests} />

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
          <SupplyRequestBoard filtered={filtered} transitioning={transitioning} handleTransition={handleTransition} />
        ) : (
          <>
            <ListToolbar
              page={page}
              pageCount={pageCount}
              totalLabel={`${filtered.length} supply requests`}
              onPrev={prev}
              onNext={next}
              onExport={() => void exportCsvHandler()}
            />
            {selectedIds.size > 0 && (
              <div className="mb-4 flex items-center justify-between rounded-lg border bg-(--color-md-surface-container) p-3" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                <span className="text-sm font-semibold">{selectedIds.size} selected</span>
                <div className="flex gap-2">
                  <button type="button" disabled={!!batchTransitioning} onClick={() => runBatchTransition('ACKNOWLEDGE')} className="portal-btn portal-btn--primary text-xs">
                    {batchTransitioning === 'ACKNOWLEDGE' ? '...' : 'Acknowledge'}
                  </button>
                  <button type="button" disabled={!!batchTransitioning} onClick={() => runBatchTransition('START_PRODUCTION')} className="portal-btn portal-btn--primary text-xs !bg-(--color-md-warning)">
                    {batchTransitioning === 'START_PRODUCTION' ? '...' : 'Start Production'}
                  </button>
                  <button type="button" disabled={!!batchTransitioning} onClick={() => runBatchTransition('MARK_READY')} className="portal-btn portal-btn--primary text-xs !bg-(--color-md-success)">
                    {batchTransitioning === 'MARK_READY' ? '...' : 'Mark Ready'}
                  </button>
                  <button type="button" disabled={!!batchTransitioning} onClick={() => runBatchTransition('CANCEL')} className="portal-btn portal-btn--primary text-xs !bg-(--color-md-error)">
                    {batchTransitioning === 'CANCEL' ? '...' : 'Cancel'}
                  </button>
                </div>
              </div>
            )}
            <SupplyRequestList pageItems={pageItems} selectedIds={selectedIds} transitioning={transitioning} handleToggleAll={handleToggleAll} handleToggleOne={handleToggleOne} handleTransition={handleTransition} />
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

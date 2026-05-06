'use client';

import { useState, useEffect, useCallback, useMemo, useRef } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageSkeleton } from '@/components/Skeleton';

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
  const [transitioning, setTransitioning] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);
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

  useEffect(() => {
    const refreshLiveData = () => {
      if (document.visibilityState === 'visible') {
        void fetchRequests({ background: true, silent: true });
      }
    };

    const handleOnline = () => {
      setIsOffline(false);
      toast('Connection restored. Refreshing supply queue.', 'info');
      void fetchRequests({ background: true, silent: true });
    };

    const handleOffline = () => {
      setIsOffline(true);
      toast('Offline. Showing the last synced supply queue.', 'warning');
    };

    const interval = window.setInterval(refreshLiveData, LIVE_REFRESH_MS);
    window.addEventListener('online', handleOnline);
    window.addEventListener('offline', handleOffline);
    document.addEventListener('visibilitychange', refreshLiveData);

    return () => {
      window.clearInterval(interval);
      window.removeEventListener('online', handleOnline);
      window.removeEventListener('offline', handleOffline);
      document.removeEventListener('visibilitychange', refreshLiveData);
    };
  }, [fetchRequests, toast]);

  const filtered = useMemo(
    () => (filter === 'ALL' ? requests : requests.filter((request) => request.state === filter)),
    [filter, requests],
  );

  const runtimeMessage = isOffline
    ? `Offline — showing last sync from ${formatSyncTime(lastSyncedAt)}`
    : error && requests.length > 0
      ? `${error} Last sync ${formatSyncTime(lastSyncedAt)}`
      : refreshing
        ? `Refreshing live queue — last sync ${formatSyncTime(lastSyncedAt)}`
        : `Live sync active — last sync ${formatSyncTime(lastSyncedAt)}`;

  const handleTransition = async (requestId: string, action: string) => {
    setTransitioning(requestId);
    try {
      const res = await apiFetch(`/v1/factory/supply-requests/${requestId}`, {
        method: 'PATCH',
        body: JSON.stringify({ action }),
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

  if (loading) {
    return (
      <PageTransition>
        <div className="p-6 space-y-4">
          <h1 className="text-xl font-semibold">Supply Requests</h1>
          <PageSkeleton />
        </div>
      </PageTransition>
    );
  }

  if (error && requests.length === 0) {
    return (
      <PageTransition>
        <div className="p-6 space-y-4">
          <div className="flex items-center justify-between">
            <h1 className="text-xl font-semibold">Incoming Supply Requests</h1>
            <button
              onClick={() => void fetchRequests()}
              className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
            >
              <Icon name="refresh" size={16} /> Retry
            </button>
          </div>
          <div
            className="rounded-2xl border p-6 text-sm"
            style={{
              borderColor: 'var(--color-md-outline-variant)',
              background: 'var(--color-md-surface-container-lowest)',
              color: 'var(--color-md-on-surface-variant)',
            }}
          >
            {error}
          </div>
        </div>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between gap-4">
          <div>
            <h1 className="text-xl font-semibold">Incoming Supply Requests</h1>
            <p className="mt-1 text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              {filtered.length} request{filtered.length !== 1 ? 's' : ''} in view
            </p>
          </div>
          <button
            onClick={() => void fetchRequests({ background: requests.length > 0 })}
            className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
        </div>

        <div
          className="rounded-2xl border px-4 py-3 text-sm"
          style={{
            borderColor: isOffline || error ? 'var(--color-md-warning)' : 'var(--color-md-outline-variant)',
            background: isOffline || error ? 'var(--color-md-surface-container-high)' : 'var(--color-md-surface-container-low)',
            color: 'var(--color-md-on-surface-variant)',
          }}
        >
          {runtimeMessage}
        </div>

        <div className="flex gap-2 flex-wrap">
          {(['ALL', 'SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'] as FilterState[]).map((value) => (
            <button
              key={value}
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

        {filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-3" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            <Icon name="transfers" size={40} />
            <p className="text-sm">No supply requests found</p>
          </div>
        ) : (
          <div className="overflow-x-auto rounded-xl border" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
            <table className="w-full text-sm">
              <thead>
                <tr style={{ background: 'var(--color-md-surface-container)' }}>
                  <th className="text-left px-4 py-3 font-medium">Warehouse</th>
                  <th className="text-left px-4 py-3 font-medium">Priority</th>
                  <th className="text-left px-4 py-3 font-medium">State</th>
                  <th className="text-left px-4 py-3 font-medium">Volume (VU)</th>
                  <th className="text-left px-4 py-3 font-medium">Delivery Date</th>
                  <th className="text-left px-4 py-3 font-medium">Created</th>
                  <th className="text-right px-4 py-3 font-medium">Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((request) => (
                  <tr key={request.request_id} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                    <td className="px-4 py-3">
                      <div className="font-medium">{request.warehouse_name || request.warehouse_id.slice(0, 8)}</div>
                      <div className="text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {request.request_id.slice(0, 8)}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded-full text-xs font-medium" style={{ color: PRIORITY_COLORS[request.priority] || 'inherit' }}>
                        {request.priority}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded text-xs font-medium" style={{ color: STATE_COLORS[request.state] || 'inherit' }}>
                        {request.state.replace(/_/g, ' ')}
                      </span>
                    </td>
                    <td className="px-4 py-3 tabular-nums">{request.total_volume_vu.toLocaleString()}</td>
                    <td className="px-4 py-3">
                      {request.requested_delivery_date ? new Date(request.requested_delivery_date).toLocaleDateString() : '—'}
                    </td>
                    <td className="px-4 py-3 text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                      {new Date(request.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex gap-2 justify-end">
                        {(ACTIONS[request.state] || []).map((action) => (
                          <button
                            key={action.action}
                            onClick={() => void handleTransition(request.request_id, action.action)}
                            disabled={transitioning === request.request_id}
                            className="px-3 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                            style={{ background: action.color, color: 'white' }}
                          >
                            {transitioning === request.request_id ? '...' : action.label}
                          </button>
                        ))}
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </PageTransition>
  );
}

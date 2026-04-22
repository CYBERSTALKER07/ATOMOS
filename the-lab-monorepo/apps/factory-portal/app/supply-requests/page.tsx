'use client';

import { useState, useEffect, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
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

export default function SupplyRequestsPage() {
  const [requests, setRequests] = useState<SupplyRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<FilterState>('ALL');
  const [transitioning, setTransitioning] = useState<string | null>(null);

  const fetchRequests = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/supply-requests');
      if (res.ok) {
        const data = await res.json();
        setRequests(data.requests || []);
      }
    } catch (e) {
      console.error('[SUPPLY REQUESTS]', e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { fetchRequests(); }, [fetchRequests]);

  const handleTransition = async (requestId: string, action: string) => {
    setTransitioning(requestId);
    try {
      const res = await apiFetch(`/v1/factory/supply-requests/${requestId}`, {
        method: 'PATCH',
        body: JSON.stringify({ action }),
      });
      if (res.ok) {
        fetchRequests();
      } else {
        const err = await res.json().catch(() => ({}));
        alert(err.error || 'Transition failed');
      }
    } catch (e) {
      console.error('[SUPPLY REQUEST TRANSITION]', e);
    } finally {
      setTransitioning(null);
    }
  };

  const filtered = filter === 'ALL' ? requests : requests.filter(r => r.state === filter);

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

  return (
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between">
          <h1 className="text-xl font-semibold">Incoming Supply Requests</h1>
          <span className="text-sm" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            {filtered.length} request{filtered.length !== 1 ? 's' : ''}
          </span>
        </div>

        {/* Filter chips */}
        <div className="flex gap-2 flex-wrap">
          {(['ALL', 'SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'] as FilterState[]).map(f => (
            <button
              key={f}
              onClick={() => setFilter(f)}
              className="px-3 py-1.5 rounded-full text-xs font-medium transition-colors border"
              style={{
                background: filter === f ? 'var(--color-md-primary)' : 'transparent',
                color: filter === f ? 'var(--color-md-on-primary)' : 'var(--color-md-on-surface-variant)',
                borderColor: filter === f ? 'var(--color-md-primary)' : 'var(--color-md-outline-variant)',
              }}
            >
              {f.replace(/_/g, ' ')}
            </button>
          ))}
        </div>

        {filtered.length === 0 ? (
          <div className="flex flex-col items-center justify-center py-16 gap-3"
               style={{ color: 'var(--color-md-on-surface-variant)' }}>
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
                {filtered.map(req => (
                  <tr key={req.request_id} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                    <td className="px-4 py-3">
                      <div className="font-medium">{req.warehouse_name || req.warehouse_id.slice(0, 8)}</div>
                      <div className="text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                        {req.request_id.slice(0, 8)}
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded-full text-xs font-medium"
                            style={{ color: PRIORITY_COLORS[req.priority] || 'inherit' }}>
                        {req.priority}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="px-2 py-0.5 rounded text-xs font-medium"
                            style={{ color: STATE_COLORS[req.state] || 'inherit' }}>
                        {req.state.replace(/_/g, ' ')}
                      </span>
                    </td>
                    <td className="px-4 py-3 tabular-nums">{req.total_volume_vu.toLocaleString()}</td>
                    <td className="px-4 py-3">
                      {req.requested_delivery_date
                        ? new Date(req.requested_delivery_date).toLocaleDateString()
                        : '—'}
                    </td>
                    <td className="px-4 py-3 text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                      {new Date(req.created_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 text-right">
                      <div className="flex gap-2 justify-end">
                        {(ACTIONS[req.state] || []).map(a => (
                          <button
                            key={a.action}
                            onClick={() => handleTransition(req.request_id, a.action)}
                            disabled={transitioning === req.request_id}
                            className="px-3 py-1 rounded-lg text-xs font-medium transition-opacity disabled:opacity-50"
                            style={{ background: a.color, color: 'white' }}
                          >
                            {transitioning === req.request_id ? '...' : a.label}
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

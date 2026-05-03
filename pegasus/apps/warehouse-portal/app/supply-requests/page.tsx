'use client';

import { useEffect, useEffectEvent, useState } from 'react';
import { apiFetch, subscribeWarehouseWS, type WarehouseSocketStatus } from '@/lib/auth';
import Icon from '@/components/Icon';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { useToast } from '@/components/Toast';
import type { WarehouseLiveEvent, WarehouseSupplyRequest } from '@pegasus/types';

const STATE_FILTERS = ['ALL', 'DRAFT', 'SUBMITTED', 'ACKNOWLEDGED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'];

function chipClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    SUBMITTED: 'status-chip--submitted',
    ACKNOWLEDGED: 'status-chip--acknowledged',
    IN_PRODUCTION: 'status-chip--in-production',
    READY: 'status-chip--ready',
    FULFILLED: 'status-chip--fulfilled',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || 'status-chip--draft';
}

export default function SupplyRequestsPage() {
  const router = useRouter();
  const { toast } = useToast();
  const [requests, setRequests] = useState<WarehouseSupplyRequest[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('ALL');
  const [socketStatus, setSocketStatus] = useState<WarehouseSocketStatus>('connecting');

  const loadRequests = useEffectEvent(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/warehouse/supply-requests');
      if (res.ok) {
        const data = await res.json() as WarehouseSupplyRequest[] | { supply_requests?: WarehouseSupplyRequest[] };
        setRequests(Array.isArray(data) ? data : (data.supply_requests || []));
      }
    } catch {
      toast('Failed to load supply requests', 'error');
    } finally {
      setLoading(false);
    }
  });

  const handleWarehouseLiveEvent = useEffectEvent((event: WarehouseLiveEvent) => {
    if (event.type !== 'SUPPLY_REQUEST_UPDATE') {
      return;
    }
    void loadRequests();
  });

  useEffect(() => {
    void loadRequests();
  }, [loadRequests]);

  useEffect(() => {
    return subscribeWarehouseWS({
      onStatusChange: setSocketStatus,
      onMessage: payload => {
      try {
        handleWarehouseLiveEvent(JSON.parse(payload) as WarehouseLiveEvent);
      } catch {
        // Ignore unrelated frames.
      }
      },
    });
  }, [handleWarehouseLiveEvent]);

  const filtered = filter === 'ALL' ? requests : requests.filter(r => r.state === filter);

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Supply Requests</h1>
        <div className="flex items-center gap-2">
          <button
            onClick={loadRequests}
            className="flex items-center gap-1.5 px-3 py-2 rounded-lg text-sm button--secondary border border-[var(--border)]"
          >
            <Icon name="refresh" size={16} />
            Refresh
          </button>
          <Link
            href="/supply-requests/new"
            className="flex items-center gap-1.5 px-4 py-2 rounded-lg text-sm font-semibold button--primary"
          >
            <Icon name="plus" size={16} />
            New Request
          </Link>
        </div>
      </div>

      {/* State filter tabs */}
      <div className="flex gap-1 overflow-x-auto pb-1">
        {STATE_FILTERS.map(s => (
          <button
            key={s}
            onClick={() => setFilter(s)}
            className={`px-3 py-1.5 rounded-lg text-xs font-semibold whitespace-nowrap transition-colors ${
              filter === s
                ? 'bg-[var(--accent)] text-[var(--accent-foreground)]'
                : 'text-[var(--muted)] hover:bg-[var(--surface)]'
            }`}
          >
            {s.replace('_', ' ')}
            {s !== 'ALL' && (
              <span className="ml-1 opacity-70">
                ({requests.filter(r => s === 'ALL' || r.state === s).length})
              </span>
            )}
          </button>
        ))}
      </div>

      {socketStatus !== 'idle' && socketStatus !== 'live' && (
        <div className={`rounded-xl border px-4 py-3 text-sm ${socketStatus === 'offline'
          ? 'border-[var(--danger)]/30 bg-[var(--danger)]/8 text-[var(--danger)]'
          : 'border-[var(--warning)]/30 bg-[var(--warning)]/8 text-[var(--warning)]'}`}>
          {socketStatus === 'offline'
            ? 'Offline. Live supply-request updates are paused until the network returns.'
            : socketStatus === 'reconnecting'
              ? 'Live supply-request updates are reconnecting. Current data may be stale.'
              : 'Connecting live supply-request updates…'}
        </div>
      )}

      {loading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="md-skeleton md-skeleton-row" />
          ))}
        </div>
      ) : filtered.length === 0 ? (
        <div className="text-center py-20 text-[var(--muted)]">
          <Icon name="supplyRequests" size={48} className="mx-auto mb-3 opacity-30" />
          <p className="text-sm">No supply requests found</p>
        </div>
      ) : (
        <div className="border border-[var(--border)] rounded-xl overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]" style={{ background: 'var(--surface)' }}>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">ID</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Factory</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">State</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Priority</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Delivery Date</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Volume</th>
                <th className="text-left px-4 py-3 font-semibold text-[var(--muted)]">Created</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map(req => (
                <tr
                  key={req.request_id}
                  onClick={() => router.push(`/supply-requests/${req.request_id}`)}
                  className="border-b border-[var(--border)] last:border-b-0 hover:bg-[var(--surface)] cursor-pointer transition-colors"
                >
                  <td className="px-4 py-3 font-mono text-xs">{req.request_id.slice(0, 8)}...</td>
                  <td className="px-4 py-3">{req.factory_id.slice(0, 8)}</td>
                  <td className="px-4 py-3">
                    <span className={`status-chip ${chipClass(req.state)}`}>{req.state}</span>
                  </td>
                  <td className="px-4 py-3">
                    <span className={`text-xs font-semibold ${
                      req.priority === 'CRITICAL' ? 'text-[var(--danger)]' :
                      req.priority === 'URGENT' ? 'text-[var(--warning)]' : 'text-[var(--muted)]'
                    }`}>{req.priority}</span>
                  </td>
                  <td className="px-4 py-3 text-xs">
                    {req.requested_delivery_date
                      ? new Date(req.requested_delivery_date).toLocaleDateString()
                      : '—'}
                  </td>
                  <td className="px-4 py-3">{req.total_volume_vu || 0} VU</td>
                  <td className="px-4 py-3 text-xs text-[var(--muted)]">
                    {new Date(req.created_at).toLocaleDateString()}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

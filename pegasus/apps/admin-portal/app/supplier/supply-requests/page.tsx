'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import EmptyState from '@/components/EmptyState';
import Icon from '@/components/Icon';

interface SupplyRequestHistoryRow {
  request_id: string;
  warehouse_id: string;
  warehouse_name: string;
  factory_id: string;
  factory_name: string;
  state: string;
  priority: string;
  total_volume_vu: number;
  transfer_order_id?: string;
  transfer_state?: string;
  lane_type?: string;
  is_nearby_factory: boolean;
  created_at: string;
  updated_at?: string;
}

const STATE_COLORS: Record<string, string> = {
  SUBMITTED: 'var(--color-md-info)',
  ACKNOWLEDGED: 'var(--color-md-primary)',
  IN_PRODUCTION: 'var(--color-md-warning)',
  READY: 'var(--color-md-tertiary)',
  FULFILLED: 'var(--color-md-success)',
  CANCELLED: 'var(--color-md-error)',
};

export default function SupplyRequestsPage() {
  const [items, setItems] = useState<SupplyRequestHistoryRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [stateFilter, setStateFilter] = useState('');

  const fetchHistory = useCallback(async () => {
    setLoading(true);
    setError('');
    try {
      const query = stateFilter ? `?state=${encodeURIComponent(stateFilter)}` : '';
      const res = await apiFetch(`/v1/supplier/supply-requests/history${query}`);
      if (!res.ok) throw new Error(`Failed (${res.status})`);
      const data = await res.json();
      setItems(Array.isArray(data.items) ? data.items : []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load supply requests');
    } finally {
      setLoading(false);
    }
  }, [stateFilter]);

  useEffect(() => { fetchHistory(); }, [fetchHistory]);

  const internalCount = items.filter(i => i.lane_type === 'INTERNAL').length;
  const truckCount = items.filter(i => i.lane_type !== 'INTERNAL').length;

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      <div className="flex flex-col md:flex-row md:items-center md:justify-between gap-4 mb-8">
        <div>
          <h1 className="md-typescale-headline-medium">Supply Requests</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
            Warehouse-to-factory replenishment audit across truck and internal lanes
          </p>
        </div>
        <div className="flex items-center gap-2 flex-wrap">
          {['', 'SUBMITTED', 'IN_PRODUCTION', 'READY', 'FULFILLED', 'CANCELLED'].map(s => (
            <button
              key={s || 'all'}
              type="button"
              onClick={() => setStateFilter(s)}
              className="px-3 py-1.5 md-typescale-label-medium"
              style={{
                background: stateFilter === s ? 'var(--accent-soft)' : 'var(--surface)',
                color: stateFilter === s ? 'var(--accent)' : 'var(--muted)',
                border: '1px solid var(--border)',
              }}
            >
              {s || 'All'}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <div className="md-card md-card-elevated p-4">
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Total Requests</p>
          <p className="md-typescale-headline-small">{items.length}</p>
        </div>
        <div className="md-card md-card-elevated p-4">
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Internal Lane</p>
          <p className="md-typescale-headline-small">{internalCount}</p>
        </div>
        <div className="md-card md-card-elevated p-4">
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Truck Lane</p>
          <p className="md-typescale-headline-small">{truckCount}</p>
        </div>
        <div className="md-card md-card-elevated p-4">
          <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Open</p>
          <p className="md-typescale-headline-small">
            {items.filter(i => !['FULFILLED', 'CANCELLED'].includes(i.state)).length}
          </p>
        </div>
      </div>

      {loading ? (
        <div className="flex justify-center py-20">
          <div className="w-8 h-8 border-2 border-t-transparent rounded-full animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
        </div>
      ) : error ? (
        <EmptyState icon="error" headline="Failed to load" body={error} action="Retry" onAction={fetchHistory} />
      ) : items.length === 0 ? (
        <EmptyState icon="warehouse" headline="No supply requests" body="Warehouse admins create requests from the warehouse portal." />
      ) : (
        <div className="md-card md-card-elevated overflow-hidden">
          <table className="md-table w-full">
            <thead>
              <tr>
                <th className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Request</th>
                <th className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Warehouse</th>
                <th className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Factory</th>
                <th className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>State</th>
                <th className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Lane</th>
                <th className="px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Transfer</th>
                <th className="px-4 py-3 text-right md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Volume VU</th>
              </tr>
            </thead>
            <tbody>
              {items.map(row => (
                <tr key={row.request_id} style={{ borderBottom: '1px solid var(--border)' }}>
                  <td className="px-4 py-3 md-typescale-body-small font-mono">{row.request_id.slice(0, 8)}…</td>
                  <td className="px-4 py-3">
                    <div className="md-typescale-body-medium">{row.warehouse_name}</div>
                    {row.is_nearby_factory && (
                      <span className="md-typescale-label-small" style={{ color: 'var(--accent)' }}>Nearby factory</span>
                    )}
                  </td>
                  <td className="px-4 py-3 md-typescale-body-medium">{row.factory_name}</td>
                  <td className="px-4 py-3">
                    <span style={{ color: STATE_COLORS[row.state] || 'var(--foreground)' }}>{row.state}</span>
                  </td>
                  <td className="px-4 py-3 md-typescale-body-small">{row.lane_type || 'TRUCK'}</td>
                  <td className="px-4 py-3 md-typescale-body-small">
                    {row.transfer_order_id ? (
                      <span>{row.transfer_state || '—'}</span>
                    ) : (
                      <span style={{ color: 'var(--muted)' }}>—</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-right md-typescale-body-medium" style={{ fontVariantNumeric: 'tabular-nums' }}>
                    {row.total_volume_vu.toFixed(1)}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      <div className="mt-6 flex items-center gap-2 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
        <Icon name="info" size={16} />
        Internal lane requests skip truck dispatch when the warehouse is flagged as co-located with its primary factory.
      </div>
    </div>
  );
}

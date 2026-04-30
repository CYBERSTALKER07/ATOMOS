'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Link from 'next/link';
import Icon from '@/components/Icon';

interface Transfer {
  id: string;
  source_factory_id: string;
  destination_warehouse_id: string;
  warehouse_name: string;
  state: string;
  priority: string;
  total_items: number;
  total_volume_m3: number;
  created_at: string;
  updated_at: string;
}

const STATE_FILTERS = ['ALL', 'DRAFT', 'APPROVED', 'LOADING', 'DISPATCHED', 'IN_TRANSIT', 'ARRIVED', 'RECEIVED', 'CANCELLED'];

export default function TransfersPage() {
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [stateFilter, setStateFilter] = useState('ALL');

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const qs = stateFilter !== 'ALL' ? `?state=${stateFilter}` : '';
      const res = await apiFetch(`/v1/factory/transfers${qs}`);
      if (res.ok) {
        const data = await res.json();
        setTransfers(data.transfers || []);
      }
    } catch { /* empty */ } finally {
      setLoading(false);
    }
  }, [stateFilter]);

  useEffect(() => { load(); }, [load]);

  const stateClass = (s: string) => {
    const map: Record<string, string> = {
      DRAFT: 'status-chip--draft', APPROVED: 'status-chip--approved',
      LOADING: 'status-chip--loading', DISPATCHED: 'status-chip--dispatched',
      IN_TRANSIT: 'status-chip--in-transit', ARRIVED: 'status-chip--arrived',
      RECEIVED: 'status-chip--received', CANCELLED: 'status-chip--cancelled',
    };
    return map[s] || '';
  };

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Transfers</h1>
        <button onClick={() => load()} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {/* Filter chips */}
      <div className="flex flex-wrap gap-2">
        {STATE_FILTERS.map(f => (
          <button
            key={f}
            onClick={() => setStateFilter(f)}
            className={`px-3 py-1 rounded-full text-xs font-semibold border transition-colors ${
              stateFilter === f
                ? 'bg-[var(--accent)] text-[var(--accent-foreground)] border-transparent'
                : 'bg-transparent text-[var(--muted)] border-[var(--border)] hover:border-[var(--accent)]'
            }`}
          >
            {f}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : transfers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="transfers" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No transfers found</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)]">
                <th className="table__column text-left py-2 px-3 font-medium">Warehouse</th>
                <th className="table__column text-left py-2 px-3 font-medium">State</th>
                <th className="table__column text-left py-2 px-3 font-medium">Priority</th>
                <th className="table__column text-right py-2 px-3 font-medium">Items</th>
                <th className="table__column text-right py-2 px-3 font-medium">Volume</th>
                <th className="table__column text-right py-2 px-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {transfers.map(t => (
                <tr key={t.id} className="table__row">
                  <td className="py-2.5 px-3">
                    <Link href={`/transfers/${t.id}`} className="font-medium hover:underline">
                      {t.warehouse_name || t.destination_warehouse_id.slice(0, 8)}
                    </Link>
                  </td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${stateClass(t.state)}`}>{t.state}</span>
                  </td>
                  <td className="py-2.5 px-3 text-[var(--muted)]">{t.priority}</td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{t.total_items}</td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{t.total_volume_m3.toFixed(1)} m³</td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">{new Date(t.created_at).toLocaleDateString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface Insight {
  id: string;
  warehouse_id: string;
  warehouse_name: string;
  sku_id: string;
  product_name: string;
  urgency: string;
  current_stock: number;
  daily_velocity: number;
  reorder_qty: number;
  days_to_empty: number;
  lead_time_days: number;
  status: string;
  created_at: string;
}

export default function InsightsPage() {
  const [insights, setInsights] = useState<Insight[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/replenishment/insights');
      if (res.ok) {
        const data = await res.json();
        setInsights(data.insights || []);
      }
    } catch { /* handled */ } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  const urgencyClass = (u: string) => {
    if (u === 'CRITICAL') return 'status-chip--critical';
    if (u === 'WARNING') return 'status-chip--warning';
    return 'status-chip--stable';
  };

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between">
        <h1 className="text-xl font-bold tracking-tight">Replenishment Insights</h1>
        <button onClick={() => load()} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : insights.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="insights" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No replenishment insights at this time</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="table__header border-b border-[var(--border)]">
                <th className="table__column text-left py-2 px-3 font-medium">Warehouse</th>
                <th className="table__column text-left py-2 px-3 font-medium">Product</th>
                <th className="table__column text-left py-2 px-3 font-medium">Urgency</th>
                <th className="table__column text-right py-2 px-3 font-medium">Stock</th>
                <th className="table__column text-right py-2 px-3 font-medium">Velocity/day</th>
                <th className="table__column text-right py-2 px-3 font-medium">Days Left</th>
                <th className="table__column text-right py-2 px-3 font-medium">Reorder Qty</th>
                <th className="table__column text-left py-2 px-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {insights.map(ins => (
                <tr key={ins.id} className="table__row">
                  <td className="py-2.5 px-3 font-medium">{ins.warehouse_name}</td>
                  <td className="py-2.5 px-3">{ins.product_name}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${urgencyClass(ins.urgency)}`}>{ins.urgency}</span>
                  </td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{ins.current_stock}</td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{ins.daily_velocity.toFixed(1)}</td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{ins.days_to_empty.toFixed(1)}</td>
                  <td className="py-2.5 px-3 text-right tabular-nums">{ins.reorder_qty}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${ins.status === 'ACTIVE' ? 'status-chip--approved' : 'status-chip--draft'}`}>
                      {ins.status}
                    </span>
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

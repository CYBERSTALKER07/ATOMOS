'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface Order {
  order_id: string;
  retailer_name: string;
  state: string;
  total_uzs: number;
  created_at: string;
}

const STATE_CLASSES: Record<string, string> = {
  PENDING: 'status-chip--draft',
  LOADED: 'status-chip--ready',
  IN_TRANSIT: 'status-chip--active',
  ARRIVED: 'status-chip--ready',
  COMPLETED: 'status-chip--stable',
  CANCELLED: 'status-chip--critical',
};

export default function OrdersPage() {
  const [orders, setOrders] = useState<Order[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState('');

  const load = useCallback(async () => {
    try {
      const q = filter ? `?state=${filter}` : '';
      const res = await apiFetch(`/v1/warehouse/ops/orders${q}`);
      if (res.ok) {
        const data = await res.json();
        setOrders(data.orders || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, [filter]);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Orders</h1>
        <div className="flex gap-2 items-center">
          <select
            value={filter}
            onChange={e => { setFilter(e.target.value); setLoading(true); }}
            className="px-3 py-1.5 rounded-lg border text-sm"
            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
          >
            <option value="">All States</option>
            {['PENDING', 'LOADED', 'IN_TRANSIT', 'ARRIVED', 'COMPLETED', 'CANCELLED'].map(s => (
              <option key={s} value={s}>{s}</option>
            ))}
          </select>
          <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} /> Refresh
          </button>
        </div>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : orders.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="orders" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No orders found</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Order ID</th>
                <th className="text-left py-2 px-3 font-medium">Retailer</th>
                <th className="text-left py-2 px-3 font-medium">State</th>
                <th className="text-right py-2 px-3 font-medium">Total (UZS)</th>
                <th className="text-right py-2 px-3 font-medium">Created</th>
              </tr>
            </thead>
            <tbody>
              {orders.map(o => (
                <tr key={o.order_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-mono text-xs">{o.order_id.slice(0, 8)}...</td>
                  <td className="py-2.5 px-3">{o.retailer_name || '—'}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${STATE_CLASSES[o.state] || ''}`}>{o.state}</span>
                  </td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(o.total_uzs)}</td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {new Date(o.created_at).toLocaleDateString()}
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

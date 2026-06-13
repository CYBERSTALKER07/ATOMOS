'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface Retailer {
  retailer_id: string;
  business_name: string;
  total_orders: number;
  total_revenue: number;
  last_order_date: string;
}

export default function CRMPage() {
  const [retailers, setRetailers] = useState<Retailer[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/crm');
      if (res.ok) {
        const data = await res.json();
        setRetailers(data.retailers || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Retailer CRM</h1>
        <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 5 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : retailers.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="crm" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No retailer relationships yet</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Business Name</th>
                <th className="text-right py-2 px-3 font-medium">Total Orders</th>
                <th className="text-right py-2 px-3 font-medium">Revenue (UZS)</th>
                <th className="text-right py-2 px-3 font-medium">Last Order</th>
              </tr>
            </thead>
            <tbody>
              {retailers.map(r => (
                <tr key={r.retailer_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-medium">{r.business_name || '—'}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(r.total_orders)}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(r.total_revenue)}</td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {r.last_order_date ? new Date(r.last_order_date).toLocaleDateString() : '—'}
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

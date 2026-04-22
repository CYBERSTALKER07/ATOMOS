'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface ReturnItem {
  line_item_id: string;
  order_id: string;
  product_name: string;
  quantity: number;
  status: string;
  updated_at: string;
}

export default function ReturnsPage() {
  const [items, setItems] = useState<ReturnItem[]>([]);
  const [loading, setLoading] = useState(true);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/warehouse/ops/returns');
      if (res.ok) {
        const data = await res.json();
        setItems(data.items || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Returns & Rejects</h1>
        <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 5 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="returns" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No returns or rejects</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Product</th>
                <th className="text-left py-2 px-3 font-medium">Order</th>
                <th className="text-right py-2 px-3 font-medium">Qty</th>
                <th className="text-left py-2 px-3 font-medium">Reason</th>
                <th className="text-right py-2 px-3 font-medium">Date</th>
              </tr>
            </thead>
            <tbody>
              {items.map(item => (
                <tr key={item.line_item_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-medium">{item.product_name}</td>
                  <td className="py-2.5 px-3 font-mono text-xs text-[var(--muted)]">{item.order_id.slice(0, 8)}...</td>
                  <td className="py-2.5 px-3 text-right font-mono">{item.quantity}</td>
                  <td className="py-2.5 px-3">
                    <span className="status-chip status-chip--critical">{item.status.replace(/_/g, ' ')}</span>
                  </td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {new Date(item.updated_at).toLocaleDateString()}
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

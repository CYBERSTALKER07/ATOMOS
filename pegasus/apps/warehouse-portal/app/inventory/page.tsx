'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

interface InventoryItem {
  product_id: string;
  product_name: string;
  quantity: number;
  reorder_threshold: number;
  sku: string;
}

export default function InventoryPage() {
  const [items, setItems] = useState<InventoryItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');
  const [lowOnly, setLowOnly] = useState(false);
  const [adjusting, setAdjusting] = useState<string | null>(null);
  const [adjustVal, setAdjustVal] = useState('');

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams();
      if (search) params.set('search', search);
      if (lowOnly) params.set('low_stock', 'true');
      const q = params.toString() ? `?${params}` : '';
      const res = await apiFetch(`/v1/warehouse/ops/inventory${q}`);
      if (res.ok) {
        const data = await res.json();
        setItems(data.items || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, [search, lowOnly]);

  useEffect(() => { load(); }, [load]);

  async function handleAdjust(productId: string) {
    const qty = parseInt(adjustVal, 10);
    if (isNaN(qty)) return;
    try {
      const res = await apiFetch('/v1/warehouse/ops/inventory', {
        method: 'PATCH',
        body: JSON.stringify({ product_id: productId, quantity: qty }),
      });
      if (res.ok) {
        setAdjusting(null);
        setAdjustVal('');
        load();
      }
    } catch { /* handled */ }
  }

  return (
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Inventory</h1>
        <div className="flex gap-2 items-center">
          <input
            placeholder="Search products..."
            value={search}
            onChange={e => { setSearch(e.target.value); setLoading(true); }}
            className="px-3 py-1.5 rounded-lg border text-sm w-48"
            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
          />
          <label className="flex items-center gap-1.5 text-sm text-[var(--muted)] cursor-pointer">
            <input type="checkbox" checked={lowOnly} onChange={e => { setLowOnly(e.target.checked); setLoading(true); }} />
            Low stock only
          </label>
          <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} />
          </button>
        </div>
      </div>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : items.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="inventory" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No inventory items found</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Product</th>
                <th className="text-left py-2 px-3 font-medium">SKU</th>
                <th className="text-right py-2 px-3 font-medium">Quantity</th>
                <th className="text-right py-2 px-3 font-medium">Reorder At</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
                <th className="text-right py-2 px-3 font-medium">Action</th>
              </tr>
            </thead>
            <tbody>
              {items.map(item => {
                const isLow = item.quantity <= item.reorder_threshold;
                return (
                  <tr key={item.product_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                    <td className="py-2.5 px-3 font-medium">{item.product_name}</td>
                    <td className="py-2.5 px-3 font-mono text-xs text-[var(--muted)]">{item.sku || '—'}</td>
                    <td className="py-2.5 px-3 text-right font-mono">{item.quantity}</td>
                    <td className="py-2.5 px-3 text-right font-mono text-[var(--muted)]">{item.reorder_threshold}</td>
                    <td className="py-2.5 px-3">
                      {isLow ? (
                        <span className="status-chip status-chip--critical">LOW</span>
                      ) : (
                        <span className="status-chip status-chip--stable">OK</span>
                      )}
                    </td>
                    <td className="py-2.5 px-3 text-right">
                      {adjusting === item.product_id ? (
                        <div className="flex items-center gap-1 justify-end">
                          <input
                            type="number"
                            value={adjustVal}
                            onChange={e => setAdjustVal(e.target.value)}
                            placeholder="New qty"
                            className="w-20 px-2 py-1 rounded border text-xs"
                            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                          />
                          <button onClick={() => handleAdjust(item.product_id)} className="px-2 py-1 text-xs button--primary rounded">Set</button>
                          <button onClick={() => setAdjusting(null)} className="px-2 py-1 text-xs button--secondary rounded">X</button>
                        </div>
                      ) : (
                        <button onClick={() => { setAdjusting(item.product_id); setAdjustVal(String(item.quantity)); }} className="text-xs text-[var(--link)] hover:underline">
                          Adjust
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

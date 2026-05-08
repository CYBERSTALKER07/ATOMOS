'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';

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
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <h1 className="text-xl font-bold tracking-tight text-[var(--foreground)]">Inventory</h1>
          <div className="flex gap-2 items-center">
            <input
              placeholder="Search products..."
              value={search}
              onChange={e => { setSearch(e.target.value); setLoading(true); }}
              className="px-3 py-1.5 rounded-lg border text-sm w-48 focus:ring-2 focus:ring-[var(--primary)] outline-none"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
            <label className="flex items-center gap-1.5 text-sm text-[var(--muted)] cursor-pointer hover:text-[var(--foreground)] transition-colors">
              <input type="checkbox" checked={lowOnly} onChange={e => { setLowOnly(e.target.checked); setLoading(true); }} className="rounded accent-[var(--primary)]" />
              Low stock only
            </label>
            <motion.button 
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => { setLoading(true); load(); }} 
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary hover-lift active-press"
            >
              <Icon name="refresh" size={16} />
            </motion.button>
          </div>
        </div>

        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : items.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-products.png"
            headline="No inventory items found"
            body={search || lowOnly ? "Try adjusting your search filters to find what you're looking for." : "There are no products in your inventory yet."}
          />
        ) : (
          <motion.div 
            initial={{ opacity: 0 }}
            animate={{ opacity: 1 }}
            className="overflow-x-auto rounded-xl border border-[var(--border)] bg-[var(--surface)]"
          >
            <table className="w-full text-sm">
              <thead>
                <tr className="table__header border-b border-[var(--border)] bg-[var(--default)]">
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Product</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">SKU</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Quantity</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Reorder At</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Action</th>
                </tr>
              </thead>
              <tbody>
                {items.map((item, index) => {
                  const isLow = item.quantity <= item.reorder_threshold;
                  return (
                    <motion.tr 
                      key={item.product_id} 
                      initial={{ opacity: 0, x: -10 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: index * 0.03 }}
                      className="table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors"
                    >
                      <td className="py-3 px-4 font-medium">{item.product_name}</td>
                      <td className="py-3 px-4 font-mono text-xs text-[var(--muted)]">{item.sku || '—'}</td>
                      <td className="py-3 px-4 text-right font-mono tabular-nums">{item.quantity}</td>
                      <td className="py-3 px-4 text-right font-mono text-[var(--muted)] tabular-nums">{item.reorder_threshold}</td>
                      <td className="py-3 px-4">
                        {isLow ? (
                          <span className="status-chip status-chip--critical">LOW</span>
                        ) : (
                          <span className="status-chip status-chip--stable">OK</span>
                        )}
                      </td>
                      <td className="py-3 px-4 text-right">
                        {adjusting === item.product_id ? (
                          <div className="flex items-center gap-1 justify-end">
                            <input
                              type="number"
                              value={adjustVal}
                              onChange={e => setAdjustVal(e.target.value)}
                              placeholder="New qty"
                              className="w-20 px-2 py-1 rounded border text-xs outline-none focus:ring-1 focus:ring-[var(--primary)]"
                              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)' }}
                            />
                            <motion.button 
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.9 }}
                              onClick={() => handleAdjust(item.product_id)} 
                              className="px-2 py-1 text-xs button--primary rounded hover-lift active-press"
                            >
                              Set
                            </motion.button>
                            <motion.button 
                              whileHover={{ scale: 1.1 }}
                              whileTap={{ scale: 0.9 }}
                              onClick={() => setAdjusting(null)} 
                              className="px-2 py-1 text-xs button--secondary rounded hover-lift active-press"
                            >
                              X
                            </motion.button>
                          </div>
                        ) : (
                          <motion.button 
                            whileHover={{ scale: 1.05, x: -2 }}
                            onClick={() => { setAdjusting(item.product_id); setAdjustVal(String(item.quantity)); }} 
                            className="text-xs text-[var(--link)] font-medium hover:underline flex items-center gap-1 ml-auto"
                          >
                            <Icon name="refresh" size={12} /> Adjust
                          </motion.button>
                        )}
                      </td>
                    </motion.tr>
                  );
                })}
              </tbody>
            </table>
          </motion.div>
        )}
      </div>
    </PageTransition>
  );
}

'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';

interface Product {
  product_id: string;
  name: string;
  sku_id: string;
  category: string;
  price_uzs: number;
  is_active: boolean;
}

export default function ProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [search, setSearch] = useState('');

  const load = useCallback(async () => {
    try {
      const q = search ? `?search=${encodeURIComponent(search)}` : '';
      const res = await apiFetch(`/v1/warehouse/ops/products${q}`);
      if (res.ok) {
        const data = await res.json();
        setProducts(data.products || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, [search]);

  useEffect(() => { load(); }, [load]);

  const fmt = (n: number) => new Intl.NumberFormat('uz-UZ').format(n);

  return (
    <PageTransition>
      <div className="p-6 space-y-4">
        <div className="flex items-center justify-between flex-wrap gap-3">
          <h1 className="text-xl font-bold tracking-tight text-[var(--foreground)]">Product Catalog</h1>
          <div className="flex gap-2 items-center">
            <input
              placeholder="Search products..."
              value={search}
              onChange={e => { setSearch(e.target.value); setLoading(true); }}
              className="px-3 py-1.5 rounded-lg border text-sm w-48 focus:ring-2 focus:ring-[var(--primary)] outline-none"
              style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
            />
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

        <p className="text-xs text-[var(--muted)]">Read-only view of supplier product catalog</p>

        {loading ? (
          <div className="space-y-1">
            {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
          </div>
        ) : products.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-products.png"
            headline="No products found"
            body={search ? `No products matching "${search}" were found in the catalog.` : "The product catalog is currently empty."}
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
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Name</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">SKU</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Category</th>
                  <th className="table__column text-right py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Price (UZS)</th>
                  <th className="table__column text-left py-3 px-4 font-medium uppercase tracking-wider text-[11px]">Status</th>
                </tr>
              </thead>
              <tbody>
                {products.map((p, index) => (
                  <motion.tr 
                    key={p.product_id} 
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: index * 0.03 }}
                    className="table__row border-b border-[var(--border)] last:border-0 hover:bg-[var(--default)]/50 transition-colors"
                  >
                    <td className="py-3 px-4 font-medium">{p.name}</td>
                    <td className="py-3 px-4 font-mono text-xs text-[var(--muted)]">{p.sku_id}</td>
                    <td className="py-3 px-4">{p.category || '—'}</td>
                    <td className="py-3 px-4 text-right font-mono tabular-nums">{fmt(p.price_uzs)}</td>
                    <td className="py-3 px-4">
                      <span className={`status-chip ${p.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                        {p.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                  </motion.tr>
                ))}
              </tbody>
            </table>
          </motion.div>
        )}
      </div>
    </PageTransition>
  );
}

'use client';

import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';

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
    <div className="p-6 space-y-4 md-animate-in">
      <div className="flex items-center justify-between flex-wrap gap-3">
        <h1 className="text-xl font-bold tracking-tight">Product Catalog</h1>
        <div className="flex gap-2 items-center">
          <input
            placeholder="Search products..."
            value={search}
            onChange={e => { setSearch(e.target.value); setLoading(true); }}
            className="px-3 py-1.5 rounded-lg border text-sm w-48"
            style={{ background: 'var(--field-background)', borderColor: 'var(--field-border)', color: 'var(--field-foreground)' }}
          />
          <button onClick={() => { setLoading(true); load(); }} className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary">
            <Icon name="refresh" size={16} />
          </button>
        </div>
      </div>

      <p className="text-xs text-[var(--muted)]">Read-only view of supplier product catalog</p>

      {loading ? (
        <div className="space-y-1">
          {Array.from({ length: 6 }).map((_, i) => <div key={i} className="md-skeleton md-skeleton-row" />)}
        </div>
      ) : products.length === 0 ? (
        <div className="flex flex-col items-center justify-center py-20 text-[var(--muted)]">
          <Icon name="catalog" size={48} className="mb-3 opacity-40" />
          <p className="text-sm">No products found</p>
        </div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">Name</th>
                <th className="text-left py-2 px-3 font-medium">SKU</th>
                <th className="text-left py-2 px-3 font-medium">Category</th>
                <th className="text-right py-2 px-3 font-medium">Price (UZS)</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
              </tr>
            </thead>
            <tbody>
              {products.map(p => (
                <tr key={p.product_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-medium">{p.name}</td>
                  <td className="py-2.5 px-3 font-mono text-xs text-[var(--muted)]">{p.sku_id}</td>
                  <td className="py-2.5 px-3">{p.category || '—'}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{fmt(p.price_uzs)}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${p.is_active ? 'status-chip--stable' : 'status-chip--draft'}`}>
                      {p.is_active ? 'Active' : 'Inactive'}
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

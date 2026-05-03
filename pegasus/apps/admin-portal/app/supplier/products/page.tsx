'use client';

import Image from 'next/image';
import { useState, useEffect, useMemo, useCallback } from 'react';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import { useRouter } from 'next/navigation';
import EmptyState from '@/components/EmptyState';
import Icon from '@/components/Icon';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

interface Product {
  sku_id: string;
  name: string;
  description: string;
  image_url: string;
  sell_by_block: boolean;
  units_per_block: number;
  base_price: number;
  is_active: boolean;
  category_id: string;
  category_name: string;
  volumetric_unit: number;
  minimum_order_qty: number;
  step_size: number;
  created_at: string;
}

interface CategoryOption {
  id: string;
  name: string;
}

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('uz-UZ').format(amount);
}

export default function MyProductsPage() {
  const [products, setProducts] = useState<Product[]>([]);
  const [categoryOptions, setCategoryOptions] = useState<CategoryOption[]>([]);
  const [activeCategory, setActiveCategory] = useState('all');
  const [search, setSearch] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [toggling, setToggling] = useState<string | null>(null);

  const token = useToken();
  const router = useRouter();

  const fetchProducts = useCallback(() => {
    if (!token) return;
    setError('');
    setLoading(true);

    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

    Promise.all([
      fetch(`${API}/v1/supplier/products`, { headers })
        .then(r => { if (!r.ok) throw new Error(`Fetch failed: ${r.status}`); return r.json(); })
        .then(json => setProducts(Array.isArray(json) ? json : (json.data || []))),
      fetch(`${API}/v1/supplier/profile`, { headers })
        .then(r => r.ok ? r.json() : null)
        .then(async profile => {
          if (!profile?.operating_categories?.length) return;
          try {
            const catRes = await fetch(`${API}/v1/catalog/platform-categories`);
            if (!catRes.ok) return;
            const catJson = await catRes.json();
            const allCats: { category_id: string; display_name: string }[] = catJson.data || [];
            const opCats: string[] = profile.operating_categories;
            const mapped = opCats
              .map((id: string) => {
                const found = allCats.find(c => c.category_id === id);
                return found ? { id: found.category_id, name: found.display_name } : null;
              })
              .filter(Boolean) as CategoryOption[];
            setCategoryOptions(mapped);
          } catch { /* non-critical */ }
        }),
    ])
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(() => { fetchProducts(); }, [fetchProducts]);

  const filtered = useMemo(() => {
    let list = products;
    if (activeCategory !== 'all') {
      list = list.filter(p => p.category_id === activeCategory);
    }
    if (search.trim()) {
      const q = search.toLowerCase();
      list = list.filter(p =>
        p.name.toLowerCase().includes(q) ||
        p.sku_id.toLowerCase().includes(q) ||
        (p.description && p.description.toLowerCase().includes(q))
      );
    }
    return list;
  }, [products, activeCategory, search]);

  const activeCount = filtered.filter(p => p.is_active).length;
  const totalValue = filtered.reduce((s, p) => s + p.base_price, 0);

  const handleToggleActive = async (p: Product) => {
    if (!token) return;
    setToggling(p.sku_id);
    try {
      const res = await fetch(`${API}/v1/supplier/products/${p.sku_id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ is_active: !p.is_active }),
      });
      if (!res.ok) throw new Error('Toggle failed');
      fetchProducts();
    } catch { /* silent */ }
    finally { setToggling(null); }
  };

  if (loading) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="w-6 h-6 border-2 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
      </div>
    );
  }

  if (error) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="p-6 rounded-2xl" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <header className="mb-8 flex items-start justify-between gap-4 flex-wrap">
        <div>
          <h1 className="md-typescale-headline-medium">My Products</h1>
          <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>
            Product catalog overview &mdash; {products.length} SKUs registered
          </p>
        </div>
        <a
          href="/supplier/catalog"
          className="inline-flex items-center gap-2 px-5 py-2.5 rounded-lg bg-accent-soft text-accent-soft-foreground hover:opacity-80 transition-opacity md-typescale-label-large"
        >
          <Icon name="add" size={18} />
          Add Product
        </a>
      </header>

      {/* KPI strip */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-6">
        {[
          { label: 'Total SKUs', value: String(filtered.length) },
          { label: 'Active', value: String(activeCount) },
          { label: 'Inactive', value: String(filtered.length - activeCount) },
          { label: 'Catalog Value', value: `${formatAmount(totalValue)}` },
        ].map(kpi => (
          <div key={kpi.label} className="md-card md-card-elevated p-4">
            <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>{kpi.label}</p>
            <p className="md-typescale-title-large mt-1" style={{ color: 'var(--foreground)' }}>{kpi.value}</p>
          </div>
        ))}
      </div>

      {/* Search + filter bar */}
      <div className="flex gap-3 mb-6 flex-wrap items-center">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <Icon name="search" size={18} className="absolute left-3 top-1/2 -translate-y-1/2 pointer-events-none text-muted" />
          <input
            type="text"
            placeholder="Search products..."
            value={search}
            onChange={e => setSearch(e.target.value)}
            className="md-input-outlined w-full pl-10 pr-4 py-2.5 text-sm rounded-xl"
          />
        </div>
        <Button
          variant="outline"
          onPress={fetchProducts}
          className="md-typescale-label-large flex items-center gap-1.5"
        >
          <Icon name="refresh" size={16} />
          Refresh
        </Button>
      </div>

      {/* Category chips */}
      {categoryOptions.length > 0 && (
        <div className="flex gap-2 mb-6 flex-wrap">
          <button
            type="button"
            onClick={() => setActiveCategory('all')}
            className="px-4 py-2 rounded-lg text-sm font-medium transition-all duration-150"
            style={{
              background: activeCategory === 'all' ? 'var(--accent-soft)' : 'var(--surface)',
              color: activeCategory === 'all' ? 'var(--accent-soft-foreground)' : 'var(--muted)',
              border: activeCategory === 'all' ? 'none' : '1px solid var(--border)',
            }}
          >
            All ({products.length})
          </button>
          {categoryOptions.map(cat => {
            const count = products.filter(p => p.category_id === cat.id).length;
            const isActive = activeCategory === cat.id;
            return (
              <button
                key={cat.id}
                type="button"
                onClick={() => setActiveCategory(cat.id)}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-all duration-150"
                style={{
                  background: isActive ? 'var(--accent-soft)' : 'var(--surface)',
                  color: isActive ? 'var(--accent-soft-foreground)' : 'var(--muted)',
                  border: isActive ? 'none' : '1px solid var(--border)',
                }}
              >
                {cat.name} ({count})
              </button>
            );
          })}
        </div>
      )}

      {/* Product Grid */}
      {filtered.length === 0 ? (
        <EmptyState
          icon="catalog"
          headline={search ? 'No matching products' : activeCategory === 'all' ? 'No products yet' : 'No products in this category'}
          body={search ? 'Try a different search term.' : 'Add products from the Catalog page.'}
        />
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
          {filtered.map(p => (
            <div
              key={p.sku_id}
              className="md-card md-card-elevated md-shape-lg overflow-hidden flex flex-col transition-shadow hover:shadow-lg cursor-pointer"
              style={{ opacity: p.is_active ? 1 : 0.6 }}
              onClick={() => router.push(`/supplier/products/${p.sku_id}`)}
            >
              {/* Image */}
              <div className="relative aspect-[4/3] overflow-hidden" style={{ background: 'var(--surface)' }}>
                {p.image_url ? (
                  <Image
                    src={p.image_url}
                    alt={p.name}
                    fill
                    sizes="(max-width: 640px) 100vw, (max-width: 1024px) 50vw, 25vw"
                    className="w-full h-full object-cover"
                  />
                ) : (
                  <div className="w-full h-full flex items-center justify-center">
                    <svg width="40" height="40" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--border)' }}>
                      <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
                    </svg>
                  </div>
                )}
                {/* Status badge */}
                <span
                  className="absolute top-2 right-2 text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider"
                  style={{
                    background: p.is_active
                      ? 'color-mix(in srgb, var(--success) 90%, transparent)'
                      : 'color-mix(in srgb, var(--danger) 90%, transparent)',
                    color: '#fff',
                  }}
                >
                  {p.is_active ? 'Active' : 'Inactive'}
                </span>
              </div>

              {/* Content */}
              <div className="p-4 flex-1 flex flex-col gap-2">
                {/* Category chip */}
                <span
                  className="text-[10px] font-semibold uppercase tracking-wider px-2 py-0.5 rounded-full self-start"
                  style={{ background: 'color-mix(in srgb, var(--muted) 12%, transparent)', color: 'var(--muted)' }}
                >
                  {p.category_name || p.category_id}
                </span>

                <h3 className="md-typescale-title-small font-semibold leading-tight" style={{ color: 'var(--foreground)' }}>
                  {p.name}
                </h3>

                {p.description && (
                  <p className="md-typescale-body-small line-clamp-2" style={{ color: 'var(--muted)' }}>
                    {p.description}
                  </p>
                )}

                <div className="mt-auto pt-3 flex items-end justify-between gap-2" style={{ borderTop: '1px solid var(--border)' }}>
                  <div>
                    <p className="md-typescale-title-medium font-bold font-mono" style={{ color: 'var(--foreground)' }}>
                      {formatAmount(p.base_price)} <span className="text-[10px] font-normal"></span>
                    </p>
                    <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                      {p.sell_by_block ? `${p.units_per_block} units/block` : 'Single unit'}
                    </p>
                  </div>

                  <Button
                    isIconOnly
                    variant="ghost"
                    onPress={() => handleToggleActive(p)}
                    isDisabled={toggling === p.sku_id}
                    className="w-9 h-9 rounded-full shrink-0"
                    style={{
                      background: p.is_active
                        ? 'color-mix(in srgb, var(--danger) 12%, transparent)'
                        : 'color-mix(in srgb, var(--success) 12%, transparent)',
                      color: p.is_active ? 'var(--danger)' : 'var(--success)',
                    }}
                    aria-label={p.is_active ? 'Deactivate' : 'Activate'}
                  >
                    {toggling === p.sku_id ? (
                      <div className="w-4 h-4 border-2 rounded-full animate-spin" style={{ borderColor: 'currentColor', borderTopColor: 'transparent' }} />
                    ) : (
                      <Icon name={p.is_active ? 'visibility_off' : 'visibility'} size={18} />
                    )}
                  </Button>
                </div>
              </div>

              {/* SKU footer */}
              <div className="px-4 py-2" style={{ background: 'var(--surface)', borderTop: '1px solid var(--border)' }}>
                <p className="md-typescale-label-small font-mono truncate" style={{ color: 'var(--border)' }}>
                  SKU: {p.sku_id}
                </p>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

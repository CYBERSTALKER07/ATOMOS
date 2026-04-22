'use client';

import Image from 'next/image';
import SupplierProductForm from '@/components/SupplierProductForm';
import SupplierPromotionForm from '@/components/SupplierPromotionForm';
import EmptyState from '@/components/EmptyState';
import { useState, useEffect, useMemo, useCallback, useRef } from 'react';
import { useToken } from '@/lib/auth';

const API = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const MAX_IMAGE_SIZE = 5 * 1024 * 1024;

interface CatalogProduct {
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

export default function CatalogDashboard() {
  const [catalog, setCatalog] = useState<CatalogProduct[]>([]);
  const [categoryOptions, setCategoryOptions] = useState<CategoryOption[]>([]);
  const [activeCategory, setActiveCategory] = useState('all');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Edit modal state
  const [editProduct, setEditProduct] = useState<CatalogProduct | null>(null);
  const [editForm, setEditForm] = useState<Record<string, unknown>>({});
  const [editFile, setEditFile] = useState<File | null>(null);
  const [editPreview, setEditPreview] = useState<string | null>(null);
  const [editStatus, setEditStatus] = useState('');
  const [toggling, setToggling] = useState<string | null>(null);
  const editFileRef = useRef<HTMLInputElement>(null);

  const token = useToken();

  const fetchCatalog = useCallback(() => {
    if (!token) return;
    setError('');
    setLoading(true);

    const headers = { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' };

    Promise.all([
      fetch(`${API}/v1/supplier/products`, { headers })
        .then(r => { if (!r.ok) throw new Error(`Catalog fetch failed: ${r.status}`); return r.json(); })
        .then(json => setCatalog(json.data || [])),
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
              .map(id => {
                const found = allCats.find(c => c.category_id === id);
                return found ? { id: found.category_id, name: found.display_name } : null;
              })
              .filter(Boolean) as CategoryOption[];
            setCategoryOptions(mapped);
          } catch { /* category metadata non-critical */ }
        }),
    ])
      .catch(e => setError(e.message))
      .finally(() => setLoading(false));
  }, [token]);

  useEffect(() => { fetchCatalog(); }, [fetchCatalog]);

  const filtered = useMemo(
    () => activeCategory === 'all' ? catalog : catalog.filter(p => p.category_id === activeCategory),
    [catalog, activeCategory],
  );

  const activeCount = filtered.filter(p => p.is_active).length;
  const totalValue = filtered.reduce((s, p) => s + p.base_price, 0);

  const availableSkus = catalog.map(p => ({
    id: p.sku_id,
    name: p.name,
    vu: p.volumetric_unit ?? 1.0,
  }));

  const openEdit = (p: CatalogProduct) => {
    setEditProduct(p);
    setEditForm({
      name: p.name,
      description: p.description,
      base_price: p.base_price,
      sell_by_block: p.sell_by_block,
      units_per_block: p.units_per_block,
      minimum_order_qty: p.minimum_order_qty,
      step_size: p.step_size,
    });
    setEditFile(null);
    setEditPreview(null);
    setEditStatus('');
  };

  const handleEditSubmit = async () => {
    if (!editProduct || !token) return;
    setEditStatus('SAVING...');

    try {
      let imageUrl: string | undefined;

      if (editFile) {
        if (!ALLOWED_IMAGE_TYPES.includes(editFile.type)) {
          setEditStatus('Image must be JPEG, PNG, or WebP.');
          return;
        }
        if (editFile.size > MAX_IMAGE_SIZE) {
          setEditStatus('Image must be under 5 MB.');
          return;
        }

        const ext = editFile.name.split('.').pop()?.toLowerCase() || 'jpg';
        const ticketRes = await fetch(`${API}/v1/supplier/products/upload-ticket?ext=${ext}`, {
          headers: { Authorization: `Bearer ${token}` },
        });
        if (!ticketRes.ok) throw new Error('Upload ticket rejected');
        const { upload_url, image_url } = await ticketRes.json();

        const isPlaceholder = upload_url.includes('placehold.co');
        if (!isPlaceholder) {
          const uploadRes = await fetch(upload_url, {
            method: 'PUT',
            body: editFile,
            headers: { 'Content-Type': editFile.type || 'image/jpeg' },
          });
          if (!uploadRes.ok) throw new Error('GCS upload failed');
        }
        imageUrl = image_url;
      }

      const payload: Record<string, unknown> = { ...editForm };
      if (imageUrl !== undefined) payload.image_url = imageUrl;

      const res = await fetch(`${API}/v1/supplier/products/${editProduct.sku_id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify(payload),
      });
      if (!res.ok) {
        const msg = await res.text();
        throw new Error(msg || 'Update failed');
      }

      setEditProduct(null);
      fetchCatalog();
    } catch (err: unknown) {
      setEditStatus(err instanceof Error ? err.message : String(err));
    }
  };

  const handleToggleActive = async (p: CatalogProduct) => {
    if (!token) return;
    setToggling(p.sku_id);
    try {
      const res = await fetch(`${API}/v1/supplier/products/${p.sku_id}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ is_active: !p.is_active }),
      });
      if (!res.ok) throw new Error('Toggle failed');
      fetchCatalog();
    } catch { /* silent — UI will stay stale */ }
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
      <header className="mb-8 pb-4" style={{ borderBottom: '1px solid var(--border)' }}>
        <h1 className="md-typescale-headline-medium">Inventory Control</h1>
        <p className="md-typescale-body-small mt-1" style={{ color: 'var(--muted)' }}>Catalog Injection, Product Ledger &amp; Promotional Routing</p>
      </header>

      {/* KPI strip */}
      <div className="grid grid-cols-2 sm:grid-cols-4 gap-4 mb-8">
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

      {/* ─── Category Filter Chips ─── */}
      {categoryOptions.length > 0 && (
        <div className="flex gap-2 mb-8 flex-wrap">
          <button
            type="button"
            onClick={() => setActiveCategory('all')}
            className="px-4 py-2 rounded-lg text-sm font-medium transition-all duration-150"
            style={{
              background: activeCategory === 'all'
                ? 'var(--accent-soft)'
                : 'var(--surface)',
              color: activeCategory === 'all'
                ? 'var(--accent-soft-foreground)'
                : 'var(--muted)',
              border: activeCategory === 'all'
                ? 'none'
                : '1px solid var(--border)',
            }}
          >
            All ({catalog.length})
          </button>
          {categoryOptions.map(cat => {
            const count = catalog.filter(p => p.category_id === cat.id).length;
            const isActive = activeCategory === cat.id;
            return (
              <button
                key={cat.id}
                type="button"
                onClick={() => setActiveCategory(cat.id)}
                className="px-4 py-2 rounded-lg text-sm font-medium transition-all duration-150"
                style={{
                  background: isActive
                    ? 'var(--accent-soft)'
                    : 'var(--surface)',
                  color: isActive
                    ? 'var(--accent-soft-foreground)'
                    : 'var(--muted)',
                  border: isActive
                    ? 'none'
                    : '1px solid var(--border)',
                }}
              >
                {cat.name} ({count})
              </button>
            );
          })}
        </div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-2 gap-8 items-start">
        <SupplierProductForm supplierToken={token} onProductCreated={fetchCatalog} />
        <SupplierPromotionForm supplierToken={token} availableSkus={availableSkus} />
      </div>

      {/* ─── Product Ledger ─── */}
      <div className="mt-10 rounded-2xl overflow-hidden" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
        <div className="px-6 py-4 flex items-center justify-between" style={{ borderBottom: '1px solid var(--border)' }}>
          <h3 className="text-base font-bold tracking-tight" style={{ color: 'var(--foreground)' }}>Active Product Ledger</h3>
          <span className="text-xs font-medium px-3 py-1 rounded-full" style={{ background: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)' }}>
            {filtered.length} products
          </span>
        </div>

        {filtered.length === 0 ? (
          <EmptyState
            icon="catalog"
            headline={activeCategory === 'all' ? 'No products in catalog' : 'No products in this category'}
            body={activeCategory === 'all' ? 'Add a product above to begin.' : 'Switch filter or add a new product.'}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr style={{ background: 'var(--surface)' }}>
                  <th className="text-left px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>Product</th>
                  <th className="text-left px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>Category</th>
                  <th className="text-right px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>Price</th>
                  <th className="text-center px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>VU</th>
                  <th className="text-center px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>Block</th>
                  <th className="text-center px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>MOQ / Step</th>
                  <th className="text-center px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>Status</th>
                  <th className="text-left px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>SKU ID</th>
                  <th className="text-center px-4 py-3 font-semibold" style={{ color: 'var(--muted)' }}>Actions</th>
                </tr>
              </thead>
              <tbody>
                {filtered.map((p, i) => (
                  <tr key={p.sku_id} style={{ borderBottom: '1px solid var(--border)', background: i % 2 === 0 ? 'transparent' : 'var(--background)' }}>
                    <td className="px-4 py-3">
                      <div className="flex items-center gap-3">
                        {p.image_url ? (
                          <Image
                            src={p.image_url}
                            alt={p.name}
                            width={40}
                            height={40}
                            className="w-10 h-10 rounded-lg object-cover shrink-0"
                            style={{ border: '1px solid var(--border)' }}
                            sizes="40px"
                          />
                        ) : (
                          <div className="w-10 h-10 rounded-lg flex items-center justify-center shrink-0" style={{ background: 'var(--surface)' }}>
                            <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--border)' }}>
                              <path d="M21 19V5c0-1.1-.9-2-2-2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2zM8.5 13.5l2.5 3.01L14.5 12l4.5 6H5l3.5-4.5z"/>
                            </svg>
                          </div>
                        )}
                        <div className="min-w-0">
                          <p className="font-semibold truncate" style={{ color: 'var(--foreground)' }}>{p.name}</p>
                          {p.description && (
                            <p className="text-xs truncate max-w-[200px]" style={{ color: 'var(--muted)' }}>{p.description}</p>
                          )}
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs font-medium px-2.5 py-1 rounded-full" style={{ background: 'color-mix(in srgb, var(--muted) 12%, transparent)', color: 'var(--muted)' }}>
                        {p.category_name || p.category_id}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-right font-mono font-semibold" style={{ color: 'var(--foreground)' }}>
                      {formatAmount(p.base_price)}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span className="text-xs font-bold px-2 py-0.5 rounded-full" style={{ background: 'color-mix(in srgb, var(--accent) 12%, transparent)', color: 'var(--accent)' }}>
                        {p.volumetric_unit}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center text-xs" style={{ color: 'var(--muted)' }}>
                      {p.sell_by_block ? `${p.units_per_block} u/blk` : 'Single'}
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span
                        className="text-xs font-mono px-2 py-0.5 rounded"
                        style={{ background: 'color-mix(in srgb, var(--muted) 12%, transparent)', color: 'var(--muted)' }}
                        title={`Min order: ${p.minimum_order_qty ?? 1} units · Step: ${p.step_size ?? 1} units/case`}
                      >
                        {p.minimum_order_qty ?? 1} / {p.step_size ?? 1}
                      </span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <span
                        className="text-xs font-semibold px-2.5 py-1 rounded-full"
                        style={{
                          background: p.is_active
                            ? 'color-mix(in srgb, var(--success) 15%, transparent)'
                            : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                          color: p.is_active ? 'var(--success)' : 'var(--danger)',
                        }}
                      >
                        {p.is_active ? 'Active' : 'Inactive'}
                      </span>
                    </td>
                    <td className="px-4 py-3">
                      <span className="text-xs font-mono" style={{ color: 'var(--muted)' }}>{p.sku_id.slice(0, 8)}…</span>
                    </td>
                    <td className="px-4 py-3 text-center">
                      <div className="flex items-center justify-center gap-2">
                        <button
                          type="button"
                          onClick={() => openEdit(p)}
                          className="text-xs font-medium px-3 py-1.5 rounded-lg transition-colors"
                          style={{ background: 'var(--accent-soft)', color: 'var(--accent-soft-foreground)' }}
                        >
                          Edit
                        </button>
                        <button
                          type="button"
                          disabled={toggling === p.sku_id}
                          onClick={() => handleToggleActive(p)}
                          className="text-xs font-medium px-3 py-1.5 rounded-lg transition-colors disabled:opacity-38"
                          style={{
                            background: p.is_active
                              ? 'color-mix(in srgb, var(--danger) 12%, transparent)'
                              : 'color-mix(in srgb, var(--success) 12%, transparent)',
                            color: p.is_active ? 'var(--danger)' : 'var(--success)',
                          }}
                        >
                          {toggling === p.sku_id ? '...' : p.is_active ? 'Deactivate' : 'Activate'}
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* ─── Edit Product Modal ─── */}
      {editProduct && (
        <div className="fixed inset-0 z-50 flex items-center justify-center" style={{ background: 'rgba(0,0,0,0.5)' }}>
          <div className="w-full max-w-lg rounded-2xl p-6 m-4 max-h-[90vh] overflow-y-auto" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>
            <div className="flex justify-between items-center mb-6">
              <h2 className="text-base font-bold">Edit Product</h2>
              <button type="button" onClick={() => setEditProduct(null)} className="text-xs font-medium px-3 py-1 rounded-lg" style={{ background: 'var(--surface)', color: 'var(--muted)' }}>Close</button>
            </div>

            {editStatus && (
              <div className="mb-4 p-3 rounded-lg text-xs" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>
                {editStatus}
              </div>
            )}

            <div className="space-y-4">
              <div>
                <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>Name</label>
                <input
                  className="w-full p-3 rounded-lg text-sm"
                  style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                  value={(editForm.name as string) || ''}
                  onChange={e => setEditForm({ ...editForm, name: e.target.value })}
                />
              </div>
              <div>
                <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>Description</label>
                <textarea
                  className="w-full p-3 rounded-lg text-sm resize-none"
                  rows={2}
                  style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                  value={(editForm.description as string) || ''}
                  onChange={e => setEditForm({ ...editForm, description: e.target.value })}
                />
              </div>
              <div>
                <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>Base Price (Amount)</label>
                <input
                  type="number"
                  min={1}
                  className="w-full p-3 rounded-lg text-sm"
                  style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                  value={(editForm.base_price as number) || ''}
                  onChange={e => setEditForm({ ...editForm, base_price: parseInt(e.target.value) || 0 })}
                />
              </div>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>MOQ</label>
                  <input type="number" min={1}
                    className="w-full p-3 rounded-lg text-sm"
                    style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                    value={(editForm.minimum_order_qty as number) || ''}
                    onChange={e => setEditForm({ ...editForm, minimum_order_qty: parseInt(e.target.value) || 1 })}
                  />
                </div>
                <div>
                  <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>Step</label>
                  <input type="number" min={1}
                    className="w-full p-3 rounded-lg text-sm"
                    style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                    value={(editForm.step_size as number) || ''}
                    onChange={e => setEditForm({ ...editForm, step_size: parseInt(e.target.value) || 1 })}
                  />
                </div>
                <div>
                  <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>Units/Block</label>
                  <input type="number" min={1}
                    className="w-full p-3 rounded-lg text-sm"
                    style={{ background: 'var(--background)', border: '1px solid var(--border)', color: 'var(--foreground)' }}
                    value={(editForm.units_per_block as number) || ''}
                    onChange={e => setEditForm({ ...editForm, units_per_block: parseInt(e.target.value) || 1 })}
                  />
                </div>
              </div>

              {/* Image Re-upload */}
              <div>
                <label className="text-[11px] font-medium block mb-1" style={{ color: 'var(--muted)' }}>Product Image</label>
                <div className="flex items-center gap-3 mb-2">
                  {(editPreview || editProduct.image_url) && (
                    <Image
                      src={editPreview || editProduct.image_url}
                      alt="Current"
                      width={48}
                      height={48}
                      unoptimized={Boolean(editPreview)}
                      className="w-12 h-12 rounded-lg object-cover"
                      style={{ border: '1px solid var(--border)' }}
                      sizes="48px"
                    />
                  )}
                  <span className="text-[11px]" style={{ color: 'var(--muted)' }}>
                    {editFile ? editFile.name : 'Keep current image'}
                  </span>
                </div>
                <input
                  ref={editFileRef}
                  type="file"
                  accept="image/jpeg,image/png,image/webp"
                  className="w-full p-3 text-xs rounded-lg cursor-pointer"
                  style={{ background: 'var(--background)', border: '1px dashed var(--border)' }}
                  onChange={e => {
                    const f = e.target.files?.[0] || null;
                    setEditFile(f);
                    setEditPreview(f ? URL.createObjectURL(f) : null);
                  }}
                />
              </div>

              <div className="flex gap-3 pt-2">
                <button
                  type="button"
                  onClick={() => setEditProduct(null)}
                  className="flex-1 py-3 rounded-full text-sm font-medium"
                  style={{ background: 'var(--surface)', color: 'var(--muted)' }}
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={handleEditSubmit}
                  disabled={editStatus === 'SAVING...'}
                  className="flex-1 py-3 rounded-full text-sm font-medium disabled:opacity-38"
                  style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
                >
                  {editStatus === 'SAVING...' ? 'Saving...' : 'Save Changes'}
                </button>
              </div>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

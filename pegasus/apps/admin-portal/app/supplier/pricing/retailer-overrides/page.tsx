'use client';

import React, { useState, useEffect, useCallback } from 'react';
import { Button } from '@heroui/react';
import { apiFetch } from '@/lib/auth';
import StatsCard from '@/components/StatsCard';
import Drawer from '@/components/Drawer';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';

/* ── Types ────────────────────────────────────────────────────────────────── */

interface PriceOverride {
  override_id: string;
  supplier_id: string;
  retailer_id: string;
  sku_id: string;
  price: number;
  set_by: string;
  set_by_role: string;
  is_active: boolean;
  notes?: string;
  expires_at?: string;
  created_at: string;
}

interface Product {
  sku_id: string;
  name: string;
  base_price: number;
}

type ProductApiRecord = Partial<{
  sku_id: string;
  name: string;
  base_price: number;
}>;

/* ── Helpers ──────────────────────────────────────────────────────────────── */

const fieldStyle = {
  background: 'var(--field-background)',
  color: 'var(--field-foreground)',
  border: '1px solid var(--field-border)',
  borderRadius: '8px',
};

function formatPrice(uzs: number): string {
  return new Intl.NumberFormat('en-US').format(uzs);
}

function normalizeProduct(input: unknown): Product | null {
  if (!input || typeof input !== 'object') return null;

  const raw = input as ProductApiRecord;
  if (typeof raw.sku_id !== 'string' || raw.sku_id.length === 0) return null;

  return {
    sku_id: raw.sku_id,
    name: typeof raw.name === 'string' ? raw.name : raw.sku_id,
    base_price: typeof raw.base_price === 'number' ? raw.base_price : 0,
  };
}

/* ── Page ─────────────────────────────────────────────────────────────────── */

export default function RetailerPricingOverridesPage() {
  const [overrides, setOverrides] = useState<PriceOverride[]>([]);
  const [products, setProducts] = useState<Product[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Filters
  const [filterRetailer, setFilterRetailer] = useState('');
  const [filterSku, setFilterSku] = useState('');

  // Drawer
  const [drawerOpen, setDrawerOpen] = useState(false);
  const [drawerMode, setDrawerMode] = useState<'create' | 'detail'>('create');
  const [selected, setSelected] = useState<PriceOverride | null>(null);

  // Form
  const [form, setForm] = useState({ retailer_id: '', sku_id: '', price: '', notes: '', expires_at: '' });
  const [submitting, setSubmitting] = useState(false);
  const [formError, setFormError] = useState('');

  /* ── Fetch ──────────────────────────────────────────────────────────────── */

  const fetchOverrides = useCallback(async () => {
    try {
      setLoading(true);
      let path = '/v1/supplier/pricing/retailer-overrides';
      const params = new URLSearchParams();
      if (filterRetailer) params.set('retailer_id', filterRetailer);
      if (filterSku) params.set('sku_id', filterSku);
      if (params.toString()) path += `?${params}`;

      const res = await apiFetch(path);
      if (!res.ok) throw new Error('Failed to fetch overrides');
      const data = await res.json();
      setOverrides(data.overrides || []);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [filterRetailer, filterSku]);

  const fetchProducts = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/supplier/products');
      if (!res.ok) return;
      const data = await res.json();
      const items = Array.isArray(data)
        ? data
        : Array.isArray(data?.data)
          ? data.data
          : Array.isArray(data?.products)
            ? data.products
            : [];
      setProducts(items.map(normalizeProduct).filter((product: Product | null): product is Product => product !== null));
    } catch { /* non-critical */ }
  }, []);

  useEffect(() => { fetchOverrides(); }, [fetchOverrides]);
  useEffect(() => { fetchProducts(); }, [fetchProducts]);

  /* ── Create ─────────────────────────────────────────────────────────────── */

  const handleCreate = async () => {
    setFormError('');
    if (!form.retailer_id || !form.sku_id || !form.price) {
      setFormError('Retailer ID, SKU, and price are required');
      return;
    }
    const price = parseInt(form.price, 10);
    if (isNaN(price) || price <= 0) {
      setFormError('Price must be a positive number');
      return;
    }

    setSubmitting(true);
    try {
      const res = await apiFetch('/v1/supplier/pricing/retailer-overrides', {
        method: 'POST',
        body: JSON.stringify({
          retailer_id: form.retailer_id,
          sku_id: form.sku_id,
          price,
          notes: form.notes || undefined,
          expires_at: form.expires_at || undefined,
        }),
      });
      if (!res.ok) {
        const data = await res.json();
        throw new Error(data.error || 'Failed to create override');
      }
      setDrawerOpen(false);
      setForm({ retailer_id: '', sku_id: '', price: '', notes: '', expires_at: '' });
      fetchOverrides();
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed');
    } finally {
      setSubmitting(false);
    }
  };

  /* ── Deactivate ─────────────────────────────────────────────────────────── */

  const handleDeactivate = async (id: string) => {
    if (!confirm('Deactivate this price override?')) return;
    const res = await apiFetch(`/v1/supplier/pricing/retailer-overrides/${id}`, { method: 'DELETE' });
    if (res.ok) {
      setDrawerOpen(false);
      fetchOverrides();
    } else {
      const data = await res.json();
      alert(data.error || 'Failed');
    }
  };

  /* ── Drawer handlers ────────────────────────────────────────────────────── */

  const openCreate = () => {
    setDrawerMode('create');
    setSelected(null);
    setFormError('');
    setDrawerOpen(true);
  };

  const openDetail = (o: PriceOverride) => {
    setDrawerMode('detail');
    setSelected(o);
    setDrawerOpen(true);
  };

  /* ── KPIs ───────────────────────────────────────────────────────────────── */

  const uniqueRetailers = new Set(overrides.map(o => o.retailer_id)).size;
  const uniqueSkus = new Set(overrides.map(o => o.sku_id)).size;
  const expiringSoon = overrides.filter(o => {
    if (!o.expires_at) return false;
    const diff = new Date(o.expires_at).getTime() - Date.now();
    return diff > 0 && diff < 7 * 24 * 60 * 60 * 1000; // 7 days
  }).length;

  /* ── Render ─────────────────────────────────────────────────────────────── */

  return (
    <div className="min-h-full p-6 md:p-10" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Header */}
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="md-typescale-headline-medium">Retailer Pricing Overrides</h1>
          <p className="md-typescale-body-medium mt-1" style={{ color: 'var(--muted)' }}>
            Set custom prices per retailer per SKU — overrides tier-based discounts
          </p>
        </div>
        <Button className="button--primary" onPress={openCreate}>
          <Icon name="pricing" size={18} className="mr-2" />
          New Override
        </Button>
      </div>

      {/* KPI row */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-4 mb-8">
        <StatsCard label="Active Overrides" value={String(overrides.length)} delay={0} />
        <StatsCard label="Retailers" value={String(uniqueRetailers)} delay={50} />
        <StatsCard label="SKUs" value={String(uniqueSkus)} delay={100} />
        <StatsCard label="Expiring Soon" value={String(expiringSoon)} sub="within 7 days" delay={150} accent="var(--color-md-warning, var(--accent))" />
      </div>

      {/* Filters */}
      <div className="flex gap-3 mb-6">
        <input
          className="w-56 px-3 py-2 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
          style={fieldStyle}
          placeholder="Filter by Retailer ID"
          value={filterRetailer}
          onChange={e => setFilterRetailer(e.target.value)}
          onKeyDown={e => e.key === 'Enter' && fetchOverrides()}
        />
        <select
          className="w-48 px-3 py-2 md-typescale-body-medium outline-none"
          style={fieldStyle}
          value={filterSku}
          onChange={e => { setFilterSku(e.target.value); }}
        >
          <option value="">All SKUs</option>
          {products.map(p => (
            <option key={p.sku_id} value={p.sku_id}>{p.name} ({p.sku_id.slice(0, 8)})</option>
          ))}
        </select>
        <Button size="sm" variant="outline" onPress={fetchOverrides}>
          Apply
        </Button>
      </div>

      {/* Table */}
      {loading ? (
        <div className="flex items-center justify-center py-20">
          <div className="w-8 h-8 border-2 border-t-transparent rounded-full animate-spin" style={{ borderColor: 'var(--accent)', borderTopColor: 'transparent' }} />
        </div>
      ) : error ? (
        <EmptyState icon="error" headline="Failed to load" body={error} action="Retry" onAction={fetchOverrides} />
      ) : overrides.length === 0 ? (
        <EmptyState icon="pricing" headline="No pricing overrides" body="Create a per-retailer price override to offer custom pricing." action="New Override" onAction={openCreate} />
      ) : (
        <div className="md-card md-card-elevated overflow-hidden">
          <table className="md-table w-full">
            <thead>
              <tr>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>SKU</th>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Retailer</th>
                <th className="table__column px-4 py-3 text-right md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Override Price</th>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Set By</th>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Expires</th>
                <th className="table__column px-4 py-3 text-left md-typescale-label-medium" style={{ color: 'var(--muted)' }}>Notes</th>
                <th className="table__column px-4 py-3 text-center md-typescale-label-medium" style={{ color: 'var(--muted)' }}></th>
              </tr>
            </thead>
            <tbody>
              {overrides.map((o, i) => {
                const product = products.find(p => p.sku_id === o.sku_id);
                return (
                  <tr
                    key={o.override_id}
                    onClick={() => openDetail(o)}
                    className="cursor-pointer transition-colors md-animate-in"
                    style={{ animationDelay: `${i * 20}ms`, borderBottom: '1px solid var(--border)' }}
                    onMouseEnter={e => (e.currentTarget.style.background = 'var(--surface)')}
                    onMouseLeave={e => (e.currentTarget.style.background = 'transparent')}
                  >
                    <td className="px-4 py-3">
                      <p className="md-typescale-body-medium font-medium">{product?.name || o.sku_id.slice(0, 12)}</p>
                      <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>{o.sku_id.slice(0, 8)}</p>
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)', fontVariantNumeric: 'tabular-nums' }}>
                      {o.retailer_id.slice(0, 12)}...
                    </td>
                    <td className="px-4 py-3 text-right md-typescale-body-medium font-medium" style={{ fontVariantNumeric: 'tabular-nums' }}>
                      {formatPrice(o.price)} UZS
                    </td>
                    <td className="px-4 py-3">
                      <span
                        className="md-typescale-label-small px-2 py-0.5 rounded-full"
                        style={{
                          background: o.set_by_role === 'NODE_ADMIN' ? 'var(--accent-soft)' : 'color-mix(in srgb, var(--success) 15%, transparent)',
                          color: o.set_by_role === 'NODE_ADMIN' ? 'var(--accent)' : 'var(--success)',
                        }}
                      >
                        {o.set_by_role}
                      </span>
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                      {o.expires_at ? new Date(o.expires_at).toLocaleDateString() : '—'}
                    </td>
                    <td className="px-4 py-3 md-typescale-body-small" style={{ color: 'var(--muted)' }}>
                      {o.notes || '—'}
                    </td>
                    <td className="px-4 py-3 text-center" onClick={e => e.stopPropagation()}>
                      <button
                        onClick={() => handleDeactivate(o.override_id)}
                        className="md-typescale-label-small px-2 py-1 rounded hover:opacity-80"
                        style={{ color: 'var(--danger)' }}
                        title="Deactivate"
                      >
                        <Icon name="cancel" size={16} />
                      </button>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {/* Drawer */}
      <Drawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        title={drawerMode === 'create' ? 'New Price Override' : 'Override Details'}
      >
        {drawerMode === 'create' && (
          <div className="p-6 space-y-5">
            {formError && (
              <div className="md-typescale-body-small px-3 py-2 rounded-lg" style={{ background: 'color-mix(in srgb, var(--danger) 10%, transparent)', color: 'var(--danger)' }}>
                {formError}
              </div>
            )}

            <div>
              <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Retailer ID *</label>
              <input
                className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
                style={fieldStyle}
                placeholder="Enter retailer UUID"
                value={form.retailer_id}
                onChange={e => setForm(f => ({ ...f, retailer_id: e.target.value }))}
              />
            </div>

            <div>
              <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Product (SKU) *</label>
              <select
                className="w-full px-3 py-2.5 md-typescale-body-medium outline-none"
                style={fieldStyle}
                value={form.sku_id}
                onChange={e => setForm(f => ({ ...f, sku_id: e.target.value }))}
              >
                <option value="">Select a product</option>
                {products.map(p => (
                  <option key={p.sku_id} value={p.sku_id}>{p.name}</option>
                ))}
              </select>
            </div>

            <div>
              <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Override Price (UZS) *</label>
              <input
                className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
                style={fieldStyle}
                type="number"
                min="1"
                placeholder="e.g. 125000"
                value={form.price}
                onChange={e => setForm(f => ({ ...f, price: e.target.value }))}
              />
            </div>

            <div>
              <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Expires At</label>
              <input
                className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)]"
                style={fieldStyle}
                type="datetime-local"
                value={form.expires_at}
                onChange={e => setForm(f => ({ ...f, expires_at: e.target.value ? new Date(e.target.value).toISOString() : '' }))}
              />
            </div>

            <div>
              <label className="md-typescale-label-medium block mb-1.5" style={{ color: 'var(--muted)' }}>Notes</label>
              <textarea
                className="w-full px-3 py-2.5 md-typescale-body-medium outline-none focus:ring-2 focus:ring-[var(--accent)] resize-none"
                style={fieldStyle}
                rows={3}
                placeholder="Reason for this override"
                value={form.notes}
                onChange={e => setForm(f => ({ ...f, notes: e.target.value }))}
              />
            </div>

            <div className="flex gap-3 pt-4" style={{ borderTop: '1px solid var(--border)' }}>
              <Button className="button--primary flex-1" isPending={submitting} onPress={handleCreate}>
                Create Override
              </Button>
              <Button variant="outline" className="flex-1" onPress={() => setDrawerOpen(false)}>
                Cancel
              </Button>
            </div>
          </div>
        )}

        {drawerMode === 'detail' && selected && (
          <div className="p-6 space-y-5">
            <div className="md-card md-card-elevated p-4 space-y-3">
              <div className="flex items-center justify-between">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>SKU</p>
                <p className="md-typescale-body-medium font-medium">{products.find(p => p.sku_id === selected.sku_id)?.name || selected.sku_id}</p>
              </div>
              <div className="flex items-center justify-between">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Retailer</p>
                <p className="md-typescale-body-small" style={{ fontVariantNumeric: 'tabular-nums' }}>{selected.retailer_id}</p>
              </div>
              <div className="flex items-center justify-between">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Override Price</p>
                <p className="md-typescale-headline-small font-medium" style={{ fontVariantNumeric: 'tabular-nums' }}>{formatPrice(selected.price)} UZS</p>
              </div>
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="md-card md-card-elevated p-3">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Set By</p>
                <span
                  className="md-typescale-label-small px-2 py-0.5 rounded-full inline-block mt-1"
                  style={{
                    background: selected.set_by_role === 'NODE_ADMIN' ? 'var(--accent-soft)' : 'color-mix(in srgb, var(--success) 15%, transparent)',
                    color: selected.set_by_role === 'NODE_ADMIN' ? 'var(--accent)' : 'var(--success)',
                  }}
                >
                  {selected.set_by_role}
                </span>
              </div>
              <div className="md-card md-card-elevated p-3">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Created</p>
                <p className="md-typescale-body-small mt-1">{new Date(selected.created_at).toLocaleDateString()}</p>
              </div>
            </div>

            {selected.expires_at && (
              <div className="md-card md-card-elevated p-3">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Expires</p>
                <p className="md-typescale-body-medium mt-1">{new Date(selected.expires_at).toLocaleString()}</p>
              </div>
            )}

            {selected.notes && (
              <div className="md-card md-card-elevated p-3">
                <p className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>Notes</p>
                <p className="md-typescale-body-medium mt-1">{selected.notes}</p>
              </div>
            )}

            <div className="pt-4" style={{ borderTop: '1px solid var(--border)' }}>
              <Button
                className="w-full border-[var(--color-md-error)] text-[var(--color-md-error)]"
                variant="outline"
                onPress={() => handleDeactivate(selected.override_id)}
              >
                Deactivate Override
              </Button>
            </div>
          </div>
        )}
      </Drawer>
    </div>
  );
}

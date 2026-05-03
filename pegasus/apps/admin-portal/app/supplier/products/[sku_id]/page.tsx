'use client';

import { useState, useEffect, useCallback } from 'react';
import { useParams, useRouter } from 'next/navigation';
import Image from 'next/image';
import { Button } from '@heroui/react';
import { useToken } from '@/lib/auth';
import Icon from '@/components/Icon';
import { buildSupplierProductUpdateIdempotencyKey } from '../../_shared/idempotency';

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
  length_cm?: number;
  width_cm?: number;
  height_cm?: number;
}

function formatAmount(amount: number): string {
  return new Intl.NumberFormat('uz-UZ').format(amount);
}

export default function ProductDetailPage() {
  const params = useParams();
  const router = useRouter();
  const token = useToken();
  const skuId = params.sku_id as string;

  const [product, setProduct] = useState<Product | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  // Edit mode
  const [editing, setEditing] = useState(false);
  const [saving, setSaving] = useState(false);
  const [saveMsg, setSaveMsg] = useState<{ ok: boolean; text: string } | null>(null);

  // Editable fields
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [imageUrl, setImageUrl] = useState('');
  const [basePrice, setBasePrice] = useState('');
  const [minimumOrderQty, setMinimumOrderQty] = useState('');
  const [stepSize, setStepSize] = useState('');
  const [unitsPerBlock, setUnitsPerBlock] = useState('');
  const [lengthCM, setLengthCM] = useState('');
  const [widthCM, setWidthCM] = useState('');
  const [heightCM, setHeightCM] = useState('');

  const fetchProduct = useCallback(async () => {
    if (!token || !skuId) return;
    setLoading(true);
    setError('');
    try {
      const res = await fetch(`${API}/v1/supplier/products/${encodeURIComponent(skuId)}`, {
        headers: { Authorization: `Bearer ${token}` },
      });
      if (!res.ok) throw new Error(res.status === 404 ? 'Product not found' : `HTTP ${res.status}`);
      const data: Product = await res.json();
      setProduct(data);
      // Populate form
      setName(data.name);
      setDescription(data.description || '');
      setImageUrl(data.image_url || '');
      setBasePrice(String(data.base_price));
      setMinimumOrderQty(String(data.minimum_order_qty));
      setStepSize(String(data.step_size));
      setUnitsPerBlock(String(data.units_per_block));
      setLengthCM(data.length_cm != null ? String(data.length_cm) : '');
      setWidthCM(data.width_cm != null ? String(data.width_cm) : '');
      setHeightCM(data.height_cm != null ? String(data.height_cm) : '');
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Unknown error');
    } finally {
      setLoading(false);
    }
  }, [token, skuId]);

  useEffect(() => { fetchProduct(); }, [fetchProduct]);

  const handleSave = async () => {
    if (!token || !product) return;
    setSaving(true);
    setSaveMsg(null);
    try {
      const body: Record<string, unknown> = {};
      if (name !== product.name) body.name = name;
      if (description !== (product.description || '')) body.description = description;
      if (imageUrl !== (product.image_url || '')) body.image_url = imageUrl;
      const newPrice = Number(basePrice);
      if (!isNaN(newPrice) && newPrice !== product.base_price) body.base_price = newPrice;
      const newMinQty = Number(minimumOrderQty);
      if (!isNaN(newMinQty) && newMinQty !== product.minimum_order_qty) body.minimum_order_qty = newMinQty;
      const newStep = Number(stepSize);
      if (!isNaN(newStep) && newStep !== product.step_size) body.step_size = newStep;
      const newUnits = Number(unitsPerBlock);
      if (!isNaN(newUnits) && newUnits !== product.units_per_block) body.units_per_block = newUnits;
      if (lengthCM) body.length_cm = Number(lengthCM);
      if (widthCM) body.width_cm = Number(widthCM);
      if (heightCM) body.height_cm = Number(heightCM);

      if (Object.keys(body).length === 0) {
        setSaveMsg({ ok: true, text: 'No changes to save' });
        setEditing(false);
        setSaving(false);
        return;
      }

      const res = await fetch(`${API}/v1/supplier/products/${encodeURIComponent(skuId)}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
          'Idempotency-Key': buildSupplierProductUpdateIdempotencyKey(skuId, body),
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const errText = await res.text();
        throw new Error(errText || `HTTP ${res.status}`);
      }
      setSaveMsg({ ok: true, text: 'Product updated' });
      setEditing(false);
      fetchProduct();
    } catch (e: unknown) {
      setSaveMsg({ ok: false, text: e instanceof Error ? e.message : 'Save failed' });
    } finally {
      setSaving(false);
    }
  };

  const handleToggleActive = async () => {
    if (!token || !product) return;
    setSaving(true);
    try {
      const payload = { is_active: !product.is_active };
      const res = await fetch(`${API}/v1/supplier/products/${encodeURIComponent(skuId)}`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
          'Idempotency-Key': buildSupplierProductUpdateIdempotencyKey(skuId, payload),
        },
        body: JSON.stringify(payload),
      });
      if (res.ok) fetchProduct();
    } finally {
      setSaving(false);
    }
  };

  if (loading) {
    return (
      <div className="min-h-full flex items-center justify-center" style={{ background: 'var(--background)' }}>
        <div className="w-6 h-6 border-2 rounded-full animate-spin" style={{ borderColor: 'var(--border)', borderTopColor: 'var(--accent)' }} />
      </div>
    );
  }

  if (error || !product) {
    return (
      <div className="min-h-full flex flex-col items-center justify-center gap-4" style={{ background: 'var(--background)' }}>
        <div className="p-6 rounded-2xl" style={{ background: 'var(--danger)', color: 'var(--danger-foreground)' }}>
          {error || 'Product not found'}
        </div>
        <Button variant="secondary" onPress={() => router.push('/supplier/products')}>
          Back to Products
        </Button>
      </div>
    );
  }

  return (
    <div className="min-h-full p-6 md:p-8" style={{ background: 'var(--background)', color: 'var(--foreground)' }}>
      {/* Back nav */}
      <button
        onClick={() => router.push('/supplier/products')}
        className="flex items-center gap-1.5 mb-6 md-typescale-label-large"
        style={{ color: 'var(--muted)' }}
      >
        <Icon name="arrow_back" size={18} />
        Back to Products
      </button>

      {/* Header row */}
      <header className="flex items-start justify-between gap-4 mb-8 flex-wrap">
        <div className="flex items-center gap-4">
          {/* Image thumbnail */}
          <div
            className="w-20 h-20 rounded-xl overflow-hidden shrink-0 flex items-center justify-center"
            style={{ background: 'var(--surface)' }}
          >
            {product.image_url ? (
              <Image
                src={product.image_url}
                alt={product.name}
                width={80}
                height={80}
                className="w-full h-full object-cover"
                sizes="80px"
              />
            ) : (
              <Icon name="image" size={32} className="text-muted" />
            )}
          </div>
          <div>
            <h1 className="md-typescale-headline-small">{product.name}</h1>
            <div className="flex items-center gap-2 mt-1">
              <span
                className="text-[10px] font-bold px-2 py-0.5 rounded-full uppercase tracking-wider"
                style={{
                  background: product.is_active
                    ? 'color-mix(in srgb, var(--success) 15%, transparent)'
                    : 'color-mix(in srgb, var(--danger) 15%, transparent)',
                  color: product.is_active ? 'var(--success)' : 'var(--danger)',
                }}
              >
                {product.is_active ? 'Active' : 'Inactive'}
              </span>
              <span className="md-typescale-label-small" style={{ color: 'var(--muted)' }}>
                {product.category_name || product.category_id}
              </span>
            </div>
            <p className="md-typescale-body-small font-mono mt-1" style={{ color: 'var(--border)' }}>
              SKU: {product.sku_id}
            </p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            onPress={handleToggleActive}
            isDisabled={saving}
            className="md-typescale-label-large"
          >
            {product.is_active ? 'Deactivate' : 'Activate'}
          </Button>
          {!editing ? (
            <Button variant="primary" onPress={() => setEditing(true)} className="md-typescale-label-large">
              Edit Product
            </Button>
          ) : (
            <>
              <Button variant="outline" onPress={() => { setEditing(false); fetchProduct(); }} className="md-typescale-label-large">
                Cancel
              </Button>
              <Button variant="primary" onPress={handleSave} isDisabled={saving} className="md-typescale-label-large">
                {saving ? 'Saving...' : 'Save Changes'}
              </Button>
            </>
          )}
        </div>
      </header>

      {/* Save toast */}
      {saveMsg && (
        <div
          className="mb-6 px-4 py-3 rounded-xl md-typescale-body-small"
          style={{
            background: saveMsg.ok ? 'color-mix(in srgb, var(--success) 12%, transparent)' : 'var(--danger)',
            color: saveMsg.ok ? 'var(--success)' : 'var(--danger-foreground)',
          }}
        >
          {saveMsg.text}
        </div>
      )}

      {/* Detail grid */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Left: Info */}
        <section className="md-card md-card-elevated md-shape-lg p-6">
          <h2 className="md-typescale-title-medium mb-5" style={{ color: 'var(--foreground)' }}>
            Product Details
          </h2>
          <div className="flex flex-col gap-4">
            <Field label="Name" editing={editing} value={name} onChange={setName} />
            <Field label="Description" editing={editing} value={description} onChange={setDescription} multiline />
            <Field label="Image URL" editing={editing} value={imageUrl} onChange={setImageUrl} placeholder="https://..." />
            <Field
              label="Base Price (Amount)"
              editing={editing}
              value={editing ? basePrice : `${formatAmount(product.base_price)}`}
              onChange={setBasePrice}
              type="number"
            />
          </div>
        </section>

        {/* Right: Logistics */}
        <section className="md-card md-card-elevated md-shape-lg p-6">
          <h2 className="md-typescale-title-medium mb-5" style={{ color: 'var(--foreground)' }}>
            Logistics &amp; Ordering
          </h2>
          <div className="flex flex-col gap-4">
            <div className="grid grid-cols-2 gap-4">
              <Field label="Min Order Qty" editing={editing} value={editing ? minimumOrderQty : String(product.minimum_order_qty)} onChange={setMinimumOrderQty} type="number" />
              <Field label="Step Size" editing={editing} value={editing ? stepSize : String(product.step_size)} onChange={setStepSize} type="number" />
            </div>
            <div className="grid grid-cols-2 gap-4">
              <ReadonlyField label="Sell by Block" value={product.sell_by_block ? 'Yes' : 'No'} />
              <Field label="Units/Block" editing={editing} value={editing ? unitsPerBlock : String(product.units_per_block)} onChange={setUnitsPerBlock} type="number" />
            </div>
            <ReadonlyField label="Volumetric Unit" value={`${product.volumetric_unit.toFixed(4)} VU`} />

            <h3 className="md-typescale-label-large mt-2" style={{ color: 'var(--muted)' }}>
              Dimensions (cm)
            </h3>
            <div className="grid grid-cols-3 gap-4">
              <Field label="Length" editing={editing} value={editing ? lengthCM : (product.length_cm != null ? String(product.length_cm) : '-')} onChange={setLengthCM} type="number" />
              <Field label="Width" editing={editing} value={editing ? widthCM : (product.width_cm != null ? String(product.width_cm) : '-')} onChange={setWidthCM} type="number" />
              <Field label="Height" editing={editing} value={editing ? heightCM : (product.height_cm != null ? String(product.height_cm) : '-')} onChange={setHeightCM} type="number" />
            </div>

            <ReadonlyField label="Created" value={new Date(product.created_at).toLocaleDateString('en-GB', { day: 'numeric', month: 'short', year: 'numeric' })} />
          </div>
        </section>
      </div>
    </div>
  );
}

// ── Shared field components ───────────────────────────────────────────────

function Field({
  label, editing, value, onChange, type = 'text', placeholder, multiline,
}: {
  label: string;
  editing: boolean;
  value: string;
  onChange: (v: string) => void;
  type?: string;
  placeholder?: string;
  multiline?: boolean;
}) {
  return (
    <div>
      <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>
        {label}
      </label>
      {editing ? (
        multiline ? (
          <textarea
            value={value}
            onChange={(e) => onChange(e.target.value)}
            rows={3}
            className="md-input-outlined w-full px-3 py-2 text-sm rounded-lg"
            placeholder={placeholder}
          />
        ) : (
          <input
            type={type}
            value={value}
            onChange={(e) => onChange(e.target.value)}
            className="md-input-outlined w-full px-3 py-2 text-sm rounded-lg"
            placeholder={placeholder}
          />
        )
      ) : (
        <p className="md-typescale-body-medium" style={{ color: 'var(--foreground)' }}>
          {value || '-'}
        </p>
      )}
    </div>
  );
}

function ReadonlyField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <label className="md-typescale-label-small block mb-1" style={{ color: 'var(--muted)' }}>
        {label}
      </label>
      <p className="md-typescale-body-medium" style={{ color: 'var(--foreground)' }}>
        {value}
      </p>
    </div>
  );
}

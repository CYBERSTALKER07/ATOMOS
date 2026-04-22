'use client';

import Image from 'next/image';
import { useEffect, useMemo, useRef, useState } from 'react';

type PlatformCategory = {
  category_id: string;
  display_name: string;
  icon_url?: string;
  display_order: number;
};

type SupplierProfile = {
  is_configured: boolean;
  operating_categories: string[];
};

type ProductFormData = {
  name: string;
  description: string;
  category_id: string;
  sell_by_block: boolean;
  units_per_block: number;
  base_price: number;
  minimum_order_qty: number;
  step_size: number;
};

const EMPTY_FORM: ProductFormData = {
  name: '',
  description: '',
  category_id: '',
  sell_by_block: false,
  units_per_block: 1,
  base_price: 0,
  minimum_order_qty: 1,
  step_size: 1,
};

function SkeletonBlock({ className = '' }: { className?: string }) {
  return (
    <div
      className={`rounded animate-pulse ${className}`.trim()}
      style={{ background: 'var(--surface)' }}
    />
  );
}

const ALLOWED_IMAGE_TYPES = ['image/jpeg', 'image/png', 'image/webp'];
const MAX_IMAGE_SIZE = 5 * 1024 * 1024; // 5 MB

export default function SupplierProductForm({ supplierToken, onProductCreated }: { supplierToken: string; onProductCreated?: () => void }) {
  const [file, setFile] = useState<File | null>(null);
  const [imagePreview, setImagePreview] = useState<string | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [status, setStatus] = useState<string>('AWAITING_INPUT');
  const [fieldError, setFieldError] = useState<string>('');
  const [catalogLoading, setCatalogLoading] = useState(true);
  const [vuPreset, setVuPreset] = useState('1.0');
  const [customVu, setCustomVu] = useState('');
  const [vuMode, setVuMode] = useState<'preset' | 'dimensions'>('preset');
  const [lengthCM, setLengthCM] = useState('');
  const [widthCM, setWidthCM] = useState('');
  const [heightCM, setHeightCM] = useState('');
  const [allCategories, setAllCategories] = useState<PlatformCategory[]>([]);
  const [allowedCategoryIds, setAllowedCategoryIds] = useState<string[]>([]);
  const [formData, setFormData] = useState<ProductFormData>(EMPTY_FORM);

  useEffect(() => {
    let isMounted = true;

    async function loadCatalogConstraints() {
      try {
        setCatalogLoading(true);
        const [profileRes, categoriesRes] = await Promise.all([
          fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/profile`, {
            headers: { Authorization: `Bearer ${supplierToken}` },
          }),
          fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/catalog/platform-categories`),
        ]);

        if (!profileRes.ok) {
          throw new Error('Supplier profile unavailable. Complete supplier onboarding first.');
        }
        if (!categoriesRes.ok) {
          throw new Error('Platform category catalog unavailable.');
        }

        const profile: SupplierProfile = await profileRes.json();
        const categoriesJson = await categoriesRes.json();
        const categories: PlatformCategory[] = categoriesJson.data || [];

        if (!isMounted) return;

        setAllCategories(categories);
        setAllowedCategoryIds(profile.operating_categories || []);
        setFormData((current) => ({
          ...current,
          category_id: current.category_id || profile.operating_categories?.[0] || '',
        }));

        if (!profile.is_configured || (profile.operating_categories || []).length === 0) {
          setFieldError('Complete supplier onboarding and choose operating categories before adding products.');
        } else {
          setFieldError('');
        }
      } catch (error: unknown) {
        if (!isMounted) return;
        setFieldError(error instanceof Error ? error.message : 'Failed to load supplier catalog settings.');
      } finally {
        if (isMounted) {
          setCatalogLoading(false);
        }
      }
    }

    void loadCatalogConstraints();
    return () => {
      isMounted = false;
    };
  }, [supplierToken]);

  const allowedCategories = useMemo(() => {
    const allowed = new Set(allowedCategoryIds);
    return allCategories.filter((category) => allowed.has(category.category_id));
  }, [allCategories, allowedCategoryIds]);

  const computedVU = useMemo(() => {
    if (vuMode !== 'dimensions') return null;
    const l = parseFloat(lengthCM);
    const w = parseFloat(widthCM);
    const h = parseFloat(heightCM);
    if (l > 0 && w > 0 && h > 0) return parseFloat(((l * w * h) / 5000).toFixed(4));
    return null;
  }, [vuMode, lengthCM, widthCM, heightCM]);

  const validateForm = () => {
    if (catalogLoading) return 'Loading supplier category access...';
    if (allowedCategories.length === 0) return 'No operating categories available. Configure supplier onboarding first.';
    if (formData.name.trim().length < 2) return 'Product name must be at least 2 characters.';
    if (!formData.category_id) return 'Choose a category for this product.';
    if (!allowedCategoryIds.includes(formData.category_id)) return 'Selected category is not enabled for this supplier.';
    if (formData.base_price <= 0) return 'Base price must be greater than zero.';
    if (formData.units_per_block <= 0) return 'Units per block must be at least 1.';
    if (formData.step_size < 1) return 'Step size must be at least 1.';
    if (formData.minimum_order_qty < formData.step_size) return 'Minimum order quantity must be ≥ step size.';
    if (!file) return 'Product image is required.';
    if (!ALLOWED_IMAGE_TYPES.includes(file.type)) return 'Image must be JPEG, PNG, or WebP.';
    if (file.size > MAX_IMAGE_SIZE) return 'Image must be under 5 MB.';
    if (vuMode === 'preset' && vuPreset === 'custom' && (!customVu || Number(customVu) <= 0)) return 'Custom VU must be greater than zero.';
    if (vuMode === 'dimensions' && (!lengthCM || !widthCM || !heightCM || computedVU === null)) return 'Provide all three dimensions (L × W × H) to compute VU.';
    return '';
  };

  const handleUploadAndSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    const validationError = validateForm();
    if (validationError) {
      setFieldError(validationError);
      setStatus('VALIDATION_BLOCKED');
      return;
    }

    setFieldError('');

    try {
      setStatus('REQUESTING_UPLOAD_TICKET...');
      const ext = file?.name.split('.').pop() || 'jpg';

      const ticketRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/products/upload-ticket?ext=${ext}`, {
        headers: { Authorization: `Bearer ${supplierToken}` },
      });

      if (!ticketRes.ok) throw new Error('Ticket rejected by matrix.');
      const { upload_url, image_url } = await ticketRes.json();

      // Skip actual GCS upload when backend returns a placeholder (local dev without credentials)
      const isPlaceholder = upload_url.includes('placehold.co');
      if (!isPlaceholder) {
        setStatus('UPLOADING_TO_STORAGE...');
        const uploadRes = await fetch(upload_url, {
          method: 'PUT',
          body: file,
          headers: { 'Content-Type': file?.type || 'image/jpeg' },
        });
        if (!uploadRes.ok) throw new Error('Google Cloud rejected the payload. Check CORS.');
      } else {
        setStatus('LOCAL_DEV_MODE — skipping GCS upload');
      }

      setStatus('LOCKING_SPANNER_LEDGER...');
      const vuValue = vuMode === 'dimensions'
        ? (computedVU ?? 1.0)
        : vuPreset === 'custom' ? parseFloat(customVu) || 1.0 : parseFloat(vuPreset);
      const payload: Record<string, unknown> = {
        ...formData,
        image_url,
        volumetric_unit: vuValue,
      };
      if (vuMode === 'dimensions' && lengthCM && widthCM && heightCM) {
        payload.length_cm = parseFloat(lengthCM);
        payload.width_cm = parseFloat(widthCM);
        payload.height_cm = parseFloat(heightCM);
      }

      const dbRes = await fetch(`${process.env.NEXT_PUBLIC_API_URL}/v1/supplier/products`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${supplierToken}`,
        },
        body: JSON.stringify(payload),
      });

      if (!dbRes.ok) {
        const message = await dbRes.text();
        throw new Error(message || 'Ledger write fault.');
      }

      setStatus('TRANSACTION_COMPLETE');
      onProductCreated?.();
      setFormData({
        ...EMPTY_FORM,
        category_id: allowedCategories[0]?.category_id || '',
      });
      setVuPreset('1.0');
      setCustomVu('');
      setVuMode('preset');
      setLengthCM('');
      setWidthCM('');
      setHeightCM('');
      setFile(null);
      setImagePreview(null);
      if (fileInputRef.current) fileInputRef.current.value = '';
    } catch (error: unknown) {
      setStatus(`CRITICAL_FAULT: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  if (catalogLoading) {
    return (
      <div className="w-full max-w-2xl md-shape-lg md-elevation-1 p-6 font-sans" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>
        <div className="flex justify-between items-center pb-4 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
          <div className="space-y-2">
            <SkeletonBlock className="h-4 w-36" />
            <SkeletonBlock className="h-3 w-64" />
          </div>
          <span className="text-[11px] font-medium" style={{ color: 'var(--accent)' }}>SYNCING CONFIG...</span>
        </div>
        <div className="space-y-6">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2"><SkeletonBlock className="h-3 w-24" /><SkeletonBlock className="h-12 w-full" /></div>
            <div className="space-y-2"><SkeletonBlock className="h-3 w-28" /><SkeletonBlock className="h-12 w-full" /></div>
          </div>
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            <div className="space-y-2"><SkeletonBlock className="h-3 w-20" /><SkeletonBlock className="h-12 w-full" /></div>
            <div className="space-y-2"><SkeletonBlock className="h-3 w-24" /><SkeletonBlock className="h-12 w-full" /></div>
          </div>
          <div className="space-y-2"><SkeletonBlock className="h-3 w-24" /><SkeletonBlock className="h-24 w-full" /></div>
          <SkeletonBlock className="h-24 w-full" />
          <SkeletonBlock className="h-20 w-full" />
          <SkeletonBlock className="h-14 w-full rounded-full" />
        </div>
      </div>
    );
  }

  return (
    <div className="w-full max-w-2xl md-shape-lg md-elevation-1 p-6 font-sans" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>
      <div className="flex justify-between items-center pb-4 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
        <div>
          <h2 className="text-[16px] font-medium">Catalog Injection</h2>
          <p className="text-[11px] mt-1" style={{ color: 'var(--muted)' }}>
            Only configured supplier categories are available for product creation.
          </p>
        </div>
        <span className="text-[11px] font-medium text-right" style={{ color: status.includes('FAULT') || status.includes('BLOCKED') ? 'var(--danger)' : 'var(--accent)' }}>
          {status}
        </span>
      </div>

      <form onSubmit={handleUploadAndSubmit} className="space-y-6">
        {fieldError && (
          <div className="md-card md-card-outlined p-4" style={{ background: 'var(--danger)', borderColor: 'var(--danger)' }}>
            <p className="text-[12px]" style={{ color: 'var(--danger-foreground)' }}>{fieldError}</p>
          </div>
        )}

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Product Name</label>
            <input
              required
              value={formData.name}
              className="w-full p-3 md-shape-xs text-[13px] transition-colors focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              onChange={(e) => setFormData({ ...formData, name: e.target.value })}
            />
          </div>
          <div>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Base Price (Amount)</label>
            <input
              required
              type="number"
              min="1"
              value={formData.base_price || ''}
              className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              onChange={(e) => setFormData({ ...formData, base_price: parseInt(e.target.value, 10) || 0 })}
            />
          </div>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Category</label>
            <select
              required
              disabled={catalogLoading || allowedCategories.length === 0}
              className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              value={formData.category_id}
              onChange={(e) => setFormData({ ...formData, category_id: e.target.value })}
            >
              <option value="">Select product category</option>
              {allowedCategories.map((category) => (
                <option key={category.category_id} value={category.category_id}>
                  {category.display_name}
                </option>
              ))}
            </select>
            <p className="text-[10px] mt-1" style={{ color: 'var(--muted)' }}>
              {catalogLoading ? 'Loading supplier categories...' : `${allowedCategories.length} categories enabled for this supplier`}
            </p>
          </div>

          <div>
            <div className="flex items-center justify-between mb-2">
              <label className="text-[11px] font-medium" style={{ color: 'var(--muted)' }}>Volumetric Unit</label>
              <div className="flex rounded overflow-hidden text-[10px] font-medium" style={{ border: '1px solid var(--muted)' }}>
                <button
                  type="button"
                  onClick={() => setVuMode('preset')}
                  className="px-3 py-1 transition-colors"
                  style={vuMode === 'preset'
                    ? { background: 'var(--accent)', color: 'var(--accent-foreground)' }
                    : { background: 'var(--background)', color: 'var(--muted)' }}
                >
                  Preset
                </button>
                <button
                  type="button"
                  onClick={() => setVuMode('dimensions')}
                  className="px-3 py-1 transition-colors"
                  style={vuMode === 'dimensions'
                    ? { background: 'var(--accent)', color: 'var(--accent-foreground)' }
                    : { background: 'var(--background)', color: 'var(--muted)' }}
                >
                  L×W×H
                </button>
              </div>
            </div>
            {vuMode === 'preset' ? (
              <select
                className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
                style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
                value={vuPreset}
                onChange={(e) => setVuPreset(e.target.value)}
              >
                <option value="0.01">Tiny (0.01 VU)</option>
                <option value="0.1">Small (0.1 VU)</option>
                <option value="0.5">Medium (0.5 VU)</option>
                <option value="1.0">Standard (1.0 VU)</option>
                <option value="2.0">Bulky (2.0 VU)</option>
                <option value="50.0">Pallet (50.0 VU)</option>
                <option value="custom">Custom</option>
              </select>
            ) : (
              <div className="space-y-2">
                <div className="grid grid-cols-3 gap-2">
                  {([['L', lengthCM, setLengthCM], ['W', widthCM, setWidthCM], ['H', heightCM, setHeightCM]] as [string, string, React.Dispatch<React.SetStateAction<string>>][]).map(([axis, val, setter]) => (
                    <div key={axis}>
                      <label className="block text-[10px] mb-1" style={{ color: 'var(--muted)' }}>{axis} (cm)</label>
                      <input
                        type="number" min="0.1" step="0.1"
                        value={val}
                        className="w-full p-2 md-shape-xs text-[12px] focus:outline-none"
                        style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
                        onChange={(e) => setter(e.target.value)}
                        placeholder="0"
                      />
                    </div>
                  ))}
                </div>
                <div className="flex items-center justify-between px-2 py-1 md-shape-xs" style={{ background: 'var(--accent-soft)' }}>
                  <span className="text-[10px]" style={{ color: 'var(--accent-soft-foreground)' }}>Computed VU</span>
                  <span className="text-[12px] font-semibold" style={{ color: 'var(--accent-soft-foreground)' }}>
                    {computedVU !== null ? `${computedVU} VU` : '—'}
                  </span>
                </div>
              </div>
            )}
          </div>
        </div>

        {vuMode === 'preset' && vuPreset === 'custom' && (
          <div>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Custom VU</label>
            <input
              type="number"
              step="0.01"
              min="0.01"
              value={customVu}
              className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              onChange={(e) => setCustomVu(e.target.value)}
              placeholder="e.g. 3.5"
            />
          </div>
        )}

        <div>
          <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Description</label>
          <textarea
            value={formData.description}
            className="w-full p-3 h-24 md-shape-xs text-[13px] focus:outline-none"
            style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
            onChange={(e) => setFormData({ ...formData, description: e.target.value })}
            placeholder="Retail-facing product description"
          />
        </div>

        <div className="p-4 md-shape-md" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
          <label className="flex items-center space-x-3 cursor-pointer mb-4">
            <input
              type="checkbox"
              className="w-5 h-5"
              style={{ accentColor: 'var(--accent)' }}
              checked={formData.sell_by_block}
              onChange={(e) => setFormData({ ...formData, sell_by_block: e.target.checked })}
            />
            <span className="text-[13px] font-medium">Sell strictly by the block (case / pallet)</span>
          </label>

          <div className={`${formData.sell_by_block ? 'opacity-100' : 'opacity-60'} transition-opacity`}>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Units Per Block</label>
            <input
              type="number"
              min="1"
              value={formData.units_per_block}
              className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              onChange={(e) => setFormData({ ...formData, units_per_block: parseInt(e.target.value, 10) || 1 })}
            />
          </div>
        </div>

        {/* Packaging Constraints — drives AI Worker UOM Collision Guard */}
        <div className="p-4 md-shape-md" style={{ background: 'var(--surface)', border: '1px solid var(--border)' }}>
          <div className="flex items-center gap-2 mb-1">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="currentColor" style={{ color: 'var(--accent)', flexShrink: 0 }}>
              <path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z"/>
            </svg>
            <p className="text-[12px] font-semibold" style={{ color: 'var(--foreground)' }}>Packaging Constraints (AI Auto-Order)</p>
          </div>
          <p className="text-[10px] mb-4" style={{ color: 'var(--muted)' }}>
            The AI Worker will ceil predicted quantities to the nearest <strong>Step Size</strong> multiple and enforce the <strong>MOQ</strong> before firing. Example: AI needs 14 bottles → Step&nbsp;24 → auto-rounds to 24 (1&nbsp;case).
          </p>
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Step Size (units/case)</label>
              <input
                type="number"
                min="1"
                value={formData.step_size}
                className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
                style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
                onChange={(e) => {
                  const s = parseInt(e.target.value, 10) || 1;
                  setFormData({
                    ...formData,
                    step_size: s,
                    minimum_order_qty: Math.max(formData.minimum_order_qty, s),
                  });
                }}
              />
              <p className="text-[10px] mt-1" style={{ color: 'var(--muted)' }}>Order qty must be a multiple of this (e.g. 24 for a case of 24)</p>
            </div>
            <div>
              <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Min Order Qty (MOQ)</label>
              <input
                type="number"
                min={formData.step_size}
                step={formData.step_size}
                value={formData.minimum_order_qty}
                className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
                style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
                onChange={(e) => setFormData({ ...formData, minimum_order_qty: parseInt(e.target.value, 10) || formData.step_size })}
              />
              <p className="text-[10px] mt-1" style={{ color: 'var(--muted)' }}>Minimum total units per order (must be ≥ step size)</p>
            </div>
          </div>
        </div>

        <div>
          <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Product Image (High Res)</label>
          {imagePreview && (
            <div className="mb-3 flex items-center gap-3">
              <Image
                src={imagePreview}
                alt="Preview"
                width={64}
                height={64}
                unoptimized
                className="w-16 h-16 rounded-lg object-cover"
                style={{ border: '1px solid var(--border)' }}
              />
              <span className="text-[11px]" style={{ color: 'var(--muted)' }}>{file?.name} — {file ? (file.size / 1024).toFixed(0) : 0} KB</span>
            </div>
          )}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/jpeg,image/png,image/webp"
            className="w-full p-8 text-center cursor-pointer md-shape-md text-[13px] transition-colors file:mr-4 file:py-2 file:px-4 file:border-0 file:text-sm file:font-medium file:md-shape-full"
            style={{ background: 'var(--background)', border: '2px dashed var(--border)', color: 'var(--muted)' }}
            onChange={(e) => {
              const f = e.target.files?.[0] || null;
              setFile(f);
              if (f) {
                setImagePreview(URL.createObjectURL(f));
              } else {
                setImagePreview(null);
              }
            }}
          />
        </div>

        <button
          type="submit"
          disabled={catalogLoading || status.includes('UPLOADING') || status.includes('LOCKING')}
          className="w-full font-medium text-[14px] py-4 md-shape-full transition-colors disabled:opacity-38"
          style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
        >
          Execute Injection
        </button>
      </form>
    </div>
  );
}
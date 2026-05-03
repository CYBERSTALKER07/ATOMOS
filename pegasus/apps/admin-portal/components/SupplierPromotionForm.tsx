'use client';

import { useState } from 'react';
import { buildSupplierPricingRuleUpsertIdempotencyKey } from '@/app/supplier/_shared/idempotency';
import { apiFetch } from '@/lib/auth';

export default function SupplierPromotionForm({ availableSkus }: { availableSkus: { id: string; name: string }[] }) {
  const [status, setStatus] = useState<string>('AWAITING_INPUT');
  const [formData, setFormData] = useState({
    sku_id: '',
    min_pallets: 10,
    discount_percent: 5,
    valid_until: ''
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!formData.sku_id) {
      setStatus('FAULT: TARGET SKU REQUIRED');
      return;
    }

    try {
      setStatus('LOCKING_PROMOTION...');
      const tierId = crypto.randomUUID();
      const payload = { ...formData, tier_id: tierId };
      const res = await apiFetch('/v1/supplier/pricing/rules', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': buildSupplierPricingRuleUpsertIdempotencyKey(payload),
        },
        body: JSON.stringify(payload)
      });
      const body = await res.json().catch(() => ({} as { queued?: boolean; error?: string; message?: string }));

      if (body.queued) {
        setStatus('QUEUED_OFFLINE');
        return;
      }
      if (!res.ok) throw new Error(body.error || body.message || 'Ledger rejected promotion matrix.');
      
      setStatus('PROMOTION_LOCKED');
    } catch (error: unknown) {
      setStatus(`FAULT: ${error instanceof Error ? error.message : String(error)}`);
    }
  };

  return (
    <div className="w-full md-shape-lg md-elevation-1 p-6 font-sans" style={{ background: 'var(--surface)', color: 'var(--foreground)' }}>
      <div className="flex justify-between items-center pb-4 mb-6" style={{ borderBottom: '1px solid var(--border)' }}>
        <h2 className="text-[16px] font-medium">Promotion Engine</h2>
        <span className="text-[11px] font-medium" style={{ color: status.includes('FAULT') ? 'var(--danger)' : 'var(--accent)' }}>
          {status}
        </span>
      </div>

      <form onSubmit={handleSubmit} className="space-y-6">
        <div>
          <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Target SKU</label>
          <select
            required
            className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
            style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
            onChange={(e) => setFormData({...formData, sku_id: e.target.value})}
          >
            <option value="">Select target SKU</option>
            {availableSkus.map(sku => (
              <option key={sku.id} value={sku.id}>{sku.name} ({sku.id})</option>
            ))}
          </select>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Min Pallets (Threshold)</label>
            <input
              required
              type="number"
              min="1"
              className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              onChange={(e) => setFormData({...formData, min_pallets: parseInt(e.target.value)})}
            />
          </div>
          <div>
            <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Discount (%)</label>
            <input
              required
              type="number"
              min="1"
              max="100"
              className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
              style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
              onChange={(e) => setFormData({...formData, discount_percent: parseInt(e.target.value)})}
            />
          </div>
        </div>

        <div>
          <label className="block text-[11px] font-medium mb-2" style={{ color: 'var(--muted)' }}>Valid Until</label>
          <input
            required
            type="datetime-local"
            className="w-full p-3 md-shape-xs text-[13px] focus:outline-none"
            style={{ background: 'var(--background)', border: '1px solid var(--muted)', color: 'var(--foreground)' }}
            onChange={(e) => setFormData({...formData, valid_until: new Date(e.target.value).toISOString()})}
          />
        </div>

        {/* M3 Filled Button */}
        <button
          type="submit"
          disabled={status.includes('LOCKING')}
          className="w-full font-medium text-[14px] py-4 md-shape-full transition-colors disabled:opacity-38"
          style={{ background: 'var(--accent)', color: 'var(--accent-foreground)' }}
        >
          Inject Discount Rule
        </button>
      </form>
    </div>
  );
}

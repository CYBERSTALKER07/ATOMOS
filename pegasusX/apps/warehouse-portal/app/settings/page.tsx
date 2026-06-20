'use client';

import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { LocationPicker, type LocationValue } from '@/components/LocationPicker';
import type { DeliveryFeeRules, WarehouseOpsSettings, WarehouseOpsSettingsPatchRequest } from '@pegasusx/types';

type WarehouseLocation = {
  warehouse_id: string;
  name: string;
  address?: string;
  place_id?: string;
  lat: number;
  lng: number;
};

export default function WarehouseSettingsPage() {
  const { toast } = useToast();
  const [settings, setSettings] = useState<WarehouseOpsSettings | null>(null);
  const [policy, setPolicy] = useState<'REJECT' | 'ACCEPT_BACKORDER'>('REJECT');
  const [showStockCounts, setShowStockCounts] = useState(false);
  const [preorderMinLeadDays, setPreorderMinLeadDays] = useState('3');
  const [preorderMaxLeadDays, setPreorderMaxLeadDays] = useState('90');
  const [orderLineMin, setOrderLineMin] = useState('');
  const [orderLineMax, setOrderLineMax] = useState('');
  const [feeBaseMinor, setFeeBaseMinor] = useState('0');
  const [feeCurrency, setFeeCurrency] = useState('UZS');
  const [feeTierKm, setFeeTierKm] = useState('5');
  const [feeTierMinor, setFeeTierMinor] = useState('100000');
  const [scheduleJSON, setScheduleJSON] = useState('{\n  "is_24h": true\n}');
  const [location, setLocation] = useState<LocationValue>({ address: '', lat: '0', lng: '0' });
  const [saving, setSaving] = useState(false);
  const [savingLocation, setSavingLocation] = useState(false);

  const load = useCallback(async () => {
    const res = await apiFetch('/v1/warehouse/ops/settings');
    if (res.ok) {
      const data = (await res.json()) as WarehouseOpsSettings;
      setSettings(data);
      setPolicy(data.default_out_of_stock_policy === 'ACCEPT_BACKORDER' ? 'ACCEPT_BACKORDER' : 'REJECT');
      setShowStockCounts(Boolean(data.show_stock_counts_to_retailers));
      setPreorderMinLeadDays(String(data.preorder_min_lead_days ?? 3));
      setPreorderMaxLeadDays(String(data.preorder_max_lead_days ?? 90));
      setOrderLineMin(data.order_line_min_quantity != null ? String(data.order_line_min_quantity) : '');
      setOrderLineMax(data.order_line_max_quantity != null ? String(data.order_line_max_quantity) : '');
      const rules = data.delivery_fee_rules;
      if (rules) {
        setFeeBaseMinor(String(rules.base_fee_minor ?? 0));
        setFeeCurrency(rules.currency || 'UZS');
        const openTier = rules.tiers?.find((t) => t.max_km == null);
        const nearTier = rules.tiers?.find((t) => t.max_km != null);
        if (nearTier?.max_km != null) setFeeTierKm(String(nearTier.max_km));
        if (openTier) setFeeTierMinor(String(openTier.fee_minor));
      }
      if (data.operating_schedule) {
        setScheduleJSON(JSON.stringify(data.operating_schedule, null, 2));
      }
    }
    const locRes = await apiFetch('/v1/warehouse/ops/location');
    if (locRes.ok) {
      const loc = (await locRes.json()) as WarehouseLocation;
      setLocation({
        address: loc.address ?? '',
        lat: String(loc.lat ?? 0),
        lng: String(loc.lng ?? 0),
        place_id: loc.place_id,
      });
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    setSaving(true);
    try {
      let operating_schedule: unknown;
      try {
        operating_schedule = JSON.parse(scheduleJSON);
      } catch {
        toast('Operating schedule must be valid JSON', 'error');
        return;
      }
      const minLead = Number.parseInt(preorderMinLeadDays, 10);
      const maxLead = Number.parseInt(preorderMaxLeadDays, 10);
      if (!Number.isFinite(minLead) || !Number.isFinite(maxLead)) {
        toast('Pre-order lead days must be valid numbers', 'error');
        return;
      }
      const deliveryFeeRules: DeliveryFeeRules = {
        currency: feeCurrency.trim() || 'UZS',
        base_fee_minor: Number.parseInt(feeBaseMinor, 10) || 0,
        tiers: [
          { max_km: Number.parseFloat(feeTierKm), fee_minor: 0 },
          { max_km: null, fee_minor: Number.parseInt(feeTierMinor, 10) || 0 },
        ],
      };
      const patch: WarehouseOpsSettingsPatchRequest = {
        default_out_of_stock_policy: policy,
        show_stock_counts_to_retailers: showStockCounts,
        operating_schedule: operating_schedule as Record<string, unknown>,
        preorder_min_lead_days: minLead,
        preorder_max_lead_days: maxLead,
        delivery_fee_rules: deliveryFeeRules,
      };
      const minQty = orderLineMin.trim() ? Number.parseInt(orderLineMin, 10) : null;
      const maxQty = orderLineMax.trim() ? Number.parseInt(orderLineMax, 10) : null;
      if (orderLineMin.trim()) {
        patch.order_line_min_quantity = minQty;
      } else {
        patch.clear_order_line_min_quantity = true;
      }
      if (orderLineMax.trim()) {
        patch.order_line_max_quantity = maxQty;
      } else {
        patch.clear_order_line_max_quantity = true;
      }
      const res = await apiFetch('/v1/warehouse/ops/settings', {
        method: 'PATCH',
        body: JSON.stringify(patch),
      });
      if (res.ok) {
        toast('Warehouse settings saved', 'success');
        await load();
      } else {
        const data = await res.json().catch(() => ({}));
        toast((data as { error?: string }).error || 'Save failed', 'error');
      }
    } finally {
      setSaving(false);
    }
  }

  async function saveLocation() {
    if (!location.address.trim()) {
      toast('Depot address is required', 'error');
      return;
    }
    setSavingLocation(true);
    try {
      const res = await apiFetch('/v1/warehouse/ops/location', {
        method: 'PATCH',
        body: JSON.stringify({
          address: location.address.trim(),
          place_id: location.place_id,
          lat: Number.parseFloat(location.lat),
          lng: Number.parseFloat(location.lng),
        }),
      });
      if (res.ok) {
        toast('Depot location saved', 'success');
        await load();
      } else {
        const data = await res.json().catch(() => ({}));
        toast((data as { error?: string }).error || 'Save failed', 'error');
      }
    } finally {
      setSavingLocation(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        title="Warehouse settings"
        description="Checkout policy, pre-order lead window, per-line quantity limits, delivery surcharges, and depot location."
      >
        <div className="max-w-2xl space-y-6">
          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Depot location</h2>
            <p className="text-xs text-[var(--muted)]">
              Delivery fee distance is measured warehouse lat/lng → retailer delivery pin at checkout.
            </p>
            <LocationPicker value={location} onChange={setLocation} label="Warehouse address" />
            <button
              type="button"
              disabled={savingLocation}
              onClick={() => void saveLocation()}
              className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
            >
              {savingLocation ? 'Saving…' : 'Save location'}
            </button>
          </section>

          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Pre-order lead window (days)</h2>
            <div className="grid grid-cols-2 gap-3">
              <label className="text-sm">
                Min lead
                <input className="w-full mt-1 rounded border p-2 text-sm" value={preorderMinLeadDays} onChange={(e) => setPreorderMinLeadDays(e.target.value)} />
              </label>
              <label className="text-sm">
                Max lead
                <input className="w-full mt-1 rounded border p-2 text-sm" value={preorderMaxLeadDays} onChange={(e) => setPreorderMaxLeadDays(e.target.value)} />
              </label>
            </div>
          </section>

          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Line quantity limits</h2>
            <p className="text-xs text-[var(--muted)]">Leave blank to clear a limit. Applies to standard and scheduled checkout.</p>
            <div className="grid grid-cols-2 gap-3">
              <label className="text-sm">
                Min per SKU
                <input className="w-full mt-1 rounded border p-2 text-sm" value={orderLineMin} onChange={(e) => setOrderLineMin(e.target.value)} />
              </label>
              <label className="text-sm">
                Max per SKU
                <input className="w-full mt-1 rounded border p-2 text-sm" value={orderLineMax} onChange={(e) => setOrderLineMax(e.target.value)} />
              </label>
            </div>
          </section>

          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Delivery fee tiers</h2>
            <div className="grid grid-cols-2 gap-3">
              <label className="text-sm">
                Base fee (minor)
                <input className="w-full mt-1 rounded border p-2 text-sm" value={feeBaseMinor} onChange={(e) => setFeeBaseMinor(e.target.value)} />
              </label>
              <label className="text-sm">
                Currency
                <input className="w-full mt-1 rounded border p-2 text-sm" value={feeCurrency} onChange={(e) => setFeeCurrency(e.target.value)} />
              </label>
              <label className="text-sm">
                Free within km
                <input className="w-full mt-1 rounded border p-2 text-sm" value={feeTierKm} onChange={(e) => setFeeTierKm(e.target.value)} />
              </label>
              <label className="text-sm">
                Beyond-tier fee (minor)
                <input className="w-full mt-1 rounded border p-2 text-sm" value={feeTierMinor} onChange={(e) => setFeeTierMinor(e.target.value)} />
              </label>
            </div>
          </section>

          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Out-of-stock orders</h2>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={showStockCounts} onChange={(e) => setShowStockCounts(e.target.checked)} />
              Show stock counts to retailers
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="radio" checked={policy === 'REJECT'} onChange={() => setPolicy('REJECT')} />
              Reject orders when out of stock
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="radio" checked={policy === 'ACCEPT_BACKORDER'} onChange={() => setPolicy('ACCEPT_BACKORDER')} />
              Accept orders — warn retailer, fulfill when stock arrives
            </label>
          </section>

          <section className="border border-[var(--border)] rounded-xl p-4 space-y-3">
            <h2 className="text-sm font-semibold">Operating hours (display only)</h2>
            <textarea
              className="w-full min-h-[140px] font-mono text-xs rounded-lg border p-3"
              style={{ borderColor: 'var(--field-border)', background: 'var(--field-background)' }}
              value={scheduleJSON}
              onChange={(e) => setScheduleJSON(e.target.value)}
            />
          </section>

          <button
            type="button"
            disabled={saving}
            onClick={() => void save()}
            className="px-4 py-2 rounded-lg text-sm font-semibold button--primary disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save settings'}
          </button>

          {settings?.ops_always_available && (
            <p className="text-xs text-[var(--muted)]">
              Ops note: warehouse dispatch is always available regardless of operating hours.
            </p>
          )}
        </div>
      </PageChrome>
    </PageTransition>
  );
}

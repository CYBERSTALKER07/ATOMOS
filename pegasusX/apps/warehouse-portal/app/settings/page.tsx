'use client';

import { useCallback, useEffect, useState, type ChangeEvent } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { useToast } from '@/components/Toast';
import { LocationPicker, resolveLocationValue, type LocationValue } from '@/components/LocationPicker';
import { hasValidCoordinates } from '@/lib/geocode';
import { PortalField, PortalInput, PortalSection } from '@/components/portal';
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
  const [enforceOrderAcceptance, setEnforceOrderAcceptance] = useState(false);
  const [scheduleIs24h, setScheduleIs24h] = useState(true);
  const [scheduleTimezone, setScheduleTimezone] = useState('UTC');
  const [weekdayOpen, setWeekdayOpen] = useState('09:00');
  const [weekdayClose, setWeekdayClose] = useState('17:00');
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
        const sched = data.operating_schedule as Record<string, unknown>;
        setEnforceOrderAcceptance(Boolean(sched.enforce_order_acceptance));
        setScheduleIs24h(Boolean(sched.is_24h ?? true));
        if (typeof sched.timezone === 'string') setScheduleTimezone(sched.timezone);
        const weekdays = sched.schedules as Record<string, { open?: string; close?: string }> | undefined;
        const mon = weekdays?.monday;
        if (mon?.open) setWeekdayOpen(mon.open);
        if (mon?.close) setWeekdayClose(mon.close);
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
      let operating_schedule: Record<string, unknown>;
      try {
        operating_schedule = JSON.parse(scheduleJSON) as Record<string, unknown>;
      } catch {
        operating_schedule = {};
      }
      const weekdayWindow = { open: weekdayOpen, close: weekdayClose };
      operating_schedule = {
        ...operating_schedule,
        enforce_order_acceptance: enforceOrderAcceptance,
        is_24h: scheduleIs24h,
        timezone: scheduleTimezone,
        schedules: {
          monday: weekdayWindow,
          tuesday: weekdayWindow,
          wednesday: weekdayWindow,
          thursday: weekdayWindow,
          friday: weekdayWindow,
        },
      };
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
      let resolved = location;
      if (!hasValidCoordinates(location.lat, location.lng)) {
        const next = await resolveLocationValue(location);
        if (!next) {
          toast('Pick an address from the suggestions or share your location', 'error');
          return;
        }
        resolved = next;
        setLocation(next);
      }
      const res = await apiFetch('/v1/warehouse/ops/location', {
        method: 'PATCH',
        body: JSON.stringify({
          address: resolved.address.trim(),
          place_id: resolved.place_id,
          lat: Number.parseFloat(resolved.lat),
          lng: Number.parseFloat(resolved.lng),
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
        icon="settings"
        title="Warehouse settings"
        description="Checkout policy, pre-order lead window, per-line quantity limits, delivery surcharges, and depot location. Changes sync with dispatch and delivery routing."
      >
        <div className="max-w-2xl space-y-6">
          <PortalSection icon="warehouse" title="Depot location" description="Delivery fee distance is measured warehouse lat/lng → retailer delivery pin at checkout.">
            <LocationPicker value={location} onChange={setLocation} label="Warehouse address" />
            <button
              type="button"
              disabled={savingLocation}
              onClick={() => void saveLocation()}
              className="portal-btn portal-btn--primary disabled:opacity-50"
            >
              {savingLocation ? 'Saving…' : 'Save location'}
            </button>
          </PortalSection>

          <PortalSection icon="orders" title="Pre-order lead window" description="Days between order placement and earliest fulfillment.">
            <div className="grid grid-cols-2 gap-3">
              <PortalField id="preorderMinLeadDays" label="Min lead (days)">
                <PortalInput id="preorderMinLeadDays" value={preorderMinLeadDays} onChange={(e: ChangeEvent<HTMLInputElement>) => setPreorderMinLeadDays(e.target.value)} />
              </PortalField>
              <PortalField id="preorderMaxLeadDays" label="Max lead (days)">
                <PortalInput id="preorderMaxLeadDays" value={preorderMaxLeadDays} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setPreorderMaxLeadDays(e.target.value)} />
              </PortalField>
            </div>
          </PortalSection>

          <PortalSection icon="inventory" title="Line quantity limits" description="Leave blank to clear a limit. Applies to standard and scheduled checkout.">
            <div className="grid grid-cols-2 gap-3">
              <PortalField id="orderLineMin" label="Min per SKU">
                <PortalInput id="orderLineMin" value={orderLineMin} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setOrderLineMin(e.target.value)} />
              </PortalField>
              <PortalField id="orderLineMax" label="Max per SKU">
                <PortalInput id="orderLineMax" value={orderLineMax} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setOrderLineMax(e.target.value)} />
              </PortalField>
            </div>
          </PortalSection>

          <PortalSection icon="payment" title="Delivery fee tiers">
            <div className="grid grid-cols-2 gap-3">
              <PortalField id="feeBaseMinor" label="Base fee (minor)">
                <PortalInput id="feeBaseMinor" value={feeBaseMinor} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeBaseMinor(e.target.value)} />
              </PortalField>
              <PortalField id="feeCurrency" label="Currency">
                <PortalInput id="feeCurrency" value={feeCurrency} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeCurrency(e.target.value)} />
              </PortalField>
              <PortalField id="feeTierKm" label="Free within km">
                <PortalInput id="feeTierKm" value={feeTierKm} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeTierKm(e.target.value)} />
              </PortalField>
              <PortalField id="feeTierMinor" label="Beyond-tier fee (minor)">
                <PortalInput id="feeTierMinor" value={feeTierMinor} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setFeeTierMinor(e.target.value)} />
              </PortalField>
            </div>
          </PortalSection>

          <PortalSection icon="settings" title="Out-of-stock orders">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={showStockCounts} onChange={(e: ChangeEvent<HTMLInputElement>) => setShowStockCounts(e.target.checked)} />
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
          </PortalSection>

          <PortalSection icon="settings" title="Order acceptance hours" description="When enforcement is on, retailers cannot preview or create orders outside the window.">
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={enforceOrderAcceptance} onChange={(e: ChangeEvent<HTMLInputElement>) => setEnforceOrderAcceptance(e.target.checked)} />
              Enforce order acceptance hours
            </label>
            <label className="flex items-center gap-2 text-sm">
              <input type="checkbox" checked={scheduleIs24h} onChange={(e: ChangeEvent<HTMLInputElement>) => setScheduleIs24h(e.target.checked)} />
              Open 24 hours
            </label>
            <PortalField id="scheduleTimezone" label="Timezone">
              <PortalInput id="scheduleTimezone" value={scheduleTimezone} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setScheduleTimezone(e.target.value)} />
            </PortalField>
            <div className="grid grid-cols-2 gap-3">
              <PortalField id="weekdayOpen" label="Weekday open">
                <PortalInput id="weekdayOpen" value={weekdayOpen} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setWeekdayOpen(e.target.value)} />
              </PortalField>
              <PortalField id="weekdayClose" label="Weekday close">
                <PortalInput id="weekdayClose" value={weekdayClose} onChange={(e: ChangeEvent<HTMLInputElement | HTMLSelectElement>) => setWeekdayClose(e.target.value)} />
              </PortalField>
            </div>
            <h3 className="text-xs font-semibold text-[var(--muted)]">Advanced JSON</h3>
            <textarea
              className="w-full min-h-[140px] font-mono text-xs rounded-lg border p-3"
              style={{ borderColor: 'var(--field-border)', background: 'var(--field-background)' }}
              value={scheduleJSON}
              onChange={(e: ChangeEvent<HTMLTextAreaElement>) => setScheduleJSON(e.target.value)}
            />
          </PortalSection>

          <button
            type="button"
            disabled={saving}
            onClick={() => void save()}
            className="portal-btn portal-btn--primary disabled:opacity-50"
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

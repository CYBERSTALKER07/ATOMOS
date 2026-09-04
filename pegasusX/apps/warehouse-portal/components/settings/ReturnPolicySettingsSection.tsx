'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import { warehouseHomeNodeId } from '@/lib/warehouse-scope';
import { warehouseReturnPolicyPutKey } from '@pegasusx/api-core';
import type { WarehouseReturnPolicy } from '@pegasusx/types';

export function ReturnPolicySettingsSection() {
  const t = usePortalT();
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [saved, setSaved] = useState(false);
  const [reverseSla, setReverseSla] = useState('24');
  const [canOverride, setCanOverride] = useState(false);
  const [retailerWindow, setRetailerWindow] = useState('');
  const [supplierId, setSupplierId] = useState('');

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const wh = warehouseHomeNodeId();
      const q = new URLSearchParams();
      if (wh) q.set('warehouse_id', wh);
      const res = await apiFetch(`/v1/warehouse/return-policy?${q.toString()}`);
      if (!res.ok) {
        throw new Error(`load failed (${res.status})`);
      }
      const p = (await res.json()) as WarehouseReturnPolicy;
      setSupplierId(p.supplier_id || '');
      setReverseSla(
        p.reverse_dock_sla_hours != null ? String(p.reverse_dock_sla_hours) : '24',
      );
      setCanOverride(Boolean(p.can_override_retailer_window));
      setRetailerWindow(
        p.retailer_file_window_hours != null
          ? String(p.retailer_file_window_hours)
          : '',
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load return policy');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function save() {
    setSaving(true);
    setError(null);
    setSaved(false);
    try {
      const wh = warehouseHomeNodeId();
      const body: WarehouseReturnPolicy = {
        supplier_id: supplierId,
        can_override_retailer_window: canOverride,
      };
      const sla = Number(reverseSla);
      if (!Number.isNaN(sla) && sla > 0) {
        body.reverse_dock_sla_hours = sla;
      }
      if (canOverride) {
        const hours = Number(retailerWindow);
        if (Number.isNaN(hours) || hours < 1) {
          throw new Error('Retailer file window hours required when override is enabled');
        }
        body.retailer_file_window_hours = hours;
      }
      const q = new URLSearchParams();
      if (wh) q.set('warehouse_id', wh);
      const res = await apiFetch(`/v1/warehouse/return-policy?${q.toString()}`, {
        method: 'PUT',
        headers: {
          'Idempotency-Key': warehouseReturnPolicyPutKey(wh || 'default', supplierId),
        },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const text = await res.text();
        throw new Error(text || `save failed (${res.status})`);
      }
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Save failed');
    } finally {
      setSaving(false);
    }
  }

  return (
    <section className="mt-10 space-y-4 rounded-2xl border border-[var(--desk-border)] p-4">
      <div>
        <h2 className="text-lg font-medium">{t("warehouse_portal.settings.return_policy_settings_section.text.returns_and_reverse_sla")}</h2>
        <p className="text-sm text-[var(--muted)]">
          Reverse-dock SLA and optional retailer claim-window override. Override may only
          lengthen the supplier base window.
        </p>
      </div>
      {loading ? (
        <p className="text-sm text-[var(--muted)]">{t("warehouse_portal.bins.text.loading")}</p>
      ) : (
        <>
          {error && <p className="text-sm font-semibold text-red-600">{error}</p>}
          {saved && <p className="text-sm text-emerald-700">{t("warehouse_portal.settings.return_policy_settings_section.text.return_policy_saved")}</p>}
          <label className="block text-sm">
            Reverse dock SLA (hours)
            <input
              type="number"
              min={1}
              max={168}
              value={reverseSla}
              onChange={(e) => setReverseSla(e.target.value)}
              className="mt-1 w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2"
            />
          </label>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={canOverride}
              onChange={(e) => setCanOverride(e.target.checked)}
            />
            Override retailer claim filing window (lengthen only)
          </label>
          {canOverride && (
            <label className="block text-sm">
              Retailer file window (hours)
              <input
                type="number"
                min={1}
                max={168}
                value={retailerWindow}
                onChange={(e) => setRetailerWindow(e.target.value)}
                className="mt-1 w-full rounded-xl border border-[var(--desk-border)] bg-[var(--desk-surface)] px-3 py-2"
              />
            </label>
          )}
          <button
            type="button"
            disabled={saving}
            onClick={() => void save()}
            className="portal-btn portal-btn--primary disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Save return policy'}
          </button>
        </>
      )}
    </section>
  );
}

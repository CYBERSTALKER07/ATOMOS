'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { CycleCount, InventoryAdjustment } from '@pegasusx/types';

export default function CycleCountsPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [counts, setCounts] = useState<CycleCount[]>([]);
  const [adjustments, setAdjustments] = useState<InventoryAdjustment[]>([]);
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [locationId, setLocationId] = useState('');
  const [productId, setProductId] = useState('');
  const [expectedQty, setExpectedQty] = useState('');
  const [creating, setCreating] = useState(false);
  const [submitQty, setSubmitQty] = useState<Record<string, string>>({});
  const [busyId, setBusyId] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [cRes, aRes] = await Promise.all([
        apiFetch('/v1/warehouse/ops/cycle-counts'),
        apiFetch('/v1/warehouse/ops/inventory-adjustments'),
      ]);
      if (cRes.status === 409) {
        setEnabled(false);
        setCounts([]);
        setAdjustments([]);
        return;
      }
      if (cRes.ok) {
        const data = await cRes.json();
        setCounts(data.counts || []);
        setEnabled(data.cycle_counts_enabled !== false);
      } else {
        toast('Failed to load cycle counts', 'error');
      }
      if (aRes.ok) {
        const data = await aRes.json();
        setAdjustments(data.adjustments || []);
      }
    } catch {
      toast('Failed to load cycle counts', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const createCount = async () => {
    const loc = locationId.trim();
    const pid = productId.trim();
    if (!loc || !pid) {
      toast('Location and product required', 'error');
      return;
    }
    setCreating(true);
    try {
      const body: Record<string, unknown> = { location_id: loc, product_id: pid };
      const exp = expectedQty.trim();
      if (exp !== '') {
        const n = Number(exp);
        if (!Number.isFinite(n)) {
          toast('Expected qty invalid', 'error');
          return;
        }
        body.expected_qty = n;
      }
      const res = await apiFetch('/v1/warehouse/ops/cycle-counts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `cc-${Date.now()}` },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Create failed', 'error');
        return;
      }
      toast('Count created', 'success');
      setLocationId('');
      setProductId('');
      setExpectedQty('');
      await load();
    } finally {
      setCreating(false);
    }
  };

  const submitCount = async (count: CycleCount) => {
    const raw = submitQty[count.count_id] ?? String(count.expected_qty);
    const qty = Number(raw);
    if (!Number.isFinite(qty)) {
      toast('Counted qty invalid', 'error');
      return;
    }
    setBusyId(count.count_id);
    try {
      const res = await apiFetch(`/v1/warehouse/ops/cycle-counts/${encodeURIComponent(count.count_id)}/submit`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `cc-submit-${count.count_id}-${Date.now()}` },
        body: JSON.stringify({ counted_qty: qty }),
      });
      if (!res.ok) {
        toast('Submit failed', 'error');
        return;
      }
      toast('Count submitted', 'success');
      await load();
    } finally {
      setBusyId(null);
    }
  };

  return (
    <PageTransition>
      <PageChrome title={t("portal.nav.cycle_counts")} description={t("warehouse_portal.residual.text.8_7_wave_1c_stub_counts_variance_adjustments_approve_does_not_ch")}>
        {!enabled && !loading && (
          <p className="mb-4 text-sm opacity-70">
            Cycle counts are off (`WMS_CYCLE_COUNTS_ENABLED`). Enable the flag to create counts.
          </p>
        )}

        <div className="mb-6 flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-sm">
            Location ID
            <input className="border px-2 py-1 font-mono text-xs" value={locationId} onChange={(e) => setLocationId(e.target.value)} />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            Product ID
            <input className="border px-2 py-1 font-mono text-xs" value={productId} onChange={(e) => setProductId(e.target.value)} />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            Expected (optional)
            <input className="border px-2 py-1 w-24" value={expectedQty} onChange={(e) => setExpectedQty(e.target.value)} placeholder={t("warehouse_portal.cycle_counts.text.auto")} />
          </label>
          <button type="button" className="border px-3 py-1.5 text-sm" disabled={creating || !enabled} onClick={() => void createCount()}>
            {creating ? 'Creating…' : 'Create count'}
          </button>
        </div>

        {loading ? (
          <p className="text-sm opacity-70">{t("warehouse_portal.bins.text.loading")}</p>
        ) : counts.length === 0 ? (
          <EmptyState headline={t("warehouse_portal.residual.text.no_cycle_counts")} body={t("warehouse_portal.residual.text.create_an_open_count_for_a_bin_product")} />
        ) : (
          <div className="mb-10 overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-2 pr-3">{t("warehouse_portal.cycle_counts.text.count")}</th>
                  <th className="py-2 pr-3">{t("portal.nav.location")}</th>
                  <th className="py-2 pr-3">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.cycle_counts.text.expected")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.status")}</th>
                  <th className="py-2">{t("warehouse_portal.cycle_counts.text.submit")}</th>
                </tr>
              </thead>
              <tbody>
                {counts.map((c) => (
                  <tr key={c.count_id} className="border-b border-black/5">
                    <td className="py-2 pr-3 font-mono text-xs">{c.count_id.slice(0, 8)}…</td>
                    <td className="py-2 pr-3 font-mono text-xs">{c.location_id}</td>
                    <td className="py-2 pr-3 font-mono text-xs">{c.product_id}</td>
                    <td className="py-2 pr-3">
                      {c.counted_qty != null ? `${c.counted_qty}/${c.expected_qty}` : c.expected_qty}
                      {c.variance_qty != null ? ` (Δ${c.variance_qty})` : ''}
                    </td>
                    <td className="py-2 pr-3">{c.status}</td>
                    <td className="py-2">
                      {c.status === 'OPEN' ? (
                        <span className="inline-flex items-center gap-2">
                          <input
                            className="border px-1 py-0.5 w-16 text-xs"
                            value={submitQty[c.count_id] ?? String(c.expected_qty)}
                            onChange={(e) => setSubmitQty((s) => ({ ...s, [c.count_id]: e.target.value }))}
                          />
                          <button
                            type="button"
                            className="border px-2 py-0.5 text-xs"
                            disabled={busyId === c.count_id}
                            onClick={() => void submitCount(c)}
                          >
                            {t("warehouse_portal.cycle_counts.text.submit")}
                          </button>
                        </span>
                      ) : (
                        '—'
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        <section className="border-t pt-6">
          <h2 className="mb-3 text-base font-medium">{t("warehouse_portal.cycle_counts.text.inventory_adjustments")}</h2>
          <p className="mb-4 text-sm opacity-70">{t("warehouse_portal.cycle_counts.text.pending_rows_are_created_on_variance_approve_is_stub_only_no_lot")}</p>
          {adjustments.length === 0 ? (
            <p className="text-sm opacity-70">{t("warehouse_portal.cycle_counts.text.no_adjustments")}</p>
          ) : (
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-2 pr-3">{t("warehouse_portal.cycle_counts.text.adjustment")}</th>
                  <th className="py-2 pr-3">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.cycle_counts.text.delta")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.status")}</th>
                  <th className="py-2">{t("supplier_portal.admin.audit_log.table.action")}</th>
                </tr>
              </thead>
              <tbody>
                {adjustments.map((a) => (
                  <tr key={a.adjustment_id} className="border-b border-black/5">
                    <td className="py-2 pr-3 font-mono text-xs">{a.adjustment_id.slice(0, 8)}…</td>
                    <td className="py-2 pr-3 font-mono text-xs">{a.product_id}</td>
                    <td className="py-2 pr-3">{a.delta_qty}</td>
                    <td className="py-2 pr-3">{a.status}</td>
                    <td className="py-2">
                      {a.status === 'PENDING' ? (
                        <button
                          type="button"
                          className="border px-2 py-0.5 text-xs"
                          onClick={async () => {
                            const res = await apiFetch(
                              `/v1/warehouse/ops/inventory-adjustments/${encodeURIComponent(a.adjustment_id)}/approve`,
                              { method: 'POST', headers: { 'Idempotency-Key': `adj-ap-${a.adjustment_id}` } },
                            );
                            if (!res.ok) toast('Approve failed (admin?)', 'error');
                            else {
                              toast('Approved (lots updated)', 'success');
                              await load();
                            }
                          }}
                        >
                          Approve
                        </button>
                      ) : (
                        '—'
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>
      </PageChrome>
    </PageTransition>
  );
}

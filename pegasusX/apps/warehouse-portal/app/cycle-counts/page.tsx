'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { CycleCount, InventoryAdjustment } from '@pegasusx/types';

async function readApiError(res: Response, fallback: string): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    return body.error || fallback;
  } catch {
    return fallback;
  }
}

function formatSignedDelta(value: number): string {
  return value > 0 ? `+${value}` : String(value);
}

function adjustmentBadgeClass(status: string): string {
  switch (status) {
    case 'APPROVED':
      return 'border-emerald-600/40 bg-emerald-50 text-emerald-800';
    case 'REJECTED':
      return 'border-rose-600/40 bg-rose-50 text-rose-800';
    default:
      return 'border-amber-600/40 bg-amber-50 text-amber-800';
  }
}

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
  const [reasonCode, setReasonCode] = useState<Record<string, string>>({});
  const [busyId, setBusyId] = useState<string | null>(null);
  const [busyAdjId, setBusyAdjId] = useState<string | null>(null);

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
      } else if (aRes.status !== 409) {
        toast('Failed to load inventory adjustments', 'error');
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
        if (!Number.isInteger(n) || n < 0) {
          toast('Expected qty must be a non-negative integer', 'error');
          return;
        }
        body.expected_qty = n;
      }
      const res = await apiFetch('/v1/warehouse/ops/cycle-counts', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `cc-${crypto.randomUUID()}` },
        body: JSON.stringify(body),
      });
      if (!res.ok) {
        toast(await readApiError(res, 'Create failed'), 'error');
        return;
      }
      toast('Count created', 'success');
      setLocationId('');
      setProductId('');
      setExpectedQty('');
      await load();
    } catch {
      toast('Create failed', 'error');
    } finally {
      setCreating(false);
    }
  };

  const submitCount = async (count: CycleCount) => {
    const raw = submitQty[count.count_id] ?? String(count.expected_qty);
    const trimmed = raw.trim();
    if (trimmed === '') {
      toast('Counted qty required', 'error');
      return;
    }
    const qty = Number(trimmed);
    if (!Number.isInteger(qty) || qty < 0) {
      toast('Counted qty must be a non-negative integer', 'error');
      return;
    }
    const reason = (reasonCode[count.count_id] ?? '').trim();
    setBusyId(count.count_id);
    try {
      const res = await apiFetch(`/v1/warehouse/ops/cycle-counts/${encodeURIComponent(count.count_id)}/submit`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': `cc-submit-${count.count_id}-${crypto.randomUUID()}`,
        },
        body: JSON.stringify({
          counted_qty: qty,
          ...(reason !== '' ? { reason_code: reason } : {}),
        }),
      });
      if (!res.ok) {
        toast(await readApiError(res, 'Submit failed'), 'error');
        return;
      }
      toast('Count submitted', 'success');
      setSubmitQty((prev) => {
        const next = { ...prev };
        delete next[count.count_id];
        return next;
      });
      setReasonCode((prev) => {
        const next = { ...prev };
        delete next[count.count_id];
        return next;
      });
      await load();
    } catch {
      toast('Submit failed', 'error');
    } finally {
      setBusyId(null);
    }
  };

  const handleAdjustment = async (adjustmentId: string, action: 'approve' | 'reject') => {
    setBusyAdjId(adjustmentId);
    try {
      const res = await apiFetch(
        `/v1/warehouse/ops/inventory-adjustments/${encodeURIComponent(adjustmentId)}/${action}`,
        {
          method: 'POST',
          headers: {
            'Idempotency-Key': `adj-${action}-${adjustmentId}-${crypto.randomUUID()}`,
          },
        },
      );
      if (!res.ok) {
        const fallback = action === 'approve' ? 'Approve failed (admin?)' : 'Reject failed (admin?)';
        toast(await readApiError(res, fallback), 'error');
        return;
      }
      toast(action === 'approve' ? 'Adjustment approved' : 'Adjustment rejected', 'success');
      await load();
    } catch {
      toast(action === 'approve' ? 'Approve failed' : 'Reject failed', 'error');
    } finally {
      setBusyAdjId(null);
    }
  };

  return (
    <PageTransition>
      <PageChrome title={t("portal.nav.cycle_counts")} description={t("warehouse_portal.cycle_counts.text.cycle_counts_variance_adjustments_description")}>
        {!enabled && !loading && (
          <p className="mb-4 text-sm opacity-70">
            Cycle counts are off (`WMS_CYCLE_COUNTS_ENABLED`). Enable the flag to create counts.
          </p>
        )}

        <div className="mb-6 flex flex-wrap items-end gap-3">
          <label className="flex flex-col gap-1 text-sm">
            Location ID
            <input id="location-id-input-4" aria-label="Location ID" className="border px-2 py-1 font-mono text-xs" value={locationId} onChange={(e) => setLocationId(e.target.value)} />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            Product ID
            <input id="product-id-input-3" aria-label="Product ID" className="border px-2 py-1 font-mono text-xs" value={productId} onChange={(e) => setProductId(e.target.value)} />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            Expected (optional)
            <input id="expected-optional-input-2" aria-label="Expected (optional)" className="border px-2 py-1 w-24" value={expectedQty} onChange={(e) => setExpectedQty(e.target.value)} placeholder={t("warehouse_portal.cycle_counts.text.auto")} />
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
                {counts.map((c) => {
                  const variance =
                    c.variance_qty != null
                      ? c.variance_qty
                      : c.counted_qty != null
                        ? c.counted_qty - c.expected_qty
                        : null;
                  return (
                    <tr key={c.count_id} className="border-b border-black/5">
                      <td className="py-2 pr-3 font-mono text-xs">{c.count_id.slice(0, 8)}…</td>
                      <td className="py-2 pr-3 font-mono text-xs">{c.location_id}</td>
                      <td className="py-2 pr-3 font-mono text-xs">{c.product_id}</td>
                      <td className="py-2 pr-3">
                        <span>{c.counted_qty != null ? `${c.counted_qty}/${c.expected_qty}` : c.expected_qty}</span>
                        {variance != null && (
                          <span
                            className={`ml-2 inline-flex items-center rounded border px-1.5 py-0.5 font-mono text-xs ${
                              variance === 0
                                ? 'border-neutral-400/40 bg-neutral-100 text-neutral-700'
                                : variance < 0
                                  ? 'border-amber-600/40 bg-amber-50 text-amber-800'
                                  : 'border-sky-600/40 bg-sky-50 text-sky-800'
                            }`}
                          >
                            Δ{formatSignedDelta(variance)}
                          </span>
                        )}
                      </td>
                      <td className="py-2 pr-3">{c.status}</td>
                      <td className="py-2">
                        {c.status === 'OPEN' ? (
                          <span className="inline-flex flex-wrap items-center gap-2">
                            <input
                              id={`cycle-count-qty-${c.count_id}`}
                              aria-label={`Counted quantity for ${c.product_id}`}
                              type="number"
                              min={0}
                              step={1}
                              className="w-16 border px-1.5 py-0.5 text-xs"
                              value={submitQty[c.count_id] ?? String(c.expected_qty)}
                              onChange={(e) => setSubmitQty((s) => ({ ...s, [c.count_id]: e.target.value }))}
                            />
                            <input
                              id={`cycle-count-reason-${c.count_id}`}
                              aria-label={`Reason code for ${c.product_id}`}
                              className="w-28 border px-1.5 py-0.5 text-xs"
                              placeholder="Reason (opt.)"
                              value={reasonCode[c.count_id] ?? ''}
                              onChange={(e) => setReasonCode((s) => ({ ...s, [c.count_id]: e.target.value }))}
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
                  );
                })}
              </tbody>
            </table>
          </div>
        )}

        <section className="border-t pt-6">
          <h2 className="mb-3 text-base font-medium">{t("warehouse_portal.cycle_counts.text.inventory_adjustments")}</h2>
          <p className="mb-4 text-sm opacity-70">{t("warehouse_portal.cycle_counts.text.pending_rows_created_on_variance_approve_mutates_lots")}</p>
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
                    <td className="py-2 pr-3 font-mono text-xs">{formatSignedDelta(a.delta_qty)}</td>
                    <td className="py-2 pr-3">
                      <span
                        data-status={a.status}
                        className={`inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium ${adjustmentBadgeClass(a.status)}`}
                      >
                        {a.status}
                      </span>
                    </td>
                    <td className="py-2">
                      {a.status === 'PENDING' ? (
                        <span className="inline-flex items-center gap-2">
                          <button
                            type="button"
                            className="border px-2 py-0.5 text-xs"
                            disabled={busyAdjId === a.adjustment_id}
                            onClick={() => void handleAdjustment(a.adjustment_id, 'approve')}
                          >
                            Approve
                          </button>
                          <button
                            type="button"
                            className="border border-rose-600/40 px-2 py-0.5 text-xs text-rose-800"
                            disabled={busyAdjId === a.adjustment_id}
                            onClick={() => void handleAdjustment(a.adjustment_id, 'reject')}
                          >
                            Reject
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
          )}
        </section>
      </PageChrome>
    </PageTransition>
  );
}

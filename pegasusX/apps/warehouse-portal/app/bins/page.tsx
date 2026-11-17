'use client';

import { usePortalT } from '@/lib/i18n';
import { ColdLotWarningBanner } from '@/components/ColdLotWarningBanner';
import {
  analyzeWarehouseLots,
  isColdCompatibleZone,
  recommendedColdBins,
  type LotColdAssessment,
} from '@/lib/cold-lot-warnings';
import { useCallback, useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { WarehouseBinLocation, StockLot } from '@pegasusx/types';

export default function BinsPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [bins, setBins] = useState<WarehouseBinLocation[]>([]);
  const [lots, setLots] = useState<StockLot[]>([]);
  const [lotsEnabled, setLotsEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [zone, setZone] = useState('A');
  const [locType, setLocType] = useState('PICK');
  const [creating, setCreating] = useState(false);
  const [putawayLotId, setPutawayLotId] = useState('');
  const [putawayTargetLoc, setPutawayTargetLoc] = useState('');
  const [puttingAway, setPuttingAway] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const [binsRes, lotsRes] = await Promise.all([
        apiFetch('/v1/warehouse/ops/bins'),
        apiFetch('/v1/warehouse/ops/lots'),
      ]);
      if (binsRes.ok) {
        const data = await binsRes.json();
        setBins(data.bins || []);
        if (typeof data.lots_enabled === 'boolean') setLotsEnabled(data.lots_enabled);
      }
      if (lotsRes.ok) {
        const data = await lotsRes.json();
        setLots(data.lots || []);
        if (typeof data.lots_enabled === 'boolean') setLotsEnabled(data.lots_enabled);
      }
    } catch {
      toast('Failed to load bins/lots', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const createBin = async () => {
    setCreating(true);
    try {
      const res = await apiFetch('/v1/warehouse/ops/bins', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `bin-${Date.now()}` },
        body: JSON.stringify({ zone, location_type: locType, pick_sequence: bins.length + 1 }),
      });
      if (!res.ok) {
        toast('Create bin failed', 'error');
        return;
      }
      toast('Bin created', 'success');
      await load();
    } finally {
      setCreating(false);
    }
  };

  return (
    <PageTransition>
      <PageChrome title={t("portal.nav.bins_lots")} description={t("warehouse_portal.residual.text.8_7_wave_1a_warehouse_locations_and_stock_lots_fefo")}>
        {!lotsEnabled && (
          <p className="mb-4 text-sm opacity-70">
            Lot mode is off (`WMS_LOTS_ENABLED`). Bins can still be managed; putaway requires the flag.
          </p>
        )}
        <div className="mb-6 flex flex-wrap items-end gap-3">
          <label htmlFor="bin-zone-input" className="flex flex-col gap-1 text-sm">
            {t("warehouse_portal.bins.text.zone")}
            <input
              id="bin-zone-input"
              aria-label={t("warehouse_portal.bins.text.zone")}
              className="border px-2 py-1"
              value={zone}
              onChange={(e) => setZone(e.target.value)}
            />
          </label>
          <label className="flex flex-col gap-1 text-sm">
            {t("warehouse_portal.bins.text.type")}
            <select className="border px-2 py-1" value={locType} onChange={(e) => setLocType(e.target.value)}>
              <option value="PICK">PICK</option>
              <option value="BULK">BULK</option>
              <option value="STAGE">STAGE</option>
              <option value="QUARANTINE">QUARANTINE</option>
            </select>
          </label>
          <button
            type="button"
            className="border px-3 py-1.5 text-sm"
            disabled={creating}
            onClick={() => void createBin()}
          >
            {creating ? 'Creating…' : 'Add bin'}
          </button>
        </div>

        {loading ? (
          <p className="text-sm opacity-70">{t("warehouse_portal.bins.text.loading")}</p>
        ) : bins.length === 0 ? (
          <EmptyState headline={t("warehouse_portal.residual.text.no_bins")} body={t("warehouse_portal.residual.text.create_a_pick_or_stage_location_to_putaway_lots")} />
        ) : (
          <div className="mb-10 overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-2 pr-3">{t("portal.nav.location")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.zone")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.type")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.seq")}</th>
                  <th className="py-2">{t("warehouse_portal.bins.text.active")}</th>
                </tr>
              </thead>
              <tbody>
                {bins.map((b) => (
                  <tr key={b.location_id} className="border-b border-black/5">
                    <td className="py-2 pr-3 font-mono text-xs">{b.location_id}</td>
                    <td className="py-2 pr-3">{b.zone || '—'}</td>
                    <td className="py-2 pr-3">{b.location_type}</td>
                    <td className="py-2 pr-3">{b.pick_sequence}</td>
                    <td className="py-2">{b.is_active ? 'yes' : 'no'}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {(() => {
          const coldSummary = analyzeWarehouseLots(lots, bins);
          const putawayQueue = coldSummary.coldWarningLots.length > 0
            ? coldSummary.coldWarningLots
            : coldSummary.stagingLots;
          const activeItem =
            putawayQueue.find((a) => a.lot.lot_id === putawayLotId) ?? putawayQueue[0];
          const candidates = activeItem ? recommendedColdBins(bins, activeItem.requiredZone) : [];
          const chosenBin = activeItem
            ? bins.find((b) => b.location_id === (putawayTargetLoc || candidates[0]?.location_id))
            : undefined;
          const targetZoneMismatch =
            Boolean(activeItem?.requiredZone) &&
            chosenBin !== undefined &&
            activeItem?.requiredZone !== null &&
            !isColdCompatibleZone(chosenBin.zone, activeItem.requiredZone);
          return (
            <>
              <ColdLotWarningBanner
                t={t}
                coldWarningLots={coldSummary.coldWarningLots}
                stagingLots={coldSummary.stagingLots}
                expiryAlertLots={coldSummary.expiryAlertLots}
                onSelectPutawayLot={(assessment: LotColdAssessment) => {
                  setPutawayLotId(assessment.lot.lot_id);
                  const rec = recommendedColdBins(bins, assessment.requiredZone);
                  setPutawayTargetLoc(rec[0]?.location_id ?? '');
                }}
              />
              {activeItem && (
                <div className="mb-6 rounded-xl border border-black/10 bg-black/[0.02] p-4 dark:border-white/10">
              <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
                <h3 className="text-sm font-semibold">Migrated Staging &amp; Cold-Lot Putaway</h3>
                {activeItem.requiredZone && (
                  <span className="rounded bg-sky-100 px-2 py-0.5 font-mono text-xs text-sky-900">
                    Required zone: {activeItem.requiredZone}
                  </span>
                )}
              </div>
              <div className="flex flex-wrap items-end gap-3">
                <label className="text-xs">
                  <span className="mb-1 block opacity-70">Lot to put away</span>
                  <select
                    aria-label="Migrated lot to put away"
                    value={activeItem.lot.lot_id}
                    onChange={(e) => {
                      const next = putawayQueue.find((a) => a.lot.lot_id === e.target.value);
                      setPutawayLotId(e.target.value);
                      if (next) {
                        const rec = recommendedColdBins(bins, next.requiredZone);
                        setPutawayTargetLoc(rec[0]?.location_id ?? '');
                      }
                    }}
                    className="rounded border border-black/15 px-2.5 py-1.5 text-sm"
                  >
                    {putawayQueue.map((item) => (
                      <option key={item.lot.lot_id} value={item.lot.lot_id}>
                        {item.lot.product_name || item.lot.product_id} · {item.lot.lot_code || item.lot.lot_id.slice(0, 8)} @ {item.lot.location_id} ({item.lot.quantity_on_hand})
                      </option>
                    ))}
                  </select>
                </label>
                <label className="text-xs">
                  <span className="mb-1 block opacity-70">Target bin</span>
                  <select
                    aria-label="Target warehouse bin"
                    value={chosenBin?.location_id ?? ''}
                    onChange={(e) => setPutawayTargetLoc(e.target.value)}
                    className="rounded border border-black/15 px-2.5 py-1.5 text-sm"
                  >
                    <option value="">Select target bin…</option>
                    {bins.map((b) => (
                      <option key={b.location_id} value={b.location_id}>
                        {b.location_id} · {b.zone || 'AMBIENT'} ({b.location_type})
                      </option>
                    ))}
                  </select>
                </label>
                <button
                  type="button"
                  disabled={puttingAway || !chosenBin || activeItem.lot.quantity_on_hand <= 0}
                  onClick={async () => {
                    if (!chosenBin) return;
                    setPuttingAway(true);
                    try {
                      const res = await apiFetch('/v1/warehouse/ops/lots/putaway', {
                        method: 'POST',
                        headers: {
                          'Content-Type': 'application/json',
                          'Idempotency-Key': `warehouse-putaway-${activeItem.lot.lot_id}-${chosenBin.location_id}-${crypto.randomUUID()}`,
                        },
                        body: JSON.stringify({
                          product_id: activeItem.lot.product_id,
                          location_id: chosenBin.location_id,
                          quantity: activeItem.lot.quantity_on_hand,
                          lot_code: activeItem.lot.lot_code || undefined,
                          expiry_date: activeItem.lot.expiry_date || undefined,
                        }),
                      });
                      if (!res.ok) {
                        const errBody = (await res.json().catch(() => ({}))) as { error?: string };
                        toast(errBody.error || 'Putaway failed', 'error');
                        return;
                      }
                      toast(
                        activeItem.requiredZone
                          ? t('warehouse_portal.cold_lot.confirm_cold_putaway')
                          : 'Putaway completed',
                        'success',
                      );
                      setPutawayLotId('');
                      await load();
                    } finally {
                      setPuttingAway(false);
                    }
                  }}
                  className="rounded bg-black px-3 py-1.5 text-xs font-medium text-white disabled:opacity-50"
                >
                  {activeItem.requiredZone
                    ? t('warehouse_portal.cold_lot.confirm_cold_putaway')
                    : 'Confirm Putaway'}
                </button>
              </div>
              {targetZoneMismatch && activeItem.requiredZone && (
                <p className="mt-2 text-xs font-medium text-red-700">
                  {t('warehouse_portal.cold_lot.zone_mismatch', {
                    sku: activeItem.lot.product_name || activeItem.lot.product_id,
                    required_zone: activeItem.requiredZone,
                  })}
                </p>
              )}
                </div>
              )}
            </>
          );
        })()}
        <h2 className="mb-3 text-lg font-medium">{t("warehouse_portal.bins.text.lots")}</h2>
        {lots.length === 0 ? (
          <EmptyState headline={t("warehouse_portal.residual.text.no_lots")} body={t("warehouse_portal.residual.text.receive_putaway_with_location_expiry_perishables_when_wms_lots_a")} />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-2 pr-3">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.lot")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.expiry")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.on_hand")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.reserved")}</th>
                  <th className="py-2">{t("warehouse_portal.bins.text.status")}</th>
                </tr>
              </thead>
              <tbody>
                {lots.map((l) => (
                  <tr key={l.lot_id} className="border-b border-black/5">
                    <td className="py-2 pr-3 font-mono text-xs">{l.product_id}</td>
                    <td className="py-2 pr-3">{l.lot_code || l.lot_id.slice(0, 8)}</td>
                    <td className="py-2 pr-3">{l.expiry_date || '—'}</td>
                    <td className="py-2 pr-3">{l.quantity_on_hand}</td>
                    <td className="py-2 pr-3">{l.quantity_reserved}</td>
                    <td className="py-2">{l.status}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}

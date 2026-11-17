'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useState } from 'react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { apiFetch } from '@/lib/auth';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { useToast } from '@/components/Toast';
import type { PickWave, PickTask, StockLot, WarehouseBinLocation } from '@pegasusx/types';
import { ColdLotWarningBanner } from '@/components/ColdLotWarningBanner';
import { analyzePickWavePreSealColdState } from '@/lib/cold-lot-warnings';

async function readApiError(res: Response, fallback: string): Promise<string> {
  try {
    const body = (await res.json()) as { error?: string };
    return body.error || fallback;
  } catch {
    return fallback;
  }
}

function pickTaskBadgeClass(status: string): string {
  switch (status) {
    case 'CONFIRMED':
      return 'border-emerald-600/40 bg-emerald-50 text-emerald-800';
    case 'SHORT':
      return 'border-amber-600/40 bg-amber-50 text-amber-800';
    case 'SHORT_WAIVED':
      return 'border-sky-600/40 bg-sky-50 text-sky-800';
    default:
      return 'border-neutral-400/40 bg-neutral-100 text-neutral-700';
  }
}

export default function PickWavesPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const searchParams = useSearchParams();
  const [waves, setWaves] = useState<PickWave[]>([]);
  const [enabled, setEnabled] = useState(false);
  const [loading, setLoading] = useState(true);
  const [selected, setSelected] = useState<PickWave | null>(null);
  const [manifestId, setManifestId] = useState('');
  const [creating, setCreating] = useState(false);
  const [busyTask, setBusyTask] = useState<string | null>(null);
  const [waiving, setWaiving] = useState(false);
  const [pickQtyInput, setPickQtyInput] = useState<Record<string, string>>({});
  const [lots, setLots] = useState<StockLot[]>([]);
  const [bins, setBins] = useState<WarehouseBinLocation[]>([]);
  const [tempReadingsCount, setTempReadingsCount] = useState<number | null>(null);
  const [bayTempInput, setBayTempInput] = useState('4.0');
  const [loggingTemp, setLoggingTemp] = useState(false);

  useEffect(() => {
    const mid = searchParams.get('manifest_id');
    if (mid) setManifestId(mid);
  }, [searchParams]);

  const load = useCallback(async () => {
    setLoading(true);
    try {
      const res = await apiFetch('/v1/warehouse/ops/pick-waves');
      if (res.status === 409) {
        setEnabled(false);
        setWaves([]);
        return;
      }
      if (!res.ok) {
        toast('Failed to load pick waves', 'error');
        return;
      }
      const data = await res.json();
      setWaves(data.waves || []);
      setEnabled(data.pick_waves_enabled !== false);
      void Promise.all([
        apiFetch('/v1/warehouse/ops/lots').catch(() => null),
        apiFetch('/v1/warehouse/ops/bins').catch(() => null),
      ]).then(async ([lotsRes, binsRes]) => {
        if (lotsRes?.ok) {
          const lotsJson = (await lotsRes.json().catch(() => null)) as { lots?: StockLot[] } | null;
          if (lotsJson?.lots) setLots(lotsJson.lots);
        }
        if (binsRes?.ok) {
          const binsJson = (await binsRes.json().catch(() => null)) as { bins?: WarehouseBinLocation[] } | null;
          if (binsJson?.bins) setBins(binsJson.bins);
        }
      });
    } catch {
      toast('Failed to load pick waves', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

  const loadManifestTemp = useCallback(async (manifestIdValue: string) => {
    setTempReadingsCount(null);
    if (!manifestIdValue) return;
    try {
      const res = await apiFetch(`/v1/warehouse/ops/manifests/${encodeURIComponent(manifestIdValue)}/temperature`);
      if (!res.ok) return;
      const body = (await res.json().catch(() => null)) as { readings?: unknown[] } | null;
      if (body && Array.isArray(body.readings)) {
        setTempReadingsCount(body.readings.length);
      }
    } catch {
      // Non-fatal pre-seal advisory check
    }
  }, []);

  useEffect(() => {
    if (selected?.manifest_id) {
      void loadManifestTemp(selected.manifest_id);
    } else {
      setTempReadingsCount(null);
    }
  }, [selected?.manifest_id, loadManifestTemp]);

  const recordBayBaselineTemp = async () => {
    if (!selected?.manifest_id) return;
    const temp = Number(bayTempInput.trim());
    if (!Number.isFinite(temp)) {
      toast('Enter valid °C reading', 'error');
      return;
    }
    setLoggingTemp(true);
    try {
      const res = await apiFetch(
        `/v1/warehouse/ops/manifests/${encodeURIComponent(selected.manifest_id)}/temperature`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Idempotency-Key': `temp-baseline-${selected.manifest_id}-${crypto.randomUUID()}`,
          },
          body: JSON.stringify({ sensor_id: 'BAY-BASELINE', temp_c: temp }),
        },
      );
      if (!res.ok) {
        toast(await readApiError(res, 'Temperature log failed'), 'error');
        return;
      }
      toast('Cold-chain bay baseline logged', 'success');
      await loadManifestTemp(selected.manifest_id);
      await openWave(selected.wave_id);
    } catch {
      toast('Temperature log failed', 'error');
    } finally {
      setLoggingTemp(false);
    }
  };

  const openWave = async (waveId: string) => {
    try {
      const res = await apiFetch(`/v1/warehouse/ops/pick-waves/${encodeURIComponent(waveId)}`);
      if (!res.ok) {
        toast('Failed to load wave', 'error');
        return;
      }
      setSelected(await res.json());
    } catch {
      toast('Failed to load wave', 'error');
    }
  };

  const createWave = async () => {
    const mid = manifestId.trim();
    if (!mid) {
      toast('Manifest ID required', 'error');
      return;
    }
    setCreating(true);
    try {
      const res = await apiFetch('/v1/warehouse/ops/pick-waves', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `pick-wave-${crypto.randomUUID()}` },
        body: JSON.stringify({ manifest_id: mid }),
      });
      if (!res.ok) {
        toast(await readApiError(res, 'Create wave failed'), 'error');
        return;
      }
      const wave = (await res.json()) as PickWave;
      toast('Pick wave created', 'success');
      setManifestId('');
      await load();
      setSelected(wave);
    } catch {
      toast('Create wave failed', 'error');
    } finally {
      setCreating(false);
    }
  };

  const confirmTask = async (task: PickTask, explicitQty?: number) => {
    if (!selected) return;
    const raw =
      explicitQty !== undefined
        ? String(explicitQty)
        : (pickQtyInput[task.task_id] ??
            String(task.status === 'SHORT' ? task.quantity_picked : task.quantity_requested));
    const trimmed = raw.trim();
    if (trimmed === '') {
      toast('Picked qty required', 'error');
      return;
    }
    const qty = Number(trimmed);
    if (!Number.isInteger(qty) || qty < 0 || qty > task.quantity_requested) {
      toast(`Picked qty must be 0–${task.quantity_requested}`, 'error');
      return;
    }
    setBusyTask(task.task_id);
    try {
      const res = await apiFetch(
        `/v1/warehouse/ops/pick-waves/${encodeURIComponent(selected.wave_id)}/tasks/${encodeURIComponent(task.task_id)}/confirm`,
        {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            'Idempotency-Key': `pick-confirm-${task.task_id}-${crypto.randomUUID()}`,
          },
          body: JSON.stringify({ quantity_picked: qty }),
        },
      );
      if (!res.ok) {
        toast(await readApiError(res, 'Confirm failed'), 'error');
        return;
      }
      const wave = (await res.json()) as PickWave;
      setSelected(wave);
      setPickQtyInput((prev) => {
        const next = { ...prev };
        delete next[task.task_id];
        return next;
      });
      if (wave.status === 'READY_TO_SEAL') {
        toast('Wave ready to seal', 'success');
      } else if (qty < task.quantity_requested) {
        toast('Short pick recorded', 'success');
      } else {
        toast('Pick confirmed', 'success');
      }
      await load();
    } catch {
      toast('Confirm failed', 'error');
    } finally {
      setBusyTask(null);
    }
  };

  const waiveShorts = async () => {
    if (!selected) return;
    setWaiving(true);
    try {
      const res = await apiFetch(
        `/v1/warehouse/ops/pick-waves/${encodeURIComponent(selected.wave_id)}/waive-shorts`,
        {
          method: 'POST',
          headers: {
            'Idempotency-Key': `pick-waive-${selected.wave_id}-${crypto.randomUUID()}`,
          },
        },
      );
      if (!res.ok) {
        toast(await readApiError(res, 'Waive shorts failed'), 'error');
        return;
      }
      const wave = (await res.json()) as PickWave;
      setSelected(wave);
      toast(wave.status === 'READY_TO_SEAL' ? 'Wave ready to seal' : 'Shorts waived', 'success');
      await load();
    } catch {
      toast('Waive shorts failed', 'error');
    } finally {
      setWaiving(false);
    }
  };

  const hasShortTasks = Boolean(selected?.tasks?.some((task) => task.status === 'SHORT'));

  return (
    <PageTransition>
      <PageChrome title={t("portal.nav.pick_waves")} description={t("warehouse_portal.residual.text.8_7_wave_1b_manifest_picks_ready_to_seal_before_payload_seal")}>
        {!enabled && !loading && (
          <p className="mb-4 text-sm opacity-70">
            Pick waves are off (`WMS_PICK_WAVES_ENABLED`). Enable the flag to create waves and gate seal.
          </p>
        )}
        <p className="mb-4 text-sm opacity-70">
          Create from a DRAFT/LOADING{' '}
          <Link href="/manifests" className="underline">
            manifest
          </Link>
          , confirm tasks in pick sequence, then seal.
        </p>

        <div className="mb-6 flex flex-wrap items-end gap-3">
          <label htmlFor="pick-waves-manifest-id" className="flex flex-col gap-1 text-sm">
            Manifest ID
            <input
              id="pick-waves-manifest-id"
              aria-label="Manifest ID"
              className="border px-2 py-1 font-mono text-xs min-w-[16rem]"
              value={manifestId}
              onChange={(e) => setManifestId(e.target.value)}
              placeholder={t("warehouse_portal.pick_waves.text.manifest_uuid")}
            />
          </label>
          <button
            type="button"
            className="border px-3 py-1.5 text-sm"
            disabled={creating || !enabled}
            onClick={() => void createWave()}
          >
            {creating ? 'Creating…' : 'Create wave'}
          </button>
        </div>

        {loading ? (
          <p className="text-sm opacity-70">{t("warehouse_portal.bins.text.loading")}</p>
        ) : waves.length === 0 ? (
          <EmptyState headline={t("warehouse_portal.residual.text.no_pick_waves")} body={t("warehouse_portal.residual.text.create_a_wave_from_a_draft_or_loading_manifest")} />
        ) : (
          <div className="mb-8 overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead>
                <tr className="border-b">
                  <th className="py-2 pr-3">{t("warehouse_portal.pick_waves.text.wave")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.manifests.text.manifest")}</th>
                  <th className="py-2 pr-3">{t("warehouse_portal.bins.text.status")}</th>
                  <th className="py-2">{t("warehouse_portal.pick_waves.text.open")}</th>
                </tr>
              </thead>
              <tbody>
                {waves.map((w) => (
                  <tr key={w.wave_id} className="border-b border-black/5">
                    <td className="py-2 pr-3 font-mono text-xs">{w.wave_id.slice(0, 8)}…</td>
                    <td className="py-2 pr-3 font-mono text-xs">{w.manifest_id.slice(0, 8)}…</td>
                    <td className="py-2 pr-3">{w.status}</td>
                    <td className="py-2">
                      <button type="button" className="underline text-sm" onClick={() => void openWave(w.wave_id)}>
                        Tasks
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}

        {selected && (
          <section className="border-t pt-6">
            {(() => {
              const preSealCold = analyzePickWavePreSealColdState(selected, lots, bins, tempReadingsCount);
              if (!preSealCold.hasPreSealWarning) return null;
              return (
                <div className="mb-4">
                  <ColdLotWarningBanner
                    t={t}
                    coldWarningLots={preSealCold.zoneMismatchLots}
                    missingManifestBaseline={preSealCold.missingBaseline}
                  />
                  {preSealCold.missingBaseline && (
                    <div className="flex flex-wrap items-center gap-2 rounded-lg border border-amber-500/40 bg-amber-50/70 px-3 py-2 text-xs">
                      <label className="flex items-center gap-1.5">
                        <span>Bay Temp (°C)</span>
                        <input
                          type="number"
                          step="0.5"
                          aria-label="Bay baseline temperature Celsius"
                          value={bayTempInput}
                          onChange={(e) => setBayTempInput(e.target.value)}
                          className="w-20 rounded border border-black/15 px-2 py-1 text-xs"
                        />
                      </label>
                      <button
                        type="button"
                        disabled={loggingTemp}
                        onClick={recordBayBaselineTemp}
                        className="rounded bg-black px-2.5 py-1 text-xs font-medium text-white disabled:opacity-50"
                      >
                        Log Bay Baseline
                      </button>
                      <Link href="/bins" className="ml-auto underline">
                        Inspect Lots &amp; Bins →
                      </Link>
                    </div>
                  )}
                </div>
              );
            })()}
            <div className="mb-2 flex flex-wrap items-center justify-between gap-3">
              <div>
                <h2 className="text-base font-medium">
                  Wave {selected.wave_id.slice(0, 8)}… — {selected.status}
                </h2>
                <p className="font-mono text-xs opacity-70">Manifest {selected.manifest_id}</p>
              </div>
              {hasShortTasks && (
                <button
                  type="button"
                  className="border border-amber-600/50 bg-amber-50 px-3 py-1.5 text-xs font-medium text-amber-900"
                  disabled={waiving || busyTask !== null}
                  onClick={() => void waiveShorts()}
                >
                  {waiving ? 'Waiving…' : 'Waive Shorts'}
                </button>
              )}
            </div>
            {(selected.tasks || []).length === 0 ? (
              <p className="text-sm opacity-70">{t("warehouse_portal.pick_waves.text.no_tasks")}</p>
            ) : (
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b">
                    <th className="py-2 pr-3">{t("warehouse_portal.bins.text.seq")}</th>
                    <th className="py-2 pr-3">{t("supplier_portal.admin.empathy.hierarchy.product.level")}</th>
                    <th className="py-2 pr-3">{t("warehouse_portal.bins.text.lot")}</th>
                    <th className="py-2 pr-3">{t("warehouse_portal.pick_waves.text.bin")}</th>
                    <th className="py-2 pr-3">{t("warehouse_portal.pick_waves.text.qty")}</th>
                    <th className="py-2 pr-3">{t("warehouse_portal.bins.text.status")}</th>
                    <th className="py-2">{t("supplier_portal.admin.audit_log.table.action")}</th>
                  </tr>
                </thead>
                <tbody>
                  {(selected.tasks || []).map((task) => {
                    const defaultQty =
                      task.status === 'SHORT' ? task.quantity_picked : task.quantity_requested;
                    const inputValue = pickQtyInput[task.task_id] ?? String(defaultQty);
                    return (
                      <tr key={task.task_id} className="border-b border-black/5">
                        <td className="py-2 pr-3">{task.pick_sequence}</td>
                        <td className="py-2 pr-3 font-mono text-xs">{task.product_id}</td>
                        <td className="py-2 pr-3 font-mono text-xs">{task.lot_id.slice(0, 8)}…</td>
                        <td className="py-2 pr-3 font-mono text-xs">{task.location_id || '—'}</td>
                        <td className="py-2 pr-3">
                          {task.quantity_picked}/{task.quantity_requested}
                        </td>
                        <td className="py-2 pr-3">
                          <span
                            data-status={task.status}
                            className={`inline-flex items-center rounded border px-2 py-0.5 text-xs font-medium ${pickTaskBadgeClass(task.status)}`}
                          >
                            {task.status}
                          </span>
                        </td>
                        <td className="py-2">
                          {task.status === 'PENDING' || task.status === 'SHORT' ? (
                            <span className="inline-flex flex-wrap items-center gap-2">
                              <input
                                id={`pick-qty-${task.task_id}`}
                                aria-label={`Picked quantity for ${task.product_id}`}
                                type="number"
                                min={0}
                                max={task.quantity_requested}
                                step={1}
                                className="w-16 border px-1.5 py-0.5 text-xs"
                                value={inputValue}
                                disabled={busyTask === task.task_id || waiving}
                                onChange={(e) =>
                                  setPickQtyInput((prev) => ({
                                    ...prev,
                                    [task.task_id]: e.target.value,
                                  }))
                                }
                              />
                              <button
                                type="button"
                                className="border px-2 py-1 text-xs"
                                disabled={busyTask === task.task_id || waiving}
                                onClick={() => void confirmTask(task)}
                              >
                                Confirm
                              </button>
                              <button
                                type="button"
                                className="border border-amber-600/40 px-2 py-1 text-xs text-amber-800"
                                disabled={busyTask === task.task_id || waiving}
                                onClick={() => void confirmTask(task, 0)}
                              >
                                Short (0)
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
            )}
          </section>
        )}
      </PageChrome>
    </PageTransition>
  );
}

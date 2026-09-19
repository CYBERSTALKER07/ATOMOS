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
import type { PickWave, PickTask } from '@pegasusx/types';

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
    } catch {
      toast('Failed to load pick waves', 'error');
    } finally {
      setLoading(false);
    }
  }, [toast]);

  useEffect(() => {
    void load();
  }, [load]);

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
        headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `pick-wave-${mid}-${Date.now()}` },
        body: JSON.stringify({ manifest_id: mid }),
      });
      if (!res.ok) {
        const err = await res.json().catch(() => ({}));
        toast(err.error || 'Create wave failed', 'error');
        return;
      }
      const wave = (await res.json()) as PickWave;
      toast('Pick wave created', 'success');
      setManifestId('');
      await load();
      setSelected(wave);
    } finally {
      setCreating(false);
    }
  };

  const confirmTask = async (task: PickTask, qty?: number) => {
    if (!selected) return;
    setBusyTask(task.task_id);
    try {
      const res = await apiFetch(
        `/v1/warehouse/ops/pick-waves/${encodeURIComponent(selected.wave_id)}/tasks/${encodeURIComponent(task.task_id)}/confirm`,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json', 'Idempotency-Key': `pick-confirm-${task.task_id}-${Date.now()}` },
          body: JSON.stringify({ quantity_picked: qty ?? task.quantity_requested }),
        },
      );
      if (!res.ok) {
        toast('Confirm failed', 'error');
        return;
      }
      const wave = (await res.json()) as PickWave;
      setSelected(wave);
      toast(wave.status === 'READY_TO_SEAL' ? 'Wave ready to seal' : 'Pick confirmed', 'success');
      await load();
    } finally {
      setBusyTask(null);
    }
  };

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
          <label className="flex flex-col gap-1 text-sm">
            Manifest ID
            <input
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
            <h2 className="mb-2 text-base font-medium">
              Wave {selected.wave_id.slice(0, 8)}… — {selected.status}
            </h2>
            <p className="mb-4 font-mono text-xs opacity-70">Manifest {selected.manifest_id}</p>
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
                  {(selected.tasks || []).map((t) => (
                    <tr key={t.task_id} className="border-b border-black/5">
                      <td className="py-2 pr-3">{t.pick_sequence}</td>
                      <td className="py-2 pr-3 font-mono text-xs">{t.product_id}</td>
                      <td className="py-2 pr-3 font-mono text-xs">{t.lot_id.slice(0, 8)}…</td>
                      <td className="py-2 pr-3 font-mono text-xs">{t.location_id || '—'}</td>
                      <td className="py-2 pr-3">
                        {t.quantity_picked}/{t.quantity_requested}
                      </td>
                      <td className="py-2 pr-3">{t.status}</td>
                      <td className="py-2">
                        {t.status === 'PENDING' || t.status === 'SHORT' ? (
                          <button
                            type="button"
                            className="border px-2 py-1 text-xs"
                            disabled={busyTask === t.task_id}
                            onClick={() => void confirmTask(t)}
                          >
                            Confirm
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
        )}
      </PageChrome>
    </PageTransition>
  );
}

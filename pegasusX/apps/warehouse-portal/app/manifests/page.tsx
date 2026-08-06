'use client';

import { usePortalT } from "@/lib/i18n";
import { useEffect, useState, useCallback } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';

interface Manifest {
  manifest_id: string;
  driver_name: string;
  vehicle_label: string;
  license_plate: string;
  stop_count: number;
  status: string;
  created_at: string;
}

export default function ManifestsPage() {
  const t = usePortalT();
  const [manifests, setManifests] = useState<Manifest[]>([]);
  const [loading, setLoading] = useState(true);
  const [date, setDate] = useState(() => new Date().toISOString().slice(0, 10));

  const load = useCallback(async () => {
    try {
      const res = await apiFetch(`/v1/warehouse/ops/manifests?date=${date}`);
      if (res.ok) {
        const data = await res.json();
        setManifests(data.manifests || []);
      }
    } catch { /* handled */ }
    finally { setLoading(false); }
  }, [date]);

  useEffect(() => { load(); }, [load]);

  return (
    <PageTransition>
      <PageChrome
        icon="manifests"
        title={t("portal.nav.manifests")}
        description={t("warehouse_portal.residual.text.daily_loading_manifests_for_drivers_and_vehicles_at_this_node")}
        loading={loading}
        empty={!loading && manifests.length === 0}
        emptyMessage={`No manifests for ${date}.`}
        actions={
          <div className="flex gap-2 items-center">
            <input
              type="date"
              value={date}
              onChange={(e) => {
                setDate(e.target.value);
                setLoading(true);
              }}
              className="px-3 py-1.5 rounded-lg border text-sm"
              style={{
                background: 'var(--field-background)',
                borderColor: 'var(--field-border)',
                color: 'var(--field-foreground)',
              }}
            />
            <button
              type="button"
              onClick={() => {
                setLoading(true);
                load();
              }}
              className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-sm button--secondary"
            >
              <Icon name="refresh" size={16} />
            </button>
          </div>
        }
      >
        <div className="overflow-x-auto">
          <table className="desk-table w-full text-sm">
            <thead>
              <tr className="border-b border-[var(--border)]">
                <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.manifest")}</th>
                <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.driver")}</th>
                <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.vehicle")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.stops")}</th>
                <th className="text-left py-2 px-3 font-medium">{t("warehouse_portal.bins.text.status")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.created")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.pick")}</th>
                <th className="text-right py-2 px-3 font-medium">{t("warehouse_portal.manifests.text.labels")}</th>
              </tr>
            </thead>
            <tbody>
              {manifests.map(m => (
                <tr key={m.manifest_id} className="border-b border-[var(--border)] hover:bg-[var(--surface)] transition-colors">
                  <td className="py-2.5 px-3 font-mono text-xs">{m.manifest_id.slice(0, 8)}...</td>
                  <td className="py-2.5 px-3">{m.driver_name || '—'}</td>
                  <td className="py-2.5 px-3">{m.vehicle_label || m.license_plate || '—'}</td>
                  <td className="py-2.5 px-3 text-right font-mono">{m.stop_count}</td>
                  <td className="py-2.5 px-3">
                    <span className={`status-chip ${m.status === 'DISPATCHED' ? 'status-chip--active' : m.status === 'COMPLETED' ? 'status-chip--stable' : 'status-chip--draft'}`}>
                      {m.status}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 text-right text-[var(--muted)]">
                    {new Date(m.created_at).toLocaleTimeString()}
                  </td>
                  <td className="py-2.5 px-3 text-right">
                    <a
                      href={`/pick-waves?manifest_id=${encodeURIComponent(m.manifest_id)}`}
                      className="md-btn md-btn-text text-xs"
                    >
                      Pick wave
                    </a>
                  </td>
                  <td className="py-2.5 px-3 text-right">
                    <button
                      type="button"
                      className="md-btn md-btn-text text-xs"
                      onClick={async () => {
                        try {
                          const res = await apiFetch(`/v1/warehouse/manifests/${encodeURIComponent(m.manifest_id)}/labels`, {
                            method: 'POST',
                            headers: { 'Content-Type': 'application/json' },
                            body: '{}',
                          });
                          if (!res.ok) return;
                          const zpl = await res.text();
                          const blob = new Blob([zpl], { type: 'text/plain' });
                          const url = URL.createObjectURL(blob);
                          const a = document.createElement('a');
                          a.href = url;
                          a.download = `${m.manifest_id.slice(0, 8)}-labels.zpl`;
                          a.click();
                          URL.revokeObjectURL(url);
                        } catch { /* ignore */ }
                      }}
                    >
                      Print labels
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </PageChrome>
    </PageTransition>
  );
}

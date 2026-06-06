'use client';

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
        title="Manifests"
        description="Daily loading manifests for drivers and vehicles at this node."
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
                <th className="text-left py-2 px-3 font-medium">Manifest</th>
                <th className="text-left py-2 px-3 font-medium">Driver</th>
                <th className="text-left py-2 px-3 font-medium">Vehicle</th>
                <th className="text-right py-2 px-3 font-medium">Stops</th>
                <th className="text-left py-2 px-3 font-medium">Status</th>
                <th className="text-right py-2 px-3 font-medium">Created</th>
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
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </PageChrome>
    </PageTransition>
  );
}

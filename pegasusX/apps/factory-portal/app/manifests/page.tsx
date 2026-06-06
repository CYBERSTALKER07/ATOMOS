'use client';

import Link from 'next/link';
import { useCallback, useEffect, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import FactoryPageState from '@/components/FactoryPageState';
import { nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';

interface ManifestRow {
  manifest_id: string;
  state: string;
  transfer_count?: number;
  total_volume_vu?: number;
  max_volume_vu?: number;
  driver_id?: string;
  vehicle_id?: string;
  updated_at?: string;
}

function stateClass(state: string): string {
  const map: Record<string, string> = {
    DRAFT: 'status-chip--draft',
    LOADING: 'status-chip--loading',
    SEALED: 'status-chip--approved',
    DISPATCHED: 'status-chip--dispatched',
    COMPLETED: 'status-chip--received',
    CANCELLED: 'status-chip--cancelled',
  };
  return map[state] || '';
}

export default function ManifestsPage() {
  const [manifests, setManifests] = useState<ManifestRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch('/v1/factory/manifests');
      if (!res.ok) throw new Error(`Unable to load manifests (${res.status})`);
      const data = await res.json();
      setManifests(data.manifests || []);
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Unable to load manifests');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: (payload) => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) return;
        if (event.type === 'FACTORY_MANIFEST_UPDATE' || event.type === 'FACTORY_TRANSFER_UPDATE') {
          void load();
        }
      },
    });
    return unsubscribe;
  }, [load]);

  return (
    <PageTransition className="space-y-6 p-6 md:p-8">
      <div className="flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight text-[var(--foreground)]">Manifests</h1>
          <p className="mt-2 max-w-3xl text-sm leading-6 text-[var(--muted)]">
            LEO loading gate — advance manifests through draft, loading, sealed, dispatched, and completed states.
          </p>
        </div>
        <button
          type="button"
          onClick={() => void load()}
          className="button--secondary inline-flex h-10 items-center gap-2 rounded-full px-4 text-sm font-medium"
        >
          <Icon name="refresh" size={16} /> Refresh
        </button>
      </div>

      {loading ? (
        <FactoryPageState kind="loading" title="Manifests" subtitle="Loading manifest pipeline." />
      ) : error ? (
        <FactoryPageState kind="error" headline="Unable to load manifests" body={error} actionLabel="Retry" onAction={() => void load()} />
      ) : manifests.length === 0 ? (
        <FactoryPageState kind="empty" headline="No manifests" body="Dispatch transfers or create a manifest to begin the loading gate workflow." />
      ) : (
        <div className="md-card md-elevation-1 md-shape-md overflow-hidden">
          <table className="w-full">
            <thead>
              <tr style={{ background: 'var(--color-md-surface-container)' }}>
                {['Manifest', 'State', 'Transfers', 'Volume (VU)', 'Next step', ''].map((h) => (
                  <th key={h} className="md-typescale-label-small px-4 py-3 text-left" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                    {h}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {manifests.map((manifest) => {
                const next = nextManifestLifecycleAction(manifest.state);
                return (
                  <tr key={manifest.manifest_id} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                    <td className="md-typescale-body-small px-4 py-3 font-mono">{manifest.manifest_id}</td>
                    <td className="px-4 py-3">
                      <span className={`status-chip ${stateClass(manifest.state)}`}>{manifest.state}</span>
                    </td>
                    <td className="md-typescale-body-small px-4 py-3">{manifest.transfer_count ?? '—'}</td>
                    <td className="md-typescale-body-small px-4 py-3">{manifest.total_volume_vu ?? '—'}</td>
                    <td className="md-typescale-body-small px-4 py-3">{next?.label ?? '—'}</td>
                    <td className="px-4 py-3 text-right">
                      <Link href={`/manifests/${manifest.manifest_id}`} className="md-btn md-btn-tonal md-typescale-label-large px-4 py-2">
                        Open
                      </Link>
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </PageTransition>
  );
}

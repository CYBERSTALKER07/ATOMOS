'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useToast } from '@/components/Toast';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import EmptyState from '@/components/EmptyState';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { manifestTransitionIdempotencyKey, nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';

interface ManifestRow {
  manifest_id: string;
  state: string;
  transfer_count?: number;
  total_volume_vu?: number;
}

export default function FactoryPayloadLoadPage() {
  const { toast } = useToast();
  const [manifests, setManifests] = useState<ManifestRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [acting, setActing] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await apiFetch('/v1/factory/manifests');
      if (!res.ok) throw new Error(`Unable to load factory manifests (${res.status})`);
      const data = await res.json();
      const rows: ManifestRow[] = data.manifests || [];
      setManifests(rows.filter((m) => m.state === 'DRAFT' || m.state === 'LOADING'));
    } catch (e: unknown) {
      setError(e instanceof Error ? e.message : 'Unable to load payload');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { void load(); }, [load]);
  useFactorySessionReconcile(() => { void load(); });
  useEffect(() => {
    return subscribeFactoryWS({
      onMessage: (payload) => {
        const event = parseFactoryLiveEvent(payload);
        if (event?.type.startsWith('TRANSFER_') || event?.type.startsWith('MANIFEST_') || event?.type.startsWith('WAREHOUSE_TRANSFER_')) {
          void load();
        }
      },
    });
  }, [load]);

  const draftCount = useMemo(() => manifests.filter((m) => m.state === 'DRAFT').length, [manifests]);
  const loadingCount = useMemo(() => manifests.filter((m) => m.state === 'LOADING').length, [manifests]);

  async function runAction(manifest: ManifestRow) {
    const next = nextManifestLifecycleAction(manifest.state);
    if (!next || (next.path !== 'start-loading' && next.path !== 'seal')) return;
    setActing(manifest.manifest_id);
    try {
      const res = await apiFetch(`/v1/factory/manifests/${manifest.manifest_id}/${next.path}`, {
        method: 'POST',
        headers: {
          'Idempotency-Key': manifestTransitionIdempotencyKey(manifest.manifest_id, next.path, factoryOperatorId()),
        },
        body: JSON.stringify({ reason: 'factory-payload-load' }),
      });
      const body = await res.json().catch(() => ({}));
      if (!res.ok) throw new Error(body.error || `${next.label} failed`);
      toast(next.label, 'success');
      await load();
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Action failed', 'error');
    } finally {
      setActing(null);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="loadingBay"
        title="Payload / Load"
        description="Factory-plane start-loading and seal only. Last-mile payloader lists are not merged here."
        loading={loading}
        error={error}
        skeletonVariant="table"
      >
        <KpiStatGrid columns={2}>
          <KpiStatCard label="Draft" value={draftCount} sub="Ready to start loading" />
          <KpiStatCard label="Loading" value={loadingCount} sub="Ready to seal" />
        </KpiStatGrid>
        {manifests.length === 0 ? (
          <EmptyState
            variant="no-data"
            headline="No factory payloads to load"
            body="Dispatch creates FactoryTruckManifests drafts. Start loading and seal them here."
          />
        ) : (
          <div className="mt-6 overflow-x-auto">
            <table className="desk-table w-full text-sm">
              <thead>
                <tr className="border-b" style={{ borderColor: 'var(--desk-border)' }}>
                  {['Manifest', 'State', 'Transfers', 'VU', ''].map((h) => (
                    <th key={h} className="px-4 py-3 text-left font-medium">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {manifests.map((m) => {
                  const next = nextManifestLifecycleAction(m.state);
                  const allowed = next && (next.path === 'start-loading' || next.path === 'seal');
                  return (
                    <tr key={m.manifest_id} className="border-t" style={{ borderColor: 'var(--desk-border)' }}>
                      <td className="px-4 py-3 font-mono text-xs">{m.manifest_id}</td>
                      <td className="px-4 py-3">{m.state}</td>
                      <td className="px-4 py-3">{m.transfer_count ?? '—'}</td>
                      <td className="px-4 py-3">{m.total_volume_vu ?? '—'}</td>
                      <td className="px-4 py-3 text-right">
                        {allowed ? (
                          <button
                            type="button"
                            className="portal-btn portal-btn--primary text-sm"
                            disabled={acting === m.manifest_id}
                            onClick={() => void runAction(m)}
                          >
                            {acting === m.manifest_id ? 'Applying…' : next.label}
                          </button>
                        ) : '—'}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </PageChrome>
    </PageTransition>
  );
}

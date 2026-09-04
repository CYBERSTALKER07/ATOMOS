'use client';

import { usePortalT } from "@/lib/i18n";
import Link from 'next/link';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import EmptyState from '@/components/EmptyState';
import { nextManifestLifecycleAction } from '@/lib/manifest-lifecycle';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';

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
  const t = usePortalT();
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

  useFactorySessionReconcile(() => {
    void load();
  });

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: (payload) => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) return;
        if (event.type.startsWith('TRANSFER_') || event.type.startsWith('MANIFEST_') || event.type.startsWith('WAREHOUSE_TRANSFER_')) {
          void load();
        }
      },
    });
    return unsubscribe;
  }, [load]);

  const activeCount = useMemo(
    () => manifests.filter((m) => ['DRAFT', 'LOADING', 'SEALED'].includes(m.state)).length,
    [manifests],
  );
  const dispatchedCount = useMemo(
    () => manifests.filter((m) => m.state === 'DISPATCHED').length,
    [manifests],
  );
  const completedCount = useMemo(
    () => manifests.filter((m) => m.state === 'COMPLETED').length,
    [manifests],
  );

  return (
    <PageTransition>
      <PageChrome
        icon="manifests"
        title={t("portal.nav.manifests")}
        description={t("factory_portal.residual.text.leo_loading_gate_advance_manifests_through_draft_loading_sealed_")}
        loading={loading}
        skeletonVariant="table"
        error={error}
        actions={
          <button
            type="button"
            onClick={() => void load()}
            className="portal-btn portal-btn--ghost inline-flex h-10 items-center gap-2"
          >
            <Icon name="refresh" size={16} /> Refresh
          </button>
        }
      >
        <KpiStatGrid columns={4}>
          <KpiStatCard label={t("factory_portal.residual.text.total_manifests")} value={manifests.length} sub="Visible in pipeline" />
          <KpiStatCard label={t("factory_portal.residual.text.active_gate")} value={activeCount} sub="Draft, loading, or sealed" />
          <KpiStatCard label={t("supplier_portal.dispatch.text.dispatched")} value={dispatchedCount} sub="Outbound this cycle" />
          <KpiStatCard label={t("portal.page.orders.filter.completed")} value={completedCount} sub="Fully closed manifests" />
        </KpiStatGrid>

        {manifests.length === 0 ? (
          <EmptyState
            variant="no-data"
            headline={t("factory_portal.residual.text.no_manifests")}
            body={t("factory_portal.residual.text.dispatch_transfers_or_create_a_manifest_to_begin_the_loading_gat")}
          />
        ) : (
          <PageSection title={t("factory_portal.manifests.text.manifest_pipeline")} description={t("factory_portal.residual.text.open_a_row_to_advance_lifecycle_actions")} className="mt-6">
            <div className="overflow-x-auto -mx-5 px-5">
              <table className="desk-table w-full text-sm">
                <thead>
                  <tr className="border-b" style={{ borderColor: 'var(--desk-border)' }}>
                    {['Manifest', 'State', 'Transfers', 'Volume (VU)', 'Next step', ''].map((h) => (
                      <th key={h} className="px-4 py-3 text-left font-medium" style={{ color: 'var(--desk-text-secondary)' }}>
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {manifests.map((manifest) => {
                    const next = nextManifestLifecycleAction(manifest.state);
                    return (
                      <tr key={manifest.manifest_id} className="border-t" style={{ borderColor: 'var(--desk-border)' }}>
                        <td className="px-4 py-3 font-mono text-sm">{manifest.manifest_id}</td>
                        <td className="px-4 py-3">
                          <span className={`status-chip ${stateClass(manifest.state)}`}>{manifest.state}</span>
                        </td>
                        <td className="px-4 py-3">{manifest.transfer_count ?? '—'}</td>
                        <td className="px-4 py-3">{manifest.total_volume_vu ?? '—'}</td>
                        <td className="px-4 py-3">{next?.label ?? '—'}</td>
                        <td className="px-4 py-3 text-right">
                          <Link href={`/manifests/${manifest.manifest_id}`} className="portal-btn portal-btn--ghost inline-flex text-sm">
                            Open
                          </Link>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </PageSection>
        )}
      </PageChrome>
    </PageTransition>
  );
}

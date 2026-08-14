'use client';

import { usePortalT } from "@/lib/i18n";
import { useCallback, useEffect, useMemo, useState } from 'react';
import { ExplainStatusBanner, explainFromApiError } from '@pegasusx/explain-ui';
import type { StatusExplain } from '@pegasusx/types';
import { factoryBatchDispatchKey } from '@pegasusx/api-client';
import { factoryOperatorId } from '@/lib/factory-scope';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import EmptyState from '@/components/EmptyState';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { PageSection } from '@/components/PageSection';
import HandoffTimelinePanel from '@/components/HandoffTimelinePanel';
import LoadingBayControls from '@/components/loading-bay/LoadingBayControls';
import LoadingBayGrid from '@/components/loading-bay/LoadingBayGrid';
type TransferState = 'APPROVED' | 'LOADING' | 'DISPATCHED';

interface Transfer {
  id: string;
  warehouse_name: string;
  total_items: number;
  total_volume_m3: number;
  state: string;
  created_at: string;
  updated_at: string;
}

const COLUMNS: { key: TransferState; label: string; css: string }[] = [
  { key: 'APPROVED', label: 'Ready for Loading', css: 'status-chip--approved' },
  { key: 'LOADING', label: 'Now Loading', css: 'status-chip--loading' },
  { key: 'DISPATCHED', label: 'Dispatched', css: 'status-chip--dispatched' },
];

export default function LoadingBayPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [dispatching, setDispatching] = useState(false);
  const [dispatchExplain, setDispatchExplain] = useState<StatusExplain | null>(null);
  const [forceCapacity, setForceCapacity] = useState(false);
  const [acceptPartial, setAcceptPartial] = useState(false);
  const [lastDispatch, setLastDispatch] = useState<{
    created_manifest_count?: number;
    optimizer_class?: string;
    dispatch_algo?: string;
    unassigned?: string[];
    status?: string;
  } | null>(null);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/transfers?states=APPROVED,LOADING,DISPATCHED');
      if (res.ok) {
        const data = await res.json();
        setTransfers(data.transfers || []);
      }
    } catch {
      // handled by empty state
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { load(); }, [load]);

  useFactorySessionReconcile(() => {
    void load();
  });

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) {
          return;
        }
        if (event.type !== 'FACTORY_TRANSFER_UPDATE' && event.type !== 'FACTORY_MANIFEST_UPDATE') {
          return;
        }
        void load();
      },
    });

    return () => {
      unsubscribe();
    };
  }, [load]);

  const grouped = useMemo(
    () =>
      COLUMNS.map((column) => ({
        ...column,
        items: transfers.filter((transfer) => transfer.state === column.key),
      })),
    [transfers],
  );
  const totalVolume = useMemo(
    () => transfers.reduce((sum, transfer) => sum + transfer.total_volume_m3, 0),
    [transfers],
  );
  const latestUpdatedAt = transfers.reduce<string | null>((latest, transfer) => {
    if (!latest) return transfer.updated_at;
    return new Date(transfer.updated_at).getTime() > new Date(latest).getTime() ? transfer.updated_at : latest;
  }, null);
  const readyCount = grouped.find((column) => column.key === 'APPROVED')?.items.length ?? 0;
  const loadingCount = grouped.find((column) => column.key === 'LOADING')?.items.length ?? 0;
  const dispatchedCount = grouped.find((column) => column.key === 'DISPATCHED')?.items.length ?? 0;

  async function handleDispatch() {
    setDispatching(true);
    setDispatchExplain(null);
    try {
      const transferIds = grouped.find((column) => column.key === 'APPROVED')?.items.map((t) => t.id) ?? [];
      const res = await apiFetch('/v1/factory/dispatch', {
        method: 'POST',
        headers: {
          'Idempotency-Key': factoryBatchDispatchKey(factoryOperatorId(), transferIds),
        },
        body: JSON.stringify({
          mode: 'AUTO',
          transfer_ids: transferIds,
          force_capacity: forceCapacity,
          accept_partial: acceptPartial,
          reason: 'factory-portal-loading-bay',
        }),
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok) {
        setLastDispatch(data);
        const count = data.created_manifest_count ?? data.manifests_created ?? 0;
        if (count === 0) {
          toast('No transfers to dispatch', 'info');
        } else {
          toast(`Dispatched ${count} manifest(s) · ${data.dispatch_algo || data.optimizer_class || 'solver'}`, 'success');
        }
        load();
      } else {
        setDispatchExplain(explainFromApiError(data));
        toast(data.error || 'Dispatch failed', 'error');
      }
    } catch {
      toast('Dispatch request failed', 'error');
    } finally {
      setDispatching(false);
    }
  }

  return (
    <PageTransition>
      <PageChrome
        icon="loadingBay"
        title={t("factory_portal.loading_bay.text.loading_bay")}
        description={t("factory_portal.residual.text.review_approved_factory_transfers_advance_active_loading_and_dis")}
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <LoadingBayControls
            onRefresh={() => void load()}
            onDispatch={handleDispatch}
            dispatching={dispatching}
          />
        }
      >
        {dispatchExplain ? (
          <ExplainStatusBanner explain={dispatchExplain} className="mb-4" />
        ) : null}
        <div className="mb-4 flex flex-wrap items-center gap-4 text-sm">
          <label className="inline-flex items-center gap-2">
            <input type="checkbox" checked={forceCapacity} onChange={(e) => setForceCapacity(e.target.checked)} />
            Force capacity
          </label>
          <label className="inline-flex items-center gap-2">
            <input type="checkbox" checked={acceptPartial} onChange={(e) => setAcceptPartial(e.target.checked)} />
            Accept partial (orphans)
          </label>
        </div>
        {lastDispatch ? (
          <p className="mb-4 text-sm text-[var(--desk-text-secondary)]">
            Last AUTO: {lastDispatch.status || 'ok'} · {lastDispatch.dispatch_algo || '—'} · class {lastDispatch.optimizer_class || '—'}
            {lastDispatch.unassigned && lastDispatch.unassigned.length > 0
              ? ` · unassigned ${lastDispatch.unassigned.length}`
              : ''}
          </p>
        ) : null}
        <KpiStatGrid columns={4}>
          <KpiStatCard label={t("factory_portal.residual.text.ready_to_load")} value={readyCount} sub="Awaiting operator attention" />
          <KpiStatCard label={t("factory_portal.residual.text.now_loading")} value={loadingCount} sub="Active bay work" />
          <KpiStatCard label={t("supplier_portal.dispatch.text.dispatched")} value={dispatchedCount} sub="Already moved out" />
          <KpiStatCard
            label={t("factory_portal.residual.text.total_volume")}
            value={`${totalVolume.toFixed(1)} m³`}
            sub={latestUpdatedAt ? `Updated ${new Date(latestUpdatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : 'No updates yet'}
          />
        </KpiStatGrid>

        <PageSection
          title={t("factory_portal.loading_bay.text.handoff_timeline")}
          description={t("factory_portal.residual.text.preorder_accept_dispatch_seal_events_from_the_factory_pulse_feed")}
          className="mt-6"
        >
          <HandoffTimelinePanel />
        </PageSection>

        {transfers.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-production-line.png"
            headline={t("factory_portal.residual.text.no_active_transfers_in_the_loading_bay")}
            body="Approved factory→warehouse transfers appear here. AUTO dispatch with an empty queue is a no-op (no invented manifests)."
          />
        ) : (
          <LoadingBayGrid grouped={grouped} />
        )}
      </PageChrome>
    </PageTransition>
  );
}

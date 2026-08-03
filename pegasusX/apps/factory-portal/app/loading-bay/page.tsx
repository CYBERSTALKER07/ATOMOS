'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import { ExplainStatusBanner, explainFromApiError } from '@pegasusx/explain-ui';
import type { StatusExplain } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
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
  const { toast } = useToast();
  const [transfers, setTransfers] = useState<Transfer[]>([]);
  const [loading, setLoading] = useState(true);
  const [dispatching, setDispatching] = useState(false);
  const [dispatchExplain, setDispatchExplain] = useState<StatusExplain | null>(null);

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
      const res = await apiFetch('/v1/factory/dispatch', { method: 'POST' });
      if (res.ok) {
        const data = await res.json();
        toast(`Dispatched ${data.manifests_created || 0} manifest(s)`, 'success');
        load();
      } else {
        const err = await res.json().catch(() => ({}));
        setDispatchExplain(explainFromApiError(err));
        toast(err.error || 'Dispatch failed', 'error');
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
        title="Loading bay"
        description="Review approved factory transfers, advance active loading, and dispatch manifests without losing warehouse context."
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
        <KpiStatGrid columns={4}>
          <KpiStatCard label="Ready to load" value={readyCount} sub="Awaiting operator attention" />
          <KpiStatCard label="Now loading" value={loadingCount} sub="Active bay work" />
          <KpiStatCard label="Dispatched" value={dispatchedCount} sub="Already moved out" />
          <KpiStatCard
            label="Total volume"
            value={`${totalVolume.toFixed(1)} m³`}
            sub={latestUpdatedAt ? `Updated ${new Date(latestUpdatedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}` : 'No updates yet'}
          />
        </KpiStatGrid>

        <PageSection
          title="Handoff timeline"
          description="Preorder → accept → dispatch → seal events from the factory pulse feed."
          className="mt-6"
        >
          <HandoffTimelinePanel />
        </PageSection>

        {transfers.length === 0 ? (
          <EmptyState
            imageUrl="/images/empty-production-line.png"
            headline="No active transfers in the loading bay"
            body="Approved transfers will appear here as soon as warehouse demand is accepted."
          />
        ) : (
          <LoadingBayGrid grouped={grouped} />
        )}
      </PageChrome>
    </PageTransition>
  );
}

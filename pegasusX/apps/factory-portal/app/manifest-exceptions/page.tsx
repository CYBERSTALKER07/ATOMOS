'use client';
import { usePolling } from '@pegasusx/api-react';


import { usePortalT } from "@/lib/i18n";
import { useState, useEffect, useCallback, useRef, useMemo } from 'react';
import { ExplainStatusBanner, explainFromApiError } from '@pegasusx/explain-ui';
import type { StatusExplain } from '@pegasusx/types';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import { useToast } from '@/components/Toast';
import Icon from '@/components/Icon';
import PageTransition from '@/components/PageTransition';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import FactoryRuntimeBanner from '@/components/FactoryRuntimeBanner';
import EmptyState from '@/components/EmptyState';

import { ManifestException, ManifestExceptionsList } from '../../components/manifest-exceptions/ManifestExceptionsList';

const LIVE_REFRESH_MS = 30_000;



function exceptionSignature(items: ManifestException[]) {
  return items
    .map((item) => `${item.exception_id}:${item.attempt_count}:${item.escalated}`)
    .join('|');
}

function formatSyncTime(value: number | null) {
  if (!value) return 'Waiting for first sync';
  return new Date(value).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
}


export default function ManifestExceptionsPage() {
  const t = usePortalT();
  const { toast } = useToast();
  const [exceptions, setExceptions] = useState<ManifestException[]>([]);
  const [loading, setLoading] = useState(true);
  const [escalatedOnly, setEscalatedOnly] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [fetchExplain, setFetchExplain] = useState<StatusExplain | null>(null);
  const [refreshing, setRefreshing] = useState(false);
  const [lastSyncedAt, setLastSyncedAt] = useState<number | null>(null);
  const [isOffline, setIsOffline] = useState(() => (typeof navigator === 'undefined' ? false : !navigator.onLine));
  const [resolvingId, setResolvingId] = useState<string | null>(null);
  const previousSignatureRef = useRef('');

  const fetchExceptions = useCallback(async (options?: { background?: boolean; silent?: boolean }) => {
    const background = options?.background ?? false;

    if (background) {
      setRefreshing(true);
    } else if (exceptions.length === 0) {
      setLoading(true);
    }

    try {
      const qs = escalatedOnly ? '?escalated=true' : '';
      const res = await apiFetch(`/v1/factory/manifest-exceptions${qs}`);
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setFetchExplain(explainFromApiError(body));
        throw new Error(body.error || `Factory API responded with ${res.status}`);
      }
      const data = await res.json();
      const next: ManifestException[] = data.exceptions || [];
      previousSignatureRef.current = exceptionSignature(next);
      setExceptions(next);
      setError(null);
      setLastSyncedAt(Date.now());
    } catch (e: unknown) {
      if (e instanceof DOMException && e.name === 'AbortError') return;
      setError(e instanceof Error ? e.message : 'Failed to load exceptions');
    } finally {
      if (background) {
        setRefreshing(false);
      } else {
        setLoading(false);
      }
    }
  }, [escalatedOnly, exceptions.length]);

  useEffect(() => {
    void fetchExceptions();
  }, [fetchExceptions]);

  useFactorySessionReconcile(() => {
    void fetchExceptions({ background: true, silent: true });
  });

  useEffect(() => {
    const onOnline = () => setIsOffline(false);
    const onOffline = () => setIsOffline(true);
    window.addEventListener('online', onOnline);
    window.addEventListener('offline', onOffline);
    return () => {
      window.removeEventListener('online', onOnline);
      window.removeEventListener('offline', onOffline);
    };
  }, []);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await fetchExceptions({ background: true, silent: true });
    },
    LIVE_REFRESH_MS,
    [fetchExceptions],
  );

  useEffect(() => {
    const unsubscribe = subscribeFactoryWS({
      onMessage: (raw) => {
        const event = parseFactoryLiveEvent(raw);
        if (!event?.type) return;
        if (event.type.startsWith('TRANSFER_') || event.type.startsWith('MANIFEST_') || event.type.startsWith('WAREHOUSE_TRANSFER_')) {
          void fetchExceptions({ background: true, silent: true });
        }
      },
    });
    return unsubscribe;
  }, [fetchExceptions]);

  const dlqCount = useMemo(
    () => exceptions.filter((ex) => ex.attempt_count >= 3).length,
    [exceptions],
  );
  const escalatedCount = useMemo(
    () => exceptions.filter((ex) => ex.escalated).length,
    [exceptions],
  );

  const runtimeMessage = isOffline
    ? `Offline — showing last sync from ${formatSyncTime(lastSyncedAt)}`
    : error && exceptions.length > 0
      ? `${error} Last sync ${formatSyncTime(lastSyncedAt)}`
      : refreshing
        ? `Refreshing exception inbox — last sync ${formatSyncTime(lastSyncedAt)}`
        : `Live sync active — last sync ${formatSyncTime(lastSyncedAt)}`;

  const runtimeTone = isOffline
    ? 'offline'
    : error && exceptions.length > 0
      ? 'warning'
      : refreshing
        ? 'refreshing'
        : 'live';

  const showFatalError = Boolean(error) && exceptions.length === 0 && !loading;

  const handleResolve = useCallback(async (exceptionId: string) => {
    setResolvingId(exceptionId);
    try {
      const res = await apiFetch(`/v1/factory/manifest-exceptions/${encodeURIComponent(exceptionId)}/resolve`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ resolution: 'RESOLVED', note: '' }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        throw new Error(body.error || `Resolve failed (${res.status})`);
      }
      toast('Exception resolved', 'success');
      await fetchExceptions({ background: true, silent: true });
    } catch (e: unknown) {
      toast(e instanceof Error ? e.message : 'Failed to resolve exception', 'error');
    } finally {
      setResolvingId(null);
    }
  }, [fetchExceptions, toast]);

  return (
    <PageTransition>
      <PageChrome
        icon="insights"
        title={t("factory_portal.manifest_exceptions.text.gate_exceptions")}
        description={t("factory_portal.residual.text.transfers_removed_from_manifests_during_loading_overflow_damage_")}
        loading={loading}
        skeletonVariant="table"
        error={showFatalError ? error : null}
        actions={
          <button
            type="button"
            className={`portal-btn ${escalatedOnly ? 'portal-btn--primary' : 'portal-btn--ghost'} inline-flex items-center gap-2`}
            onClick={() => setEscalatedOnly((value) => !value)}
          >
            <Icon name="error" size={18} />
            {escalatedOnly ? 'Showing escalated' : 'Escalated only'}
          </button>
        }
      >
        <FactoryRuntimeBanner tone={runtimeTone} message={runtimeMessage} />
        {fetchExplain ? <ExplainStatusBanner explain={fetchExplain} className="mt-4" /> : null}

        <div className="mt-4">
        <KpiStatGrid columns={4}>
          <KpiStatCard label={t("supplier_portal.live_ops_map.text.open_exceptions")} value={exceptions.length} sub={escalatedOnly ? 'Escalated filter on' : 'All reasons'} />
          <KpiStatCard label={t("factory_portal.residual.text.dlq_threshold")} value={dlqCount} sub="3+ overflow attempts" />
          <KpiStatCard label={t("factory_portal.residual.text.escalated")} value={escalatedCount} sub="Requires operator review" />
          <KpiStatCard label={t("factory_portal.residual.text.last_sync")} value={formatSyncTime(lastSyncedAt)} sub={isOffline ? 'Offline' : 'Live inbox'} />
        </KpiStatGrid>
        </div>

        {exceptions.length === 0 ? (
          <EmptyState
            variant="no-data"
            headline={escalatedOnly ? 'No escalated exceptions' : 'No exceptions'}
            body={
              escalatedOnly
                ? 'No transfers have hit the DLQ threshold (3+ overflows).'
                : 'All manifest loading operations completed without exceptions.'
            }
            action={escalatedOnly ? 'Show all' : undefined}
            onAction={escalatedOnly ? () => setEscalatedOnly(false) : undefined}
          />
        ) : (
          <ManifestExceptionsList
            exceptions={exceptions}
            resolvingId={resolvingId}
            onResolve={(id) => void handleResolve(id)}
          />
        )}
      </PageChrome>
    </PageTransition>
  );
}

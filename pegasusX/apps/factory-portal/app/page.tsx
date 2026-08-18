'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { etagFromResponse, usePolling } from '@pegasusx/api-client';
import {
  FACTORY_DRIVER_DUTY,
  FACTORY_QC_RESULTS,
  FACTORY_SLA_STATUSES,
  FACTORY_TRANSFER_STATES,
  FACTORY_VEHICLE_STATES,
  MANIFEST_STATES,
  canonicalizeFactoryTransfer,
  canonicalizeFactoryVehicle,
  emptyFactoryDriverDuty,
  emptyFactoryTransferCounts,
  emptyFactoryVehicleCounts,
  emptyManifestStateCounts,
  type FactoryDashboardException,
} from '@pegasusx/types';
import { shouldRefetchDashboardRollup } from '@pegasusx/ws-refresh-contract';
import { HealthStrip, SourceChip, StatusStack } from '@pegasusx/ui-kit/portal';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import EmptyState from '@/components/EmptyState';
import { PageChrome } from '@/components/PageChrome';
import PageTransition from '@/components/PageTransition';
import NetworkPulsePanel from '@/components/NetworkPulsePanel';

import { FactoryStats } from '@/components/dashboard/types';
import { DashboardActionGrid } from '@/components/dashboard/DashboardActionGrid';
import { DashboardMetrics } from '@/components/dashboard/DashboardMetrics';
import { DashboardAlerts } from '@/components/dashboard/DashboardAlerts';
import { usePortalT } from '@/lib/i18n';

function foldCounts<T extends string>(
  raw: unknown,
  empty: () => Record<T, number>,
  canonicalize: (key: string) => string,
): Record<T, number> {
  const next = empty();
  if (!raw || typeof raw !== 'object') return next;
  for (const [key, value] of Object.entries(raw as Record<string, number>)) {
    const normalized = canonicalize(key);
    if (normalized in next && Number.isFinite(Number(value))) {
      next[normalized as T] = Number(value);
    }
  }
  return next;
}

function foldExceptions(raw: unknown): FactoryDashboardException[] {
  if (!Array.isArray(raw)) return [];
  return raw.filter((row): row is FactoryDashboardException => !!row && typeof row === 'object');
}
const LIVE_REFRESH_MS = 60_000;
type DashboardLoadIssue = 'offline' | 'restricted' | 'error';

export default function FactoryDashboard() {
  const t = usePortalT();
  const router = useRouter();
  const [stats, setStats] = useState<FactoryStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadIssue, setLoadIssue] = useState<DashboardLoadIssue | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const etagRef = useRef<string | null>(null);

  const load = useCallback(async () => {
    try {
      const headers: Record<string, string> = {};
      if (etagRef.current) {
        headers['If-None-Match'] = etagRef.current;
      }
      const res = await apiFetch('/v1/factory/dashboard', { headers });
      if (res.status === 304) {
        setLoadIssue(null);
        return;
      }
      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          setLoadIssue('restricted');
        } else {
          setLoadIssue('error');
        }
        return;
      }

      etagRef.current = etagFromResponse(res);
      const row = (await res.json()) as Record<string, unknown>;
      setStats({
        source: typeof row.source === 'string' && row.source ? row.source : 'empty',
        plane: typeof row.plane === 'string' && row.plane ? row.plane : 'factory_trucks',
        pending_transfers: Number(row.pending_transfers ?? 0),
        loading_transfers: Number(row.loading_transfers ?? 0),
        active_manifests: Number(row.active_manifests ?? 0),
        dispatched_today: Number(row.dispatched_today ?? 0),
        vehicles_total: Number(row.vehicles_total ?? 0),
        vehicles_available: Number(row.vehicles_available ?? 0),
        staff_on_shift: Number(row.staff_on_shift ?? 0),
        critical_insights: Number(row.critical_insights ?? 0),
        transfers_by_state: foldCounts(row.transfers_by_state, emptyFactoryTransferCounts, canonicalizeFactoryTransfer),
        manifests_by_state: foldCounts(row.manifests_by_state, emptyManifestStateCounts, (key) => key.trim().toUpperCase()),
        vehicles_by_state: foldCounts(row.vehicles_by_state, emptyFactoryVehicleCounts, canonicalizeFactoryVehicle),
        driver_duty: foldCounts(row.driver_duty, emptyFactoryDriverDuty, (key) => key.trim().toUpperCase()),
        sla_by_status: foldCounts(
          row.sla_by_status,
          () => Object.fromEntries(FACTORY_SLA_STATUSES.map((key) => [key, 0])) as Record<(typeof FACTORY_SLA_STATUSES)[number], number>,
          (key) => key.trim().toUpperCase(),
        ),
        qc_by_result: foldCounts(
          row.qc_by_result,
          () => Object.fromEntries(FACTORY_QC_RESULTS.map((key) => [key, 0])) as Record<(typeof FACTORY_QC_RESULTS)[number], number>,
          (key) => key.trim().toUpperCase(),
        ),
        qc_available: row.qc_available === true,
        bay_loading_transfers: Number(row.bay_loading_transfers ?? 0),
        bay_loading_manifests: Number(row.bay_loading_manifests ?? 0),
        exceptions: foldExceptions(row.exceptions),
      });
      setLoadIssue(null);
    } catch {
      setLoadIssue('offline');
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    setLoading(true);
    void load();
  }, [load, reloadToken]);

  useFactorySessionReconcile(() => {
    setLoading(true);
    setReloadToken((v) => v + 1);
  });

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await load();
    },
    LIVE_REFRESH_MS,
    [load, reloadToken],
    { pauseWhenHidden: true, immediate: false },
  );

  useEffect(() => {
    let liveRefreshTimer: number | null = null;

    const queueLiveRefresh = () => {
      if (liveRefreshTimer !== null) {
        window.clearTimeout(liveRefreshTimer);
      }
      liveRefreshTimer = window.setTimeout(() => {
        void load();
      }, 600);
    };

    const unsubscribe = subscribeFactoryWS({
      onMessage: payload => {
        const event = parseFactoryLiveEvent(payload);
        if (!event) return;
        if (event.type === 'DRIVER_LOCATION_UPDATED' || !shouldRefetchDashboardRollup(event.type)) {
          return;
        }
        queueLiveRefresh();
      },
    });

    return () => {
      if (liveRefreshTimer !== null) {
        window.clearTimeout(liveRefreshTimer);
      }
      unsubscribe();
    };
  }, [load, reloadToken]);

  if (loading && !stats) {
    return (
      <PageTransition>
        <PageChrome
          icon="dashboard"
          title={t('portal.page.dashboard.factory.title')}
          description={t('portal.page.dashboard.factory.description')}
          loading
          skeletonVariant="dashboard"
        />
      </PageTransition>
    );
  }

  if (!stats && loadIssue) {
    const stateContent: Record<DashboardLoadIssue, { headline: string; body: string }> = {
      offline: {
        headline: 'You are offline',
        body: 'Factory dashboard data could not be refreshed because the network is unavailable.',
      },
      restricted: {
        headline: 'Access restricted',
        body: 'Your current session does not have permission to view factory dashboard metrics.',
      },
      error: {
        headline: 'Unable to load dashboard',
        body: 'A server issue blocked this view. Retry to fetch the latest factory status.',
      },
    };

    const content = stateContent[loadIssue];

    return (
      <PageTransition>
        <PageChrome icon="dashboard" title={t('portal.page.dashboard.factory.title')} description={t('portal.page.dashboard.factory.description')}>
          <EmptyState
            variant={loadIssue}
            headline={content.headline}
            body={content.body}
            action="Retry"
            onAction={() => {
              setLoading(true);
              setLoadIssue(null);
              setReloadToken(v => v + 1);
            }}
          />
        </PageChrome>
      </PageTransition>
    );
  }

  if (!stats) {
    return (
      <PageTransition>
        <PageChrome icon="dashboard" title={t('portal.page.dashboard.factory.title')} description={t('portal.page.dashboard.factory.description')}>
          <EmptyState
            variant="no-data"
            headline={t("factory_portal.residual.text.no_factory_metrics_yet")}
            body={t("factory_portal.residual.text.once_transfer_and_loading_activity_starts_operational_metrics_wi")}
            action="Refresh"
            onAction={() => {
              setLoading(true);
              setReloadToken(v => v + 1);
            }}
          />
        </PageChrome>
      </PageTransition>
    );
  }

  return (
    <PageTransition>
      <PageChrome
        icon="dashboard"
        title={t('portal.page.dashboard.factory.title')}
        description={t('portal.page.dashboard.factory.description')}
      >
        <div className="space-y-8">
          <div className="flex flex-wrap items-center gap-3" data-testid="gs-u-factory-source">
            <SourceChip source={stats.source} />
            <p className="text-sm text-[var(--muted)]">
              {stats.source === 'empty' ? 'No factory rows yet' : `Dashboard ${stats.source}`}
            </p>
          </div>

          <DashboardActionGrid stats={stats} />

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 space-y-4">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Factory plane</p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Transfers</h2>
            </div>
            <StatusStack
              dictionary={FACTORY_TRANSFER_STATES}
              counts={stats.transfers_by_state}
              source={stats.source}
              onSelect={(key) => router.push(`/transfers?state=${key}`)}
            />
          </section>

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 space-y-4" data-testid="gs-u-factory-trucks">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Dual plane</p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Factory trucks</h2>
              <p className="mt-1 text-sm text-[var(--muted)]">
                FactoryTruckManifests only. Last-mile retailer IN_TRANSIT is not a factory truck.
              </p>
            </div>
            <StatusStack
              dictionary={MANIFEST_STATES}
              counts={stats.manifests_by_state}
              source={stats.source}
              onSelect={() => router.push('/loading-bay')}
            />
          </section>

          <section className="grid gap-4 xl:grid-cols-2">
            <div className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 space-y-4">
              <h2 className="text-xl font-semibold tracking-tight text-[var(--foreground)]">Factory fleet</h2>
              <StatusStack dictionary={FACTORY_VEHICLE_STATES} counts={stats.vehicles_by_state} source={stats.source} />
              <StatusStack dictionary={FACTORY_DRIVER_DUTY} counts={stats.driver_duty} source={stats.source} />
              <p className="text-sm text-[var(--muted)]">
                Bay loading: {stats.bay_loading_transfers} transfers · {stats.bay_loading_manifests} manifests
              </p>
            </div>
            <div className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 space-y-4">
              <h2 className="text-xl font-semibold tracking-tight text-[var(--foreground)]">SLA and QC</h2>
              <HealthStrip
                items={FACTORY_SLA_STATUSES.map((key) => ({
                  key,
                  label: key.replaceAll('_', ' '),
                  count: stats.sla_by_status[key] ?? 0,
                  href: '/supply-requests',
                }))}
                onSelect={() => router.push('/supply-requests')}
              />
              {stats.qc_available ? (
                <HealthStrip
                  items={FACTORY_QC_RESULTS.map((key) => ({
                    key,
                    label: key,
                    count: stats.qc_by_result[key] ?? 0,
                    href: '/supply-requests',
                  }))}
                  onSelect={() => router.push('/supply-requests')}
                />
              ) : (
                <div className="flex items-center gap-2" data-testid="gs-u-qc-unavailable">
                  <SourceChip source="unavailable" />
                  <p className="text-sm text-[var(--muted)]">QC counts unavailable</p>
                </div>
              )}
            </div>
          </section>

          {stats.exceptions.length > 0 ? (
            <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 space-y-3">
              <h2 className="text-xl font-semibold tracking-tight text-[var(--foreground)]">Exceptions</h2>
              {stats.exceptions.map((row) => (
                <p key={row.exception_id ?? `${row.manifest_id}-${row.reason}`} className="text-sm text-[var(--foreground)]">
                  {row.reason ?? 'EXCEPTION'} · {row.manifest_id ?? 'manifest'} · {row.transfer_id || 'transfer_id missing'}
                </p>
              ))}
            </section>
          ) : null}

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">{t("factory_portal.app.text.network_pulse")}</p>
            <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">{t("factory_portal.app.text.cross_role_timeline")}</h2>
            <div className="mt-5">
              <NetworkPulsePanel />
            </div>
          </section>

          <DashboardMetrics stats={stats} />

          <DashboardAlerts stats={stats} />
        </div>
      </PageChrome>
    </PageTransition>
  );
}

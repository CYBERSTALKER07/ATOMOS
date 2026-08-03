'use client';

import { useCallback, useEffect, useState } from 'react';
import { usePolling } from '@pegasusx/api-client';
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
const LIVE_REFRESH_MS = 30_000;
type DashboardLoadIssue = 'offline' | 'restricted' | 'error';

export default function FactoryDashboard() {
  const [stats, setStats] = useState<FactoryStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadIssue, setLoadIssue] = useState<DashboardLoadIssue | null>(null);
  const [reloadToken, setReloadToken] = useState(0);

  const load = useCallback(async () => {
    try {
      const res = await apiFetch('/v1/factory/dashboard');
      if (!res.ok) {
        if (res.status === 401 || res.status === 403) {
          setLoadIssue('restricted');
        } else {
          setLoadIssue('error');
        }
        return;
      }

      setStats(await res.json());
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
          title="Factory dashboard"
          description="Transfer, bay, fleet, and staffing metrics for this node."
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
        <PageChrome icon="dashboard" title="Factory dashboard" description="Transfer, bay, fleet, and staffing metrics for this node.">
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
        <PageChrome icon="dashboard" title="Factory dashboard" description="Transfer, bay, fleet, and staffing metrics for this node.">
          <EmptyState
            variant="no-data"
            headline="No factory metrics yet"
            body="Once transfer and loading activity starts, operational metrics will appear here."
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
        title="Factory dashboard"
        description="Transfer, bay, fleet, and staffing metrics for this node."
      >
        <div className="space-y-8">
          <DashboardActionGrid stats={stats} />

          <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
            <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Network pulse</p>
            <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Cross-role timeline</h2>
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

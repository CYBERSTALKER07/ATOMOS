'use client';

import { useCallback, useEffect, useState } from 'react';
import { usePolling } from '@pegasusx/api-client';
import Link from 'next/link';
import { apiFetch, parseFactoryLiveEvent, subscribeFactoryWS } from '@/lib/auth';
import { useFactorySessionReconcile } from '@/lib/use-factory-session-reconcile';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import { PageChrome } from '@/components/PageChrome';
import { motion } from 'framer-motion';
import PageTransition from '@/components/PageTransition';
import NetworkPulsePanel from '@/components/NetworkPulsePanel';

interface FactoryStats {
  pending_transfers: number;
  loading_transfers: number;
  active_manifests: number;
  dispatched_today: number;
  vehicles_total: number;
  vehicles_available: number;
  staff_on_shift: number;
  critical_insights: number;
}

const EMPTY_STATS: FactoryStats = {
  pending_transfers: 0,
  loading_transfers: 0,
  active_manifests: 0,
  dispatched_today: 0,
  vehicles_total: 0,
  vehicles_available: 0,
  staff_on_shift: 0,
  critical_insights: 0,
};
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

  const s = stats;
  const primaryKpis = [
    { label: 'Pending Transfers', value: s.pending_transfers, icon: 'transfers', href: '/transfers', detail: 'Approved and waiting to load' },
    { label: 'Now Loading', value: s.loading_transfers, icon: 'loadingBay', href: '/loading-bay', detail: 'Active bay work this shift' },
    { label: 'Active Manifests', value: s.active_manifests, icon: 'manifests', href: '/loading-bay', detail: 'Payloads currently staged' },
    { label: 'Dispatched Today', value: s.dispatched_today, icon: 'fleet', href: '/transfers', detail: 'Outbound transfers completed' },
  ];
  const secondaryKpis = [
    { label: 'Vehicles Total', value: s.vehicles_total, icon: 'fleet', href: '/fleet', detail: `${s.vehicles_available} available now` },
    { label: 'Staff On Shift', value: s.staff_on_shift, icon: 'staff', href: '/staff', detail: 'Operators currently assigned' },
    { label: 'Gate Exceptions', value: s.critical_insights, icon: 'warning', href: '/manifest-exceptions', detail: 'Transfers removed during loading', danger: s.critical_insights > 0 },
  ];
  const actionCards = [
    {
      href: '/loading-bay',
      title: 'Move approved transfers into loading',
      description: 'Open the loading board, inspect ready payloads, and trigger dispatch without leaving the workspace.',
      icon: 'loadingBay',
    },
    {
      href: '/transfers',
      title: 'Inspect transfer pipeline',
      description: 'Review transfer states, warehouse destinations, priorities, and update cadence from the table view.',
      icon: 'transfers',
    },
    {
      href: '/manifests',
      title: 'Advance manifest lifecycle',
      description: 'Move manifests through draft, loading, sealed, dispatched, and completed states.',
      icon: 'manifests',
    },
    {
      href: '/manifest-exceptions',
      title: 'Review gate exceptions',
      description: 'Work transfers removed during loading and DLQ escalations before dispatch stalls.',
      icon: 'warning',
    },
    {
      href: '/insights',
      title: 'Review replenishment insights',
      description: 'Warehouse stock velocity and reorder pressure linked to this factory node.',
      icon: 'insights',
    },
    {
      href: '/analytics',
      title: 'Open analytics overview',
      description: 'Factory throughput, active manifests, exception queue, and lead time.',
      icon: 'analytics',
    },
  ];
  const readinessCards = [
    {
      label: 'Dispatch pressure',
      value: s.pending_transfers + s.loading_transfers,
      tone: s.pending_transfers + s.loading_transfers > 0 ? 'status-chip--warning' : 'status-chip--stable',
      helper: s.pending_transfers > 0 ? 'Approved transfers are waiting for bay time.' : 'Transfer queue is under control.',
    },
    {
      label: 'Fleet coverage',
      value: `${s.vehicles_available}/${s.vehicles_total}`,
      tone: s.vehicles_available > 0 ? 'status-chip--stable' : 'status-chip--critical',
      helper: s.vehicles_available > 0 ? 'Vehicles are available for new dispatch.' : 'No free vehicles right now.',
    },
    {
      label: 'Shift coverage',
      value: s.staff_on_shift,
      tone: s.staff_on_shift > 0 ? 'status-chip--stable' : 'status-chip--critical',
      helper: s.staff_on_shift > 0 ? 'Operators are clocked in for this cycle.' : 'No active operators detected.',
    },
  ];

  return (
    <PageTransition>
      <PageChrome
        icon="dashboard"
        title="Factory dashboard"
        description="Keep outbound transfer flow visible before loading gets congested."
        actions={
          <button
            type="button"
            className="desk-icon-btn"
            onClick={() => setReloadToken((v) => v + 1)}
            aria-label="Refresh dashboard"
          >
            <Icon name="refresh" size={18} />
          </button>
        }
      >
    <div className="space-y-8">
      <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6 shadow-[var(--shadow-md-elevation-1)]">
        <div className="grid gap-6 xl:grid-cols-[1.25fr_0.75fr]">
          <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
              {actionCards.map((card) => (
                <Link
                  key={card.href}
                  href={card.href}
                >
                <motion.div
                  whileTap={{ scale: 0.98 }}
                  className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)] hover-lift"
                >
                  <div className="flex items-center justify-between">
                    <div className="flex h-11 w-11 items-center justify-center rounded-2xl bg-[var(--background)]">
                      <Icon name={card.icon} size={20} />
                    </div>
                    <Icon name="chevronR" size={16} className="text-[var(--muted)]" />
                  </div>
                  <h3 className="mt-4 text-base font-semibold text-[var(--foreground)]">{card.title}</h3>
                  <p className="mt-1 text-sm leading-6 text-[var(--muted)]">{card.description}</p>
                </motion.div>
              </Link>
              ))}
          </div>

          <div className="grid gap-3 sm:grid-cols-3 xl:grid-cols-1">
            {readinessCards.map((item) => (
              <div key={item.label} className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4">
                <div className="flex items-center justify-between gap-3">
                  <span className={`status-chip ${item.tone}`}>{item.label}</span>
                  <span className="text-lg font-semibold tabular-nums text-[var(--foreground)]">{item.value}</span>
                </div>
                <p className="mt-3 text-sm leading-6 text-[var(--muted)]">{item.helper}</p>
              </div>
            ))}
          </div>
        </div>
      </section>

      <section className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
        <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Network pulse</p>
        <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Cross-role timeline</h2>
        <div className="mt-5">
          <NetworkPulsePanel />
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        {primaryKpis.map((kpi) => (
          <Link key={kpi.label} href={kpi.href}>
            <motion.div
              whileTap={{ scale: 0.98 }}
              className="rounded-2xl border border-[var(--border)] bg-[var(--background)] p-5 transition-colors hover:border-[var(--accent)] hover-lift h-full"
            >
              <div className="flex items-center justify-between">
                <span className="text-sm font-medium text-[var(--muted)]">{kpi.label}</span>
                <Icon name={kpi.icon} size={18} className="text-[var(--muted)]" />
              </div>
              <div className="mt-5 text-4xl font-semibold tracking-tight text-[var(--foreground)] tabular-nums">{kpi.value}</div>
              <p className="mt-2 text-sm text-[var(--muted)]">{kpi.detail}</p>
            </motion.div>
          </Link>
        ))}
      </section>

      <section className="grid gap-4 xl:grid-cols-[1.05fr_0.95fr]">
        <div className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
          <div className="flex items-center justify-between gap-3">
            <div>
              <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Operational health</p>
              <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Shift support signals</h2>
            </div>
            {s.critical_insights > 0 && <span className="status-chip status-chip--critical">Attention needed</span>}
          </div>

          <div className="mt-5 grid gap-4 md:grid-cols-3">
            {secondaryKpis.map((kpi) => (
              <Link key={kpi.label} href={kpi.href}>
                <motion.div
                  whileTap={{ scale: 0.98 }}
                  className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)] hover-lift h-full"
                >
                  <div className="flex items-center justify-between">
                    <Icon name={kpi.icon} size={18} className="text-[var(--muted)]" />
                    {kpi.danger && <span className="status-chip status-chip--critical">Alert</span>}
                  </div>
                  <div className="mt-4 text-2xl font-semibold tabular-nums text-[var(--foreground)]">{kpi.value}</div>
                  <div className="mt-1 text-sm font-medium text-[var(--foreground)]">{kpi.label}</div>
                  <p className="mt-2 text-sm leading-6 text-[var(--muted)]">{kpi.detail}</p>
                </motion.div>
              </Link>
            ))}
          </div>
        </div>

        <div className="rounded-[28px] border border-[var(--border)] bg-[var(--background)] p-6">
          <p className="text-[11px] font-semibold uppercase tracking-[0.18em] text-[var(--muted)]">Recommended next steps</p>
          <h2 className="mt-1 text-xl font-semibold tracking-tight text-[var(--foreground)]">Operator actions for this shift</h2>

          <div className="mt-5 space-y-3">
            {[
              {
                href: '/loading-bay',
                title: 'Clear approved transfers first',
                description: `${s.pending_transfers} transfer(s) are waiting for bay attention before dispatch can start.`,
              },
              {
                href: '/fleet',
                title: 'Confirm vehicle availability',
                description: `${s.vehicles_available} of ${s.vehicles_total} vehicles are free for assignment right now.`,
              },
              {
                href: '/manifest-exceptions',
                title: 'Review gate exceptions before dispatch',
                description: `${s.critical_insights} exception(s) need attention in the gate queue.`,
              },
            ].map((step) => (
              <Link key={step.href} href={step.href}>
                <motion.div
                  whileTap={{ scale: 0.98 }}
                  className="flex items-start gap-4 rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)] hover-lift h-full"
                >
                  <div className="mt-1 flex h-9 w-9 items-center justify-center rounded-full bg-[var(--background)]">
                    <Icon name="chevronR" size={16} />
                  </div>
                  <div className="min-w-0">
                    <h3 className="text-sm font-semibold text-[var(--foreground)]">{step.title}</h3>
                    <p className="mt-1 text-sm leading-6 text-[var(--muted)]">{step.description}</p>
                  </div>
                </motion.div>
              </Link>
            ))}
          </div>
        </div>
      </section>
    </div>
      </PageChrome>
    </PageTransition>
  );
}

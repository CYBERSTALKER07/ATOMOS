'use client';
import { usePolling } from '@pegasusx/api-react';


import { useCallback, useEffect, useRef, useState } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ApiError, moneyCurrency } from '@pegasusx/api-core';
import {
  ORDER_STATUS_FUNNEL,
  TRUCK_DUTY_STATUSES,
  canonicalizeOrderStatus,
  canonicalizeTruckDuty,
  emptyOrderStatusCounts,
  emptyTruckDutyCounts,
  type OrderStatusFunnel,
  type TruckDutyFunnel,
  type WarehouseHoldReason,
} from '@pegasusx/types';
import { warehouseApi } from '@/lib/warehouse-api';
import { useWarehouseSessionReconcile } from '@/lib/use-warehouse-session-reconcile';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
import PageTransition from '@/components/PageTransition';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import NetworkPulsePanel from '@/components/NetworkPulsePanel';
import { PageSection } from '@/components/PageSection';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';
import { SourceChip, StatusStack } from '@pegasusx/ui-kit/portal';
import { usePortalT } from '@/lib/i18n';

interface DashboardData {
  active_orders: number;
  completed_today_available: boolean;
  pending_dispatch: number;
  total_drivers: number;
  total_vehicles: number;
  today_revenue_available: boolean;
  low_stock_count: number;
  total_staff: number;
  history_available: boolean;
  orders_by_status: Record<OrderStatusFunnel, number>;
  truck_duty: Record<TruckDutyFunnel, number>;
  hold_reasons: WarehouseHoldReason[];
  demand_source: string;
}

function foldOrderCounts(raw: unknown): Record<OrderStatusFunnel, number> {
  const next = emptyOrderStatusCounts();
  if (!raw || typeof raw !== 'object') return next;
  for (const [key, value] of Object.entries(raw as Record<string, number>)) {
    const normalized = canonicalizeOrderStatus(key);
    if (normalized in next && Number.isFinite(Number(value))) {
      next[normalized as OrderStatusFunnel] = Number(value);
    }
  }
  return next;
}

function foldTruckDuty(raw: unknown): Record<TruckDutyFunnel, number> {
  const next = emptyTruckDutyCounts();
  if (!raw || typeof raw !== 'object') return next;
  for (const [key, value] of Object.entries(raw as Record<string, number>)) {
    const normalized = canonicalizeTruckDuty(key);
    if (normalized in next && Number.isFinite(Number(value))) {
      next[normalized as TruckDutyFunnel] = Number(value);
    }
  }
  return next;
}

function foldHoldReasons(raw: unknown): WarehouseHoldReason[] {
  if (!Array.isArray(raw)) return [];
  return raw
    .map((item) => {
      if (!item || typeof item !== 'object') return null;
      const row = item as { code?: string; count?: number };
      if (!row.code) return null;
      return { code: String(row.code), count: Number(row.count ?? 0) };
    })
    .filter((row): row is WarehouseHoldReason => row !== null);
}

type DashboardLoadIssue = 'offline' | 'restricted' | 'error';

export default function WarehouseDashboard() {
  const t = usePortalT();
  const router = useRouter();
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadIssue, setLoadIssue] = useState<DashboardLoadIssue | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const etagRef = useRef<string | null>(null);

  useWarehouseSessionReconcile(() => {
    setReloadToken((token) => token + 1);
  });

  const load = useCallback(async (silent = false) => {
    if (!silent) {
      setLoading(true);
    }
    try {
      const result = await warehouseApi.getWarehouseOpsDashboardConditional({}, etagRef.current ?? undefined);
      if (result.notModified) {
        if (result.etag) etagRef.current = result.etag;
        setLoadIssue(null);
        return;
      }
      etagRef.current = result.etag;
      const dashboard = result.data;
      const row = dashboard as unknown as Record<string, unknown>;
      setData({
        active_orders: Number(row.active_orders ?? 0),
        completed_today_available: row.completed_today_available === true,
        pending_dispatch: Number(row.pending_dispatch ?? 0),
        total_drivers: Number(row.total_drivers ?? 0),
        total_vehicles: Number(row.total_vehicles ?? 0),
        today_revenue_available: row.today_revenue_available === true,
        low_stock_count: Number(row.low_stock_count ?? 0),
        total_staff: Number(row.total_staff ?? 0),
        history_available: row.history_available === true,
        orders_by_status: foldOrderCounts(row.orders_by_status),
        truck_duty: foldTruckDuty(row.truck_duty),
        hold_reasons: foldHoldReasons(row.hold_reasons),
        demand_source: typeof row.demand_source === 'string' && row.demand_source ? row.demand_source : 'empty',
      });
      setLoadIssue(null);
    } catch (err) {
      if (err instanceof ApiError && (err.status === 401 || err.status === 403)) {
        setLoadIssue('restricted');
      } else if (typeof navigator !== 'undefined' && !navigator.onLine) {
        setLoadIssue('offline');
      } else {
        setLoadIssue('error');
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, reloadToken]);

  usePolling(
    async (signal) => {
      if (signal.aborted) return;
      await load(true);
    },
    60_000,
    [load, reloadToken],
    { pauseWhenHidden: true, immediate: false },
  );

  if (!data && loadIssue) {
    const stateContent: Record<DashboardLoadIssue, { headline: string; body: string }> = {
      offline: {
        headline: 'You are offline',
        body: 'Warehouse metrics are unavailable because the network connection dropped.',
      },
      restricted: {
        headline: 'Access restricted',
        body: 'Your role does not currently allow access to warehouse dashboard data.',
      },
      error: {
        headline: 'Unable to load dashboard',
        body: 'A server issue blocked this dashboard. Retry to load warehouse operations status.',
      },
    };

    const content = stateContent[loadIssue];

    return (
      <PageTransition className="p-6 space-y-6">
        <PageChrome icon="dashboard" title={t('portal.page.dashboard.warehouse.title')} description={t('portal.page.dashboard.warehouse.description')}>
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

  if (!data) {
    return (
      <PageTransition className="p-6 space-y-6">
        <PageChrome icon="dashboard" title={t('portal.page.dashboard.warehouse.title')} description={t('portal.page.dashboard.warehouse.description')}>
          <EmptyState
            variant="no-data"
            headline={t("warehouse_portal.residual.text.no_warehouse_metrics_yet")}
            body={t("warehouse_portal.residual.text.as_dispatch_fleet_and_inventory_activity_starts_this_dashboard_w")}
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

  const d = data;

  const fmt = (n: number) => new Intl.NumberFormat('en-US').format(n);
  const packCode = moneyCurrency();

  const kpis: {
    label: string;
    value: string;
    icon: string;
    href: string;
    bay: 'ops' | 'inventory' | 'fleet' | 'finance';
    flag?: 'alert' | 'ok';
  }[] = [
    { label: 'Pending dispatch', value: fmt(d.pending_dispatch), icon: 'dispatch', href: '/dispatch', bay: 'ops', flag: d.pending_dispatch > 5 ? 'alert' : undefined },
    { label: 'Active orders', value: fmt(d.active_orders), icon: 'orders', href: '/orders', bay: 'ops' },
    { label: 'Vehicles', value: fmt(d.total_vehicles), icon: 'fleet', href: '/vehicles', bay: 'fleet' },
    { label: 'Low stock items', value: fmt(d.low_stock_count), icon: 'warning', href: '/inventory', bay: 'inventory', flag: d.low_stock_count > 0 ? 'alert' : undefined },
    { label: 'Drivers', value: fmt(d.total_drivers), icon: 'fleet', href: '/drivers', bay: 'fleet' },
    { label: 'Staff', value: fmt(d.total_staff), icon: 'staff', href: '/staff', bay: 'ops' },
    { label: 'Completed today', value: d.completed_today_available ? 'live' : 'unavailable', icon: 'check', href: '/orders', bay: 'ops' },
    { label: 'Today revenue', value: d.today_revenue_available ? (packCode || 'live') : 'unavailable', icon: 'treasury', href: '/treasury', bay: 'finance' },
  ];

  return (
    <PageTransition>
      <PageChrome
        icon="dashboard"
        title={t('portal.page.dashboard.warehouse.title')}
        description={t('portal.page.dashboard.warehouse.description')}
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <motion.button
            whileHover={{ scale: 1.05 }}
            whileTap={{ scale: 0.95 }}
            onClick={() => setReloadToken((v) => v + 1)}
            className="desk-icon-btn"
            aria-label={t('portal.page.dashboard.action.refresh')}
          >
            <Icon name="refresh" size={18} />
          </motion.button>
        }
      >
      <KpiStatGrid columns={4}>
        {kpis.map((kpi) => (
          <KpiStatCard
            key={kpi.label}
            label={kpi.label}
            value={kpi.value}
            icon={kpi.icon}
            href={kpi.href}
            bay={kpi.bay}
            flag={kpi.flag}
          />
        ))}
      </KpiStatGrid>
      {!d.history_available ? (
        <p className="md-typescale-label-medium flex items-center gap-2" style={{ color: 'var(--wh-ink-muted)' }}>
          <SourceChip source="unavailable" />
          History unavailable
        </p>
      ) : null}

      <PageSection title="Orders now" description="Full warehouse order dictionary. Zero is a real count." bay="ops">
        <StatusStack
          dictionary={ORDER_STATUS_FUNNEL}
          counts={d.orders_by_status}
          source="live"
          onSelect={(key) => router.push(`/orders?state=${encodeURIComponent(key)}`)}
        />
      </PageSection>

      <PageSection title="Truck duty" description="Every duty key stays visible. Idle is not everything except in-transit." bay="fleet">
        <StatusStack dictionary={TRUCK_DUTY_STATUSES} counts={d.truck_duty} source="live" />
        {d.hold_reasons.length > 0 ? (
          <ul className="mt-3 flex flex-col gap-1" data-testid="gs-u-hold-reasons">
            {d.hold_reasons.map((row) => (
              <li key={row.code} className="md-typescale-body-small" style={{ color: 'var(--wh-ink-muted)' }}>
                {row.code} · {row.count}
              </li>
            ))}
          </ul>
        ) : (
          <p className="md-typescale-body-small mt-3" style={{ color: 'var(--wh-ink-muted)' }}>
            No hold reasons
          </p>
        )}
      </PageSection>

      <PageSection title="Demand" description="Planner source only. Empty is not a zero series." bay="ops">
        <div className="flex items-center gap-2" data-testid="gs-u-demand-source">
          <SourceChip source={d.demand_source === 'empty' ? 'empty' : d.demand_source} />
          <span className="md-typescale-body-small" style={{ color: 'var(--wh-ink-muted)' }}>
            {d.demand_source === 'empty' ? 'Planner empty' : `source ${d.demand_source}`}
          </span>
        </div>
        <Link href="/demand-forecast" className="portal-btn portal-btn--ghost text-sm mt-3 inline-flex">
          Open forecast
        </Link>
      </PageSection>

      <PageSection
        title={t("warehouse_portal.app.text.network_pulse")}
        description={t("warehouse_portal.residual.text.cross_role_timeline_for_this_warehouse_node")}
        bay="ops"
      >
        <NetworkPulsePanel />
      </PageSection>

      <PageSection
        title={t("warehouse_portal.dispatch.text.live_fleet_map")}
        description={t("warehouse_portal.residual.text.active_sealed_routes_and_driver_positions_for_this_node")}
        bay="fleet"
        className="overflow-hidden"
      >
        <FleetLiveMapPanel className="h-[360px] w-full -mx-5 -mb-5" />
      </PageSection>
      </PageChrome>
    </PageTransition>
  );
}

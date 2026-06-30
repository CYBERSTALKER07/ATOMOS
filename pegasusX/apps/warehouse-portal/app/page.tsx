'use client';

import { useEffect, useState } from 'react';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import { motion } from 'framer-motion';
import PageTransition from '@/components/PageTransition';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import NetworkPulsePanel from '@/components/NetworkPulsePanel';
import { PageSection } from '@/components/PageSection';
import { PageChrome } from '@/components/PageChrome';
import { KpiStatCard, KpiStatGrid } from '@/components/KpiStatCard';

interface DashboardData {
  active_orders: number;
  completed_today: number;
  pending_dispatch: number;
  total_drivers: number;
  on_route_drivers: number;
  idle_drivers: number;
  total_vehicles: number;
  today_revenue: number;
  low_stock_count: number;
  total_staff: number;
  fleet_status: FleetStatusRow[];
  sparkline_active_orders?: number[];
  sparkline_revenue?: number[];
  sparkline_completed?: number[];
}

type FleetStatusRow = { status: string; count: number };

function normalizeFleetStatus(raw: unknown): FleetStatusRow[] {
  if (Array.isArray(raw)) {
    return raw
      .map((item) => {
        if (!item || typeof item !== 'object') return null;
        const row = item as { status?: string; count?: number };
        if (!row.status) return null;
        return { status: row.status, count: Number(row.count ?? 0) };
      })
      .filter((row): row is FleetStatusRow => row !== null);
  }
  if (raw && typeof raw === 'object') {
    return Object.entries(raw as Record<string, number>).map(([status, count]) => ({
      status,
      count: Number(count),
    }));
  }
  return [];
}

type DashboardLoadIssue = 'offline' | 'restricted' | 'error';

export default function WarehouseDashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadIssue, setLoadIssue] = useState<DashboardLoadIssue | null>(null);
  const [reloadToken, setReloadToken] = useState(0);
  const [dateRange, setDateRange] = useState<'today' | '7d' | '30d'>('today');

  useEffect(() => {
    setLoading(true);
    async function load() {
      try {
        const dashboard = await warehouseApi.getWarehouseOpsDashboard({
          range: dateRange
        } as any);
        const row = dashboard as unknown as Record<string, unknown>;
        setData({
          active_orders: Number(row.active_orders ?? 0),
          completed_today: Number(row.completed_today ?? 0),
          pending_dispatch: Number(row.pending_dispatch ?? 0),
          total_drivers: Number(row.total_drivers ?? 0),
          on_route_drivers: Number(row.drivers_on_route ?? row.on_route_drivers ?? 0),
          idle_drivers: Number(row.drivers_idle ?? row.idle_drivers ?? 0),
          total_vehicles: Number(row.total_vehicles ?? 0),
          today_revenue: Number(row.today_revenue ?? 0),
          low_stock_count: Number(row.low_stock_count ?? 0),
          total_staff: Number(row.total_staff ?? 0),
          fleet_status: normalizeFleetStatus(row.fleet_status),
          sparkline_active_orders: Array.isArray(row.sparkline_active_orders) ? row.sparkline_active_orders.map(Number) : undefined,
          sparkline_revenue: Array.isArray(row.sparkline_revenue) ? row.sparkline_revenue.map(Number) : undefined,
          sparkline_completed: Array.isArray(row.sparkline_completed) ? row.sparkline_completed.map(Number) : undefined,
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
      }
      finally { setLoading(false); }
    }
    load();
  }, [reloadToken, dateRange]);

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
        <PageChrome icon="dashboard" title="Warehouse dashboard" description="Live node operations, fleet, and inventory at a glance.">
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
        <PageChrome icon="dashboard" title="Warehouse dashboard" description="Live node operations, fleet, and inventory at a glance.">
          <EmptyState
            variant="no-data"
            headline="No warehouse metrics yet"
            body="As dispatch, fleet, and inventory activity starts, this dashboard will populate automatically."
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
  const fmtCurrency = (n: number) => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'UZS', maximumFractionDigits: 0 }).format(n);

  const kpis: {
    label: string;
    value: string;
    icon: string;
    href: string;
    bay: 'ops' | 'inventory' | 'fleet' | 'finance';
    flag?: 'alert' | 'ok';
    sparkline?: number[];
  }[] = [
    { label: 'Active orders', value: fmt(d.active_orders), icon: 'orders', href: '/orders', bay: 'ops', sparkline: d.sparkline_active_orders },
    { label: 'Completed today', value: fmt(d.completed_today), icon: 'check', href: '/orders', bay: 'ops', flag: d.completed_today > 0 ? 'ok' : undefined, sparkline: d.sparkline_completed },
    { label: 'Pending dispatch', value: fmt(d.pending_dispatch), icon: 'dispatch', href: '/dispatch', bay: 'ops', flag: d.pending_dispatch > 5 ? 'alert' : undefined },
    { label: 'Today revenue', value: fmtCurrency(d.today_revenue), icon: 'treasury', href: '/treasury', bay: 'finance', sparkline: d.sparkline_revenue },
    { label: 'Drivers on route', value: `${d.on_route_drivers} / ${d.total_drivers}`, icon: 'fleet', href: '/drivers', bay: 'fleet' },
    { label: 'Idle drivers', value: fmt(d.idle_drivers), icon: 'fleet', href: '/drivers', bay: 'fleet' },
    { label: 'Vehicles', value: fmt(d.total_vehicles), icon: 'fleet', href: '/vehicles', bay: 'fleet' },
    { label: 'Low stock items', value: fmt(d.low_stock_count), icon: 'warning', href: '/inventory', bay: 'inventory', flag: d.low_stock_count > 0 ? 'alert' : undefined },
    { label: 'Total staff', value: fmt(d.total_staff), icon: 'staff', href: '/staff', bay: 'ops' },
  ];

  return (
    <PageTransition>
      <PageChrome
        icon="dashboard"
        title="Warehouse dashboard"
        description="Live node operations, fleet, and inventory at a glance."
        loading={loading}
        skeletonVariant="dashboard"
        actions={
          <div className="flex items-center gap-3">
            <select
              className="h-8 rounded-lg border border-[var(--wh-border)] bg-[var(--wh-surface)] px-2 text-xs text-[var(--wh-ink-main)] outline-none"
              value={dateRange}
              onChange={(e) => setDateRange(e.target.value as 'today' | '7d' | '30d')}
            >
              <option value="today">Today</option>
              <option value="7d">Last 7 days</option>
              <option value="30d">Last 30 days</option>
            </select>
            <motion.button
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
              onClick={() => setReloadToken((v) => v + 1)}
              className="desk-icon-btn"
              aria-label="Refresh dashboard"
            >
              <Icon name="refresh" size={18} />
            </motion.button>
          </div>
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
            sparkline={kpi.sparkline}
          />
        ))}
      </KpiStatGrid>

      <PageSection
        title="Network pulse"
        description="Cross-role timeline for this warehouse node."
        bay="ops"
      >
        <NetworkPulsePanel />
      </PageSection>

      <PageSection
        title="Live fleet map"
        description="Active sealed routes and driver positions for this node."
        bay="fleet"
        className="overflow-hidden"
      >
        <FleetLiveMapPanel className="h-[360px] w-full -mx-5 -mb-5" />
      </PageSection>

      {d.fleet_status.length > 0 && (
        <PageSection
          title="Fleet status"
          description="Manifest and driver state breakdown for this node."
          bay="fleet"
        >
          <div className="flex flex-wrap gap-3">
            {d.fleet_status.map(({ status, count }) => (
              <span
                key={status}
                className="inline-flex items-center gap-2 px-3 py-1.5 text-xs font-semibold rounded-lg"
                style={{
                  border: '1px solid var(--wh-border)',
                  background: 'var(--wh-surface-raised)',
                  color: 'var(--wh-ink-muted)',
                }}
              >
                {status.replace(/_/g, ' ')}
                <span className="wh-ops-card-amount" style={{ color: 'var(--wh-accent)' }}>
                  {count}
                </span>
              </span>
            ))}
          </div>
        </PageSection>
      )}
      </PageChrome>
    </PageTransition>
  );
}

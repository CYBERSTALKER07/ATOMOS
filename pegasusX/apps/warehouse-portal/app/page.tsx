'use client';

import { useEffect, useState } from 'react';
import { ApiError } from '@pegasusx/api-client';
import { warehouseApi } from '@/lib/warehouse-api';
import Icon from '@/components/Icon';
import EmptyState from '@/components/EmptyState';
import Link from 'next/link';
import { motion } from 'framer-motion';
import PageTransition from '@/components/PageTransition';
import FleetLiveMapPanel from '@/components/FleetLiveMapPanel';
import { PageSection } from '@/components/PageSection';
import { PageSkeleton } from '@/components/Skeleton';

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

  useEffect(() => {
    setLoading(true);
    async function load() {
      try {
        const dashboard = await warehouseApi.getWarehouseOpsDashboard();
        const row = dashboard as Record<string, unknown>;
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
  }, [reloadToken]);

  if (loading) {
    return (
      <div className="desk-page">
        <div className="desk-page-header">
          <div>
            <h1 className="desk-page-title">Warehouse dashboard</h1>
            <p className="desk-page-subtitle">Live node operations, fleet, and inventory at a glance.</p>
          </div>
        </div>
        <PageSkeleton variant="dashboard" />
      </div>
    );
  }

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
      </PageTransition>
    );
  }

  if (!data) {
    return (
      <PageTransition className="p-6 space-y-6">
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
      </PageTransition>
    );
  }

  const d = data || {
    active_orders: 0, completed_today: 0, pending_dispatch: 0,
    total_drivers: 0, on_route_drivers: 0, idle_drivers: 0,
    total_vehicles: 0, today_revenue: 0, low_stock_count: 0,
    total_staff: 0, fleet_status: [],
  };

  const fmt = (n: number) => new Intl.NumberFormat('en-US').format(n);
  const fmtCurrency = (n: number) => new Intl.NumberFormat('en-US', { style: 'currency', currency: 'UZS', maximumFractionDigits: 0 }).format(n);

  const kpis: { label: string; value: string; icon: string; href: string; danger?: boolean; highlight?: boolean }[] = [
    { label: 'Active Orders', value: fmt(d.active_orders), icon: 'orders', href: '/orders' },
    { label: 'Completed Today', value: fmt(d.completed_today), icon: 'check', href: '/orders', highlight: d.completed_today > 0 },
    { label: 'Pending Dispatch', value: fmt(d.pending_dispatch), icon: 'dispatch', href: '/dispatch', danger: d.pending_dispatch > 5 },
    { label: 'Today Revenue', value: fmtCurrency(d.today_revenue), icon: 'treasury', href: '/treasury' },
    { label: 'Drivers (On Route)', value: `${d.on_route_drivers} / ${d.total_drivers}`, icon: 'fleet', href: '/drivers' },
    { label: 'Idle Drivers', value: fmt(d.idle_drivers), icon: 'fleet', href: '/drivers' },
    { label: 'Vehicles', value: fmt(d.total_vehicles), icon: 'fleet', href: '/vehicles' },
    { label: 'Low Stock Items', value: fmt(d.low_stock_count), icon: 'warning', href: '/inventory', danger: d.low_stock_count > 0 },
    { label: 'Total Staff', value: fmt(d.total_staff), icon: 'staff', href: '/staff' },
  ];

  return (
    <PageTransition>
      <div className="desk-page space-y-6">
        <div className="desk-page-header">
          <div>
            <h1 className="desk-page-title">Warehouse dashboard</h1>
            <p className="desk-page-subtitle">Live node operations, fleet, and inventory at a glance.</p>
          </div>
          <div className="desk-toolbar">
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
        </div>

      <motion.div
        initial="hidden"
        animate="show"
        variants={{
          hidden: { opacity: 0 },
          show: { opacity: 1, transition: { staggerChildren: 0.05 } }
        }}
        className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4"
      >
        {kpis.map(kpi => (
          <Link key={kpi.label} href={kpi.href}>
            <motion.div
              variants={{
                hidden: { opacity: 0, y: 10 },
                show: { opacity: 1, y: 0 }
              }}
              whileHover={{ y: -4, scale: 1.02 }}
              whileTap={{ scale: 0.98 }}
              className="rounded-2xl border border-[var(--border)] p-5 flex flex-col gap-3 transition-all hover:border-[var(--primary)] hover:shadow-xl hover:shadow-[var(--primary)]/10 h-full bg-[var(--surface)] relative overflow-hidden group"
            >
              <div className="absolute top-0 left-0 w-1 h-full bg-[var(--primary)] opacity-0 group-hover:opacity-100 transition-opacity" />
              <div className="flex items-center justify-between">
                <div className="p-2 rounded-lg bg-[var(--default)] text-[var(--muted)] group-hover:text-[var(--primary)] transition-colors">
                  <Icon name={kpi.icon} size={20} />
                </div>
                {kpi.danger && (
                  <span className="status-chip status-chip--critical text-[10px] font-bold tracking-tighter">ALERT</span>
                )}
                {kpi.highlight && (
                  <span className="status-chip status-chip--ready text-[10px] font-bold tracking-tighter">DONE</span>
                )}
              </div>
              <div>
                <div className="text-2xl font-bold tracking-tight font-mono">{kpi.value}</div>
                <div className="text-xs font-bold uppercase tracking-widest text-[var(--muted)] mt-1">{kpi.label}</div>
              </div>
            </motion.div>
          </Link>
        ))}
      </motion.div>

      <PageSection
        title="Live fleet map"
        description="Active sealed routes and driver positions for this node."
        className="overflow-hidden"
      >
        <FleetLiveMapPanel className="h-[360px] w-full -mx-5 -mb-5" />
      </PageSection>

      {d.fleet_status.length > 0 && (
        <PageSection title="Fleet status tracking" description="Manifest and driver state breakdown for this node.">
          <div className="flex flex-wrap gap-3">
            {d.fleet_status.map(({ status, count }) => (
              <motion.span
                key={status}
                whileHover={{ scale: 1.05 }}
                className="px-4 py-2 rounded-xl text-xs font-bold border border-[var(--border)] bg-[var(--default)] transition-colors hover:border-[var(--primary)]"
              >
                {status.replace(/_/g, ' ')}: <span className="text-[var(--primary)] ml-1">{count}</span>
              </motion.span>
            ))}
          </div>
        </PageSection>
      )}
    </div>
    </PageTransition>
  );
}

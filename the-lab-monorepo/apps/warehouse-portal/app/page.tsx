'use client';

import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import Link from 'next/link';

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
  fleet_status: Record<string, number>;
}

export default function WarehouseDashboard() {
  const [data, setData] = useState<DashboardData | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const res = await apiFetch('/v1/warehouse/ops/dashboard');
        if (res.ok) setData(await res.json());
      } catch { /* empty state */ }
      finally { setLoading(false); }
    }
    load();
  }, []);

  if (loading) {
    return (
      <div className="p-6 space-y-6">
        <div className="md-skeleton md-skeleton-title" />
        <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
          {Array.from({ length: 8 }).map((_, i) => (
            <div key={i} className="md-skeleton md-skeleton-card" />
          ))}
        </div>
      </div>
    );
  }

  const d = data || {
    active_orders: 0, completed_today: 0, pending_dispatch: 0,
    total_drivers: 0, on_route_drivers: 0, idle_drivers: 0,
    total_vehicles: 0, today_revenue: 0, low_stock_count: 0,
    total_staff: 0, fleet_status: {},
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
    <div className="p-6 space-y-6 md-animate-in">
      <h1 className="text-xl font-bold tracking-tight">Warehouse Dashboard</h1>

      <div className="grid grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
        {kpis.map(kpi => (
          <Link
            key={kpi.label}
            href={kpi.href}
            className="rounded-xl border border-[var(--border)] p-4 flex flex-col gap-2 hover:border-[var(--accent)] transition-colors"
            style={{ background: 'var(--background)' }}
          >
            <div className="flex items-center justify-between">
              <Icon name={kpi.icon} size={20} className="text-[var(--muted)]" />
              {kpi.danger && (
                <span className="status-chip status-chip--critical text-[10px]">ALERT</span>
              )}
              {kpi.highlight && (
                <span className="status-chip status-chip--ready text-[10px]">DONE</span>
              )}
            </div>
            <div className="text-2xl font-bold">{kpi.value}</div>
            <div className="text-xs text-[var(--muted)]">{kpi.label}</div>
          </Link>
        ))}
      </div>

      {/* Fleet Status Breakdown */}
      {Object.keys(d.fleet_status).length > 0 && (
        <div>
          <h2 className="text-sm font-semibold mb-3 text-[var(--muted)]">Fleet Status</h2>
          <div className="flex flex-wrap gap-2">
            {Object.entries(d.fleet_status).map(([status, count]) => (
              <span
                key={status}
                className="px-3 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)]"
                style={{ background: 'var(--surface)' }}
              >
                {status.replace(/_/g, ' ')}: <strong>{count}</strong>
              </span>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

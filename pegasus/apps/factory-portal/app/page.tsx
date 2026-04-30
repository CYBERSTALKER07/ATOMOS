'use client';

import { useEffect, useState } from 'react';
import { apiFetch } from '@/lib/auth';
import Icon from '@/components/Icon';
import Link from 'next/link';

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

export default function FactoryDashboard() {
  const [stats, setStats] = useState<FactoryStats | null>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function load() {
      try {
        const res = await apiFetch('/v1/factory/dashboard');
        if (res.ok) {
          setStats(await res.json());
        }
      } catch {
        // empty state handled below
      } finally {
        setLoading(false);
      }
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

  const s = stats || {
    pending_transfers: 0, loading_transfers: 0, active_manifests: 0,
    dispatched_today: 0, vehicles_total: 0, vehicles_available: 0,
    staff_on_shift: 0, critical_insights: 0,
  };

  const kpis = [
    { label: 'Pending Transfers', value: s.pending_transfers, icon: 'transfers', href: '/transfers' },
    { label: 'Now Loading', value: s.loading_transfers, icon: 'loadingBay', href: '/loading-bay' },
    { label: 'Active Manifests', value: s.active_manifests, icon: 'manifests', href: '/loading-bay' },
    { label: 'Dispatched Today', value: s.dispatched_today, icon: 'fleet', href: '/transfers' },
    { label: 'Vehicles Total', value: s.vehicles_total, icon: 'fleet', href: '/fleet' },
    { label: 'Vehicles Available', value: s.vehicles_available, icon: 'fleet', href: '/fleet' },
    { label: 'Staff On Shift', value: s.staff_on_shift, icon: 'staff', href: '/staff' },
    { label: 'Critical Insights', value: s.critical_insights, icon: 'insights', href: '/insights', danger: s.critical_insights > 0 },
  ];

  return (
    <div className="p-6 space-y-6 md-animate-in">
      <h1 className="text-xl font-bold tracking-tight">Factory Dashboard</h1>

      <div className="grid grid-cols-2 lg:grid-cols-4 gap-4">
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
            </div>
            <div className="text-2xl font-bold">{kpi.value}</div>
            <div className="text-xs text-[var(--muted)]">{kpi.label}</div>
          </Link>
        ))}
      </div>
    </div>
  );
}

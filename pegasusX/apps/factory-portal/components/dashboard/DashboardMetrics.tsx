import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import { FactoryStats } from './types';

export function DashboardMetrics({ stats }: { stats: FactoryStats }) {
  const primaryKpis = [
    { label: 'Pending Transfers', value: stats.pending_transfers, icon: 'transfers', href: '/transfers', detail: 'Approved and waiting to load' },
    { label: 'Now Loading', value: stats.loading_transfers, icon: 'loadingBay', href: '/loading-bay', detail: 'Active bay work this shift' },
    { label: 'Active Manifests', value: stats.active_manifests, icon: 'manifests', href: '/loading-bay', detail: 'Payloads currently staged' },
    { label: 'Dispatched Today', value: stats.dispatched_today, icon: 'fleet', href: '/transfers', detail: 'Outbound transfers completed' },
  ];

  return (
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
  );
}

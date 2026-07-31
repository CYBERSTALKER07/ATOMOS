import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import { FactoryStats } from './types';

export function DashboardActionGrid({ stats }: { stats: FactoryStats }) {
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
      value: stats.pending_transfers + stats.loading_transfers,
      tone: stats.pending_transfers + stats.loading_transfers > 0 ? 'status-chip--warning' : 'status-chip--stable',
      helper: stats.pending_transfers > 0 ? 'Approved transfers are waiting for bay time.' : 'Transfer queue is under control.',
    },
    {
      label: 'Fleet coverage',
      value: `${stats.vehicles_available}/${stats.vehicles_total}`,
      tone: stats.vehicles_available > 0 ? 'status-chip--stable' : 'status-chip--critical',
      helper: stats.vehicles_available > 0 ? 'Vehicles are available for new dispatch.' : 'No free vehicles right now.',
    },
    {
      label: 'Shift coverage',
      value: stats.staff_on_shift,
      tone: stats.staff_on_shift > 0 ? 'status-chip--stable' : 'status-chip--critical',
      helper: stats.staff_on_shift > 0 ? 'Operators are clocked in for this cycle.' : 'No active operators detected.',
    },
  ];

  return (
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
                className="rounded-2xl border border-[var(--border)] bg-[var(--surface)] p-4 transition-colors hover:border-[var(--accent)] hover-lift h-full"
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
  );
}

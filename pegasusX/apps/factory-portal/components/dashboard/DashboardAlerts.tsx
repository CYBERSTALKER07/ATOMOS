import Link from 'next/link';
import { motion } from 'framer-motion';
import Icon from '@/components/Icon';
import { FactoryStats } from './types';

export function DashboardAlerts({ stats }: { stats: FactoryStats }) {
  const s = stats;
  const secondaryKpis = [
    { label: 'Vehicles Total', value: s.vehicles_total, icon: 'fleet', href: '/fleet', detail: `${s.vehicles_available} available now` },
    { label: 'Staff On Shift', value: s.staff_on_shift, icon: 'staff', href: '/staff', detail: 'Operators currently assigned' },
    { label: 'Gate Exceptions', value: s.critical_insights, icon: 'warning', href: '/manifest-exceptions', detail: 'Transfers removed during loading', danger: s.critical_insights > 0 },
  ];

  return (
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
                  <Icon name={kpi.icon as any} size={18} className="text-[var(--muted)]" />
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
  );
}

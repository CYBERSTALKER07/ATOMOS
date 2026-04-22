'use client';

import type { ThroughputBucket } from '@/hooks/useAdvancedAnalytics';

interface Props { data: ThroughputBucket[] }

export default function KPIStatsRow({ data }: Props) {
  if (!data.length) return null;

  const totalOrders = data.reduce((s, d) => s + d.order_count, 0);
  const totalCompleted = data.reduce((s, d) => s + d.completed_count, 0);
  const totalCancelled = data.reduce((s, d) => s + d.cancelled_count, 0);
  const completionRate = totalOrders > 0 ? (totalCompleted / totalOrders) * 100 : 0;
  const avgDaily = data.length > 0 ? Math.round(totalOrders / data.length) : 0;

  const stats = [
    { label: 'Total Orders', value: totalOrders.toLocaleString(), color: 'var(--color-md-primary)' },
    { label: 'Completed', value: totalCompleted.toLocaleString(), color: 'var(--color-md-success)' },
    { label: 'Cancelled', value: totalCancelled.toLocaleString(), color: 'var(--color-md-error)' },
    { label: 'Completion Rate', value: `${completionRate.toFixed(1)}%`, color: 'var(--color-md-tertiary)' },
    { label: 'Avg Daily', value: avgDaily.toLocaleString(), color: 'var(--color-md-secondary)' },
  ];

  return (
    <div className="flex flex-col gap-2 h-full justify-center">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Key Performance
      </h3>
      <div className="grid grid-cols-2 sm:grid-cols-3 gap-3">
        {stats.map((s) => (
          <div key={s.label} className="flex flex-col">
            <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>
              {s.label}
            </span>
            <span className="md-typescale-headline-small font-bold" style={{ color: s.color }}>
              {s.value}
            </span>
          </div>
        ))}
      </div>
    </div>
  );
}

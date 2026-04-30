'use client';

import CircularProgress from './CircularProgress';
import type { NodeMetric } from '@/hooks/useAdvancedAnalytics';

interface Props { data: NodeMetric[] }

export default function NodeEfficiencyGrid({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No node data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Warehouse Efficiency
      </h3>
      <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 gap-4 overflow-y-auto flex-1">
        {data.map((n) => {
          const rate = (n.on_time_rate ?? 0) * 100;
          const color = rate >= 80 ? 'var(--color-md-success)' : rate >= 50 ? 'var(--color-md-warning)' : 'var(--color-md-error)';
          return (
            <div
              key={n.warehouse_id}
              className="flex flex-col items-center gap-2 p-3"
              style={{ borderLeft: '2px solid var(--color-md-outline-variant)' }}
            >
              <CircularProgress value={rate} size={64} strokeWidth={5} color={color} />
              <span className="md-typescale-label-medium text-center truncate w-full" style={{ color: 'var(--color-md-on-surface)' }}>
                {n.warehouse_name || n.warehouse_id.slice(0, 8)}
              </span>
              <div className="flex gap-3">
                <div className="flex flex-col items-center">
                  <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>Orders</span>
                  <span className="md-typescale-body-small font-semibold" style={{ color: 'var(--color-md-on-surface)' }}>{n.order_count}</span>
                </div>
                <div className="flex flex-col items-center">
                  <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>Avg Min</span>
                  <span className="md-typescale-body-small font-semibold" style={{ color: 'var(--color-md-on-surface)' }}>{Math.round(n.avg_cycle_min)}</span>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}

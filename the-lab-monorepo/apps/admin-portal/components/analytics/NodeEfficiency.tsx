'use client';

import type { NodeMetric } from '@/hooks/useAnalytics';

export default function NodeEfficiency({ data }: { data: NodeMetric[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm font-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        No warehouse efficiency data
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Node Efficiency — Warehouses
      </h3>
      <div className="flex-1 overflow-auto">
        <table className="w-full text-left">
          <thead>
            <tr style={{ borderBottom: '1px solid var(--color-md-outline-variant)' }}>
              <th className="md-typescale-label-small py-2 pr-3" style={{ color: 'var(--color-md-on-surface-variant)' }}>Warehouse</th>
              <th className="md-typescale-label-small py-2 pr-3 text-right" style={{ color: 'var(--color-md-on-surface-variant)' }}>Orders</th>
              <th className="md-typescale-label-small py-2 pr-3 text-right" style={{ color: 'var(--color-md-on-surface-variant)' }}>Avg Cycle</th>
              <th className="md-typescale-label-small py-2 text-right" style={{ color: 'var(--color-md-on-surface-variant)' }}>On-Time</th>
            </tr>
          </thead>
          <tbody>
            {data.map((node) => {
              const onTimePct = (node.on_time_rate * 100);
              const onTimeColor = onTimePct >= 90
                ? 'var(--color-md-success)'
                : onTimePct >= 70
                  ? 'var(--color-md-warning)'
                  : 'var(--color-md-error)';
              return (
                <tr key={node.warehouse_id} style={{ borderBottom: '1px solid var(--color-md-outline-variant)' }}>
                  <td className="md-typescale-body-small py-2.5 pr-3" style={{ color: 'var(--color-md-on-surface)' }}>
                    {node.warehouse_name || node.warehouse_id.slice(0, 8)}
                  </td>
                  <td className="md-typescale-body-small py-2.5 pr-3 text-right tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                    {node.order_count.toLocaleString()}
                  </td>
                  <td className="md-typescale-body-small py-2.5 pr-3 text-right tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                    {node.avg_cycle_min.toFixed(0)}m
                  </td>
                  <td className="md-typescale-body-small py-2.5 text-right tabular-nums font-medium" style={{ color: onTimeColor }}>
                    {onTimePct.toFixed(1)}%
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    </div>
  );
}

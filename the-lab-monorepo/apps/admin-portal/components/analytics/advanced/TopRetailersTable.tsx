'use client';

import type { TopRetailer } from '@/hooks/useAdvancedAnalytics';

interface Props { data: TopRetailer[] }

export default function TopRetailersTable({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No retailer data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full overflow-hidden">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Top Retailers
      </h3>
      <div className="overflow-y-auto flex-1">
        <table className="w-full text-left">
          <thead>
            <tr>
              <th className="md-typescale-label-small pb-2 pr-4" style={{ color: 'var(--color-md-on-surface-variant)' }}>#</th>
              <th className="md-typescale-label-small pb-2 pr-4" style={{ color: 'var(--color-md-on-surface-variant)' }}>Shop</th>
              <th className="md-typescale-label-small pb-2 pr-4 text-right" style={{ color: 'var(--color-md-on-surface-variant)' }}>Orders</th>
              <th className="md-typescale-label-small pb-2 pr-4 text-right" style={{ color: 'var(--color-md-on-surface-variant)' }}>Revenue</th>
              <th className="md-typescale-label-small pb-2 text-right" style={{ color: 'var(--color-md-on-surface-variant)' }}>Avg</th>
            </tr>
          </thead>
          <tbody>
            {data.map((r, i) => (
              <tr key={r.retailer_id} className="border-t" style={{ borderColor: 'var(--color-md-outline-variant)' }}>
                <td className="md-typescale-body-small py-2 pr-4" style={{ color: 'var(--color-md-on-surface-variant)' }}>{i + 1}</td>
                <td className="md-typescale-body-small py-2 pr-4 truncate max-w-[140px]" style={{ color: 'var(--color-md-on-surface)' }}>
                  {r.shop_name}
                </td>
                <td className="md-typescale-body-small py-2 pr-4 text-right tabular-nums" style={{ color: 'var(--color-md-on-surface)' }}>
                  {r.order_count}
                </td>
                <td className="md-typescale-body-small py-2 pr-4 text-right tabular-nums font-medium" style={{ color: 'var(--color-md-primary)' }}>
                  {r.total_revenue.toLocaleString()}
                </td>
                <td className="md-typescale-body-small py-2 text-right tabular-nums" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                  {r.avg_order_value.toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}

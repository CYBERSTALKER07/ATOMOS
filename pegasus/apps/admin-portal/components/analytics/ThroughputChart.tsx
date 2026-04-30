'use client';

import {
  AreaChart,
  Area,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Legend,
} from 'recharts';
import type { ThroughputBucket } from '@/hooks/useAnalytics';

export default function ThroughputChart({ data }: { data: ThroughputBucket[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm font-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        No throughput data
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Order Throughput — 30 Days
      </h3>
      <div className="flex-1 min-h-[200px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={data} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-md-outline-variant)" vertical={false} />
            <XAxis
              dataKey="date"
              stroke="var(--color-md-outline)"
              tick={{ fill: 'var(--color-md-on-surface-variant)', fontSize: 11 }}
              tickLine={false}
              tickFormatter={(v) => v.slice(5)}
            />
            <YAxis
              stroke="var(--color-md-outline)"
              tick={{ fill: 'var(--color-md-on-surface-variant)', fontSize: 11 }}
              tickLine={false}
              axisLine={false}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-md-surface-container)',
                border: '1px solid var(--color-md-outline-variant)',
                borderRadius: '12px',
                fontSize: 12,
              }}
            />
            <Legend wrapperStyle={{ fontSize: 11 }} />
            <Area
              type="monotone"
              dataKey="completed_count"
              name="Completed"
              stroke="oklch(0.723 0.219 149.579)"
              fill="oklch(0.723 0.219 149.579 / 0.15)"
              strokeWidth={2}
            />
            <Area
              type="monotone"
              dataKey="cancelled_count"
              name="Cancelled"
              stroke="oklch(0.577 0.245 27.325)"
              fill="oklch(0.577 0.245 27.325 / 0.1)"
              strokeWidth={2}
            />
            <Area
              type="monotone"
              dataKey="order_count"
              name="Total"
              stroke="oklch(0.6 0.118 264.376)"
              fill="oklch(0.6 0.118 264.376 / 0.08)"
              strokeWidth={1.5}
              strokeDasharray="4 4"
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

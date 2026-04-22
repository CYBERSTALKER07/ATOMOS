'use client';

import {
  BarChart,
  Bar,
  XAxis,
  YAxis,
  Tooltip,
  ResponsiveContainer,
  CartesianGrid,
  Cell,
} from 'recharts';
import type { LoadBucket } from '@/hooks/useAnalytics';

const CLASS_COLORS: Record<string, string> = {
  CLASS_A: 'oklch(0.6 0.118 264.376)',
  CLASS_B: 'oklch(0.723 0.219 149.579)',
  CLASS_C: 'oklch(0.769 0.188 70.08)',
};

export default function LoadDistribution({ data }: { data: LoadBucket[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full text-sm font-medium" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        No fleet load data
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <div className="flex items-center gap-1">
        <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
          Fleet Load Distribution
        </h3>
        <span className="md-typescale-label-small ml-auto" style={{ color: 'var(--color-md-on-surface-variant)' }}>
          7-day window
        </span>
      </div>
      <div className="flex-1 min-h-[180px]">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--color-md-outline-variant)" vertical={false} />
            <XAxis
              dataKey="vehicle_class"
              stroke="var(--color-md-outline)"
              tick={{ fill: 'var(--color-md-on-surface-variant)', fontSize: 11 }}
              tickLine={false}
            />
            <YAxis
              stroke="var(--color-md-outline)"
              tick={{ fill: 'var(--color-md-on-surface-variant)', fontSize: 11 }}
              tickLine={false}
              axisLine={false}
              domain={[0, 100]}
              tickFormatter={(v) => `${v}%`}
            />
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-md-surface-container)',
                border: '1px solid var(--color-md-outline-variant)',
                borderRadius: '12px',
                fontSize: 12,
              }}
              formatter={(value) => `${Number(value).toFixed(1)}%`}
            />
            <Bar dataKey="avg_load_pct" name="Avg Load %" radius={[6, 6, 0, 0]}>
              {data.map((entry) => (
                <Cell key={entry.vehicle_class} fill={CLASS_COLORS[entry.vehicle_class] || 'var(--color-md-primary)'} />
              ))}
            </Bar>
          </BarChart>
        </ResponsiveContainer>
      </div>
      {/* Summary chips */}
      <div className="flex flex-wrap gap-2">
        {data.map((b) => (
          <div
            key={b.vehicle_class}
            className="md-chip md-typescale-label-small px-3 py-1"
            style={{ borderColor: 'var(--color-md-outline-variant)' }}
          >
            {b.vehicle_class}: {b.vehicle_count} trucks, max {b.max_load_pct.toFixed(0)}%
          </div>
        ))}
      </div>
    </div>
  );
}

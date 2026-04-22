'use client';

import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts';
import type { FactoryDayBucket } from '@/hooks/useAdvancedAnalytics';

interface Props { data: FactoryDayBucket[] }

export default function FactoryActivityChart({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No factory data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Factory Activity
      </h3>
      <div className="flex-1 min-h-[200px]">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 4, right: 4, left: -10, bottom: 0 }}>
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
            <Bar dataKey="transfers_created" name="Created" fill="oklch(0.6 0.118 264.376)" radius={[4, 4, 0, 0]} />
            <Bar dataKey="transfers_shipped" name="Shipped" fill="oklch(0.723 0.219 149.579)" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

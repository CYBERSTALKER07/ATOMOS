'use client';

import {
  BarChart, Bar, XAxis, YAxis, Tooltip, ResponsiveContainer, CartesianGrid,
} from 'recharts';
import type { SLAEntry } from '@/hooks/useAdvancedAnalytics';

interface Props { data: SLAEntry[] }

export default function SLAStackedBar({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No SLA data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        SLA Compliance
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
            <Bar dataKey="on_time" name="On Time" fill="oklch(0.723 0.219 149.579)" stackId="sla" radius={[0, 0, 0, 0]} />
            <Bar dataKey="late" name="Late" fill="oklch(0.75 0.183 55.934)" stackId="sla" />
            <Bar dataKey="breached" name="Breached" fill="oklch(0.577 0.245 27.325)" stackId="sla" radius={[4, 4, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

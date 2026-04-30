'use client';

import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer, Legend } from 'recharts';
import type { FactoryStatusSummary } from '@/hooks/useAdvancedAnalytics';

const STATE_COLORS: Record<string, string> = {
  PENDING: 'oklch(0.65 0.12 60)',
  IN_TRANSIT: 'oklch(0.6 0.2 240)',
  SHIPPED: 'oklch(0.723 0.219 149.579)',
  RECEIVED: 'oklch(0.6 0.15 300)',
  CANCELLED: 'oklch(0.577 0.245 27.325)',
};

const FALLBACK_COLORS = [
  'oklch(0.6 0.118 264.376)',
  'oklch(0.7 0.15 200)',
  'oklch(0.55 0.2 350)',
];

interface Props { data: FactoryStatusSummary[] }

export default function FactoryTransfersPie({ data }: Props) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-full" style={{ color: 'var(--color-md-on-surface-variant)' }}>
        <span className="md-typescale-body-medium">No transfer data</span>
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-3 h-full">
      <h3 className="md-typescale-title-small" style={{ color: 'var(--color-md-on-surface)' }}>
        Transfers by State
      </h3>
      <div className="flex-1 min-h-[180px]">
        <ResponsiveContainer width="100%" height="100%">
          <PieChart>
            <Pie
              data={data}
              dataKey="count"
              nameKey="state"
              cx="50%"
              cy="50%"
              innerRadius="35%"
              outerRadius="65%"
              paddingAngle={2}
            >
              {data.map((d, i) => (
                <Cell key={i} fill={STATE_COLORS[d.state] ?? FALLBACK_COLORS[i % FALLBACK_COLORS.length]} />
              ))}
            </Pie>
            <Tooltip
              contentStyle={{
                backgroundColor: 'var(--color-md-surface-container)',
                border: '1px solid var(--color-md-outline-variant)',
                borderRadius: '12px',
                fontSize: 12,
              }}
            />
            <Legend wrapperStyle={{ fontSize: 11 }} />
          </PieChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

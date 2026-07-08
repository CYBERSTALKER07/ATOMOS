'use client';

import {
  Area,
  AreaChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
  Legend
} from 'recharts';

type ProductionForecastChartProps = {
  className?: string;
};

// Removed Mock Data
const MOCK_FORECAST_DATA: Record<string, unknown>[] = [];

export default function ProductionForecastChart({ className }: ProductionForecastChartProps) {
  return (
    <div className={className}>
      <ResponsiveContainer width="100%" height="100%">
        <AreaChart data={MOCK_FORECAST_DATA} margin={{ top: 10, right: 30, left: 0, bottom: 0 }}>
          <defs>
            <linearGradient id="colorMaterials" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--desk-accent)" stopOpacity={0.3}/>
              <stop offset="95%" stopColor="var(--desk-accent)" stopOpacity={0}/>
            </linearGradient>
            <linearGradient id="colorBurn" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="var(--desk-danger, #ef4444)" stopOpacity={0.3}/>
              <stop offset="95%" stopColor="var(--desk-danger, #ef4444)" stopOpacity={0}/>
            </linearGradient>
          </defs>
          <XAxis dataKey="day" stroke="var(--desk-text-secondary)" tick={{ fill: 'var(--desk-text-secondary)', fontSize: 12 }} />
          <YAxis stroke="var(--desk-text-secondary)" tick={{ fill: 'var(--desk-text-secondary)', fontSize: 12 }} />
          <CartesianGrid strokeDasharray="3 3" stroke="var(--desk-border)" vertical={false} />
          <Tooltip 
            contentStyle={{ backgroundColor: 'var(--desk-surface)', borderColor: 'var(--desk-border)', borderRadius: '8px' }}
            itemStyle={{ color: 'var(--desk-text-primary)' }}
          />
          <Legend wrapperStyle={{ paddingTop: '20px' }} />
          <Area 
            type="monotone" 
            dataKey="rawMaterials" 
            name="Raw Materials Stock (Tons)"
            stroke="var(--desk-accent)" 
            fillOpacity={1} 
            fill="url(#colorMaterials)" 
          />
          <Area 
            type="monotone" 
            dataKey="burnRate" 
            name="Expected Burn Rate (Tons)"
            stroke="var(--desk-danger, #ef4444)" 
            fillOpacity={1} 
            fill="url(#colorBurn)" 
          />
        </AreaChart>
      </ResponsiveContainer>
    </div>
  );
}

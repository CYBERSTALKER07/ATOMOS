'use client';

import { 
  BarChart, 
  Bar, 
  XAxis, 
  YAxis, 
  Tooltip, 
  ResponsiveContainer,
  CartesianGrid
} from 'recharts';

type SkuVelocity = {
  sku_id: string;
  total_pallets: number;
  gross_volume: number;
};

export default function VelocityChart({ data }: { data: SkuVelocity[] }) {
  if (!data || data.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 md-shape-lg md-elevation-1 text-sm font-medium" style={{ background: 'var(--surface)', color: 'var(--muted)' }}>
        No velocity data detected
      </div>
    );
  }

  const formatAmount = (value: number) => 
    new Intl.NumberFormat('uz-UZ', { style: 'currency', currency: 'UZS', maximumFractionDigits: 0 }).format(value);

  return (
    <div className="w-full md-shape-lg md-elevation-1 p-6" style={{ background: 'var(--surface)' }}>
      <h3 className="text-[11px] font-medium tracking-wide mb-6" style={{ color: 'var(--muted)' }}>
        SKU Velocity — Volume (Amount)
      </h3>
      <div className="h-80">
        <ResponsiveContainer width="100%" height="100%">
          <BarChart data={data} margin={{ top: 0, right: 0, left: 20, bottom: 0 }}>
            <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" vertical={false} />
            <XAxis 
              dataKey="sku_id" 
              stroke="var(--border)" 
              tick={{ fill: 'var(--muted)', fontSize: 11, fontWeight: 500 }} 
              tickLine={false}
              axisLine={{ stroke: 'var(--border)' }}
            />
            <YAxis 
              stroke="var(--border)" 
              tick={{ fill: 'var(--muted)', fontSize: 11 }}
              tickFormatter={(value) => `${(value / 1000000).toFixed(1)}M`}
              tickLine={false}
              axisLine={false}
            />
            <Tooltip 
              cursor={{ fill: 'var(--surface)' }}
              contentStyle={{ 
                backgroundColor: 'var(--surface)', 
                border: '1px solid var(--border)', 
                borderRadius: '12px', 
                boxShadow: '0 2px 6px 2px rgba(0,0,0,0.15), 0 1px 2px 0 rgba(0,0,0,0.3)',
                fontSize: '13px',
                color: 'var(--foreground)',
              }}
              itemStyle={{ color: 'var(--foreground)', fontWeight: 500 }}
              formatter={(value) => [formatAmount(Number(value ?? 0)), 'Gross Volume']}
            />
            <Bar dataKey="gross_volume" fill="var(--accent)" radius={[6, 6, 0, 0]} />
          </BarChart>
        </ResponsiveContainer>
      </div>
    </div>
  );
}

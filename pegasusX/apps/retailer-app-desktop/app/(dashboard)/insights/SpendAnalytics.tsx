'use client';

import {
  PieChart,
  Pie,
  Cell,
  BarChart,
  Bar,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  Legend
} from 'recharts';

type SpendAnalyticsProps = {
  className?: string;
};

// Removed Mock Data
const MOCK_CATEGORY_SPEND: Record<string, unknown>[] = [];

const COLORS = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444'];

const MOCK_SPEND_TREND: Record<string, unknown>[] = [];

const fmtCurrency = (n: number) => new Intl.NumberFormat('uz-UZ', { maximumFractionDigits: 0 }).format(n);

export default function SpendAnalytics({ className }: SpendAnalyticsProps) {
  return (
    <div className={className}>
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        
        {/* Category Pie Chart */}
        <div className="h-80 w-full p-4 border rounded-xl" style={{ borderColor: 'var(--color-md-outline-variant)', background: 'var(--color-md-surface-container)' }}>
          <h3 className="text-sm font-semibold mb-2 text-center" style={{ color: 'var(--color-md-on-surface)' }}>Spend by Category</h3>
          <ResponsiveContainer width="100%" height="90%">
            <PieChart>
              <Pie
                data={MOCK_CATEGORY_SPEND}
                cx="50%"
                cy="50%"
                innerRadius={60}
                outerRadius={80}
                paddingAngle={5}
                dataKey="value"
              >
                {MOCK_CATEGORY_SPEND.map((entry, index) => (
                  <Cell key={`cell-${index}`} fill={COLORS[index % COLORS.length]} />
                ))}
              </Pie>
              <Tooltip 
                formatter={(value) => [`${fmtCurrency(Number(Array.isArray(value) ? value[0] : value) || 0)} UZS`, 'Spend']} 
                contentStyle={{ backgroundColor: 'var(--color-md-surface)', borderColor: 'var(--color-md-outline)', borderRadius: '8px', color: 'var(--color-md-on-surface)' }}
              />
              <Legend verticalAlign="bottom" height={36} />
            </PieChart>
          </ResponsiveContainer>
        </div>

        {/* 7-Day Spend Trend */}
        <div className="h-80 w-full p-4 border rounded-xl" style={{ borderColor: 'var(--color-md-outline-variant)', background: 'var(--color-md-surface-container)' }}>
          <h3 className="text-sm font-semibold mb-2 text-center" style={{ color: 'var(--color-md-on-surface)' }}>7-Day Spending Trend</h3>
          <ResponsiveContainer width="100%" height="90%">
            <BarChart data={MOCK_SPEND_TREND} margin={{ top: 10, right: 10, left: 10, bottom: 0 }}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--color-md-outline-variant)" vertical={false} />
              <XAxis dataKey="day" stroke="var(--color-md-on-surface-variant)" tick={{ fontSize: 12, fill: 'var(--color-md-on-surface-variant)' }} />
              <YAxis tickFormatter={(val) => `${(val / 1000000).toFixed(1)}M`} stroke="var(--color-md-on-surface-variant)" tick={{ fontSize: 12, fill: 'var(--color-md-on-surface-variant)' }} />
              <Tooltip 
                formatter={(value) => [`${fmtCurrency(Number(Array.isArray(value) ? value[0] : value) || 0)} UZS`, 'Spend']} 
                contentStyle={{ backgroundColor: 'var(--color-md-surface)', borderColor: 'var(--color-md-outline)', borderRadius: '8px', color: 'var(--color-md-on-surface)' }}
              />
              <Bar dataKey="spend" fill="var(--color-md-primary)" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        </div>

      </div>
    </div>
  );
}

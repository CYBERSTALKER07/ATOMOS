import { BarChart, Bar, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts';

interface AnalyticsChartGridProps {
  dailySeries: { date: string; revenue: number; orders: number }[];
  fmtCurrency: (n: number) => string;
}

export default function AnalyticsChartGrid({ dailySeries, fmtCurrency }: AnalyticsChartGridProps) {
  return (
    <div className="grid grid-cols-1 gap-6">
      {/* Daily Revenue Chart — VelocityGauge unmounted (no avg-dispatch SoT) */}
      <div className="rounded-xl border border-[var(--border)] p-4" style={{ background: 'var(--background)' }}>
        <h2 className="text-sm font-semibold mb-4">Daily Revenue</h2>
        {dailySeries.length > 0 ? (
          <ResponsiveContainer width="100%" height={240}>
            <BarChart data={dailySeries}>
              <CartesianGrid strokeDasharray="3 3" stroke="var(--border)" />
              <XAxis
                dataKey="date"
                tick={{ fontSize: 11 }}
                stroke="var(--muted)"
                tickFormatter={(value: string) => (value.length >= 10 ? value.slice(5, 10) : value)}
              />
              <YAxis tick={{ fontSize: 11 }} stroke="var(--muted)" />
              <Tooltip formatter={(value) => [`${fmtCurrency(Number(value ?? 0))} UZS`, 'Revenue']} />
              <Bar dataKey="revenue" fill="var(--accent)" radius={[4, 4, 0, 0]} />
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <p className="text-sm text-[var(--muted)] py-8 text-center">
            No completed-order revenue in this period. Daily breakdown populates from Spanner `daily_breakdown`.
          </p>
        )}
      </div>
    </div>
  );
}

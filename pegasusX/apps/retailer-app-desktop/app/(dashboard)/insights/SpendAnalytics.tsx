'use client';

type SpendAnalyticsProps = {
  className?: string;
  categorySpend?: { name: string; value: number }[];
  spendTrend?: { day: string; spend: number }[];
};

export default function SpendAnalytics({
  className,
  categorySpend = [],
  spendTrend = [],
}: SpendAnalyticsProps) {
  const empty = categorySpend.length === 0 && spendTrend.length === 0;

  return (
    <div className={className}>
      {empty ? (
        <div
          className="h-80 w-full p-6 border rounded-xl flex items-center justify-center"
          style={{
            borderColor: 'var(--color-md-outline-variant)',
            background: 'var(--color-md-surface-container)',
          }}
        >
          <p className="text-sm text-center" style={{ color: 'var(--color-md-on-surface-variant)' }}>
            No spend analytics yet. Charts appear after completed orders produce expense history.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div
            className="h-80 w-full p-4 border rounded-xl overflow-auto"
            style={{
              borderColor: 'var(--color-md-outline-variant)',
              background: 'var(--color-md-surface-container)',
            }}
          >
            <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-md-on-surface)' }}>
              Spend by Category
            </h3>
            {categorySpend.length === 0 ? (
              <p className="text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                No category breakdown.
              </p>
            ) : (
              <ul className="space-y-2 text-sm">
                {categorySpend.map((row) => (
                  <li key={row.name} className="flex justify-between gap-4">
                    <span>{row.name}</span>
                    <span className="font-mono tabular-nums">{row.value}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
          <div
            className="h-80 w-full p-4 border rounded-xl overflow-auto"
            style={{
              borderColor: 'var(--color-md-outline-variant)',
              background: 'var(--color-md-surface-container)',
            }}
          >
            <h3 className="text-sm font-semibold mb-3" style={{ color: 'var(--color-md-on-surface)' }}>
              7-Day Spending Trend
            </h3>
            {spendTrend.length === 0 ? (
              <p className="text-xs" style={{ color: 'var(--color-md-on-surface-variant)' }}>
                No trend series.
              </p>
            ) : (
              <ul className="space-y-2 text-sm">
                {spendTrend.map((row) => (
                  <li key={row.day} className="flex justify-between gap-4">
                    <span>{row.day}</span>
                    <span className="font-mono tabular-nums">{row.spend}</span>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

'use client';

type ProductionForecastChartProps = {
  className?: string;
  series?: { day: string; rawMaterials: number; burnRate: number }[];
};

export default function ProductionForecastChart({
  className,
  series = [],
}: ProductionForecastChartProps) {
  return (
    <div className={className}>
      {series.length === 0 ? (
        <div className="h-full min-h-[240px] flex items-center justify-center px-6">
          <p className="text-sm text-center text-[var(--desk-text-secondary)]">
            No production forecast series yet. Data appears when materials and burn-rate analytics are available.
          </p>
        </div>
      ) : (
        <ul className="space-y-2 p-4 text-sm">
          {series.map((row) => (
            <li key={row.day} className="flex justify-between gap-4 border-b border-[var(--desk-border)] py-2">
              <span>{row.day}</span>
              <span className="font-mono tabular-nums text-[var(--desk-text-secondary)]">
                stock {row.rawMaterials} · burn {row.burnRate}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

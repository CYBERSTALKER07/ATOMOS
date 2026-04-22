export default function StatsCard({
  label,
  value,
  sub,
  trend,
  accent,
  delay = 0,
  className = '',
}: {
  label: string;
  value: string;
  sub?: string;
  trend?: 'up' | 'down' | 'neutral';
  accent?: string;
  delay?: number;
  className?: string;
}) {
  const trendIcon = trend === 'up' ? '↑' : trend === 'down' ? '↓' : null;
  const trendColor = trend === 'up'
    ? 'var(--success)'
    : trend === 'down'
    ? 'var(--danger)'
    : 'var(--muted)';

  return (
    <div
      className={`md-card md-card-elevated p-6 flex flex-col justify-between cursor-default md-animate-in overflow-hidden relative ${className}`}
      style={{ animationDelay: `${delay}ms` }}
    >
      {accent && (
        <div
          className="absolute top-0 left-0 w-full h-1"
          style={{ background: accent }}
          aria-hidden="true"
        />
      )}
      <p className="md-typescale-label-small mb-4" style={{ color: 'var(--muted)' }}>
        {label}
      </p>
      <div>
        <div className="flex items-baseline gap-2">
          <p
            className="md-typescale-headline-small tracking-tight"
            style={{ color: 'var(--foreground)', fontVariantNumeric: 'tabular-nums' }}
          >
            {value}
          </p>
          {trendIcon && (
            <span className="md-typescale-label-medium" style={{ color: trendColor }}>
              {trendIcon}
            </span>
          )}
        </div>
        {sub && (
          <p className="md-typescale-label-small mt-1" style={{ color: 'var(--muted)' }}>
            {sub}
          </p>
        )}
      </div>
    </div>
  );
}

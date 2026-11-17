import { motion } from 'framer-motion';
import CountUp from './CountUp';

export default function StatsCard({
  label,
  value,
  sub,
  trend,
  trendValue,
  accent,
  delay = 0,
  sparkline,
  className = '',
}: {
  label: string;
  value: string;
  sub?: string;
  trend?: 'up' | 'down' | 'neutral';
  trendValue?: string;
  accent?: string;
  delay?: number;
  sparkline?: number[];
  className?: string;
}) {
  const trendIcon = trend === 'up' ? '↑' : trend === 'down' ? '↓' : trend === 'neutral' ? '→' : null;
  const deltaClass =
    trend === 'up' ? 'desk-stat-delta desk-stat-delta--up'
    : trend === 'down' ? 'desk-stat-delta desk-stat-delta--down'
    : trend === 'neutral' ? 'desk-stat-delta desk-stat-delta--neutral'
    : '';

  // Generate simple SVG sparkline path
  const sparklinePath = sparkline && sparkline.length > 1 ? (() => {
    const max = Math.max(...sparkline);
    const min = Math.min(...sparkline);
    const range = max - min || 1;
    const h = 36;
    const w = 100;
    const step = w / (sparkline.length - 1);
    return sparkline
      .map((v, i) => {
        const x = i * step;
        const y = h - ((v - min) / range) * (h - 4) - 2;
        return `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`;
      })
      .join(' ');
  })() : null;

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.35, delay: delay / 1000, ease: [0.16, 1, 0.3, 1] }}
      className={`desk-kpi-card ${className}`}
      style={{ position: 'relative' }}
    >
      {/* Label row */}
      <span className="desk-kpi-card-label">{label}</span>

      {/* Value + trend */}
      <div style={{ display: 'flex', alignItems: 'baseline', gap: 10 }}>
        <div className="desk-kpi-card-value">
          <CountUp value={value} delay={delay / 1000 + 0.2} className="tabular-nums" />
        </div>
        {trendIcon && (
          <span className={deltaClass}>
            {trendIcon} {trendValue || ''}
          </span>
        )}
      </div>

      {/* Sparkline */}
      {sparklinePath && (
        <svg className="desk-sparkline" viewBox="0 0 100 36" preserveAspectRatio="none">
          <path d={sparklinePath} />
        </svg>
      )}

      {/* Sub text */}
      {sub && (
        <div className="desk-kpi-card-meta" style={{ marginTop: 4 }}>
          <span style={{ font: 'var(--type-caption-sm)', color: 'var(--desk-text-tertiary)' }}>{sub}</span>
        </div>
      )}

      {/* Top accent bar */}
      {accent && (
        <div
          aria-hidden="true"
          style={{
            position: 'absolute',
            top: 0,
            left: 0,
            right: 0,
            height: 2,
            background: accent,
            borderTopLeftRadius: 'inherit',
            borderTopRightRadius: 'inherit',
          }}
        />
      )}
    </motion.div>
  );
}

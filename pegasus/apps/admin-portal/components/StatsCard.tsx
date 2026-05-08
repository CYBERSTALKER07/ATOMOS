import { motion } from 'framer-motion';
import CountUp from './CountUp';

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
  const trendIcon = trend === 'up' ? '↑' : trend === 'down' ? '↓' : trend === 'neutral' ? '→' : null;
  const deltaClass =
    trend === 'up' ? 'desk-stat-delta desk-stat-delta--up'
    : trend === 'down' ? 'desk-stat-delta desk-stat-delta--down'
    : trend === 'neutral' ? 'desk-stat-delta desk-stat-delta--neutral'
    : '';

  return (
    <motion.div
      initial={{ opacity: 0, y: 8 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      transition={{ duration: 0.35, delay: delay / 1000, ease: [0.16, 1, 0.3, 1] }}
      className={`desk-kpi-card ${className}`}
      style={{ position: 'relative' }}
    >
      <div className="desk-kpi-card-meta">
        <span>{label}</span>
        {trendIcon && <span className={deltaClass}>{trendIcon}</span>}
      </div>
      <div className="desk-kpi-card-value">
        <CountUp value={value} delay={delay / 1000 + 0.2} className="tabular-nums" />
      </div>
      {sub && <div className="desk-kpi-card-meta" style={{ marginTop: 4 }}>{sub}</div>}
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

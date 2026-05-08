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
  const trendIcon = trend === 'up' ? '↑' : trend === 'down' ? '↓' : null;
  const trendColor = trend === 'up'
    ? 'var(--success)'
    : trend === 'down'
    ? 'var(--danger)'
    : 'var(--muted)';

  return (
    <motion.div
      initial={{ opacity: 0, y: 12 }}
      whileInView={{ opacity: 1, y: 0 }}
      viewport={{ once: true }}
      whileHover={{ y: -6, transition: { duration: 0.2 } }}
      transition={{ 
        duration: 0.5, 
        delay: delay / 1000, 
        ease: [0.16, 1, 0.3, 1] 
      }}
      className={`md-card glass-premium p-6 flex flex-col justify-between cursor-default overflow-hidden relative shadow-premium bg-surface/40 backdrop-blur-md border-white/5 ${className}`}
    >
      {accent && (
        <div
          className="absolute top-0 left-0 w-full h-1 opacity-50"
          style={{ background: accent }}
          aria-hidden="true"
        />
      )}
      <div className="flex justify-between items-start mb-4 relative z-10">
        <p className="md-typescale-label-small uppercase tracking-wider font-semibold opacity-70">
          {label}
        </p>
        {trendIcon && (
          <div 
            className="flex items-center gap-0.5 px-2 py-0.5 rounded-full md-typescale-label-small font-bold"
            style={{ backgroundColor: `${trendColor}20`, color: trendColor }}
          >
            {trendIcon}
          </div>
        )}
      </div>
      <div className="relative z-10">
        <div className="flex items-baseline gap-2">
          <CountUp
            value={value}
            delay={delay / 1000 + 0.3}
            className="text-3xl font-bold tracking-tight text-foreground tabular-nums"
          />
        </div>
        {sub && (
          <p className="md-typescale-label-small mt-1 opacity-60">
            {sub}
          </p>
        )}
      </div>
      
      {/* Subtle background glow */}
      <div 
        className="absolute -bottom-10 -right-10 w-32 h-32 blur-3xl opacity-10 pointer-events-none"
        style={{ background: accent || 'var(--color-md-primary)' }}
      />
    </motion.div>
  );
}

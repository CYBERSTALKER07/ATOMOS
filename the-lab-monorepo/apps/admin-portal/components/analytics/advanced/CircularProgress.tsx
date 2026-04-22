'use client';

interface CircularProgressProps {
  value: number;       // 0-100
  size?: number;       // px, default 80
  strokeWidth?: number;
  label?: string;
  color?: string;
}

export default function CircularProgress({
  value,
  size = 80,
  strokeWidth = 6,
  label,
  color = 'var(--color-md-primary)',
}: CircularProgressProps) {
  const radius = (size - strokeWidth) / 2;
  const circumference = 2 * Math.PI * radius;
  const offset = circumference - (Math.min(Math.max(value, 0), 100) / 100) * circumference;

  return (
    <div className="flex flex-col items-center gap-1">
      <svg width={size} height={size} className="transform -rotate-90">
        {/* Track */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke="var(--color-md-surface-container-highest)"
          strokeWidth={strokeWidth}
        />
        {/* Progress */}
        <circle
          cx={size / 2}
          cy={size / 2}
          r={radius}
          fill="none"
          stroke={color}
          strokeWidth={strokeWidth}
          strokeLinecap="round"
          strokeDasharray={circumference}
          strokeDashoffset={offset}
          style={{ transition: 'stroke-dashoffset 0.8s cubic-bezier(0.4, 0, 0.2, 1)' }}
        />
      </svg>
      <span className="md-typescale-title-medium font-semibold" style={{ color, marginTop: -size / 2 - 8, position: 'relative' }}>
        {Math.round(value)}%
      </span>
      {label && (
        <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)', marginTop: size / 2 - 14 }}>
          {label}
        </span>
      )}
    </div>
  );
}

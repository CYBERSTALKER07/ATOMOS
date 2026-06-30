'use client';

interface VuCapacityBarProps {
  used: number;
  max: number;
  label?: string;
  compact?: boolean;
}

export default function VuCapacityBar({ used, max, label, compact }: VuCapacityBarProps) {
  const safeMax = Math.max(max, 1);
  const pct = Math.min(100, (used / safeMax) * 100);
  const tone = pct >= 90 ? 'var(--danger)' : pct >= 70 ? 'var(--warning)' : 'var(--success)';

  return (
    <div className={compact ? 'space-y-0.5' : 'space-y-1'}>
      {label && (
        <div className="flex items-center justify-between text-xs text-(--muted)">
          <span>{label}</span>
          <span className="font-mono tabular-nums">{used.toFixed(0)} / {max.toFixed(0)} VU</span>
        </div>
      )}
      <div
        className="h-2 rounded-full overflow-hidden"
        style={{ background: 'color-mix(in srgb, var(--border) 60%, transparent)' }}
      >
        <div
          className="h-full rounded-full transition-all"
          style={{ width: `${pct}%`, background: tone }}
        />
      </div>
      {!label && (
        <div className="text-[10px] text-right font-mono tabular-nums text-(--muted)">
          {pct.toFixed(0)}%
        </div>
      )}
    </div>
  );
}

'use client';

import { type DateRangePreset, type DateRange } from '@/hooks/useAdvancedAnalytics';

const PRESETS: { label: string; value: DateRangePreset }[] = [
  { label: '7D', value: '7d' },
  { label: '14D', value: '14d' },
  { label: '30D', value: '30d' },
  { label: '90D', value: '90d' },
  { label: '180D', value: '180d' },
  { label: '1Y', value: '365d' },
];

interface DateRangePickerProps {
  dateRange: DateRange;
  onPreset: (p: DateRangePreset) => void;
  onCustom: (dr: DateRange) => void;
}

export default function DateRangePicker({ dateRange, onPreset, onCustom }: DateRangePickerProps) {
  return (
    <div className="flex items-center gap-2 flex-wrap">
      {/* Preset chips */}
      <div className="flex gap-1">
        {PRESETS.map((p) => (
          <button
            key={p.value}
            onClick={() => onPreset(p.value)}
            className={`md-typescale-label-medium px-3 py-1.5 transition-colors ${
              dateRange.preset === p.value
                ? 'md-btn md-btn-filled'
                : 'md-btn md-btn-outlined'
            }`}
            style={dateRange.preset === p.value ? {} : { color: 'var(--color-md-on-surface-variant)' }}
          >
            {p.label}
          </button>
        ))}
      </div>

      {/* Custom date inputs */}
      <div className="flex items-center gap-1.5 ml-2">
        <input
          type="date"
          value={dateRange.from}
          onChange={(e) =>
            onCustom({ from: e.target.value, to: dateRange.to, preset: 'custom' })
          }
          className="md-input-outlined md-typescale-label-medium px-2 py-1"
          style={{
            background: 'var(--color-md-surface-container)',
            color: 'var(--color-md-on-surface)',
            border: '1px solid var(--color-md-outline-variant)',
          }}
        />
        <span className="md-typescale-label-small" style={{ color: 'var(--color-md-on-surface-variant)' }}>→</span>
        <input
          type="date"
          value={dateRange.to}
          onChange={(e) =>
            onCustom({ from: dateRange.from, to: e.target.value, preset: 'custom' })
          }
          className="md-input-outlined md-typescale-label-medium px-2 py-1"
          style={{
            background: 'var(--color-md-surface-container)',
            color: 'var(--color-md-on-surface)',
            border: '1px solid var(--color-md-outline-variant)',
          }}
        />
      </div>
    </div>
  );
}

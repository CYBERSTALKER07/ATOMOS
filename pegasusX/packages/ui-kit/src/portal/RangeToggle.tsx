"use client";

import type { DashboardHistoryRange } from "@pegasusx/types";

export type RangeToggleProps = {
  value: DashboardHistoryRange;
  onChange: (range: DashboardHistoryRange) => void;
  className?: string;
};

const RANGES: DashboardHistoryRange[] = ["today", "7d", "30d"];

/** History range only. Now KPIs stay today. */
export function RangeToggle({ value, onChange, className }: RangeToggleProps) {
  return (
    <div className={className} data-testid="gs-u-range-toggle" role="group" aria-label="History range">
      {RANGES.map((range) => (
        <button
          key={range}
          type="button"
          data-range={range}
          aria-pressed={value === range}
          onClick={() => onChange(range)}
          style={{
            marginRight: 6,
            padding: "4px 10px",
            borderRadius: 8,
            border: "1px solid var(--desk-border, #e5e7eb)",
            background: value === range ? "var(--desk-surface-raised, #fff)" : "transparent",
            color: "inherit",
            cursor: "pointer",
            fontSize: "0.75rem",
          }}
        >
          {range}
        </button>
      ))}
    </div>
  );
}

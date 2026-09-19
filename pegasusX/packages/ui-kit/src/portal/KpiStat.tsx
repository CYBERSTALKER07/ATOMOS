"use client";

import type { HistorySeries } from "@pegasusx/types";
import { guardHistorySeries } from "@pegasusx/types";
import type { ReactNode } from "react";
import { SourceChip } from "./SourceChip";

export type KpiStatProps = {
  label: string;
  value: ReactNode;
  delta?: ReactNode;
  spark?: HistorySeries | null;
  source: string;
  className?: string;
};

function Spark({ points }: { points: number[] }) {
  const max = Math.max(...points);
  const min = Math.min(...points);
  const range = max - min || 1;
  const d = points
    .map((n, i) => {
      const x = (i / (points.length - 1)) * 100;
      const y = 100 - ((n - min) / range) * 100;
      return `${x},${y}`;
    })
    .join(" ");
  return (
    <svg data-testid="gs-u-kpi-spark" viewBox="0 0 100 100" width={48} height={24} aria-hidden>
      <polyline points={d} fill="none" stroke="currentColor" strokeWidth="6" />
    </svg>
  );
}

/** KPI number. Spark renders only when HistorySeries is live. */
export function KpiStat({ label, value, delta, spark, source, className }: KpiStatProps) {
  const live = guardHistorySeries(spark);
  return (
    <div className={className} data-testid="gs-u-kpi-stat">
      <div style={{ display: "flex", justifyContent: "space-between", gap: 8, marginBottom: 6 }}>
        <span style={{ fontSize: "0.75rem", color: "var(--desk-text-secondary, #6b7280)" }}>{label}</span>
        <SourceChip source={source} />
      </div>
      <div style={{ display: "flex", alignItems: "flex-end", justifyContent: "space-between", gap: 8 }}>
        <div>
          <div style={{ fontSize: "1.5rem", fontVariantNumeric: "tabular-nums" }}>{value}</div>
          {delta ? (
            <div style={{ fontSize: "0.75rem", color: "var(--desk-text-secondary, #6b7280)" }}>{delta}</div>
          ) : null}
        </div>
        {live ? <Spark points={live.points} /> : null}
      </div>
    </div>
  );
}

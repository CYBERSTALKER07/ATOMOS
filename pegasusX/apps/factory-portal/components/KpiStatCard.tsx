"use client";

import type { ReactNode } from "react";

type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  className?: string;
  align?: "start" | "center";
};

export function KpiStatCard({ label, value, sub, className = "", align = "start" }: KpiStatCardProps) {
  return (
    <div
      className={`desk-card p-5 flex flex-col justify-between gap-3 ${align === "center" ? "items-center text-center" : ""} ${className}`.trim()}
    >
      <p className="md-kpi-label" style={{ color: "var(--desk-text-secondary)" }}>
        {label}
      </p>
      <div className={align === "center" ? "w-full" : undefined}>
        <div className="md-kpi-value" style={{ color: "var(--desk-text-primary)" }}>
          {value}
        </div>
        {sub ? (
          <p className="md-kpi-sub mt-1" style={{ color: "var(--desk-text-tertiary)" }}>
            {sub}
          </p>
        ) : null}
      </div>
    </div>
  );
}

export function KpiStatGrid({ children, columns = 3 }: { children: ReactNode; columns?: 2 | 3 | 4 }) {
  const colClass =
    columns === 4
      ? "grid-cols-1 sm:grid-cols-2 xl:grid-cols-4"
      : columns === 2
        ? "grid-cols-1 sm:grid-cols-2"
        : "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3";
  return <div className={`grid ${colClass} gap-4`}>{children}</div>;
}

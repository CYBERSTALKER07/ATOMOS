"use client";

import type { ReactNode } from "react";

type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  className?: string;
  align?: "start" | "center";
  icon?: ReactNode;
};

export function KpiStatCard({
  label,
  value,
  sub,
  className = "",
  align = "start",
  icon,
}: KpiStatCardProps) {
  return (
    <div
      className={`desk-card p-5 flex flex-col justify-between gap-3 border border-[var(--desk-border)] rounded-2xl ${align === "center" ? "items-center text-center" : ""} ${className}`.trim()}
    >
      <div className="flex items-center justify-between gap-2 w-full">
        <p className="text-[11px] font-mono font-bold uppercase tracking-wider text-[var(--desk-text-secondary)]">
          {label}
        </p>
        {icon ? (
          <span className="shrink-0 w-9 h-9 rounded-xl flex items-center justify-center bg-[var(--desk-surface-subtle)] text-[var(--desk-text-primary)]">
            {icon}
          </span>
        ) : null}
      </div>
      <div className={align === "center" ? "w-full" : undefined}>
        <div className="font-mono text-2xl font-bold tracking-tight text-[var(--desk-text-primary)]">
          {value}
        </div>
        {sub ? (
          <p className="text-xs font-mono text-[var(--desk-text-tertiary)] mt-1">
            {sub}
          </p>
        ) : null}
      </div>
    </div>
  );
}

export function KpiStatGrid({ children, columns = 4 }: { children: ReactNode; columns?: 2 | 3 | 4 }) {
  const colClass =
    columns === 4
      ? "grid-cols-1 sm:grid-cols-2 xl:grid-cols-4"
      : columns === 2
        ? "grid-cols-1 sm:grid-cols-2"
        : "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3";
  return <div className={`grid ${colClass} gap-4`}>{children}</div>;
}

"use client";

import Link from "next/link";
import type { ReactNode } from "react";
import Icon from "./Icon";

type KpiStatCardProps = {
  label: string;
  value: ReactNode;
  sub?: ReactNode;
  icon?: string;
  href?: string;
  flag?: "alert" | "ok";
  bay?: "ops" | "inventory" | "fleet" | "finance";
  className?: string;
  align?: "start" | "center";
  sparkline?: number[];
};

function Sparkline({ data }: { data: number[] }) {
  if (!data || data.length < 2) return null;
  const max = Math.max(...data);
  const min = Math.min(...data);
  const range = max - min || 1;
  const points = data.map((d, i) => {
    const x = (i / (data.length - 1)) * 100;
    const y = 100 - ((d - min) / range) * 100;
    return `${x},${y}`;
  }).join(' ');

  return (
    <svg className="h-8 w-16 opacity-50" viewBox="0 -10 100 120" preserveAspectRatio="none">
      <polyline points={points} fill="none" stroke="currentColor" strokeWidth="6" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

const BAY_CLASS: Record<NonNullable<KpiStatCardProps["bay"]>, string> = {
  ops: "wh-bay--ops",
  inventory: "wh-bay--inventory",
  fleet: "wh-bay--fleet",
  finance: "wh-bay--finance",
};

export function KpiStatCard({
  label,
  value,
  sub,
  icon,
  href,
  flag,
  bay = "ops",
  className = "",
  align = "start",
  sparkline,
}: KpiStatCardProps) {
  const body = (
    <>
      <div className={`flex items-center justify-between gap-2 ${align === "center" ? "w-full" : ""}`}>
        {icon ? (
          <div className="wh-kpi-icon" aria-hidden>
            <Icon name={icon} size={18} />
          </div>
        ) : (
          <span />
        )}
        {flag === "alert" ? <span className="wh-kpi-flag wh-kpi-flag--alert">Alert</span> : null}
        {flag === "ok" ? <span className="wh-kpi-flag wh-kpi-flag--ok">Done</span> : null}
      </div>
      </div>
      <div className={`mt-2 flex items-end justify-between ${align === "center" ? "w-full flex-col items-center" : ""}`}>
        <div>
          <p className="wh-kpi-label">{label}</p>
          <div className="wh-kpi-value mt-1">{value}</div>
          {sub ? (
            <p className="mt-1 text-xs" style={{ color: "var(--wh-ink-faint)" }}>
              {sub}
            </p>
          ) : null}
        </div>
        {sparkline && (
          <div className="text-(--accent) shrink-0 ml-2">
            <Sparkline data={sparkline} />
          </div>
        )}
      </div>
    </>
  );

  const classes = `wh-bay-panel wh-kpi-card ${BAY_CLASS[bay]} ${align === "center" ? "items-center text-center" : ""} ${className}`.trim();

  if (href) {
    return (
      <Link href={href} className={classes}>
        {body}
      </Link>
    );
  }

  return <div className={classes}>{body}</div>;
}

export function KpiStatGrid({ children, columns = 3 }: { children: ReactNode; columns?: 2 | 3 | 4 }) {
  const colClass =
    columns === 4
      ? "grid-cols-1 sm:grid-cols-2 xl:grid-cols-4"
      : columns === 2
        ? "grid-cols-1 sm:grid-cols-2"
        : "grid-cols-1 sm:grid-cols-2 lg:grid-cols-3";
  return <div className={`wh-kpi-grid grid ${colClass} gap-4`}>{children}</div>;
}

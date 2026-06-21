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
};

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
      <div className={align === "center" ? "w-full" : undefined}>
        <p className="wh-kpi-label">{label}</p>
        <div className="wh-kpi-value mt-2">{value}</div>
        {sub ? (
          <p className="text-xs mt-1" style={{ color: "var(--wh-ink-faint)" }}>
            {sub}
          </p>
        ) : null}
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

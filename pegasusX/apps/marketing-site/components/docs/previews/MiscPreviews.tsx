"use client";

import { ORDER_LIFECYCLE } from "@/lib/constants";

export function StatusChipPreview() {
  return (
    <div className="flex flex-wrap gap-2">
      {ORDER_LIFECYCLE.map((status, i) => (
        <span
          key={status}
          className={`status-chip ${i === ORDER_LIFECYCLE.length - 1 ? "status-chip--filled" : ""}`}
        >
          {status.replace("_", " ")}
        </span>
      ))}
    </div>
  );
}

export function KpiStatCardPreview() {
  return (
    <div className="kpi-stat-card">
      <p className="kpi-stat-card__value">1,284</p>
      <p className="kpi-stat-card__label">Active pre-orders</p>
    </div>
  );
}

export function RoleBadgePreview() {
  return (
    <div className="flex flex-wrap gap-2">
      {["Supplier", "Warehouse", "Driver"].map((label, i) => (
        <span key={label} className="role-badge">
          <span className={`role-badge__dot ${i === 1 ? "role-badge__dot--outline" : ""}`} />
          {label}
        </span>
      ))}
    </div>
  );
}

export function ScrollSectionPreview() {
  return (
    <div className="mkt-card p-4 text-sm text-[var(--mkt-muted)]">
      PinSection + usePinSection — scroll-linked pin/scrub for landing sections.
    </div>
  );
}

export function PortalCardPreview() {
  return (
    <div className="grid gap-3 sm:grid-cols-2">
      <div className="desk-card p-4">
        <p className="font-mono text-[10px] uppercase text-[var(--mkt-subtle)]">Stat</p>
        <p className="text-2xl font-light">42</p>
      </div>
      <div className="desk-card p-4">
        <p className="font-semibold">Control card</p>
        <p className="mt-1 text-xs text-[var(--mkt-muted)]">Bento grid anchor size</p>
      </div>
    </div>
  );
}

export function TopologyGraphPreview() {
  return (
    <svg viewBox="0 0 320 160" className="w-full" aria-hidden>
      {[
        [40, 80, 120, 40],
        [120, 40, 200, 80],
        [200, 80, 280, 40],
        [120, 40, 120, 120],
      ].map(([x1, y1, x2, y2], i) => (
        <line key={i} x1={x1} y1={y1} x2={x2} y2={y2} stroke="var(--mkt-border-strong)" strokeWidth="1.5" />
      ))}
      {[
        [40, 80],
        [120, 40],
        [200, 80],
        [280, 40],
        [120, 120],
      ].map(([cx, cy], i) => (
        <circle key={i} cx={cx} cy={cy} r="10" fill={i === 0 ? "var(--mkt-text)" : "var(--mkt-surface)"} stroke="var(--mkt-text)" strokeWidth="1.5" />
      ))}
    </svg>
  );
}

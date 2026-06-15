"use client";

import type { CSSProperties } from "react";

export function Skeleton({ className = "", style }: { className?: string; style?: CSSProperties }) {
  return <div className={`skeleton ${className}`.trim()} style={style} aria-hidden />;
}

export function PageSkeleton({
  variant = "dashboard",
}: {
  variant?: "dashboard" | "table" | "form";
}) {
  if (variant === "form") {
    return (
      <div className="page-skeleton" style={{ maxWidth: 640, margin: "0 auto" }}>
        <Skeleton style={{ height: 32, width: 200, borderRadius: 8 }} />
        <Skeleton style={{ height: 14, width: "60%", borderRadius: 6 }} />
        <div className="flex flex-col gap-4 mt-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="flex flex-col gap-2">
              <Skeleton style={{ height: 12, width: 100, borderRadius: 4 }} />
              <Skeleton style={{ height: 48, borderRadius: 8 }} />
            </div>
          ))}
        </div>
      </div>
    );
  }

  if (variant === "table") {
    return (
      <div className="page-skeleton">
        <Skeleton className="skeleton-header" />
        <div className="skeleton-kpi-grid">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="skeleton-kpi" />
          ))}
        </div>
        <Skeleton className="skeleton-table" />
      </div>
    );
  }

  return (
    <div className="page-skeleton">
      <Skeleton className="skeleton-header" />
      <div className="skeleton-kpi-grid">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="skeleton-kpi" />
        ))}
      </div>
      <Skeleton style={{ height: 280, borderRadius: 12 }} />
    </div>
  );
}

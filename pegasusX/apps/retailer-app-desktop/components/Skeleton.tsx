"use client";

import type { CSSProperties } from "react";

export function Skeleton({ className = "", style }: { className?: string; style?: CSSProperties }) {
  return <div className={`skeleton ${className}`.trim()} style={style} aria-hidden />;
}

export function PageSkeleton({
  variant = "dashboard",
}: {
  variant?: "dashboard" | "table" | "form" | "catalog";
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

  if (variant === "table" || variant === "catalog") {
    return (
      <div className="page-skeleton">
        <Skeleton className="skeleton-header" />
        <div className="skeleton-kpi-grid">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="skeleton-kpi" />
          ))}
        </div>
        <div
          className={
            variant === "catalog"
              ? "grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6"
              : undefined
          }
        >
          {variant === "catalog"
            ? Array.from({ length: 6 }).map((_, i) => (
                <Skeleton key={i} style={{ height: 288, borderRadius: 16 }} />
              ))
            : <Skeleton className="skeleton-table" />}
        </div>
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

export function ListRowSkeleton({ count = 4 }: { count?: number }) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <Skeleton key={i} style={{ height: 96, borderRadius: 16 }} />
      ))}
    </>
  );
}

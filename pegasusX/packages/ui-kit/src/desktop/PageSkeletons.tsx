"use client";

import type { CSSProperties } from "react";

function Block({ className = "", style }: { className?: string; style?: CSSProperties }) {
  return (
    <div
      className={`animate-pulse rounded-lg bg-[var(--desk-surface-subtle,var(--color-md-surface-container-high,#e8eaed))] ${className}`.trim()}
      style={style}
      aria-hidden
    />
  );
}

export function PageSkeleton({
  variant = "dashboard",
}: {
  variant?: "dashboard" | "table" | "form" | "catalog";
}) {
  if (variant === "form") {
    return (
      <div className="flex max-w-[640px] flex-col gap-4 mx-auto">
        <Block style={{ height: 32, width: 200 }} />
        <Block style={{ height: 14, width: "60%" }} />
        {Array.from({ length: 4 }).map((_, i) => (
          <div key={i} className="flex flex-col gap-2">
            <Block style={{ height: 12, width: 100 }} />
            <Block style={{ height: 48 }} />
          </div>
        ))}
      </div>
    );
  }

  if (variant === "table" || variant === "catalog") {
    return (
      <div className="flex flex-col gap-4">
        <Block style={{ height: 28, width: 220 }} />
        <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <Block key={i} style={{ height: 88 }} />
          ))}
        </div>
        {variant === "catalog" ? (
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
            {Array.from({ length: 6 }).map((_, i) => (
              <Block key={i} style={{ height: 288 }} />
            ))}
          </div>
        ) : (
          <Block style={{ height: 320 }} />
        )}
      </div>
    );
  }

  return (
    <div className="flex flex-col gap-4">
      <Block style={{ height: 28, width: 220 }} />
      <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Block key={i} style={{ height: 88 }} />
        ))}
      </div>
      <Block style={{ height: 280 }} />
    </div>
  );
}

export function BentoGridSkeleton({ count = 4 }: { count?: number }) {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: count }).map((_, i) => (
        <Block key={i} style={{ height: 120 }} />
      ))}
    </div>
  );
}

export function ListRowSkeleton({ count = 4 }: { count?: number }) {
  return (
    <>
      {Array.from({ length: count }).map((_, i) => (
        <Block key={i} className="mb-2" style={{ height: 96 }} />
      ))}
    </>
  );
}

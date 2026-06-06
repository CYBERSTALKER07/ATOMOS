"use client";

import type { ReactNode } from "react";

type BentoSize = "stat" | "anchor" | "list" | "control" | "wide" | "full";

interface BentoGridProps {
  children: ReactNode;
  className?: string;
  theme?: "brutalist" | "apple";
}

export function BentoGrid({
  children,
  className = "",
  theme = "apple",
}: BentoGridProps) {
  const themeClass = theme === "apple" ? "bento-apple" : "";
  return <div className={`bento-grid ${themeClass} ${className}`.trim()}>{children}</div>;
}

interface BentoCardProps {
  children: ReactNode;
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}

export function BentoCard({
  children,
  size,
  span = 1,
  rowSpan = false,
  className = "",
}: BentoCardProps) {
  let sizeClass: string;
  if (size) {
    sizeClass = `bento-${size}`;
  } else {
    sizeClass = `bento-span-${span}${rowSpan ? " bento-row-2" : ""}`;
  }

  return (
    <div className={`bento-card ${sizeClass} ${className}`.trim()}>
      <div className="relative h-full w-full overflow-hidden rounded-[inherit]">
        {children}
      </div>
    </div>
  );
}

interface BentoSkeletonProps {
  size?: BentoSize;
  span?: 1 | 2 | 3 | 4;
  rowSpan?: boolean;
  className?: string;
}

export function BentoSkeleton({
  size,
  span = 1,
  rowSpan = false,
  className = "",
}: BentoSkeletonProps) {
  let sizeClass: string;
  if (size) {
    sizeClass = `bento-${size}`;
  } else {
    sizeClass = `bento-span-${span}${rowSpan ? " bento-row-2" : ""}`;
  }

  return (
    <div
      className={`bento-skeleton ${sizeClass} ${className}`.trim()}
      aria-hidden
    />
  );
}

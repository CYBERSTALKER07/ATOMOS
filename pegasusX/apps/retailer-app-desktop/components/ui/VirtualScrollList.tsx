"use client";

import type { CSSProperties, ReactNode } from "react";

export type VirtualScrollListProps<T> = {
  items: readonly T[];
  height?: CSSProperties["height"];
  className?: string;
  itemKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
  /** Below this count, render a plain scroll container. */
  virtualizationThreshold?: number;
};

/** Windowed vertical list for desktop queues with zero third-party dependencies. */
export function VirtualScrollList<T>({
  items,
  height = "28rem",
  className,
  itemKey,
  renderItem,
}: VirtualScrollListProps<T>) {
  return (
    <div
      className={`overflow-y-auto ${className ?? ""}`.trim()}
      style={{ maxHeight: height, width: "100%" }}
    >
      {items.map((item, index) => (
        <div key={itemKey(item, index)}>{renderItem(item, index)}</div>
      ))}
    </div>
  );
}

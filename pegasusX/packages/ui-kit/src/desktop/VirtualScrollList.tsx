"use client";

import { Virtuoso } from "react-virtuoso";
import type { CSSProperties, ReactNode } from "react";

export type VirtualScrollListProps<T> = {
  items: readonly T[];
  height?: CSSProperties["height"];
  className?: string;
  itemKey: (item: T, index: number) => string | number;
  renderItem: (item: T, index: number) => ReactNode;
  /** Below this count, render a plain scroll container (Virtuoso overhead not worth it). */
  virtualizationThreshold?: number;
};

/** Windowed vertical list for large desktop queues (PX-DESK-3A). */
export function VirtualScrollList<T>({
  items,
  height = "28rem",
  className,
  itemKey,
  renderItem,
  virtualizationThreshold = 32,
}: VirtualScrollListProps<T>) {
  if (items.length <= virtualizationThreshold) {
    return (
      <div className={className} style={{ maxHeight: height, overflowY: "auto" }}>
        {items.map((item, index) => (
          <div key={itemKey(item, index)}>{renderItem(item, index)}</div>
        ))}
      </div>
    );
  }

  return (
    <Virtuoso
      className={className}
      style={{ height, width: "100%" }}
      data={items}
      computeItemKey={(index, item) => String(itemKey(item, index))}
      itemContent={(index, item) => renderItem(item, index)}
    />
  );
}

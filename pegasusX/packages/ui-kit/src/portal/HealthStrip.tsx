"use client";

export type HealthStripItem = {
  key: string;
  label: string;
  count: number;
  href?: string;
};

export type HealthStripProps = {
  items: HealthStripItem[];
  onSelect?: (key: string) => void;
  className?: string;
};

/** Compact exception / fiscal / shop-closed counts. Zero is honest, not hidden. */
export function HealthStrip({ items, onSelect, className }: HealthStripProps) {
  return (
    <div className={className} data-testid="gs-u-health-strip">
      {items.map((item) => (
        <button
          key={item.key}
          type="button"
          data-health={item.key}
          onClick={onSelect ? () => onSelect(item.key) : undefined}
          disabled={!onSelect}
          style={{
            display: "flex",
            justifyContent: "space-between",
            gap: 12,
            width: "100%",
            padding: "8px 0",
            border: "none",
            borderBottom: "1px solid var(--desk-border, #e5e7eb)",
            background: "transparent",
            color: "inherit",
            cursor: onSelect ? "pointer" : "default",
          }}
        >
          <span style={{ fontSize: "0.8125rem" }}>{item.label}</span>
          <span style={{ fontVariantNumeric: "tabular-nums", fontSize: "0.8125rem" }}>{item.count}</span>
        </button>
      ))}
    </div>
  );
}

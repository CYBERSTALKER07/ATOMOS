"use client";

import type { CSSProperties } from "react";

export type DemandSourceCode = "STORE_POS" | "WHOLESALE_HISTORY" | string;

const LABELS: Record<string, string> = {
  STORE_POS: "Store POS",
  WHOLESALE_HISTORY: "Wholesale",
};

export function demandSourceLabel(code: string): string {
  return LABELS[code] ?? code;
}

export function normalizeDemandSources(sources?: string[] | null): string[] {
  if (sources && sources.length > 0) return sources;
  return ["WHOLESALE_HISTORY"];
}

export type DemandSourceChipsProps = {
  sources?: string[] | null;
  className?: string;
  /** Compact pills for tables */
  size?: "sm" | "md";
};

/**
 * Enterprise demand-source chips for reorder / sell-through surfaces.
 * STORE_POS = floor sales velocity; WHOLESALE_HISTORY = B2B demand sensing.
 */
export function DemandSourceChips({
  sources,
  className,
  size = "sm",
}: DemandSourceChipsProps) {
  const list = normalizeDemandSources(sources);
  const pad = size === "sm" ? "0.125rem 0.5rem" : "0.25rem 0.625rem";
  const fontSize = size === "sm" ? "0.625rem" : "0.6875rem";

  return (
    <span
      className={className}
      style={{ display: "inline-flex", flexWrap: "wrap", gap: "0.25rem" }}
      role="list"
      aria-label={`Demand sources: ${list.map(demandSourceLabel).join(", ")}`}
    >
      {list.map((code) => {
        const isPos = code === "STORE_POS";
        const style: CSSProperties = {
          display: "inline-flex",
          alignItems: "center",
          borderRadius: 9999,
          padding: pad,
          fontSize,
          fontWeight: 600,
          letterSpacing: "0.04em",
          textTransform: "uppercase",
          background: isPos
            ? "color-mix(in srgb, var(--desk-success, #16a34a) 18%, transparent)"
            : "var(--desk-surface-muted, #f3f4f6)",
          color: isPos
            ? "var(--desk-success, #15803d)"
            : "var(--desk-text-secondary, #6b7280)",
          border: `1px solid ${
            isPos
              ? "color-mix(in srgb, var(--desk-success, #16a34a) 40%, transparent)"
              : "var(--desk-border, #e5e7eb)"
          }`,
        };
        return (
          <span
            key={code}
            role="listitem"
            style={style}
            aria-label={`Demand source: ${demandSourceLabel(code)}`}
          >
            {demandSourceLabel(code)}
          </span>
        );
      })}
    </span>
  );
}

"use client";

import { statusStackModel, type StatusStackMode } from "@pegasusx/types";
import { SourceChip } from "./SourceChip";

export type StatusStackProps = {
  dictionary: readonly string[];
  counts?: Record<string, number> | null;
  available?: boolean;
  source?: string;
  onSelect?: (key: string) => void;
  labelFor?: (key: string) => string;
  className?: string;
};

function defaultLabel(key: string): string {
  return key.replaceAll("_", " ");
}

function sourceForMode(mode: StatusStackMode, source?: string): string {
  if (source) return source;
  if (mode === "unavailable") return "unavailable";
  if (mode === "empty") return "empty";
  return "live";
}

/** Stacked bar + chip rail. Every dictionary key stays visible on zero/live. */
export function StatusStack({
  dictionary,
  counts,
  available = true,
  source,
  onSelect,
  labelFor = defaultLabel,
  className,
}: StatusStackProps) {
  const model = statusStackModel(dictionary, counts, available);
  const chipSource = sourceForMode(model.mode, source);

  return (
    <div className={className} data-testid="gs-u-status-stack" data-mode={model.mode}>
      <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: 8 }}>
        <SourceChip source={chipSource} />
      </div>
      {model.mode === "empty" ? (
        <p
          data-testid="gs-u-status-stack-empty"
          style={{ color: "var(--desk-text-secondary, #6b7280)", fontSize: "0.8125rem", margin: 0 }}
        >
          No status counts
        </p>
      ) : null}
      {model.mode === "unavailable" ? (
        <p
          data-testid="gs-u-status-stack-unavailable"
          style={{ color: "var(--desk-text-secondary, #6b7280)", fontSize: "0.8125rem", margin: 0 }}
        >
          Status counts unavailable
        </p>
      ) : null}
      {model.mode === "live" ? (
        <div
          data-testid="gs-u-status-stack-bar"
          role="img"
          aria-label="Order status mix"
          style={{
            display: "flex",
            height: 10,
            width: "100%",
            overflow: "hidden",
            borderRadius: 9999,
            background: "var(--desk-surface-muted, #f3f4f6)",
            marginBottom: 10,
          }}
        >
          {model.rows.map((row) =>
            row.share > 0 ? (
              <span
                key={row.key}
                title={`${labelFor(row.key)} ${row.count}`}
                style={{
                  width: `${row.share * 100}%`,
                  background: "var(--desk-accent, #111827)",
                  opacity: 0.35 + row.share * 0.65,
                }}
              />
            ) : null,
          )}
        </div>
      ) : null}
      {model.mode !== "empty" ? (
        <div
          data-testid="gs-u-status-stack-chips"
          style={{ display: "flex", flexWrap: "wrap", gap: 8 }}
        >
          {model.rows.map((row) => {
            const value = row.count == null ? "—" : String(row.count);
            return (
              <button
                key={row.key}
                type="button"
                data-status={row.key}
                onClick={onSelect ? () => onSelect(row.key) : undefined}
                disabled={!onSelect}
                style={{
                  display: "inline-flex",
                  alignItems: "center",
                  gap: 8,
                  padding: "6px 10px",
                  borderRadius: 8,
                  border: "1px solid var(--desk-border, #e5e7eb)",
                  background: "var(--desk-surface-raised, #fff)",
                  color: "inherit",
                  cursor: onSelect ? "pointer" : "default",
                }}
              >
                <span style={{ fontSize: "0.6875rem", color: "var(--desk-text-secondary, #6b7280)" }}>
                  {labelFor(row.key)}
                </span>
                <span style={{ fontVariantNumeric: "tabular-nums", fontSize: "0.8125rem" }}>{value}</span>
              </button>
            );
          })}
        </div>
      ) : null}
    </div>
  );
}

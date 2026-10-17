"use client";

export type HonestySource =
  | "live"
  | "spanner"
  | "empty"
  | "unavailable"
  | "memory"
  | "env_default";

const LABELS: Record<HonestySource, string> = {
  live: "live",
  spanner: "spanner",
  empty: "empty",
  unavailable: "unavailable",
  memory: "memory",
  env_default: "env default",
};

export function sourceChipLabel(source: string): string {
  const key = String(source || "").trim().toLowerCase().replace(/-/g, "_") as HonestySource;
  return LABELS[key] ?? (String(source || "").trim() || "unknown");
}

export type SourceChipProps = {
  source: string;
  className?: string;
};

/** Honesty chip. Never decorate unavailable as zero. */
export function SourceChip({ source, className }: SourceChipProps) {
  const label = sourceChipLabel(source);
  const warn = label === "unavailable" || label === "memory" || label === "env default";
  const mute = label === "empty";
  return (
    <span
      className={className}
      data-testid="gs-u-source-chip"
      data-source={label}
      style={{
        display: "inline-flex",
        alignItems: "center",
        borderRadius: 9999,
        padding: "0.125rem 0.5rem",
        fontSize: "0.625rem",
        fontWeight: 600,
        letterSpacing: "0.04em",
        textTransform: "uppercase",
        border: "1px solid var(--desk-border, #e5e7eb)",
        background: warn
          ? "color-mix(in srgb, var(--desk-warning, #d97706) 14%, transparent)"
          : mute
            ? "var(--desk-surface-muted, #f3f4f6)"
            : "var(--desk-surface-raised, #fff)",
        color: warn
          ? "var(--desk-warning, #b45309)"
          : "var(--desk-text-secondary, #6b7280)",
      }}
    >
      {label}
    </span>
  );
}

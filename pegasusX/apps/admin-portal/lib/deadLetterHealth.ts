/** GS-U8: dead-letter KPI is COUNT(*), never len(page). Unavailable ≠ zero. */

export type DeadLetterSummary = {
  dead_letter_count?: number;
  dead_letter_available?: boolean;
  available?: boolean;
  page_count?: number;
  count?: number;
  items?: unknown[];
};

export type DeadLetterHealth =
  | { kind: "unavailable" }
  | { kind: "zero" }
  | { kind: "count"; count: number };

export function deadLetterHealth(summary: DeadLetterSummary | null | undefined): DeadLetterHealth {
  if (!summary || summary.dead_letter_available !== true || typeof summary.dead_letter_count !== "number") {
    return { kind: "unavailable" };
  }
  if (summary.dead_letter_count === 0) return { kind: "zero" };
  return { kind: "count", count: summary.dead_letter_count };
}

export function deadLetterLabel(h: DeadLetterHealth): string {
  if (h.kind === "unavailable") return "unavailable";
  if (h.kind === "zero") return "empty";
  return String(h.count);
}

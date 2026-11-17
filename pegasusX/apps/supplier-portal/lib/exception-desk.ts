import type { SupplierExceptionRow } from "@pegasusx/types";
import { formatGroupedInteger, minorToWholeTrunc } from "@/lib/money";

export const EXCEPTION_HANDLER_GROUPS = [
  "FISCAL_AND_EXPORT_BLOCKERS",
  "CASH_DISCREPANCY",
  "CLAIMS_REVIEW",
  "ROUTE_AND_CLOSED_DOOR",
  "CREDIT_AND_PROMISE",
  "FULFILLMENT_AND_CATALOG",
] as const;

export type ExceptionHandlerGroup = (typeof EXCEPTION_HANDLER_GROUPS)[number];

export type ExceptionSeverityTier = "CRITICAL" | "HIGH" | "MEDIUM" | "LOW";

export interface SupplierExceptionDeskRow extends SupplierExceptionRow {
  handler_group?: ExceptionHandlerGroup | string;
  severity_tier?: ExceptionSeverityTier | string;
  severity?: string;
  money_at_risk_minor?: number;
  display_name?: string;
  retailer_name?: string;
  tin?: string;
  phone?: string;
  posting_id?: string;
  document_key?: string;
  dedup_cluster_id?: string;
  attempt_id?: string;
  driver_id?: string;
}

export interface BatchResolveRequestItem {
  kind: string;
  id: string;
  action?: string;
  note?: string;
  merged_into_id?: string;
  onec_ref?: string;
  credit_note_id?: string;
}

export interface BatchResolveResultItem {
  kind: string;
  id: string;
  status: "resolved" | "failed" | string;
  error?: string;
}

export interface BatchResolveResponse {
  resolved: number;
  failed?: Array<{ kind: string; id: string; error: string }>;
  results?: BatchResolveResultItem[];
}

export interface HandlerMeta {
  id: ExceptionHandlerGroup;
  label: string;
  description: string;
}

export const HANDLER_META: Record<ExceptionHandlerGroup, HandlerMeta> = {
  FISCAL_AND_EXPORT_BLOCKERS: {
    id: "FISCAL_AND_EXPORT_BLOCKERS",
    label: "Fiscal & 1C Export Blockers",
    description: "Soliq receipt retries, dead-letter fiscal receipts, and 1C accounting export blockers.",
  },
  CASH_DISCREPANCY: {
    id: "CASH_DISCREPANCY",
    label: "Cash Discrepancy",
    description: "Driver shift cash bag variances awaiting acceptance or write-off.",
  },
  CLAIMS_REVIEW: {
    id: "CLAIMS_REVIEW",
    label: "Claims & Credit Notes",
    description: "OS&D damage/shortage claims and draft credit notes pending issue.",
  },
  ROUTE_AND_CLOSED_DOOR: {
    id: "ROUTE_AND_CLOSED_DOOR",
    label: "Route & Closed Door",
    description: "Shop-closed escalations, early route completion requests, and manifest gate blocks.",
  },
  CREDIT_AND_PROMISE: {
    id: "CREDIT_AND_PROMISE",
    label: "Credit & Promise",
    description: "Frozen credit lines, overdue limits, and breached payment promises.",
  },
  FULFILLMENT_AND_CATALOG: {
    id: "FULFILLMENT_AND_CATALOG",
    label: "Fulfillment & Catalog",
    description: "Duplicate TIN partner links, order mismatches, and catalog exceptions.",
  },
};

const VALID_HANDLER_SET = new Set<string>(EXCEPTION_HANDLER_GROUPS);

const SEVERITY_RANK: Record<ExceptionSeverityTier, number> = {
  CRITICAL: 4,
  HIGH: 3,
  MEDIUM: 2,
  LOW: 1,
};

/** Formats minor tiyin into whole so'm strictly without tiyin decimals. */
export function formatWholeSom(minor: number | null | undefined): string {
  return `${formatGroupedInteger(minorToWholeTrunc(minor ?? 0))} so'm`;
}

/** Converts user-entered whole so'm amount into wire minor tiyin (so'm * 100). */
export function wholeSomToMinor(input: string | number): number | null {
  const cleaned = typeof input === "number" ? input : Number(String(input).replace(/[\s,]+/g, "").trim());
  if (!Number.isFinite(cleaned) || cleaned < 0) return null;
  return Math.round(cleaned) * 100;
}

/** Maps any exception row deterministically to one of the 6 canonical handler groups. */
export function resolveHandlerGroup(row: Pick<SupplierExceptionDeskRow, "handler_group" | "kind">): ExceptionHandlerGroup {
  const explicit = (row.handler_group ?? "").toUpperCase().trim();
  if (VALID_HANDLER_SET.has(explicit)) {
    return explicit as ExceptionHandlerGroup;
  }
  const kind = (row.kind ?? "").toUpperCase().trim();
  if (
    kind.includes("FISCAL") ||
    kind.includes("EXPORT") ||
    kind.includes("ONEC") ||
    kind.includes("1C")
  ) {
    return "FISCAL_AND_EXPORT_BLOCKERS";
  }
  if (kind.includes("CASH") || kind.includes("RECONCILIATION")) {
    return "CASH_DISCREPANCY";
  }
  if (kind.includes("CLAIM") || kind.includes("CREDIT_NOTE") || kind.includes("OSND")) {
    return "CLAIMS_REVIEW";
  }
  if (
    kind.includes("SHOP_CLOSED") ||
    kind.includes("EARLY_COMPLETE") ||
    kind.includes("ROUTE") ||
    kind.includes("MANIFEST")
  ) {
    return "ROUTE_AND_CLOSED_DOOR";
  }
  if (
    kind.includes("CREDIT_FREEZE") ||
    kind.includes("CREDIT_HOLD") ||
    kind.includes("PROMISE") ||
    kind.includes("OVERDUE")
  ) {
    return "CREDIT_AND_PROMISE";
  }
  return "FULFILLMENT_AND_CATALOG";
}

export function resolveSeverityTier(
  row: Pick<SupplierExceptionDeskRow, "severity_tier" | "severity" | "kind" | "money_at_risk_minor">,
): ExceptionSeverityTier {
  const raw = (row.severity_tier ?? row.severity ?? "").toUpperCase().trim();
  if (raw === "CRITICAL" || raw === "HIGH" || raw === "MEDIUM" || raw === "LOW") {
    return raw;
  }
  const kind = (row.kind ?? "").toUpperCase();
  if (kind.includes("DEAD_LETTER") || kind.includes("FISCAL_FAILED") || kind.includes("EXPORT_REJECTED")) {
    return "CRITICAL";
  }
  if (kind.includes("CASH_DISCREPANCY") || kind.includes("CREDIT_FREEZE") || kind.includes("SHOP_CLOSED")) {
    return "HIGH";
  }
  if ((row.money_at_risk_minor ?? 0) >= 100_000_00) {
    return "HIGH";
  }
  return "MEDIUM";
}

export function sortExceptionsByRisk(rows: SupplierExceptionDeskRow[]): SupplierExceptionDeskRow[] {
  return [...rows].sort((a, b) => {
    const tierDiff = SEVERITY_RANK[resolveSeverityTier(b)] - SEVERITY_RANK[resolveSeverityTier(a)];
    if (tierDiff !== 0) return tierDiff;
    const riskDiff = (b.money_at_risk_minor ?? 0) - (a.money_at_risk_minor ?? 0);
    if (riskDiff !== 0) return riskDiff;
    const scoreDiff = (b.score ?? 0) - (a.score ?? 0);
    if (scoreDiff !== 0) return scoreDiff;
    return (b.updated_at || "").localeCompare(a.updated_at || "");
  });
}

export function sumMoneyAtRiskMinor(rows: SupplierExceptionDeskRow[]): number {
  return rows.reduce((acc, row) => acc + (row.money_at_risk_minor ?? 0), 0);
}

export function exceptionRowKey(row: Pick<SupplierExceptionDeskRow, "kind" | "order_id" | "posting_id">): string {
  return `${row.kind}:${row.order_id || row.posting_id || ""}`;
}

export function resolveCounterpartyLabel(row: SupplierExceptionDeskRow): {
  primaryName: string;
  secondaryId: string | null;
} {
  const name = (row.display_name || row.retailer_name || "").trim();
  const fallbackId = (row.retailer_id || row.driver_id || row.order_id || "").trim();
  if (name) {
    return {
      primaryName: name,
      secondaryId: fallbackId && fallbackId !== name ? fallbackId : null,
    };
  }
  return {
    primaryName: fallbackId || "Unknown entity",
    secondaryId: null,
  };
}

export function defaultBatchResolveItem(
  row: SupplierExceptionDeskRow,
  note?: string,
): BatchResolveRequestItem {
  const kind = (row.kind ?? "").toUpperCase();
  const id = row.order_id || row.posting_id || row.document_key || row.manifest_id || "";
  const cleanNote = note?.trim() || undefined;
  switch (kind) {
    case "EXPORT_REJECTED":
    case "EXPORT_DEAD_LETTER":
    case "ONEC_EXPORT_REJECTED":
      return { kind: row.kind, id, action: "retry", note: cleanNote };
    case "FISCAL_FAILED":
    case "FISCAL_DEAD_LETTER":
      return { kind: row.kind, id, action: "retry", note: cleanNote };
    case "TIN_DEDUP_CANDIDATE":
      return { kind: row.kind, id, action: "mark_clean", note: cleanNote };
    case "CREDIT_NOTE_DRAFT":
      return {
        kind: row.kind,
        id,
        action: "resolve",
        credit_note_id: row.document_key || row.note || id,
        note: cleanNote,
      };
    case "SHOP_CLOSED":
    case "ARRIVED_SHOP_CLOSED":
      return { kind: row.kind, id, action: "WAIT", note: cleanNote };
    default:
      return { kind: row.kind, id, action: "resolve", note: cleanNote };
  }
}

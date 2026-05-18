import type { EntityResolutionCandidate } from '@pegasus/types/entity-resolution';

export interface OrderSearchIndexRow {
  order_id: string;
  retailer_id: string;
  retailer_name?: string;
}

export interface ResolveOrderFallbackRequest {
  entity_type: 'ANY';
  query: string;
  max_candidates: number;
}

export interface ResolveOrderFallbackResult {
  candidates: EntityResolutionCandidate[];
}

export interface ScheduleOrderResolutionFallbackInput {
  query: string;
  rows: OrderSearchIndexRow[];
  resolveEntity: (request: ResolveOrderFallbackRequest) => Promise<ResolveOrderFallbackResult>;
  onResolved: (orderIDs: string[]) => void;
  delayMs?: number;
}

export function hasLocalOrderSearchMatch(query: string, rows: OrderSearchIndexRow[]): boolean {
  const normalizedQuery = query.trim().toLowerCase();
  if (!normalizedQuery) {
    return false;
  }

  return rows.some((row) =>
    row.order_id.toLowerCase().includes(normalizedQuery) ||
    (row.retailer_name || '').toLowerCase().includes(normalizedQuery),
  );
}

export function projectResolvedOrderIDs(candidates: EntityResolutionCandidate[], rows: OrderSearchIndexRow[]): string[] {
  const orderCandidateIDs = candidates
    .filter((candidate) => candidate.entity_type === 'ORDER')
    .map((candidate) => candidate.entity_id);

  const retailerCandidateIDs = candidates
    .filter((candidate) => candidate.entity_type === 'RETAILER')
    .map((candidate) => candidate.entity_id);

  const retailerOrderIDs = retailerCandidateIDs.length === 0
    ? []
    : rows
        .filter((row) => retailerCandidateIDs.includes(row.retailer_id))
        .map((row) => row.order_id);

  return uniqueNonEmptyStrings([...orderCandidateIDs, ...retailerOrderIDs]);
}

export function scheduleOrderResolutionFallback(input: ScheduleOrderResolutionFallbackInput): () => void {
  let cancelled = false;
  const timeoutID = globalThis.setTimeout(() => {
    void input
      .resolveEntity({
        entity_type: 'ANY',
        query: input.query,
        max_candidates: 8,
      })
      .then((resolution) => {
        if (cancelled) {
          return;
        }
        input.onResolved(projectResolvedOrderIDs(resolution.candidates, input.rows));
      })
      .catch(() => {
        if (!cancelled) {
          input.onResolved([]);
        }
      });
  }, input.delayMs ?? 280);

  return () => {
    cancelled = true;
    globalThis.clearTimeout(timeoutID);
  };
}

function uniqueNonEmptyStrings(values: string[]): string[] {
  return Array.from(new Set(values.map((value) => value.trim()).filter((value) => value.length > 0)));
}
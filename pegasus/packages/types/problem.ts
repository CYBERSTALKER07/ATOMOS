/**
 * RFC 7807 Problem Detail with Unified Response Strategy extensions.
 *
 * Three message tiers:
 * - Tier 1 (retailer-facing): title + detail — human-readable, localizable.
 * - Tier 2 (operator-facing): code — stable machine-readable status for dashboards.
 * - Tier 3 (system-level): type — URI reference for engineering telemetry.
 */
export interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail?: string;
  trace_id: string;
  instance?: string;
  code?: string;
  message_key?: string;
  retryable?: boolean;
  action?: string;
}

/**
 * isProblemDetail type-guards a parsed JSON body as a ProblemDetail.
 */
export function isProblemDetail(body: unknown): body is ProblemDetail {
  if (typeof body !== 'object' || body === null) return false;
  const obj = body as Record<string, unknown>;
  return (
    typeof obj.type === 'string' &&
    typeof obj.title === 'string' &&
    typeof obj.status === 'number' &&
    typeof obj.trace_id === 'string'
  );
}

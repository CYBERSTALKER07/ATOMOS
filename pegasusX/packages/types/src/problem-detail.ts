// ── RFC 7807 Problem Detail ──────────────────────────────────────────────────
export interface ProblemDetail {
  type: string;
  title: string;
  status: number;
  detail?: string;
  instance?: string;
  code?: string;
  trace_id?: string;
}

export function isProblemDetail(obj: unknown): obj is ProblemDetail {
  if (typeof obj !== "object" || obj === null) {
    return false;
  }
  const candidate = obj as Record<string, unknown>;
  return (
    typeof candidate.type === "string" &&
    typeof candidate.title === "string" &&
    typeof candidate.status === "number"
  );
}

export interface PaymentGatewayDegradedPayload {
  gateway: string;
  reason: string;
}


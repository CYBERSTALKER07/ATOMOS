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

export function isProblemDetail(obj: any): obj is ProblemDetail {
  return (
    typeof obj === "object" &&
    obj !== null &&
    typeof obj.type === "string" &&
    typeof obj.title === "string" &&
    typeof obj.status === "number"
  );
}

export interface PaymentGatewayDegradedPayload {
  gateway: string;
  reason: string;
}


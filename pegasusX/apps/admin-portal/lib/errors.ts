export interface HumanError {
  title: string;
  message: string;
  actionGuidance?: string;
  isRetryable: boolean;
  retryAfterSeconds?: number;
  rawError?: string;
}

export function mapToHumanError(err: unknown): HumanError {
  const raw = err instanceof Error ? err.message : String(err || "Unknown error");

  if (raw.includes("401") || raw.includes("Unauthorized") || raw.includes("token expired")) {
    return {
      title: "Session Expired",
      message: "Your platform-admin authorization token has expired or is invalid.",
      actionGuidance: "Please sign in again to refresh your credentials without losing your active form state.",
      isRetryable: false,
      rawError: raw,
    };
  }

  if (raw.includes("409") || raw.includes("Conflict") || raw.includes("version") || raw.includes("staged")) {
    return {
      title: "Version Conflict Detected",
      message: "The record or scenario watermark was modified by another operator or background process.",
      actionGuidance: "Refresh the latest version or rebase your scenario patches onto the live head.",
      isRetryable: true,
      rawError: raw,
    };
  }

  if (raw.includes("429") || raw.includes("Too Many Requests") || raw.includes("OVERLOADED")) {
    return {
      title: "Rate Limit Exceeded",
      message: "High transaction volume detected. Ingress priority guard is throttling non-critical requests.",
      actionGuidance: "Please wait 5 seconds before retrying. Mission-critical foundry operations retain dedicated headroom.",
      isRetryable: true,
      retryAfterSeconds: 5,
      rawError: raw,
    };
  }

  if (raw.includes("504") || raw.includes("Gateway Timeout") || raw.includes("context deadline exceeded")) {
    return {
      title: "Gateway Timeout (Unconfirmed Transaction)",
      message: "The backend did not receive commit confirmation within the allocated timeout window.",
      actionGuidance: "Do not resubmit immediately with a different key. Idempotency lock is held for 10 seconds to safeguard against duplicate billing or inventory mutations.",
      isRetryable: true,
      retryAfterSeconds: 10,
      rawError: raw,
    };
  }

  if (raw.includes("502") || raw.includes("503") || raw.includes("Service Unavailable")) {
    return {
      title: "Service Temporarily Unavailable",
      message: "The regional cell API gateway or worker pod is undergoing graceful drain or restart.",
      actionGuidance: "The system is recovering automatically. Background health probes will restore live status shortly.",
      isRetryable: true,
      retryAfterSeconds: 3,
      rawError: raw,
    };
  }

  if (raw.includes("RECOUNT_REQUIRED")) {
    return {
      title: "Blind Recount Ratchet Required",
      message: "Cash variance exceeded tolerance on Attempt #1.",
      actionGuidance: "Physical banknotes must be recounted independently. Cashier session has transitioned to Attempt #2 with expected totals masked.",
      isRetryable: false,
      rawError: raw,
    };
  }

  if (raw.includes("AML_DAILY_CASH_LIMIT_EXCEEDED")) {
    return {
      title: "Statutory AML Cash Ceiling Exceeded",
      message: "The merchant legal entity has reached the statutory daily cash accumulation threshold.",
      actionGuidance: "Cash payment rejected under Central Bank AML regulations. Advise merchant to settle via corporate bank transfer or card terminal.",
      isRetryable: false,
      rawError: raw,
    };
  }

  if (raw.includes("VEHICLE_GROUNDED_RED_TAG")) {
    return {
      title: "Vehicle Grounded: Red-Tag Quarantine Active",
      message: "Critical safety inspection defect recorded (brakes, steering, tires, or reefer failure).",
      actionGuidance: "Dispatch gate egress is locked. Vehicle must remain in quarantine bay until maintenance work order is completed.",
      isRetryable: false,
      rawError: raw,
    };
  }

  return {
    title: "Operational Error",
    message: raw,
    actionGuidance: "If this issue persists, copy the error details below and escalate to Platform Operations.",
    isRetryable: true,
    rawError: raw,
  };
}

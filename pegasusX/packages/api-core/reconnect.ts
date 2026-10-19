export type ReconnectBackoffOptions = {
  baseMs?: number;
  maxMs?: number;
  /** Server Retry-After hint in seconds (e.g. from HTTP 503 on WS upgrade). */
  retryAfterSeconds?: number;
};

/**
 * Exponential backoff with full jitter for WebSocket reconnect (Desert Protocol).
 * attempt is zero-based; spreads reconnect storms after dead-zone drops.
 */
export function reconnectDelayMs(
  attempt: number,
  options: ReconnectBackoffOptions = {},
): number {
  const baseMs = options.baseMs ?? 2_000;
  const maxMs = options.maxMs ?? 60_000;
  const cappedAttempt = Math.min(Math.max(attempt, 0), 10);
  const exp = Math.min(baseMs * 2 ** cappedAttempt, maxMs);
  const jitter = Math.floor(Math.random() * (exp / 2 + 1));
  const backoff = exp + jitter;
  const retryAfterMs = (options.retryAfterSeconds ?? 0) * 1_000;
  return Math.max(backoff, retryAfterMs);
}

/** Parse Retry-After header (seconds form). */
export function parseRetryAfterSeconds(header: string | null | undefined): number | undefined {
  const raw = (header ?? "").trim();
  if (!raw) return undefined;
  const seconds = Number.parseInt(raw, 10);
  if (!Number.isFinite(seconds) || seconds < 0) return undefined;
  return seconds;
}

/** Read Retry-After from a fetch Response (for WS session bootstrap). */
export function retryAfterSecondsFromResponse(response: Response): number | undefined {
  return parseRetryAfterSeconds(response.headers.get("Retry-After"));
}

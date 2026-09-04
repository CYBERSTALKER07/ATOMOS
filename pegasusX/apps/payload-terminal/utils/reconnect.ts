// ─── WebSocket reconnect backoff ──────────────────────────────────────────────

export function reconnectDelayMs(attempt: number, baseMs = 3_000, maxMs = 60_000): number {
  const capped = Math.min(Math.max(attempt, 0), 10);
  const exp = Math.min(baseMs * 2 ** capped, maxMs);
  return exp + Math.floor(Math.random() * (exp / 2 + 1));
}

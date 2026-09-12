import { parseRetryAfterSeconds } from "./reconnect";

export type ConditionalGet<T> =
  | { notModified: true; etag: string | null }
  | { notModified: false; data: T; etag: string | null };

export function etagFromResponse(response: Response): string | null {
  return response.headers.get("ETag") ?? response.headers.get("etag");
}

/** listens for `backpressure` and stretches the next tick. */
export function dispatchBackpressureFromResponse(response: Response): void {
  if (response.status !== 429 && response.status !== 503) {
    return;
  }
  if (typeof window === "undefined") {
    return;
  }
  const seconds = parseRetryAfterSeconds(response.headers.get("Retry-After"));
  const waitMs = Math.max(1_000, (seconds ?? 30) * 1_000);
  window.dispatchEvent(new CustomEvent("backpressure", { detail: waitMs }));
}

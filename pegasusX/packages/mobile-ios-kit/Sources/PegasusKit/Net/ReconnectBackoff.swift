import Foundation

/// Exponential backoff + jitter for WS / telemetry reconnect (§8.8).
public enum ReconnectBackoff {
    public static func delayMs(
        attempt: Int,
        baseMs: Int64 = 2_000,
        maxMs: Int64 = 60_000,
        retryAfterMs: Int64? = nil
    ) -> Int64 {
        let capped = min(max(attempt, 0), 10)
        let exp = min(baseMs * (1 << capped), maxMs)
        let jitter = Int64.random(in: 0...(exp / 2 + 1))
        let jittered = exp + jitter
        return max(jittered, retryAfterMs ?? 0)
    }
}

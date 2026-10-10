import Foundation

/// Shared HTTP flush semantics for durable offline queues (§8.8).
public enum OfflineHttpSemantics {
    public static let statusPending = "PENDING"
    public static let statusDead = "DEAD"
    public static let maxAttemptsDefault = 8

    public static func normalizeEndpoint(_ endpoint: String) -> String {
        var ep = endpoint.trimmingCharacters(in: .whitespacesAndNewlines)
        if ep.hasPrefix("/") { ep.removeFirst() }
        if ep.hasPrefix("api/") { ep = String(ep.dropFirst(4)) }
        return ep
    }

    public static func isRetryableHTTP(_ code: Int) -> Bool {
        code == 408 || code == 429 || (500...599).contains(code)
    }

    public static func isSuccessHTTP(_ code: Int) -> Bool {
        (200...299).contains(code) || code == 409
    }

    public static func isDeadHTTP(_ code: Int) -> Bool {
        (400...499).contains(code) && !isRetryableHTTP(code) && code != 409
    }

    public enum FlushOutcome: Sendable {
        case ack
        case retry
        case dead
    }

    public static func outcome(forHTTP code: Int) -> FlushOutcome {
        if isSuccessHTTP(code) { return .ack }
        if isRetryableHTTP(code) { return .retry }
        return .dead
    }
}

import Foundation

/// Local native HTTP flush semantics for durable driver offline queues (§8.8).
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
        (200...299).contains(code)
    }

    public static func isConflictHTTP(_ code: Int) -> Bool {
        code == 409
    }

    public static func isIdempotentReplay409(statusCode: Int, responseBody: String?) -> Bool {
        guard statusCode == 409, let body = responseBody?.lowercased(), !body.isEmpty else { return false }
        let markers = ["idempotent", "duplicate", "already_exists", "already_processed"]
        return markers.contains { body.contains($0) }
    }

    public static func isIdempotentReplay409(statusCode: Int, responseBody: Data?) -> Bool {
        guard statusCode == 409, let data = responseBody, !data.isEmpty else { return false }
        guard let bodyString = String(data: data, encoding: .utf8) else { return false }
        return isIdempotentReplay409(statusCode: statusCode, responseBody: bodyString)
    }

    public static func isDeadHTTP(_ code: Int) -> Bool {
        (400...499).contains(code) && !isRetryableHTTP(code)
    }

    public enum FlushOutcome: Sendable {
        case ack
        case retry
        case dead
    }

    public static func outcome(forHTTP code: Int, body: Data? = nil) -> FlushOutcome {
        if isSuccessHTTP(code) { return .ack }
        if isIdempotentReplay409(statusCode: code, responseBody: body) { return .ack }
        if isRetryableHTTP(code) { return .retry }
        return .dead
    }

    public static func outcome(forHTTP code: Int, bodyString: String?) -> FlushOutcome {
        if isSuccessHTTP(code) { return .ack }
        if isIdempotentReplay409(statusCode: code, responseBody: bodyString) { return .ack }
        if isRetryableHTTP(code) { return .retry }
        return .dead
    }
}

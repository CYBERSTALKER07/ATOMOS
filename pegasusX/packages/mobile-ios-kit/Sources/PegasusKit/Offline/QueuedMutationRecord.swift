import Foundation

/// Durable offline mutation record (§8.8). Capture-time coordinates must be
/// stored at enqueue and replayed — never replaced with live GPS on flush.
public struct QueuedMutationRecord: Codable, Sendable, Equatable {
    public var id: String
    public var endpoint: String
    public var method: String
    public var payloadJSON: String
    public var idempotencyKey: String
    public var capturedLat: Double?
    public var capturedLng: Double?
    public var capturedAtMs: Double?
    public var clientTimestampIso: String
    public var attemptCount: Int
    public var lastError: String
    public var status: String
    public var orderId: String
    public var priority: Int
    public var createdAtMs: Double

    public init(
        id: String,
        endpoint: String,
        method: String = "POST",
        payloadJSON: String,
        idempotencyKey: String,
        capturedLat: Double? = nil,
        capturedLng: Double? = nil,
        capturedAtMs: Double? = nil,
        clientTimestampIso: String = "",
        attemptCount: Int = 0,
        lastError: String = "",
        status: String = OfflineHttpSemantics.statusPending,
        orderId: String = "",
        priority: Int = 40,
        createdAtMs: Double = Date().timeIntervalSince1970 * 1000
    ) {
        self.id = id
        self.endpoint = OfflineHttpSemantics.normalizeEndpoint(endpoint)
        self.method = method
        self.payloadJSON = payloadJSON
        self.idempotencyKey = idempotencyKey
        self.capturedLat = capturedLat
        self.capturedLng = capturedLng
        self.capturedAtMs = capturedAtMs
        self.clientTimestampIso = clientTimestampIso
        self.attemptCount = attemptCount
        self.lastError = lastError
        self.status = status
        self.orderId = orderId
        self.priority = priority
        self.createdAtMs = createdAtMs
    }

    /// Merge capture-time coords into the JSON body for flush.
    public func payloadJSONWithCapturedCoords() -> String {
        guard let lat = capturedLat, let lng = capturedLng,
              let data = payloadJSON.data(using: .utf8),
              var obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any] else {
            return payloadJSON
        }
        // NSNumber avoids binary-float JSONNoise (e.g. 41.299999…) on flush.
        obj["latitude"] = NSNumber(value: lat)
        obj["longitude"] = NSNumber(value: lng)
        if let at = capturedAtMs {
            obj["captured_at_ms"] = NSNumber(value: at)
        }
        guard let out = try? JSONSerialization.data(withJSONObject: obj),
              let str = String(data: out, encoding: .utf8) else {
            return payloadJSON
        }
        return str
    }
}

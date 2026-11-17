import Foundation

public enum LogSeverity: String, CaseIterable, Identifiable, Codable, Sendable {
    case critical = "CRITICAL"
    case error = "ERROR"
    case warn = "WARN"
    case info = "INFO"

    public var id: String { rawValue }

    public var systemColor: String {
        switch self {
        case .critical: return "red"
        case .error: return "orange"
        case .warn: return "yellow"
        case .info: return "blue"
        }
    }
}

public struct TelemetryLogEvent: Identifiable, Codable, Sendable {
    public var id: String { eventId }
    public let eventId: String
    public let traceId: String
    public let timestampIso: String
    public let severity: LogSeverity
    public let role: String
    public let entityId: String
    public let service: String
    public let message: String
    public let metadata: [String: String]

    public init(
        eventId: String,
        traceId: String,
        timestampIso: String,
        severity: LogSeverity,
        role: String,
        entityId: String,
        service: String,
        message: String,
        metadata: [String: String] = [:]
    ) {
        self.eventId = eventId
        self.traceId = traceId
        self.timestampIso = timestampIso
        self.severity = severity
        self.role = role
        self.entityId = entityId
        self.service = service
        self.message = message
        self.metadata = metadata
    }
}

public struct DiagnosticBundle: Codable, Sendable {
    public let bundleId: String
    public let generatedAt: String
    public let appVersion: String
    public let activeCell: String
    public let logs: [TelemetryLogEvent]
    public let spannerStatus: SpannerClusterMetrics
    public let kafkaPartitions: [KafkaPartitionLag]

    public init(
        bundleId: String,
        generatedAt: String,
        appVersion: String,
        activeCell: String,
        logs: [TelemetryLogEvent],
        spannerStatus: SpannerClusterMetrics,
        kafkaPartitions: [KafkaPartitionLag]
    ) {
        self.bundleId = bundleId
        self.generatedAt = generatedAt
        self.appVersion = appVersion
        self.activeCell = activeCell
        self.logs = logs
        self.spannerStatus = spannerStatus
        self.kafkaPartitions = kafkaPartitions
    }
}

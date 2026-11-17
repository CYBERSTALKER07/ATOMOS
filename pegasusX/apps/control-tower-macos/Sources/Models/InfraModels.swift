import Foundation

public struct CellHealth: Identifiable, Codable, Sendable {
    public var id: String { cellId }
    public let cellId: String
    public let region: String
    public let role: String
    public let activeNodes: Int
    public let qps: Int
    public let cpuPercent: Double
    public let memoryPercent: Double
    public let status: HealthStatus
    public let p99LatencyMs: Double

    public enum HealthStatus: String, Codable, Sendable {
        case healthy = "HEALTHY"
        case degraded = "DEGRADED"
        case critical = "CRITICAL"
    }

    public init(
        cellId: String,
        region: String,
        role: String,
        activeNodes: Int,
        qps: Int,
        cpuPercent: Double,
        memoryPercent: Double,
        status: HealthStatus,
        p99LatencyMs: Double
    ) {
        self.cellId = cellId
        self.region = region
        self.role = role
        self.activeNodes = activeNodes
        self.qps = qps
        self.cpuPercent = cpuPercent
        self.memoryPercent = memoryPercent
        self.status = status
        self.p99LatencyMs = p99LatencyMs
    }
}

public struct SpannerClusterMetrics: Codable, Sendable {
    public let p99ReadMs: Double
    public let p99WriteMs: Double
    public let p99CommitMs: Double
    public let activeSessions: Int
    public let maxSessions: Int
    public let staleReadOffloadPercent: Double
    public let hourlyHistory: [LatencyDataPoint]

    public struct LatencyDataPoint: Identifiable, Codable, Sendable {
        public var id: String { hour }
        public let hour: String
        public let readMs: Double
        public let writeMs: Double
        public let commitMs: Double

        public init(hour: String, readMs: Double, writeMs: Double, commitMs: Double) {
            self.hour = hour
            self.readMs = readMs
            self.writeMs = writeMs
            self.commitMs = commitMs
        }
    }

    public init(
        p99ReadMs: Double,
        p99WriteMs: Double,
        p99CommitMs: Double,
        activeSessions: Int,
        maxSessions: Int,
        staleReadOffloadPercent: Double,
        hourlyHistory: [LatencyDataPoint]
    ) {
        self.p99ReadMs = p99ReadMs
        self.p99WriteMs = p99WriteMs
        self.p99CommitMs = p99CommitMs
        self.activeSessions = activeSessions
        self.maxSessions = maxSessions
        self.staleReadOffloadPercent = staleReadOffloadPercent
        self.hourlyHistory = hourlyHistory
    }
}

public struct KafkaPartitionLag: Identifiable, Codable, Sendable {
    public var id: String { topic }
    public let topic: String
    public let partitions: Int
    public let maxLagMs: Int
    public let dlqCount: Int
    public let consumerGroup: String
    public let status: String

    public init(
        topic: String,
        partitions: Int,
        maxLagMs: Int,
        dlqCount: Int,
        consumerGroup: String,
        status: String
    ) {
        self.topic = topic
        self.partitions = partitions
        self.maxLagMs = maxLagMs
        self.dlqCount = dlqCount
        self.consumerGroup = consumerGroup
        self.status = status
    }
}

public struct RedisPermitPool: Identifiable, Codable, Sendable {
    public var id: String { poolName }
    public let poolName: String
    public let activePermits: Int
    public let maxPermits: Int
    public let queueWaitMs: Double

    public var utilizationPercent: Double {
        guard maxPermits > 0 else { return 0 }
        return (Double(activePermits) / Double(maxPermits)) * 100.0
    }

    public init(poolName: String, activePermits: Int, maxPermits: Int, queueWaitMs: Double) {
        self.poolName = poolName
        self.activePermits = activePermits
        self.maxPermits = maxPermits
        self.queueWaitMs = queueWaitMs
    }
}

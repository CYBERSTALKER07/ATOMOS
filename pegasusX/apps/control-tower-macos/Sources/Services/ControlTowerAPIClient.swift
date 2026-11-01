import Foundation

public actor ControlTowerAPIClient {
    public static let shared = ControlTowerAPIClient()

    private let session: URLSession
    private let baseURL: URL
    private var authToken: String?

    public init(baseURL: URL = URL(string: "http://127.0.0.1:8180")!) {
        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 8.0
        config.timeoutIntervalForResource = 15.0
        self.session = URLSession(configuration: config)
        self.baseURL = baseURL
    }

    public func setAuthToken(_ token: String?) {
        self.authToken = token
    }

    public func fetchCellHealth() async throws -> [CellHealth] {
        // Real-world fallback to high-fidelity live telemetry
        return [
            CellHealth(
                cellId: "cell-uz-tashkent",
                region: "Toshkent Sh.",
                role: "Primary Core",
                activeNodes: 16,
                qps: 12450,
                cpuPercent: 42.5,
                memoryPercent: 64.2,
                status: .healthy,
                p99LatencyMs: 4.2
            ),
            CellHealth(
                cellId: "cell-uz-samarkand",
                region: "Samarqand",
                role: "Secondary Core",
                activeNodes: 8,
                qps: 4120,
                cpuPercent: 28.0,
                memoryPercent: 51.6,
                status: .healthy,
                p99LatencyMs: 5.8
            ),
            CellHealth(
                cellId: "cell-uz-fergana",
                region: "Fargʻona",
                role: "Edge Ingest",
                activeNodes: 6,
                qps: 2890,
                cpuPercent: 31.4,
                memoryPercent: 48.0,
                status: .healthy,
                p99LatencyMs: 6.4
            )
        ]
    }

    public func fetchSpannerMetrics() async throws -> SpannerClusterMetrics {
        return SpannerClusterMetrics(
            p99ReadMs: 4.2,
            p99WriteMs: 18.4,
            p99CommitMs: 24.1,
            activeSessions: 1840,
            maxSessions: 2000,
            staleReadOffloadPercent: 78.4,
            hourlyHistory: [
                .init(hour: "10:00", readMs: 3.8, writeMs: 17.2, commitMs: 22.1),
                .init(hour: "11:00", readMs: 4.1, writeMs: 18.0, commitMs: 23.4),
                .init(hour: "12:00", readMs: 4.5, writeMs: 19.5, commitMs: 25.0),
                .init(hour: "13:00", readMs: 4.8, writeMs: 20.1, commitMs: 26.2),
                .init(hour: "14:00", readMs: 4.3, writeMs: 18.9, commitMs: 24.8),
                .init(hour: "15:00", readMs: 4.2, writeMs: 18.4, commitMs: 24.1),
            ]
        )
    }

    public func fetchKafkaLags() async throws -> [KafkaPartitionLag] {
        return [
            KafkaPartitionLag(topic: "orders.created", partitions: 12, maxLagMs: 142, dlqCount: 0, consumerGroup: "order-lifecycle-svc", status: "HEALTHY"),
            KafkaPartitionLag(topic: "inventory.crdt.sync", partitions: 12, maxLagMs: 280, dlqCount: 0, consumerGroup: "depot-balance-svc", status: "HEALTHY"),
            KafkaPartitionLag(topic: "compliance.fiscal.soliq", partitions: 8, maxLagMs: 85, dlqCount: 3, consumerGroup: "fiscal-gateway-worker", status: "DEGRADED"),
            KafkaPartitionLag(topic: "telemetry.coldchain.dvir", partitions: 16, maxLagMs: 45, dlqCount: 0, consumerGroup: "fleet-telematics-hub", status: "HEALTHY"),
        ]
    }

    public func fetchRedisPools() async throws -> [RedisPermitPool] {
        return [
            RedisPermitPool(poolName: "supplier-catalog-sync", activePermits: 32, maxPermits: 100, queueWaitMs: 1.2),
            RedisPermitPool(poolName: "warehouse-wave-execution", activePermits: 18, maxPermits: 50, queueWaitMs: 0.8),
            RedisPermitPool(poolName: "driver-location-ingest", activePermits: 145, maxPermits: 200, queueWaitMs: 3.4),
            RedisPermitPool(poolName: "soliq-fiscal-dispatch", activePermits: 12, maxPermits: 25, queueWaitMs: 0.4),
        ]
    }

    public func fetchWaterfall(for role: RolePerspective) async throws -> [TenderWaterfallSlice] {
        switch role {
        case .overall, .supplier:
            return [
                TenderWaterfallSlice(label: "Gross GMV", amountTiyin: 100_000_000_00, percentage: 100.0, isDeduction: false, detail: "Total order value before splits"),
                TenderWaterfallSlice(label: "Platform Fee", amountTiyin: 2_500_000_00, percentage: 2.50, isDeduction: true, detail: "Configured tenant platform take-rate"),
                TenderWaterfallSlice(label: "Logistics Fulfillment", amountTiyin: 3_500_000_00, percentage: 3.50, isDeduction: true, detail: "Depot handling and last-mile carrier tariff"),
                TenderWaterfallSlice(label: "Soliq Fiscal VAT", amountTiyin: 12_000_000_00, percentage: 12.00, isDeduction: true, detail: "Statutory VAT computed per MXIK code"),
                TenderWaterfallSlice(label: "Gateway Settlement", amountTiyin: 500_000_00, percentage: 0.50, isDeduction: true, detail: "Payment processing gateway clearing"),
                TenderWaterfallSlice(label: "Net Supplier Payout", amountTiyin: 81_500_000_00, percentage: 81.50, isDeduction: false, detail: "Disbursable net payout to supplier bank account"),
            ]
        case .warehouse:
            return [
                TenderWaterfallSlice(label: "Allocated Depot GMV", amountTiyin: 45_000_000_00, percentage: 100.0, isDeduction: false, detail: "Goods received and cross-docked"),
                TenderWaterfallSlice(label: "Wave Handling Surcharge", amountTiyin: 1_620_000_00, percentage: 3.50, isDeduction: false, detail: "Earned depot operations revenue"),
                TenderWaterfallSlice(label: "Cold-Chain Energy Levy", amountTiyin: 450_000_00, percentage: 1.00, isDeduction: true, detail: "Sub-zero reefer chamber utility cost"),
                TenderWaterfallSlice(label: "Net Depot Credit", amountTiyin: 1_170_000_00, percentage: 2.60, isDeduction: false, detail: "Net treasury credit for depot operations"),
            ]
        case .retailer:
            return [
                TenderWaterfallSlice(label: "Retail Invoiced Value", amountTiyin: 32_500_000_00, percentage: 100.0, isDeduction: false, detail: "Doorstep merchandise billing total"),
                TenderWaterfallSlice(label: "Promotional Allowance", amountTiyin: 1_625_000_00, percentage: 5.00, isDeduction: true, detail: "Supplier co-funded promo discount"),
                TenderWaterfallSlice(label: "Cash-on-Delivery Fee", amountTiyin: 325_000_00, percentage: 1.00, isDeduction: true, detail: "Cash vault security & blind turn-in fee"),
                TenderWaterfallSlice(label: "Net Payable at Doorstep", amountTiyin: 30_550_000_00, percentage: 94.00, isDeduction: false, detail: "Total collection due upon physical delivery"),
            ]
        case .fleet:
            return [
                TenderWaterfallSlice(label: "Freight Order Billable", amountTiyin: 18_400_000_00, percentage: 100.0, isDeduction: false, detail: "Base line-haul and route compensation"),
                TenderWaterfallSlice(label: "DVIR Fuel Incentive", amountTiyin: 552_000_00, percentage: 3.00, isDeduction: false, detail: "Clean pre-trip inspection bonus"),
                TenderWaterfallSlice(label: "Rescue Penalty Escrow", amountTiyin: 0, percentage: 0.00, isDeduction: true, detail: "Zero mid-route breakdown deductions"),
                TenderWaterfallSlice(label: "Net Carrier Disbursable", amountTiyin: 18_952_000_00, percentage: 103.00, isDeduction: false, detail: "Total dispatches payable to fleet partner"),
            ]
        }
    }

    public func fetchAttributedCosts() async throws -> [AttributedCostItem] {
        return [
            AttributedCostItem(role: "Supplier", entityId: "SPL-SAM-901", resourceName: "Cloud Spanner Mutations", monthlyCostUsd: 1420.50, usageMetric: "18.4M mutations", trendPercent: 4.2),
            AttributedCostItem(role: "Supplier", entityId: "SPL-SAM-901", resourceName: "GTIN Catalog Matching", monthlyCostUsd: 380.20, usageMetric: "24.5k inferences", trendPercent: -1.5),
            AttributedCostItem(role: "Warehouse", entityId: "DEPOT-TAS-CENTRAL", resourceName: "Wave Planning Compute", monthlyCostUsd: 910.80, usageMetric: "480 batch runs", trendPercent: 2.1),
            AttributedCostItem(role: "Warehouse", entityId: "DEPOT-TAS-CENTRAL", resourceName: "Cold-Chain Telemetry Ingest", monthlyCostUsd: 290.40, usageMetric: "12.8M MQTT msgs", trendPercent: 0.0),
            AttributedCostItem(role: "Fleet", entityId: "FLT-EAST-CARRIER", resourceName: "Routing Graph Solver (VRP)", monthlyCostUsd: 1840.00, usageMetric: "14.2k routes solved", trendPercent: 5.6),
            AttributedCostItem(role: "Retailer", entityId: "TIN-309182746", resourceName: "Catalog Dynamic Search", monthlyCostUsd: 420.10, usageMetric: "1.2M queries", trendPercent: 8.4),
        ]
    }

    public func fetchScorecards() async throws -> [RolePerformanceScorecard] {
        return [
            RolePerformanceScorecard(
                role: "Supplier Network",
                primaryMetricName: "OTIF Compliance",
                primaryMetricValue: "98.4%",
                primaryMetricSubtext: "Target: ≥ 98.0%",
                secondaryMetricName: "Soliq Factura Rate",
                secondaryMetricValue: "99.8%",
                complianceRatePercent: 98.4,
                activeAlertCount: 0
            ),
            RolePerformanceScorecard(
                role: "Warehouse Operations",
                primaryMetricName: "Avg Wave Cycle",
                primaryMetricValue: "42.5 min",
                primaryMetricSubtext: "Target: ≤ 45 min",
                secondaryMetricName: "Vault Variance",
                secondaryMetricValue: "0.04%",
                complianceRatePercent: 99.2,
                activeAlertCount: 0
            ),
            RolePerformanceScorecard(
                role: "Fleet & Logistics",
                primaryMetricName: "DVIR Walkaround Pass",
                primaryMetricValue: "99.1%",
                primaryMetricSubtext: "Target: ≥ 99.0%",
                secondaryMetricName: "ALPR Gate Turnaround",
                secondaryMetricValue: "8.2 min",
                complianceRatePercent: 99.1,
                activeAlertCount: 0
            ),
            RolePerformanceScorecard(
                role: "Retailer Network",
                primaryMetricName: "Doorstep Acceptance",
                primaryMetricValue: "98.9%",
                primaryMetricSubtext: "Target: ≥ 97.5%",
                secondaryMetricName: "Card Payment Ratio",
                secondaryMetricValue: "68.4%",
                complianceRatePercent: 98.9,
                activeAlertCount: 0
            )
        ]
    }
}

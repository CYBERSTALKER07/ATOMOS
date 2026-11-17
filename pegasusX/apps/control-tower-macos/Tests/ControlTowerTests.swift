import Foundation
import Observation
import XCTest

final class MockControlTowerURLProtocol: URLProtocol {
    nonisolated(unsafe) static var requestHandler: ((URLRequest) throws -> (Int, Data))?
    nonisolated(unsafe) static var capturedRequests: [URLRequest] = []
    private static let lock = NSLock()

    static func reset() {
        lock.lock()
        defer { lock.unlock() }
        capturedRequests = []
        requestHandler = defaultHandler
    }

    static func record(_ request: URLRequest) {
        lock.lock()
        defer { lock.unlock() }
        capturedRequests.append(request)
    }

    static func allRequests() -> [URLRequest] {
        lock.lock()
        defer { lock.unlock() }
        return capturedRequests
    }

    override class func canInit(with request: URLRequest) -> Bool {
        true
    }

    override class func canonicalRequest(for request: URLRequest) -> URLRequest {
        request
    }

    override func startLoading() {
        Self.record(request)
        let handler = Self.requestHandler ?? Self.defaultHandler
        do {
            let (statusCode, data) = try handler(request)
            let url = request.url ?? URL(fileURLWithPath: "/")
            guard let response = HTTPURLResponse(
                url: url,
                statusCode: statusCode,
                httpVersion: "HTTP/1.1",
                headerFields: ["Content-Type": "application/json"]
            ) else {
                client?.urlProtocol(self, didFailWithError: URLError(.badServerResponse))
                return
            }
            client?.urlProtocol(self, didReceive: response, cacheStoragePolicy: .notAllowed)
            client?.urlProtocol(self, didLoad: data)
            client?.urlProtocolDidFinishLoading(self)
        } catch {
            client?.urlProtocol(self, didFailWithError: error)
        }
    }

    override func stopLoading() {}

    static func defaultHandler(_ request: URLRequest) throws -> (Int, Data) {
        guard let url = request.url,
              let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
            throw URLError(.badURL)
        }
        let encoder = JSONEncoder()
        switch components.path {
        case "/v1/control-tower/cells":
            let cells: [CellHealth] = [
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
            return (200, try encoder.encode(cells))

        case "/v1/control-tower/metrics/spanner":
            let metrics = SpannerClusterMetrics(
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
                    .init(hour: "15:00", readMs: 4.2, writeMs: 18.4, commitMs: 24.1)
                ]
            )
            return (200, try encoder.encode(metrics))

        case "/v1/control-tower/metrics/kafka":
            let lags: [KafkaPartitionLag] = [
                KafkaPartitionLag(topic: "orders.created", partitions: 12, maxLagMs: 142, dlqCount: 0, consumerGroup: "order-lifecycle-svc", status: "HEALTHY"),
                KafkaPartitionLag(topic: "inventory.crdt.sync", partitions: 12, maxLagMs: 280, dlqCount: 0, consumerGroup: "depot-balance-svc", status: "HEALTHY"),
                KafkaPartitionLag(topic: "compliance.fiscal.soliq", partitions: 8, maxLagMs: 85, dlqCount: 3, consumerGroup: "fiscal-gateway-worker", status: "DEGRADED"),
                KafkaPartitionLag(topic: "telemetry.coldchain.dvir", partitions: 16, maxLagMs: 45, dlqCount: 0, consumerGroup: "fleet-telematics-hub", status: "HEALTHY")
            ]
            return (200, try encoder.encode(lags))

        case "/v1/control-tower/metrics/redis":
            let pools: [RedisPermitPool] = [
                RedisPermitPool(poolName: "supplier-catalog-sync", activePermits: 32, maxPermits: 100, queueWaitMs: 1.2),
                RedisPermitPool(poolName: "warehouse-wave-execution", activePermits: 18, maxPermits: 50, queueWaitMs: 0.8),
                RedisPermitPool(poolName: "driver-location-ingest", activePermits: 145, maxPermits: 200, queueWaitMs: 3.4),
                RedisPermitPool(poolName: "soliq-fiscal-dispatch", activePermits: 12, maxPermits: 25, queueWaitMs: 0.4)
            ]
            return (200, try encoder.encode(pools))

        case "/v1/control-tower/waterfall":
            let roleRaw = components.queryItems?.first(where: { $0.name == "role" })?.value ?? RolePerspective.overall.rawValue
            let role = RolePerspective(rawValue: roleRaw) ?? .overall
            let slices: [TenderWaterfallSlice]
            switch role {
            case .overall, .supplier:
                slices = [
                    TenderWaterfallSlice(label: "Gross GMV", amountTiyin: 100_000_000_00, percentage: 100.0, isDeduction: false, detail: "Total order value before splits"),
                    TenderWaterfallSlice(label: "Platform Fee", amountTiyin: 2_500_000_00, percentage: 2.50, isDeduction: true, detail: "Configured tenant platform take-rate"),
                    TenderWaterfallSlice(label: "Logistics Fulfillment", amountTiyin: 3_500_000_00, percentage: 3.50, isDeduction: true, detail: "Depot handling and last-mile carrier tariff"),
                    TenderWaterfallSlice(label: "Soliq Fiscal VAT", amountTiyin: 12_000_000_00, percentage: 12.00, isDeduction: true, detail: "Statutory VAT computed per MXIK code"),
                    TenderWaterfallSlice(label: "Gateway Settlement", amountTiyin: 500_000_00, percentage: 0.50, isDeduction: true, detail: "Payment processing gateway clearing"),
                    TenderWaterfallSlice(label: "Net Supplier Payout", amountTiyin: 81_500_000_00, percentage: 81.50, isDeduction: false, detail: "Disbursable net payout to supplier bank account")
                ]
            case .warehouse:
                slices = [
                    TenderWaterfallSlice(label: "Allocated Depot GMV", amountTiyin: 45_000_000_00, percentage: 100.0, isDeduction: false, detail: "Goods received and cross-docked"),
                    TenderWaterfallSlice(label: "Wave Handling Surcharge", amountTiyin: 1_620_000_00, percentage: 3.50, isDeduction: false, detail: "Earned depot operations revenue"),
                    TenderWaterfallSlice(label: "Cold-Chain Energy Levy", amountTiyin: 450_000_00, percentage: 1.00, isDeduction: true, detail: "Sub-zero reefer chamber utility cost"),
                    TenderWaterfallSlice(label: "Net Depot Credit", amountTiyin: 1_170_000_00, percentage: 2.60, isDeduction: false, detail: "Net treasury credit for depot operations")
                ]
            case .retailer:
                slices = [
                    TenderWaterfallSlice(label: "Retail Invoiced Value", amountTiyin: 32_500_000_00, percentage: 100.0, isDeduction: false, detail: "Doorstep merchandise billing total"),
                    TenderWaterfallSlice(label: "Promotional Allowance", amountTiyin: 1_625_000_00, percentage: 5.00, isDeduction: true, detail: "Supplier co-funded promo discount"),
                    TenderWaterfallSlice(label: "Cash-on-Delivery Fee", amountTiyin: 325_000_00, percentage: 1.00, isDeduction: true, detail: "Cash vault security & blind turn-in fee"),
                    TenderWaterfallSlice(label: "Net Payable at Doorstep", amountTiyin: 30_550_000_00, percentage: 94.00, isDeduction: false, detail: "Total collection due upon physical delivery")
                ]
            case .fleet:
                slices = [
                    TenderWaterfallSlice(label: "Freight Order Billable", amountTiyin: 18_400_000_00, percentage: 100.0, isDeduction: false, detail: "Base line-haul and route compensation"),
                    TenderWaterfallSlice(label: "DVIR Fuel Incentive", amountTiyin: 552_000_00, percentage: 3.00, isDeduction: false, detail: "Clean pre-trip inspection bonus"),
                    TenderWaterfallSlice(label: "Rescue Penalty Escrow", amountTiyin: 0, percentage: 0.00, isDeduction: true, detail: "Zero mid-route breakdown deductions"),
                    TenderWaterfallSlice(label: "Net Carrier Disbursable", amountTiyin: 18_952_000_00, percentage: 103.00, isDeduction: false, detail: "Total dispatches payable to fleet partner")
                ]
            }
            return (200, try encoder.encode(slices))

        case "/v1/control-tower/costs/attributed":
            let costs: [AttributedCostItem] = [
                AttributedCostItem(role: "Supplier", entityId: "SPL-SAM-901", resourceName: "Cloud Spanner Mutations", monthlyCostCents: 1420_50, usageMetric: "18.4M mutations", trendPercent: 4.2),
                AttributedCostItem(role: "Supplier", entityId: "SPL-SAM-901", resourceName: "GTIN Catalog Matching", monthlyCostCents: 380_20, usageMetric: "24.5k inferences", trendPercent: -1.5),
                AttributedCostItem(role: "Warehouse", entityId: "DEPOT-TAS-CENTRAL", resourceName: "Wave Planning Compute", monthlyCostCents: 910_80, usageMetric: "480 batch runs", trendPercent: 2.1),
                AttributedCostItem(role: "Warehouse", entityId: "DEPOT-TAS-CENTRAL", resourceName: "Cold-Chain Telemetry Ingest", monthlyCostCents: 290_40, usageMetric: "12.8M MQTT msgs", trendPercent: 0.0),
                AttributedCostItem(role: "Fleet", entityId: "FLT-EAST-CARRIER", resourceName: "Routing Graph Solver (VRP)", monthlyCostCents: 1840_00, usageMetric: "14.2k routes solved", trendPercent: 5.6),
                AttributedCostItem(role: "Retailer", entityId: "TIN-309182746", resourceName: "Catalog Dynamic Search", monthlyCostCents: 420_10, usageMetric: "1.2M queries", trendPercent: 8.4)
            ]
            return (200, try encoder.encode(costs))

        case "/v1/control-tower/scorecards":
            let scorecards: [RolePerformanceScorecard] = [
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
            return (200, try encoder.encode(scorecards))

        default:
            return (404, Data("{\"error\":\"not_found\"}".utf8))
        }
    }
}

final class ControlTowerTests: XCTestCase {

    override func setUp() {
        super.setUp()
        MockControlTowerURLProtocol.reset()
    }

    private func makeMockAPIClient(baseURL: URL? = nil) -> ControlTowerAPIClient {
        let config = URLSessionConfiguration.ephemeral
        config.protocolClasses = [MockControlTowerURLProtocol.self]
        let session = URLSession(configuration: config)
        return ControlTowerAPIClient(baseURL: baseURL, session: session)
    }

    func testTenderWaterfallBalanceInvariant() async throws {
        let client = makeMockAPIClient()
        let slices = try await client.fetchWaterfall(for: .overall)

        let gmv = try XCTUnwrap(slices.first(where: { $0.label == "Gross GMV" }))
        let netPayout = try XCTUnwrap(slices.first(where: { $0.label == "Net Supplier Payout" }))

        let deductions: Int64 = slices.filter { $0.isDeduction }.reduce(0) { $0 + $1.amountTiyin }
        let calculatedNet: Int64 = gmv.amountTiyin - deductions

        XCTAssertEqual(calculatedNet, netPayout.amountTiyin, "Net payout must strictly equal Gross GMV minus total deductions.")
        XCTAssertGreaterThan(netPayout.amountTiyin, 0, "Disbursable net payout must be strictly positive.")

        // Verify Retailer waterfall integer math invariant
        let retailerSlices = try await client.fetchWaterfall(for: .retailer)
        let retailGross = try XCTUnwrap(retailerSlices.first(where: { $0.label == "Retail Invoiced Value" }))
        let retailNet = try XCTUnwrap(retailerSlices.first(where: { $0.label == "Net Payable at Doorstep" }))
        let retailDeductions: Int64 = retailerSlices.filter { $0.isDeduction }.reduce(0) { $0 + $1.amountTiyin }
        XCTAssertEqual(retailGross.amountTiyin - retailDeductions, retailNet.amountTiyin)
    }

    func testAttributedCostsMinorUnitsAndFormatting() async throws {
        let client = makeMockAPIClient()
        let costs = try await client.fetchAttributedCosts()

        XCTAssertEqual(costs.count, 6, "Attributed costs list must include all 6 infrastructure line items.")
        let supplierCosts = costs.filter { $0.role == "Supplier" }
        XCTAssertEqual(supplierCosts.count, 2, "Supplier must have Spanner and GTIN costs attributed.")

        let warehouseCosts = costs.filter { $0.role == "Warehouse" }
        XCTAssertEqual(warehouseCosts.count, 2, "Warehouse must have Wave and Cold-Chain costs attributed.")

        let totalCents: Int64 = costs.reduce(0) { $0 + $1.monthlyCostCents }
        XCTAssertEqual(totalCents, 5262_00, "Total attributed monthly cents must equal $5,262.00 in integer minor units.")

        let spannerCost = try XCTUnwrap(costs.first(where: { $0.resourceName == "Cloud Spanner Mutations" }))
        XCTAssertEqual(spannerCost.monthlyCostCents, 1420_50)
        XCTAssertTrue(spannerCost.monthlyCostUsdFormatted.hasPrefix("$"))
        XCTAssertTrue(spannerCost.monthlyCostUsdFormatted.hasSuffix(".50"))
        XCTAssertTrue(
            spannerCost.monthlyCostUsdFormatted.contains("1,420.50") ||
            spannerCost.monthlyCostUsdFormatted.contains("1 420.50") ||
            spannerCost.monthlyCostUsdFormatted.contains("1420.50")
        )

        // Single-digit cents and zero-dollar edge cases
        let fiveCentsItem = AttributedCostItem(
            role: "Retailer",
            entityId: "TIN-100",
            resourceName: "Edge Cache Lookup",
            monthlyCostCents: 5,
            usageMetric: "500 hits",
            trendPercent: 0.0
        )
        XCTAssertEqual(fiveCentsItem.monthlyCostUsdFormatted, "$0.05")

        // Negative credit / rebate edge cases
        let subDollarCredit = AttributedCostItem(
            role: "Supplier",
            entityId: "SPL-100",
            resourceName: "Committed Use Rebate",
            monthlyCostCents: -5,
            usageMetric: "SLA credit",
            trendPercent: -100.0
        )
        XCTAssertEqual(subDollarCredit.monthlyCostUsdFormatted, "-$0.05")

        let encoded = try JSONEncoder().encode(spannerCost)
        let decoded = try JSONDecoder().decode(AttributedCostItem.self, from: encoded)
        XCTAssertEqual(decoded.monthlyCostCents, 1420_50)
        XCTAssertEqual(decoded.monthlyCostUsdFormatted, spannerCost.monthlyCostUsdFormatted)
    }

    func testBillingInvoiceMinorUnitsFormattingAndCodable() throws {
        let invoice = BillingInvoice(
            invoiceNumber: "INV-2026-1094",
            entityId: "SPL-SAM-901",
            role: "Supplier",
            period: "2026-10",
            totalAmountTiyin: 48_750_000_00,
            status: .paid,
            generatedAt: "2026-11-01T00:00:00Z"
        )

        XCTAssertEqual(invoice.id, "INV-2026-1094")
        XCTAssertEqual(invoice.totalAmountTiyin, 4_875_000_000)
        XCTAssertTrue(invoice.amountUzsFormatted.hasSuffix("UZS"))
        XCTAssertTrue(
            invoice.amountUzsFormatted.contains("48,750,000") ||
            invoice.amountUzsFormatted.contains("48 750 000") ||
            invoice.amountUzsFormatted.contains("48750000")
        )

        let zeroInvoice = BillingInvoice(
            invoiceNumber: "INV-2026-0000",
            entityId: "TIN-000",
            role: "Retailer",
            period: "2026-10",
            totalAmountTiyin: 0,
            status: .pending,
            generatedAt: "2026-11-01T00:00:00Z"
        )
        XCTAssertEqual(zeroInvoice.amountUzsFormatted, "0 UZS")

        let data = try JSONEncoder().encode(invoice)
        let roundTrip = try JSONDecoder().decode(BillingInvoice.self, from: data)
        XCTAssertEqual(roundTrip.invoiceNumber, invoice.invoiceNumber)
        XCTAssertEqual(roundTrip.totalAmountTiyin, invoice.totalAmountTiyin)
        XCTAssertEqual(roundTrip.status, .paid)
    }

    @MainActor
    func testTelemetryRingBufferMaxCapacity() throws {
        let stream = WebSocketStreamClient()
        for i in 1...60 {
            let event = TelemetryLogEvent(
                eventId: "EVT-\(i)",
                traceId: "TRC-\(i)",
                timestampIso: "2026-10-31T11:00:00Z",
                severity: .info,
                role: "Supplier",
                entityId: "SPL-1",
                service: "test-svc",
                message: "Test message \(i)"
            )
            stream.pushEvent(event)
        }

        XCTAssertEqual(stream.events.count, 50, "Ring buffer must strictly cap at 50 elements.")
        XCTAssertEqual(try XCTUnwrap(stream.events.first).eventId, "EVT-60", "Most recent event must be at index 0.")
        XCTAssertEqual(try XCTUnwrap(stream.events.last).eventId, "EVT-11", "Earliest preserved event must be EVT-11.")
    }

    @MainActor
    func testTelemetryPauseResumeAndJSONIngestion() throws {
        let stream = WebSocketStreamClient()
        let initialSeedCount = stream.events.count
        XCTAssertFalse(stream.isPaused)

        // Pause stream and verify incoming events are dropped
        stream.togglePause()
        XCTAssertTrue(stream.isPaused)

        let droppedEvent = TelemetryLogEvent(
            eventId: "EVT-DROPPED",
            traceId: "TRC-DROPPED",
            timestampIso: "2026-10-31T12:00:00Z",
            severity: .warn,
            role: "Warehouse",
            entityId: "DEPOT-1",
            service: "wave-svc",
            message: "Should not be ingested while paused"
        )
        stream.pushEvent(droppedEvent)
        XCTAssertEqual(stream.events.count, initialSeedCount)
        XCTAssertFalse(stream.events.contains { $0.eventId == "EVT-DROPPED" })

        // Resume stream and verify JSON payload ingestion + malformed payload resilience
        stream.togglePause()
        XCTAssertFalse(stream.isPaused)

        let malformedPayload = Data("{not-valid-json".utf8)
        stream.ingestRawPayload(malformedPayload)
        XCTAssertEqual(stream.events.count, initialSeedCount, "Malformed JSON frames must fail safely without mutating ring buffer.")

        let validEvent = TelemetryLogEvent(
            eventId: "EVT-INGEST-1",
            traceId: "TRC-INGEST-99",
            timestampIso: "2026-10-31T12:01:00Z",
            severity: .critical,
            role: "Fleet",
            entityId: "FLT-99",
            service: "coldchain-monitor",
            message: "Reefer excursion threshold alert"
        )
        let validPayload = try JSONEncoder().encode(validEvent)
        stream.ingestRawPayload(validPayload)
        XCTAssertEqual(try XCTUnwrap(stream.events.first).eventId, "EVT-INGEST-1")
    }

    func testSpannerClusterMetricsAndRedisPoolSafety() throws {
        let jsonString = """
        {
            "p99ReadMs": 4.2,
            "p99WriteMs": 18.4,
            "p99CommitMs": 24.1,
            "activeSessions": 1840,
            "maxSessions": 2000,
            "staleReadOffloadPercent": 78.4,
            "hourlyHistory": []
        }
        """
        let json = try XCTUnwrap(jsonString.data(using: .utf8))

        let metrics = try JSONDecoder().decode(SpannerClusterMetrics.self, from: json)
        XCTAssertEqual(metrics.p99ReadMs, 4.2)
        XCTAssertEqual(metrics.activeSessions, 1840)
        XCTAssertEqual(metrics.staleReadOffloadPercent, 78.4)

        // Zero maxPermits division-by-zero guard
        let zeroPool = RedisPermitPool(poolName: "empty-pool", activePermits: 5, maxPermits: 0, queueWaitMs: 0.0)
        XCTAssertEqual(zeroPool.utilizationPercent, 0.0)

        let activePool = RedisPermitPool(poolName: "active-pool", activePermits: 25, maxPermits: 100, queueWaitMs: 1.1)
        XCTAssertEqual(activePool.utilizationPercent, 25.0)
    }

    @MainActor
    func testTelemetryHUDStringFormatting() async throws {
        let viewModel = ControlTowerViewModel(apiClient: makeMockAPIClient())
        XCTAssertTrue(viewModel.telemetryHUDString.contains("SPK-UZ: 4.2ms"))
        XCTAssertTrue(viewModel.telemetryHUDString.contains("WS SYNC"))

        await viewModel.initialLoad()
        viewModel.streamClient.connect()

        XCTAssertTrue(viewModel.telemetryHUDString.contains("SPK-UZ: 4.2ms"))
        XCTAssertTrue(viewModel.telemetryHUDString.contains("WS LIVE"))
        XCTAssertTrue(viewModel.telemetryHUDString.contains("CELLS: 3"))

        viewModel.streamClient.disconnect()
        XCTAssertFalse(viewModel.streamClient.isConnected)
        XCTAssertTrue(viewModel.telemetryHUDString.contains("WS SYNC"))
    }

    @MainActor
    func testObservableViewModelRoleSwitchingAndDLQTriage() async throws {
        let client = makeMockAPIClient()
        let stream = WebSocketStreamClient()
        let viewModel = ControlTowerViewModel(apiClient: client, streamClient: stream)
        await viewModel.initialLoad()
        XCTAssertNil(viewModel.lastErrorMessage)
        XCTAssertEqual(viewModel.cells.first?.cellId, "cell-uz-tashkent")

        // Verify @Observable property tracking triggers onChange when selectedSection mutates
        let mutationExpectation = expectation(description: "Observed selectedSection mutation")
        withObservationTracking {
            _ = viewModel.selectedSection
        } onChange: {
            mutationExpectation.fulfill()
        }

        viewModel.triageDLQInTelemetry()
        await fulfillment(of: [mutationExpectation], timeout: 1.0)
        XCTAssertEqual(viewModel.selectedSection, .telemetry)
        XCTAssertEqual(viewModel.selectedSeverity, .error)

        // Verify role switching updates waterfall slices for each perspective
        viewModel.selectedRole = .warehouse
        await viewModel.loadRoleSpecificData()
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "Allocated Depot GMV" })
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "Cold-Chain Energy Levy" })

        viewModel.selectedRole = .retailer
        await viewModel.loadRoleSpecificData()
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "Retail Invoiced Value" })
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "Cash-on-Delivery Fee" })

        viewModel.selectedRole = .fleet
        await viewModel.loadRoleSpecificData()
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "Freight Order Billable" })
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "DVIR Fuel Incentive" })

        viewModel.selectedRole = .supplier
        await viewModel.loadRoleSpecificData()
        XCTAssertTrue(viewModel.waterfallSlices.contains { $0.label == "Net Supplier Payout" })
    }

    @MainActor
    func testCombinedRoleSeverityAndQueryTelemetryFiltering() throws {
        let viewModel = ControlTowerViewModel(apiClient: makeMockAPIClient())
        let initialCount = viewModel.filteredTelemetryEvents.count
        XCTAssertEqual(initialCount, 5)

        // 1. Filter by Role (.supplier) -> 2 events (EVT-8921 info, EVT-8925 error)
        viewModel.selectedRole = .supplier
        XCTAssertEqual(viewModel.filteredTelemetryEvents.count, 2)
        XCTAssertTrue(viewModel.filteredTelemetryEvents.allSatisfy { $0.role == "Supplier" })

        // 2. Narrow by Severity (.error) -> 1 event (EVT-8925)
        viewModel.selectedSeverity = .error
        XCTAssertEqual(viewModel.filteredTelemetryEvents.count, 1)
        XCTAssertEqual(try XCTUnwrap(viewModel.filteredTelemetryEvents.first).eventId, "EVT-8925")

        // 3. Narrow further by Trace ID query ("TRC-FISCAL") -> 1 event
        viewModel.searchQuery = "TRC-FISCAL"
        XCTAssertEqual(viewModel.filteredTelemetryEvents.count, 1)

        // 4. Disjoint role (.fleet) with .error + "TRC-FISCAL" -> 0 events
        viewModel.selectedRole = .fleet
        XCTAssertTrue(viewModel.filteredTelemetryEvents.isEmpty)

        // 5. Reset all three axes -> restores all 5 seeded events
        viewModel.selectedRole = .overall
        viewModel.selectedSeverity = nil
        viewModel.searchQuery = ""
        XCTAssertEqual(viewModel.filteredTelemetryEvents.count, initialCount)
    }

    @MainActor
    func testURLInitializationSafetyAndAuthTokenIsolation() async throws {
        // Default non-optional URL resolution with zero force-unwraps
        let defaultClient = ControlTowerAPIClient(baseURL: nil)
        XCTAssertEqual(defaultClient.baseURL.absoluteString, "http://127.0.0.1:8180")

        let customBaseURL = try XCTUnwrap(URL(string: "https://noc.pegasusx.internal:8443"))
        let customClient = makeMockAPIClient(baseURL: customBaseURL)
        XCTAssertEqual(customClient.baseURL, customBaseURL)

        let initialToken = await customClient.currentAuthToken()
        XCTAssertNil(initialToken)
        await customClient.setAuthToken("pgx-session-token")
        let updatedToken = await customClient.currentAuthToken()
        XCTAssertEqual(updatedToken, "pgx-session-token")

        _ = try await customClient.fetchCellHealth()
        let captured = try XCTUnwrap(MockControlTowerURLProtocol.allRequests().last)
        XCTAssertEqual(captured.value(forHTTPHeaderField: "Authorization"), "Bearer pgx-session-token")
        XCTAssertEqual(captured.url?.path, "/v1/control-tower/cells")

        // Non-2xx HTTP status propagates error to ViewModel.lastErrorMessage
        MockControlTowerURLProtocol.requestHandler = { _ in
            (503, Data("{\"error\":\"service_unavailable\"}".utf8))
        }
        let vm = ControlTowerViewModel(apiClient: customClient, streamClient: WebSocketStreamClient())
        await vm.initialLoad()
        XCTAssertNotNil(vm.lastErrorMessage)

        let defaultStream = WebSocketStreamClient(url: nil)
        XCTAssertEqual(defaultStream.url.absoluteString, "ws://127.0.0.1:8180/ws/telemetry")

        let customStreamURL = try XCTUnwrap(URL(string: "wss://noc.pegasusx.internal:8443/ws/telemetry"))
        let customStream = WebSocketStreamClient(url: customStreamURL)
        XCTAssertEqual(customStream.url, customStreamURL)
    }

    func testKafkaDLQEscalation() async throws {
        let client = makeMockAPIClient()
        let partitions = try await client.fetchKafkaLags()

        let soliqTopic = try XCTUnwrap(partitions.first(where: { $0.topic == "compliance.fiscal.soliq" }))
        XCTAssertEqual(soliqTopic.dlqCount, 3, "Soliq topic must report 3 poison dead letters.")
        XCTAssertEqual(soliqTopic.status, "DEGRADED", "Topic with poison DLQ must be in DEGRADED status.")
    }

    func testFinancialMinorUnitsFormatting() {
        let slice = TenderWaterfallSlice(
            label: "Gross GMV",
            amountTiyin: 100_000_000_00,
            percentage: 100.0,
            isDeduction: false,
            detail: "Total GMV"
        )

        XCTAssertEqual(slice.amountTiyin, 10_000_000_000)
        XCTAssertTrue(
            slice.amountUzsFormatted.contains("100,000,000") ||
            slice.amountUzsFormatted.contains("100 000 000"),
            "Formatted string must contain correct integer UZS value."
        )
        XCTAssertTrue(slice.amountUzsFormatted.hasSuffix("UZS"))
    }

    @MainActor
    func testCheatsheetInspectorAndClipboardState() {
        let viewModel = ControlTowerViewModel(apiClient: makeMockAPIClient())
        XCTAssertFalse(viewModel.isCheatsheetPresented)
        XCTAssertTrue(viewModel.isInspectorPresented)

        viewModel.isCheatsheetPresented = true
        XCTAssertTrue(viewModel.isCheatsheetPresented)

        viewModel.isCheatsheetPresented = false
        XCTAssertFalse(viewModel.isCheatsheetPresented)

        viewModel.isInspectorPresented = false
        XCTAssertFalse(viewModel.isInspectorPresented)

        viewModel.copyToClipboard("TRC-SPANNER-999", label: "Trace ID")
        XCTAssertEqual(viewModel.copiedToastMessage, "Copied Trace ID to clipboard: TRC-SPANNER-999")
    }

    @MainActor
    func testHardenedURLValidationAndSignOutCleanup() async throws {
        XCTAssertTrue(ControlTowerAPIClient.isAllowedEndpointURL(try XCTUnwrap(URL(string: "https://api.pegasusx.app"))))
        XCTAssertTrue(ControlTowerAPIClient.isAllowedEndpointURL(try XCTUnwrap(URL(string: "http://127.0.0.1:8180"))))
        XCTAssertFalse(ControlTowerAPIClient.isAllowedEndpointURL(try XCTUnwrap(URL(string: "http://api.pegasusx.app"))))
        XCTAssertFalse(ControlTowerAPIClient.isAllowedEndpointURL(try XCTUnwrap(URL(string: "file:///etc/passwd"))))

        XCTAssertTrue(WebSocketStreamClient.isAllowedStreamURL(try XCTUnwrap(URL(string: "wss://api.pegasusx.app/ws/telemetry"))))
        XCTAssertTrue(WebSocketStreamClient.isAllowedStreamURL(try XCTUnwrap(URL(string: "ws://localhost:8180/ws/telemetry"))))
        XCTAssertFalse(WebSocketStreamClient.isAllowedStreamURL(try XCTUnwrap(URL(string: "ws://api.pegasusx.app/ws/telemetry"))))

        let insecureClient = makeMockAPIClient(baseURL: try XCTUnwrap(URL(string: "http://api.pegasusx.app")))
        do {
            _ = try await insecureClient.fetchCellHealth()
            XCTFail("Expected non-loopback HTTP endpoint to be rejected")
        } catch let err as URLError {
            XCTAssertEqual(err.code, .unsupportedURL)
        }

        let apiClient = makeMockAPIClient()
        await apiClient.setAuthToken("secret-admin-jwt")
        let vm = ControlTowerViewModel(apiClient: apiClient, streamClient: WebSocketStreamClient())
        await vm.initialLoad()
        XCTAssertFalse(vm.cells.isEmpty)
        await vm.signOut()
        let cleared = await apiClient.currentAuthToken()
        XCTAssertNil(cleared)
        XCTAssertTrue(vm.cells.isEmpty)
        XCTAssertTrue(vm.streamClient.events.isEmpty)
        XCTAssertFalse(vm.streamClient.isConnected)
    }
}

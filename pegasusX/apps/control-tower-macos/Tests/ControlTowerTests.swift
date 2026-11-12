import Observation
import XCTest

final class ControlTowerTests: XCTestCase {

    func testTenderWaterfallBalanceInvariant() async throws {
        let client = ControlTowerAPIClient()
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
        let client = ControlTowerAPIClient()
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
        let viewModel = ControlTowerViewModel()
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
        let client = ControlTowerAPIClient()
        let stream = WebSocketStreamClient()
        let viewModel = ControlTowerViewModel(apiClient: client, streamClient: stream)
        await viewModel.initialLoad()

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
        let viewModel = ControlTowerViewModel()
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
        let customClient = ControlTowerAPIClient(baseURL: customBaseURL)
        XCTAssertEqual(customClient.baseURL, customBaseURL)

        let initialToken = await customClient.currentAuthToken()
        XCTAssertNil(initialToken)
        await customClient.setAuthToken("pgx-session-token")
        let updatedToken = await customClient.currentAuthToken()
        XCTAssertEqual(updatedToken, "pgx-session-token")

        let defaultStream = WebSocketStreamClient(url: nil)
        XCTAssertEqual(defaultStream.url.absoluteString, "ws://127.0.0.1:8180/ws/telemetry")

        let customStreamURL = try XCTUnwrap(URL(string: "wss://noc.pegasusx.internal:8443/ws/telemetry"))
        let customStream = WebSocketStreamClient(url: customStreamURL)
        XCTAssertEqual(customStream.url, customStreamURL)
    }

    func testKafkaDLQEscalation() async throws {
        let client = ControlTowerAPIClient()
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
        let viewModel = ControlTowerViewModel()
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
}

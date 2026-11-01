import XCTest

final class ControlTowerTests: XCTestCase {

    func testTenderWaterfallBalanceInvariant() async throws {
        let client = ControlTowerAPIClient()
        let slices = try await client.fetchWaterfall(for: .overall)

        guard let gmv = slices.first(where: { $0.label == "Gross GMV" }),
              let netPayout = slices.first(where: { $0.label == "Net Supplier Payout" }) else {
            XCTFail("Missing GMV or Net Payout in waterfall")
            return
        }

        let deductions = slices.filter { $0.isDeduction }.reduce(0) { $0 + $1.amountTiyin }
        let calculatedNet = gmv.amountTiyin - deductions

        XCTAssertEqual(calculatedNet, netPayout.amountTiyin, "Net payout must strictly equal Gross GMV minus total deductions.")
        XCTAssertGreaterThan(netPayout.amountTiyin, 0, "Disbursable net payout must be strictly positive.")
    }

    func testAttributedCostsFiltering() async throws {
        let client = ControlTowerAPIClient()
        let costs = try await client.fetchAttributedCosts()

        XCTAssertFalse(costs.isEmpty, "Attributed costs list cannot be empty.")
        let supplierCosts = costs.filter { $0.role == "Supplier" }
        XCTAssertEqual(supplierCosts.count, 2, "Supplier must have Spanner and GTIN costs attributed.")

        let warehouseCosts = costs.filter { $0.role == "Warehouse" }
        XCTAssertEqual(warehouseCosts.count, 2, "Warehouse must have Wave and Cold-Chain costs attributed.")
    }

    @MainActor
    func testTelemetryRingBufferMaxCapacity() throws {
        let stream = WebSocketStreamClient()
        // Push 60 events, exceeding max capacity of 50
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
        XCTAssertEqual(stream.events.first?.eventId, "EVT-60", "Most recent event must be at index 0.")
        XCTAssertEqual(stream.events.last?.eventId, "EVT-11", "Earliest preserved event must be EVT-11.")
    }

    func testSpannerClusterMetricsDecoding() throws {
        let json = """
        {
            "p99ReadMs": 4.2,
            "p99WriteMs": 18.4,
            "p99CommitMs": 24.1,
            "activeSessions": 1840,
            "maxSessions": 2000,
            "staleReadOffloadPercent": 78.4,
            "hourlyHistory": []
        }
        """.data(using: .utf8)!

        let metrics = try JSONDecoder().decode(SpannerClusterMetrics.self, from: json)
        XCTAssertEqual(metrics.p99ReadMs, 4.2)
        XCTAssertEqual(metrics.activeSessions, 1840)
        XCTAssertEqual(metrics.staleReadOffloadPercent, 78.4)
    }
}

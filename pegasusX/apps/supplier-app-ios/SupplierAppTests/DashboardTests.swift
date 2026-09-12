import XCTest
@testable import SupplierAppIOS

final class DashboardTests: XCTestCase {
    func testDashboardParsesOrdersByStatus() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true,
          "inventory_skus": 150,
          "pending_orders": 25,
          "updated_at": "2026-06-30T10:00:00Z",
          "orders_by_status": { "PENDING": 3, "COMPLETED": 1 }
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertEqual(dash.ordersByStatus["PENDING"], 3)
        XCTAssertEqual(dash.ordersByStatus["COMPLETED"], 1)
        XCTAssertEqual(orderStatusFunnel.count, 17)
    }

    func testDashboardMissingOrdersByStatusIsEmptyMap() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true,
          "inventory_skus": 150,
          "pending_orders": 25,
          "updated_at": "2026-06-30T10:00:00Z"
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertTrue(dash.ordersByStatus.isEmpty)
    }

    func testStatusStackModesStayDistinct() {
        let empty = statusStackModel(counts: nil)
        XCTAssertEqual(empty.mode, .empty)
        XCTAssertTrue(empty.rows.isEmpty)

        let zero = statusStackModel(counts: [:])
        XCTAssertEqual(zero.mode, .zero)
        XCTAssertEqual(zero.rows.count, 17)
        XCTAssertTrue(zero.rows.allSatisfy { $0.count == 0 })

        let unavailable = statusStackModel(counts: ["PENDING": 9], available: false)
        XCTAssertEqual(unavailable.mode, .unavailable)
        XCTAssertEqual(unavailable.rows.count, 17)
        XCTAssertTrue(unavailable.rows.allSatisfy { $0.count == nil })

        let live = statusStackModel(counts: ["PENDING": 2, "COMPLETED": 2])
        XCTAssertEqual(live.mode, .live)
        XCTAssertEqual(live.total, 4)
        XCTAssertEqual(live.rows.count, 17)
        XCTAssertEqual(live.rows.first { $0.key == "PENDING" }?.share, 0.5)
        XCTAssertEqual(live.rows.first { $0.key == "LOADED" }?.count, 0)
    }

    func testFiscalFailedIncrementsChip() {
        let next = incrementOrderStatusCount([:], status: "FISCAL_FAILED")
        XCTAssertEqual(next["FISCAL_FAILED"], 1)
        XCTAssertEqual(next.count, 17)
        XCTAssertEqual(incrementOrderStatusCount(next, status: "DISPATCHED")["LOADED"], 1)
    }

    func testDashboardParsesRevenueWithoutInventingCurrency() throws {
        let json = Data("""
        {
          "supplier_id": "sup_123",
          "is_configured": true,
          "inventory_skus": 150,
          "pending_orders": 25,
          "updated_at": "2026-06-30T10:00:00Z",
          "orders_by_status": { "FISCAL_FAILED": 1 },
          "today_revenue_minor": 1500
        }
        """.utf8)
        let dash = try JSONDecoder().decode(SupplierDashboard.self, from: json)
        XCTAssertEqual(dash.todayRevenueMinor, 1500)
        XCTAssertEqual(dash.ordersByStatus["FISCAL_FAILED"], 1)
        XCTAssertEqual(formatPackMoney(dash.todayRevenueMinor, pack: nil), "15")
        XCTAssertFalse(formatPackMoney(dash.todayRevenueMinor, pack: nil).contains("UZS"))
    }

    func testCommandChipUsesFunnelStatusNotCoarseTab() {
        let fiscal = resolveSupplierOrdersQuery(commandStatus: "FISCAL_FAILED", coarseFilter: "ACTIVE")
        XCTAssertEqual(fiscal.status, "FISCAL_FAILED")
        XCTAssertNil(fiscal.filter)
        let dispatched = resolveSupplierOrdersQuery(commandStatus: "DISPATCHED", coarseFilter: "ACTIVE")
        XCTAssertEqual(dispatched.status, "LOADED")
        let coarse = resolveSupplierOrdersQuery(commandStatus: nil, coarseFilter: "ACTIVE")
        XCTAssertEqual(coarse.filter, "ACTIVE")
        XCTAssertNil(coarse.status)
    }

    func testPulseHonestyKeepsPreviousOnFailure() {
        let previous = ["old"]
        let failed = PulseHonesty.apply(ok: false, incoming: nil as [String]?, previous: previous)
        XCTAssertEqual(failed.events, previous)
        XCTAssertEqual(failed.error, PulseHonesty.failed)
        let empty = PulseHonesty.apply(ok: true, incoming: [String](), previous: previous)
        XCTAssertEqual(empty.events, [])
        XCTAssertNil(empty.error)
    }
}

import XCTest
@testable import WarehouseAppIOS

final class DashboardTests: XCTestCase {
    func testFullStacksAndDemandSource() throws {
        let raw = """
        {
          "pending_dispatch": 2,
          "orders_by_status": { "PENDING": 2, "FISCAL_FAILED": 1 },
          "truck_duty": {
            "AVAILABLE": 1,
            "OFF_SHIFT": 1,
            "RETURNING_TO_WAREHOUSE": 1,
            "UNASSIGNED": 1,
            "VEHICLE_INACTIVE": 1
          },
          "hold_reasons": [{ "code": "MAINTENANCE", "count": 1 }],
          "demand_source": "empty",
          "history_available": false
        }
        """.data(using: .utf8)!
        let dash = try JSONDecoder().decode(DashboardData.self, from: raw)
        XCTAssertEqual(dash.ordersByStatus["FISCAL_FAILED"], 1)
        XCTAssertEqual(dash.truckDuty["VEHICLE_INACTIVE"], 1)
        XCTAssertEqual(dash.holdReasons.first?.code, "MAINTENANCE")
        XCTAssertEqual(dash.demandSource, "empty")
        XCTAssertFalse(dash.historyAvailable)
        XCTAssertTrue(truckDutyStatuses.contains("OFF_SHIFT"))
        XCTAssertEqual(incrementOrderStatusCount(dash.ordersByStatus, status: "FISCAL_FAILED")["FISCAL_FAILED"], 2)
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

import XCTest
@testable import FactoryAppIOS

final class DashboardTests: XCTestCase {
    func testFactoryCommandStacksAndSource() throws {
        let raw = """
        {
          "source": "spanner",
          "plane": "factory_trucks",
          "pending_transfers": 1,
          "transfers_by_state": { "CREATED": 1, "CANCELLED": 1 },
          "manifests_by_state": { "LOADING": 1, "SEALED": 1 },
          "vehicles_by_state": { "READY": 1, "UNAVAILABLE": 1 },
          "driver_duty": { "ON_SHIFT": 1, "OFF_SHIFT": 1 },
          "qc_available": true,
          "qc_by_result": { "FAIL": 1 }
        }
        """.data(using: .utf8)!
        let dash = try JSONDecoder().decode(DashboardStats.self, from: raw)
        XCTAssertEqual(dash.source, "spanner")
        XCTAssertEqual(dash.plane, "factory_trucks")
        XCTAssertEqual(dash.transfersByState["CREATED"], 1)
        XCTAssertEqual(dash.manifestsByState["SEALED"], 1)
        XCTAssertEqual(dash.vehiclesByState["UNAVAILABLE"], 1)
        XCTAssertEqual(dash.driverDuty["OFF_SHIFT"], 1)
        XCTAssertTrue(factoryTransferStates.contains("PENDING"))
        XCTAssertTrue(manifestStates.contains("DRAFT"))
        XCTAssertEqual(statusStackModel(dictionary: factoryTransferStates, counts: dash.transfersByState).rows.count, factoryTransferStates.count)
        XCTAssertFalse(factoryTransferStates.contains("FISCAL_FAILED"))
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

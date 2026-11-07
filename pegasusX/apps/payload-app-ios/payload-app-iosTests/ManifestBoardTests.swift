import XCTest
@testable import payload_app_ios

final class ManifestBoardTests: XCTestCase {
    func testBoardHasFourManifestStateColumns() {
        let trucks = [
            Truck.fixture(id: "d", truckStatus: "DRAFT"),
            Truck.fixture(id: "l", truckStatus: "LOADING"),
            Truck.fixture(id: "s", truckStatus: "SEALED"),
            Truck.fixture(id: "x", truckStatus: "DISPATCHED"),
            Truck.fixture(id: "done", truckStatus: "COMPLETED"),
            Truck.fixture(id: "none", truckStatus: ""),
        ]
        let cols = ManifestBoard.group(trucks)
        XCTAssertEqual(cols.map(\.state), ["DRAFT", "LOADING", "SEALED", "DISPATCHED"])
        XCTAssertEqual(cols[0].trucks.map(\.id), ["d"])
        XCTAssertEqual(cols[1].trucks.map(\.id), ["l"])
        XCTAssertEqual(cols[2].trucks.map(\.id), ["s"])
        XCTAssertEqual(cols[3].trucks.map(\.id), ["x"])
        XCTAssertEqual(ManifestBoard.unassigned(trucks).map(\.id), ["done", "none"])
    }

    func testAttachDoesNotInventDraftFromCompleted() {
        let trucks = [Truck.fixture(id: "veh-1"), Truck.fixture(id: "veh-2")]
        let manifests = [
            Manifest(manifestId: "m1", vehicleId: "veh-1", state: "SEALED", totalVolumeVu: 8, maxVolumeVu: 40, stopCount: 2),
            Manifest(manifestId: "m2", vehicleId: "veh-2", state: "COMPLETED"),
        ]
        let attached = ManifestBoard.attach(trucks: trucks, manifests: manifests)
        XCTAssertEqual(attached[0].truckStatus, "SEALED")
        XCTAssertEqual(attached[1].truckStatus ?? "", "")
        let cols = ManifestBoard.group(attached)
        XCTAssertEqual(cols[2].trucks.count, 1)
        XCTAssertTrue(cols[0].trucks.isEmpty)
    }

    func testEmptyBoardIsEmptyColumns() {
        let cols = ManifestBoard.group([])
        XCTAssertEqual(cols.count, 4)
        XCTAssertTrue(cols.allSatisfy(\.trucks.isEmpty))
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

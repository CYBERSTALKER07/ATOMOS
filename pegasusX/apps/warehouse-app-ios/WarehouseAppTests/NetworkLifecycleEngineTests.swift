import XCTest
@testable import WarehouseAppIOS

@MainActor
final class NetworkLifecycleEngineTests: XCTestCase {

    func testOfflineStateTransitions() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        XCTAssertFalse(engine.isOnline)
        XCTAssertEqual(engine.state, .offline)
    }

    func testFlappingHysteresisEntersCooldown() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        engine.setOnlineForTesting(true)
        engine.setOnlineForTesting(false)

        XCTAssertEqual(engine.state, .flapping)
        XCTAssertFalse(engine.isOnline)
    }

    func testReconnectCoordinatesEpochRefresh() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        let initialEpoch = engine.syncEpoch

        engine.setOnlineForTesting(true)
        await engine.triggerSyncEpoch()

        XCTAssertEqual(engine.syncEpoch, initialEpoch + 1)
        XCTAssertTrue(engine.isOnline)
    }
}

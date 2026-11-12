import Testing
import Foundation
@testable import retailerapp

@MainActor
struct NetworkLifecycleEngineTests {

    @Test func offlineStateTransitions() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        #expect(engine.isOnline == false)
        #expect(engine.state == .offline)
    }

    @Test func flappingHysteresisEntersCooldown() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        // 3 rapid state toggles
        engine.setOnlineForTesting(false)
        engine.setOnlineForTesting(true)
        engine.setOnlineForTesting(false)

        #expect(engine.state == .flapping)
        #expect(engine.isOnline == false)
    }

    @Test func reconnectCoordinatesEpochRefresh() async {
        let engine = NetworkLifecycleEngine.shared
        engine.resetForTesting()

        engine.setOnlineForTesting(false)
        let initialEpoch = engine.syncEpoch

        engine.setOnlineForTesting(true)
        await engine.triggerSyncEpoch()

        #expect(engine.syncEpoch == initialEpoch + 1)
        #expect(engine.isOnline == true)
    }
}

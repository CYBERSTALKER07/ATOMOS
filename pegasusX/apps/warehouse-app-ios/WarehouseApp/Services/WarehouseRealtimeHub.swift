import Foundation
import Observation

/// Cross-screen refresh epoch driven by WebSocket operational events.
@Observable
final class WarehouseRealtimeHub {
    var refreshEpoch: Int = 0
    var reconnectEpoch: Int = 0

    func bump() {
        refreshEpoch += 1
    }

    func bumpReconnect() {
        reconnectEpoch += 1
        bump()
    }
}

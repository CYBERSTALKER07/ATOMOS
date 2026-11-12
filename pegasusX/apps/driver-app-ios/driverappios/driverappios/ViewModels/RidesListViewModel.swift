//
//  RidesListViewModel.swift
//  driverappios
//

import Foundation
import Observation

@Observable
@MainActor
final class RidesListViewModel {
    var loadingMode = false
    var showEarlyComplete = false
    var isRequestingEarlyComplete = false
    var earlyCompleteMessage: String?

    private let sendEarlyComplete: @Sendable (String, String) async throws -> Void

    init(
        sendEarlyComplete: @escaping @Sendable (String, String) async throws -> Void = { reason, note in
            _ = try await APIClient.shared.requestEarlyComplete(reason: reason, note: note)
        }
    ) {
        self.sendEarlyComplete = sendEarlyComplete
    }

    func submitEarlyComplete(reason: String, note: String, reloadMissions: () async -> Void) async {
        isRequestingEarlyComplete = true
        earlyCompleteMessage = nil
        defer { isRequestingEarlyComplete = false }
        do {
            try await sendEarlyComplete(reason, note)
            Haptics.success()
            earlyCompleteMessage = "Early complete request submitted."
            await reloadMissions()
        } catch {
            Haptics.error()
            earlyCompleteMessage = error.localizedDescription
        }
    }

    func handleReconnect(reloadMissions: () async -> Void) async {
        let hadInFlight = isRequestingEarlyComplete
        if isRequestingEarlyComplete {
            isRequestingEarlyComplete = false
            earlyCompleteMessage = DriverReconnectRecovery.hint
        }
        await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
        await reloadMissions()
    }

    func loadSeqLabel(for index: Int, total: Int) -> String? {
        guard loadingMode, total > 0 else { return nil }
        if index == 0 { return "Load #\(index + 1) · Back of Truck" }
        if index == total - 1 { return "Load #\(index + 1) · By the Doors" }
        return "Load #\(index + 1)"
    }
}

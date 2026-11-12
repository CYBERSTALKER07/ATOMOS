//
//  DriverHomeViewModel.swift
//  driverappios
//

import Foundation
import Observation
import SwiftData

@Observable
@MainActor
final class DriverHomeViewModel {
    var showNotificationInbox = false
    var showSupplyTransfers = false
    var showRescueSheet = false
    var showSyncQueue = false
    var showQRScanner = false
    var showOfflineVerifier = false
    var selectedRecentMission: Mission?

    var showHandoffAlert = false
    var handoffAlertMessage: String?

    var pulseEvents: [PulseEvent] = []
    var pulseLoading = true
    var pulseError: String?
    var offlinePending = 0

    private let fetchPulse: @Sendable () async throws -> PulseResponse

    init(
        fetchPulse: @escaping @Sendable () async throws -> PulseResponse = {
            try await APIClient.shared.getPulse()
        }
    ) {
        self.fetchPulse = fetchPulse
    }

    func loadPulse() async {
        pulseLoading = true
        pulseError = nil
        defer { pulseLoading = false }
        do {
            let response = try await fetchPulse()
            let result = PulseHonesty.apply(ok: true, incoming: response.events, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        } catch {
            let result = PulseHonesty.apply(ok: false, incoming: nil, previous: pulseEvents)
            pulseEvents = result.events
            pulseError = result.error
        }
    }

    func refreshOfflinePending(container: ModelContainer) {
        DriverOfflineQueue.shared.attach(container: container)
        offlinePending = DriverOfflineQueue.shared.pendingCount()
    }

    func handleHandoffLink(_ link: String, onNavigate: ((HandoffDestination) -> Void)?) {
        switch HandoffPathResolver.resolve(link) {
        case .home:
            onNavigate?(.home)
        case .fleetMap:
            onNavigate?(.fleetMap)
        case .manifestList, .manifestDetail, .orderDetail:
            onNavigate?(.manifestList)
        case .unresolved:
            handoffAlertMessage = "Open in portal — no native route for this link"
            showHandoffAlert = true
        }
    }

    func clearHandoffAlert() {
        showHandoffAlert = false
        handoffAlertMessage = nil
    }

    func greetingText(for date: Date = Date(), calendar: Calendar = .current) -> String {
        let hour = calendar.component(.hour, from: date)
        switch hour {
        case 5..<12:  return "GOOD MORNING"
        case 12..<17: return "GOOD AFTERNOON"
        case 17..<21: return "GOOD EVENING"
        default:      return "GOOD NIGHT"
        }
    }
}

//
//  RequestRescueViewModel.swift
//  driverappios
//

import Foundation
import Observation

@Observable
@MainActor
final class RequestRescueViewModel {
    static let availableReasons = ["Engine Failure", "Flat Tire", "Accident", "Other"]

    var reason: String = "Engine Failure"
    var note: String = ""
    var isSubmitting = false
    var errorMessage: String?

    private let fleetService: FleetServiceProtocol

    init(fleetService: FleetServiceProtocol = FleetServiceLive.shared) {
        self.fleetService = fleetService
    }

    func submitRescue() async -> Bool {
        isSubmitting = true
        errorMessage = nil
        defer { isSubmitting = false }
        do {
            try await fleetService.requestRescue(reason: reason, note: note)
            Haptics.success()
            return true
        } catch {
            Haptics.error()
            errorMessage = error.localizedDescription
            return false
        }
    }
}

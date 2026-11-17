//
//  YardGateViewModel.swift
//  driverappios
//

import Foundation
import Observation

@Observable
@MainActor
final class YardGateViewModel {

    var depotId: String = "depot-tashkent-south"
    var supplierId: String = "supplier-tashkent-1"
    var vehicleId: String = "vh-isuzu-01"
    var licensePlate: String = "01 777 AAA"
    var direction: String = "GATE_OUT"
    var vehiclePurpose: String = "OUTBOUND_ROUTE"
    var boltSealBarcode: String = "SEAL-UZ-44912"
    var sealIntact: Bool = true
    var assignedBaySlot: String = "OUTBOUND_LOADING"

    var isSubmitting: Bool = false
    var recordedEvent: YardGateCheckEvent? = nil
    var statusMessage: String? = nil
    var errorMessage: String? = nil

    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func updateBaySlot() {
        assignedBaySlot = sealIntact
            ? (direction == "GATE_OUT" ? "OUTBOUND_LOADING" : "BAY_3_INSPECTION")
            : "SECURITY_QUARANTINE_BAY"
    }

    func submitGateCheck() async {
        isSubmitting = true
        errorMessage = nil
        statusMessage = nil

        let event = YardGateCheckEvent(
            depotId: depotId,
            supplierId: supplierId,
            vehicleId: vehicleId,
            licensePlate: licensePlate,
            direction: direction,
            vehiclePurpose: vehiclePurpose,
            boltSealBarcode: boltSealBarcode,
            sealIntact: sealIntact,
            assignedBaySlot: assignedBaySlot
        )

        do {
            let res = try await apiClient.recordGateEvent(event: event)
            isSubmitting = false
            recordedEvent = res
            statusMessage = "Gate Event Logged: \(direction) for \(licensePlate) -> Bay: \(assignedBaySlot)"
        } catch {
            isSubmitting = false
            errorMessage = error.localizedDescription
        }
    }
}

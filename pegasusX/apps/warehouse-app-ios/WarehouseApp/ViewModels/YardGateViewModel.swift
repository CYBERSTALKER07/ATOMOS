//
//  YardGateViewModel.swift
//  WarehouseAppIOS
//

import Foundation
import Observation

struct YardVehiclePreset: Identifiable, Sendable, Equatable {
    let id: String
    let plate: String
    let vehicleId: String
    let label: String
}

@Observable
@MainActor
final class YardGateViewModel {
    var depotId: String = "depot-tashkent-south"
    var supplierId: String = "supplier-tashkent-1"
    var vehicleId: String = "vh-isuzu-01"
    var licensePlate: String = "01 777 AAA"
    var direction: String = "GATE_IN"
    var vehiclePurpose: String = "INBOUND_SUPPLIER_DROP"
    var boltSealBarcode: String = "SEAL-UZ-11029"
    var sealIntact: Bool = true
    var assignedBaySlot: String = "BAY_3_INSPECTION"

    var isSubmitting: Bool = false
    var recordedEvent: YardGateCheckEvent? = nil
    var statusMessage: String? = nil
    var errorMessage: String? = nil

    let vehiclePresets: [YardVehiclePreset] = [
        YardVehiclePreset(id: "vh-isuzu-01", plate: "01 777 AAA", vehicleId: "vh-isuzu-01", label: "Isuzu NPR (Primary)"),
        YardVehiclePreset(id: "vh-man-04", plate: "01 882 BBB", vehicleId: "vh-man-04", label: "MAN TGM Reefer"),
        YardVehiclePreset(id: "vh-hyundai-09", plate: "20 441 CCC", vehicleId: "vh-hyundai-09", label: "Hyundai HD78")
    ]

    let sealPresets = ["SEAL-UZ-11029", "SEAL-UZ-88123", "SEAL-UZ-99412"]

    var isOutboundGateBlocked: Bool {
        direction == "GATE_OUT" && !sealIntact
    }

    private let recordGateEventHandler: @Sendable (YardGateCheckEvent) async throws -> YardGateCheckEvent

    init(
        recordGateEventHandler: @escaping @Sendable (YardGateCheckEvent) async throws -> YardGateCheckEvent = { event in
            try await WarehouseService.recordGateEvent(event: event)
        }
    ) {
        self.recordGateEventHandler = recordGateEventHandler
    }

    func applyPreset(_ preset: YardVehiclePreset) {
        licensePlate = preset.plate
        vehicleId = preset.vehicleId
    }

    func updateSealIntegrity(_ intact: Bool) {
        sealIntact = intact
        assignedBaySlot = intact ? "BAY_3_INSPECTION" : "SECURITY_QUARANTINE_BAY"
    }

    func submitGateEvent() async {
        guard !isOutboundGateBlocked else {
            errorMessage = "Outbound gate exit is hard-blocked when bolt seal is tampered."
            PegasusHaptics.error()
            return
        }

        isSubmitting = true
        errorMessage = nil
        statusMessage = nil

        let evt = YardGateCheckEvent(
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
            let res = try await recordGateEventHandler(evt)
            isSubmitting = false
            recordedEvent = res
            statusMessage = "Gate Event Logged: \(direction) -> \(assignedBaySlot)"
            PegasusHaptics.success()
        } catch {
            isSubmitting = false
            errorMessage = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}

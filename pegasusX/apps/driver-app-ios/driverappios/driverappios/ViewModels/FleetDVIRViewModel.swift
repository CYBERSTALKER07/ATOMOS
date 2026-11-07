//
//  FleetDVIRViewModel.swift
//  driverappios
//

import Foundation
import SwiftUI

@Observable
final class FleetDVIRViewModel {

    var supplierId: String = "supplier-tashkent-1"
    var vehicleId: String = "vh-isuzu-01"
    var driverId: String = "driver-uz-09"
    var odometerKm: Int64 = 125400
    var requiresColdChain: Bool = true
    var reeferTempCelsius: Double = 3.5

    var checklist = DVIRChecklist8Point()

    var isSubmitting: Bool = false
    var isRedTagQuarantined: Bool = false
    var redTagReason: String? = nil
    var workOrderId: String? = nil
    var successMessage: String? = nil
    var errorMessage: String? = nil

    private let apiClient: APIClient

    init(apiClient: APIClient = .shared) {
        self.apiClient = apiClient
    }

    func toggleSteeringAndHorn() {
        checklist.steeringAndHornOk.toggle()
    }

    func toggleBrakes() {
        checklist.serviceAndParkingBrakesOk.toggle()
    }

    func toggleTires() {
        checklist.tiresTreadAndPressureOk.toggle()
    }

    func toggleLights() {
        checklist.lightsAndHazardsOk.toggle()
    }

    func toggleWipers() {
        checklist.wipersAndFluidsOk.toggle()
    }

    func toggleEmergencyKit() {
        checklist.emergencyKitAndExtinguisherOk.toggle()
    }

    func toggleCargoSanitation() {
        checklist.cargoBoxSanitationOk.toggle()
    }

    func toggleReeferCalibration() {
        checklist.reeferTempCalibratedOk.toggle()
    }

    var failedPoints: [String] {
        var list: [String] = []
        if !checklist.steeringAndHornOk { list.append("steering_and_horn") }
        if !checklist.serviceAndParkingBrakesOk { list.append("service_and_parking_brakes") }
        if !checklist.tiresTreadAndPressureOk { list.append("tires_tread_and_pressure") }
        if !checklist.lightsAndHazardsOk { list.append("lights_and_hazards") }
        if !checklist.wipersAndFluidsOk { list.append("wipers_and_fluids") }
        if !checklist.emergencyKitAndExtinguisherOk { list.append("emergency_kit_and_extinguisher") }
        if !checklist.cargoBoxSanitationOk { list.append("cargo_box_sanitation") }
        if requiresColdChain && !checklist.reeferTempCalibratedOk { list.append("reefer_temp_calibrated") }
        return list
    }

    var hasCriticalDefect: Bool {
        !failedPoints.isEmpty
    }

    @MainActor
    func submitInspection() async {
        isSubmitting = true
        errorMessage = nil
        successMessage = nil

        let defects = failedPoints
        let defectFound = !defects.isEmpty
        let status = defectFound ? "RED_TAG_QUARANTINE" : "CLEARED_FOR_DISPATCH"
        let generatedWO = defectFound ? "WO-\(vehicleId)-\(Int.random(in: 1000...9999))" : nil

        let inspection = FleetDVIRInspection(
            supplierId: supplierId,
            vehicleId: vehicleId,
            driverId: driverId,
            odometerKm: odometerKm,
            requiresColdChain: requiresColdChain,
            reeferTempCelsius: requiresColdChain ? reeferTempCelsius : nil,
            checklist: checklist,
            failedPoints: defects,
            criticalDefectFound: defectFound,
            redTagApplied: defectFound,
            status: status,
            maintenanceWorkOrderId: generatedWO
        )

        do {
            let result = try await apiClient.submitDVIR(inspection: inspection)
            isSubmitting = false
            if result.redTagApplied || result.status == "RED_TAG_QUARANTINE" {
                isRedTagQuarantined = true
                workOrderId = result.maintenanceWorkOrderId ?? generatedWO
                redTagReason = "CRITICAL DEFECT DETECTED: \(defects.joined(separator: ", ")). Vehicle quarantined and driver pairing revoked."
            } else {
                isRedTagQuarantined = false
                successMessage = "Vehicle Cleared for Dispatch! Inspection record saved."
            }
        } catch let apiError as APIError {
            isSubmitting = false
            switch apiError {
            case .httpError(let code) where code == 409:
                isRedTagQuarantined = true
                workOrderId = generatedWO
                redTagReason = "RED-TAG LOCKOUT (409): Inspection failed. Grounding protocol enforced."
            default:
                errorMessage = "Failed to submit inspection: \(apiError)"
            }
        } catch {
            isSubmitting = false
            errorMessage = error.localizedDescription
        }
    }
}

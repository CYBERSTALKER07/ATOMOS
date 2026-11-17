//
//  FleetMapViewModel.swift
//  driverappios
//

import Foundation
import MapKit
import Observation
import SwiftUI

@Observable
@MainActor
final class FleetMapViewModel {
    var navPath = NavigationPath()
    var cameraPosition: MapCameraPosition = .userLocation(followsHeading: true, fallback: .automatic)
    var isCameraLocked: Bool = false
    var userPannedAt: Date?

    var phase: MapPhase = .pickingOrder
    var selectedMission: Mission?
    var zoomFocus: ZoomFocus = .both
    var validatedQR: ValidateQRResponse?
    var scannedQRToken: String = ""
    var offloadResponse: ConfirmOffloadResponse?
    var showRescueSheet = false

    func resetNavigationAndReturnToPicker() {
        navPath = NavigationPath()
        phase = .pickingOrder
        selectedMission = nil
    }

    func handleValidatedQR(_ response: ValidateQRResponse, token: String) {
        validatedQR = response
        scannedQRToken = token
        navPath.append("offload-review")
    }

    func handleConfirmedOffload(_ result: ConfirmOffloadResponse) {
        offloadResponse = result
        navPath.append("payment-waiting")
    }

    func handleIdleRecenter(for pannedAt: Date?) async {
        guard let pannedAt, phase == .activeDelivery else { return }
        try? await Task.sleep(nanoseconds: MapCameraConfig.idleRecenterMs)
        guard !Task.isCancelled else { return }
        if userPannedAt == pannedAt {
            isCameraLocked = true
            userPannedAt = nil
        }
    }
}

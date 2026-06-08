//
//  ScannerViewModel.swift
//  driverappios
//

import AVFoundation
import SwiftUI

@Observable
@MainActor
final class ScannerViewModel {

    // MARK: - State

    var isProcessing = false
    var alertTitle = ""
    var alertMessage = ""
    var showAlert = false
    var scanSucceeded = false
    var validatedResponse: ValidateQRResponse?

    private var scanLocked = false
    private let fleetService: FleetServiceProtocol

    // MARK: - Init

    convenience init() {
        self.init(fleetService: FleetServiceLive.shared)
    }

    init(fleetService: FleetServiceProtocol) {
        self.fleetService = fleetService
    }

    // MARK: - Camera Permission

    func checkCameraPermission() async -> Bool {
        switch AVCaptureDevice.authorizationStatus(for: .video) {
        case .authorized:
            return true
        case .notDetermined:
            return await AVCaptureDevice.requestAccess(for: .video)
        default:
            return false
        }
    }

    // MARK: - Handle QR Scan → Validate Only (no state change)

    func handleScan(_ stringValue: String, onValidated: @escaping (ValidateQRResponse) -> Void) {
        guard !scanLocked else { return }
        scanLocked = true
        isProcessing = true

        Task {
            try? await Task.sleep(nanoseconds: 2_000_000_000)
            scanLocked = false
        }

        guard let data = stringValue.data(using: .utf8),
              let payload = try? JSONDecoder().decode(QRPayload.self, from: data) else {
            isProcessing = false
            alertTitle = "Scan Error"
            alertMessage = "Invalid QR code format."
            showAlert = true
            Haptics.error()
            return
        }

        Task {
            do {
                let response = try await fleetService.validateQR(orderId: payload.order_id, scannedToken: payload.token)
                isProcessing = false
                scanSucceeded = true
                validatedResponse = response
                Haptics.success()
                onValidated(response)
            } catch {
                isProcessing = false
                alertTitle = "QR Validation Failed"
                alertMessage = error.localizedDescription
                showAlert = true
                Haptics.error()
            }
        }
    }
}

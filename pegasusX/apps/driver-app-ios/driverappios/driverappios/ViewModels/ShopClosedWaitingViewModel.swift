//
//  ShopClosedWaitingViewModel.swift
//  driverappios
//

import Foundation
import Observation
import PhotosUI
import SwiftUI
import UIKit

@Observable
@MainActor
final class ShopClosedWaitingViewModel {
    let orderId: String
    let driverId: String

    var reported = false
    var isReporting = false
    var reportError: String?
    var retailerResponse: String?
    var bypassToken: String?
    var bypassInput = ""
    var isSubmittingBypass = false
    var bypassError: String?
    var isEscalated = false
    var countdown: Int = 180
    var pickedPhoto: PhotosPickerItem?
    var photoURL = ""
    var photoLocalPath = ""
    var isUploadingPhoto = false

    private var timer: Timer?
    private let sendShopClosed: @Sendable (String, String, String) async throws -> Void
    private let sendBypass: @Sendable (String, String) async throws -> Void
    private let uploadPhotoJPEG: @Sendable (UIImage, String, String) async throws -> String

    init(
        orderId: String,
        driverId: String,
        sendShopClosed: @escaping @Sendable (String, String, String) async throws -> Void = { orderId, photoURL, ts in
            _ = try await APIClient.shared.reportShopClosed(
                orderId: orderId,
                photoURL: photoURL,
                clientTimestamp: ts
            )
        },
        sendBypass: @escaping @Sendable (String, String) async throws -> Void = { orderId, token in
            _ = try await APIClient.shared.bypassOffload(orderId: orderId, token: token)
        },
        uploadPhotoJPEG: @escaping @Sendable (UIImage, String, String) async throws -> String = { image, purpose, orderId in
            try await MediaUploadService.uploadJPEG(image: image, purpose: purpose, orderId: orderId)
        }
    ) {
        self.orderId = orderId
        self.driverId = driverId
        self.sendShopClosed = sendShopClosed
        self.sendBypass = sendBypass
        self.uploadPhotoJPEG = uploadPhotoJPEG
    }

    var statusIcon: String {
        if !reported { return "camera.fill" }
        if isReporting { return "clock.fill" }
        if retailerResponse != nil { return "bubble.left.fill" }
        if isEscalated { return "exclamationmark.triangle.fill" }
        if bypassToken != nil { return "key.fill" }
        return "door.left.hand.closed"
    }

    var statusColor: Color {
        if retailerResponse != nil { return LabTheme.success }
        if isEscalated { return LabTheme.warning }
        if countdown <= 30 { return LabTheme.destructive }
        return Color.orange
    }

    var statusTitle: String {
        if !reported { return "Shop Closed" }
        if isReporting { return "Reporting..." }
        if retailerResponse != nil { return "Retailer Responded" }
        if isEscalated { return "Escalated" }
        return "Shop Closed"
    }

    var countdownFormatted: String {
        let m = countdown / 60
        let s = countdown % 60
        return String(format: "%d:%02d", m, s)
    }

    var hasPhoto: Bool {
        !photoURL.isEmpty || !photoLocalPath.isEmpty
    }

    func uploadPickedPhoto(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        isUploadingPhoto = true
        reportError = nil
        defer { isUploadingPhoto = false }
        do {
            guard let data = try await item.loadTransferable(type: Data.self),
                  let image = UIImage(data: data),
                  let jpeg = image.jpegData(compressionQuality: 0.82) else {
                reportError = "Could not read photo."
                return
            }
            photoLocalPath = (try? MediaUploadService.savePodJPEG(jpeg, prefix: "shop-closed")) ?? ""
            do {
                photoURL = try await uploadPhotoJPEG(image, "credit_proof", orderId)
            } catch {
                photoURL = ""
                if photoLocalPath.isEmpty {
                    reportError = "Photo upload failed: \(error.localizedDescription)"
                }
            }
        } catch {
            reportError = "Photo upload failed: \(error.localizedDescription)"
        }
    }

    func reportShopClosed() async {
        guard hasPhoto else {
            Haptics.warning()
            reportError = "Photo required before reporting shop closed."
            return
        }
        isReporting = true
        reportError = nil
        let ts = DriverOfflineActionCatalog.nowIso()
        do {
            if photoURL.isEmpty {
                throw URLError(.notConnectedToInternet)
            }
            try await sendShopClosed(orderId, photoURL, ts)
            reported = true
            isReporting = false
            Haptics.success()
            if let notice = DriverSocketState.shared.outdatedNotice {
                reportError = notice.message
                return
            }
            startCountdown()
        } catch {
            if DriverOfflineActionCatalog.isNetworkEnqueueable(error) || photoURL.isEmpty {
                var body: [String: Any] = [
                    "order_id": orderId,
                    "reason": "CLOSED",
                    "client_timestamp": ts,
                ]
                if !photoURL.isEmpty {
                    body["photo_url"] = photoURL
                } else if !photoLocalPath.isEmpty {
                    body["photo_local_path"] = photoLocalPath
                }
                DriverOfflineQueue.shared.enqueueJSONObject(
                    endpoint: DriverOfflineActionCatalog.shopClosed,
                    body: body,
                    idempotencyKey: DriverIdempotency.reportShopClosed(orderId: orderId),
                    orderId: orderId,
                    clientTimestampIso: ts
                )
                reported = true
                isReporting = false
                reportError = "Offline — shop-closed queued for sync"
                startCountdown()
            } else {
                isReporting = false
                Haptics.error()
                reportError = "Failed to report: \(error.localizedDescription)"
            }
        }
    }

    func submitBypass() async -> Bool {
        isSubmittingBypass = true
        bypassError = nil
        do {
            try await sendBypass(orderId, bypassInput)
            isSubmittingBypass = false
            Haptics.success()
            return true
        } catch {
            isSubmittingBypass = false
            Haptics.error()
            bypassError = error.localizedDescription
            return false
        }
    }

    func startCountdown() {
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor [weak self] in
                guard let self else { return }
                if self.countdown > 0 {
                    self.countdown -= 1
                } else {
                    self.stopCountdown()
                }
            }
        }
    }

    func stopCountdown() {
        timer?.invalidate()
        timer = nil
    }

    func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        if let notice = DriverSocketState.shared.outdatedNotice {
            reportError = notice.message
            Haptics.warning()
            return
        }

        guard let event, event.orderId == orderId else { return }

        switch event.type {
        case "SHOP_CLOSED_RESPONSE":
            retailerResponse = event.response
            Haptics.medium()
        case "BYPASS_TOKEN_ISSUED":
            bypassToken = event.bypassToken
            Haptics.medium()
        case "SHOP_CLOSED_ESCALATED":
            isEscalated = true
            Haptics.warning()
        default:
            break
        }
    }

    func handleReconnect() async {
        let hadInFlight = isReporting || isSubmittingBypass
        await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
        if hadInFlight {
            isReporting = false
            isSubmittingBypass = false
            reportError = DriverReconnectRecovery.hint
        }
    }
}

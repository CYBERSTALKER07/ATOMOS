//
//  ShopClosedWaitingView.swift
//  driverappios
//
//  Shown when driver reports shop closed.
//  Requires a storefront photo first, then waits for retailer response or bypass token.
//

import PhotosUI
import SwiftUI
import UIKit

struct ShopClosedWaitingView: View {

    let orderId: String
    let driverId: String
    let onResolved: () -> Void
    let onCancel: () -> Void

    @State private var reported = false
    @State private var isReporting = false
    @State private var reportError: String?
    @State private var retailerResponse: String?
    @State private var bypassToken: String?
    @State private var bypassInput = ""
    @State private var isSubmittingBypass = false
    @State private var bypassError: String?
    @State private var isEscalated = false
    @State private var countdown: Int = 180
    @State private var timer: Timer?
    @State private var driverSocketState = DriverSocketState.shared
    @State private var pickedPhoto: PhotosPickerItem?
    @State private var photoURL = ""
    @State private var photoLocalPath = ""
    @State private var isUploadingPhoto = false

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            Image(systemName: statusIcon)
                .font(.system(size: 64))
                .foregroundStyle(statusColor)
                .padding(.bottom, LabTheme.s16)

            Text(statusTitle)
                .font(.system(size: 24, weight: .bold))
                .foregroundStyle(LabTheme.fg)
                .padding(.bottom, LabTheme.s8)

            Text(orderId)
                .font(.system(size: 15, weight: .semibold, design: .monospaced))
                .foregroundStyle(LabTheme.fgSecondary)
                .padding(.bottom, LabTheme.s16)

            if !reported {
                VStack(spacing: 12) {
                    Text("Photograph the closed storefront, then report.")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, LabTheme.s24)

                    PhotosPicker(selection: $pickedPhoto, matching: .images) {
                        Label(
                            photoURL.isEmpty && photoLocalPath.isEmpty
                                ? "Take / choose photo"
                                : "Photo ready — change",
                            systemImage: "camera.fill"
                        )
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .disabled(isUploadingPhoto || isReporting)
                    .onChange(of: pickedPhoto) { _, item in
                        Task { await uploadPickedPhoto(item) }
                    }

                    if isUploadingPhoto {
                        ProgressView("Uploading photo…")
                    }

                    Button {
                        Task { await reportShopClosed() }
                    } label: {
                        HStack {
                            if isReporting { ProgressView().tint(LabTheme.buttonFg) }
                            Text("Report shop closed")
                                .font(.system(size: 15, weight: .bold))
                        }
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .disabled(isReporting || isUploadingPhoto || (photoURL.isEmpty && photoLocalPath.isEmpty))
                    .padding(.horizontal, LabTheme.s24)
                }
            } else if isReporting {
                ProgressView()
                    .scaleEffect(1.2)
                    .padding(.bottom, LabTheme.s8)
                Text("mobile_driver.ui.reporting_shop_closed")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            } else if let response = retailerResponse {
                retailerResponseView(response)
            } else if isEscalated {
                Text("mobile_driver.ui.escalated_to_supplier_awaiting_resolution")
                    .font(.system(size: 14, weight: .medium))
                    .foregroundStyle(LabTheme.warning)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, LabTheme.s24)
            } else {
                Text(countdownFormatted)
                    .font(.system(size: 48, weight: .bold, design: .monospaced))
                    .foregroundStyle(countdown <= 30 ? LabTheme.destructive : LabTheme.fg)
                    .padding(.bottom, LabTheme.s8)

                Text("mobile_driver.ui.waiting_for_retailer_response")
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.fgTertiary)
            }

            if let token = bypassToken {
                VStack(spacing: 12) {
                    Divider().padding(.horizontal, LabTheme.s24)

                    Text("mobile_driver.ui.bypass_token_issued")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(LabTheme.success)

                    Text(token)
                        .font(.system(size: 32, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                        .tracking(8)

                    TextField("mobile_driver.ui.enter_token", text: $bypassInput)
                        .font(.system(size: 18, weight: .semibold, design: .monospaced))
                        .multilineTextAlignment(.center)
                        .keyboardType(.numberPad)
                        .textFieldStyle(.roundedBorder)
                        .frame(maxWidth: 200)

                    if let err = bypassError {
                        Text(err)
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(LabTheme.destructive)
                    }

                    Button {
                        submitBypass()
                    } label: {
                        HStack(spacing: 8) {
                            if isSubmittingBypass {
                                ProgressView().tint(LabTheme.buttonFg)
                            }
                            Text("mobile_driver.ui.confirm_bypass")
                                .font(.system(size: 15, weight: .bold))
                        }
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                    }
                    .disabled(bypassInput.count != 6 || isSubmittingBypass)
                    .padding(.horizontal, LabTheme.s24)
                }
                .padding(.top, LabTheme.s16)
            }

            Spacer()

            if let error = reportError {
                Text(error)
                    .font(.system(size: 13, weight: .medium))
                    .foregroundStyle(LabTheme.destructive)
                    .padding(.horizontal, LabTheme.s24)
                    .padding(.bottom, LabTheme.s8)
            }

            Button {
                onCancel()
            } label: {
                Text("common.action.back")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(LabTheme.fg.opacity(0.08), in: .rect(cornerRadius: LabTheme.buttonRadius))
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(LabTheme.bg)
        .task(id: driverSocketState.eventSequence) {
            handleDriverSocketEvent(driverSocketState.lastEvent)
        }
        .onDisappear {
            timer?.invalidate()
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task {
                let hadInFlight = isReporting || isSubmittingBypass
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: hadInFlight)
                if hadInFlight {
                    isReporting = false
                    isSubmittingBypass = false
                    reportError = DriverReconnectRecovery.hint
                }
            }
        }
    }

    private var statusIcon: String {
        if !reported { return "camera.fill" }
        if isReporting { return "clock.fill" }
        if retailerResponse != nil { return "bubble.left.fill" }
        if isEscalated { return "exclamationmark.triangle.fill" }
        if bypassToken != nil { return "key.fill" }
        return "door.left.hand.closed"
    }

    private var statusColor: Color {
        if retailerResponse != nil { return LabTheme.success }
        if isEscalated { return LabTheme.warning }
        if countdown <= 30 { return LabTheme.destructive }
        return Color.orange
    }

    private var statusTitle: String {
        if !reported { return "Shop Closed" }
        if isReporting { return "Reporting..." }
        if retailerResponse != nil { return "Retailer Responded" }
        if isEscalated { return "Escalated" }
        return "Shop Closed"
    }

    private var countdownFormatted: String {
        let m = countdown / 60
        let s = countdown % 60
        return String(format: "%d:%02d", m, s)
    }

    @ViewBuilder
    private func retailerResponseView(_ response: String) -> some View {
        let (label, icon): (String, String) = switch response {
        case "OPEN_NOW": ("Retailer says they are open now", "door.left.hand.open")
        case "5_MIN": ("Retailer will be ready in 5 minutes", "clock.badge.checkmark")
        case "CALL_ME": ("Retailer requests a phone call", "phone.fill")
        case "CLOSED_TODAY": ("Retailer confirmed closed today", "xmark.circle.fill")
        default: ("Response: \(response)", "questionmark.circle")
        }

        VStack(spacing: 8) {
            Image(systemName: icon)
                .font(.system(size: 28))
                .foregroundStyle(LabTheme.fg)
            Text(label)
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(LabTheme.fg)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
        .padding(.vertical, LabTheme.s16)
    }

    private func uploadPickedPhoto(_ item: PhotosPickerItem?) async {
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
                photoURL = try await MediaUploadService.uploadJPEG(
                    image: image,
                    purpose: "credit_proof",
                    orderId: orderId
                )
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

    private func reportShopClosed() async {
        guard !photoURL.isEmpty || !photoLocalPath.isEmpty else {
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
            _ = try await APIClient.shared.reportShopClosed(
                orderId: orderId,
                photoURL: photoURL,
                clientTimestamp: ts
            )
            reported = true
            isReporting = false
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
                reportError = "Failed to report: \(error.localizedDescription)"
            }
        }
    }

    private func submitBypass() {
        isSubmittingBypass = true
        bypassError = nil
        Task {
            do {
                _ = try await APIClient.shared.bypassOffload(orderId: orderId, token: bypassInput)
                Haptics.success()
                onResolved()
            } catch {
                isSubmittingBypass = false
                bypassError = error.localizedDescription
            }
        }
    }

    private func startCountdown() {
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { _ in
            if countdown > 0 {
                countdown -= 1
            } else {
                timer?.invalidate()
            }
        }
    }

    private func handleDriverSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        if let notice = DriverSocketState.shared.outdatedNotice {
            reportError = notice.message
            Haptics.warning()
            return
        }

        guard let event else { return }
        guard event.orderId == orderId else { return }

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
}

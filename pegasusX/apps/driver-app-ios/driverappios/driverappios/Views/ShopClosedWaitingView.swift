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

    @State private var vm: ShopClosedWaitingViewModel
    @State private var driverSocketState = DriverSocketState.shared

    init(
        orderId: String,
        driverId: String,
        onResolved: @escaping () -> Void,
        onCancel: @escaping () -> Void
    ) {
        self.orderId = orderId
        self.driverId = driverId
        self.onResolved = onResolved
        self.onCancel = onCancel
        self._vm = State(initialValue: ShopClosedWaitingViewModel(orderId: orderId, driverId: driverId))
    }

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(spacing: 0) {
                Spacer(minLength: 48)

                Image(systemName: vm.statusIcon)
                    .font(.system(.largeTitle, weight: .bold))
                    .foregroundStyle(vm.statusColor)
                    .padding(.bottom, LabTheme.s16)
                    .accessibilityHidden(true)

                Text(vm.statusTitle)
                    .font(.system(.title2, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.bottom, LabTheme.s8)

                Text(orderId)
                    .font(.system(.subheadline, design: .monospaced, weight: .semibold))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .padding(.bottom, LabTheme.s16)

                if !vm.reported {
                    VStack(spacing: 12) {
                        Text("Photograph the closed storefront, then report.")
                            .font(.system(.subheadline, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                            .multilineTextAlignment(.center)
                            .padding(.horizontal, LabTheme.s24)

                        PhotosPicker(selection: $vm.pickedPhoto, matching: .images) {
                            Label(
                                !vm.hasPhoto ? "Take / choose photo" : "Photo ready — change",
                                systemImage: "camera.fill"
                            )
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(LabTheme.fg)
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .padding(.vertical, 6)
                            .background(LabTheme.fg.opacity(0.06), in: .rect(cornerRadius: LabTheme.buttonRadius))
                        }
                        .disabled(vm.isUploadingPhoto || vm.isReporting)
                        .onChange(of: vm.pickedPhoto) { _, item in
                            Task { await vm.uploadPickedPhoto(item) }
                        }

                        if vm.isUploadingPhoto {
                            ProgressView("Uploading photo…")
                        }

                        Button {
                            Task { await vm.reportShopClosed() }
                        } label: {
                            HStack {
                                if vm.isReporting { ProgressView().tint(LabTheme.buttonFg) }
                                Text("Report shop closed")
                                    .font(.system(.subheadline, weight: .bold))
                            }
                            .foregroundStyle(LabTheme.buttonFg)
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .padding(.vertical, 8)
                            .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                        }
                        .disabled(vm.isReporting || vm.isUploadingPhoto || !vm.hasPhoto)
                        .padding(.horizontal, LabTheme.s24)
                    }
                } else if vm.isReporting {
                    ProgressView()
                        .scaleEffect(1.2)
                        .padding(.bottom, LabTheme.s8)
                    Text("mobile_driver.ui.reporting_shop_closed")
                        .font(.system(.footnote, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                } else if let response = vm.retailerResponse {
                    retailerResponseView(response)
                } else if vm.isEscalated {
                    Text("mobile_driver.ui.escalated_to_supplier_awaiting_resolution")
                        .font(.system(.subheadline, weight: .medium))
                        .foregroundStyle(LabTheme.warning)
                        .multilineTextAlignment(.center)
                        .padding(.horizontal, LabTheme.s24)
                } else {
                    Text(vm.countdownFormatted)
                        .font(.system(.largeTitle, design: .monospaced, weight: .bold))
                        .foregroundStyle(vm.countdown <= 30 ? LabTheme.destructive : LabTheme.fg)
                        .padding(.bottom, LabTheme.s8)

                    Text("mobile_driver.ui.waiting_for_retailer_response")
                        .font(.system(.footnote, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                if let token = vm.bypassToken {
                    VStack(spacing: 12) {
                        Divider().padding(.horizontal, LabTheme.s24)

                        Text("mobile_driver.ui.bypass_token_issued")
                            .font(.system(.footnote, weight: .bold))
                            .foregroundStyle(LabTheme.success)

                        Text(token)
                            .font(.system(.title, design: .monospaced, weight: .bold))
                            .foregroundStyle(LabTheme.fg)
                            .tracking(8)

                        TextField("mobile_driver.ui.enter_token", text: $vm.bypassInput)
                            .font(.system(.headline, design: .monospaced, weight: .semibold))
                            .multilineTextAlignment(.center)
                            .keyboardType(.numberPad)
                            .textFieldStyle(.roundedBorder)
                            .frame(maxWidth: 200)

                        if let err = vm.bypassError {
                            Text(err)
                                .font(.system(.caption, weight: .medium))
                                .foregroundStyle(LabTheme.destructive)
                        }

                        Button {
                            Task {
                                if await vm.submitBypass() {
                                    onResolved()
                                }
                            }
                        } label: {
                            HStack(spacing: 8) {
                                if vm.isSubmittingBypass {
                                    ProgressView().tint(LabTheme.buttonFg)
                                }
                                Text("mobile_driver.ui.confirm_bypass")
                                    .font(.system(.subheadline, weight: .bold))
                            }
                            .foregroundStyle(LabTheme.buttonFg)
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .padding(.vertical, 8)
                            .background(LabTheme.fg, in: .rect(cornerRadius: LabTheme.buttonRadius))
                        }
                        .disabled(vm.bypassInput.count != 6 || vm.isSubmittingBypass)
                        .padding(.horizontal, LabTheme.s24)
                    }
                    .padding(.top, LabTheme.s16)
                }

                Spacer(minLength: 48)

                if let error = vm.reportError {
                    Text(error)
                        .font(.system(.footnote, weight: .medium))
                        .foregroundStyle(LabTheme.destructive)
                        .padding(.horizontal, LabTheme.s24)
                        .padding(.bottom, LabTheme.s8)
                }

                Button {
                    Haptics.light()
                    onCancel()
                } label: {
                    Text("common.action.back")
                        .font(.system(.subheadline, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .padding(.vertical, 8)
                        .background(LabTheme.fg.opacity(0.08), in: .rect(cornerRadius: LabTheme.buttonRadius))
                }
                .padding(.horizontal, LabTheme.s24)
                .padding(.bottom, LabTheme.s24)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.bg)
        .task(id: driverSocketState.eventSequence) {
            vm.handleDriverSocketEvent(driverSocketState.lastEvent)
        }
        .onDisappear {
            vm.stopCountdown()
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task { await vm.handleReconnect() }
        }
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
                .font(.system(.title))
                .foregroundStyle(LabTheme.fg)
            Text(label)
                .font(.system(.subheadline, weight: .medium))
                .foregroundStyle(LabTheme.fg)
                .multilineTextAlignment(.center)
                .padding(.horizontal, LabTheme.s24)
        }
        .padding(.vertical, LabTheme.s16)
        .accessibilityElement(children: .combine)
    }
}

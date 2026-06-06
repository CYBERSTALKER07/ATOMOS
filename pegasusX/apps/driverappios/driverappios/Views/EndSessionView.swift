//
//  EndSessionView.swift
//  driverappios
//

import SwiftUI

/// Offline reason codes matching the backend validation in HandleDriverAvailability.
enum OfflineReason: String, CaseIterable {
    case SHIFT_COMPLETE
    case TRUCK_DAMAGED
    case PERSONAL
    case OTHER

    var label: String {
        switch self {
        case .SHIFT_COMPLETE: "Shift Complete"
        case .TRUCK_DAMAGED: "Truck Damaged"
        case .PERSONAL: "Personal"
        case .OTHER: "Other"
        }
    }

    var icon: String {
        switch self {
        case .SHIFT_COMPLETE: "moon.fill"
        case .TRUCK_DAMAGED: "wrench.fill"
        case .PERSONAL: "person.fill"
        case .OTHER: "questionmark.circle.fill"
        }
    }
}

/// Bottom sheet for ending the driver's session with a reason code.
struct EndSessionView: View {
    @Bindable var vm: FleetViewModel
    @Environment(\.dismiss) private var dismiss

    @State private var selectedReason: OfflineReason?
    @State private var note = ""

    private var canConfirm: Bool {
        guard let reason = selectedReason else { return false }
        if vm.hasActiveOrders { return false }
        if vm.isEndingSession { return false }
        if reason == .OTHER && note.trimmingCharacters(in: .whitespaces).isEmpty { return false }
        return true
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Drag handle
            HStack {
                Spacer()
                RoundedRectangle(cornerRadius: 2)
                    .fill(LabTheme.fgTertiary)
                    .frame(width: 32, height: 4)
                Spacer()
            }
            .padding(.top, 12)
            .padding(.bottom, 16)

            VStack(alignment: .leading, spacing: LabTheme.s20) {
                // Title
                VStack(alignment: .leading, spacing: 4) {
                    Text("End Session")
                        .font(.system(size: 24, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                    Text("Go offline and end your driving session")
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                // Active orders warning
                if vm.hasActiveOrders {
                    HStack(spacing: 10) {
                        Image(systemName: "exclamationmark.triangle.fill")
                            .font(.system(size: 14))
                            .foregroundStyle(LabTheme.destructive)
                        Text("Complete or return active orders before ending your session.")
                            .font(.system(size: 13, weight: .medium))
                            .foregroundStyle(LabTheme.destructive)
                    }
                    .padding(12)
                    .background(LabTheme.destructive.opacity(0.08), in: .rect(cornerRadius: 12))
                }

                // Reason label
                Text("REASON")
                    .font(.system(size: 10, weight: .heavy, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .tracking(1)

                // Reason options
                VStack(spacing: 8) {
                    ForEach(OfflineReason.allCases, id: \.self) { reason in
                        reasonOption(reason)
                    }
                }

                // Note field
                if selectedReason == .OTHER || selectedReason == .TRUCK_DAMAGED {
                    TextField(
                        selectedReason == .OTHER ? "Describe reason (required)" : "Describe damage (optional)",
                        text: $note,
                        axis: .vertical
                    )
                    .lineLimit(2...4)
                    .textFieldStyle(.roundedBorder)
                    .transition(.move(edge: .top).combined(with: .opacity))
                }

                // Error
                if let error = vm.endSessionError {
                    Text(error)
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(LabTheme.destructive)
                }

                // Confirm button
                Button {
                    guard let reason = selectedReason else { return }
                    Haptics.heavy()
                    Task {
                        await vm.endSession(reason: reason.rawValue, note: note.isEmpty ? nil : note)
                    }
                } label: {
                    HStack {
                        Spacer()
                        if vm.isEndingSession {
                            ProgressView()
                                .tint(LabTheme.buttonFg)
                        } else {
                            Text("End Session")
                                .font(.system(size: 15, weight: .bold))
                                .foregroundStyle(canConfirm ? LabTheme.buttonFg : LabTheme.fgTertiary)
                        }
                        Spacer()
                    }
                    .padding(.vertical, 14)
                    .background(
                        canConfirm ? LabTheme.destructive : LabTheme.fgTertiary.opacity(0.3),
                        in: .rect(cornerRadius: 16)
                    )
                }
                .disabled(!canConfirm)
            }
            .padding(.horizontal, LabTheme.s20)
            .padding(.bottom, 32)
        }
        .background(LabTheme.bg)
        .animation(.smooth(duration: 0.25), value: selectedReason)
    }

    private func reasonOption(_ reason: OfflineReason) -> some View {
        Button {
            selectedReason = reason
        } label: {
            HStack(spacing: 14) {
                Image(systemName: reason.icon)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(selectedReason == reason ? LabTheme.fg : LabTheme.fgSecondary)
                    .frame(width: 20)

                Text(reason.label)
                    .font(.system(size: 16, weight: .semibold))
                    .foregroundStyle(selectedReason == reason ? LabTheme.fg : LabTheme.fgSecondary)

                Spacer()
            }
            .padding(LabTheme.s16)
            .background(
                selectedReason == reason ? LabTheme.fg.opacity(0.04) : .clear,
                in: .rect(cornerRadius: 12)
            )
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(
                        selectedReason == reason ? LabTheme.fg : LabTheme.fgTertiary.opacity(0.2),
                        lineWidth: selectedReason == reason ? 1.5 : 1
                    )
            )
        }
        .buttonStyle(.pressable)
    }
}

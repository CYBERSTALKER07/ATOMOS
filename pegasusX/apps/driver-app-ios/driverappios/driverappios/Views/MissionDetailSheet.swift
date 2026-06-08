//
//  MissionDetailSheet.swift
//  driverappios
//

import CoreLocation
import SwiftUI

struct MissionDetailSheet: View {
    @Environment(\.dismiss) private var dismiss

    let mission: Mission
    let driverLocation: CLLocationCoordinate2D?
    let isInRange: Bool
    let distance: Double?
    let onScan: () -> Void
    let onCorrection: () -> Void

    @State private var showOutOfRangeAlert = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            // MARK: - Header
            ZStack(alignment: .topTrailing) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("ORD-\(mission.order_id.suffix(6).uppercased())") // Increased tactical length
                        .font(.system(size: 26, weight: .black, design: .monospaced)) // Black weight
                        .foregroundStyle(LabTheme.fg)
                        .tracking(1.2)

                    HStack(spacing: 6) {
                        Text(mission.gateway.uppercased())
                            .font(.system(size: 10, weight: .black, design: .monospaced)) // Tactical badge
                            .padding(.horizontal, 10)
                            .padding(.vertical, 4)
                            .background(LabTheme.fg.opacity(0.1), in: Capsule())
                        
                        Text(mission.amount.formattedAmount)
                            .font(.system(size: 14, weight: .bold, design: .monospaced)) // Bold mono
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                Button { dismiss() } label: {
                    Image(systemName: "xmark")
                        .font(.system(size: 11, weight: .bold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(width: 28, height: 28)
                        .background(LabTheme.fg.opacity(0.06), in: Circle())
                }
                .accessibilityLabel("Close")
            }
            .padding(.horizontal, LabTheme.s24)
            .padding(.top, LabTheme.s24)
            .padding(.bottom, LabTheme.s16)

            // MARK: - Delivery Endpoint Card
            VStack(alignment: .leading, spacing: 10) {
                Text("TACTICAL ENDPOINT") // Strategic relabel
                    .font(.system(size: 10, weight: .black, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .tracking(1.4)

                Text(String(format: "%.5f, %.5f", mission.target_lat, mission.target_lng)) // High precision
                    .font(.system(size: 16, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)

                HStack(spacing: 8) {
                    Circle()
                        .fill(isInRange ? LabTheme.success : LabTheme.warning)
                        .frame(width: 8, height: 8)

                    if let dist = distance {
                        Text(formattedDistance(dist))
                            .font(.system(size: 13, weight: .black, design: .monospaced)) // Black mono
                    }

                    Text(isInRange ? "GEOFENCE_CLEARED" : "PROXIMITY_FAULT") // Tactical caps
                        .font(.system(size: 12, weight: .black, design: .monospaced))
                        .foregroundStyle(isInRange ? LabTheme.success : LabTheme.warning)
                }
            }
            .padding(LabTheme.s20) // Spacing token
            .frame(maxWidth: .infinity, alignment: .leading)
            .background {
                RoundedRectangle(cornerRadius: LabTheme.cardRadius - 4, style: .continuous)
                    .fill(LabTheme.fg.opacity(0.04))
                    .overlay {
                        RoundedRectangle(cornerRadius: LabTheme.cardRadius - 4, style: .continuous)
                            .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
            .padding(.horizontal, LabTheme.s24)

            Spacer()

            // MARK: - Correction Link
            Button {
                Haptics.light()
                onCorrection()
            } label: {
                HStack(spacing: 8) {
                    Image(systemName: "pencil.and.list.clipboard")
                        .font(.system(size: 14, weight: .bold))
                    Text("PROCEDURAL_CORRECTION")
                        .font(.system(size: 11, weight: .black, design: .monospaced))
                        .tracking(1.2)
                }
                .foregroundStyle(LabTheme.fgSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 14)
                .background {
                    RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                        .fill(LabTheme.fg.opacity(0.06))
                        .overlay {
                            RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                                .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1)
                        }
                }
            }
            .buttonStyle(.pressable)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, 12)

            // MARK: - Action Button
            Button {
                if isInRange {
                    Haptics.medium()
                    onScan()
                } else {
                    Haptics.error()
                    showOutOfRangeAlert = true
                }
            } label: {
                Text(isInRange ? "INITIATE_POD" : "AWAITING_PROXIMITY")
                    .font(.system(size: 14, weight: .black, design: .monospaced))
                    .tracking(1.2)
                    .foregroundStyle(isInRange ? LabTheme.buttonFg : LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 18)
                    .background {
                        RoundedRectangle(cornerRadius: LabTheme.buttonRadius, style: .continuous)
                            .fill(isInRange ? LabTheme.fg : LabTheme.fg.opacity(0.08))
                    }
            }
            .buttonStyle(.pressable)
            .disabled(!isInRange)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24 + 10) // Extra padding for home bar
        }
        .background {
            LabTheme.bg
                .ignoresSafeArea()
                .overlay(.ultraThinMaterial)
        }
        .alert("Out of Range", isPresented: $showOutOfRangeAlert) {
            Button("OK", role: .cancel) { }
        } message: {
            Text("You must be within the geofence boundary to initiate proof of delivery.")
        }
    }
}

#Preview {
    MissionDetailSheet(
        mission: Mission.mockMissions[0],
        driverLocation: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
        isInRange: true,
        distance: 245,
        onScan: {},
        onCorrection: {}
    )
}

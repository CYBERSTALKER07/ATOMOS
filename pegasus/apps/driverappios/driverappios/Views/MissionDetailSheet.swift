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
                    Text(mission.order_id)
                        .font(.system(size: 24, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)

                    HStack(spacing: 6) {
                        Text(mission.gateway)
                            .font(.system(size: 12, weight: .semibold))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 3)
                            .background(LabTheme.fg.opacity(0.08), in: Capsule())
                        Text(mission.amount.formattedAmount)
                            .font(.system(size: 13, weight: .medium))
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
                Text("DELIVERY ENDPOINT")
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)

                Text(String(format: "%.4f, %.4f", mission.target_lat, mission.target_lng))
                    .font(.system(size: 16, weight: .semibold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)

                HStack(spacing: 8) {
                    Circle()
                        .fill(isInRange ? LabTheme.success : LabTheme.warning)
                        .frame(width: 8, height: 8)

                    if let dist = distance {
                        Text(formattedDistance(dist))
                            .font(.system(size: 13, weight: .bold, design: .monospaced))
                    }

                    Text(isInRange ? "Geofence cleared" : "Proximity check fault")
                        .font(.system(size: 13, weight: .medium))
                        .foregroundStyle(isInRange ? LabTheme.success : LabTheme.warning)
                }
            }
            .padding(LabTheme.s16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: LabTheme.cardRadius - 4))
            .overlay {
                RoundedRectangle(cornerRadius: LabTheme.cardRadius - 4)
                    .stroke(LabTheme.separator, lineWidth: 0.5)
            }
            .padding(.horizontal, LabTheme.s24)

            Spacer()

            // MARK: - Correction Link
            Button {
                Haptics.light()
                onCorrection()
            } label: {
                HStack(spacing: 6) {
                    Image(systemName: "pencil.and.list.clipboard")
                        .font(.system(size: 12))
                    Text("Delivery Correction")
                        .font(.system(size: 13, weight: .semibold))
                }
                .foregroundStyle(LabTheme.fgSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 12)
                .background(LabTheme.fg.opacity(0.04), in: .rect(cornerRadius: LabTheme.buttonRadius))
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
                Text(isInRange ? "Initiate Proof of Delivery" : "Approach Target for Scan")
                    .font(.system(size: 15, weight: .bold))
                    .foregroundStyle(isInRange ? LabTheme.buttonFg : LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(
                        isInRange ? LabTheme.fg : LabTheme.fg.opacity(0.06),
                        in: .rect(cornerRadius: LabTheme.buttonRadius)
                    )
            }
            .buttonStyle(.pressable)
            .disabled(!isInRange)
            .padding(.horizontal, LabTheme.s24)
            .padding(.bottom, LabTheme.s24)
        }
        .background(.ultraThinMaterial)
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

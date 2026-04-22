//
//  MissionListCard.swift
//  driverappios
//

import CoreLocation
import SwiftUI

struct MissionListCard: View {

    let missions: [Mission]
    let isLoading: Bool
    let driverLocation: CLLocationCoordinate2D?
    let onSelect: (Mission) -> Void
    let onRefresh: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            if isLoading {
                HStack(spacing: 10) {
                    ProgressView()
                        .tint(LabTheme.fg)
                    Text("Syncing fleet...")
                        .font(.subheadline)
                        .foregroundStyle(LabTheme.fgSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 32)
            } else if missions.isEmpty {
                VStack(spacing: 14) {
                    Image(systemName: "shippingbox")
                        .font(.system(size: 28))
                        .foregroundStyle(LabTheme.fgTertiary)

                    Text("No active missions")
                        .font(.subheadline.weight(.medium))
                        .foregroundStyle(LabTheme.fgSecondary)

                    Button {
                        Haptics.light()
                        onRefresh()
                    } label: {
                        Text("Refresh")
                            .font(.subheadline.weight(.bold))
                            .foregroundStyle(LabTheme.fg)
                            .padding(.horizontal, 20)
                            .padding(.vertical, 10)
                            .background(LabTheme.fg.opacity(0.08), in: Capsule())
                    }
                    .buttonStyle(.pressable)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 32)
            } else {
                LazyVStack(spacing: 12) {
                    ForEach(Array(missions.enumerated()), id: \.element.id) { index, mission in
                        missionRow(mission, index: index)
                    }
                }
                .padding(.horizontal, LabTheme.s16)
                .padding(.vertical, LabTheme.s12)
            }
        }
    }

    // MARK: - Mission Row

    private func missionRow(_ mission: Mission, index: Int) -> some View {
        Button {
            onSelect(mission)
        } label: {
            HStack(spacing: 14) {
                // Icon
                ZStack {
                    RoundedRectangle(cornerRadius: 12)
                        .fill(LabTheme.fg.opacity(0.06))
                        .frame(width: 44, height: 44)

                    Image(systemName: "shippingbox.fill")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                }

                VStack(alignment: .leading, spacing: 4) {
                    Text(mission.order_id)
                        .font(.system(size: 14, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)

                    HStack(spacing: 4) {
                        Text(mission.gateway)
                        Text("·")
                        Text(mission.amount.formattedAmount)
                    }
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)

                    distanceLabel(for: mission)
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .labCard()
        }
        .buttonStyle(.pressable)
        .staggeredAppear(index: index)
    }

    // MARK: - Distance Label

    @ViewBuilder
    private func distanceLabel(for mission: Mission) -> some View {
        if let loc = driverLocation {
            let target = CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
            let dist = haversineDistance(from: loc, to: target)
            let inRange = dist <= 100

            HStack(spacing: 4) {
                Circle()
                    .fill(inRange ? LabTheme.success : LabTheme.fgTertiary)
                    .frame(width: 5, height: 5)

                Text(formattedDistance(dist))
                    .font(.system(size: 11, weight: .semibold, design: .monospaced))
                    .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgTertiary)

                if inRange {
                    Text("In Range")
                        .font(.system(size: 11, weight: .medium))
                        .foregroundStyle(LabTheme.success)
                }
            }
        }
    }
}

#Preview {
    ScrollView {
        MissionListCard(
            missions: Mission.mockMissions,
            isLoading: false,
            driverLocation: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
            onSelect: { _ in },
            onRefresh: {}
        )
    }
    .background(LabTheme.bg)
}

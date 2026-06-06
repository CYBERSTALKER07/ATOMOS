//
//  ActiveRideBar.swift
//  driverappios
//

import SwiftUI
import CoreLocation

/// Spotify-style floating "Now Active" bar shown above the tab bar
/// when a route is active and the user is NOT on the map tab.
struct ActiveRideBar: View {
    @Environment(\.colorScheme) private var cs
    let mission: Mission
    let driverLocation: CLLocationCoordinate2D?
    let onTap: () -> Void

    @State private var appeared = false
    @State private var pulse = false

    private var distance: Double? {
        guard let loc = driverLocation else { return nil }
        return haversineDistance(
            from: loc,
            to: CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
        )
    }

    var body: some View {
        Button(action: {
            Haptics.medium()
            onTap()
        }) {
            HStack(spacing: 12) {
                // Pulsing indicator
                ZStack {
                    Circle()
                        .fill(LabTheme.live)
                        .frame(width: 8, height: 8)

                    Circle()
                        .fill(LabTheme.live.opacity(0.3))
                        .frame(width: 8, height: 8)
                        .scaleEffect(pulse ? 2.5 : 1)
                        .opacity(pulse ? 0 : 0.6)
                }
                .frame(width: 20, height: 20)

                // Order info
                VStack(alignment: .leading, spacing: 2) {
                    Text(mission.order_id)
                        .font(.system(size: 13, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)

                    HStack(spacing: 4) {
                        Text(mission.gateway)
                            .font(.system(size: 11, weight: .medium))
                        if let dist = distance {
                            Text("·")
                            Text(formattedDistance(dist))
                                .font(.system(size: 11, weight: .semibold, design: .monospaced))
                        }
                    }
                    .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                // Chevron
                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(
                ZStack {
                    RoundedRectangle(cornerRadius: 22, style: .continuous)
                        .fill(.ultraThinMaterial)
                    RoundedRectangle(cornerRadius: 22, style: .continuous)
                        .stroke(LabTheme.separator, lineWidth: 0.5)
                }
                .shadow(color: .black.opacity(cs == .dark ? 0.45 : 0.08), radius: 20, y: 8)
            )
        }
        .buttonStyle(.pressable)
        .padding(.horizontal, LabTheme.s16)
        .offset(y: appeared ? 0 : 60)
        .opacity(appeared ? 1 : 0)
        .onAppear {
            withAnimation(Anim.bouncy) { appeared = true }
            withAnimation(Anim.breathe) { pulse = true }
        }
    }
}

#Preview {
    ZStack {
        LabTheme.bg.ignoresSafeArea()
        VStack {
            Spacer()
            ActiveRideBar(
                mission: Mission.mockMissions[0],
                driverLocation: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
                onTap: {}
            )
            .padding(.bottom, 80)
        }
    }
}

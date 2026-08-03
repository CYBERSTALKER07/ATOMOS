import SwiftUI
import MapKit

struct ActiveDeliverySheet: View {
    let mission: Mission
    var vm: FleetViewModel
    var bottomInset: Double
    @Binding var navPath: NavigationPath
    @Binding var showRescueSheet: Bool
    
    var body: some View {
        let dist = vm.distanceToMission(mission)
        let inRange = vm.isInRange(mission)
        let order = vm.orders.first { $0.id == mission.order_id }

        return VStack(alignment: .leading, spacing: 12) {
            SheetHandle()

            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 6) {
                        Circle().fill(LabTheme.live).frame(width: 7, height: 7)
                        Text("ACTIVE")
                            .font(.system(size: 9, weight: .heavy, design: .monospaced))
                            .foregroundStyle(LabTheme.live)
                    }
                    Text(mission.order_id)
                        .font(.system(size: 17, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                }
                Spacer()
                Text(mission.gateway)
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.horizontal, 10).padding(.vertical, 5)
                    .background(.ultraThinMaterial, in: Capsule())
                    .overlay(Capsule().stroke(LabTheme.separator, lineWidth: 0.5))
            }
            .padding(.horizontal, LabTheme.s20)

            Rectangle().fill(LabTheme.separator).frame(height: 0.5)
                .padding(.horizontal, LabTheme.s20)

            HStack {
                HStack(spacing: 4) {
                    Image(systemName: "banknote").font(.system(size: 10))
                    Text(mission.amount.formattedAmount).font(.system(size: 12, weight: .semibold))
                }
                .foregroundStyle(LabTheme.fgSecondary)
                Spacer()
                if let d = dist {
                    HStack(spacing: 4) {
                        Circle().fill(inRange ? LabTheme.success : LabTheme.warning).frame(width: 5, height: 5)
                        Text(formattedDistance(d))
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgSecondary)
                    }
                }
            }
            .padding(.horizontal, LabTheme.s20)

            // ETA Row
            if let order = order, let eta = order.estimatedArrivalAt {
                HStack(spacing: 5) {
                    Image(systemName: "clock.fill")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Text("ETA \(formatETATime(eta))")
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                    if let dur = order.etaDurationSec {
                        Text("· \(formatDuration(dur))")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    if let distM = order.etaDistanceM {
                        Text("· \(formatETADistance(distM))")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                }
                .padding(.horizontal, LabTheme.s20)
            }

            VStack(spacing: 8) {
                // Navigate button
                Button {
                    Haptics.light()
                    openDestinationInMaps(lat: mission.target_lat, lng: mission.target_lng, name: mission.order_id)
                } label: {
                    HStack(spacing: 7) {
                        Image(systemName: "arrow.triangle.turn.up.right.diamond.fill")
                            .font(.system(size: 13, weight: .semibold))
                        Text("Navigate in Maps")
                            .font(.system(size: 14, weight: .bold))
                    }
                    .foregroundStyle(LabTheme.buttonFg)
                    .frame(maxWidth: .infinity).padding(.vertical, 14)
                    .background(LabTheme.fg, in: .rect(cornerRadius: 14))
                }
                .buttonStyle(.pressable)

                Button {
                    Haptics.medium()
                    if inRange { navPath.append("scanner") }
                    else { Haptics.warning() }
                } label: {
                    HStack(spacing: 7) {
                        Image(systemName: inRange ? "qrcode.viewfinder" : "location.north.fill")
                            .font(.system(size: 13, weight: .semibold))
                        Text(inRange ? "Scan Proof of Delivery" : "Approach Target")
                            .font(.system(size: 14, weight: .bold))
                    }
                    .foregroundStyle(inRange ? LabTheme.fg : LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity).padding(.vertical, 14)
                    .background(
                        inRange ? LabTheme.fg.opacity(0.08) : LabTheme.fg.opacity(0.04),
                        in: .rect(cornerRadius: 14)
                    )
                }
                .buttonStyle(.pressable)

                HStack(spacing: 8) {
                    Button {
                        Haptics.light()
                        navPath.append("correction")
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "pencil.and.list.clipboard").font(.system(size: 11))
                            Text("Delivery Correction").font(.system(size: 12, weight: .semibold))
                        }
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(maxWidth: .infinity).padding(.vertical, 10)
                        .background(LabTheme.fg.opacity(0.04), in: .rect(cornerRadius: 12))
                    }
                    .buttonStyle(.pressable)
                    
                    Button {
                        Haptics.light()
                        showRescueSheet = true
                    } label: {
                        HStack(spacing: 4) {
                            Image(systemName: "exclamationmark.triangle.fill").font(.system(size: 11))
                            Text("Rescue").font(.system(size: 12, weight: .semibold))
                        }
                        .foregroundStyle(LabTheme.warning)
                        .frame(maxWidth: .infinity).padding(.vertical, 10)
                        .background(LabTheme.fg.opacity(0.04), in: .rect(cornerRadius: 12))
                    }
                    .buttonStyle(.pressable)
                }
            }
            .padding(.horizontal, LabTheme.s20)
        }
        .padding(.bottom, bottomInset + LabTheme.s4)
        .background(GlassSheet())
    }
    
    // ETA Helpers
    private func formatETATime(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: iso) ?? ISO8601DateFormatter().date(from: iso) {
            let tf = DateFormatter()
            tf.dateFormat = "HH:mm"
            return tf.string(from: date)
        }
        return iso.suffix(8).prefix(5).description
    }

    private func formatDuration(_ totalSec: Int) -> String {
        let m = totalSec / 60
        if m < 60 { return "\(m)m" }
        let h = m / 60
        let rem = m % 60
        return rem > 0 ? "\(h)h \(rem)m" : "\(h)h"
    }

    private func formatETADistance(_ meters: Int) -> String {
        if meters < 1000 { return "\(meters)m" }
        let km = Double(meters) / 1000.0
        return String(format: "%.1f km", km)
    }

    // Apple Maps Navigation
    private func openDestinationInMaps(lat: Double, lng: Double, name: String) {
        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lng)
        let placemark = MKPlacemark(coordinate: coord)
        let mapItem = MKMapItem(placemark: placemark)
        mapItem.name = name
        mapItem.openInMaps(launchOptions: [
            MKLaunchOptionsDirectionsModeKey: MKLaunchOptionsDirectionsModeDriving
        ])
    }
}

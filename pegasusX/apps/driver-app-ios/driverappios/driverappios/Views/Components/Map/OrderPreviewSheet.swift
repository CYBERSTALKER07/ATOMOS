import SwiftUI
import MapKit

struct OrderPreviewSheet: View {
    let mission: Mission
    var vm: FleetViewModel
    var bottomInset: Double
    @Binding var phase: MapPhase
    @Binding var selectedMission: Mission?
    @Binding var isCameraLocked: Bool
    @Binding var userPannedAt: Date?
    var startTelemetry: () -> Void
    
    var body: some View {
        let dist = vm.distanceToMission(mission)
        let inRange = vm.isInRange(mission)
        let order = vm.orders.first { $0.id == mission.order_id }

        return VStack(alignment: .leading, spacing: 16) {
            SheetHandle()

            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(L10n.format("mobile_driver.ui.ord_uppercased", "\(mission.order_id.suffix(6).uppercased())"))
                        .font(.system(size: 20, weight: .black, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                        .tracking(1.2)
                    HStack(spacing: 6) {
                        Text(mission.gateway.uppercased())
                            .font(.system(size: 9, weight: .black, design: .monospaced))
                            .padding(.horizontal, 10).padding(.vertical, 4)
                            .background(LabTheme.fg.opacity(0.1), in: Capsule())
                        Text(mission.amount.formattedAmount)
                            .font(.system(size: 13, weight: .bold, design: .monospaced))
                            .foregroundStyle(LabTheme.fgSecondary)
                        if let feeLabel = order?.deliveryFeeLabel {
                            Text(feeLabel)
                                .font(.system(size: 9, weight: .bold, design: .monospaced))
                                .foregroundStyle(LabTheme.warning)
                        }
                    }
                }
                Spacer()
                StatusPill(label: mission.state.uppercased(), color: LabTheme.fg)
            }
            .padding(.horizontal, LabTheme.s20)

            VStack(alignment: .leading, spacing: 7) {
                Text("STRATEGIC_ENDPOINT")
                    .font(.system(size: 9, weight: .black, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .tracking(1.4)
                Text(String(format: "%.5f, %.5f", mission.target_lat, mission.target_lng))
                    .font(.system(size: 14, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)
                if let d = dist {
                    HStack(spacing: 5) {
                        Circle().fill(inRange ? LabTheme.success : LabTheme.warning).frame(width: 6, height: 6)
                        Text(formattedDistance(d))
                            .font(.system(size: 12, weight: .black, design: .monospaced))
                            .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgSecondary)
                        Text(inRange ? "GEOFENCE_ACTIVE" : "APPROACHING_TARGET")
                            .font(.system(size: 10, weight: .black, design: .monospaced))
                            .foregroundStyle(inRange ? LabTheme.success : LabTheme.warning)
                    }
                }

                // ETA Row
                if let order = order, let eta = order.estimatedArrivalAt {
                    HStack(spacing: 5) {
                        Image(systemName: "clock.fill")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(LabTheme.fgTertiary)
                        Text(L10n.format("mobile_driver.ui.eta_formatetatime", "\(formatETATime(eta))"))
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .foregroundStyle(LabTheme.fg)
                        if let dur = order.etaDurationSec {
                            Text(L10n.format("mobile_driver.ui.formatduration", "\(formatDuration(dur))"))
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }
                        if let distM = order.etaDistanceM {
                            Text(L10n.format("mobile_driver.ui.formatetadistance", "\(formatETADistance(distM))"))
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }
                    }
                }
            }
            .padding(LabTheme.s16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: 14))
            .overlay {
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(LabTheme.separator, lineWidth: 0.5)
            }
            .padding(.horizontal, LabTheme.s20)

            VStack(spacing: 8) {
                Button {
                    Haptics.heavy()
                    vm.activeMission = mission
                    vm.refreshPlannedRoute()
                    withAnimation(Anim.sheetReveal) { phase = .activeDelivery; isCameraLocked = true; userPannedAt = nil }
                    startTelemetry()
                } label: {
                    Text("START_OPERATIONAL_FLOW")
                        .font(.system(size: 14, weight: .black, design: .monospaced))
                        .tracking(1.2)
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity).padding(.vertical, 18)
                        .background(LabTheme.fg)
                        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
                .buttonStyle(.pressable)

                // Navigate in Apple Maps
                Button {
                    Haptics.light()
                    openDestinationInMaps(lat: mission.target_lat, lng: mission.target_lng, name: mission.order_id)
                } label: {
                    HStack(spacing: 8) {
                        Image(systemName: "arrow.triangle.turn.up.right.diamond.fill")
                            .font(.system(size: 14, weight: .bold))
                        Text("EXTERNAL_NAVIGATION")
                            .font(.system(size: 12, weight: .black, design: .monospaced))
                            .tracking(1.2)
                    }
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity).padding(.vertical, 14)
                    .background {
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .fill(LabTheme.fg.opacity(0.06))
                            .overlay {
                                RoundedRectangle(cornerRadius: 16, style: .continuous)
                                    .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1)
                            }
                    }
                }
                .buttonStyle(.pressable)

                Button {
                    Haptics.light()
                    withAnimation(Anim.snappy) { selectedMission = nil; phase = .pickingOrder }
                } label: {
                    Text("mobile_driver.ui.choose_another")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(maxWidth: .infinity).padding(.vertical, 11)
                }
                .buttonStyle(.pressable)
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

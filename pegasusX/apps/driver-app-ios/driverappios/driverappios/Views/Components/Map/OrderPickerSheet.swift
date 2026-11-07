import SwiftUI
import MapKit

struct OrderPickerSheet: View {
    var vm: FleetViewModel
    @Binding var selectedMission: Mission?
    @Binding var phase: MapPhase
    @Binding var zoomFocus: ZoomFocus
    var bottomInset: Double
    var zoomTo: (ZoomFocus, Mission) -> Void
    
    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            SheetHandle()

            VStack(alignment: .leading, spacing: 4) {
                Text("mobile_driver.ui.select_order")
                    .font(.system(size: 9, weight: .heavy, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .tracking(1)
                Text("mobile_driver.ui.choose_a_delivery")
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
            }
            .padding(.horizontal, LabTheme.s20)
            .padding(.bottom, LabTheme.s12)

            if vm.isLoadingMissions {
                loadingRow
            } else if vm.pendingMissions.isEmpty {
                emptyRow
            } else {
                ScrollView {
                    LazyVStack(spacing: 6) {
                        ForEach(Array(vm.pendingMissions.enumerated()), id: \.element.id) { i, m in
                            pickerRow(m, index: i)
                        }
                    }
                    .padding(.horizontal, LabTheme.s16)
                    .padding(.bottom, LabTheme.s8)
                }
                .scrollIndicators(.hidden)
                .frame(maxHeight: 260)
            }
        }
        .padding(.bottom, bottomInset + LabTheme.s4)
        .background(GlassSheet())
    }
    
    private var loadingRow: some View {
        HStack(spacing: 8) {
            ProgressView().tint(LabTheme.fg)
            Text("supplier_portal.admin.audit_log.state.loading").font(.subheadline).foregroundStyle(LabTheme.fgSecondary)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 40)
    }

    private var emptyRow: some View {
        VStack(spacing: 8) {
            Image(systemName: "shippingbox").font(.system(size: 24)).foregroundStyle(LabTheme.fgTertiary)
            Text("mobile_driver.ui.no_pending_deliveries").font(.subheadline.weight(.medium)).foregroundStyle(LabTheme.fgSecondary)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 40)
    }
    
    private func pickerRow(_ mission: Mission, index: Int) -> some View {
        Button {
            Haptics.medium()
            withAnimation(Anim.sheetReveal) {
                selectedMission = mission
                phase = .previewingOrder
                zoomFocus = .both
            }
            zoomTo(.both, mission)
        } label: {
            HStack(spacing: 12) {
                ZStack {
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .fill(LabTheme.fg.opacity(0.06))
                        .frame(width: 38, height: 38)
                    Image(systemName: "shippingbox.fill")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(L10n.format("mobile_driver.ui.ord_uppercased", "\(mission.order_id.suffix(4).uppercased())"))
                        .font(.system(size: 13, weight: .black, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                    HStack(spacing: 4) {
                        Text(mission.gateway.uppercased())
                        Text("—")
                        Text(mission.amount.formattedAmount)
                    }
                    .font(.system(size: 10, weight: .bold, design: .monospaced))
                    .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                if let loc = vm.location {
                    let d = haversineDistance(from: loc, to: CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng))
                    Text(formattedDistance(d))
                        .font(.system(size: 10, weight: .black, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                Image(systemName: "chevron.right")
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s16)
            .background {
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .fill(LabTheme.fg.opacity(0.04))
                    .overlay {
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .stroke(LabTheme.separator.opacity(0.12), lineWidth: 1)
                    }
            }
        }
        .buttonStyle(.pressable)
        .staggeredAppear(index: index)
    }
}

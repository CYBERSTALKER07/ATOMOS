import MapKit
import SwiftUI

struct FleetLiveMapSection: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = FleetLiveMapViewModel()

    var mapHeight: CGFloat = 280
    var showsExpand: Bool = true
    var onExpand: (() -> Void)?

    var body: some View {
        @Bindable var vm = vm
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text("portal.nav.live_fleet")
                        .font(.headline)
                    Text(L10n.format("mobile_warehouse.ui.count_active_routecount_1_s", "\(vm.routes.count)", "\(vm.routes.count == 1 ? "" : "s")"))
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                .accessibilityElement(children: .combine)
                Spacer()
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .labelStyle(.iconOnly)
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh live fleet map")
                if showsExpand, let onExpand {
                    Button {
                        PegasusHaptics.light()
                        onExpand()
                    } label: {
                        Label("mobile_warehouse.ui.expand", systemImage: "arrow.up.left.and.arrow.down.right")
                            .labelStyle(.iconOnly)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Expand live fleet map")
                }
            }

            Group {
                if vm.loading && vm.routes.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: mapHeight)
                } else if let error = vm.error, vm.routes.isEmpty {
                    ContentUnavailableView("Live fleet unavailable", systemImage: "exclamationmark.triangle", description: Text(error))
                } else if vm.routes.isEmpty {
                    ContentUnavailableView(
                        "No active routes",
                        systemImage: "map",
                        description: Text("mobile_warehouse.ui.sealed_manifests_with_route_geometry_appear_during_dispatch")
                    )
                    .frame(maxWidth: .infinity, minHeight: mapHeight)
                } else {
                    TimelineView(.animation) { timeline in
                        let drivers = vm.animator.snapshot(now: timeline.date)
                        Map(position: $vm.cameraPosition) {
                            ForEach(Array(vm.routes.enumerated()), id: \.element.id) { index, route in
                                if let geometry = route.routeGeometry, geometry.coordinates.count >= 2 {
                                    let coordinates = geometry.coordinates.map {
                                        CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                                    }
                                    MapPolyline(coordinates: coordinates)
                                        .stroke(vm.routeColor(index), lineWidth: 4)
                                }
                            }
                            ForEach(drivers) { driver in
                                if let route = vm.routes.first(where: { $0.driverId == driver.id }) {
                                    Annotation(route.driverName ?? route.driverId, coordinate: CLLocationCoordinate2D(latitude: driver.lat, longitude: driver.lng)) {
                                        Circle()
                                            .fill(driver.stale ? vm.routeColor(for: route).opacity(0.45) : vm.routeColor(for: route))
                                            .frame(width: 14, height: 14)
                                            .overlay(Circle().stroke(.white, lineWidth: 2))
                                            .accessibilityLabel("Driver \(route.driverName ?? route.driverId)")
                                    }
                                }
                            }
                        }
                        .mapStyle(.standard(elevation: .realistic))
                        .frame(height: mapHeight)
                        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD, style: .continuous))
                    }
                }
            }
        }
        .labCard()
        .task {
            await vm.load()
            await vm.startPolling()
        }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await vm.load(silent: true) }
        }
    }
}

import MapKit
import SwiftUI

struct FleetLiveMapView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var viewModel = FleetLiveMapViewModel()

    private let publishActions = ["REROUTE", "FREEZE_DISPATCH", "PRIORITY_BOOST"]

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.routes.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.live_map")
            } else if let error = viewModel.error, viewModel.routes.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.routes.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_active_routes",
                    message: "mobile_supplier.empty.routes_message"
                )
            } else {
                VStack(spacing: 0) {
                    if let overlayError = viewModel.overlayError {
                        Text(overlayError)
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.destructive)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding(.horizontal)
                            .padding(.top, 8)
                    }
                    if !viewModel.zoneOverrides.isEmpty {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("mobile_supplier.ui.active_control_tower_zones")
                                .font(.subheadline.bold())
                            ForEach(viewModel.zoneOverrides.prefix(3)) { override in
                                Text(L10n.format("mobile_supplier.ui.action_expires_prefix", "\(override.action)", "\(override.ttlExpiresAt.prefix(19))"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding()
                        .background(.ultraThinMaterial)
                    }
                    TimelineView(.animation) { timeline in
                        let drivers = viewModel.animator.snapshot(now: timeline.date)
                        Map(position: $vm.cameraPosition) {
                            ForEach(Array(viewModel.routes.enumerated()), id: \.element.id) { index, route in
                                if let geometry = route.routeGeometry, geometry.coordinates.count >= 2 {
                                    let coordinates = geometry.coordinates.map {
                                        CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                                    }
                                    MapPolyline(coordinates: coordinates)
                                        .stroke(routeColor(index), lineWidth: 4)
                                }
                            }
                            ForEach(viewModel.exceptionCells) { cell in
                                Annotation("mobile_supplier.chrome.exception", coordinate: CLLocationCoordinate2D(latitude: cell.lat, longitude: cell.lng)) {
                                    Circle()
                                        .fill(exceptionColor(cell.severity))
                                        .frame(width: 12, height: 12)
                                        .overlay(Circle().stroke(.white, lineWidth: 1))
                                }
                            }
                            ForEach(drivers) { driver in
                                if let route = viewModel.routes.first(where: { $0.driverId == driver.id }) {
                                    Annotation(route.driverName ?? route.driverId, coordinate: CLLocationCoordinate2D(
                                        latitude: driver.lat,
                                        longitude: driver.lng
                                    )) {
                                        Circle()
                                            .fill(driver.stale ? routeColor(for: route).opacity(0.45) : routeColor(for: route))
                                            .frame(width: 14, height: 14)
                                            .overlay(Circle().stroke(.white, lineWidth: 2))
                                    }
                                }
                            }
                        }
                        .mapStyle(.standard(elevation: .realistic))
                        .frame(maxHeight: viewModel.exceptionCells.isEmpty ? .infinity : 320)
                        .onMapCameraChange(frequency: .onEnd) { context in
                            viewModel.mapRegion = context.region
                        }
                    }
                    if !viewModel.exceptionCells.isEmpty {
                        ResponsiveGridContentWrapper {
                            ForEach(viewModel.exceptionCells) { cell in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(L10n.format("mobile_supplier.ui.cell_prefix_severity", "\(cell.h3Cell.prefix(12))", "\(cell.severity)"))
                                        .font(.subheadline.bold())
                                    Text(L10n.format("mobile_supplier.chrome.cell_shop_closed", "\(cell.counts["total", default: 0])", "\(cell.counts["shop_closed", default: 0])"))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .accessibilityElement(children: .combine)
                            }
                        }
                    }
                }
            }
        }
        .navigationTitle("portal.nav.live_fleet")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_supplier.ui.publish_zone", systemImage: "map") {
                    viewModel.showPublishSheet = true
                }
            }
        }
        .sheet(isPresented: $vm.showPublishSheet) {
            NavigationStack {
                Form {
                    Picker("mobile_supplier.chrome.action", selection: $vm.publishAction) {
                        ForEach(publishActions, id: \.self) { action in
                            Text(action).tag(action)
                        }
                    }
                    if let publishStatus = viewModel.publishStatus {
                        Text(publishStatus)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                .navigationTitle("supplier_portal.control_tower_command_panel.text.control_tower")
                .toolbar {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(LocalizedStringKey(viewModel.publishing ? "mobile_supplier.chrome.publishing" : "mobile_supplier.chrome.publish")) {
                            Task { await viewModel.publishZoneOverride() }
                        }
                        .disabled(viewModel.publishing)
                    }
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.close") { viewModel.showPublishSheet = false }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .task {
            await viewModel.load()
            await viewModel.startPolling()
        }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await viewModel.load(silent: true) }
        }
    }

    private func routeColor(_ index: Int) -> Color {
        let palette: [Color] = [.blue, .green, .red, .orange, .purple, .teal]
        return palette[index % palette.count]
    }

    private func routeColor(for route: SupplierFleetLiveRoute) -> Color {
        guard let index = viewModel.routes.firstIndex(where: { $0.id == route.id }) else { return .blue }
        return routeColor(index)
    }

    private func exceptionColor(_ severity: String) -> Color {
        switch severity.lowercased() {
        case "high": return SupplierTheme.destructive
        case "medium": return SupplierTheme.warning
        default: return SupplierTheme.warning
        }
    }
}

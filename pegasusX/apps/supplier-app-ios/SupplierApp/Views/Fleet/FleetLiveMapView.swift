import MapKit
import SwiftUI

struct FleetLiveMapView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var routes: [SupplierFleetLiveRoute] = []
    @State private var exceptionCells: [ExceptionMapCell] = []
    @State private var zoneOverrides: [ControlTowerZoneOverride] = []
    @State private var loading = true
    @State private var error: String?
    @State private var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
            span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
        )
    )
    @State private var animator = FleetDriverMarkerAnimator()
    @State private var publishAction = "REROUTE"
    @State private var publishStatus: String?
    @State private var publishing = false
    @State private var showPublishSheet = false
    @State private var mapRegion = MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
        span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
    )

    private let publishActions = ["REROUTE", "FREEZE_DISPATCH", "PRIORITY_BOOST"]

    var body: some View {
        Group {
            if loading && routes.isEmpty {
                SupplierLoadingView(title: "Loading live fleet map…")
            } else if let error, routes.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else if routes.isEmpty {
                SupplierEmptyView(
                    title: "No active routes",
                    message: "Sealed manifests with route geometry will appear here during dispatch."
                )
            } else {
                VStack(spacing: 0) {
                if !zoneOverrides.isEmpty {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("mobile_supplier.ui.active_control_tower_zones")
                            .font(.subheadline.bold())
                        ForEach(zoneOverrides.prefix(3)) { override in
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
                    let drivers = animator.snapshot(now: timeline.date)
                    Map(position: $cameraPosition) {
                        ForEach(Array(routes.enumerated()), id: \.element.id) { index, route in
                            if let geometry = route.routeGeometry, geometry.coordinates.count >= 2 {
                                let coordinates = geometry.coordinates.map {
                                    CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                                }
                                MapPolyline(coordinates: coordinates)
                                    .stroke(routeColor(index), lineWidth: 4)
                            }
                        }
                        ForEach(exceptionCells) { cell in
                            Annotation("Exception", coordinate: CLLocationCoordinate2D(latitude: cell.lat, longitude: cell.lng)) {
                                Circle()
                                    .fill(exceptionColor(cell.severity))
                                    .frame(width: 12, height: 12)
                                    .overlay(Circle().stroke(.white, lineWidth: 1))
                            }
                        }
                        ForEach(drivers) { driver in
                            if let route = routes.first(where: { $0.driverId == driver.id }) {
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
                    .frame(maxHeight: exceptionCells.isEmpty ? .infinity : 320)
                    .onMapCameraChange(frequency: .onEnd) { context in
                        mapRegion = context.region
                    }
                }
                if !exceptionCells.isEmpty {
                    ResponsiveGridContentWrapper {
                        ForEach(exceptionCells) { cell in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(L10n.format("mobile_supplier.ui.cell_prefix_severity", "\(cell.h3Cell.prefix(12))", "\(cell.severity)"))
                                    .font(.subheadline.bold())
                                Text("Total \(cell.counts["total", default: 0]) · shop closed \(cell.counts["shop_closed", default: 0])")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
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
                    showPublishSheet = true
                }
            }
        }
        .sheet(isPresented: $showPublishSheet) {
            NavigationStack {
                Form {
                    Picker("Action", selection: $publishAction) {
                        ForEach(publishActions, id: \.self) { action in
                            Text(action).tag(action)
                        }
                    }
                    if let publishStatus {
                        Text(publishStatus)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                }
                .navigationTitle("supplier_portal.control_tower_command_panel.text.control_tower")
                .toolbar {
                    ToolbarItem(placement: .confirmationAction) {
                        Button(publishing ? "Publishing…" : "Publish") {
                            Task { await publishZoneOverride() }
                        }
                        .disabled(publishing)
                    }
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.close") { showPublishSheet = false }
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .task {
            await load()
            await startPolling()
        }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await load(silent: true) }
        }
    }

    private func routeColor(_ index: Int) -> Color {
        let palette: [Color] = [.blue, .green, .red, .orange, .purple, .teal]
        return palette[index % palette.count]
    }

    private func routeColor(for route: SupplierFleetLiveRoute) -> Color {
        guard let index = routes.firstIndex(where: { $0.id == route.id }) else { return .blue }
        return routeColor(index)
    }

    private func exceptionColor(_ severity: String) -> Color {
        switch severity.lowercased() {
        case "high": return .red
        case "medium": return .orange
        default: return .yellow
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let response = try await SupplierOperationsService.fleetLiveMap()
            routes = response.routes
            animator.updateTargets(routes)
            if let map = try? await SupplierOperationsService.exceptionMap() {
                exceptionCells = map.cells
            }
            if let overrides = try? await SupplierOperationsService.controlTowerZoneOverrides() {
                zoneOverrides = overrides
            }
            fitCamera()
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    private func fitCamera() {
        var coordinates: [CLLocationCoordinate2D] = []
        for route in routes {
            if let geometry = route.routeGeometry {
                coordinates.append(contentsOf: geometry.coordinates.map {
                    CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                })
            }
            if let location = route.driverLocation, route.liveLocationAvailable {
                coordinates.append(CLLocationCoordinate2D(latitude: location.latitude, longitude: location.longitude))
            }
        }
        guard !coordinates.isEmpty else { return }
        let lats = coordinates.map(\.latitude)
        let lngs = coordinates.map(\.longitude)
        let center = CLLocationCoordinate2D(
            latitude: (lats.min()! + lats.max()!) / 2,
            longitude: (lngs.min()! + lngs.max()!) / 2
        )
        let span = MKCoordinateSpan(
            latitudeDelta: max(0.03, (lats.max()! - lats.min()!) * 1.4),
            longitudeDelta: max(0.03, (lngs.max()! - lngs.min()!) * 1.4)
        )
        cameraPosition = .region(MKCoordinateRegion(center: center, span: span))
    }

    private func startPolling() async {
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: 15_000_000_000)
            await load(silent: true)
        }
    }

    @MainActor
    private func publishZoneOverride() async {
        publishing = true
        publishStatus = nil
        defer { publishing = false }
        let polygon = polygonFromCurrentCamera()
        let fingerprint = polygon.coordinates.description
        let scope = SupplierIdempotencyKeys.supplierScopeId()
        let key = SupplierIdempotencyKeys.controlTowerZoneOverride(
            scopeId: scope,
            action: publishAction,
            polygonFingerprint: fingerprint
        )
        let body = ControlTowerZoneOverrideCreateRequest(
            action: publishAction,
            ttlSeconds: 1800,
            polygonGeojson: polygon
        )
        do {
            let row = try await SupplierOperationsService.createControlTowerZoneOverride(body, idempotencyKey: key)
            publishStatus = "Override \(row.overrideId.prefix(8)) active"
            zoneOverrides = (try? await SupplierOperationsService.controlTowerZoneOverrides()) ?? zoneOverrides
        } catch {
            publishStatus = error.localizedDescription
        }
    }

    private func polygonFromCurrentCamera() -> GeoJSONPolygonPayload {
        let center = mapRegion.center
        let latD = mapRegion.span.latitudeDelta / 2
        let lngD = mapRegion.span.longitudeDelta / 2
        let ring: [[Double]] = [
            [center.longitude - lngD, center.latitude - latD],
            [center.longitude + lngD, center.latitude - latD],
            [center.longitude + lngD, center.latitude + latD],
            [center.longitude - lngD, center.latitude + latD],
            [center.longitude - lngD, center.latitude - latD],
        ]
        return GeoJSONPolygonPayload(coordinates: [ring])
    }
}

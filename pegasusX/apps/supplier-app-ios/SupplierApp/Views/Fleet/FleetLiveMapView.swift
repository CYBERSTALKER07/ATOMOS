import MapKit
import SwiftUI

struct FleetLiveMapView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var routes: [SupplierFleetLiveRoute] = []
    @State private var exceptionCells: [ExceptionMapCell] = []
    @State private var loading = true
    @State private var error: String?
    @State private var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
            span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
        )
    )
    @State private var animator = FleetDriverMarkerAnimator()

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
                }
                if !exceptionCells.isEmpty {
                    List(exceptionCells) { cell in
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Cell \(cell.h3Cell.prefix(12))… · \(cell.severity)")
                                .font(.subheadline.bold())
                            Text("Total \(cell.counts["total", default: 0]) · shop closed \(cell.counts["shop_closed", default: 0])")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .listStyle(.plain)
                }
                }
            }
        }
        .navigationTitle("Live fleet")
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
}

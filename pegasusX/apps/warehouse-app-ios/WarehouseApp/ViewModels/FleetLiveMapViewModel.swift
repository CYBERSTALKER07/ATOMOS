import Foundation
import MapKit
import Observation
import SwiftUI

@Observable
@MainActor
final class FleetLiveMapViewModel {
    typealias FetchLiveMap = @MainActor (String?) async throws -> WarehouseFleetLiveMapResponse
    typealias WarehouseIdProvider = @MainActor () -> String?

    var routes: [WarehouseFleetLiveRoute] = []
    var loading = true
    var error: String?
    var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
            span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
        )
    )
    var animator = FleetDriverMarkerAnimator()
    var hasFittedInitialCamera = false

    private let fetchLiveMap: FetchLiveMap
    private let warehouseIdProvider: WarehouseIdProvider
    private let pollIntervalNanoseconds: UInt64

    init(
        fetchLiveMap: FetchLiveMap? = nil,
        warehouseIdProvider: WarehouseIdProvider? = nil,
        pollIntervalNanoseconds: UInt64 = 15_000_000_000
    ) {
        self.fetchLiveMap = fetchLiveMap ?? { warehouseId in
            try await WarehouseService.fleetLiveMap(warehouseId: warehouseId)
        }
        self.warehouseIdProvider = warehouseIdProvider ?? {
            TokenStore.shared.warehouseId
        }
        self.pollIntervalNanoseconds = pollIntervalNanoseconds
    }

    func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { if !silent { loading = false } }
        do {
            let response = try await fetchLiveMap(warehouseIdProvider())
            routes = response.routes
            animator.updateTargets(routes)
            if !silent || !hasFittedInitialCamera {
                fitCamera()
                if !routes.isEmpty {
                    hasFittedInitialCamera = true
                }
            }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    func startPolling() async {
        while !Task.isCancelled {
            try? await Task.sleep(nanoseconds: pollIntervalNanoseconds)
            guard !Task.isCancelled else { break }
            await load(silent: true)
        }
    }

    func fitCamera() {
        var coordinates: [CLLocationCoordinate2D] = []
        for route in routes {
            if let geometry = route.routeGeometry {
                coordinates.append(contentsOf: geometry.coordinates.map {
                    CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                })
            }
            if let location = route.driverLocation, route.liveLocationAvailable {
                coordinates.append(CLLocationCoordinate2D(latitude: location.resolvedLatitude, longitude: location.resolvedLongitude))
            }
        }
        guard !coordinates.isEmpty else { return }
        let lats = coordinates.map(\.latitude)
        let lngs = coordinates.map(\.longitude)
        guard let minLat = lats.min(), let maxLat = lats.max(),
              let minLng = lngs.min(), let maxLng = lngs.max() else {
            return
        }
        let center = CLLocationCoordinate2D(
            latitude: (minLat + maxLat) / 2,
            longitude: (minLng + maxLng) / 2
        )
        let span = MKCoordinateSpan(
            latitudeDelta: max(0.03, (maxLat - minLat) * 1.4),
            longitudeDelta: max(0.03, (maxLng - minLng) * 1.4)
        )
        cameraPosition = .region(MKCoordinateRegion(center: center, span: span))
    }

    func routeColor(_ index: Int) -> Color {
        let palette: [Color] = [
            TacticalTheme.accentCobalt,
            TacticalTheme.accentOrange,
            TacticalTheme.accentLime,
            TacticalTheme.accentCyan,
            PegasusMonochromeTheme.label,
            PegasusMonochromeTheme.secondaryLabel
        ]
        return palette[index % palette.count]
    }

    func routeColor(for route: WarehouseFleetLiveRoute) -> Color {
        guard let index = routes.firstIndex(where: { $0.id == route.id }) else {
            return TacticalTheme.accentCobalt
        }
        return routeColor(index)
    }
}

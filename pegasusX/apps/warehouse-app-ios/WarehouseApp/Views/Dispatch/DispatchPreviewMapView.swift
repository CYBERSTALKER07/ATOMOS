import MapKit
import SwiftUI

struct DispatchPreviewMapView: View {
    let routes: [DispatchProposedRoute]

    @State private var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
            span: MKCoordinateSpan(latitudeDelta: 0.18, longitudeDelta: 0.18)
        )
    )

    private var routesWithGeometry: [DispatchProposedRoute] {
        routes.filter { ($0.routeGeometry?.coordinates.count ?? 0) >= 2 }
    }

    var body: some View {
        Group {
            if routesWithGeometry.isEmpty {
                Text("supplier_portal.dispatch_preview_map.text.route_preview_unavailable_until_optimizer_proposes_stops_with_co")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)
                    .frame(maxWidth: .infinity, minHeight: 200)
                    .padding()
                    .background(.quaternary.opacity(0.25), in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            } else {
                Map(position: $cameraPosition) {
                    ForEach(Array(routesWithGeometry.enumerated()), id: \.offset) { index, route in
                        if let geometry = route.routeGeometry, geometry.coordinates.count >= 2 {
                            let coordinates = geometry.coordinates.map {
                                CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                            }
                            MapPolyline(coordinates: coordinates)
                                .stroke(dispatchRouteColor(index), lineWidth: 4)
                        }
                    }
                }
                .mapStyle(.standard(elevation: .realistic))
                .frame(height: 240)
                .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD))
                .onAppear { fitCamera() }
                .onChange(of: routes.map(\.id)) { _, _ in fitCamera() }
            }
        }
    }

    private func fitCamera() {
        var coordinates: [CLLocationCoordinate2D] = []
        for route in routesWithGeometry {
            if let geometry = route.routeGeometry {
                coordinates.append(contentsOf: geometry.coordinates.map {
                    CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                })
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

    private func dispatchRouteColor(_ index: Int) -> Color {
        let palette: [Color] = [.blue, .green, .red, .orange, .purple, .teal]
        return palette[index % palette.count]
    }
}

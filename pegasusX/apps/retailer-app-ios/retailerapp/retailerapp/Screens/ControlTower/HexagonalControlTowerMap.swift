import SwiftUI
import MapKit

struct HexagonalControlTowerMap: View {
    @State private var cameraPosition: MapCameraPosition = .camera(
        MapCamera(
            centerCoordinate: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
            distance: 10000,
            heading: 0,
            pitch: 45
        )
    )
    
    
    var body: some View {
        Map(position: $cameraPosition)
            .mapStyle(.standard(elevation: .realistic))
    }
    
    private func colorForDensity(_ density: Double) -> Color {
        if density > 0.8 {
            return .red
        } else if density > 0.5 {
            return .orange
        } else if density > 0.2 {
            return .yellow
        } else {
            return .blue
        }
    }
}

#Preview {
    HexagonalControlTowerMap()
}

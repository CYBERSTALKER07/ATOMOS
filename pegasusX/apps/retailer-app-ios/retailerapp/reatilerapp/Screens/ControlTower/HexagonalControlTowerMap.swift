import SwiftUI
import MapKit

struct HexagonalControlTowerMap: View {
    @State private var cameraPosition: MapCameraPosition = .camera(
        MapCamera(
            centerCoordinate: CLLocationCoordinate2D(latitude: 37.7749, longitude: -122.4194),
            distance: 10000,
            heading: 0,
            pitch: 45
        )
    )
    
    // We will generate the grid on init to simulate the H3 Hex bins from the backend
    @State private var hexGrid: [MockH3Index] = []
    
    var body: some View {
        Map(position: $cameraPosition) {
            ForEach(hexGrid, id: \.value) { hex in
                let vertices = MockH3.getHexagonVertices(center: hex.center, radiusMeters: 400)
                
                MapPolygon(coordinates: vertices)
                    .foregroundStyle(colorForDensity(hex.density).opacity(0.6))
                    .stroke(.white.opacity(0.2), lineWidth: 1)
            }
        }
        .mapStyle(.standard(elevation: .realistic))
        .onAppear {
            generateData()
        }
    }
    
    private func generateData() {
        let sfCenter = CLLocationCoordinate2D(latitude: 37.7749, longitude: -122.4194)
        // Generate hex grid of 400m radius covering a 5km radius area
        hexGrid = MockH3.generateGrid(center: sfCenter, radiusInMeters: 5000, hexSizeMeters: 400)
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

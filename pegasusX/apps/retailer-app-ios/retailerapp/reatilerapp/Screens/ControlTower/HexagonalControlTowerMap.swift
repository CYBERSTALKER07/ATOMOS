import SwiftUI
import MapKit

struct HexagonalControlTowerMap: View {
    @State private var region = MKCoordinateRegion(
        center: CLLocationCoordinate2D(latitude: 37.7749, longitude: -122.4194),
        span: MKCoordinateSpan(latitudeDelta: 0.1, longitudeDelta: 0.1)
    )

    var body: some View {
        ZStack {
            MapViewRepresentable(region: $region)
                .edgesIgnoringSafeArea(.all)
                .colorScheme(.dark) // force dark map
            
            VStack {
                Text("Predictive Order Density")
                    .font(.headline)
                    .foregroundColor(.white)
                    .padding()
                    .background(Color.black.opacity(0.7))
                    .cornerRadius(8)
                    .padding(.top, 16)
                Spacer()
            }
        }
    }
}

struct MapViewRepresentable: UIViewRepresentable {
    @Binding var region: MKCoordinateRegion

    func makeUIView(context: Context) -> MKMapView {
        let mapView = MKMapView()
        mapView.delegate = context.coordinator
        mapView.setRegion(region, animated: false)
        mapView.overrideUserInterfaceStyle = .dark
        
        // Add dummy hexagons to simulate H3
        let hexagons = generateMockHexagons(center: region.center)
        for hex in hexagons {
            let polygon = MKPolygon(coordinates: hex.coordinates, count: hex.coordinates.count)
            polygon.title = hex.colorHex
            mapView.addOverlay(polygon)
        }
        
        return mapView
    }

    func updateUIView(_ uiView: MKMapView, context: Context) {}

    func makeCoordinator() -> Coordinator {
        Coordinator(self)
    }

    class Coordinator: NSObject, MKMapViewDelegate {
        var parent: MapViewRepresentable

        init(_ parent: MapViewRepresentable) {
            self.parent = parent
        }

        func mapView(_ mapView: MKMapView, rendererFor overlay: MKOverlay) -> MKOverlayRenderer {
            if let polygon = overlay as? MKPolygon {
                let renderer = MKPolygonRenderer(polygon: polygon)
                let colorHex = polygon.title ?? "#10B981"
                renderer.fillColor = UIColor(Color(hex: colorHex)).withAlphaComponent(0.4)
                renderer.strokeColor = UIColor(Color(hex: colorHex))
                renderer.lineWidth = 2
                return renderer
            }
            return MKOverlayRenderer(overlay: overlay)
        }
    }
    
    struct Hex {
        let coordinates: [CLLocationCoordinate2D]
        let colorHex: String
    }
    
    private func generateMockHexagons(center: CLLocationCoordinate2D) -> [Hex] {
        var hexes: [Hex] = []
        let offsets = [
            (0.0, 0.0), (0.01, 0.0), (-0.01, 0.0),
            (0.005, 0.008), (-0.005, 0.008),
            (0.005, -0.008), (-0.005, -0.008)
        ]
        
        for (idx, offset) in offsets.enumerated() {
            let c = CLLocationCoordinate2D(latitude: center.latitude + offset.0, longitude: center.longitude + offset.1)
            let radius = 0.005
            var coords: [CLLocationCoordinate2D] = []
            for i in 0..<6 {
                let angle = Double(i) * .pi / 3.0
                let lat = c.latitude + radius * cos(angle)
                let lon = c.longitude + radius * sin(angle)
                coords.append(CLLocationCoordinate2D(latitude: lat, longitude: lon))
            }
            let color = idx % 2 == 0 ? "#10B981" : "#3B82F6"
            hexes.append(Hex(coordinates: coords, colorHex: color))
        }
        
        return hexes
    }
}

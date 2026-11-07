import SwiftUI
import MapKit

private struct HexCellZone: Identifiable {
    let id: String
    let label: String
    let center: CLLocationCoordinate2D
    let density: Double
    let vertices: [CLLocationCoordinate2D]
}

struct HexagonalControlTowerMap: View {
    @State private var cameraPosition: MapCameraPosition = .camera(
        MapCamera(
            centerCoordinate: CLLocationCoordinate2D(latitude: packMapCoordinate().lat, longitude: packMapCoordinate().lng),
            distance: 10000,
            heading: 0,
            pitch: 45
        )
    )
    @State private var selectedCellId: String?

    private var hexCells: [HexCellZone] {
        let origin = packMapCoordinate()
        let radiusLat = 0.0085
        let radiusLng = 0.0110
        let ringOffsets: [(id: String, label: String, dLat: Double, dLng: Double, density: Double)] = [
            ("h3-core-0", "Core Hub", 0.0, 0.0, 0.88),
            ("h3-ring-1", "North Sector", radiusLat * 1.65, 0.0, 0.64),
            ("h3-ring-2", "NE Corridor", radiusLat * 0.82, radiusLng * 1.45, 0.52),
            ("h3-ring-3", "SE Industrial", -radiusLat * 0.82, radiusLng * 1.45, 0.34),
            ("h3-ring-4", "South Depot", -radiusLat * 1.65, 0.0, 0.72),
            ("h3-ring-5", "SW Retail Belt", -radiusLat * 0.82, -radiusLng * 1.45, 0.45),
            ("h3-ring-6", "NW Cross-Dock", radiusLat * 0.82, -radiusLng * 1.45, 0.18),
        ]
        return ringOffsets.map { item in
            let center = CLLocationCoordinate2D(latitude: origin.lat + item.dLat, longitude: origin.lng + item.dLng)
            let vertices = (0..<6).map { idx -> CLLocationCoordinate2D in
                let angle = Double(idx) * (.pi / 3.0)
                return CLLocationCoordinate2D(
                    latitude: center.latitude + radiusLat * sin(angle),
                    longitude: center.longitude + radiusLng * cos(angle)
                )
            }
            return HexCellZone(
                id: item.id,
                label: item.label,
                center: center,
                density: item.density,
                vertices: vertices
            )
        }
    }

    var body: some View {
        ZStack(alignment: .bottomLeading) {
            Map(position: $cameraPosition) {
                ForEach(hexCells) { cell in
                    let tone = colorForDensity(cell.density)
                    let isSelected = selectedCellId == cell.id
                    MapPolygon(coordinates: cell.vertices)
                        .foregroundStyle(tone.opacity(isSelected ? 0.36 : 0.22))
                        .stroke(tone.opacity(isSelected ? 0.95 : 0.65), lineWidth: isSelected ? 2.5 : 1.5)
                    Annotation(cell.label, coordinate: cell.center) {
                        Button {
                            selectedCellId = cell.id
                        } label: {
                            Text("\(Int(cell.density * 100))%")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .padding(.horizontal, 6)
                                .padding(.vertical, 3)
                                .background(.ultraThinMaterial, in: Capsule())
                        }
                        .buttonStyle(.plain)
                    }
                }
            }
            .mapStyle(.standard(elevation: .realistic))

            HStack(spacing: 10) {
                legendDot(color: .blue, label: "Low")
                legendDot(color: .yellow, label: "Moderate")
                legendDot(color: .orange, label: "High")
                legendDot(color: .red, label: "Surge")
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(.ultraThinMaterial, in: Capsule())
            .padding(12)
        }
    }

    private func legendDot(color: Color, label: String) -> some View {
        HStack(spacing: 4) {
            Circle()
                .fill(color)
                .frame(width: 8, height: 8)
            Text(label)
                .font(.system(.caption2, design: .rounded, weight: .medium))
        }
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

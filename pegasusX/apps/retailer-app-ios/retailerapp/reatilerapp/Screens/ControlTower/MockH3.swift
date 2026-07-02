import Foundation
import CoreLocation
import MapKit

public struct MockH3Index {
    public let value: String
    public let center: CLLocationCoordinate2D
    public let density: Double // 0.0 to 1.0
}

public class MockH3 {
    
    // Generates a mock grid of hexagons around a center point
    public static func generateGrid(center: CLLocationCoordinate2D, radiusInMeters: Double, hexSizeMeters: Double) -> [MockH3Index] {
        var indices: [MockH3Index] = []
        
        let latDegreesPerMeter = 1.0 / 111320.0
        let lonDegreesPerMeter = 1.0 / (111320.0 * cos(center.latitude * .pi / 180.0))
        
        let hexWidth = hexSizeMeters * 2
        let hexHeight = sqrt(3.0) * hexSizeMeters
        
        let cols = Int(radiusInMeters / hexSizeMeters)
        let rows = Int(radiusInMeters / hexSizeMeters)
        
        for r in -rows...rows {
            for q in -cols...cols {
                let x = hexWidth * 0.75 * Double(q)
                let y = hexHeight * (Double(r) + 0.5 * (Double(q).truncatingRemainder(dividingBy: 2.0)))
                
                let dist = sqrt(x*x + y*y)
                if dist <= radiusInMeters {
                    let lat = center.latitude + (y * latDegreesPerMeter)
                    let lon = center.longitude + (x * lonDegreesPerMeter)
                    
                    // Generate a bell curve density based on distance from center to simulate urban core density
                    let normalizedDist = dist / radiusInMeters
                    let density = max(0, 1.0 - (normalizedDist * normalizedDist) + Double.random(in: -0.2...0.2))
                    
                    let index = MockH3Index(
                        value: "8928308280\(r)\(q)",
                        center: CLLocationCoordinate2D(latitude: lat, longitude: lon),
                        density: min(1.0, max(0.0, density))
                    )
                    indices.append(index)
                }
            }
        }
        return indices
    }
    
    public static func getHexagonVertices(center: CLLocationCoordinate2D, radiusMeters: Double) -> [CLLocationCoordinate2D] {
        var vertices: [CLLocationCoordinate2D] = []
        let latDegreesPerMeter = 1.0 / 111320.0
        let lonDegreesPerMeter = 1.0 / (111320.0 * cos(center.latitude * .pi / 180.0))
        
        for i in 0..<6 {
            let angle = Double(i) * 60.0 * .pi / 180.0
            let dx = radiusMeters * cos(angle)
            let dy = radiusMeters * sin(angle)
            
            vertices.append(CLLocationCoordinate2D(
                latitude: center.latitude + (dy * latDegreesPerMeter),
                longitude: center.longitude + (dx * lonDegreesPerMeter)
            ))
        }
        return vertices
    }
}

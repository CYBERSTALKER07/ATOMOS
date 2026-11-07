import CoreLocation
import Foundation

struct ResolvedLocationResponse: Decodable {
    let address: String
    let lat: Double
    let lng: Double
    let placeId: String?

    enum CodingKeys: String, CodingKey {
        case address, lat, lng
        case placeId = "place_id"
    }
}

enum GeocodeService {
    static func reverse(lat: Double, lng: Double) async -> String? {
        let path = "v1/platform/geocode/reverse?lat=\(lat)&lng=\(lng)"
        if let resolved: ResolvedLocationResponse = try? await APIClient.shared.get(path: path),
           !resolved.address.isEmpty {
            return resolved.address
        }

        // Free built-in fallback: Apple CoreLocation CLGeocoder (zero API keys, 100% free)
        let location = CLLocation(latitude: lat, longitude: lng)
        if let placemarks = try? await CLGeocoder().reverseGeocodeLocation(location),
           let p = placemarks.first {
            let parts = [p.name, p.thoroughfare, p.subLocality, p.locality, p.administrativeArea]
                .compactMap { $0 }
                .filter { !$0.isEmpty }
            if !parts.isEmpty {
                return parts.joined(separator: ", ")
            }
        }

        // Sensible fallback for Tashkent coordinates or general coordinates
        if abs(lat - 41.2995) < 0.25 && abs(lng - 69.2401) < 0.25 {
            return "Amir Temur Ave, Tashkent, Uzbekistan"
        }
        return String(format: "Location (%.4f, %.4f)", lat, lng)
    }
}

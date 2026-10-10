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
        guard let resolved: ResolvedLocationResponse = try? await APIClient.shared.get(path: path) else {
            return nil
        }
        return resolved.address.isEmpty ? nil : resolved.address
    }
}

import Foundation

struct WarehouseLocationResponse: Decodable {
    let warehouseId: String
    let name: String
    let address: String
    let placeId: String?
    let lat: Double
    let lng: Double

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case name, address
        case placeId = "place_id"
        case lat, lng
    }
}

struct WarehouseLocationPatchRequest: Encodable {
    let address: String
    let placeId: String?
    let lat: Double
    let lng: Double

    enum CodingKeys: String, CodingKey {
        case address
        case placeId = "place_id"
        case lat, lng
    }
}

struct WarehouseSetupRequest: Encodable {
    let name: String
    let address: String
    let placeId: String?
    let lat: Double
    let lng: Double

    enum CodingKeys: String, CodingKey {
        case name, address, lat, lng
        case placeId = "place_id"
    }
}

struct WarehouseSetupResponse: Decodable {
    let warehouseId: String
    let token: String?
    let refreshToken: String?
    let isConfigured: Bool

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case token
        case refreshToken = "refresh_token"
        case isConfigured = "is_configured"
    }
}

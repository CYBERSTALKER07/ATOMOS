import Foundation

struct WarehouseLocationResponse: Decodable {
    let warehouseId: String
    let name: String
    let address: String
    let placeId: String?
    let lat: Double
    let lng: Double
    let countryCode: String
    let packCountryCode: String
    let currencyCode: String

    enum CodingKeys: String, CodingKey {
        case warehouseId = "warehouse_id"
        case name, address
        case placeId = "place_id"
        case lat, lng
        case countryCode = "country_code"
        case packCountryCode = "pack_country_code"
        case currencyCode = "currency_code"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        warehouseId = try c.decodeIfPresent(String.self, forKey: .warehouseId) ?? ""
        name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        address = try c.decodeIfPresent(String.self, forKey: .address) ?? ""
        placeId = try c.decodeIfPresent(String.self, forKey: .placeId)
        lat = try c.decodeIfPresent(Double.self, forKey: .lat) ?? 0
        lng = try c.decodeIfPresent(Double.self, forKey: .lng) ?? 0
        countryCode = try c.decodeIfPresent(String.self, forKey: .countryCode) ?? ""
        packCountryCode = try c.decodeIfPresent(String.self, forKey: .packCountryCode) ?? countryCode
        currencyCode = try c.decodeIfPresent(String.self, forKey: .currencyCode) ?? ""
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

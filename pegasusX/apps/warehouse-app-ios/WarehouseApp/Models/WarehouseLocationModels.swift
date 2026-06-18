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

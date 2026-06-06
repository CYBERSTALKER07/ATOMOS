import Foundation

// MARK: - Login Request

struct LoginRequest: Encodable {
    let phoneNumber: String
    let password: String

    enum CodingKeys: String, CodingKey {
        case phoneNumber = "phone_number"
        case password
    }
}

// MARK: - Register Request

struct RegisterRequest: Encodable {
    let phoneNumber: String
    let password: String
    let storeName: String
    let ownerName: String
    let addressText: String
    let latitude: Double
    let longitude: Double
    let taxId: String?
    let receivingWindowOpen: String?
    let receivingWindowClose: String?
    let accessType: String?
    let storageCeilingHeightCM: Double?

    enum CodingKeys: String, CodingKey {
        case phoneNumber = "phone_number"
        case password
        case storeName = "store_name"
        case ownerName = "owner_name"
        case addressText = "address_text"
        case latitude, longitude
        case taxId = "tax_id"
        case receivingWindowOpen = "receiving_window_open"
        case receivingWindowClose = "receiving_window_close"
        case accessType = "access_type"
        case storageCeilingHeightCM = "storage_ceiling_height_cm"
    }
}

import Foundation

struct RetailerCardToken: Identifiable, Codable {
    var id: String { tokenId } // Convenience for Identifiable
    
    let tokenId: String
    let retailerId: String
    let gateway: String
    let cardLast4: String
    let cardType: String
    let isDefault: Bool
    let isActive: Bool
    let expiresAt: Date?
    let createdAt: Date
    
    enum CodingKeys: String, CodingKey {
        case tokenId = "token_id"
        case retailerId = "retailer_id"
        case gateway
        case cardLast4 = "card_last4"
        case cardType = "card_type"
        case isDefault = "is_default"
        case isActive = "is_active"
        case expiresAt = "expires_at"
        case createdAt = "created_at"
    }
}

struct RetailerCardsResponse: Codable {
    let cards: [RetailerCardToken]
}

struct PSPListing: Codable, Hashable {
    let code: String
    let status: String
    let selectable: Bool
    let nationalCards: Bool?

    enum CodingKeys: String, CodingKey {
        case code, status, selectable
        case nationalCards = "national_cards"
    }
}

struct RetailerPaymentCatalogResponse: Codable {
    let currencyCode: String
    let marketCode: String?
    let catalog: [PSPListing]

    enum CodingKeys: String, CodingKey {
        case currencyCode = "currency_code"
        case marketCode = "market_code"
        case catalog
    }
}

struct CardInitiateRequest: Codable {
    let gateway: String
}

struct CardInitiateResponse: Codable {
    let cardToken: String
    let requiresOtp: Bool
    
    enum CodingKeys: String, CodingKey {
        case cardToken = "card_token"
        case requiresOtp = "requires_otp"
    }
}

struct CardConfirmRequest: Codable {
    let cardToken: String
    let otpCode: String
    
    enum CodingKeys: String, CodingKey {
        case cardToken = "card_token"
        case otpCode = "otp_code"
    }
}

struct CardConfirmResponse: Codable {
    let tokenId: String
    let cardLast4: String
    let cardType: String
    let confirmed: Bool
    
    enum CodingKeys: String, CodingKey {
        case tokenId = "token_id"
        case cardLast4 = "card_last4"
        case cardType = "card_type"
        case confirmed
    }
}

struct CardIdRequest: Codable {
    let tokenId: String
    
    enum CodingKeys: String, CodingKey {
        case tokenId = "token_id"
    }
}

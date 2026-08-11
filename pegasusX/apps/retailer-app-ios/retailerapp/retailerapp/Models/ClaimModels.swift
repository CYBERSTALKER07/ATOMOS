import Foundation

// MARK: - Logistics claims (post-delivery)

struct RetailerClaimLine: Codable, Identifiable, Hashable {
    var id: String { sku }
    let sku: String
    let quantity: Int64
    let reason: String?
    let unitPriceMinor: Int64?
    let amountMinor: Int64?

    enum CodingKeys: String, CodingKey {
        case sku
        case quantity
        case reason
        case unitPriceMinor = "unit_price_minor"
        case amountMinor = "amount_minor"
    }
}

struct RetailerClaimEvidence: Codable, Hashable {
    let evidenceType: String
    let uri: String
    let mimeType: String?

    enum CodingKeys: String, CodingKey {
        case evidenceType = "evidence_type"
        case uri
        case mimeType = "mime_type"
    }
}

struct RetailerClaim: Codable, Identifiable, Hashable {
    var id: String { claimId }
    let claimId: String
    let orderId: String
    let claimType: String
    let status: String
    let description: String?
    let amountMinor: Int64?
    let currency: String?
    let lineItems: [RetailerClaimLine]?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case claimId = "claim_id"
        case orderId = "order_id"
        case claimType = "claim_type"
        case status
        case description
        case amountMinor = "amount_minor"
        case currency
        case lineItems = "line_items"
        case createdAt = "created_at"
    }
}

struct RetailerClaimsListResponse: Codable {
    let claims: [RetailerClaim]
}

struct ClaimEligibility: Codable, Hashable {
    let eligible: Bool
    let endsAt: String?
    let windowHours: Int
    let hoursRemaining: Double
    let policySource: String
    let photoRequiredTypes: [String]?
    let orderStatus: String?
    let reason: String?

    enum CodingKeys: String, CodingKey {
        case eligible
        case endsAt = "ends_at"
        case windowHours = "window_hours"
        case hoursRemaining = "hours_remaining"
        case policySource = "policy_source"
        case photoRequiredTypes = "photo_required_types"
        case orderStatus = "order_status"
        case reason
    }
}

struct FileClaimEvidenceBody: Encodable {
    let evidenceType: String
    let uri: String
    let mimeType: String

    enum CodingKeys: String, CodingKey {
        case evidenceType = "evidence_type"
        case uri
        case mimeType = "mime_type"
    }
}

struct FileClaimLineBody: Encodable {
    let sku: String
    let quantity: Int64
    let reason: String
}

struct FileClaimRequestBody: Encodable {
    let claimType: String
    let description: String
    let lineItems: [FileClaimLineBody]
    let evidences: [FileClaimEvidenceBody]

    enum CodingKeys: String, CodingKey {
        case claimType = "claim_type"
        case description
        case lineItems = "line_items"
        case evidences
    }
}

import Foundation

struct PulseEvent: Decodable, Identifiable {
    let id: String
    let kind: String
    let title: String
    let description: String?
    let occurredAt: String
    let deepLink: String?
    let orderId: String?
    let manifestId: String?

    enum CodingKeys: String, CodingKey {
        case id, kind, title, description
        case occurredAt = "occurred_at"
        case deepLink = "deep_link"
        case orderId = "order_id"
        case manifestId = "manifest_id"
    }
}

struct PulseResponse: Decodable {
    let events: [PulseEvent]
    let fetchedAt: String
    let unreadCount: Int?

    enum CodingKeys: String, CodingKey {
        case events
        case fetchedAt = "fetched_at"
        case unreadCount = "unread_count"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        events = try c.decodeIfPresent([PulseEvent].self, forKey: .events) ?? []
        fetchedAt = try c.decodeIfPresent(String.self, forKey: .fetchedAt) ?? ""
        unreadCount = try c.decodeIfPresent(Int.self, forKey: .unreadCount)
    }
}

struct StatusExplain: Decodable, Equatable {
    let code: String
    let title: String
    let summary: String
    let nextSteps: [String]?
    let deepLink: String?
    let recoverable: Bool

    enum CodingKeys: String, CodingKey {
        case code, title, summary, recoverable
        case nextSteps = "next_steps"
        case deepLink = "deep_link"
    }
}

struct HandoffCardMetadata: Decodable, Equatable {
    let kind: String
    let title: String
    let subtitle: String?
    let primaryCta: String?
    let primaryLink: String?
    let entityType: String?
    let entityId: String?
    let fields: [String: String]?

    enum CodingKeys: String, CodingKey {
        case kind, title, subtitle, fields
        case primaryCta = "primary_cta"
        case primaryLink = "primary_link"
        case entityType = "entity_type"
        case entityId = "entity_id"
    }
}

struct ApiErrorBody: Decodable {
    let error: String?
    let detail: String?
    let message: String?
    let explain: StatusExplain?
}

enum ApiExplainParser {
    static func parse(from data: Data) -> (message: String, explain: StatusExplain?)? {
        guard let body = try? JSONDecoder().decode(ApiErrorBody.self, from: data) else { return nil }
        let message = body.message ?? body.detail ?? body.error ?? body.explain?.summary
        guard let message, !message.isEmpty else { return nil }
        return (message, body.explain)
    }
}

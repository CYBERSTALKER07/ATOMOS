//
//  DriverNotification.swift
//  driverappios
//

import Foundation

struct DriverNotification: Identifiable, Decodable {
    let id: String
    let type: String
    let title: String
    let body: String
    let payload: String
    let channel: String
    let readAt: String?
    let createdAt: String
    let handoffMetadata: HandoffCardMetadata?

    var isUnread: Bool { readAt == nil }

    enum CodingKeys: String, CodingKey {
        case id, type, title, body, payload, channel
        case readAt = "read_at"
        case createdAt = "created_at"
        case handoffMetadata = "handoff_metadata"
    }
}

struct DriverNotificationsResponse: Decodable {
    let notifications: [DriverNotification]
    let unreadCount: Int
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case notifications
        case unreadCount = "unread_count"
        case hasMore = "has_more"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        notifications = try c.decodeIfPresent([DriverNotification].self, forKey: .notifications) ?? []
        unreadCount = try c.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
        hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
    }
}

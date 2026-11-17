import Foundation
import Observation

struct WarehouseNotificationsPage: Decodable {
    let notifications: [WarehouseNotificationItem]
    let unreadCount: Int
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case notifications
        case unreadCount = "unread_count"
        case hasMore = "has_more"
    }

    init(notifications: [WarehouseNotificationItem], unreadCount: Int, hasMore: Bool = false) {
        self.notifications = notifications
        self.unreadCount = unreadCount
        self.hasMore = hasMore
    }
}

struct WarehouseNotificationItem: Decodable, Identifiable {
    let id: String
    let type: String
    let title: String
    let body: String
    let payload: String
    let channel: String
    let readAt: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id, type, title, body, payload, channel
        case readAt = "read_at"
        case createdAt = "created_at"
    }

    init(
        id: String,
        type: String = "",
        title: String = "",
        body: String = "",
        payload: String = "",
        channel: String = "IN_APP",
        readAt: String? = nil,
        createdAt: String = ""
    ) {
        self.id = id
        self.type = type
        self.title = title
        self.body = body
        self.payload = payload
        self.channel = channel
        self.readAt = readAt
        self.createdAt = createdAt
    }
}

struct MarkNotificationsReadRequest: Encodable {
    let markAll: Bool

    enum CodingKeys: String, CodingKey {
        case markAll = "mark_all"
    }
}

@Observable
@MainActor
final class NotificationInboxViewModel {
    typealias FetchPage = @MainActor () async throws -> WarehouseNotificationsPage
    typealias MarkAllRead = @MainActor () async throws -> Void

    var items: [WarehouseNotificationItem] = []
    var unreadCount = 0
    var loading = true
    var error: String?

    private let fetchPage: FetchPage
    private let markAllReadFn: MarkAllRead

    init(
        fetchPage: FetchPage? = nil,
        markAllRead: MarkAllRead? = nil
    ) {
        self.fetchPage = fetchPage ?? {
            try await APIClient.shared.get(
                "v1/user/notifications",
                query: ["limit": "50", "offset": "0"]
            )
        }
        self.markAllReadFn = markAllRead ?? {
            try await APIClient.shared.postVoid(
                "v1/user/notifications/read",
                body: MarkNotificationsReadRequest(markAll: true)
            )
        }
    }

    func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let page = try await fetchPage()
            items = page.notifications
            unreadCount = page.unreadCount
        } catch let apiError as APIError {
            switch apiError {
            case .httpError(503):
                self.error = "Notifications unavailable on this stack"
            default:
                self.error = "Failed to load notifications"
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func markAllRead() async {
        do {
            try await markAllReadFn()
            PegasusHaptics.success()
            await load()
        } catch {
            self.error = error.localizedDescription
            PegasusHaptics.error()
        }
    }
}

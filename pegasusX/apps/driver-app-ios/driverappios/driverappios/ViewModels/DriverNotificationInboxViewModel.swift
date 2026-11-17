//
//  DriverNotificationInboxViewModel.swift
//  driverappios
//

import Foundation
import Observation

private struct NotificationEmptyResponse: Decodable {}

@Observable
@MainActor
final class DriverNotificationInboxViewModel {
    var items: [DriverNotification] = []
    var unreadCount: Int = 0
    var isLoading = true
    var errorMessage: String?

    private let api: APIClient

    init(api: APIClient = .shared) {
        self.api = api
    }

    func load() async {
        isLoading = true
        errorMessage = nil
        do {
            let pageSize = 100
            var offset = 0
            var merged: [DriverNotification] = []
            var totalUnread = 0
            var hasMore = true
            while hasMore && offset < 2500 {
                let resp: DriverNotificationsResponse = try await api.get(
                    "/v1/user/notifications?limit=\(pageSize)&offset=\(offset)"
                )
                merged.append(contentsOf: resp.notifications)
                totalUnread = resp.unreadCount
                hasMore = resp.hasMore
                offset += pageSize
            }
            items = merged
            unreadCount = totalUnread
        } catch {
            errorMessage = Self.message(for: error)
        }
        isLoading = false
    }

    private static func message(for error: Error) -> String {
        switch error {
        case let APIError.problemDetail(problem):
            return problem.detail ?? problem.title ?? "Unable to load notifications."
        case APIError.networkError:
            return "You appear to be offline. Check your connection and try again."
        case APIError.forbidden:
            return "You do not have permission to view notifications right now."
        case APIError.unauthorized:
            return "Your session expired. Sign in again and retry."
        default:
            return error.localizedDescription
        }
    }

    func markRead(_ id: String) async {
        struct Payload: Encodable { let notification_ids: [String] }
        let _: NotificationEmptyResponse? = try? await api.post("/v1/user/notifications/read", body: Payload(notification_ids: [id]))
        if let idx = items.firstIndex(where: { $0.id == id }) {
            items[idx] = DriverNotification(
                id: items[idx].id, type: items[idx].type,
                title: items[idx].title, body: items[idx].body,
                payload: items[idx].payload, channel: items[idx].channel,
                readAt: "now", createdAt: items[idx].createdAt,
                handoffMetadata: items[idx].handoffMetadata
            )
            unreadCount = max(0, unreadCount - 1)
        }
    }

    func markAllRead() async {
        struct Payload: Encodable { let mark_all: Bool }
        let _: NotificationEmptyResponse? = try? await api.post("/v1/user/notifications/read", body: Payload(mark_all: true))
        items = items.map {
            DriverNotification(
                id: $0.id, type: $0.type, title: $0.title, body: $0.body,
                payload: $0.payload, channel: $0.channel,
                readAt: $0.readAt ?? "now", createdAt: $0.createdAt,
                handoffMetadata: $0.handoffMetadata
            )
        }
        unreadCount = 0
    }
}

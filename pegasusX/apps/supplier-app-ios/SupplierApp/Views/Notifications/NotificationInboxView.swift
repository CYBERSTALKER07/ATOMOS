import SwiftUI

struct NotificationInboxView: View {
    @State private var items: [SupplierNotificationItem] = []
    @State private var unreadCount = 0
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading notifications…")
            } else if let error {
                SupplierErrorView(message: error) { load() }
            } else if items.isEmpty {
                ContentUnavailableView("No notifications", systemImage: "bell")
            } else {
                List {
                    ForEach(items) { item in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(item.title.isEmpty ? (item.type.isEmpty ? "Notification" : item.type) : item.title)
                                .fontWeight(item.readAt == nil ? .semibold : .regular)
                            if !item.body.isEmpty {
                                Text(item.body)
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                            }
                            if !item.createdAt.isEmpty {
                                Text(item.createdAt)
                                    .font(.caption)
                                    .foregroundStyle(.tertiary)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle(unreadCount > 0 ? "Notifications (\(unreadCount))" : "Notifications")
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                if unreadCount > 0 {
                    Button("Mark all read") {
                        Task { await markAllRead() }
                    }
                }
                Button("Refresh", systemImage: "arrow.clockwise") {
                    load()
                }
            }
        }
        .refreshable { load() }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            defer { loading = false }
            do {
                let page: SupplierNotificationsPage = try await APIClient.shared.get(
                    "v1/user/notifications",
                    query: ["limit": "50", "offset": "0"],
                )
                items = page.notifications
                unreadCount = page.unreadCount
            } catch {
                self.error = error.localizedDescription
            }
        }
    }

    private func markAllRead() async {
        do {
            try await APIClient.shared.postVoid(
                "v1/user/notifications/read",
                body: MarkNotificationsReadRequest(markAll: true),
            )
            load()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private struct SupplierNotificationsPage: Decodable {
    let notifications: [SupplierNotificationItem]
    let unreadCount: Int
    let hasMore: Bool

    enum CodingKeys: String, CodingKey {
        case notifications
        case unreadCount = "unread_count"
        case hasMore = "has_more"
    }
}

struct SupplierNotificationItem: Decodable, Identifiable {
    let id: String
    let type: String
    let title: String
    let body: String
    let payload: String
    let channel: String
    let readAt: String?
    let createdAt: String

    enum CodingKeys: String, CodingKey {
        case id = "notification_id"
        case type, title, body, payload, channel
        case readAt = "read_at"
        case createdAt = "created_at"
    }
}

private struct MarkNotificationsReadRequest: Encodable {
    let markAll: Bool

    enum CodingKeys: String, CodingKey {
        case markAll = "mark_all"
    }
}

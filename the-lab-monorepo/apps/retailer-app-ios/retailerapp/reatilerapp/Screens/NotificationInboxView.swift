import SwiftUI

// MARK: - Model

struct NotificationItem: Identifiable, Decodable {
    let id: String
    let type: String
    let title: String
    let body: String
    let payload: String
    let channel: String
    let readAt: String?
    let createdAt: String

    var isUnread: Bool { readAt == nil }

    enum CodingKeys: String, CodingKey {
        case id, type, title, body, payload, channel
        case readAt = "read_at"
        case createdAt = "created_at"
    }
}

struct NotificationsResponse: Decodable {
    let notifications: [NotificationItem]
    let unreadCount: Int

    enum CodingKeys: String, CodingKey {
        case notifications
        case unreadCount = "unread_count"
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        notifications = try c.decodeIfPresent([NotificationItem].self, forKey: .notifications) ?? []
        unreadCount = try c.decodeIfPresent(Int.self, forKey: .unreadCount) ?? 0
    }
}

// MARK: - ViewModel

@Observable
final class NotificationInboxViewModel {
    var items: [NotificationItem] = []
    var unreadCount: Int = 0
    var isLoading = true

    private let api = APIClient.shared

    func load() async {
        do {
            let resp: NotificationsResponse = try await api.get(path: "/v1/user/notifications?limit=50")
            items = resp.notifications
            unreadCount = resp.unreadCount
        } catch {
            // silent — empty state shows
        }
        isLoading = false
    }

    func markRead(_ id: String) async {
        struct Payload: Encodable { let notification_ids: [String] }
        let _: EmptyOK = (try? await api.post(path: "/v1/user/notifications/read", body: Payload(notification_ids: [id]))) ?? EmptyOK()
        if let idx = items.firstIndex(where: { $0.id == id }) {
            items[idx] = NotificationItem(
                id: items[idx].id, type: items[idx].type,
                title: items[idx].title, body: items[idx].body,
                payload: items[idx].payload, channel: items[idx].channel,
                readAt: "now", createdAt: items[idx].createdAt
            )
            unreadCount = max(0, unreadCount - 1)
        }
    }

    func markAllRead() async {
        struct Payload: Encodable { let mark_all: Bool }
        let _: EmptyOK = (try? await api.post(path: "/v1/user/notifications/read", body: Payload(mark_all: true))) ?? EmptyOK()
        items = items.map {
            NotificationItem(
                id: $0.id, type: $0.type, title: $0.title, body: $0.body,
                payload: $0.payload, channel: $0.channel,
                readAt: $0.readAt ?? "now", createdAt: $0.createdAt
            )
        }
        unreadCount = 0
    }
}

private struct EmptyOK: Decodable {}

// MARK: - View

struct NotificationInboxView: View {
    @State private var vm = NotificationInboxViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Group {
                if vm.isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if vm.items.isEmpty {
                    ContentUnavailableView(
                        "No Notifications",
                        systemImage: "bell.slash",
                        description: Text("You'll be notified about order updates here")
                    )
                } else {
                    List(vm.items) { notif in
                        NotificationRow(notification: notif)
                            .listRowBackground(notif.isUnread ? Color(.systemGray6) : Color.clear)
                            .onTapGesture {
                                if notif.isUnread {
                                    Task { await vm.markRead(notif.id) }
                                }
                            }
                    }
                    .listStyle(.plain)
                }
            }
            .navigationTitle("Notifications")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Done") { dismiss() }
                }
                if vm.unreadCount > 0 {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            Task { await vm.markAllRead() }
                        } label: {
                            Label("Read All", systemImage: "checkmark.circle")
                                .labelStyle(.titleAndIcon)
                                .font(.caption)
                        }
                    }
                }
            }
            .task { await vm.load() }
        }
    }
}

// MARK: - Row

private struct NotificationRow: View {
    let notification: NotificationItem

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: typeIcon)
                .font(.system(size: 18))
                .foregroundStyle(notification.isUnread ? .blue : .secondary)
                .frame(width: 24)

            VStack(alignment: .leading, spacing: 2) {
                HStack {
                    Text(notification.title)
                        .font(.subheadline)
                        .fontWeight(notification.isUnread ? .semibold : .regular)
                        .foregroundStyle(notification.isUnread ? .primary : .secondary)
                        .lineLimit(1)

                    Spacer()

                    Text(timeAgo)
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }

                Text(notification.body)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }

            if notification.isUnread {
                Circle()
                    .fill(.blue)
                    .frame(width: 8, height: 8)
                    .padding(.top, 4)
            }
        }
        .padding(.vertical, 4)
    }

    private var typeIcon: String {
        switch notification.type {
        case "ORDER_DISPATCHED": return "shippingbox"
        case "DRIVER_ARRIVED": return "mappin.circle"
        case "ORDER_STATUS_CHANGED": return "arrow.triangle.2.circlepath"
        case "PAYLOAD_READY_TO_SEAL": return "shippingbox"
        case "PAYLOAD_SEALED": return "checkmark.seal"
        case "GLOBAL_PAYNT_SETTLED": return "creditcard"
        case "GLOBAL_PAYNT_FAILED": return "exclamationmark.triangle"
        default: return "bell"
        }
    }

    private var timeAgo: String {
        guard let date = ISO8601DateFormatter().date(from: notification.createdAt) else { return "" }
        let diff = Date().timeIntervalSince(date)
        let mins = Int(diff / 60)
        if mins < 1 { return "now" }
        if mins < 60 { return "\(mins)m" }
        let hrs = mins / 60
        if hrs < 24 { return "\(hrs)h" }
        return "\(hrs / 24)d"
    }
}

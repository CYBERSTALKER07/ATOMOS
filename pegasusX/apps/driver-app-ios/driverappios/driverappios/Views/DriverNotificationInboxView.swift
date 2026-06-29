import SwiftUI

// MARK: - Model

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

// MARK: - ViewModel

@Observable
final class DriverNotificationInboxViewModel {
    var items: [DriverNotification] = []
    var unreadCount: Int = 0
    var isLoading = true
    var errorMessage: String?

    private let api = APIClient.shared

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
        let _: EmptyResponse? = try? await api.post("/v1/user/notifications/read", body: Payload(notification_ids: [id]))
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
        let _: EmptyResponse? = try? await api.post("/v1/user/notifications/read", body: Payload(mark_all: true))
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

private struct EmptyResponse: Decodable {}

// MARK: - View

struct DriverNotificationInboxView: View {
    @State private var vm = DriverNotificationInboxViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Group {
                if vm.isLoading {
                    DriverLoadingView(
                        title: "Loading notifications",
                        message: "Checking route, payment, and dispatch updates."
                    )
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error = vm.errorMessage, vm.items.isEmpty {
                    DriverErrorView(
                        title: "Couldn't load notifications",
                        message: error,
                        retry: { Task { await vm.load() } }
                    )
                    .padding(.horizontal, LabTheme.s16)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if vm.items.isEmpty {
                    DriverEmptyView(
                        icon: "bell.slash",
                        title: "No notifications yet",
                        message: "Dispatch updates will appear here."
                    )
                    .padding(.horizontal, LabTheme.s16)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else {
                    List(vm.items) { notif in
                        DriverNotifRow(notification: notif)
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

private struct DriverNotifRow: View {
    let notification: DriverNotification

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: typeIcon)
                .font(.system(size: 18))
                .foregroundStyle(notification.isUnread ? .blue : .secondary)
                .frame(width: 24)

            VStack(alignment: .leading, spacing: 2) {
                HStack {
                    Text(notification.title)
                        .font(.system(size: 13, weight: notification.isUnread ? .bold : .medium))
                        .foregroundStyle(notification.isUnread ? LabTheme.fg : LabTheme.fgSecondary)
                        .lineLimit(1)

                    Spacer()

                    if notification.isUnread {
                        DriverStatusBadge(text: "NEW", tint: LabTheme.transit)
                    }

                    Text(timeAgo)
                        .font(.system(size: 10, weight: .medium, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                Text(notification.body)
                    .font(.system(size: 12, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .lineLimit(2)
                if let handoff = notification.handoffMetadata {
                    HandoffInboxCard(metadata: handoff)
                        .padding(.top, 4)
                }
            }
        }
        .padding(.vertical, 4)
    }

    private var typeIcon: String {
        switch notification.type {
        case "ORDER_DISPATCHED": return "shippingbox"
        case "DRIVER_ARRIVED": return "mappin.circle"
        case "ORDER_STATUS_CHANGED": return "arrow.triangle.2.circlepath"
        case "PAYMENT_SETTLED": return "creditcard"
        case "PAYMENT_FAILED": return "exclamationmark.triangle"
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

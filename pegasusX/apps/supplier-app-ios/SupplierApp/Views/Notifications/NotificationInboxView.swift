import SwiftUI

struct NotificationInboxView: View {
    @State private var vm = NotificationInboxViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.items.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.notifications")
            } else if let error = vm.error, vm.items.isEmpty {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.items.isEmpty {
                ContentUnavailableView("mobile_supplier.chrome.no_notifications", systemImage: "bell")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.items) { item in
                        VStack(alignment: .leading, spacing: 4) {
                            Text(item.title.isEmpty ? (item.type.isEmpty ? L10n.format("mobile_supplier.chrome.notification") : item.type) : item.title)
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
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .navigationTitle(vm.unreadCount > 0 ? LocalizedStringKey(L10n.format("mobile_supplier.chrome.notifications_count", "\(vm.unreadCount)")) : LocalizedStringKey("mobile_supplier.chrome.notifications"))
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                if vm.unreadCount > 0 {
                    Button("mobile_supplier.ui.mark_all_read") {
                        PegasusHaptics.light()
                        Task { await vm.markAllRead() }
                    }
                    .frame(minHeight: 44)
                }
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load(silent: true) }
                }
                .frame(minWidth: 44, minHeight: 44)
            }
        }
        .refreshable { await vm.load(silent: true) }
        .task { await vm.load() }
    }
}

struct SupplierNotificationItem: Decodable, Identifiable, Sendable {
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

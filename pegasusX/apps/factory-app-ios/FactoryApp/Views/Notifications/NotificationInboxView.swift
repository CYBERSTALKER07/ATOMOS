import SwiftUI

struct NotificationInboxView: View {
    var showsCloseButton: Bool = false
    @Environment(\.dismiss) private var dismiss
    @State private var vm = NotificationsViewModel()

    var body: some View {
        Group {
            if vm.loading {
                ProgressView()
                    .frame(maxWidth: .infinity, minHeight: 200)
            } else if let error = vm.error {
                ContentUnavailableView {
                    Label("retailer_desktop.residual.text.notifications_unavailable", systemImage: "bell.slash")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.selection()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            } else if vm.items.isEmpty {
                ContentUnavailableView("No notifications", systemImage: "bell")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.items) { item in
                        let titleText = item.title.isEmpty ? (item.type.isEmpty ? "Notification" : item.type) : item.title
                        VStack(alignment: .leading, spacing: 4) {
                            Text(titleText)
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
                        .frame(minHeight: 44, alignment: .leading)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(item.readAt == nil ? "Unread notification: " : "")\(titleText). \(item.body)")
                    }
                }
            }
        }
        .navigationTitle(vm.unreadCount > 0 ? "Notifications (\(vm.unreadCount))" : "Notifications")
        .toolbar {
            if showsCloseButton {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close") {
                        PegasusHaptics.selection()
                        dismiss()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            }
            ToolbarItemGroup(placement: .topBarTrailing) {
                if vm.unreadCount > 0 {
                    Button("mobile_factory.ui.mark_all_read") {
                        Task { await vm.markAllRead() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .accessibilityLabel("Mark all notifications as read")
                }
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh notifications")
            }
        }
        .refreshable { await vm.load() }
        .task { await vm.load() }
    }
}

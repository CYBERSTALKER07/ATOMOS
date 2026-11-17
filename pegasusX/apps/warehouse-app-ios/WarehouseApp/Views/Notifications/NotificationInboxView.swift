import SwiftUI

struct NotificationInboxView: View {
    @State private var vm = NotificationInboxViewModel()

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
                        PegasusHaptics.light()
                        Task { await vm.load() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if vm.items.isEmpty {
                ContentUnavailableView("No notifications", systemImage: "bell")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(vm.items) { item in
                        let displayTitle = item.title.isEmpty ? (item.type.isEmpty ? "Notification" : item.type) : item.title
                        VStack(alignment: .leading, spacing: 4) {
                            Text(displayTitle)
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
                        .accessibilityLabel("\(item.readAt == nil ? "Unread, " : "")\(displayTitle)\(item.body.isEmpty ? "" : ", \(item.body)")")
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle(vm.unreadCount > 0 ? "Notifications (\(vm.unreadCount))" : "Notifications")
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                if vm.unreadCount > 0 {
                    Button("mobile_warehouse.ui.mark_all_read") {
                        PegasusHaptics.light()
                        Task { await vm.markAllRead() }
                    }
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
                    .accessibilityLabel("Mark all notifications read")
                }
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh notifications")
            }
        }
        .refreshable { await vm.load() }
        .task { await vm.load() }
    }
}

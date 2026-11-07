import SwiftUI

struct NotificationsSheet: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.notifications.isEmpty {
                    PayloadStateView(
                        variant: .notifications,
                        title: "NO_NOTIFICATIONS",
                        message: "New events appear here in real time.",
                        compact: false
                    )
                } else {
                    ScrollView {
                        VStack(spacing: 0) {
                            ForEach(viewModel.notifications) { n in
                                NotificationRow(item: n) {
                                    if n.isUnread { viewModel.markNotificationRead(n.notificationId) }
                                } onHandoffAction: { link in
                                    Task { await viewModel.handleHandoffLink(link) }
                                }
                            }
                        }
                        .padding()
                    }
                }
            }
            .navigationTitle("portal.nav.notifications")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close") { viewModel.toggleNotificationsPanel() }
                }
                if viewModel.unreadCount > 0 {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button("mobile_payload.ui.mark_all_read") { viewModel.markAllNotificationsRead() }
                    }
                }
            }
        }
    }
}

struct NotificationRow: View {
    let item: NotificationItem
    let onTap: () -> Void
    var onHandoffAction: ((String) -> Void)? = nil

    var body: some View {
        Button(action: onTap) {
            HStack(alignment: .top, spacing: 12) {
                Circle()
                    .fill(item.isUnread ? Color.accentColor : Color.clear)
                    .frame(width: 8, height: 8)
                    .padding(.top, 6)
                VStack(alignment: .leading, spacing: 4) {
                    Text(item.title.isEmpty ? item.type : item.title)
                        .font(.headline)
                    if !item.body.isEmpty {
                        Text(item.body).font(.subheadline).foregroundStyle(.secondary)
                    }
                    if let handoff = item.handoffMetadata {
                        HandoffInboxCard(metadata: handoff, onAction: onHandoffAction)
                    }
                    if !item.createdAt.isEmpty {
                        Text(item.createdAt).font(.caption2).foregroundStyle(.tertiary)
                    }
                }
                Spacer()
            }
            .padding(.vertical, 4)
        }
        .buttonStyle(.tactical)
    }
}

//
//  DriverNotificationInboxView.swift
//  driverappios
//

import SwiftUI

struct DriverNotificationInboxView: View {
    var onHandoffAction: ((String) -> Void)? = nil
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
                        DriverNotifRow(
                            notification: notif,
                            onHandoffAction: { link in
                                dismiss()
                                onHandoffAction?(link)
                            }
                        )
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
            .scrollDismissesKeyboard(.interactively)
            .navigationTitle("portal.nav.notifications")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("warehouse_portal.kpi_stat_card.text.done") { dismiss() }
                        .frame(minHeight: 44)
                }
                if vm.unreadCount > 0 {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            Task { await vm.markAllRead() }
                        } label: {
                            Label("mobile_driver.ui.read_all", systemImage: "checkmark.circle")
                                .labelStyle(.titleAndIcon)
                                .font(.caption)
                                .frame(minHeight: 44)
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
    var onHandoffAction: ((String) -> Void)? = nil

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Image(systemName: typeIcon)
                .font(.system(.title3))
                .foregroundStyle(notification.isUnread ? .blue : .secondary)
                .frame(width: 24)

            VStack(alignment: .leading, spacing: 2) {
                HStack {
                    Text(notification.title)
                        .font(.system(.footnote, weight: notification.isUnread ? .bold : .medium))
                        .foregroundStyle(notification.isUnread ? LabTheme.fg : LabTheme.fgSecondary)
                        .lineLimit(1)

                    Spacer()

                    if notification.isUnread {
                        DriverStatusBadge(text: "NEW", tint: LabTheme.transit)
                    }

                    Text(timeAgo)
                        .font(.system(.caption2, design: .monospaced, weight: .medium))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                Text(notification.body)
                    .font(.system(.caption, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                    .lineLimit(2)
                if let handoff = notification.handoffMetadata {
                    HandoffInboxCard(metadata: handoff, onAction: onHandoffAction)
                        .padding(.top, 4)
                }
            }
        }
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(notification.title), \(notification.body)")
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

    private static let isoFormatter = ISO8601DateFormatter()

    private var timeAgo: String {
        guard let date = Self.isoFormatter.date(from: notification.createdAt) else { return "" }
        let diff = Date().timeIntervalSince(date)
        let mins = Int(diff / 60)
        if mins < 1 { return "now" }
        if mins < 60 { return "\(mins)m" }
        let hrs = mins / 60
        if hrs < 24 { return "\(hrs)h" }
        return "\(hrs / 24)d"
    }
}

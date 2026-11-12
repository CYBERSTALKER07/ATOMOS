import SwiftUI

// MARK: - Order status history (GET /v1/order/{id}/timeline)

struct OrderStatusHistorySection: View {
    let orderId: String

    @State private var vm = OrderDetailViewModel()

    var body: some View {
        LabCardWithHeader(title: "Status history", subtitle: nil, icon: "clock.arrow.circlepath") {
            Group {
                if vm.isTimelineLoading {
                    Text("warehouse_portal.bins.text.loading")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                } else if vm.timelineItems.isEmpty {
                    Text("supplier_portal.order_timeline_panel.text.no_status_history_yet")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                        ForEach(vm.timelineItems) { entry in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(statusLabel(entry))
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Text(subtitle(for: entry))
                                    .font(.system(.caption, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(statusLabel(entry)), \(subtitle(for: entry))")

                            if entry.id != vm.timelineItems.last?.id {
                                Rectangle()
                                    .fill(AppTheme.separator.opacity(0.3))
                                    .frame(height: AppTheme.separatorHeight)
                            }
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .task(id: orderId) {
            await vm.loadTimeline(orderId: orderId)
        }
    }

    private func statusLabel(_ entry: OrderTimelineEntry) -> String {
        if let previous = entry.previousStatus, !previous.isEmpty {
            return "\(previous) → \(entry.newStatus)"
        }
        return entry.newStatus
    }

    private func subtitle(for entry: OrderTimelineEntry) -> String {
        var parts: [String] = [formatWhen(entry.createdAt)]
        if let reason = entry.reason, !reason.isEmpty {
            parts.append(reason)
        }
        if entry.eventKind == "DELAY" {
            parts.append("Delayed")
        }
        return parts.joined(separator: " · ")
    }

    private static let isoFractionalFormatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f
    }()
    private static let isoStandardFormatter = ISO8601DateFormatter()

    private func formatWhen(_ iso: String) -> String {
        if let date = Self.isoFractionalFormatter.date(from: iso) ?? Self.isoStandardFormatter.date(from: iso) {
            return date.formatted(date: .abbreviated, time: .shortened)
        }
        return iso
    }
}

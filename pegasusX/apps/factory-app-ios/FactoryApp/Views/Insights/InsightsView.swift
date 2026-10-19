import SwiftUI

struct InsightsView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var insights: [Insight] = []
    @State private var loading = true
    @State private var error: String?

    var body: some View {
        NavigationStack {
            Group {
                if loading && insights.isEmpty {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { load() }
                    }
                } else if insights.isEmpty {
                    ContentUnavailableView("No Insights", systemImage: "chart.bar.xaxis", description: Text("warehouse_portal.residual.text.no_replenishment_insights"))
                } else {
                    ResponsiveGridContentWrapper {
                        ForEach(Array(insights.enumerated()), id: \.element.id) { index, insight in
                            InsightRow(insight: insight)
                                .staggeredAppear(index: index)
                        }
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("mobile_factory.ui.replenishment_insights")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
            .task { load() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { event in
                        guard event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") || event.type.hasPrefix("FACTORY_SUPPLY_") else { return }
                        load(silent: true)
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent {
            loading = true
        }
        error = nil
        Task {
            do {
                insights = try await FactoryService.insights().insights
            } catch {
                self.error = error.localizedDescription
            }
            if !silent {
                loading = false
            }
        }
    }
}

// MARK: - Insight Row
private struct InsightRow: View {
    let insight: Insight

    private var urgencyColor: Color {
        switch insight.urgency.uppercased() {
        case "CRITICAL": .red
        case "HIGH": .orange
        case "MEDIUM": .secondary
        default: .green
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(insight.productName.isEmpty ? String(insight.productId.prefix(8)) : insight.productName)
                        .font(.subheadline.bold())
                    Text(insight.warehouseName.isEmpty ? String(insight.warehouseId.prefix(8)) : insight.warehouseName)
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                Text(insight.urgency)
                    .font(.caption2.bold())
                    .foregroundStyle(urgencyColor)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(urgencyColor.opacity(0.12))
                    .clipShape(Capsule())
            }

            HStack {
                MetricPill(label: "Stock", value: "\(insight.currentStock)")
                MetricPill(label: "Vel/day", value: String(format: "%.1f", insight.avgDailyVelocity))
                MetricPill(label: "Days", value: "\(insight.daysUntilStockout)")
                MetricPill(label: "Reorder", value: "\(insight.reorderQuantity)")
            }

            if let why = demandWhyText(insight) {
                Text(why)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(.vertical, 4)
    }

    private func demandWhyText(_ insight: Insight) -> String? {
        if let breakdown = insight.demandBreakdown, !breakdown.isEmpty {
            if let blocked = string(from: breakdown["blocked_reason"]), !blocked.isEmpty {
                return blocked == "insufficient_history"
                    ? "Insufficient history — forecast blocked"
                    : blocked.replacingOccurrences(of: "_", with: " ")
            }
            var parts: [String] = []
            if let burn = number(from: breakdown["burn_rate_7d"]) ?? number(from: breakdown["burn_rate"]) {
                parts.append(String(format: "Burn %.1f/d", burn))
            }
            if let cover = number(from: breakdown["days_cover"]) {
                parts.append(String(format: "%.1fd cover", cover))
            }
            if let confidence = number(from: breakdown["confidence"]) {
                parts.append("\(Int(confidence * 100))% conf")
            }
            if breakdown["mei_network"] != nil {
                parts.append("MEIO network transfer")
            }
            if !parts.isEmpty { return parts.joined(separator: " · ") }
        }
        if let code = insight.reasonCode, !code.isEmpty {
            return code.replacingOccurrences(of: "_", with: " ")
        }
        return nil
    }

    private func number(from codable: AnyCodable?) -> Double? {
        guard let codable else { return nil }
        if let d = codable.value as? Double { return d }
        if let i = codable.value as? Int { return Double(i) }
        if let s = codable.value as? String, let d = Double(s) { return d }
        return nil
    }

    private func string(from codable: AnyCodable?) -> String? {
        guard let codable else { return nil }
        if let s = codable.value as? String, !s.isEmpty { return s }
        return nil
    }
}

private struct MetricPill: View {
    let label: String
    let value: String

    var body: some View {
        VStack(spacing: 2) {
            Text(value)
                .font(.caption.bold())
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 6)
        .background(.quaternary)
        .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM))
    }
}

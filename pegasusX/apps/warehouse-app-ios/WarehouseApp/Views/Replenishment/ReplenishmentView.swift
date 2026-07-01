import SwiftUI

struct ReplenishmentView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var insights: [ReplenishmentInsight] = []
    @State private var loading = true
    @State private var error: String?
    @State private var actingId: String?
    @State private var statusMessage: String?

    var body: some View {
        Group {
            if loading && insights.isEmpty {
                WarehouseLoadingView(
                    title: "Loading replenishment",
                    message: "Fetching stock insights and reorder recommendations."
                )
            } else if let error, insights.isEmpty {
                WarehouseErrorView(message: error) { load() }
            } else if insights.isEmpty {
                WarehouseEmptyView(
                    title: "No insights",
                    message: "No replenishment insights for this warehouse."
                )
            } else {
                List(insights) { insight in
                    VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                        HStack(spacing: LabTheme.spacingXS) {
                            Text(insight.productName)
                                .font(.headline)
                            if insight.reasonCode == "PREDICTIVE_PUSH" {
                                Text("AI PUSH")
                                    .font(.system(size: 10, weight: .bold))
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(Color.primary)
                                    .foregroundColor(Color(UIColor.systemBackground))
                                    .clipShape(Capsule())
                            }
                        }
                        HStack(spacing: LabTheme.spacingSM) {
                            WarehouseStatusBadge(text: insight.urgency, tint: urgencyTint(insight.urgency))
                            WarehouseStatusBadge(text: insight.status)
                        }
                        Text("Stock \(insight.currentStock) · Reorder \(insight.reorderQuantity)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        Text("Days until stockout: \(insight.daysUntilStockout)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if let why = demandWhyText(insight) {
                            Text(why)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if insight.status.uppercased() == "OPEN" {
                            HStack {
                                Button("Approve") {
                                    runAction(insightId: insight.id, action: "approve")
                                }
                                .disabled(actingId == insight.id)
                                Button("Dismiss", role: .destructive) {
                                    runAction(insightId: insight.id, action: "dismiss")
                                }
                                .disabled(actingId == insight.id)
                            }
                            .buttonStyle(.bordered)
                        }
                    }
                    .padding(.vertical, LabTheme.spacingXS)
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Replenishment")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .overlay(alignment: .bottom) {
            if let statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .padding(8)
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM))
                    .padding()
                    .onAppear {
                        DispatchQueue.main.asyncAfter(deadline: .now() + 2.5) {
                            self.statusMessage = nil
                        }
                    }
            }
        }
        .task { load() }
        .refreshable { load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            load(silent: silent)
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if actingId != nil {
                actingId = nil
                statusMessage = "Connection restored — verify status before retrying."
            }
        }
    }

    private func urgencyTint(_ urgency: String) -> Color {
        switch urgency.uppercased() {
        case "PROACTIVE": return .primary
        case "CRITICAL": return LabTheme.destructive
        case "URGENT": return LabTheme.warning
        default: return LabTheme.secondaryLabel
        }
    }

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                let response = try await WarehouseService.replenishmentInsights()
                insights = response.rows
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }

    private func runAction(insightId: String, action: String) {
        actingId = insightId
        Task {
            do {
                let response = try await WarehouseService.replenishmentInsightAction(insightId: insightId, action: action)
                if action == "approve", let transferId = response.transferId, !transferId.isEmpty {
                    statusMessage = "Approved — transfer \(String(transferId.prefix(8)))"
                } else {
                    statusMessage = action == "approve" ? "Insight approved" : "Insight dismissed"
                }
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
            actingId = nil
        }
    }

    private func demandWhyText(_ insight: ReplenishmentInsight) -> String? {
        if let breakdown = insight.demandBreakdown, !breakdown.isEmpty {
            var parts: [String] = []
            if let burn = number(from: breakdown["burn_rate_7d"]) ?? number(from: breakdown["burn_rate"]) {
                parts.append(String(format: "Burn %.1f/d", burn))
            }
            if let cover = number(from: breakdown["days_cover"]) {
                parts.append(String(format: "%.1fd cover", cover))
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
}

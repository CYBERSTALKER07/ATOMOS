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
            if loading {
                WarehouseLoadingView(
                    title: "Loading replenishment",
                    message: "Fetching stock insights and reorder recommendations."
                )
            } else if let error {
                WarehouseErrorView(message: error) { load() }
            } else if insights.isEmpty {
                WarehouseEmptyView(
                    title: "No insights",
                    message: "No replenishment insights for this warehouse."
                )
            } else {
                List(insights) { insight in
                    VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
                        Text(insight.productName)
                            .font(.headline)
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
        .refreshable { load() }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if actingId != nil {
                actingId = nil
                statusMessage = "Connection restored — verify status before retrying."
            }
            load()
        }
    }

    private func urgencyTint(_ urgency: String) -> Color {
        switch urgency.uppercased() {
        case "CRITICAL": return LabTheme.destructive
        case "URGENT": return LabTheme.warning
        default: return LabTheme.secondaryLabel
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let response = try await WarehouseService.replenishmentInsights()
                insights = response.rows
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
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
}

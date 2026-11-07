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
                ReplenishmentList(
                    insights: insights,
                    actingId: actingId,
                    onApprove: { id in runAction(insightId: id, action: "approve") },
                    onDismiss: { id in runAction(insightId: id, action: "dismiss") }
                )
            }
        }
        .navigationTitle("portal.nav.replenishment")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
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
}

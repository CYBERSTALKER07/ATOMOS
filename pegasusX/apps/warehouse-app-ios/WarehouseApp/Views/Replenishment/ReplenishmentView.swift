import SwiftUI

struct ReplenishmentView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = ReplenishmentViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.insights.isEmpty {
                WarehouseLoadingView(
                    title: "Loading replenishment",
                    message: "Fetching stock insights and reorder recommendations."
                )
            } else if let error = vm.error, vm.insights.isEmpty {
                WarehouseErrorView(message: error) {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
            } else if vm.insights.isEmpty {
                WarehouseEmptyView(
                    title: "No insights",
                    message: "No replenishment insights for this warehouse."
                )
            } else {
                ReplenishmentList(
                    insights: vm.insights,
                    actingId: vm.actingId,
                    onApprove: { id in
                        PegasusHaptics.medium()
                        Task { await vm.runAction(insightId: id, action: "approve") }
                    },
                    onDismiss: { id in
                        PegasusHaptics.warning()
                        Task { await vm.runAction(insightId: id, action: "dismiss") }
                    }
                )
            }
        }
        .navigationTitle("portal.nav.replenishment")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh replenishment insights")
            }
        }
        .overlay(alignment: .bottom) {
            if let statusMessage = vm.statusMessage {
                Text(statusMessage)
                    .font(.caption)
                    .padding(8)
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusSM))
                    .padding()
                    .task(id: statusMessage) {
                        await vm.autoClearStatusMessage()
                    }
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            vm.handleReconnect()
        }
    }
}

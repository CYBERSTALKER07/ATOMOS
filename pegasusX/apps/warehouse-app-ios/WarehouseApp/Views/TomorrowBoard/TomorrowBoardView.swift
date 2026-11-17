import SwiftUI

struct TomorrowBoardView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = TomorrowBoardViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.preorders.isEmpty && vm.deliverBefore.isEmpty {
                WarehouseLoadingView(
                    title: "Loading tomorrow board",
                    message: "Fetching orders and manifests grouped by delivery date."
                )
            } else if let error = vm.error, vm.preorders.isEmpty && vm.deliverBefore.isEmpty {
                WarehouseErrorView(message: error) {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                }
            } else {
                ResponsiveGridContentWrapper {
                    Section {
                        DatePicker("Date", selection: $vm.date, displayedComponents: .date)
                            .frame(minHeight: 44)
                            .onChange(of: vm.date) { _, _ in
                                PegasusHaptics.selection()
                                Task { await vm.load() }
                            }
                    }
                    if vm.preorders.isEmpty && vm.deliverBefore.isEmpty {
                        Section {
                            Text("warehouse_portal.tomorrow_board.text.no_orders_scheduled_for_this_date")
                                .foregroundStyle(.secondary)
                        }
                    }
                    if !vm.preorders.isEmpty {
                        Section("Pre-orders") {
                            ForEach(vm.preorders) { row in
                                boardRow(row, lane: "Pre-order")
                            }
                        }
                    }
                    if !vm.deliverBefore.isEmpty {
                        Section("Deliver by") {
                            ForEach(vm.deliverBefore) { row in
                                boardRow(row, lane: "Deliver by")
                            }
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .navigationTitle("portal.nav.tomorrow_board")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load() }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh tomorrow board")
            }
        }
        .task { await vm.load() }
        .refreshable { await vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(silent: silent) }
        }
    }

    @ViewBuilder
    private func boardRow(_ row: WarehouseOpsBoardOrder, lane: String) -> some View {
        let detail = row.deliveryExpectation?.targetLabel ?? row.status
        VStack(alignment: .leading, spacing: 4) {
            Text(lane)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(row.orderId)
                .font(.headline)
            Text(detail)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(minHeight: 44, alignment: .leading)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(lane), order \(row.orderId), \(detail)")
    }
}

import SwiftUI

struct SupplyRequestDetailView: View {
    let requestId: String

    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = SupplyRequestDetailViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.request == nil {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.request == nil {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load(requestId: requestId) }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if let request = vm.request {
                ResponsiveGridContentWrapper {
                    if let statusMessage = vm.statusMessage {
                        Section { Text(statusMessage).foregroundStyle(.green) }
                    }
                    Section("Request") {
                        LabKeyValueRow(label: "ID", value: request.requestId)
                        LabKeyValueRow(label: "State", value: request.state)
                        LabKeyValueRow(label: "Priority", value: request.priority)
                        LabKeyValueRow(label: "Factory", value: request.factoryId)
                        LabKeyValueRow(label: "Volume (VU)", value: String(format: "%.1f", request.totalVolumeVu))
                        LabKeyValueRow(label: "Transfer order", value: request.transferOrderId ?? "—")
                        LabKeyValueRow(label: "Created", value: request.createdAt)
                        LabKeyValueRow(label: "QC", value: vm.qcResult.isEmpty ? "—" : vm.qcResult)
                        if !request.notes.isEmpty {
                            LabKeyValueRow(label: "Notes", value: request.notes)
                        }
                    }
                    if request.state.uppercased() == "OPEN" {
                        Section {
                            Button("mobile_warehouse.ui.cancel_request", role: .destructive) {
                                PegasusHaptics.warning()
                                Task { await vm.cancelRequest(requestId: requestId) }
                            }
                            .frame(minHeight: 44)
                            .contentShape(Rectangle())
                            .disabled(vm.busy)
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
            } else {
                ContentUnavailableView(
                    "Not found",
                    systemImage: "tray",
                    description: Text("mobile_warehouse.ui.supply_request_not_found")
                )
            }
        }
        .navigationTitle("mobile_warehouse.ui.supply_request")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load(requestId: requestId) }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh supply request")
            }
        }
        .task(id: requestId) {
            await vm.load(requestId: requestId)
        }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(requestId: requestId, silent: silent) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            vm.handleReconnect()
        }
    }
}

private struct LabKeyValueRow: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(value)
        }
        .frame(minHeight: 44, alignment: .leading)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label), \(value)")
    }
}

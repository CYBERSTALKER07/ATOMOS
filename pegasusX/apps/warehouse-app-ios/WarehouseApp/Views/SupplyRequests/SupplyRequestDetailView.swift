import SwiftUI

struct SupplyRequestDetailView: View {
    let requestId: String

    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var request: WarehouseSupplyRequest?
    @State private var loading = true
    @State private var error: String?
    @State private var busy = false
    @State private var statusMessage: String?

    var body: some View {
        Group {
            if loading && request == nil {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error, request == nil {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") { load() }
                }
            } else if let request {
                ResponsiveGridContentWrapper {
                    if let statusMessage {
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
                        if !request.notes.isEmpty {
                            LabKeyValueRow(label: "Notes", value: request.notes)
                        }
                    }
                    if request.state.uppercased() == "OPEN" {
                        Section {
                            Button("mobile_warehouse.ui.cancel_request", role: .destructive) {
                                cancelRequest()
                            }
                            .disabled(busy)
                        }
                    }
                }
            } else {
                ContentUnavailableView("Not found", systemImage: "tray", description: Text("mobile_warehouse.ui.supply_request_not_found"))
            }
        }
        .navigationTitle("mobile_warehouse.ui.supply_request")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task(id: requestId) { load() }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            load(silent: silent)
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if busy {
                busy = false
                statusMessage = "Connection restored — verify status before retrying."
            }
        }
    }

    private func load(silent: Bool = false) {
        if !silent && request == nil { loading = true }
        error = nil
        Task {
            do {
                request = try await WarehouseService.supplyRequest(id: requestId)
            } catch {
                if !silent { self.error = error.localizedDescription }
            }
            if !silent { loading = false }
        }
    }

    private func cancelRequest() {
        busy = true
        Task {
            do {
                _ = try await WarehouseService.cancelSupplyRequest(id: requestId)
                statusMessage = "Request cancelled"
                load()
            } catch {
                statusMessage = error.localizedDescription
            }
            busy = false
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
    }
}

import SwiftUI

struct SupplyRequestsHubView: View {
    @State private var requests: [WarehouseSupplyRequest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var stateFilter = "ALL"

    private let filters = ["ALL", "OPEN", "CANCELLED"]

    var body: some View {
        Group {
            if loading {
                ProgressView()
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error {
                ContentUnavailableView("Error", systemImage: "exclamationmark.triangle", description: Text(error))
            } else if filtered.isEmpty {
                ContentUnavailableView("No requests", systemImage: "tray", description: Text("No supply requests in this state."))
            } else {
                List(filtered) { request in
                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                        Text(request.requestId)
                            .font(.headline)
                        Text(request.state)
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                        if !request.notes.isEmpty {
                            Text(request.notes)
                                .font(.caption)
                        }
                    }
                }
                .listStyle(.insetGrouped)
            }
        }
        .navigationTitle("Supply Requests")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Picker("State", selection: $stateFilter) {
                    ForEach(filters, id: \.self) { Text($0).tag($0) }
                }
                .pickerStyle(.menu)
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task { load() }
        .refreshable { load() }
    }

    private var filtered: [WarehouseSupplyRequest] {
        guard stateFilter != "ALL" else { return requests }
        return requests.filter { $0.state.uppercased() == stateFilter }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let state = stateFilter == "ALL" ? nil : stateFilter
                requests = try await WarehouseService.supplyRequests(state: state)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

import SwiftUI

struct SupplyRequestsHubView: View {
    @State private var requests: [WarehouseSupplyRequest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var stateFilter = "ALL"
    @State private var showCreate = false

    private let filters = ["ALL", "OPEN", "CANCELLED"]

    var body: some View {
        Group {
            if loading {
                WarehouseLoadingView(
                    title: "Loading supply requests",
                    message: "Fetching factory supply pipeline status."
                )
            } else if let error {
                WarehouseErrorView(message: error) { load() }
            } else if filtered.isEmpty {
                WarehouseEmptyView(
                    title: "No requests",
                    message: "No supply requests in this state."
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(filtered) { request in
                    NavigationLink {
                        SupplyRequestDetailView(requestId: request.requestId)
                    } label: {
                        HStack(alignment: .top) {
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(String(request.requestId.prefix(8)))
                                    .font(.headline)
                                Text(L10n.format("mobile_warehouse.ui.priority_totalvolumevu_vu", "\(request.priority)", "\(Int(request.totalVolumeVu))"))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                if !request.notes.isEmpty {
                                    Text(request.notes)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                        .lineLimit(2)
                                }
                            }
                            Spacer()
                            WarehouseStatusBadge(text: request.state)
                        }
                    }
                }
            }
            }
        }
        .navigationTitle("portal.nav.supply_requests")
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Picker("State", selection: $stateFilter) {
                    ForEach(filters, id: \.self) { Text($0).tag($0) }
                }
                .pickerStyle(.menu)
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_warehouse.ui.new", systemImage: "plus") { showCreate = true }
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .sheet(isPresented: $showCreate) {
            CreateSupplyRequestSheet { form in
                Task {
                    do {
                        _ = try await WarehouseService.createSupplyRequest(form: form)
                        showCreate = false
                        load()
                    } catch {
                        self.error = error.localizedDescription
                    }
                }
            }
        }
        .task { load() }
        .refreshable { load() }
        .onChange(of: stateFilter) { _, _ in load() }
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

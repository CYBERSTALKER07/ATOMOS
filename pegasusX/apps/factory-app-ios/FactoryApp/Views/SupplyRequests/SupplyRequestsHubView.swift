import SwiftUI

struct SupplyRequestsHubView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var requests: [SupplyRequest] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedRequestID: String?

    var body: some View {
        NavigationSplitView {
            Group {
                if loading && requests.isEmpty {
                    FactoryLoadingView(
                        title: "Loading supply requests",
                        message: "Fetching inbound supply pipeline status."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else if requests.isEmpty {
                    FactoryStateView(
                        kind: .empty,
                        headline: "No requests",
                        message: "No supply requests pending from warehouses."
                    )
                } else {
                    List(selection: $selectedRequestID) {
                        Section {
                            FactorySectionHeader(
                                title: "Inbound supply requests",
                                subtitle: "\(requests.count) active requests"
                            )
                            .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                            .listRowBackground(Color.clear)
                        }

                        Section {
                            ForEach(requests) { request in
                                SupplyRequestRow(request: request)
                                    .tag(request.id)
                            }
                        }
                    }
                }
            }
            .navigationTitle("Supply Requests")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("Close", systemImage: "xmark") { dismiss() }
                        .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
        } detail: {
            if let selectedRequestID {
                SupplyRequestDetailView(requestId: selectedRequestID)
            } else {
                ContentUnavailableView("Select a Request", systemImage: "tray.full", description: Text("Choose a supply request to review and transition."))
            }
        }
        .task { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        do {
            requests = try await FactoryService.supplyRequests()
            if selectedRequestID == nil {
                selectedRequestID = requests.first?.id
            }
        } catch {
            self.error = error.localizedDescription
        }
        loading = false
}


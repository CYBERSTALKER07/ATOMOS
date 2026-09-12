import SwiftUI

struct TransferListView: View {
    var initialFilter: String? = nil
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var transfers: [Transfer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedFilter = "ALL"
    @State private var selectedTransferID: String?
    @State private var showCreateTransfer = false

    private let filters = ["ALL", "DRAFT", "APPROVED", "LOADING", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED"]

    private var selectedTransfer: Transfer? {
        transfers.first { $0.id == selectedTransferID }
    }

    var body: some View {
        NavigationSplitView {
            VStack(spacing: 0) {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: LabTheme.spacingSM) {
                        ForEach(filters, id: \.self) { filter in
                            Button {
                                selectedFilter = filter
                            } label: {
                                Text(filter)
                                    .font(.footnote.bold())
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 6)
                                    .background(selectedFilter == filter ? LabTheme.label : Color.clear, in: Capsule())
                                    .foregroundStyle(selectedFilter == filter ? Color(.systemBackground) : LabTheme.label)
                                    .overlay(Capsule().stroke(.quaternary))
                            }
                            .buttonStyle(PressableButtonStyle())
                        }
                    }
                    .padding(.horizontal)
                    .padding(.vertical, LabTheme.spacingSM)
                }

                Divider()

                if loading && transfers.isEmpty {
                    FactoryLoadingView(
                        title: "Loading transfers",
                        message: "Fetching the factory transfer queue and lifecycle states."
                    )
                } else if let error {
                    FactoryErrorView(message: error) {
                        Task { await load() }
                    }
                } else if transfers.isEmpty {
                    FactoryStateView(
                        kind: selectedFilter == "ALL" ? .empty : .noResults,
                        headline: selectedFilter == "ALL" ? "No transfers" : "No \(selectedFilter) transfers",
                        message: selectedFilter == "ALL"
                            ? "There are no transfers in the factory queue right now."
                            : "Adjust the filter or wait for the next queue refresh.",
                        actionTitle: selectedFilter == "ALL" ? nil : "Show All",
                        action: selectedFilter == "ALL" ? nil : { selectedFilter = "ALL" }
                    )
                } else {
                    List(selection: $selectedTransferID) {
                        Section {
                            TransferListSummary(count: transfers.count, selectedFilter: selectedFilter)
                                .listRowInsets(EdgeInsets(top: 8, leading: 0, bottom: 8, trailing: 0))
                                .listRowBackground(Color.clear)
                        }

                        Section {
                            ForEach(transfers) { transfer in
                                TransferRow(transfer: transfer)
                                    .tag(transfer.id)
                            }
                        }
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("portal.nav.transfers")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("mobile_factory.ui.create", systemImage: "plus") {
                        showCreateTransfer = true
                    }
                    .labelStyle(.iconOnly)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
            .sheet(isPresented: $showCreateTransfer) {
                CreateTransferView { transferId in
                    selectedTransferID = transferId
                    Task { await load() }
                }
            }
        } detail: {
            if let transfer = selectedTransfer {
                TransferDetailView(transferId: transfer.id)
            } else {
                ContentUnavailableView(
                    "Select a Transfer",
                    systemImage: "arrow.left.arrow.right",
                    description: Text("mobile_factory.ui.choose_a_transfer_from_the_list")
                )
            }
        }
        .task(id: initialFilter) {
            if let initialFilter, !initialFilter.isEmpty {
                selectedFilter = initialFilter
            }
        }
        .task(id: selectedFilter) { await load() }
        .onAppear {
            realtimeClient.connect(
                onStateChange: { _ in },
                onEvent: { event in
                    guard event.type.hasPrefix("TRANSFER_") || event.type.hasPrefix("MANIFEST_") || event.type.hasPrefix("WAREHOUSE_TRANSFER_") else { return }
                    Task { await load(silent: true) }
                }
            )
        }
        .onDisappear {
            realtimeClient.disconnect()
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent {
            loading = true
        }
        error = nil

        do {
            let state = selectedFilter == "ALL" ? nil : selectedFilter
            let response = try await FactoryService.transfers(state: state)
            transfers = response.transfers

            if let selectedTransferID, transfers.contains(where: { $0.id == selectedTransferID }) {
                self.selectedTransferID = selectedTransferID
            } else {
                selectedTransferID = transfers.first?.id
            }
        } catch {
            self.error = error.localizedDescription
        }

        if !silent {
            loading = false
        }
    }
}


import SwiftUI

struct TransferListView: View {
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var transfers: [Transfer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedFilter = "ALL"
    @State private var selectedTransferID: String?

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

                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") {
                            Task { await load() }
                        }
                    }
                } else if transfers.isEmpty {
                    ContentUnavailableView(
                        "No Transfers",
                        systemImage: "arrow.left.arrow.right",
                        description: Text(
                            selectedFilter == "ALL"
                                ? "There are no transfers available right now."
                                : "There are no \(selectedFilter) transfers in the current queue."
                        )
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
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Transfers")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }
            }
        } detail: {
            if let transfer = selectedTransfer {
                TransferDetailView(transferId: transfer.id)
            } else {
                ContentUnavailableView(
                    "Select a Transfer",
                    systemImage: "arrow.left.arrow.right",
                    description: Text("Choose a transfer from the list.")
                )
            }
        }
        .task(id: selectedFilter) { await load() }
        .onAppear {
            realtimeClient.connect(
                onStateChange: { _ in },
                onEvent: { event in
                    guard let eventType = event.eventType else { return }
                    guard eventType == .transferUpdate || eventType == .manifestUpdate else { return }
                    Task { await load() }
                }
            )
        }
        .onDisappear {
            realtimeClient.disconnect()
        }
    }

    @MainActor
    private func load() async {
        loading = true
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

        loading = false
    }
}

private struct TransferListSummary: View {
    let count: Int
    let selectedFilter: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text("\(count) transfers in view")
                .font(.headline)
            Text(selectedFilter == "ALL" ? "Showing every transfer state across the factory queue." : "Filtered to \(selectedFilter) transfers.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct TransferRow: View {
    let transfer: Transfer

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack(alignment: .top, spacing: LabTheme.spacingMD) {
                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                    Text(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)
                        .font(.subheadline.bold())
                    Text("Transfer \(transfer.id.prefix(8))")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                Spacer()
                VStack(alignment: .trailing, spacing: LabTheme.spacingXS) {
                    TransferRowTag(text: transfer.state)
                    TransferRowTag(text: transfer.priority.isEmpty ? "STANDARD" : transfer.priority, emphasized: false)
                }
            }

            HStack(spacing: LabTheme.spacingSM) {
                TransferRowMetric(label: "Items", value: "\(transfer.totalItems)")
                TransferRowMetric(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
            }
        }
        .padding(.vertical, LabTheme.spacingXS)
    }
}

private struct TransferRowMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.subheadline.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingSM)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct TransferRowTag: View {
    let text: String
    var emphasized = true

    var body: some View {
        Text(text)
            .font(.footnote.bold())
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .background(emphasized ? LabTheme.fill : LabTheme.tertiaryBackground, in: Capsule())
    }
}

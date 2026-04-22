import SwiftUI

struct TransferListView: View {
    @State private var transfers: [Transfer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedFilter = "ALL"
    @State private var selectedTransfer: Transfer?

    private let filters = ["ALL", "DRAFT", "APPROVED", "LOADING", "DISPATCHED", "IN_TRANSIT", "ARRIVED", "RECEIVED", "CANCELLED"]

    var body: some View {
        NavigationSplitView {
            VStack(spacing: 0) {
                // Filter chips
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: LabTheme.spacingSM) {
                        ForEach(filters, id: \.self) { filter in
                            Button {
                                selectedFilter = filter
                            } label: {
                                Text(filter)
                                    .font(.caption.bold())
                                    .padding(.horizontal, 12)
                                    .padding(.vertical, 6)
                                    .background(selectedFilter == filter ? Color.primary : Color.clear)
                                    .foregroundStyle(selectedFilter == filter ? Color(uiColor: .systemBackground) : .primary)
                                    .clipShape(Capsule())
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
                        Button("Retry") { load() }
                    }
                } else if transfers.isEmpty {
                    ContentUnavailableView("No Transfers", systemImage: "arrow.left.arrow.right", description: Text("No transfers match filter"))
                } else {
                    List(transfers, selection: Binding(
                        get: { selectedTransfer?.id },
                        set: { id in selectedTransfer = transfers.first { $0.id == id } }
                    )) { transfer in
                        TransferRow(transfer: transfer)
                            .tag(transfer.id)
                    }
                    .listStyle(.plain)
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Transfers")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
        } detail: {
            if let transfer = selectedTransfer {
                TransferDetailView(transferId: transfer.id)
            } else {
                ContentUnavailableView("Select a Transfer", systemImage: "arrow.left.arrow.right", description: Text("Choose a transfer from the list"))
            }
        }
        .task { load() }
        .onChange(of: selectedFilter) { _, _ in load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let state = selectedFilter == "ALL" ? nil : selectedFilter
                let resp = try await FactoryService.transfers(state: state)
                transfers = resp.transfers
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }
}

// MARK: - Transfer Row
private struct TransferRow: View {
    let transfer: Transfer

    var body: some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                Text(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)
                    .font(.subheadline.bold())
                Text("\(transfer.totalItems) items · \(transfer.priority)")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Text(transfer.state)
                .font(.caption2.bold())
                .padding(.horizontal, 8)
                .padding(.vertical, 3)
                .background(.quaternary)
                .clipShape(Capsule())
        }
        .padding(.vertical, 4)
    }
}

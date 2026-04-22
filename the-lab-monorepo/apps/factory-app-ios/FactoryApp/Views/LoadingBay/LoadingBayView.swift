import SwiftUI

struct LoadingBayView: View {
    @State private var transfers: [Transfer] = []
    @State private var loading = true
    @State private var error: String?
    @State private var dispatching = false

    private var approved: [Transfer] { transfers.filter { $0.state == "APPROVED" } }
    private var loadingState: [Transfer] { transfers.filter { $0.state == "LOADING" } }
    private var dispatched: [Transfer] { transfers.filter { $0.state == "DISPATCHED" } }

    var body: some View {
        NavigationStack {
            Group {
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
                    ContentUnavailableView("No Transfers", systemImage: "shippingbox", description: Text("No transfers in the loading bay"))
                } else {
                    // iPad: horizontal 3-column layout
                    HStack(alignment: .top, spacing: 0) {
                        BayColumn(title: "Ready for Loading", count: approved.count, transfers: approved)
                        Divider()
                        BayColumn(title: "Now Loading", count: loadingState.count, transfers: loadingState)
                        Divider()
                        BayColumn(title: "Dispatched", count: dispatched.count, transfers: dispatched)
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Loading Bay")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button { load() } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
                if !loadingState.isEmpty {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            batchDispatch()
                        } label: {
                            Label(dispatching ? "Dispatching…" : "Batch Dispatch", systemImage: "truck.box")
                        }
                        .disabled(dispatching)
                    }
                }
            }
            .task { load() }
        }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                let resp = try await FactoryService.loadingBayTransfers()
                transfers = resp.transfers
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func batchDispatch() {
        dispatching = true
        Task {
            do {
                let ids = loadingState.map(\.id)
                _ = try await FactoryService.dispatch(transferIds: ids)
                load()
            } catch {
                self.error = error.localizedDescription
            }
            dispatching = false
        }
    }
}

// MARK: - Bay Column
private struct BayColumn: View {
    let title: String
    let count: Int
    let transfers: [Transfer]

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                Text(title)
                    .font(.headline)
                Spacer()
                Text("\(count)")
                    .font(.caption.bold())
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(.quaternary)
                    .clipShape(Capsule())
            }
            .padding()

            Divider()

            ScrollView {
                LazyVStack(spacing: LabTheme.spacingSM) {
                    ForEach(Array(transfers.enumerated()), id: \.element.id) { index, transfer in
                        BayTransferCard(transfer: transfer)
                            .staggeredAppear(index: index)
                    }
                }
                .padding()
            }
        }
        .frame(maxWidth: .infinity)
    }
}

// MARK: - Bay Transfer Card
private struct BayTransferCard: View {
    let transfer: Transfer

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)
                .font(.subheadline.bold())
            HStack {
                Text("\(transfer.totalItems) items")
                    .font(.caption)
                    .foregroundStyle(.secondary)
                Spacer()
                Text(transfer.priority)
                    .font(.caption2.bold())
                    .padding(.horizontal, 6)
                    .padding(.vertical, 2)
                    .background(.quaternary)
                    .clipShape(Capsule())
            }
        }
        .labCard()
    }
}

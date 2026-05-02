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
                        Button("Retry") {
                            Task { await load() }
                        }
                    }
                } else if transfers.isEmpty {
                    ContentUnavailableView(
                        "No Transfers",
                        systemImage: "shippingbox",
                        description: Text("No transfers are active in the loading bay right now.")
                    )
                } else {
                    ScrollView {
                        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                            LoadingBayOverviewCard(
                                readyCount: approved.count,
                                loadingCount: loadingState.count,
                                dispatchedCount: dispatched.count
                            )

                            BaySection(
                                title: "Ready for Loading",
                                count: approved.count,
                                transfers: approved,
                                emptyMessage: "No approved transfers are waiting at the bay."
                            )
                            BaySection(
                                title: "Now Loading",
                                count: loadingState.count,
                                transfers: loadingState,
                                emptyMessage: "Nothing is actively loading right now."
                            )
                            BaySection(
                                title: "Dispatched",
                                count: dispatched.count,
                                transfers: dispatched,
                                emptyMessage: "No transfers have been dispatched in the current view."
                            )
                        }
                        .padding()
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Loading Bay")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") {
                        Task { await load() }
                    }
                    .labelStyle(.iconOnly)
                }

                if !loadingState.isEmpty {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button(dispatching ? "Dispatching" : "Batch Dispatch", systemImage: "truck.box") {
                            Task { await batchDispatch() }
                        }
                        .disabled(dispatching)
                    }
                }
            }
            .task { await load() }
        }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil

        do {
            let response = try await FactoryService.loadingBayTransfers()
            transfers = response.transfers
        } catch {
            self.error = error.localizedDescription
        }

        loading = false
    }

    @MainActor
    private func batchDispatch() async {
        dispatching = true

        do {
            let ids = loadingState.map(\.id)
            _ = try await FactoryService.dispatch(transferIds: ids)
            await load()
        } catch {
            self.error = error.localizedDescription
        }

        dispatching = false
    }
}

private struct BaySection: View {
    let title: String
    let count: Int
    let transfers: [Transfer]
    let emptyMessage: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            HStack {
                Text(title)
                    .font(.headline)
                Spacer()
                Text("\(count)")
                    .font(.footnote.bold())
                    .padding(.horizontal, LabTheme.spacingSM)
                    .padding(.vertical, LabTheme.spacingXS)
                    .background(.quaternary, in: Capsule())
            }

            if transfers.isEmpty {
                Text(emptyMessage)
                    .font(.body)
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(LabTheme.spacingLG)
                    .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            } else {
                LazyVStack(spacing: LabTheme.spacingSM) {
                    ForEach(Array(transfers.enumerated()), id: \.element.id) { index, transfer in
                        BayTransferCard(transfer: transfer)
                            .staggeredAppear(index: index)
                    }
                }
            }
        }
    }
}

private struct LoadingBayOverviewCard: View {
    let readyCount: Int
    let loadingCount: Int
    let dispatchedCount: Int

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text("Loading bay flow")
                    .font(.title2.bold())
                Text("Track approved transfers, active loading work, and dispatched volume from one queue.")
                    .font(.body)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: LabTheme.spacingSM) {
                BayOverviewMetric(label: "Ready", value: "\(readyCount)")
                BayOverviewMetric(label: "Loading", value: "\(loadingCount)")
                BayOverviewMetric(label: "Out", value: "\(dispatchedCount)")
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct BayOverviewMetric: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct BayTransferCard: View {
    let transfer: Transfer

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
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
                    TransferTag(text: transfer.state)
                    TransferTag(text: transfer.priority.isEmpty ? "STANDARD" : transfer.priority, emphasized: false)
                }
            }

            HStack(spacing: LabTheme.spacingSM) {
                BayTransferMetric(label: "Items", value: "\(transfer.totalItems)")
                BayTransferMetric(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
            }
        }
        .labCard()
    }
}

private struct BayTransferMetric: View {
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
        .padding(LabTheme.spacingMD)
        .background(LabTheme.tertiaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct TransferTag: View {
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

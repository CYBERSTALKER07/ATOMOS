import SwiftUI

struct TransferDetailView: View {
    let transferId: String
    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var transfer: Transfer?
    @State private var loading = true
    @State private var error: String?
    @State private var transitioning = false

    var body: some View {
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
            } else if let transfer {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        TransferOverviewCard(transfer: transfer)

                        LazyVGrid(
                            columns: [
                                GridItem(.flexible(), spacing: LabTheme.spacingMD),
                                GridItem(.flexible(), spacing: LabTheme.spacingMD)
                            ],
                            spacing: LabTheme.spacingMD
                        ) {
                            SummaryCard(label: "Items", value: "\(transfer.totalItems)")
                            SummaryCard(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
                        }

                        if transfer.state == "APPROVED" || transfer.state == "LOADING" {
                            HStack(spacing: LabTheme.spacingMD) {
                                if transfer.state == "APPROVED" {
                                    Button("Start Loading") {
                                        Task { await transition(to: "LOADING") }
                                    }
                                    .frame(maxWidth: .infinity)
                                    .buttonStyle(.borderedProminent)
                                }

                                if transfer.state == "LOADING" {
                                    Button("Mark Dispatched") {
                                        Task { await transition(to: "DISPATCHED") }
                                    }
                                    .frame(maxWidth: .infinity)
                                    .buttonStyle(.borderedProminent)
                                }
                            }
                            .disabled(transitioning)
                        } else {
                            Text("No manual transition is available for the current state.")
                                .font(.body)
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(LabTheme.spacingLG)
                                .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
                        }

                        Divider()

                        Text("Items")
                            .font(.headline)

                        ForEach(Array(transfer.items.enumerated()), id: \.element.id) { index, item in
                            VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
                                VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                    Text(item.productName.isEmpty ? String(item.productId.prefix(8)) : item.productName)
                                        .font(.subheadline.bold())
                                    Text(String(item.productId.prefix(8)))
                                        .font(.footnote)
                                        .foregroundStyle(.secondary)
                                }

                                HStack(spacing: LabTheme.spacingSM) {
                                    SummaryCard(label: "Qty", value: "\(item.quantity)")
                                    SummaryCard(label: "Available", value: "\(item.quantityAvailable)")
                                    SummaryCard(label: "Volume", value: String(format: "%.1fL", item.unitVolumeL))
                                }
                            }
                            .labCard()
                            .staggeredAppear(index: index)
                        }

                        if !transfer.notes.isEmpty {
                            Divider()
                            Text("Notes")
                                .font(.headline)
                            Text(transfer.notes)
                                .font(.body)
                                .foregroundStyle(.secondary)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .padding(LabTheme.spacingLG)
                                .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
                        }
                    }
                    .padding()
                }
            }
        }
        .background(LabTheme.background)
        .navigationTitle("Transfer")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Refresh", systemImage: "arrow.clockwise") {
                    Task { await load() }
                }
                .labelStyle(.iconOnly)
            }
        }
        .task { await load() }
        .onAppear {
            realtimeClient.connect(
                onStateChange: { _ in },
                onEvent: { event in
                    guard let eventType = event.eventType else { return }
                    guard eventType == .transferUpdate || eventType == .manifestUpdate else { return }
                    if !transitioning {
                        Task { await load() }
                    }
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
            transfer = try await FactoryService.transfer(id: transferId)
        } catch {
            self.error = error.localizedDescription
        }

        loading = false
    }

    @MainActor
    private func transition(to target: String) async {
        transitioning = true

        do {
            transfer = try await FactoryService.transitionTransfer(id: transferId, target: target)
        } catch {
            self.error = error.localizedDescription
        }

        transitioning = false
    }
}

private struct TransferOverviewCard: View {
    let transfer: Transfer

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingMD) {
            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                Text(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)
                    .font(.title3.bold())
                Text("Transfer \(transfer.id.prefix(8))")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
            }

            HStack(spacing: LabTheme.spacingSM) {
                DetailTag(text: transfer.state)
                DetailTag(text: transfer.priority.isEmpty ? "STANDARD" : transfer.priority, emphasized: false)
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .labCard()
    }
}

private struct SummaryCard: View {
    let label: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
            Text(value)
                .font(.headline)
            Text(label)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(LabTheme.spacingMD)
        .background(LabTheme.secondaryBackground, in: RoundedRectangle(cornerRadius: LabTheme.radiusMD))
    }
}

private struct DetailTag: View {
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

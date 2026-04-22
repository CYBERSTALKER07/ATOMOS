import SwiftUI

struct TransferDetailView: View {
    let transferId: String
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
                    Button("Retry") { load() }
                }
            } else if let transfer {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        // Summary cards
                        HStack(spacing: LabTheme.spacingMD) {
                            SummaryCard(label: "State", value: transfer.state)
                            SummaryCard(label: "Priority", value: transfer.priority)
                            SummaryCard(label: "Items", value: "\(transfer.totalItems)")
                            SummaryCard(label: "Volume", value: String(format: "%.0fL", transfer.totalVolumeL))
                        }

                        // Warehouse
                        Text("Warehouse: \(transfer.warehouseName.isEmpty ? String(transfer.warehouseId.prefix(8)) : transfer.warehouseName)")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)

                        // Actions
                        HStack(spacing: LabTheme.spacingMD) {
                            if transfer.state == "APPROVED" {
                                Button("Start Loading") { transition(to: "LOADING") }
                                    .buttonStyle(.borderedProminent)
                                    .disabled(transitioning)
                            }
                            if transfer.state == "LOADING" {
                                Button("Mark Dispatched") { transition(to: "DISPATCHED") }
                                    .buttonStyle(.borderedProminent)
                                    .disabled(transitioning)
                            }
                        }

                        Divider()

                        // Items
                        Text("Items")
                            .font(.headline)

                        ForEach(Array(transfer.items.enumerated()), id: \.element.id) { index, item in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(item.productName.isEmpty ? String(item.productId.prefix(8)) : item.productName)
                                        .font(.subheadline.bold())
                                    Text("Qty: \(item.quantity) · Available: \(item.quantityAvailable)")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                Text(String(format: "%.1fL/unit", item.unitVolumeL))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .labCard()
                            .staggeredAppear(index: index)
                        }

                        // Notes
                        if !transfer.notes.isEmpty {
                            Divider()
                            Text("Notes")
                                .font(.headline)
                            Text(transfer.notes)
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
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
                Button { load() } label: {
                    Image(systemName: "arrow.clockwise")
                }
            }
        }
        .task { load() }
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                transfer = try await FactoryService.transfer(id: transferId)
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
        }
    }

    private func transition(to target: String) {
        transitioning = true
        Task {
            do {
                transfer = try await FactoryService.transitionTransfer(id: transferId, target: target)
            } catch {
                self.error = error.localizedDescription
            }
            transitioning = false
        }
    }
}

// MARK: - Summary Card
private struct SummaryCard: View {
    let label: String
    let value: String

    var body: some View {
        VStack(spacing: 4) {
            Text(value)
                .font(.title3.bold())
            Text(label)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .frame(maxWidth: .infinity)
        .labCard()
    }
}

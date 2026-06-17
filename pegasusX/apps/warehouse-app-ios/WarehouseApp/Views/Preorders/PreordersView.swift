import SwiftUI

struct PreordersView: View {
    @State private var rows: [WarehousePreorderRow] = []
    @State private var loading = true

    var body: some View {
        Group {
            if loading {
                ProgressView()
            } else if rows.isEmpty {
                ContentUnavailableView("No pre-orders", systemImage: "calendar")
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.orderId).font(.headline)
                        Text(row.status).font(.caption)
                        if let date = row.requestedDeliveryDate {
                            Text(date).font(.caption2)
                        }
                    }
                }
            }
        }
        .navigationTitle("Pre-orders")
        .task { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            let data = try await WarehouseService.preorders()
            rows = data.preorders.isEmpty ? data.items : data.preorders
        } catch {
            rows = []
        }
    }
}

struct StockCommitmentsView: View {
    @State private var rows: [StockCommitmentRow] = []
    @State private var loading = true

    var body: some View {
        Group {
            if loading {
                ProgressView()
            } else if rows.isEmpty {
                ContentUnavailableView("No commitments", systemImage: "archivebox")
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.name ?? row.skuId).font(.headline)
                        Text("On hand \(row.onHand) · ASAP \(row.reservedAsap) · Scheduled \(row.reservedScheduled)")
                            .font(.caption)
                        if row.deficitQty > 0 {
                            Text("Short \(row.deficitQty)").font(.caption).foregroundStyle(.red)
                        }
                    }
                }
            }
        }
        .navigationTitle("Stock commitments")
        .task { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            let data = try await WarehouseService.stockCommitments()
            rows = data.items.isEmpty ? data.skus : data.items
        } catch {
            rows = []
        }
    }
}

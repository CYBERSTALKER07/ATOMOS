import SwiftUI

struct StoreStockView: View {
    @State private var rows: [StoreStockRow] = []
    @State private var locationId: String = ""
    @State private var banner: String?
    @State private var orderId = ""
    @State private var sku = ""
    @State private var qty = "1"
    @State private var countSku = ""
    @State private var countQty = "0"
    @State private var countBin = "FLOOR"
    @State private var countVersion: Int64 = 0
    @State private var busy = false

    private let api = APIClient.shared

    var body: some View {
        List {
            Section {
                Text("Receive deliveries into backroom, putaway to floor, adjust quantities.")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Receive order") {
                TextField("Order ID", text: $orderId)
                Button(busy ? "…" : "Receive to BACKROOM") {
                    Task { await receive() }
                }
                .disabled(busy || orderId.isEmpty)
            }
            Section("Putaway / adjust") {
                TextField("SKU", text: $sku)
                TextField("Qty", text: $qty)
                Button("Putaway BACKROOM→FLOOR") { Task { await transfer() } }
                    .disabled(busy || sku.isEmpty)
                Button("Adjust BACKROOM by qty") { Task { await adjust() } }
                    .disabled(busy || sku.isEmpty)
            }
            Section("Count (versioned)") {
                TextField("SKU", text: $countSku)
                TextField("Counted qty", text: $countQty)
                Picker("Bin", selection: $countBin) {
                    Text("FLOOR").tag("FLOOR")
                    Text("BACKROOM").tag("BACKROOM")
                }
                Text("Base version: \(countVersion)")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
                Button("Commit count") { Task { await commitCount(force: false) } }
                    .disabled(busy || countSku.isEmpty || locationId.isEmpty)
            }
            Section("Balances") {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 2) {
                        Text(row.sku).font(.headline)
                        Text("\(row.bin): on hand \(row.onHand) · available \(row.available)")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                }
            }
        }
        .navigationTitle("Store stock")
        .navigationBarTitleDisplayMode(.inline)
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        do {
            if locationId.isEmpty {
                let locs = try await api.getLocations()
                locationId = locs.activeLocationId ?? locs.items.first?.locationId ?? ""
            }
            let res = try await api.getStoreStock(locationId: locationId.isEmpty ? nil : locationId)
            rows = res.items.map {
                StoreStockRow(
                    id: "\($0.sku)-\($0.stockBin)",
                    sku: $0.sku,
                    bin: $0.stockBin,
                    onHand: $0.onHand,
                    available: $0.available ?? $0.onHand
                )
            }
            if !locationId.isEmpty {
                if let ver = try? await api.getStockCountVersion(locationId: locationId, stockBin: countBin) {
                    countVersion = ver.version
                }
            }
        } catch {
            banner = error.localizedDescription
        }
    }

    private func receive() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.receiveStoreStock(orderId: orderId, locationId: locationId)
            banner = "Received"
            orderId = ""
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func transfer() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.transferStoreStock(
                locationId: locationId,
                sku: sku,
                qty: Int64(qty) ?? 1
            )
            banner = "Putaway done"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func adjust() async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.adjustStoreStock(
                locationId: locationId,
                sku: sku,
                qtyDelta: Int64(qty) ?? 0
            )
            banner = "Adjusted"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }

    private func commitCount(force: Bool) async {
        busy = true
        defer { busy = false }
        do {
            _ = try await api.commitStockCount(
                locationId: locationId,
                stockBin: countBin,
                baseVersion: countVersion,
                sku: countSku,
                countedQty: Int64(countQty) ?? 0,
                force: force
            )
            banner = force ? "Count force-committed" : "Count committed"
            await load()
        } catch {
            banner = error.localizedDescription
        }
    }
}

struct StoreStockRow: Identifiable {
    let id: String
    let sku: String
    let bin: String
    let onHand: Int64
    let available: Int64
}



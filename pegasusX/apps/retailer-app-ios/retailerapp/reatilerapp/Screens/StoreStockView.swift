import SwiftUI

struct StoreStockView: View {
    @State private var rows: [StoreStockRow] = []
    @State private var locationId: String = ""
    @State private var banner: String?
    @State private var orderId = ""
    @State private var sku = ""
    @State private var qty = "1"
    @State private var busy = false

    @State private var showOrderPicker = false
    @State private var preferredSku: String?
    @State private var claimableOrders: [Order] = []
    @State private var pickerQuery = ""
    @State private var pickerLoading = false
    @State private var pickerError: String?
    @State private var claimOrder: Order?
    @State private var claimPreferredSku: String?

    private let api = APIClient.shared

    private var filteredOrders: [Order] {
        let q = pickerQuery.trimmingCharacters(in: .whitespacesAndNewlines).lowercased()
        if q.isEmpty { return claimableOrders }
        return claimableOrders.filter { $0.id.lowercased().contains(q) }
    }

    var body: some View {
        List {
            Section {
                Text("Receive deliveries into backroom, putaway to floor, adjust quantities.")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                Button("Request return / chargeback") {
                    openRequestReturn(skuHint: nil)
                }
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
            Section("Balances") {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.sku).font(.headline)
                        Text("\(row.bin): on hand \(row.onHand) · available \(row.available)")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                        Button("Request return") {
                            openRequestReturn(skuHint: row.sku)
                        }
                        .font(.caption)
                    }
                }
            }
        }
        .navigationTitle("Store stock")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("Return") { openRequestReturn(skuHint: nil) }
            }
        }
        .task { await load() }
        .refreshable { await load() }
        .sheet(isPresented: $showOrderPicker) {
            NavigationStack {
                List {
                    Section {
                        Text("Pick a completed delivery, then file the same claim as order detail. Window is within 48h (server enforces).")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if let preferredSku {
                            Text("Preferred SKU from stock: \(preferredSku)")
                                .font(.caption)
                        }
                        TextField("Search by order id", text: $pickerQuery)
                    }
                    if let pickerError {
                        Section {
                            Text(pickerError).foregroundStyle(.red).font(.caption)
                        }
                    }
                    Section("COMPLETED orders") {
                        if pickerLoading {
                            ProgressView()
                        } else if filteredOrders.isEmpty {
                            Text("No COMPLETED / DELIVERED_ON_CREDIT orders found.")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(filteredOrders) { order in
                                Button {
                                    Task { await pickOrder(order) }
                                } label: {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text("#\(order.id.suffix(8)) · \(order.status.rawValue.replacingOccurrences(of: "_", with: " "))")
                                            .font(.subheadline.weight(.semibold))
                                        Text(order.id)
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                }
                            }
                        }
                    }
                }
                .navigationTitle("Request return")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Close") { showOrderPicker = false }
                    }
                }
            }
        }
        .sheet(item: $claimOrder) { order in
            FileClaimView(order: order, preferredSku: claimPreferredSku)
        }
    }

    private func openRequestReturn(skuHint: String?) {
        preferredSku = skuHint
        pickerQuery = ""
        pickerError = nil
        showOrderPicker = true
        Task { await loadClaimableOrders() }
    }

    private func loadClaimableOrders() async {
        pickerLoading = true
        defer { pickerLoading = false }
        let rid = AuthManager.shared.currentUser?.id ?? ""
        guard !rid.isEmpty else {
            pickerError = "Not signed in"
            claimableOrders = []
            return
        }
        do {
            let orders: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            claimableOrders = orders.filter {
                $0.status == .completed || $0.status == .deliveredOnCredit
            }
        } catch {
            pickerError = error.localizedDescription
            claimableOrders = []
        }
    }

    private func pickOrder(_ order: Order) async {
        pickerLoading = true
        defer { pickerLoading = false }
        var next = order
        if next.items.isEmpty {
            if let tracking = try? await api.getTracking() {
                let pool = tracking.orders + (tracking.recentReceipts ?? [])
                if let hit = pool.first(where: { $0.orderId == order.id }) {
                    next = hit.asClaimOrder()
                }
            }
        }
        claimPreferredSku = preferredSku
        claimOrder = next
        showOrderPicker = false
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
}

struct StoreStockRow: Identifiable {
    let id: String
    let sku: String
    let bin: String
    let onHand: Int64
    let available: Int64
}

private extension TrackingOrder {
    func asClaimOrder() -> Order {
        let status = OrderStatus(rawValue: state) ?? .completed
        let lineItems = items.map { item in
            OrderLineItem(
                id: item.productId,
                productId: item.productId,
                productName: item.productName,
                variantId: item.productId,
                variantSize: "",
                quantity: item.quantity,
                unitPrice: item.unitPrice,
                totalPrice: item.lineTotal
            )
        }
        return Order(
            id: orderId,
            retailerId: "",
            supplierId: supplierId,
            supplierName: supplierName,
            status: status,
            items: lineItems,
            totalAmount: Int64(totalAmount),
            orderSource: orderSource,
            createdAt: createdAt,
            updatedAt: createdAt,
            estimatedDelivery: nil,
            qrCode: deliveryToken.isEmpty ? nil : deliveryToken
        )
    }
}

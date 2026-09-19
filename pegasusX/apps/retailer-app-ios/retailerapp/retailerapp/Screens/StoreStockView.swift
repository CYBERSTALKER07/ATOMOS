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
                Text("mobile_retailer.ui.receive_deliveries_into_backroom_putaway_to_floor_adjust_quantit")
                    .font(.system(.footnote, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                Button("mobile_retailer.ui.request_return_chargeback") {
                    openRequestReturn(skuHint: nil)
                }
            }
            if let banner {
                Section { Text(banner).font(.caption).foregroundStyle(AppTheme.accent) }
            }
            Section("Receive order") {
                TextField("supplier_portal.admin.control_center.field.order_id", text: $orderId)
                Button(busy ? "…" : "Receive to BACKROOM") {
                    Task { await receive() }
                }
                .disabled(busy || orderId.isEmpty)
            }
            Section("Putaway / adjust") {
                TextField("SKU", text: $sku)
                TextField("retailer_desktop.pos.text.qty", text: $qty)
                Button("mobile_retailer.ui.putaway_backroom_floor") { Task { await transfer() } }
                    .disabled(busy || sku.isEmpty)
                Button("mobile_retailer.ui.adjust_backroom_by_qty") { Task { await adjust() } }
                    .disabled(busy || sku.isEmpty)
            }
            Section("Balances") {
                ForEach(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.sku).font(.headline)
                        Text(L10n.format("mobile_retailer.ui.bin_on_hand_onhand_available_available", "\(row.bin)", "\(row.onHand)", "\(row.available)"))
                            .font(.caption)
                            .foregroundStyle(AppTheme.textSecondary)
                        Button("mobile_retailer.ui.request_return") {
                            openRequestReturn(skuHint: row.sku)
                        }
                        .font(.caption)
                    }
                }
            }
        }
        .navigationTitle("portal.nav.store_stock")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_retailer.ui.return") { openRequestReturn(skuHint: nil) }
            }
        }
        .task { await load() }
        .refreshable { await load() }
        .sheet(isPresented: $showOrderPicker) {
            NavigationStack {
                List {
                    Section {
                        Text("mobile_retailer.ui.pick_a_completed_delivery_then_file_the_same_claim_as_order_deta")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if let preferredSku {
                            Text(L10n.format("mobile_retailer.ui.preferred_sku_from_stock_preferredsku_2", "\(preferredSku)"))
                                .font(.caption)
                        }
                        TextField("retailer_desktop.stock_request_return_modal.text.search_by_order_id", text: $pickerQuery)
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
                            Text("mobile_retailer.ui.no_completed_delivered_on_credit_orders_found")
                                .foregroundStyle(.secondary)
                        } else {
                            ForEach(filteredOrders) { order in
                                Button {
                                    Task { await pickOrder(order) }
                                } label: {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(L10n.format("mobile_retailer.ui.suffix_replacingoccurrences", "\(order.id.suffix(8))", "\(order.status.rawValue.replacingOccurrences(of: "_", with: " "))"))
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
                .navigationTitle("mobile_retailer.ui.request_return")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.close") { showOrderPicker = false }
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

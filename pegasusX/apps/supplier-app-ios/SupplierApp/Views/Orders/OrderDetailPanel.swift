import SwiftUI

struct OrderDetailPanel: View {
    let order: SupplierOrder
    @Bindable var vm: OrdersViewModel
    @State private var warehouseDetail: WarehouseOrderDetail?
    @State private var opsReason = ""
    @State private var proposeDate = Date()
    @State private var showProposeSheet = false
    @State private var showRejectDialog = false

    var body: some View {
        List {
            Section("supplier_portal.chargebacks.claims.text.order") {
                LabeledContent("ID", value: order.orderId)
                LabeledContent("Retailer", value: warehouseDetail?.retailerName ?? order.retailerId)
                LabeledContent("Status") {
                    SupplierStatusBadge(text: warehouseDetail?.state ?? warehouseDetail?.status ?? order.status)
                }
                if let decision = order.decision, !decision.isEmpty {
                    LabeledContent("Decision", value: decision)
                }
                LabeledContent("Total", value: MoneyFormat.minor(order.totalMinor, currency: order.currency))
                LabeledContent("Updated", value: order.updatedAt)
            }

            if let items = warehouseDetail?.lineItems, !items.isEmpty {
                Section("Line items") {
                    ForEach(items) { item in
                        VStack(alignment: .leading) {
                            Text(item.productName ?? item.productId ?? "—")
                            Text(L10n.format("mobile_supplier.ui.qty_quantity_0", "\(item.quantity ?? 0)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }

            if vm.canWarehouseOps(for: order) {
                Section("Warehouse admin") {
                    Button("mobile_supplier.ui.reassign_order") {
                        Task { await vm.openReassignDialog(orderId: order.orderId) }
                    }
                    Button("supplier_portal.orders.propose_delay_dialog.text.delay_delivery") { showProposeSheet = true }
                    Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) { showRejectDialog = true }
                }
            }
        }
        .navigationTitle("supplier_portal.chargebacks.claims.text.order")
        .task { await loadWarehouseDetail() }
        .sheet(isPresented: $showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("New delivery date", selection: $proposeDate, displayedComponents: .date)
                    TextField("supplier_portal.orders.order_ops_actions.text.reason_required", text: $opsReason, axis: .vertical)
                }
                .navigationTitle("supplier_portal.orders.propose_delay_dialog.text.delay_delivery")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) { Button("common.action.cancel") { showProposeSheet = false } }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_supplier.ui.notify_retailer") {
                            Task {
                                await vm.proposeWarehouseOrder(
                                    order,
                                    proposedDeliveryDate: isoDeliveryDate(from: proposeDate),
                                    reason: opsReason
                                )
                                opsReason = ""
                                showProposeSheet = false
                                await loadWarehouseDetail()
                            }
                        }
                        .disabled(opsReason.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .alert("Cancel order", isPresented: $showRejectDialog) {
            Button("mobile_supplier.ui.reject", role: .destructive) {
                Task {
                    await vm.rejectWarehouseOrder(order, reason: opsReason)
                    opsReason = ""
                    await loadWarehouseDetail()
                }
            }
            Button("common.action.cancel", role: .cancel) { opsReason = "" }
        } message: {
            Text("mobile_supplier.ui.reason_is_required_for_reject")
        }
        .sheet(
            isPresented: Binding(
                get: { vm.reassignTarget != nil },
                set: { if !$0 { vm.closeReassignDialog() } }
            )
        ) {
            NavigationStack {
                Group {
                    if let recs = vm.reassignRecommendations {
                        if recs.recommendations.isEmpty {
                            ContentUnavailableView("No Trucks", systemImage: "car.fill", description: Text("mobile_supplier.ui.no_suitable_trucks_available"))
                        } else {
                            List {
                                Section {
                                    LabeledContent("Retailer", value: recs.retailerName)
                                    LabeledContent("Volume", value: String(format: "%.1f VU", recs.orderVolumeVu))
                                }
                                Section("Available drivers") {
                                    ForEach(recs.recommendations) { rec in
                                        VStack(alignment: .leading, spacing: 8) {
                                            HStack {
                                                Text(rec.driverName.isEmpty ? String(rec.driverId.prefix(8)) : rec.driverName)
                                                    .font(.headline)
                                                Spacer()
                                                Text(String(format: "score %.2f", rec.score))
                                                    .font(.caption)
                                                    .foregroundColor(.secondary)
                                            }
                                            Text(L10n.format("mobile_supplier.ui.licenseplate_vehicleclass", "\(rec.licensePlate)", "\(rec.vehicleClass)"))
                                                .font(.subheadline)
                                                .foregroundColor(.secondary)
                                            HStack {
                                                Spacer()
                                                Button("mobile_supplier.ui.partial") {
                                                    Task { await vm.applyReassign(orderId: order.orderId, driverId: rec.driverId, isPartial: true) }
                                                }
                                                .buttonStyle(.bordered)
                                                .disabled(vm.isReassigning)
                                                
                                                Button("mobile_supplier.ui.complete") {
                                                    Task { await vm.applyReassign(orderId: order.orderId, driverId: rec.driverId, isPartial: false) }
                                                }
                                                .buttonStyle(.borderedProminent)
                                                .disabled(vm.isReassigning)
                                            }
                                        }
                                        .padding(.vertical, 4)
                                    }
                                }
                            }
                        }
                    } else {
                        ProgressView("Loading recommendations...")
                    }
                }
                .navigationTitle("supplier_portal.orders.re_dispatch_dialog.text.reassign_order")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.close") {
                            vm.closeReassignDialog()
                        }
                        .disabled(vm.isReassigning)
                    }
                }
            }
            .presentationDetents([.medium, .large])
        }
    }

    private func isoDeliveryDate(from date: Date) -> String {
        var components = Calendar.current.dateComponents(in: TimeZone(secondsFromGMT: 5 * 3600)!, from: date)
        components.hour = 12
        components.minute = 0
        components.second = 0
        let noon = Calendar.current.date(from: components) ?? date
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        formatter.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
        return formatter.string(from: noon)
    }

    private func loadWarehouseDetail() async {
        guard let warehouseId = order.warehouseId, !warehouseId.isEmpty else { return }
        do {
            warehouseDetail = try await SupplierOperationsService.getWarehouseOrder(
                orderId: order.orderId,
                warehouseId: warehouseId
            )
        } catch {
            warehouseDetail = nil
        }
    }
}

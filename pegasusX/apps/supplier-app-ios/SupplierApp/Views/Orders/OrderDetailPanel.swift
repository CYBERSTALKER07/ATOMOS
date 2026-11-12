import SwiftUI

struct OrderDetailPanel: View {
    let order: SupplierOrder
    @Bindable var vm: OrdersViewModel
    @State private var detailVM = OrderDetailViewModel()

    var body: some View {
        @Bindable var detailVM = detailVM
        List {
            Section("supplier_portal.chargebacks.claims.text.order") {
                LabeledContent("mobile_supplier.chrome.id_label", value: order.orderId)
                LabeledContent("mobile_supplier.chrome.retailer", value: detailVM.warehouseDetail?.retailerName ?? order.retailerId)
                LabeledContent("mobile_supplier.chrome.status") {
                    SupplierStatusBadge(text: detailVM.warehouseDetail?.state ?? detailVM.warehouseDetail?.status ?? order.status)
                }
                if let decision = order.decision, !decision.isEmpty {
                    LabeledContent("mobile_supplier.chrome.decision", value: decision)
                }
                LabeledContent("mobile_supplier.chrome.total", value: MoneyFormat.minor(order.totalMinor, currency: order.currency))
                LabeledContent("mobile_supplier.chrome.updated", value: order.updatedAt)
            }

            if let detailError = detailVM.detailError, detailVM.warehouseDetail == nil {
                Section {
                    Text(detailError).foregroundStyle(SupplierTheme.destructive)
                }
            }
            if let items = detailVM.warehouseDetail?.lineItems, !items.isEmpty {
                Section("mobile_supplier.chrome.line_items") {
                    ForEach(items) { item in
                        VStack(alignment: .leading) {
                            Text(item.productName ?? item.productId ?? "—")
                            Text(L10n.format("mobile_supplier.ui.qty_quantity_0", MissingNumber.label(item.quantity)))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
            }

            if vm.canWarehouseOps(for: order) {
                Section("mobile_supplier.chrome.warehouse_admin") {
                    Button("mobile_supplier.ui.reassign_order") {
                        PegasusHaptics.light()
                        Task { await vm.openReassignDialog(orderId: order.orderId) }
                    }
                    .frame(minHeight: 44)
                    Button("supplier_portal.orders.propose_delay_dialog.text.delay_delivery") {
                        PegasusHaptics.light()
                        detailVM.showProposeSheet = true
                    }
                    .frame(minHeight: 44)
                    Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) {
                        PegasusHaptics.warning()
                        detailVM.showRejectDialog = true
                    }
                    .frame(minHeight: 44)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("supplier_portal.chargebacks.claims.text.order")
        .task(id: order.orderId) { await detailVM.loadWarehouseDetail(for: order) }
        .sheet(isPresented: $detailVM.showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("mobile_supplier.chrome.new_delivery_date", selection: $detailVM.proposeDate, displayedComponents: .date)
                    TextField("supplier_portal.orders.order_ops_actions.text.reason_required", text: $detailVM.opsReason, axis: .vertical)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("supplier_portal.orders.propose_delay_dialog.text.delay_delivery")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { detailVM.showProposeSheet = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_supplier.ui.notify_retailer") {
                            Task {
                                await vm.proposeWarehouseOrder(
                                    order,
                                    proposedDeliveryDate: detailVM.isoDeliveryDate(),
                                    reason: detailVM.opsReason
                                )
                                detailVM.opsReason = ""
                                detailVM.showProposeSheet = false
                                await detailVM.loadWarehouseDetail(for: order)
                            }
                        }
                        .disabled(detailVM.opsReason.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                    }
                }
            }
            .presentationDetents([.medium])
        }
        .alert("mobile_supplier.chrome.cancel_order", isPresented: $detailVM.showRejectDialog) {
            Button("mobile_supplier.ui.reject", role: .destructive) {
                Task {
                    await vm.rejectWarehouseOrder(order, reason: detailVM.opsReason)
                    detailVM.opsReason = ""
                    await detailVM.loadWarehouseDetail(for: order)
                }
            }
            Button("common.action.cancel", role: .cancel) { detailVM.opsReason = "" }
        } message: {
            Text("mobile_supplier.ui.reason_is_required_for_reject")
        }
        .sheet(
            isPresented: $vm.showReassignSheet,
            onDismiss: { vm.closeReassignDialog() }
        ) {
            NavigationStack {
                Group {
                    if let recs = vm.reassignRecommendations {
                        if recs.recommendations.isEmpty {
                            ContentUnavailableView(
                                "mobile_supplier.chrome.no_trucks",
                                systemImage: "car.fill",
                                description: Text("mobile_supplier.ui.no_suitable_trucks_available")
                            )
                        } else {
                            List {
                                Section {
                                    LabeledContent("mobile_supplier.chrome.retailer", value: recs.retailerName)
                                    LabeledContent("mobile_supplier.chrome.volume", value: String(format: "%.1f VU", recs.orderVolumeVu))
                                }
                                Section("mobile_supplier.chrome.available_drivers") {
                                    ForEach(recs.recommendations) { rec in
                                        VStack(alignment: .leading, spacing: 8) {
                                            HStack {
                                                Text(rec.driverName.isEmpty ? String(rec.driverId.prefix(8)) : rec.driverName)
                                                    .font(.headline)
                                                Spacer()
                                                Text(L10n.format("mobile_supplier.chrome.score_value", String(format: "%.2f", rec.score)))
                                                    .font(.caption)
                                                    .foregroundStyle(.secondary)
                                            }
                                            Text(L10n.format("mobile_supplier.ui.licenseplate_vehicleclass", "\(rec.licensePlate)", "\(rec.vehicleClass)"))
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                            HStack {
                                                Spacer()
                                                Button("mobile_supplier.ui.partial") {
                                                    PegasusHaptics.light()
                                                    Task { await vm.applyReassign(orderId: order.orderId, driverId: rec.driverId, isPartial: true) }
                                                }
                                                .buttonStyle(.bordered)
                                                .frame(minHeight: 44)
                                                .disabled(vm.isReassigning)

                                                Button("mobile_supplier.ui.complete") {
                                                    PegasusHaptics.light()
                                                    Task { await vm.applyReassign(orderId: order.orderId, driverId: rec.driverId, isPartial: false) }
                                                }
                                                .buttonStyle(.borderedProminent)
                                                .frame(minHeight: 44)
                                                .disabled(vm.isReassigning)
                                            }
                                        }
                                        .padding(.vertical, 4)
                                    }
                                }
                            }
                        }
                    } else {
                        ProgressView("mobile_supplier.loading.reassign")
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
}

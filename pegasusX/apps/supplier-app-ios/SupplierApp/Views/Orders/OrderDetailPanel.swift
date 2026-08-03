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
            Section("Order") {
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
                            Text("Qty \(item.quantity ?? 0)")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }
                }
            }

            if vm.canWarehouseOps(for: order) {
                Section("Warehouse admin") {
                    Button("Reassign order") {
                        Task { await vm.openReassignDialog(orderId: order.orderId) }
                    }
                    Button("Delay delivery") { showProposeSheet = true }
                    Button("Cancel order", role: .destructive) { showRejectDialog = true }
                }
            }
        }
        .navigationTitle("Order")
        .task { await loadWarehouseDetail() }
        .sheet(isPresented: $showProposeSheet) {
            NavigationStack {
                Form {
                    DatePicker("New delivery date", selection: $proposeDate, displayedComponents: .date)
                    TextField("Reason (required)", text: $opsReason, axis: .vertical)
                }
                .navigationTitle("Delay delivery")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) { Button("Cancel") { showProposeSheet = false } }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("Notify retailer") {
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
            Button("Reject", role: .destructive) {
                Task {
                    await vm.rejectWarehouseOrder(order, reason: opsReason)
                    opsReason = ""
                    await loadWarehouseDetail()
                }
            }
            Button("Cancel", role: .cancel) { opsReason = "" }
        } message: {
            Text("Reason is required for reject.")
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
                            ContentUnavailableView("No Trucks", systemImage: "car.fill", description: Text("No suitable trucks available."))
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
                                            Text("\(rec.licensePlate) • \(rec.vehicleClass)")
                                                .font(.subheadline)
                                                .foregroundColor(.secondary)
                                            HStack {
                                                Spacer()
                                                Button("Partial") {
                                                    Task { await vm.applyReassign(orderId: order.orderId, driverId: rec.driverId, isPartial: true) }
                                                }
                                                .buttonStyle(.bordered)
                                                .disabled(vm.isReassigning)
                                                
                                                Button("Complete") {
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
                .navigationTitle("Reassign Order")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Close") {
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

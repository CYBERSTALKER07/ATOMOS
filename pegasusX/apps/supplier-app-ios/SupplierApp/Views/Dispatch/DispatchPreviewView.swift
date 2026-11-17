import SwiftUI

struct DispatchPreviewView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    var embeddedInHub: Bool = false
    @State private var vm = DispatchPreviewViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.preview == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.dispatch")
            } else if let error = vm.error, vm.preview == nil {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else {
                ScrollView {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingLG) {
                        Picker("mobile_supplier.chrome.mode", selection: $vm.dispatchMode) {
                            ForEach(DispatchMode.allCases, id: \.self) { mode in
                                Text(LocalizedStringKey(mode == .manual ? "mobile_supplier.chrome.manual_truck" : "mobile_supplier.chrome.smart_assign")).tag(mode)
                            }
                        }
                        .pickerStyle(.segmented)
                        .onChange(of: vm.dispatchMode) { _, _ in
                            PegasusHaptics.selection()
                        }

                        if !vm.warehouses.isEmpty {
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                Text("supplier_portal.dispatch.text.warehouse_scope").font(.headline)
                                ScrollView(.horizontal, showsIndicators: false) {
                                    HStack(spacing: SupplierTheme.spacingSM) {
                                        ForEach(vm.warehouses) { wh in
                                            Button {
                                                Task { await vm.toggleWarehouse(wh.warehouseId) }
                                            } label: {
                                                Text(wh.name.isEmpty ? wh.warehouseId : wh.name)
                                                    .font(.subheadline.weight(.medium))
                                                    .padding(.horizontal, 12)
                                                    .padding(.vertical, 8)
                                                    .frame(minHeight: 44)
                                                    .background(vm.selectedWarehouseId == wh.warehouseId ? Color.primary : Color.clear)
                                                    .foregroundStyle(vm.selectedWarehouseId == wh.warehouseId ? Color(.systemBackground) : Color.primary)
                                                    .clipShape(Capsule())
                                                    .overlay {
                                                        Capsule().strokeBorder(Color.primary.opacity(0.25), lineWidth: vm.selectedWarehouseId == wh.warehouseId ? 0 : 1)
                                                    }
                                            }
                                            .buttonStyle(.plain)
                                        }
                                    }
                                }
                            }
                        }
                        if let preview = vm.preview {
                            if preview.planFingerprintMismatch {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("mobile_supplier.ui.dispatch_plan_drift")
                                        .font(.subheadline.weight(.semibold))
                                        .foregroundStyle(SupplierTheme.destructive)
                                    Text("mobile_supplier.ui.supplier_preview_differs_from_warehouse_floor_plan_refresh_wareh")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                .padding()
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .background(SupplierTheme.destructive.opacity(0.08))
                                .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
                            }
                            HStack(spacing: SupplierTheme.spacingMD) {
                                DispatchKpiCard(title: "mobile_supplier.chrome.pending", value: MissingNumber.label(preview.pendingCount))
                                DispatchKpiCard(title: "mobile_supplier.chrome.drivers", value: MissingNumber.label(preview.availableDriverCount))
                                DispatchKpiCard(title: "mobile_supplier.chrome.undispatched", value: "\(preview.undispatchedOrderCount ?? preview.undispatchedOrders.count)")
                            }
                            if vm.dispatchMode == .manual, !preview.undispatchedOrders.isEmpty {
                                manualAssignmentSection(preview: preview)
                            }
                            if vm.dispatchMode == .auto, !preview.proposedRoutes.isEmpty {
                                VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                    HStack {
                                        Text("mobile_supplier.ui.route_map").font(.headline)
                                        Spacer()
                                        if let source = preview.optimizerSource, !source.isEmpty {
                                            Text(L10n.format("mobile_supplier.ui.source_source_2", "\(source)"))
                                                .font(.caption)
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                    DispatchPreviewMapView(routes: preview.proposedRoutes)
                                        .frame(height: 320)
                                        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
                                }
                            }
                        }
                        if let executeMessage = vm.executeMessage {
                            Text(executeMessage)
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        if vm.dispatchMode == .auto {
                            Button("mobile_supplier.ui.execute_auto_dispatch") {
                                PegasusHaptics.light()
                                vm.showAutoConfirm = true
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.primary)
                            .frame(minHeight: 44)
                            .disabled(vm.loading || vm.executing || vm.preview == nil)
                        } else {
                            Button(L10n.format("mobile_supplier.ui.manual_dispatch_count", "\(vm.selectedOrderIds.count)")) {
                                PegasusHaptics.light()
                                Task { await vm.executeManual(forceCapacity: false) }
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.primary)
                            .frame(minHeight: 44)
                            .disabled(vm.loading || vm.executing || vm.selectedDriverId.isEmpty || vm.selectedOrderIds.isEmpty)
                        }
                        Button("mobile_supplier.ui.refresh_preview") {
                            PegasusHaptics.light()
                            Task { await vm.load() }
                        }
                        .frame(minHeight: 44)
                        .disabled(vm.loading || vm.executing)
                    }
                    .padding()
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle(embeddedInHub ? LocalizedStringKey("") : LocalizedStringKey("mobile_supplier.chrome.dispatch"))
        .task { await vm.load() }
        .onChange(of: vm.preview?.availableDrivers.map(\.driverId)) { _, _ in
            vm.syncDriverSelection()
        }
        .onChange(of: vm.preview?.undispatchedOrders.map(\.orderId)) { _, ids in
            vm.pruneSelectedOrders(validIds: ids)
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            vm.handleReconnect()
        }
        .alert("mobile_supplier.chrome.execute_auto_dispatch", isPresented: $vm.showAutoConfirm) {
            Button("mobile_supplier.ui.confirm", role: .destructive) { Task { await vm.executeAuto() } }
            Button("common.action.cancel", role: .cancel) {}
        } message: {
            Text("mobile_supplier.ui.assign_pending_orders_via_the_optimizer")
        }
        .alert("mobile_supplier.chrome.capacity_exceeded", isPresented: $vm.showCapacityOverride) {
            Button("supplier_portal.bulk_import_wizard.text.continue", role: .destructive) { Task { await vm.executeManual(forceCapacity: true) } }
            Button("retailer_desktop.stock.text.adjust", role: .cancel) {}
        } message: {
            Text("mobile_supplier.ui.selected_orders_exceed_truck_capacity_continue_anyway")
        }
    }

    @ViewBuilder
    private func manualAssignmentSection(preview: SupplierDispatchPreview) -> some View {
        @Bindable var vm = vm
        let driver = preview.availableDrivers.first { $0.driverId == vm.selectedDriverId }
        let selectedVolume = preview.undispatchedOrders
            .filter { vm.selectedOrderIds.contains($0.orderId) }
            .reduce(0.0) { $0 + $1.volumeVu }
        let truckMax = driver?.maxVolumeVu ?? 0
        let truckEffective = truckMax * DispatchPreviewViewModel.tetrisBuffer
        let capacityExceeded = truckEffective > 0 && selectedVolume > truckEffective

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("mobile_supplier.ui.manual_assignment").font(.headline)
            if truckMax > 0 {
                Text(L10n.format("mobile_supplier.chrome.selected_volume", String(format: "%.1f", selectedVolume), String(format: "%.1f", truckEffective)))
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            if capacityExceeded {
                Text("mobile_supplier.ui.insufficient_truck_space_for_selected_orders")
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.destructive)
            }
            if !preview.availableDrivers.isEmpty {
                Picker("mobile_supplier.chrome.driver", selection: $vm.selectedDriverId) {
                    ForEach(preview.availableDrivers) { d in
                        let vu = d.maxVolumeVu.map { String(format: " · %.0f VU", $0) } ?? ""
                        Text(L10n.format("mobile_supplier.ui.namevu", "\(d.name.isEmpty ? d.driverId : d.name)", "\(vu)")).tag(d.driverId)
                    }
                }
            }
            HStack {
                Text("portal.nav.orders").font(.subheadline.weight(.semibold))
                Spacer()
                Button("supplier_portal.replenishment.suggestions.text.select_all") {
                    vm.selectAllUndispatched()
                }
                .font(.caption)
                .frame(minHeight: 44)
            }
            ForEach(preview.undispatchedOrders) { order in
                let isSelected = vm.selectedOrderIds.contains(order.orderId)
                Button {
                    vm.toggleOrderSelection(order.orderId)
                } label: {
                    HStack(spacing: SupplierTheme.spacingSM) {
                        Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                            .foregroundStyle(isSelected ? Color.accentColor : Color.secondary)
                        Text(order.orderId.prefix(12) + "…")
                            .font(.caption.monospaced())
                            .foregroundStyle(.primary)
                        Spacer()
                        Text(String(format: "%.1f VU", order.volumeVu))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .frame(minHeight: 44)
                    .contentShape(Rectangle())
                }
                .buttonStyle(.plain)
                .accessibilityElement(children: .combine)
                .accessibilityAddTraits(.isButton)
            }
        }
        .padding()
        .background(SupplierTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
    }
}

private struct DispatchKpiCard: View {
    let title: String
    let value: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(LocalizedStringKey(title)).font(.caption).foregroundStyle(.secondary)
            Text(value).font(.title2.weight(.semibold))
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding()
        .background(SupplierTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: SupplierTheme.radiusLG, style: .continuous)
                .strokeBorder(Color.primary.opacity(0.12), lineWidth: 1)
        }
        .accessibilityElement(children: .combine)
    }
}

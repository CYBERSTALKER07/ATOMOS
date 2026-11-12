import SwiftUI

struct TopologyView: View {
    @State private var vm = TopologyViewModel()

    var body: some View {
        Group {
            if vm.loading && vm.warehouses.isEmpty && vm.factories.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.topology")
            } else if let error = vm.error, vm.warehouses.isEmpty && vm.factories.isEmpty && !vm.editing {
                SupplierErrorView(message: error) { Task { await vm.load() } }
            } else if vm.editing {
                editForm
            } else if vm.warehouses.isEmpty && vm.factories.isEmpty {
                VStack(spacing: SupplierTheme.spacingLG) {
                    SupplierEmptyView(title: "mobile_supplier.empty.no_nodes", message: "mobile_supplier.empty.nodes_message")
                    Button("mobile_supplier.ui.configure_topology") {
                        PegasusHaptics.light()
                        vm.editing = true
                    }
                    .frame(minHeight: 44)
                }
            } else {
                readOnlyList
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.topology.text.factories_and_warehouses")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                if vm.editing {
                    Button(LocalizedStringKey(vm.saving ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save")) {
                        PegasusHaptics.light()
                        Task { await vm.save() }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(vm.saving)
                } else {
                    Button("supplier_portal.demand.signals.text.edit") {
                        PegasusHaptics.light()
                        vm.beginEditing()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            }
            if vm.editing {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.cancel") {
                        vm.cancelEditing()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
            }
        }
        .task { await vm.load() }
    }

    private var readOnlyList: some View {
        ResponsiveGridContentWrapper {
            Section(L10n.format("mobile_supplier.ui.warehouses_count", "\(vm.warehouses.count)")) {
                ForEach(vm.warehouses) { node in
                    TopologyNodeRow(
                        name: node.name,
                        address: node.address,
                        meta: L10n.format("mobile_supplier.chrome.km_coverage", String(format: "%.0f", node.coverageRadiusKm))
                    )
                }
            }
            Section(L10n.format("mobile_supplier.ui.factories_count", "\(vm.factories.count)")) {
                ForEach(vm.factories) { node in
                    TopologyNodeRow(
                        name: node.name,
                        address: node.address,
                        meta: L10n.format(node.isActive ? "mobile_supplier.chrome.active" : "mobile_supplier.chrome.inactive")
                    )
                }
            }
        }
    }

    private var editForm: some View {
        @Bindable var vm = vm
        return Form {
            if let error = vm.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            Section("mobile_supplier.chrome.warehouses") {
                ForEach($vm.warehouseDrafts) { $draft in
                    TextField("retailer_desktop.pos.text.name", text: $draft.name)
                    AddressLocationField(value: $draft.location, label: "mobile_supplier.chrome.warehouse_address")
                    TextField("supplier_portal.warehouses.components.warehouse_form.text.coverage_km", text: $draft.coverageRadiusKm)
                        .keyboardType(.decimalPad)
                }
                Button("supplier_portal.warehouses.components.warehouse_form.text.add_warehouse") {
                    vm.addWarehouseDraft()
                }
                .frame(minHeight: 44)
            }
            Section("mobile_supplier.chrome.factories") {
                ForEach($vm.factoryDrafts) { $draft in
                    TextField("retailer_desktop.pos.text.name", text: $draft.name)
                    AddressLocationField(value: $draft.location, label: "mobile_supplier.chrome.factory_address")
                }
                Button("supplier_portal.factories.components.factory_form.text.add_factory") {
                    vm.addFactoryDraft()
                }
                .frame(minHeight: 44)
            }
        }
        .scrollDismissesKeyboard(.interactively)
    }
}

private struct TopologyNodeRow: View {
    let name: String
    let address: String
    let meta: String

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(name.isEmpty ? L10n.format("mobile_supplier.chrome.unnamed_node") : name).font(.body)
            Text(address.isEmpty ? L10n.format("mobile_supplier.chrome.coordinates_on_file") : address)
                .font(.caption)
                .foregroundStyle(.secondary)
            Text(meta)
                .font(.caption)
                .foregroundStyle(.secondary)
        }
        .accessibilityElement(children: .combine)
    }
}

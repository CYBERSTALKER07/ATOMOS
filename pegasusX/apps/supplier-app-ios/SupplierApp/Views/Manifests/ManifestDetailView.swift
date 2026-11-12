import SwiftUI

struct ManifestDetailView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    let manifestId: String
    @State private var viewModel = ManifestDetailViewModel()

    var body: some View {
        Group {
            if viewModel.loading && viewModel.detail == nil {
                SupplierLoadingView(title: "mobile_supplier.loading.manifest")
            } else if let error = viewModel.error, viewModel.detail == nil {
                SupplierErrorView(message: error) { Task { await viewModel.load(manifestId: manifestId) } }
            } else if let detail = viewModel.detail {
                manifestContent(detail, vm: viewModel)
            } else {
                SupplierEmptyView(title: "mobile_supplier.empty.not_found", message: "mobile_supplier.empty.manifest_not_found_message")
            }
        }
        .navigationTitle("warehouse_portal.manifests.text.manifest")
        .task { await viewModel.load(manifestId: manifestId) }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            viewModel.handleReconnect()
        }
    }

    @ViewBuilder
    private func manifestContent(_ detail: SupplierManifestDetail, @Bindable vm: ManifestDetailViewModel) -> some View {
        let state = (detail.state.isEmpty ? detail.status : detail.state).uppercased()
        Form {
            if let actionError = viewModel.actionError {
                Section { Text(actionError).foregroundStyle(SupplierTheme.destructive) }
            }
            Section("mobile_supplier.chrome.summary") {
                Text(detail.manifestId).font(.caption.monospaced())
                Text(L10n.format("mobile_supplier.ui.status_orderscount_orders", "\(detail.status)", "\(detail.ordersCount)"))
                Text(detail.driverName.isEmpty ? (detail.driverId ?? "—") : detail.driverName)
                if let plate = detail.vehiclePlate { Text(L10n.format("mobile_supplier.ui.vehicle_plate_2", "\(plate)")) }
            }
            if state == "DRAFT" {
                Section {
                    Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.starting" : "mobile_supplier.chrome.start_loading")) {
                        Task { await viewModel.startLoading(manifestId: manifestId) }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy)
                }
            }
            if state == "LOADING" {
                Section("mobile_supplier.ui.inject_order") {
                    TextField("supplier_portal.admin.control_center.field.order_id", text: $vm.injectOrderId)
                    Button("mobile_supplier.ui.inject_order") {
                        Task { await viewModel.injectOrder(manifestId: manifestId) }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy || viewModel.injectOrderId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
                Section {
                    Button(LocalizedStringKey(viewModel.busy ? "mobile_supplier.chrome.sealing" : "mobile_supplier.chrome.seal_manifest")) {
                        Task { await viewModel.seal(manifestId: manifestId) }
                    }
                    .frame(minHeight: 44)
                    .disabled(viewModel.busy)
                }
            }
            if !detail.orders.isEmpty {
                Section("mobile_supplier.chrome.orders") {
                    ForEach(detail.orders) { order in
                        VStack(alignment: .leading) {
                            Text(order.orderId).font(.caption.monospaced())
                            Text(order.status.isEmpty ? order.state : order.status).font(.caption)
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
    }

    static func connectionRestoredNotice() -> String {
        L10n.format("mobile_supplier.chrome.connection_restored_manifest")
    }
}

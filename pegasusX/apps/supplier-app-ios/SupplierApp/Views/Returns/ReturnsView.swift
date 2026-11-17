import SwiftUI

struct ReturnsView: View {
    @State private var viewModel = ReturnsViewModel()

    private let resolutions = ["RETURN_TO_STOCK", "WRITE_OFF"]

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.items.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.returns")
            } else if let error = viewModel.error, viewModel.items.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.items.isEmpty {
                SupplierEmptyView(
                    title: "mobile_supplier.empty.no_open_returns",
                    message: "mobile_supplier.empty.returns_message"
                )
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.items) { row in
                        VStack(alignment: .leading, spacing: 6) {
                            Text(row.productName).font(.headline)
                            Text(L10n.format("mobile_supplier.ui.qty_quantity_reason", "\(row.quantity)", "\(row.reason)"))
                                .font(.subheadline)
                            Text(L10n.format("mobile_supplier.ui.physical_physicalstatus", "\(row.physicalStatus)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                            if !row.driverName.isEmpty {
                                Text(L10n.format("mobile_supplier.ui.driver_drivername", "\(row.driverName)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            if row.receivedQty > 0 {
                                Text(L10n.format("mobile_supplier.ui.scanned_receivedqty", "\(row.receivedQty)"))
                                    .font(.caption2)
                                    .foregroundStyle(SupplierTheme.warning)
                            }

                            if viewModel.resolvingId == row.returnId {
                                Picker("mobile_supplier.chrome.resolution", selection: $vm.resolution) {
                                    ForEach(resolutions, id: \.self) { option in
                                        Text(option.replacingOccurrences(of: "_", with: " ")).tag(option)
                                    }
                                }
                                .pickerStyle(.menu)
                                TextField("supplier_portal.returns.text.notes_optional", text: $vm.notes)
                                    .textInputAutocapitalization(.sentences)
                                HStack {
                                    Button(LocalizedStringKey(viewModel.actionLoading == row.returnId ? "…" : "mobile_supplier.chrome.confirm")) {
                                        Task { await viewModel.resolve(returnId: row.returnId) }
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .frame(minHeight: 44)
                                    .disabled(viewModel.actionLoading == row.returnId)
                                    Button("common.action.cancel") { viewModel.resolvingId = nil }
                                        .buttonStyle(.bordered)
                                        .frame(minHeight: 44)
                                }
                            } else if row.physicalStatus == "RESTOCKED" || row.physicalStatus == "WRITTEN_OFF" {
                                Text("supplier_portal.returns.text.gate_resolved")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            } else {
                                Button("mobile_supplier.ui.dispute_override") {
                                    viewModel.startDispute(returnId: row.returnId)
                                }
                                .buttonStyle(.bordered)
                                .frame(minHeight: 44)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.returns")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }
}

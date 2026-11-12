import SwiftUI

struct ManifestExceptionsView: View {
    @State private var viewModel = ManifestExceptionsViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.rows.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.gate_exceptions")
            } else if let error = viewModel.error, viewModel.rows.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else if viewModel.rows.isEmpty {
                SupplierEmptyView(title: "mobile_supplier.empty.no_manifest_exceptions", message: "mobile_supplier.empty.manifest_exceptions_message")
            } else {
                ResponsiveGridContentWrapper {
                    ForEach(viewModel.rows) { row in
                        NavigationLink {
                            ManifestDetailView(manifestId: row.manifestId)
                        } label: {
                            VStack(alignment: .leading, spacing: 4) {
                                Text(row.reason).font(.headline)
                                Text(L10n.format("mobile_supplier.ui.manifest_prefix", "\(row.manifestId.prefix(8))")).font(.caption)
                                Text(L10n.format("mobile_supplier.ui.order_prefix_attempts_attemptcount", "\(row.orderId.prefix(8))", "\(row.attemptCount)")).font(.caption)
                                if row.escalated {
                                    Text("factory_portal.residual.text.escalated").font(.caption).foregroundStyle(SupplierTheme.destructive)
                                }
                            }
                            .accessibilityElement(children: .combine)
                        }
                    }
                }
            }
        }
        .navigationTitle("factory_portal.manifest_exceptions.text.gate_exceptions")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Toggle("mobile_supplier.chrome.escalated", isOn: $vm.escalatedOnly)
                    .labelsHidden()
                    .accessibilityLabel("mobile_supplier.chrome.escalated")
            }
        }
        .onChange(of: viewModel.escalatedOnly) { _, _ in Task { await viewModel.load() } }
        .task { await viewModel.load() }
    }
}

import SwiftUI

struct RetailerOverridesView: View {
    @State private var viewModel = RetailerOverridesViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Group {
            if viewModel.loading && viewModel.overrides.isEmpty {
                SupplierLoadingView(title: "mobile_supplier.loading.overrides")
            } else if let error = viewModel.error, viewModel.overrides.isEmpty {
                SupplierErrorView(message: error) { Task { await viewModel.load() } }
            } else {
                ResponsiveGridContentWrapper {
                    if let error = viewModel.error {
                        Text(error).font(.caption).foregroundStyle(SupplierTheme.destructive)
                    }
                    if viewModel.overrides.isEmpty {
                        SupplierEmptyView(title: "mobile_supplier.empty.no_overrides", message: "mobile_supplier.empty.overrides_message")
                    } else {
                        ForEach(viewModel.overrides) { row in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text(L10n.format("mobile_supplier.ui.retailer_prefix", "\(row.retailerId.prefix(10))"))
                                    .font(.headline)
                                Text(L10n.format("mobile_supplier.ui.product_prefix_price", "\(row.productId.prefix(10))", "\(row.price)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .accessibilityElement(children: .combine)
                            .swipeActions {
                                Button(role: .destructive) {
                                    Task { await viewModel.deleteOverride(row.overrideId) }
                                } label: {
                                    Text("mobile_supplier.ui.delete")
                                }
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.pricing.retailer_overrides.text.retailer_overrides")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button("mobile_supplier.chrome.add_override", systemImage: "plus") { viewModel.showCreate = true }
                    .labelStyle(.iconOnly)
                    .accessibilityLabel("mobile_supplier.chrome.add_override")
            }
        }
        .sheet(isPresented: $vm.showCreate) {
            NavigationStack {
                Form {
                    TextField("supplier_portal.chargebacks.text.retailer_id", text: $vm.retailerId)
                    TextField("factory_portal.analytics.text.product_id", text: $vm.productId)
                    TextField("mobile_supplier.ui.price_minor_units", text: $vm.price)
                        .keyboardType(.numberPad)
                    if viewModel.previewLoading {
                        Text("supplier_portal.pricing.retailer_overrides.text.calculating_impact_preview")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    } else if let preview = viewModel.preview {
                        Section("mobile_supplier.chrome.impact_preview") {
                            LabeledContent("mobile_supplier.chrome.retailers_on_sku", value: "\(preview.retailersOnSkuCount)")
                            LabeledContent("mobile_supplier.chrome.active_overrides", value: "\(preview.activeOverrideCount)")
                            LabeledContent("mobile_supplier.chrome.catalog_list_price", value: "\(preview.catalogListPrice)")
                            LabeledContent("mobile_supplier.chrome.margin_delta_unit", value: "\(preview.marginDeltaPerUnit) (\(preview.marginEstimateLabel))")
                        }
                    }
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("mobile_supplier.ui.new_override")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.cancel") { viewModel.showCreate = false }
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button(LocalizedStringKey(viewModel.creating ? "mobile_supplier.chrome.creating" : "mobile_supplier.chrome.create")) { Task { await viewModel.createOverride() } }
                            .disabled(viewModel.creating || viewModel.retailerId.isEmpty || viewModel.productId.isEmpty || viewModel.price.isEmpty)
                    }
                }
                .task(id: viewModel.previewKey) {
                    await viewModel.refreshPreview()
                }
            }
        }
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }
}

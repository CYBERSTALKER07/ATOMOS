import SwiftUI

struct RetailerOverridesView: View {
    @State private var overrides: [RetailerPriceOverride] = []
    @State private var loading = true
    @State private var error: String?
    @State private var showCreate = false
    @State private var retailerId = ""
    @State private var productId = ""
    @State private var price = ""
    @State private var creating = false
    @State private var preview: RetailerOverridePreview?
    @State private var previewLoading = false

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading overrides…")
            } else if let error, overrides.isEmpty {
                SupplierErrorView(message: error) { Task { await load() } }
            } else {
                ResponsiveGridContentWrapper {
                    if let error {
                        Text(error).font(.caption).foregroundStyle(.red)
                    }
                    if overrides.isEmpty {
                        SupplierEmptyView(title: "No overrides", message: "Create retailer-specific pricing overrides.")
                    } else {
                        ForEach(overrides) { row in
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingXS) {
                                Text("Retailer \(row.retailerId.prefix(10))…")
                                    .font(.headline)
                                Text("Product \(row.productId.prefix(10))… · \(row.price)")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }
                            .swipeActions {
                                Button(role: .destructive) {
                                    Task { await deleteOverride(row.overrideId) }
                                } label: {
                                    Text("Delete")
                                }
                            }
                        }
                    }
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("Retailer overrides")
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button { showCreate = true } label: { Image(systemName: "plus") }
            }
        }
        .sheet(isPresented: $showCreate) { createSheet }
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    private var createSheet: some View {
        NavigationStack {
            Form {
                TextField("Retailer ID", text: $retailerId)
                TextField("Product ID", text: $productId)
                TextField("Price (minor units)", text: $price)
                    .keyboardType(.numberPad)
                if previewLoading {
                    Text("Calculating impact preview…")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                } else if let preview {
                    Section("Impact preview") {
                        LabeledContent("Retailers on SKU", value: "\(preview.retailersOnSkuCount)")
                        LabeledContent("Active overrides", value: "\(preview.activeOverrideCount)")
                        LabeledContent("Catalog list price", value: "\(preview.catalogListPrice)")
                        LabeledContent("Margin delta / unit", value: "\(preview.marginDeltaPerUnit) (\(preview.marginEstimateLabel))")
                    }
                }
            }
            .navigationTitle("New override")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { showCreate = false }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(creating ? "…" : "Create") { Task { await createOverride() } }
                        .disabled(creating || retailerId.isEmpty || productId.isEmpty || price.isEmpty)
                }
            }
            .task(id: previewKey) {
                await refreshPreview()
            }
        }
    }

    private var previewKey: String {
        "\(retailerId)|\(productId)|\(price)"
    }

    @MainActor
    private func refreshPreview() async {
        guard showCreate else {
            preview = nil
            previewLoading = false
            return
        }
        guard let priceMinor = Int(price), priceMinor > 0, !productId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty else {
            preview = nil
            previewLoading = false
            return
        }
        previewLoading = true
        try? await Task.sleep(for: .milliseconds(400))
        do {
            preview = try await SupplierOperationsService.previewRetailerPriceOverride(
                RetailerOverridePreviewRequest(
                    retailerId: retailerId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : retailerId,
                    productId: productId.trimmingCharacters(in: .whitespacesAndNewlines),
                    skuId: nil,
                    proposedPrice: Int64(priceMinor)
                )
            )
        } catch {
            preview = nil
        }
        previewLoading = false
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            let resp = try await SupplierOperationsService.listRetailerPriceOverrides()
            overrides = resp.overrides
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    private func createOverride() async {
        guard let priceMinor = Int(price) else { return }
        creating = true
        defer { creating = false }
        do {
            _ = try await SupplierOperationsService.createRetailerPriceOverride(
                CreateRetailerPriceOverrideRequest(
                    retailerId: retailerId,
                    productId: productId,
                    price: priceMinor,
                    notes: nil,
                    expiresAt: nil
                )
            )
            showCreate = false
            retailerId = ""
            productId = ""
            price = ""
            preview = nil
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func deleteOverride(_ id: String) async {
        do {
            try await SupplierOperationsService.deleteRetailerPriceOverride(overrideId: id)
            await load(silent: true)
        } catch {
            self.error = error.localizedDescription
        }
    }
}

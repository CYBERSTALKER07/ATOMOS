import SwiftUI

struct CatalogView: View {
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @State private var products: [CatalogProduct] = []
    @State private var draftVU: [String: String] = [:]
    @State private var draftBarcode: [String: String] = [:]
    @State private var loading = true
    @State private var error: String?
    @State private var savingId: String?
    @State private var imageSavingId: String?
    @State private var query = ""
    @State private var showCreate = false

    private var filtered: [CatalogProduct] {
        guard !query.isEmpty else { return products }
        return products.filter { $0.name.localizedCaseInsensitiveContains(query) }
    }

    var body: some View {
        NavigationStack {
            Group {
                if loading && products.isEmpty {
                    SupplierLoadingView(title: "Loading catalog…")
                } else if let error, products.isEmpty {
                    SupplierErrorView(message: error) { Task { await load() } }
                } else if filtered.isEmpty {
                    SupplierEmptyView(
                        title: "No products",
                        message: query.isEmpty
                            ? "Tap + to create a product and set unit VU."
                            : "No matches for \"\(query)\"."
                    )
                } else {
                    CatalogList(
                        products: products,
                        query: query,
                        draftVU: $draftVU,
                        draftBarcode: $draftBarcode,
                        savingId: savingId,
                        imageSavingId: imageSavingId,
                        onSaveUnitVolume: saveUnitVolume,
                        onSaveProductImage: saveProductImage
                    )
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("portal.nav.catalog")
            .searchable(text: $query, prompt: "Product name")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        showCreate = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showCreate) {
                CatalogCreateSheet {
                    Task { await load(silent: true) }
                }
            }
            .task { await load() }
            .refreshable { await load(silent: true) }
            .silentRealtimeRefresh(
                refreshEpoch: realtimeHub.refreshEpoch,
                reconnectEpoch: realtimeHub.reconnectEpoch
            ) { silent in
                Task { await load(silent: silent) }
            }
        }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        do {
            products = try await SupplierService.catalogProducts()
            draftVU = [:]
            draftBarcode = [:]
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
        loading = false
    }

    @MainActor
    private func saveUnitVolume(_ product: CatalogProduct) async {
        let raw = draftVU[product.productId] ?? String(product.unitVolumeVu)
        guard let parsed = Double(raw), parsed > 0 else {
            error = "Unit VU must be a positive number."
            return
        }
        let barcodeRaw = (draftBarcode[product.productId] ?? product.barcode ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let barcode: String?
        if barcodeRaw.isEmpty {
            barcode = nil
        } else {
            guard let normalized = EANBarcode.normalize(barcodeRaw) else {
                error = "Invalid EAN/GTIN barcode."
                return
            }
            barcode = normalized
        }
        savingId = product.productId
        error = nil
        defer { savingId = nil }
        do {
            let updated = try await SupplierService.updateCatalogProduct(
                productId: product.productId,
                request: CatalogProductUpdateRequest(
                    name: product.name,
                    priceMinor: product.priceMinor,
                    currency: product.currency,
                    unit: product.unit,
                    unitVolumeVu: parsed,
                    imageUrl: product.imageUrl,
                    barcode: barcode,
                    isActive: product.isActive,
                    version: product.version
                )
            )
            products = products.map { row in
                row.productId == updated.productId ? updated : row
            }
            draftVU.removeValue(forKey: product.productId)
            draftBarcode.removeValue(forKey: product.productId)
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func saveProductImage(_ product: CatalogProduct, imageData: Data) async {
        imageSavingId = product.productId
        error = nil
        defer { imageSavingId = nil }
        do {
            let imageUrl = try await SupplierService.uploadCatalogImage(data: imageData, ext: "jpg")
            let updated = try await SupplierService.updateCatalogProduct(
                productId: product.productId,
                request: CatalogProductUpdateRequest(
                    name: product.name,
                    priceMinor: product.priceMinor,
                    currency: product.currency,
                    unit: product.unit,
                    unitVolumeVu: product.unitVolumeVu,
                    imageUrl: imageUrl,
                    barcode: product.barcode,
                    isActive: product.isActive,
                    version: product.version
                )
            )
            products = products.map { row in
                row.productId == updated.productId ? updated : row
            }
        } catch {
            self.error = error.localizedDescription
        }
    }
}

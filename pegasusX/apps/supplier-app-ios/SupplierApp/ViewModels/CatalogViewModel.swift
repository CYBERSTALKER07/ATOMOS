import Foundation
import Observation

@Observable
@MainActor
final class CatalogViewModel {
    var products: [CatalogProduct] = []
    var draftVU: [String: String] = [:]
    var draftBarcode: [String: String] = [:]
    var loading = true
    var error: String?
    var savingId: String?
    var imageSavingId: String?
    var query = ""
    var showCreate = false

    private var loadGeneration = 0

    var filtered: [CatalogProduct] {
        guard !query.isEmpty else { return products }
        return products.filter { $0.name.localizedCaseInsensitiveContains(query) }
    }

    func load(silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = !products.isEmpty
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        do {
            let loadedProducts = try await SupplierService.catalogProducts()
            guard generation == loadGeneration else { return }
            products = loadedProducts
            if !silent {
                draftVU = [:]
                draftBarcode = [:]
            }
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.succeed(error: &error)
        } catch {
            guard generation == loadGeneration else { return }
            SupplierScreenLoad.fail(
                silent: silent,
                hasContent: hadContent,
                underlying: error,
                error: &self.error,
                noun: "data"
            )
        }
        if generation == loadGeneration {
            loading = false
        }
    }

    func saveUnitVolume(_ product: CatalogProduct) async {
        let raw = draftVU[product.productId] ?? String(product.unitVolumeVu)
        guard let parsed = Double(raw), parsed > 0 else {
            error = L10n.format("mobile_supplier.chrome.unit_vu_positive")
            return
        }
        let barcodeRaw = (draftBarcode[product.productId] ?? product.barcode ?? "")
            .trimmingCharacters(in: .whitespacesAndNewlines)
        let barcode: String?
        if barcodeRaw.isEmpty {
            barcode = nil
        } else {
            guard let normalized = EANBarcode.normalize(barcodeRaw) else {
                error = L10n.format("mobile_supplier.chrome.invalid_barcode")
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
                ),
                idempotencyKey: SupplierIdempotencyKeys.catalogUpdate(
                    productId: product.productId,
                    priceMinor: product.priceMinor,
                    unitVolumeVu: parsed,
                    barcode: barcode
                )
            )
            products = products.map { row in
                row.productId == updated.productId ? updated : row
            }
            draftVU.removeValue(forKey: product.productId)
            draftBarcode.removeValue(forKey: product.productId)
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !products.isEmpty, error: &self.error)
        }
    }

    func saveProductImage(_ product: CatalogProduct, imageData: Data) async {
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
                ),
                idempotencyKey: SupplierIdempotencyKeys.catalogUpdate(
                    productId: product.productId,
                    priceMinor: product.priceMinor,
                    unitVolumeVu: product.unitVolumeVu,
                    barcode: imageUrl
                )
            )
            products = products.map { row in
                row.productId == updated.productId ? updated : row
            }
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: !products.isEmpty, error: &self.error)
        }
    }
}

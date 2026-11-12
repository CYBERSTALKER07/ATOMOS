import CoreLocation
import Foundation
import Observation
import PhotosUI
import SwiftUI

@Observable
@MainActor
final class CatalogCreateViewModel {
    var name = ""
    var priceMinor = ""
    var unitVu = "1"
    var barcode = ""
    var categories: [CatalogCategory] = []
    var categoryId = ""
    var currency = packCurrency(MarketPackStore.pack)
    var photoItem: PhotosPickerItem?
    var imageData: Data?
    var creating = false
    var error: String?

    func handlePhotoSelection(_ item: PhotosPickerItem?) async {
        guard let item else { return }
        do {
            let loaded = try await item.loadTransferable(type: Data.self)
            imageData = try PhotoLoad.imageData(loaded)
            error = nil
        } catch {
            self.error = error.localizedDescription
        }
    }

    func loadForm() async {
        do {
            let profile = try await SupplierService.profile()
            currency = displayPackCurrency(profile.currency)
            categories = try await SupplierService.catalogCategories(supplierId: profile.supplierId)
            categoryId = categories.first?.categoryId ?? ""
        } catch {
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
        }
    }

    func create() async -> Bool {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty, !categoryId.isEmpty else {
            error = L10n.format("mobile_supplier.chrome.name_category_required")
            PegasusHaptics.error()
            return false
        }
        guard let price = Int64(priceMinor), price >= 0 else {
            error = L10n.format("mobile_supplier.chrome.price_non_negative")
            PegasusHaptics.error()
            return false
        }
        guard let parsed = Double(unitVu), parsed > 0 else {
            error = L10n.format("mobile_supplier.chrome.unit_vu_positive")
            PegasusHaptics.error()
            return false
        }
        let barcodeRaw = barcode.trimmingCharacters(in: .whitespacesAndNewlines)
        let barcodeValue: String?
        if barcodeRaw.isEmpty {
            barcodeValue = nil
        } else {
            guard let normalized = EANBarcode.normalize(barcodeRaw) else {
                error = L10n.format("mobile_supplier.chrome.invalid_barcode")
                PegasusHaptics.error()
                return false
            }
            barcodeValue = normalized
        }
        creating = true
        error = nil
        defer { creating = false }
        do {
            var imageUrl: String?
            if let imageData {
                imageUrl = try await SupplierService.uploadCatalogImage(data: imageData, ext: "jpg")
            }
            let scope = SupplierIdempotencyKeys.supplierScopeId()
            _ = try await SupplierService.createCatalogProduct(
                CatalogProductCreateRequest(
                    categoryId: categoryId,
                    name: trimmedName,
                    description: "",
                    priceMinor: price,
                    currency: currency,
                    unitVolumeVu: parsed,
                    stockQuantity: 0,
                    unit: "UNIT",
                    imageUrl: imageUrl,
                    barcode: barcodeValue
                ),
                idempotencyKey: SupplierIdempotencyKeys.catalogCreate(
                    scopeId: scope,
                    name: trimmedName,
                    priceMinor: price,
                    barcode: barcodeValue
                )
            )
            PegasusHaptics.success()
            return true
        } catch {
            PegasusHaptics.error()
            SupplierActionFailure.record(underlying: error, hasContent: false, error: &self.error)
            return false
        }
    }
}

@Observable
@MainActor
final class CatalogDetailViewModel {
    var product: CatalogProduct?
    var loading = true
    var error: String?
    private var loadGeneration = 0

    func load(productId: String?, silent: Bool = false) async {
        loadGeneration += 1
        let generation = loadGeneration
        let hadContent = product != nil
        SupplierScreenLoad.begin(silent: silent, loading: &loading, error: &error)
        guard let productId, !productId.isEmpty else {
            if generation == loadGeneration {
                error = L10n.format("mobile_supplier.chrome.no_product_selected")
                loading = false
            }
            return
        }
        do {
            let products = try await SupplierService.catalogProducts()
            guard generation == loadGeneration else { return }
            product = products.first { $0.productId == productId }
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
}

@Observable
@MainActor
final class AddressLocationViewModel {
    var query = ""
    var suggestions: [GeocodePrediction] = []
    var error: String?
    var locating = false
    private var searchTask: Task<Void, Never>?

    func scheduleSearch(_ text: String) {
        searchTask?.cancel()
        searchTask = Task { @MainActor in
            try? await Task.sleep(nanoseconds: 280_000_000)
            guard !Task.isCancelled else { return }
            await search(text)
        }
    }

    func cancelSearch() {
        searchTask?.cancel()
        searchTask = nil
    }

    func search(_ text: String) async {
        error = nil
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard trimmed.count >= 3 else {
            suggestions = []
            return
        }
        do {
            suggestions = try await GeocodeService.autocomplete(trimmed)
        } catch {
            suggestions = []
        }
    }

    func pick(placeId: String, fallback: String) async -> AddressLocationValue? {
        do {
            let loc = try await GeocodeService.resolvePlace(placeId)
            return makeValue(loc, fallback: fallback)
        } catch let pickError {
            error = pickError.localizedDescription
            return nil
        }
    }

    func useMyLocation() async -> AddressLocationValue? {
        locating = true
        defer { locating = false }
        let manager = CLLocationManager()
        manager.requestWhenInUseAuthorization()
        guard let target = manager.location?.coordinate,
              target.latitude != 0 || target.longitude != 0 else {
            error = L10n.format("mobile_supplier.chrome.location_unavailable")
            return nil
        }
        do {
            let loc = try await GeocodeService.reverse(lat: target.latitude, lng: target.longitude)
            return makeValue(loc, fallback: query)
        } catch {
            self.error = error.localizedDescription
            return nil
        }
    }

    private func makeValue(_ loc: ResolvedLocation, fallback: String) -> AddressLocationValue {
        let next = AddressLocationValue(
            address: loc.address.isEmpty ? fallback : loc.address,
            lat: loc.lat,
            lng: loc.lng,
            placeId: loc.placeId
        )
        query = next.address
        suggestions = []
        error = nil
        return next
    }
}

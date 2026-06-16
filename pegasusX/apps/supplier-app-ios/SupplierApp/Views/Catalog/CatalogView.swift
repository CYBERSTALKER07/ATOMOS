import PhotosUI
import SwiftUI

struct CatalogView: View {
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
                if loading {
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
                    catalogList
                }
            }
            .background(SupplierTheme.background)
            .navigationTitle("Catalog")
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
        }
    }

    private var catalogList: some View {
        List {
            if let error {
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.red)
            }
            ForEach(filtered) { product in
                catalogRow(product)
            }
        }
        .listStyle(.insetGrouped)
    }

    private func catalogRow(_ product: CatalogProduct) -> some View {
        let vuBinding = Binding(
            get: { draftVU[product.productId] ?? String(product.unitVolumeVu) },
            set: { draftVU[product.productId] = $0 }
        )
        let barcodeBinding = Binding(
            get: { draftBarcode[product.productId] ?? product.barcode ?? "" },
            set: { draftBarcode[product.productId] = $0 }
        )
        let currentVU = draftVU[product.productId] ?? String(product.unitVolumeVu)
        let currentBarcode = draftBarcode[product.productId] ?? product.barcode ?? ""
        let vuDirty = currentVU != String(product.unitVolumeVu)
        let barcodeDirty = currentBarcode != (product.barcode ?? "")
        let dirty = vuDirty || barcodeDirty

        return VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text(product.name)
                .font(.headline)
            Text("\(product.priceMinor.formatted()) \(product.currency) · \(product.unit)")
                .font(.subheadline)
                .foregroundStyle(.secondary)
            if let imageUrl = product.imageUrl, !imageUrl.isEmpty, let url = URL(string: imageUrl) {
                AsyncImage(url: url) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFill()
                            .frame(width: 48, height: 48)
                            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusSM))
                    default:
                        Color.clear
                            .frame(width: 48, height: 48)
                    }
                }
            }
            CatalogProductImageButton(
                product: product,
                isSaving: imageSavingId == product.productId
            ) { data in
                await saveProductImage(product, imageData: data)
            }
            CatalogBarcodeField(value: barcodeBinding, enabled: savingId != product.productId)
            HStack {
                TextField("Unit VU", text: vuBinding)
                    .keyboardType(.decimalPad)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 120)
                Button(dirty ? "Save" : "Saved") {
                    Task { await saveUnitVolume(product) }
                }
                .buttonStyle(.borderedProminent)
                .disabled(!dirty || savingId == product.productId)
            }
        }
        .padding(.vertical, SupplierTheme.spacingXS)
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

private struct CatalogProductImageButton: View {
    let product: CatalogProduct
    let isSaving: Bool
    let onUpload: (Data) async -> Void

    @State private var photoItem: PhotosPickerItem?

    var body: some View {
        PhotosPicker(selection: $photoItem, matching: .images) {
            Text(label)
                .font(.caption)
        }
        .disabled(isSaving)
        .onChange(of: photoItem) { _, item in
            guard let item else { return }
            Task {
                if let data = try? await item.loadTransferable(type: Data.self) {
                    await onUpload(data)
                }
                photoItem = nil
            }
        }
    }

    private var label: String {
        if isSaving { return "Uploading…" }
        if let imageUrl = product.imageUrl, !imageUrl.isEmpty { return "Change image" }
        return "Add image"
    }
}

private struct CatalogCreateSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var priceMinor = ""
    @State private var unitVu = "1"
    @State private var barcode = ""
    @State private var categories: [CatalogCategory] = []
    @State private var categoryId = ""
    @State private var currency = "UZS"
    @State private var photoItem: PhotosPickerItem?
    @State private var imageData: Data?
    @State private var creating = false
    @State private var error: String?

    var onCreated: () -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("Product") {
                    TextField("Name", text: $name)
                    Picker("Category", selection: $categoryId) {
                        ForEach(categories) { category in
                            Text(category.name).tag(category.categoryId)
                        }
                    }
                    TextField("Price (minor units)", text: $priceMinor)
                        .keyboardType(.numberPad)
                    TextField("Unit VU", text: $unitVu)
                        .keyboardType(.decimalPad)
                    CatalogBarcodeField(value: $barcode, enabled: !creating)
                    PhotosPicker(selection: $photoItem, matching: .images) {
                        Label(imageData == nil ? "Add image" : "Image selected", systemImage: "photo")
                    }
                    .onChange(of: photoItem) { _, item in
                        Task {
                            imageData = try? await item?.loadTransferable(type: Data.self)
                        }
                    }
                }
                if let error {
                    Section {
                        Text(error)
                            .foregroundStyle(.red)
                            .font(.caption)
                    }
                }
            }
            .navigationTitle("Add product")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                        .disabled(creating)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(creating ? "Creating…" : "Create") {
                        Task { await create() }
                    }
                    .disabled(creating || categories.isEmpty)
                }
            }
            .task { await loadForm() }
        }
    }

    @MainActor
    private func loadForm() async {
        do {
            let profile = try await SupplierService.profile()
            currency = profile.currency.isEmpty ? "UZS" : profile.currency
            categories = try await SupplierService.catalogCategories(supplierId: profile.supplierId)
            categoryId = categories.first?.categoryId ?? ""
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func create() async {
        let trimmedName = name.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmedName.isEmpty, !categoryId.isEmpty else {
            error = "Name and category are required."
            return
        }
        guard let price = Int64(priceMinor), price >= 0 else {
            error = "Price must be a non-negative integer."
            return
        }
        guard let vu = Double(unitVu), vu > 0 else {
            error = "Unit VU must be positive."
            return
        }
        let barcodeValue: String?
        let trimmedBarcode = barcode.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedBarcode.isEmpty {
            barcodeValue = nil
        } else {
            guard let normalized = EANBarcode.normalize(trimmedBarcode) else {
                error = "Invalid EAN/GTIN barcode."
                return
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
            _ = try await SupplierService.createCatalogProduct(
                CatalogProductCreateRequest(
                    categoryId: categoryId,
                    name: trimmedName,
                    description: "",
                    priceMinor: price,
                    currency: currency,
                    unitVolumeVu: vu,
                    stockQuantity: 0,
                    unit: "UNIT",
                    imageUrl: imageUrl,
                    barcode: barcodeValue
                )
            )
            onCreated()
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
    }
}

private extension Int64 {
    func formatted() -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: self)) ?? "\(self)"
    }
}

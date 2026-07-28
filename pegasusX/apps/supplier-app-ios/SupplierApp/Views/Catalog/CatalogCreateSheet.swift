import SwiftUI
import PhotosUI

struct CatalogCreateSheet: View {
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
        guard let parsed = Double(unitVu), parsed > 0 else {
            error = "Unit VU must be a positive number."
            return
        }
        let vu = parsed
        let barcodeRaw = barcode.trimmingCharacters(in: .whitespacesAndNewlines)
        let barcodeValue: String?
        if barcodeRaw.isEmpty {
            barcodeValue = nil
        } else {
            guard let normalized = EANBarcode.normalize(barcodeRaw) else {
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

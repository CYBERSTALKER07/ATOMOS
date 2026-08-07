import SwiftUI
import PhotosUI

struct CatalogList: View {
    let products: [CatalogProduct]
    let query: String
    @Binding var draftVU: [String: String]
    @Binding var draftBarcode: [String: String]
    let savingId: String?
    let imageSavingId: String?
    let onSaveUnitVolume: (CatalogProduct) async -> Void
    let onSaveProductImage: (CatalogProduct, Data) async -> Void

    private var filtered: [CatalogProduct] {
        guard !query.isEmpty else { return products }
        return products.filter { $0.name.localizedCaseInsensitiveContains(query) }
    }

    var body: some View {
        ResponsiveGridContentWrapper {
            ForEach(filtered) { product in
                catalogRow(product)
            }
        }
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
            NavigationLink {
                CatalogDetailView(productId: product.productId)
            } label: {
                Text(product.name)
                    .font(.headline)
            }
            Text(L10n.format("mobile_supplier.ui.formatted_currency_unit", "\(product.priceMinor.formatted())", "\(product.currency)", "\(product.unit)"))
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
                await onSaveProductImage(product, data)
            }
            CatalogBarcodeField(value: barcodeBinding, enabled: savingId != product.productId)
            HStack {
                TextField("supplier_portal.catalog.components.catalog_table.text.unit_vu", text: vuBinding)
                    .keyboardType(.decimalPad)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 120)
                Button(dirty ? "Save" : "Saved") {
                    Task { await onSaveUnitVolume(product) }
                }
                .buttonStyle(.borderedProminent)
                .disabled(!dirty || savingId == product.productId)
            }
        }
        .padding(.vertical, SupplierTheme.spacingXS)
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

extension Int64 {
    func formatted() -> String {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter.string(from: NSNumber(value: self)) ?? "\(self)"
    }
}

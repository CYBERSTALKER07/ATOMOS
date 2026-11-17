import SwiftUI

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
                CatalogRowView(
                    product: product,
                    draftVU: $draftVU,
                    draftBarcode: $draftBarcode,
                    isSaving: savingId == product.productId,
                    isImageSaving: imageSavingId == product.productId,
                    onSaveUnitVolume: onSaveUnitVolume,
                    onSaveProductImage: onSaveProductImage
                )
            }
        }
    }
}

struct CatalogRowView: View {
    let product: CatalogProduct
    @Binding var draftVU: [String: String]
    @Binding var draftBarcode: [String: String]
    let isSaving: Bool
    let isImageSaving: Bool
    let onSaveUnitVolume: (CatalogProduct) async -> Void
    let onSaveProductImage: (CatalogProduct, Data) async -> Void

    @State private var vuInput = ""
    @State private var barcodeInput = ""

    private var isDirty: Bool {
        let currentVU = draftVU[product.productId] ?? String(product.unitVolumeVu)
        let currentBarcode = draftBarcode[product.productId] ?? product.barcode ?? ""
        return currentVU != String(product.unitVolumeVu) || currentBarcode != (product.barcode ?? "")
    }

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            NavigationLink {
                CatalogDetailView(productId: product.productId)
            } label: {
                Text(product.name)
                    .font(.headline)
                    .frame(minHeight: 44, alignment: .leading)
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
                            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusSM, style: .continuous))
                    default:
                        Color.clear
                            .frame(width: 48, height: 48)
                    }
                }
            }
            CatalogProductImageButton(
                product: product,
                isSaving: isImageSaving
            ) { data in
                await onSaveProductImage(product, data)
            }
            CatalogBarcodeField(value: $barcodeInput, enabled: !isSaving)
            HStack {
                TextField("supplier_portal.catalog.components.catalog_table.text.unit_vu", text: $vuInput)
                    .keyboardType(.decimalPad)
                    .textFieldStyle(.roundedBorder)
                    .frame(maxWidth: 120)
                Button(LocalizedStringKey(isDirty ? "mobile_supplier.chrome.save" : "mobile_supplier.chrome.saved")) {
                    PegasusHaptics.light()
                    Task { await onSaveUnitVolume(product) }
                }
                .buttonStyle(.borderedProminent)
                .frame(minHeight: 44)
                .disabled(!isDirty || isSaving)
            }
        }
        .padding(.vertical, SupplierTheme.spacingXS)
        .onAppear { syncInputsFromModel() }
        .onChange(of: product.unitVolumeVu) { _, _ in syncInputsFromModel() }
        .onChange(of: product.barcode) { _, _ in syncInputsFromModel() }
        .onChange(of: vuInput) { _, newValue in
            draftVU[product.productId] = newValue
        }
        .onChange(of: barcodeInput) { _, newValue in
            draftBarcode[product.productId] = newValue
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(product.name), \(product.priceMinor.formatted()) \(product.currency) per \(product.unit)")
    }

    private func syncInputsFromModel() {
        let nextVU = draftVU[product.productId] ?? String(product.unitVolumeVu)
        if vuInput != nextVU { vuInput = nextVU }
        let nextBarcode = draftBarcode[product.productId] ?? product.barcode ?? ""
        if barcodeInput != nextBarcode { barcodeInput = nextBarcode }
    }
}

extension Int64 {
    private static let decimalFormatter: NumberFormatter = {
        let formatter = NumberFormatter()
        formatter.numberStyle = .decimal
        return formatter
    }()

    func formatted() -> String {
        Self.decimalFormatter.string(from: NSNumber(value: self)) ?? "\(self)"
    }
}

enum PhotoLoad {
    static func imageData(_ data: Data?) throws -> Data {
        guard let data, !data.isEmpty else { throw PhotoLoadError.empty }
        return data
    }
}

enum PhotoLoadError: LocalizedError {
    case empty

    var errorDescription: String? {
        L10n.format("mobile_supplier.chrome.image_unreadable")
    }
}

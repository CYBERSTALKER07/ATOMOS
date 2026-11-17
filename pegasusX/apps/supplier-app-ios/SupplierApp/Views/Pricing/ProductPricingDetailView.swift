import SwiftUI

struct ProductPricingDetailView: View {
    let product: CatalogProduct
    var onUpdated: () -> Void

    @State private var viewModel = ProductPricingDetailViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Form {
            Section("mobile_supplier.chrome.product") {
                Text(product.name).font(.headline)
                Text(product.productId).font(.caption.monospaced()).foregroundStyle(.secondary)
            }
            Section("mobile_supplier.chrome.list_price") {
                TextField(L10n.format("mobile_supplier.ui.price_currency", "\(product.currency)"), text: $vm.priceMajor)
                    .keyboardType(.decimalPad)
                Text("mobile_supplier.ui.set_the_catalog_list_price_retailers_see_before_promotions")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Section("mobile_supplier.chrome.sale") {
                Toggle("mobile_supplier.chrome.on_sale", isOn: $vm.saleEnabled)
                if viewModel.saleEnabled {
                    TextField("supplier_portal.pricing._product_id_.text.discount_bps", text: $vm.saleDiscountBps)
                        .keyboardType(.numberPad)
                    Text("mobile_supplier.ui.100_bps_1_off_list_price_for_this_product")
                        .font(.caption)
                        .foregroundStyle(.secondary)
                }
            }
            if let error = viewModel.error {
                Section { Text(error).foregroundStyle(SupplierTheme.destructive) }
            }
            Section {
                Button(LocalizedStringKey(viewModel.saving ? "mobile_supplier.chrome.saving" : "mobile_supplier.chrome.save_pricing")) {
                    Task { await viewModel.save(product: product, onUpdated: onUpdated) }
                }
                .frame(minHeight: 44)
                .disabled(viewModel.saving)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("portal.nav.pricing")
        .navigationBarTitleDisplayMode(.inline)
        .task { await viewModel.bootstrap(product: product) }
    }

    static func parsePriceMinor(_ text: String) -> Int64? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return nil }
        let normalized = trimmed.replacingOccurrences(of: ",", with: ".")
        if normalized.hasPrefix("-") { return nil }
        let parts = normalized.split(separator: ".", omittingEmptySubsequences: false)
        if parts.count == 1 {
            guard let major = Int64(parts[0]) else { return nil }
            return major * 100
        } else if parts.count == 2 {
            let majorPart = parts[0].isEmpty ? "0" : String(parts[0])
            guard let major = Int64(majorPart) else { return nil }
            let minorStr = String(parts[1])
            if minorStr.isEmpty {
                return major * 100
            } else if minorStr.count == 1 {
                guard let digit = Int64(minorStr) else { return nil }
                return major * 100 + digit * 10
            } else if minorStr.count == 2 {
                guard let digits = Int64(minorStr) else { return nil }
                return major * 100 + digits
            } else {
                guard let firstTwo = Int64(minorStr.prefix(2)) else { return nil }
                let thirdChar = minorStr[minorStr.index(minorStr.startIndex, offsetBy: 2)]
                let roundUp = (thirdChar >= "5" && thirdChar <= "9") ? Int64(1) : Int64(0)
                return major * 100 + firstTwo + roundUp
            }
        }
        return nil
    }
}

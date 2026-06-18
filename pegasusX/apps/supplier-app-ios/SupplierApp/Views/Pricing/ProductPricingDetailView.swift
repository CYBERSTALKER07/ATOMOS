import SwiftUI

struct ProductPricingDetailView: View {
  let product: CatalogProduct
  var onUpdated: () -> Void

  @State private var priceMajor = ""
  @State private var saleDiscountBps = ""
  @State private var saleEnabled = false
  @State private var saving = false
  @State private var error: String?
  @State private var activeSale: SupplierPromotion?

  var body: some View {
    Form {
      Section("Product") {
        Text(product.name).font(.headline)
        Text(product.productId).font(.caption.monospaced()).foregroundStyle(.secondary)
      }
      Section("List price") {
        TextField("Price (\(product.currency))", text: $priceMajor)
          .keyboardType(.decimalPad)
        Text("Set the catalog list price retailers see before promotions.")
          .font(.caption)
          .foregroundStyle(.secondary)
      }
      Section("Sale") {
        Toggle("On sale", isOn: $saleEnabled)
        if saleEnabled {
          TextField("Discount (bps)", text: $saleDiscountBps)
            .keyboardType(.numberPad)
          Text("100 bps = 1% off list price for this product.")
            .font(.caption)
            .foregroundStyle(.secondary)
        }
      }
      if let error {
        Section { Text(error).foregroundStyle(.red) }
      }
      Section {
        Button(saving ? "Saving…" : "Save pricing") { Task { await save() } }
          .disabled(saving)
      }
    }
    .navigationTitle("Pricing")
    .navigationBarTitleDisplayMode(.inline)
    .task { await bootstrap() }
  }

  @MainActor
  private func bootstrap() async {
    priceMajor = String(format: "%.2f", Double(product.priceMinor) / 100.0)
    do {
      let promotions = try await SupplierService.promotions()
      activeSale = promotions.first {
        $0.isActive && $0.scopeType == "PRODUCT" && $0.scopeProductId == product.productId
      }
      if let activeSale {
        saleEnabled = true
        saleDiscountBps = String(activeSale.discountBps)
      }
    } catch {
      // Promotions are optional for pricing detail.
    }
  }

  @MainActor
  private func save() async {
    guard let priceMinor = parsePriceMinor(priceMajor) else {
      error = "Enter a valid list price."
      return
    }
    saving = true
    error = nil
    defer { saving = false }
    do {
      let request = CatalogProductUpdateRequest(
        name: product.name,
        priceMinor: priceMinor,
        currency: product.currency,
        unit: product.unit,
        unitVolumeVu: product.unitVolumeVu,
        imageUrl: product.imageUrl,
        barcode: product.barcode,
        isActive: product.isActive,
        version: product.version
      )
      _ = try await SupplierService.updateCatalogProduct(productId: product.productId, request: request)

      if saleEnabled {
        let bps = Int64(saleDiscountBps) ?? 0
        guard bps > 0 else {
          error = "Sale discount must be greater than zero."
          return
        }
        let promoName = "Sale · \(product.name) · \(product.productId)"
        let upsert = SupplierPromotionUpsertRequest(
          name: promoName,
          description: "Product sale pricing",
          discountBps: bps,
          scopeType: "PRODUCT",
          retailerScope: "ALL",
          scopeProductId: product.productId
        )
        if let activeSale {
          _ = try await SupplierService.updatePromotion(promotionId: activeSale.promotionId, upsert)
        } else {
          _ = try await SupplierService.createPromotion(upsert)
        }
      } else if let activeSale {
        try await SupplierService.deactivatePromotion(promotionId: activeSale.promotionId)
      }

      onUpdated()
    } catch {
      self.error = error.localizedDescription
    }
  }

  private func parsePriceMinor(_ text: String) -> Int64? {
    let normalized = text.replacingOccurrences(of: ",", with: ".").trimmingCharacters(in: .whitespacesAndNewlines)
    guard let value = Double(normalized), value >= 0 else { return nil }
    return Int64((value * 100).rounded())
  }
}

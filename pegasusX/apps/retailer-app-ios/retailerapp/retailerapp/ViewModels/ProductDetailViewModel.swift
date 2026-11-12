import Foundation
import SwiftUI
import Observation

@Observable
@MainActor
final class ProductDetailViewModel {
    var selectedVariant: Variant?
    var quantity: Int = 1
    var addedToCart = false
    var imageScale: Double = 0.95
    var variantAutoOrder: [String: Bool] = [:]
    var variantHasHistory: [String: Bool] = [:]
    var productAutoOrderEnabled: Bool = false
    var productHasHistory: Bool = false
    var pendingVariantTarget: String?
    var pendingProductEnable: Bool = false
    var showHistoryAlert: Bool = false

    private let api = APIClient.shared

    func configure(for product: Product) {
        if selectedVariant == nil || !(product.variants.contains { $0.id == selectedVariant?.id }) {
            selectedVariant = product.defaultVariant
        }
    }

    func selectVariant(_ variant: Variant) {
        Haptics.light()
        withAnimation(AnimationConstants.express) {
            selectedVariant = variant
            quantity = 1
        }
    }

    func addToCart(product: Product, cart: CartManager) {
        guard let variant = selectedVariant, !product.blocksAddToCart else { return }
        cart.add(product: product, variant: variant, quantity: quantity)
        Haptics.success()
        addedToCart.toggle()
        quantity = 1
    }

    func cancelPendingEnable() {
        if pendingProductEnable {
            productAutoOrderEnabled = false
        }
        if let skuId = pendingVariantTarget {
            variantAutoOrder[skuId] = false
        }
        pendingVariantTarget = nil
        pendingProductEnable = false
        showHistoryAlert = false
    }

    func toggleVariantAutoOrder(skuId: String, enabled: Bool, useHistory: Bool) async {
        do {
            var body: [String: Any] = ["auto_order_enabled": enabled]
            if enabled { body["use_history"] = useHistory }
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/variant/\(skuId)",
                body: AnyCodable(body)
            )
            Haptics.success()
        } catch {
            Haptics.error()
            variantAutoOrder[skuId] = !enabled
        }
    }

    func patchProductAutoOrder(productId: String, enabled: Bool, useHistory: Bool) async {
        do {
            var body: [String: Any] = ["auto_order_enabled": enabled]
            if enabled { body["use_history"] = useHistory }
            let _: [String: Bool] = try await api.patch(
                path: "/v1/retailer/settings/auto-order/product/\(productId)",
                body: AnyCodable(body)
            )
            Haptics.success()
        } catch {
            Haptics.error()
            productAutoOrderEnabled = !enabled
        }
    }

    func loadAutoOrderState(for product: Product) async {
        do {
            let s: AutoOrderSettings = try await api.get(path: "/v1/retailer/settings/auto-order")
            if let pov = s.productOverrides.first(where: { $0.productID == product.id }) {
                productAutoOrderEnabled = pov.enabled
                productHasHistory = pov.hasHistory
            }
            for variant in product.variants {
                if let vov = s.variantOverrides.first(where: { $0.skuID == variant.id }) {
                    variantAutoOrder[variant.id] = vov.enabled
                    variantHasHistory[variant.id] = vov.hasHistory
                }
            }
        } catch {
            productAutoOrderEnabled = false
        }
    }

    func confirmProductOrVariantEnable(productId: String, useHistory: Bool) async {
        showHistoryAlert = false
        if let skuId = pendingVariantTarget {
            pendingVariantTarget = nil
            variantAutoOrder[skuId] = true
            await toggleVariantAutoOrder(skuId: skuId, enabled: true, useHistory: useHistory)
        } else if pendingProductEnable {
            pendingProductEnable = false
            await patchProductAutoOrder(productId: productId, enabled: true, useHistory: useHistory)
        }
    }
}

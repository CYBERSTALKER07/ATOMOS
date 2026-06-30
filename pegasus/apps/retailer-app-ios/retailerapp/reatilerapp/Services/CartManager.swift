import Foundation
import SwiftUI

// MARK: - Cart Manager

@Observable
final class CartManager {
    var items: [CartItem] = [] {
        didSet {
            if !isHydratingCache {
                persistToLocalCache()
            }
        }
    }
    var supplierIsActive: Bool = true
    private var lastSyncedSignature: String = ""
    private let cacheKey = "pegasus.retailer.cart.cache"
    private var isHydratingCache = false

    init() {
        isHydratingCache = true
        loadFromLocalCache()
        isHydratingCache = false
    }

    var totalItems: Int {
        items.reduce(0) { $0 + $1.quantity }
    }

    var totalPrice: Double {
        items.reduce(0) { $0 + $1.totalPrice }
    }

    var displayTotal: String {
        "\(Int(totalPrice).formatted())"
    }

    var isEmpty: Bool { items.isEmpty }

    // MARK: - Add to Cart

    func add(product: Product, variant: Variant, quantity: Int = 1) {
        let itemId = "\(product.id)-\(variant.id)"
        if let index = items.firstIndex(where: { $0.id == itemId }) {
            items[index].quantity += quantity
        } else {
            let item = CartItem(
                id: itemId,
                product: product,
                variant: variant,
                quantity: quantity
            )
            items.append(item)
        }
    }

    // MARK: - Remove from Cart

    func remove(itemId: String) {
        items.removeAll { $0.id == itemId }
    }

    // MARK: - Update Quantity

    func updateQuantity(itemId: String, quantity: Int) {
        guard let index = items.firstIndex(where: { $0.id == itemId }) else { return }
        if quantity <= 0 {
            items.remove(at: index)
        } else {
            items[index].quantity = quantity
        }
    }

    // MARK: - Increment / Decrement

    func increment(itemId: String) {
        guard let index = items.firstIndex(where: { $0.id == itemId }) else { return }
        items[index].quantity += 1
    }

    func decrement(itemId: String) {
        guard let index = items.firstIndex(where: { $0.id == itemId }) else { return }
        if items[index].quantity > 1 {
            items[index].quantity -= 1
        } else {
            items.remove(at: index)
        }
    }

    // MARK: - Clear

    func clear() {
        items.removeAll()
    }

    // MARK: - Build Order Payload

    func buildCheckoutPayload(retailerId: String, paymentGateway: String, latitude: Double = 0, longitude: Double = 0) -> UnifiedCheckoutPayload {
        UnifiedCheckoutPayload(
            retailerId: retailerId,
            paymentGateway: paymentGateway,
            latitude: latitude,
            longitude: longitude,
            items: items.map { item in
                UnifiedCheckoutPayload.Item(
                    skuId: item.product.id,
                    quantity: item.quantity,
                    unitPriceUzs: Int64(item.variant.price)
                )
            }
        )
    }
}

// MARK: - Unified Checkout Payload

struct UnifiedCheckoutPayload: Codable {
    let retailerId: String
    let paymentGateway: String
    let latitude: Double
    let longitude: Double
    let items: [Item]

    struct Item: Codable {
        let skuId: String
        let quantity: Int
        let unitPriceUzs: Int64

        enum CodingKeys: String, CodingKey {
            case skuId = "sku_id"
            case quantity
            case unitPriceUzs = "unit_price"
        }
    }

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case paymentGateway = "payment_gateway"
        case latitude, longitude, items
    }
}

// MARK: - Checkout Response

struct CheckoutResponse: Codable {
    let status: String
    let invoiceId: String
    let total: Int64
    let supplierOrders: [SupplierOrderResult]?

    struct SupplierOrderResult: Codable {
        let orderId: String
        let supplierId: String
        let supplierName: String
        let total: Int64
        let itemCount: Int

        enum CodingKeys: String, CodingKey {
            case orderId = "order_id"
            case supplierId = "supplier_id"
            case supplierName = "supplier_name"
            case total = "total"
            case itemCount = "item_count"
        }
    }

    enum CodingKeys: String, CodingKey {
        case status
        case invoiceId = "invoice_id"
        case total = "total"
        case supplierOrders = "supplier_orders"
    }
}

extension CartManager {
    private struct CachedCartLine: Codable {
        let id: String
        let skuId: String
        let supplierId: String
        let quantity: Int
        let unitPrice: Double
        let productName: String
        let variantId: String
    }

    private func persistToLocalCache() {
        let lines = items.map { item in
            CachedCartLine(
                id: item.id,
                skuId: item.variant.id.isEmpty ? item.product.id : item.variant.id,
                supplierId: item.product.supplierID ?? "",
                quantity: item.quantity,
                unitPrice: item.variant.price,
                productName: item.product.name,
                variantId: item.variant.id
            )
        }
        if let data = try? JSONEncoder().encode(lines) {
            UserDefaults.standard.set(data, forKey: cacheKey)
        }
    }

    private func loadFromLocalCache() {
        guard let data = UserDefaults.standard.data(forKey: cacheKey),
              let lines = try? JSONDecoder().decode([CachedCartLine].self, from: data),
              !lines.isEmpty else { return }

        items = lines.map { line in
            let variant = Variant(
                id: line.variantId.isEmpty ? line.skuId : line.variantId,
                size: "Standard",
                pack: "Per unit",
                packCount: 1,
                weightPerUnit: "1 unit",
                price: line.unitPrice
            )
            let product = Product(
                id: line.skuId,
                name: line.productName,
                description: "",
                nutrition: "",
                imageURL: nil,
                variants: [variant],
                supplierID: line.supplierId,
                supplierName: nil,
                supplierCategory: nil,
                categoryID: nil,
                categoryName: nil,
                sellByBlock: false,
                unitsPerBlock: nil,
                price: Int(line.unitPrice),
                availableStock: nil
            )
            return CartItem(
                id: line.id,
                product: product,
                variant: variant,
                quantity: line.quantity
            )
        }
        lastSyncedSignature = signature(for: items)
    }

    private func signature(for cartItems: [CartItem]) -> String {
        cartItems
            .sorted {
                let lhs = $0.variant.id.isEmpty ? $0.product.id : $0.variant.id
                let rhs = $1.variant.id.isEmpty ? $1.product.id : $1.variant.id
                return lhs < rhs
            }
            .map { item in
                let skuID = item.variant.id.isEmpty ? item.product.id : item.variant.id
                let supplierID = item.product.supplierID ?? ""
                return "\(skuID):\(supplierID):\(item.quantity):\(Int(item.variant.price))"
            }
            .joined(separator: "|")
    }

    private func cartItem(from serverItem: CartSyncItem, existing: CartItem?) -> CartItem {
        let quantity = max(1, Int(serverItem.quantity))
        let serverPrice = Double(serverItem.unitPrice)

        if let existing {
            let updatedVariant = Variant(
                id: existing.variant.id,
                size: existing.variant.size,
                pack: existing.variant.pack,
                packCount: existing.variant.packCount,
                weightPerUnit: existing.variant.weightPerUnit,
                price: serverPrice
            )
            let updatedProduct = Product(
                id: existing.product.id,
                name: existing.product.name,
                description: existing.product.description,
                nutrition: existing.product.nutrition,
                imageURL: existing.product.imageURL,
                variants: [updatedVariant],
                supplierID: existing.product.supplierID ?? serverItem.supplierId,
                supplierName: existing.product.supplierName,
                supplierCategory: existing.product.supplierCategory,
                categoryID: existing.product.categoryID,
                categoryName: existing.product.categoryName,
                sellByBlock: existing.product.sellByBlock,
                unitsPerBlock: existing.product.unitsPerBlock,
                price: Int(serverItem.unitPrice),
                availableStock: existing.product.availableStock
            )
            return CartItem(
                id: "\(updatedProduct.id)-\(updatedVariant.id)",
                product: updatedProduct,
                variant: updatedVariant,
                quantity: quantity
            )
        }

        let fallbackVariant = Variant(
            id: serverItem.skuId,
            size: "Standard",
            pack: "Per unit",
            packCount: 1,
            weightPerUnit: "1 unit",
            price: serverPrice
        )
        let fallbackProduct = Product(
            id: serverItem.skuId,
            name: "Item",
            description: "",
            nutrition: "",
            imageURL: nil,
            variants: [fallbackVariant],
            supplierID: serverItem.supplierId,
            supplierName: nil,
            supplierCategory: nil,
            categoryID: nil,
            categoryName: nil,
            sellByBlock: false,
            unitsPerBlock: nil,
            price: Int(serverItem.unitPrice),
            availableStock: nil
        )
        return CartItem(
            id: "\(fallbackProduct.id)-\(fallbackVariant.id)",
            product: fallbackProduct,
            variant: fallbackVariant,
            quantity: quantity
        )
    }

    // MARK: - Server Hydrate
    func hydrateFromServer() async {
        guard AuthManager.shared.currentUser?.id != nil else { return }

        do {
            let response = try await APIClient.shared.getCartSync()
            var existingBySku: [String: CartItem] = [:]
            for item in items {
                existingBySku[item.product.id] = item
                existingBySku[item.variant.id] = item
            }

            let mergedItems = response.items.map { serverItem in
                cartItem(from: serverItem, existing: existingBySku[serverItem.skuId])
            }

            let newSignature = signature(for: mergedItems)
            if newSignature == lastSyncedSignature {
                return
            }

            items = mergedItems
            lastSyncedSignature = newSignature
        } catch {
            print("Failed to hydrate cart from server")
        }
    }

    // MARK: - Server Sync
    func sync() async {
        // only proceed if a user is signed in — we don't actually need the id value here
        guard AuthManager.shared.currentUser?.id != nil else { return }

        let currentSignature = signature(for: items)
        if currentSignature == lastSyncedSignature {
            return
        }

        let cartItems = items.compactMap { item -> CartSyncItem? in
            let skuID = item.variant.id.isEmpty ? item.product.id : item.variant.id
            let supplierID = item.product.supplierID ?? ""
            guard !skuID.isEmpty, !supplierID.isEmpty, item.quantity > 0 else {
                return nil
            }
            return CartSyncItem(
                cartId: nil,
                skuId: skuID,
                supplierId: supplierID,
                quantity: Int64(item.quantity),
                unitPrice: Int64(item.variant.price),
                currency: "UZS"
            )
        }
        let request = CartSyncRequest(items: cartItems)

        do {
            let response = try await APIClient.shared.syncCart(request: request)
            lastSyncedSignature = currentSignature
            print("Cart synced. Items count: \(response.items.count)")
        } catch {
            print("Failed to sync cart")
        }
    }
}

import Foundation
import SwiftUI

// MARK: - Cart Manager

@Observable
final class CartManager {
    var items: [CartItem] = []
    var supplierIsActive: Bool = true
    private var lastSyncedSignature: String = ""

    var totalItems: Int {
        items.reduce(0) { $0 + $1.quantity }
    }

    var totalPrice: Double {
        items.reduce(0) { $0 + $1.totalPrice }
    }

    var quotedSubtotalMinor: Int64?
    var quotedDiscountMinor: Int64 = 0

    var checkoutSubtotal: Double {
        if let quotedSubtotalMinor {
            return Double(quotedSubtotalMinor)
        }
        return totalPrice
    }

    var checkoutDiscount: Double {
        Double(quotedDiscountMinor)
    }

    var checkoutTotal: Double {
        max(0, checkoutSubtotal - checkoutDiscount)
    }

    var displayTotal: String {
        "\(Int(checkoutTotal).formatted())"
    }

    var isEmpty: Bool { items.isEmpty }

    // MARK: - Add to Cart

    func add(product: Product, variant: Variant, quantity: Int = 1) {
        guard !product.blocksAddToCart else { return }
        let itemId = "\(product.id)-\(variant.id)"
        let capped = min(quantity, product.cartMaxQuantity ?? quantity)
        if let index = items.firstIndex(where: { $0.id == itemId }) {
            let next = items[index].quantity + capped
            items[index].quantity = min(next, product.cartMaxQuantity ?? next)
        } else {
            let item = CartItem(
                id: itemId,
                product: product,
                variant: variant,
                quantity: capped
            )
            items.append(item)
        }
        Task { await refreshCheckoutQuote() }
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
            let cap = items[index].product.cartMaxQuantity
            items[index].quantity = cap != nil ? min(quantity, cap!) : quantity
        }
        Task { await refreshCheckoutQuote() }
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
        quotedSubtotalMinor = nil
        quotedDiscountMinor = 0
    }

    func refreshCheckoutQuote() async {
        guard !items.isEmpty else {
            quotedSubtotalMinor = nil
            quotedDiscountMinor = 0
            return
        }

        let grouped = Dictionary(grouping: items) { $0.product.supplierID ?? "" }
            .filter { !$0.key.isEmpty }

        var subtotalMinor: Int64 = 0
        var discountMinor: Int64 = 0

        do {
            for (supplierID, supplierItems) in grouped {
                let lines = supplierItems.map { item in
                    CheckoutQuoteLine(
                        productID: item.variant.id.isEmpty ? item.product.id : item.variant.id,
                        quantity: Int64(item.quantity),
                        unitPriceMinor: Int64(item.variant.price)
                    )
                }
                let quote = try await APIClient.shared.checkoutQuote(
                    supplierID: supplierID,
                    lines: lines
                )
                subtotalMinor += quote.subtotalMinor
                discountMinor += quote.discountMinor
            }
            quotedSubtotalMinor = subtotalMinor
            quotedDiscountMinor = discountMinor
        } catch {
            quotedSubtotalMinor = nil
            quotedDiscountMinor = 0
        }
    }

    // MARK: - Build Order Payload

    func buildCheckoutPayload(
        retailerId: String,
        paymentGateway: String,
        latitude: Double = 0,
        longitude: Double = 0,
        deliveryMode: String? = nil,
        requestedDeliveryDate: String? = nil,
        deliveryPriority: String? = nil,
        checkoutPolicyToken: String? = nil,
        currency: String? = nil
    ) -> UnifiedCheckoutPayload {
        UnifiedCheckoutPayload(
            retailerId: retailerId,
            paymentGateway: paymentGateway,
            latitude: latitude,
            longitude: longitude,
            items: items.map { item in
                UnifiedCheckoutPayload.Item(
                    skuId: item.variant.id.isEmpty ? item.product.id : item.variant.id,
                    quantity: item.quantity,
                    unitPriceUzs: Int64(item.variant.price)
                )
            },
            deliveryMode: deliveryMode,
            requestedDeliveryDate: requestedDeliveryDate,
            deliveryPriority: deliveryPriority,
            checkoutPolicyToken: checkoutPolicyToken,
            currency: currency
        )
    }

    func applyPreviewCaps(_ preview: CheckoutPreviewResponse) {
        let caps = preview.orderableQuantities ?? preview.maxQuantities
        guard let maxQuantities = caps else { return }
        for index in items.indices {
            let rejectPolicy = preview.defaultOutOfStockPolicy?.uppercased() != "ACCEPT_BACKORDER"
            if !rejectPolicy || items[index].product.acceptsBackorder { continue }
            let sku = items[index].variant.id.isEmpty ? items[index].product.id : items[index].variant.id
            guard let cap = maxQuantities[sku] else { continue }
            if items[index].quantity > Int(cap) {
                items[index].quantity = Int(cap)
            }
        }
    }

    func maxQuantity(for item: CartItem, preview: CheckoutPreviewResponse?) -> Int {
        let sku = item.variant.id.isEmpty ? item.product.id : item.variant.id
        let rejectPolicy = preview?.defaultOutOfStockPolicy?.uppercased() != "ACCEPT_BACKORDER"
        if rejectPolicy && !item.product.acceptsBackorder, let preview {
            let caps = preview.orderableQuantities ?? preview.maxQuantities
            if let cap = caps?[sku] {
                return max(1, Int(cap))
            }
        }
        if let cartCap = item.product.cartMaxQuantity {
            if item.product.acceptsBackorder, let lineMax = preview?.orderLineMaxQuantity {
                return max(1, min(cartCap, Int(lineMax)))
            }
            return max(1, cartCap)
        }
        if let lineMax = preview?.orderLineMaxQuantity {
            return max(1, Int(lineMax))
        }
        return 99
    }
}

// MARK: - Unified Checkout Payload

struct CheckoutQuoteLine: Encodable {
    let productID: String
    let quantity: Int64
    let unitPriceMinor: Int64
    let currency: String

    enum CodingKeys: String, CodingKey {
        case productID = "product_id"
        case quantity
        case unitPriceMinor = "unit_price_minor"
        case currency
    }

    init(productID: String, quantity: Int64, unitPriceMinor: Int64, currency: String = packCurrency(MarketPackStore.pack)) {
        self.productID = productID
        self.quantity = quantity
        self.unitPriceMinor = unitPriceMinor
        self.currency = currency
    }
}

struct CheckoutQuoteRequest: Encodable {
    let supplierID: String
    let lines: [CheckoutQuoteLine]

    enum CodingKeys: String, CodingKey {
        case supplierID = "supplier_id"
        case lines
    }
}

struct CheckoutQuoteResponse: Decodable {
    let supplierID: String
    let subtotalMinor: Int64
    let discountMinor: Int64
    let totalMinor: Int64
    let currency: String

    enum CodingKeys: String, CodingKey {
        case supplierID = "supplier_id"
        case subtotalMinor = "subtotal_minor"
        case discountMinor = "discount_minor"
        case totalMinor = "total_minor"
        case currency
    }
}

struct UnifiedCheckoutPayload: Codable {
    let retailerId: String
    let paymentGateway: String
    let latitude: Double
    let longitude: Double
    let items: [Item]
    let deliveryMode: String?
    let requestedDeliveryDate: String?
    let deliveryPriority: String?
    let checkoutPolicyToken: String?
    let currency: String?

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
        case deliveryMode = "delivery_mode"
        case requestedDeliveryDate = "requested_delivery_date"
        case deliveryPriority = "delivery_priority"
        case checkoutPolicyToken = "checkout_policy_token"
        case currency
    }
}

struct OrderCurrencyOptions: Codable {
    let enabled: Bool
    let operatingCurrency: String
    let allowlist: [String]

    enum CodingKeys: String, CodingKey {
        case enabled
        case operatingCurrency = "operating_currency"
        case allowlist
    }
}

// MARK: - Checkout Response

struct StockWarning: Codable, Hashable {
    let sku: String
    let requested: Int64
    let available: Int64
    let backorderQty: Int64
    let acceptsBackorder: Bool

    enum CodingKeys: String, CodingKey {
        case sku, requested, available
        case backorderQty = "backorder_qty"
        case acceptsBackorder = "accepts_backorder"
    }
}

struct CheckoutResponse: Codable {
    let status: String
    let invoiceId: String
    let total: Int64
    let supplierOrders: [SupplierOrderResult]?
    let backorderedItemCount: Int?
    let backorderOrderId: String?
    let stockWarnings: [StockWarning]?

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
        case backorderedItemCount = "backordered_item_count"
        case backorderOrderId = "backorder_order_id"
        case stockWarnings = "stock_warnings"
    }
}

extension CartManager {
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
                availableStock: existing.product.availableStock,
                isOutOfStockFlag: existing.product.isOutOfStockFlag,
                acceptsBackorder: existing.product.acceptsBackorder
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
            await refreshCheckoutQuote()
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
                currency: packCurrency(MarketPackStore.pack)
            )
        }
        let request = CartSyncRequest(items: cartItems)

        do {
            let response = try await APIClient.shared.syncCart(request: request)
            lastSyncedSignature = currentSignature
            await refreshCheckoutQuote()
            print("Cart synced. Items count: \(response.items.count)")
        } catch {
            print("Failed to sync cart")
        }
    }
}

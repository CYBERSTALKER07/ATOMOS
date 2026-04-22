import Foundation
import SwiftUI

// MARK: - Cart Manager

@Observable
final class CartManager {
    var items: [CartItem] = []
    var supplierIsActive: Bool = true

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

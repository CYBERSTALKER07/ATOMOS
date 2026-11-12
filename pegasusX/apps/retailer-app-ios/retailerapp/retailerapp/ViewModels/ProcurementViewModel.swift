import Foundation
import SwiftData
import SwiftUI
import Observation

struct ProcurementOrderRequest: Codable {
    let retailerId: String
    let items: [Item]

    struct Item: Codable {
        let productId: String
        let quantity: Int

        enum CodingKeys: String, CodingKey {
            case productId = "product_id"
            case quantity
        }
    }

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case items
    }
}

struct ProcurementOrderResponse: Codable {
    let status: String
    let orderId: String
    let total: Int64?

    enum CodingKeys: String, CodingKey {
        case status
        case orderId = "order_id"
        case total = "total"
    }
}

@Observable
@MainActor
final class ProcurementViewModel {
    var forecasts: [DemandForecast] = []
    var selectedItems: Set<String> = []
    var quantities: [String: Int] = [:]
    var isSubmitting = false
    var showSuccess = false
    var showError = false
    var errorMessage = ""
    var isLoading = false
    var products: [Product] = []

    private let api = APIClient.shared

    var selectedTotalUnits: Int {
        selectedItems.reduce(0) { $0 + (quantities[$1] ?? 0) }
    }

    func toggleSelectAll() {
        Haptics.light()
        withAnimation(AnimationConstants.express) {
            if selectedItems.count == forecasts.count {
                selectedItems.removeAll()
            } else {
                selectedItems = Set(forecasts.map(\.id))
                for f in forecasts {
                    quantities[f.id] = f.predictedQuantity
                }
            }
        }
    }

    func toggleSelection(for forecast: DemandForecast) {
        Haptics.light()
        withAnimation(AnimationConstants.express) {
            if selectedItems.contains(forecast.id) {
                selectedItems.remove(forecast.id)
            } else {
                selectedItems.insert(forecast.id)
                quantities[forecast.id] = forecast.predictedQuantity
            }
        }
    }

    func updateQuantity(for forecastId: String, quantity: Int) {
        Haptics.selection()
        quantities[forecastId] = quantity
    }

    func acknowledgeSuccess() {
        selectedItems.removeAll()
        quantities.removeAll()
    }

    func addToCart(cart: CartManager) {
        for forecast in forecasts where selectedItems.contains(forecast.id) {
            let qty = quantities[forecast.id] ?? forecast.predictedQuantity
            if let product = products.first(where: { $0.id == forecast.productId }),
               let variant = product.defaultVariant {
                cart.add(product: product, variant: variant, quantity: qty)
            }
        }
        Haptics.success()
        withAnimation(AnimationConstants.fluid) {
            selectedItems.removeAll()
        }
    }

    func createOrder(modelContext: ModelContext, refreshCenter: RetailerRefreshCenter) async {
        isSubmitting = true
        let rid = AuthManager.shared.currentUser?.id ?? ""
        let orderItems = forecasts.filter { selectedItems.contains($0.id) }.map {
            ProcurementOrderRequest.Item(
                productId: $0.productId,
                quantity: quantities[$0.id] ?? $0.predictedQuantity
            )
        }
        let body = ProcurementOrderRequest(retailerId: rid, items: orderItems)
        let idempotencyKey = RetailerIdempotency.orderCreate(items: orderItems)
        do {
            let _: ProcurementOrderResponse = try await api.post(
                path: "/v1/order/create",
                body: body,
                headers: ["Idempotency-Key": idempotencyKey]
            )
            isSubmitting = false
            Haptics.success()
            refreshCenter.trigger()
            showSuccess = true
        } catch let apiError as APIError {
            isSubmitting = false
            Haptics.error()
            errorMessage = RetailerErrorSupport.message(
                for: apiError,
                restricted: "Procurement order access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry procurement order.",
                fallback: "Procurement order could not be submitted. Please try again."
            )
            showError = true
        } catch {
            queuePendingProcurementOrder(body: body, idempotencyKey: idempotencyKey, modelContext: modelContext)
            isSubmitting = false
            Haptics.warning()
            errorMessage = RetailerErrorSupport.retryQueuedMessage(
                for: error,
                fallback: "Saved for retry. Procurement submission is degraded."
            )
            showError = true
        }
    }

    private func queuePendingProcurementOrder(
        body: ProcurementOrderRequest,
        idempotencyKey: String,
        modelContext: ModelContext
    ) {
        guard let data = try? JSONEncoder().encode(body) else { return }
        let pending = PendingOrder(
            payloadJson: String(data: data, encoding: .utf8) ?? "",
            endpoint: "/v1/order/create",
            method: "POST",
            idempotencyKey: idempotencyKey
        )
        modelContext.insert(pending)
        try? modelContext.save()
    }

    func loadPredictions() async {
        isLoading = true
        do {
            let p: [Product] = try await api.get(path: "/v1/catalog/products")
            products = p
        } catch {
            products = []
        }
        forecasts = []
        isLoading = false
    }
}

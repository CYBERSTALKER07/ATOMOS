import Foundation
import SwiftData
import SwiftUI
import Observation

struct ProcurementOrderRequest: Codable {
    let retailerId: String
    let items: [Item]
    let autoConfirmOptIn: Bool
    let paymentMethod: String?
    let paymentGateway: String?
    let clientOrderId: String?

    struct Item: Codable {
        let productId: String
        let quantity: Int
        let mustHave: Bool
        let allowSubstitutes: Bool

        enum CodingKeys: String, CodingKey {
            case productId = "product_id"
            case quantity
            case mustHave = "must_have"
            case allowSubstitutes = "allow_substitutes"
        }

        init(
            productId: String,
            quantity: Int,
            mustHave: Bool = false,
            allowSubstitutes: Bool = false
        ) {
            self.productId = productId
            self.quantity = quantity
            self.mustHave = mustHave
            self.allowSubstitutes = allowSubstitutes
        }

        init(from decoder: Decoder) throws {
            let c = try decoder.container(keyedBy: CodingKeys.self)
            self.productId = try c.decode(String.self, forKey: .productId)
            self.quantity = try c.decode(Int.self, forKey: .quantity)
            self.mustHave = try c.decodeIfPresent(Bool.self, forKey: .mustHave) ?? false
            self.allowSubstitutes = try c.decodeIfPresent(Bool.self, forKey: .allowSubstitutes) ?? false
        }
    }

    enum CodingKeys: String, CodingKey {
        case retailerId = "retailer_id"
        case items
        case autoConfirmOptIn = "auto_confirm_opt_in"
        case paymentMethod = "payment_method"
        case paymentGateway = "payment_gateway"
        case clientOrderId = "client_order_id"
    }

    init(
        retailerId: String,
        items: [Item],
        autoConfirmOptIn: Bool = false,
        paymentMethod: String? = nil,
        paymentGateway: String? = nil,
        clientOrderId: String? = nil
    ) {
        self.retailerId = retailerId
        self.items = items
        self.autoConfirmOptIn = autoConfirmOptIn
        self.paymentMethod = paymentMethod
        self.paymentGateway = paymentGateway
        self.clientOrderId = clientOrderId
    }

    init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.retailerId = try c.decode(String.self, forKey: .retailerId)
        self.items = try c.decode([Item].self, forKey: .items)
        self.autoConfirmOptIn = try c.decodeIfPresent(Bool.self, forKey: .autoConfirmOptIn) ?? false
        self.paymentMethod = try c.decodeIfPresent(String.self, forKey: .paymentMethod)
        self.paymentGateway = try c.decodeIfPresent(String.self, forKey: .paymentGateway)
        self.clientOrderId = try c.decodeIfPresent(String.self, forKey: .clientOrderId)
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
    var mustHaveByForecastId: [String: Bool] = [:]
    var allowSubstitutesByForecastId: [String: Bool] = [:]
    var forecastProductIdById: [String: String] = [:]
    var autoConfirmOptIn = false
    var packSuggestions: [PackMultipleSuggestion] = []
    var queuedForSyncMessage: String?
    var pendingQueueNotice: PendingOrderStatusNotice?
    var ownerApprovalRequired: OwnerApprovalRequiredDetail?
    var isSubmitting = false
    var showSuccess = false
    var showError = false
    var errorMessage = ""
    var isLoading = false
    var products: [Product] = []

    private let api: APIClient
    private let orderSender: ((ProcurementOrderRequest, String) async throws -> ProcurementOrderResponse)?

    init(
        api: APIClient? = nil,
        orderSender: ((ProcurementOrderRequest, String) async throws -> ProcurementOrderResponse)? = nil
    ) {
        self.api = api ?? .shared
        self.orderSender = orderSender
    }

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
                    forecastProductIdById[f.id] = f.productId
                }
            }
        }
    }

    func toggleSelection(for forecast: DemandForecast) {
        Haptics.light()
        withAnimation(AnimationConstants.express) {
            forecastProductIdById[forecast.id] = forecast.productId
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

    func toggleMustHave(for forecastId: String) {
        mustHaveByForecastId[forecastId] = !(mustHaveByForecastId[forecastId] ?? false)
    }

    func toggleAllowSubstitutes(for forecastId: String) {
        allowSubstitutesByForecastId[forecastId] = !(allowSubstitutesByForecastId[forecastId] ?? false)
    }

    func effectivePackSize(for productId: String) -> Int {
        guard let product = products.first(where: { $0.id == productId }) else { return 1 }
        return RetailerPackHelper.effectivePackSize(product: product, variant: product.defaultVariant)
    }

    func snapQuantityToPackMultiple(for forecastId: String, mode: PackSnapMode = .nearest) {
        let forecast = forecasts.first(where: { $0.id == forecastId })
        let productId = forecast?.productId ?? forecastProductIdById[forecastId] ?? forecastId
        let packSize = effectivePackSize(for: productId)
        let currentQty = quantities[forecastId] ?? forecast?.predictedQuantity ?? 1
        let snapped = RetailerPackHelper.snapToPackMultiple(currentQty, packSize: packSize, mode: mode)
        if snapped > 0 {
            updateQuantity(for: forecastId, quantity: snapped)
        }
        packSuggestions.removeAll { $0.skuId == productId || $0.skuId == forecastId }
        if packSuggestions.isEmpty {
            errorMessage = ""
        }
    }

    func applyPackSuggestion(skuId: String, suggestedQuantity: Int) {
        let matchedId = forecasts.first(where: { $0.productId == skuId || $0.id == skuId })?.id
            ?? forecastProductIdById.first(where: { $0.value == skuId || $0.key == skuId })?.key
            ?? skuId
        if suggestedQuantity > 0 {
            updateQuantity(for: matchedId, quantity: suggestedQuantity)
        }
        packSuggestions.removeAll { $0.skuId == skuId || $0.skuId == matchedId }
        if packSuggestions.isEmpty {
            errorMessage = ""
        }
    }

    func acknowledgeSuccess() {
        selectedItems.removeAll()
        quantities.removeAll()
        mustHaveByForecastId.removeAll()
        allowSubstitutesByForecastId.removeAll()
        packSuggestions.removeAll()
    }

    func addToCart(cart: CartManager) {
        for forecast in forecasts where selectedItems.contains(forecast.id) {
            let qty = quantities[forecast.id] ?? forecast.predictedQuantity
            if let product = products.first(where: { $0.id == forecast.productId }),
               let variant = product.defaultVariant {
                cart.add(
                    product: product,
                    variant: variant,
                    quantity: qty,
                    mustHave: mustHaveByForecastId[forecast.id] ?? false,
                    allowSubstitutes: allowSubstitutesByForecastId[forecast.id] ?? false
                )
            }
        }
        Haptics.success()
        withAnimation(AnimationConstants.fluid) {
            selectedItems.removeAll()
        }
    }

    func retryWithFallbackMethod(
        _ method: String,
        modelContext: ModelContext,
        refreshCenter: RetailerRefreshCenter
    ) async {
        let normalized = method.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        guard !normalized.isEmpty else { return }
        if let rows = try? modelContext.fetch(FetchDescriptor<PendingOrder>()) {
            for row in rows where row.status == PendingOrder.statusAwaitingOwnerApproval && row.endpoint == "/v1/order/create" {
                modelContext.delete(row)
            }
            try? modelContext.save()
        }
        await createOrder(modelContext: modelContext, refreshCenter: refreshCenter, paymentMethodOverride: normalized)
    }

    func createOrder(
        modelContext: ModelContext,
        refreshCenter: RetailerRefreshCenter,
        paymentMethodOverride: String? = nil
    ) async {
        guard !selectedItems.isEmpty, !isSubmitting else { return }
        isSubmitting = true
        packSuggestions = []
        queuedForSyncMessage = nil
        pendingQueueNotice = nil
        ownerApprovalRequired = nil
        errorMessage = ""
        showError = false

        let rid = AuthManager.shared.currentUser?.id ?? ""
        let selectedForecasts = forecasts.filter { selectedItems.contains($0.id) }
        let orderItems: [ProcurementOrderRequest.Item]
        if !selectedForecasts.isEmpty {
            orderItems = selectedForecasts.map { forecast in
                ProcurementOrderRequest.Item(
                    productId: forecast.productId,
                    quantity: quantities[forecast.id] ?? forecast.predictedQuantity,
                    mustHave: mustHaveByForecastId[forecast.id] ?? false,
                    allowSubstitutes: allowSubstitutesByForecastId[forecast.id] ?? false
                )
            }
        } else {
            orderItems = selectedItems.sorted().map { id in
                ProcurementOrderRequest.Item(
                    productId: forecastProductIdById[id] ?? id,
                    quantity: quantities[id] ?? 1,
                    mustHave: mustHaveByForecastId[id] ?? false,
                    allowSubstitutes: allowSubstitutesByForecastId[id] ?? false
                )
            }
        }

        let fallbackGateway: String? = {
            switch paymentMethodOverride {
            case "CASH": return "CASH"
            case "CARD": return "GLOBAL_PAY"
            default: return nil
            }
        }()
        let body = ProcurementOrderRequest(
            retailerId: rid,
            items: orderItems,
            autoConfirmOptIn: autoConfirmOptIn,
            paymentMethod: paymentMethodOverride,
            paymentGateway: fallbackGateway
        )
        let baseKey = RetailerIdempotency.orderCreate(
            items: orderItems,
            autoConfirmOptIn: autoConfirmOptIn
        )
        let idempotencyKey = paymentMethodOverride.map { "\(baseKey):fallback:\($0)" } ?? baseKey
        do {
            if let orderSender {
                _ = try await orderSender(body, idempotencyKey)
            } else {
                let _: ProcurementOrderResponse = try await api.post(
                    path: "/v1/order/create",
                    body: body,
                    headers: ["Idempotency-Key": idempotencyKey]
                )
            }
            isSubmitting = false
            ownerApprovalRequired = nil
            Haptics.success()
            refreshCenter.trigger()
            showSuccess = true
        } catch {
            isSubmitting = false
            let skuNames = Dictionary(
                uniqueKeysWithValues: forecasts.map { ($0.productId, $0.productName) }
            )
            let parsedPack = RetailerOrderContractHelper.parsePackMultipleSuggestions(
                from: error,
                skuNamesBySku: skuNames
            )
            if !parsedPack.isEmpty {
                packSuggestions = parsedPack
                Haptics.warning()
                errorMessage = "Adjust quantities to full pack multiples."
                return
            }

            if RetailerErrorSupport.classify(error) == .restricted {
                Haptics.error()
                errorMessage = RetailerErrorSupport.message(
                    for: error,
                    restricted: "Procurement order access is restricted for this account.",
                    offline: "Offline mode active. Reconnect and retry procurement order.",
                    fallback: "Procurement order could not be submitted. Please try again."
                )
                showError = true
                return
            }

            let ownerApproval = RetailerW2ErrorParser.parseOwnerApprovalRequired(from: error)
            let assessment = RetailerOrderContractHelper.assessPendingOrderError(error)
            switch assessment.disposition {
            case .parkAwaitingOwnerApproval, .parkRequestInProgress, .parkRateLimited, .parkAuthRequired, .retryTransient:
                let message = ownerApproval?.message ?? assessment.userMessage
                queuePendingProcurementOrder(
                    body: body,
                    idempotencyKey: idempotencyKey,
                    status: assessment.queueStatus,
                    lastError: message,
                    modelContext: modelContext
                )
                let notice = PendingOrderStatusNotice(
                    endpoint: "/v1/order/create",
                    idempotencyKey: idempotencyKey,
                    status: assessment.queueStatus,
                    message: message,
                    disposition: assessment.disposition
                )
                pendingQueueNotice = notice
                ownerApprovalRequired = ownerApproval
                PendingOrderStatusCenter.shared.publish(notice)
                queuedForSyncMessage = message
                Haptics.warning()
                errorMessage = message
                showError = (ownerApproval == nil)
            case .purgeRejectedValidation, .purgeIdempotentDuplicate:
                Haptics.error()
                errorMessage = assessment.userMessage
                showError = true
            }
        }
    }

    private func queuePendingProcurementOrder(
        body: ProcurementOrderRequest,
        idempotencyKey: String,
        status: String = PendingOrder.statusPending,
        lastError: String? = nil,
        modelContext: ModelContext
    ) {
        guard let data = try? JSONEncoder().encode(body) else { return }
        let pending = PendingOrder(
            payloadJson: String(data: data, encoding: .utf8) ?? "",
            endpoint: "/v1/order/create",
            method: "POST",
            idempotencyKey: idempotencyKey,
            status: status
        )
        pending.lastError = lastError
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

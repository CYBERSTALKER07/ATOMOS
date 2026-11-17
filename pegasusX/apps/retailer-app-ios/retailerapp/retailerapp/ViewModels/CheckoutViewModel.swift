import Foundation
import SwiftData
import SwiftUI
import Observation

struct CheckoutPaymentOption: Identifiable, Hashable {
    let id: String
    let label: String
    let isToken: Bool
}

@Observable
@MainActor
final class CheckoutViewModel {
    var selectedPaymentId = "GlobalPay"
    var paymentOptions: [CheckoutPaymentOption] = [
        CheckoutPaymentOption(id: "Cash", label: "Cash on Delivery", isToken: false),
        CheckoutPaymentOption(id: "GlobalPay", label: "GlobalPay", isToken: false)
    ]
    var showPaymentPicker = false
    var checkoutSessionId = UUID().uuidString
    var isSubmitting = false
    var showSuccess = false
    var showError = false
    var errorMessage = ""
    var oosItems: [String] = []
    var stockWarnings: [StockWarning] = []
    var showSupplierClosedWarning = false
    var preview: CheckoutPreviewResponse?
    var previewLoading = false
    var deliveryMode = "STANDARD"
    var deliveryDate = ""
    var expressPriority = false
    var catalogCodes: [String] = []
    var packDisplayCurrency = packCurrency(MarketPackStore.pack)
    var showBackorderConfirm = false
    var pendingBackorderPreview: CheckoutPreviewResponse?
    var skipBackorderConfirm = false
    var checkoutPolicyToken: String?
    var autoConfirmOptIn = false
    var packSuggestions: [PackMultipleSuggestion] = []
    var pendingQueueNotice: PendingOrderStatusNotice?
    var ownerApprovalRequired: OwnerApprovalRequiredDetail?

    private let api: APIClient
    private let previewFetcher: ((UnifiedCheckoutPayload) async throws -> CheckoutPreviewResponse)?
    private let checkoutSender: ((UnifiedCheckoutPayload, String) async throws -> CheckoutResponse)?
    private static let isoFormatter = ISO8601DateFormatter()

    init(
        api: APIClient? = nil,
        previewFetcher: ((UnifiedCheckoutPayload) async throws -> CheckoutPreviewResponse)? = nil,
        checkoutSender: ((UnifiedCheckoutPayload, String) async throws -> CheckoutResponse)? = nil
    ) {
        self.api = api ?? .shared
        self.previewFetcher = previewFetcher
        self.checkoutSender = checkoutSender
    }

    var selectedPaymentLabel: String {
        paymentOptions.first(where: { $0.id == selectedPaymentId })?.label ?? "GlobalPay"
    }

    var scheduledMinDate: String {
        let lead = Int(preview?.preorderMinLeadDays ?? 3)
        return Calendar.current.date(byAdding: .day, value: max(1, lead), to: Date())
            .map { Self.isoFormatter.string(from: $0).prefix(10) }
            .map(String.init) ?? ""
    }

    var scheduledMaxDate: String? {
        guard let maxLead = preview?.preorderMaxLeadDays, maxLead > 0 else { return nil }
        return Calendar.current.date(byAdding: .day, value: Int(maxLead), to: Date())
            .map { Self.isoFormatter.string(from: $0).prefix(10) }
            .map(String.init)
    }

    /// Map UI labels to backend gateway codes expected by /v1/checkout/unified
    func gatewayCode(for methodId: String) -> String {
        switch methodId {
        case "GlobalPay":
            return catalogCodes.contains("GLOBAL_PAY") || catalogCodes.isEmpty
                ? "GLOBAL_PAY"
                : (catalogCodes.contains("CASH") ? "CASH" : catalogCodes[0])
        case "Cash":
            return "CASH"
        default:
            if catalogCodes.contains(methodId.uppercased()) { return methodId.uppercased() }
            return catalogCodes.contains("CASH") ? "CASH" : (catalogCodes.first ?? "CASH")
        }
    }

    func skuFor(_ item: CartItem) -> String {
        item.variant.id.isEmpty ? item.product.id : item.variant.id
    }

    func isOos(_ item: CartItem) -> Bool {
        let sku = skuFor(item)
        if oosItems.contains(sku) { return true }
        if let shortfall = preview?.shortfall?[sku], shortfall > 0 { return true }
        return false
    }

    func selectPaymentOption(_ option: CheckoutPaymentOption) {
        Haptics.selection()
        withAnimation(AnimationConstants.express) {
            selectedPaymentId = option.id
        }
        showPaymentPicker = false
    }

    func selectDeliveryMode(_ mode: String) {
        Haptics.selection()
        deliveryMode = mode
    }

    func buildPayload(
        cart: CartManager,
        retailerId: String,
        gateway: String,
        paymentMethod: String? = nil,
        clientOrderId: String? = nil
    ) -> UnifiedCheckoutPayload {
        let requestedDate: String? = {
            guard deliveryMode == "SCHEDULED", !deliveryDate.isEmpty else { return nil }
            return "\(deliveryDate)T12:00:00+05:00"
        }()
        return cart.buildCheckoutPayload(
            retailerId: retailerId,
            paymentGateway: gateway,
            deliveryMode: deliveryMode,
            requestedDeliveryDate: requestedDate,
            deliveryPriority: expressPriority ? "EXPRESS" : "STANDARD",
            checkoutPolicyToken: checkoutPolicyToken,
            currency: nil,
            autoConfirmOptIn: autoConfirmOptIn,
            paymentMethod: paymentMethod,
            clientOrderId: clientOrderId
        )
    }

    func applyPackSuggestion(cart: CartManager, skuId: String, suggestedQuantity: Int) {
        cart.applyPackSuggestion(skuId: skuId, suggestedQuantity: suggestedQuantity)
        packSuggestions.removeAll { $0.skuId == skuId }
        if packSuggestions.isEmpty {
            errorMessage = ""
            showError = false
        }
    }

    func snapCartItem(cart: CartManager, itemId: String, mode: PackSnapMode = .nearest) {
        guard let item = cart.items.first(where: { $0.id == itemId }) else { return }
        cart.snapToPackMultiple(itemId: itemId, mode: mode)
        let sku = skuFor(item)
        packSuggestions.removeAll { $0.skuId == sku || $0.skuId == item.product.id }
        if packSuggestions.isEmpty {
            errorMessage = ""
            showError = false
        }
    }

    func loadPaymentMethods() async {
        await fetchPaymentCatalog()
        await fetchCards()
    }

    func fetchPreview(cart: CartManager) async {
        guard !cart.isEmpty else {
            preview = nil
            oosItems = []
            stockWarnings = []
            return
        }
        previewLoading = true
        defer { previewLoading = false }
        let rid = AuthManager.shared.currentUser?.id ?? ""
        let payload = buildPayload(cart: cart, retailerId: rid, gateway: gatewayCode(for: selectedPaymentId))
        do {
            let result: CheckoutPreviewResponse
            if let previewFetcher {
                result = try await previewFetcher(payload)
            } else {
                result = try await RetailerCheckoutService.fetchPreview(api: api, payload: payload)
            }
            preview = result
            oosItems = result.oosItems ?? result.rejectedSkus ?? []
            stockWarnings = result.stockWarnings
            cart.applyPreviewCaps(result)
            checkoutPolicyToken = result.checkoutPolicyToken
        } catch let previewError as CheckoutPreviewError {
            if case .blocked(_, let items) = previewError {
                oosItems = items
            }
        } catch {
            previewLoading = false
        }
    }

    func fetchCards() async {
        do {
            let cards = try await api.getCards()
            let tokenOptions = cards.map { card in
                let suffix = card.cardLast4.isEmpty ? "****" : card.cardLast4
                return CheckoutPaymentOption(
                    id: card.tokenId,
                    label: "•••• \(suffix) (\(card.cardType))",
                    isToken: true
                )
            }
            paymentOptions = tokenOptions + catalogPaymentOptions()
            if !paymentOptions.contains(where: { $0.id == selectedPaymentId }) {
                selectedPaymentId = paymentOptions.first?.id ?? "Cash"
            }
        } catch {
            paymentOptions = catalogPaymentOptions()
        }
    }

    func fetchPaymentCatalog() async {
        do {
            let resp = try await api.getPaymentCatalog()
            catalogCodes = selectableRetailerCatalogCodes(resp.catalog)
            if !resp.currencyCode.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty {
                packDisplayCurrency = resp.currencyCode.uppercased()
            }
        } catch {
            catalogCodes = []
            packDisplayCurrency = packCurrency(MarketPackStore.pack)
        }
        paymentOptions = catalogPaymentOptions()
        if !paymentOptions.contains(where: { $0.id == selectedPaymentId }) {
            selectedPaymentId = paymentOptions.first?.id ?? "Cash"
        }
    }

    private func catalogPaymentOptions() -> [CheckoutPaymentOption] {
        let codes = catalogCodes.isEmpty ? ["CASH"] : catalogCodes
        return codes.compactMap { code in
            switch code {
            case "CASH":
                return CheckoutPaymentOption(id: "Cash", label: "Cash on Delivery", isToken: false)
            case "GLOBAL_PAY":
                return CheckoutPaymentOption(id: "GlobalPay", label: "GlobalPay", isToken: false)
            default:
                return nil
            }
        }
    }

    func retryWithFallbackMethod(_ method: String, cart: CartManager, modelContext: ModelContext) async {
        let normalized = method.trimmingCharacters(in: .whitespacesAndNewlines).uppercased()
        guard !normalized.isEmpty else { return }
        if normalized == "CASH" {
            selectedPaymentId = "Cash"
        } else if let nonCashOption = paymentOptions.first(where: { $0.id != "Cash" && $0.id.uppercased() != "CASH" }) {
            selectedPaymentId = nonCashOption.id
        } else {
            selectedPaymentId = "GlobalPay"
        }
        clearAwaitingOwnerApprovalCheckoutRows(modelContext: modelContext)
        checkoutSessionId = "\(checkoutSessionId)-fallback-\(normalized)"
        await submitOrder(cart: cart, modelContext: modelContext, paymentMethodOverride: normalized)
    }

    private func clearAwaitingOwnerApprovalCheckoutRows(modelContext: ModelContext) {
        if let rows = try? modelContext.fetch(FetchDescriptor<PendingOrder>()) {
            for row in rows where row.status == PendingOrder.statusAwaitingOwnerApproval && row.endpoint == "/v1/checkout/unified" {
                modelContext.delete(row)
            }
            try? modelContext.save()
        }
    }

    func submitOrder(cart: CartManager, modelContext: ModelContext, paymentMethodOverride: String? = nil) async {
        isSubmitting = true
        packSuggestions = []
        pendingQueueNotice = nil
        ownerApprovalRequired = nil
        errorMessage = ""
        showError = false
        let rid = AuthManager.shared.currentUser?.id ?? ""

        let option = paymentOptions.first(where: { $0.id == selectedPaymentId })
        var finalGateway = paymentMethodOverride == "CASH"
            ? "CASH"
            : gatewayCode(for: selectedPaymentId)

        if option?.isToken == true {
            do {
                try await api.setDefaultCard(tokenId: selectedPaymentId)
                finalGateway = catalogCodes.contains("GLOBAL_PAY")
                    ? "GLOBAL_PAY"
                    : (catalogCodes.contains("CASH") ? "CASH" : (catalogCodes.first ?? "CASH"))
            } catch {
                Haptics.error()
                errorMessage = RetailerErrorSupport.message(
                    for: error,
                    restricted: "Payment method selection is restricted for this account.",
                    offline: "Offline mode active. Reconnect and retry payment method selection.",
                    fallback: "Failed to select payment method. Please try again."
                )
                showError = true
                isSubmitting = false
                return
            }
        }

        await fetchPreview(cart: cart)
        if let preview, preview.code == "order_acceptance_closed" {
            Haptics.warning()
            errorMessage = preview.message ?? "This supplier is not accepting orders at this time."
            showError = true
            isSubmitting = false
            return
        }
        if let preview, preview.blocked == true {
            Haptics.warning()
            oosItems = preview.oosItems ?? preview.rejectedSkus ?? []
            errorMessage = preview.message ?? "Checkout blocked by stock policy"
            showError = true
            isSubmitting = false
            return
        }
        if !skipBackorderConfirm, let preview, !preview.stockWarnings.isEmpty {
            Haptics.warning()
            pendingBackorderPreview = preview
            stockWarnings = preview.stockWarnings
            showBackorderConfirm = true
            isSubmitting = false
            return
        }
        skipBackorderConfirm = false

        stockWarnings = preview?.stockWarnings ?? []
        oosItems = preview?.oosItems ?? preview?.rejectedSkus ?? []

        let payload = buildPayload(
            cart: cart,
            retailerId: rid,
            gateway: finalGateway,
            paymentMethod: paymentMethodOverride
        )
        let idempotencyKey = checkoutIdempotencyKey(payload: payload, gateway: finalGateway)
        do {
            let response: CheckoutResponse
            if let checkoutSender {
                response = try await checkoutSender(payload, idempotencyKey)
            } else {
                response = try await api.post(
                    path: "/v1/checkout/unified",
                    body: payload,
                    headers: ["Idempotency-Key": idempotencyKey]
                )
            }
            stockWarnings = response.stockWarnings ?? []
            cart.clear()
            checkoutSessionId = UUID().uuidString
            Haptics.success()
            withAnimation(AnimationConstants.fluid) { showSuccess = true }
        } catch let previewError as CheckoutPreviewError {
            switch previewError {
            case .blocked(let message, let items):
                oosItems = items
                errorMessage = message
            case .backorderConfirmationRequired(let previewData):
                stockWarnings = previewData.stockWarnings
                showBackorderConfirm = true
            }
            Haptics.error()
            showError = previewError.errorDescription != nil && !showBackorderConfirm
            if showBackorderConfirm {
                errorMessage = ""
            }
            await fetchPreview(cart: cart)
        } catch {
            let skuNames = Dictionary(
                uniqueKeysWithValues: cart.items.map { (skuFor($0), $0.product.name) }
            )
            let parsedPack = RetailerOrderContractHelper.parsePackMultipleSuggestions(
                from: error,
                skuNamesBySku: skuNames
            )
            if !parsedPack.isEmpty {
                packSuggestions = parsedPack
                Haptics.warning()
                errorMessage = "Adjust quantities to full pack multiples."
                isSubmitting = false
                return
            }

            let extracted = RetailerOrderContractHelper.extractStatusCodeAndPayload(from: error)
            let normalizedCode = extracted.code ?? ""
            if extracted.status == 409,
               ["inventory_exhausted", "ALL_ITEMS_OUT_OF_STOCK", "PARTIAL_OUT_OF_STOCK_REJECTED"].contains(normalizedCode) {
                if let items = extracted.rawJson?["oos_items"] as? [String] {
                    oosItems = items
                }
                switch normalizedCode {
                case "inventory_exhausted":
                    errorMessage = "Stock changed while checking out. Review your cart and try again."
                case "ALL_ITEMS_OUT_OF_STOCK":
                    errorMessage = "All items are out of stock. Please update your cart."
                default:
                    errorMessage = "Some items are out of stock and cannot be backordered. Please update your cart."
                }
                Haptics.error()
                showError = true
                await fetchPreview(cart: cart)
                isSubmitting = false
                return
            }

            let ownerApproval = RetailerW2ErrorParser.parseOwnerApprovalRequired(from: error)
            let assessment = RetailerOrderContractHelper.assessPendingOrderError(error)
            switch assessment.disposition {
            case .parkAwaitingOwnerApproval, .parkRequestInProgress, .parkRateLimited, .parkAuthRequired, .retryTransient:
                let message = ownerApproval?.message ?? assessment.userMessage
                queueOfflineOrder(
                    payload: payload,
                    idempotencyKey: idempotencyKey,
                    status: assessment.queueStatus,
                    lastError: message,
                    modelContext: modelContext
                )
                let notice = PendingOrderStatusNotice(
                    endpoint: "/v1/checkout/unified",
                    idempotencyKey: idempotencyKey,
                    status: assessment.queueStatus,
                    message: message,
                    disposition: assessment.disposition
                )
                pendingQueueNotice = notice
                ownerApprovalRequired = ownerApproval
                PendingOrderStatusCenter.shared.publish(notice)
                Haptics.warning()
                errorMessage = message
                showError = (ownerApproval == nil)
            case .purgeRejectedValidation, .purgeIdempotentDuplicate:
                Haptics.error()
                errorMessage = assessment.userMessage
                showError = true
            }
        }
        isSubmitting = false
    }

    private func queueOfflineOrder(
        payload: UnifiedCheckoutPayload,
        idempotencyKey: String,
        status: String = PendingOrder.statusPending,
        lastError: String? = nil,
        modelContext: ModelContext
    ) {
        if let data = try? JSONEncoder().encode(payload) {
            let pending = PendingOrder(
                payloadJson: String(data: data, encoding: .utf8) ?? "",
                idempotencyKey: idempotencyKey,
                status: status
            )
            pending.lastError = lastError
            modelContext.insert(pending)
            try? modelContext.save()
        }
    }

    private func checkoutIdempotencyKey(payload: UnifiedCheckoutPayload, gateway: String, sessionId: String? = nil) -> String {
        let itemKey = payload.items
            .map { "\($0.skuId):\($0.quantity):\($0.unitPriceUzs):\($0.mustHave ? 1 : 0):\($0.allowSubstitutes ? 1 : 0)" }
            .sorted()
            .joined(separator: "|")
        let sid = sessionId ?? checkoutSessionId
        let autoSuffix = payload.autoConfirmOptIn ? ":auto" : ""
        return "retailer-checkout:\(gateway):\(sid):\(itemKey)\(autoSuffix)"
    }
}

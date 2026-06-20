import SwiftData
import SwiftUI


struct CheckoutPaymentOption: Identifiable, Hashable {
    let id: String
    let label: String
    let isToken: Bool
}

struct CheckoutView: View {
    var supplierIsActive: Bool = true

    @Environment(CartManager.self) private var cart
    @Environment(\.dismiss) private var dismiss
    @Environment(\.modelContext) private var modelContext

    @State private var selectedPaymentId = "GlobalPay"
    @State private var paymentOptions: [CheckoutPaymentOption] = [
        CheckoutPaymentOption(id: "Cash", label: "Cash on Delivery", isToken: false),
        CheckoutPaymentOption(id: "GlobalPay", label: "GlobalPay", isToken: false)
    ]
    @State private var showPaymentPicker = false
    @State private var isSubmitting = false
    @State private var showSuccess = false
    @State private var showError = false
    @State private var errorMessage = ""
    @State private var oosItems: [String] = []
    @State private var stockWarnings: [StockWarning] = []
    @State private var showSupplierClosedWarning = false
    @State private var preview: CheckoutPreviewResponse?
    @State private var previewLoading = false
    @State private var deliveryMode = "STANDARD"
    @State private var deliveryDate = ""
    @State private var expressPriority = false
    @State private var showBackorderConfirm = false
    @State private var pendingBackorderPreview: CheckoutPreviewResponse?
    @State private var skipBackorderConfirm = false
    @State private var checkoutPolicyToken: String?

    private let api = APIClient.shared

    private var selectedPaymentLabel: String {
        paymentOptions.first(where: { $0.id == selectedPaymentId })?.label ?? "GlobalPay"
    }

    /// Map UI labels to backend gateway codes expected by /v1/checkout/unified
    private func gatewayCode(for methodId: String) -> String {
        switch methodId {
        case "GlobalPay": return "GLOBAL_PAY"
        case "Adyen": return "ADYEN"
        case "Cash": return "CASH"
        default: return "GLOBAL_PAY"
        }
    }

    var body: some View {
        NavigationStack {
            ZStack {
                AppTheme.background.ignoresSafeArea()

                if showSuccess {
                    successView
                        .transition(.opacity.combined(with: .scale(scale: 0.9)))
                } else {
                    checkoutContent
                        .transition(.opacity)
                }
            }
            .animation(AnimationConstants.fluid, value: showSuccess)
            .navigationTitle("Checkout")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button {
                        dismiss()
                    } label: {
                        Image(systemName: "xmark")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(AppTheme.textSecondary)
                            .frame(width: 30, height: 30)
                            .background(AppTheme.surfaceElevated)
                            .clipShape(.circle)
                    }
                }
            }
            .task {
                await fetchCards()
            }
            .sheet(isPresented: $showPaymentPicker) {
                paymentPickerSheet
                    .presentationDetents([.medium])
                    .presentationDragIndicator(.visible)
            }
            .alert("Order Failed", isPresented: $showError) {
                Button("OK", role: .cancel) {}
            } message: {
                Text(errorMessage)
            }
            .confirmationDialog(
                "Supplier is Currently Closed",
                isPresented: $showSupplierClosedWarning,
                titleVisibility: .visible
            ) {
                Button("I Understand, Place Order") {
                    Task { await submitOrder() }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("This supplier is off-shift or outside business hours. Your order will be placed, but processing will not begin until they are back online.")
            }
            .confirmationDialog(
                "Partial backorder",
                isPresented: $showBackorderConfirm,
                titleVisibility: .visible
            ) {
                Button("Proceed") {
                    skipBackorderConfirm = true
                    Task { await submitOrder() }
                }
                Button("Cancel", role: .cancel) {
                    pendingBackorderPreview = nil
                }
            } message: {
                Text("Some items will be backordered. Delivery may be delayed. Proceed?")
            }
            .task {
                await fetchPreview()
            }
            .onChange(of: cart.totalItems) { _, _ in
                Task { await fetchPreview() }
            }
        }
    }

    private var scheduledMinDate: String {
        let lead = Int(preview?.preorderMinLeadDays ?? 3)
        return Calendar.current.date(byAdding: .day, value: max(1, lead), to: Date())
            .map { ISO8601DateFormatter().string(from: $0).prefix(10) }
            .map(String.init) ?? ""
    }

    private var scheduledMaxDate: String? {
        guard let maxLead = preview?.preorderMaxLeadDays, maxLead > 0 else { return nil }
        return Calendar.current.date(byAdding: .day, value: Int(maxLead), to: Date())
            .map { ISO8601DateFormatter().string(from: $0).prefix(10) }
            .map(String.init)
    }

    private func skuFor(_ item: CartItem) -> String {
        item.variant.id.isEmpty ? item.product.id : item.variant.id
    }

    private func isOos(_ item: CartItem) -> Bool {
        let sku = skuFor(item)
        if oosItems.contains(sku) { return true }
        if let shortfall = preview?.shortfall?[sku], shortfall > 0 { return true }
        return false
    }

    private func fetchPreview() async {
        guard !cart.isEmpty else {
            preview = nil
            oosItems = []
            stockWarnings = []
            return
        }
        previewLoading = true
        defer { previewLoading = false }
        let rid = AuthManager.shared.currentUser?.id ?? ""
        let payload = buildPayload(retailerId: rid, gateway: gatewayCode(for: selectedPaymentId))
        do {
            let result = try await RetailerCheckoutService.fetchPreview(api: api, payload: payload)
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
            // Keep last preview state when refresh fails transiently.
        }
    }

    private func buildPayload(retailerId: String, gateway: String) -> UnifiedCheckoutPayload {
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
            checkoutPolicyToken: checkoutPolicyToken
        )
    }

    // MARK: - Checkout Content

    private var checkoutContent: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
                    if previewLoading {
                        HStack(spacing: AppTheme.spacingSM) {
                            ProgressView()
                            Text("Refreshing stock availability…")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                    }
                    if !stockWarnings.isEmpty {
                        stockWarningsSection
                    }
                    if !oosItems.isEmpty {
                        oosItemsSection
                    }
                    deliverySection.slideIn(delay: 0)
                    cartItemsSection.slideIn(delay: 0)
                    paymentSection.slideIn(delay: 0.05)
                    summarySection.slideIn(delay: 0.1)
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, 100)
            }
            .scrollIndicators(.hidden)

            submitButton
        }
    }

    // MARK: - Stock Warnings

    private var stockWarningsSection: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Label("Partial backorder", systemImage: "exclamationmark.triangle.fill")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)
                Text("Some items are out of stock but your warehouse accepts backorders.")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
                ForEach(stockWarnings, id: \.sku) { warning in
                    Text("\(warning.sku): \(warning.backorderQty) of \(warning.requested) backordered")
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(AppTheme.textPrimary)
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private var oosItemsSection: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Label("Out of stock", systemImage: "xmark.circle.fill")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.destructive)
                ForEach(oosItems, id: \.self) { sku in
                    Text(sku)
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(AppTheme.textPrimary)
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private var deliverySection: some View {
        LabCardWithHeader(title: "Delivery", icon: "truck.box.fill") {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(spacing: AppTheme.spacingSM) {
                    deliveryModeButton("Standard", mode: "STANDARD", subtitle: "ASAP")
                    deliveryModeButton("Scheduled", mode: "SCHEDULED", subtitle: "T+\(preview?.preorderMinLeadDays ?? 3)")
                }
                if deliveryMode == "SCHEDULED" {
                    TextField("YYYY-MM-DD", text: $deliveryDate)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(.caption, design: .monospaced))
                    Text("Choose \(scheduledMinDate)\(scheduledMaxDate.map { " to \($0)" } ?? " or later")")
                        .font(.caption2)
                        .foregroundStyle(AppTheme.textTertiary)
                }
                Toggle("Express priority (+fee)", isOn: $expressPriority)
                    .font(.system(.caption, design: .rounded))
                if let fee = preview?.deliveryFeeMinor, fee > 0 {
                    Text("Delivery fee: \(Int(fee).formatted()) UZS\(preview?.deliveryDistanceKm.map { " · \(String(format: "%.1f", $0)) km" } ?? "")")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func deliveryModeButton(_ title: String, mode: String, subtitle: String) -> some View {
        Button {
            deliveryMode = mode
        } label: {
            VStack(alignment: .leading, spacing: 2) {
                Text(title)
                    .font(.system(.caption, design: .rounded, weight: .semibold))
                Text(subtitle)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(AppTheme.spacingMD)
            .background(deliveryMode == mode ? AppTheme.accentSoft.opacity(0.45) : AppTheme.surfaceElevated)
            .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusSM))
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .stroke(deliveryMode == mode ? AppTheme.accent : AppTheme.separator.opacity(0.4), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }

    // MARK: - Cart Items

    private var cartItemsSection: some View {
        LabCardWithHeader(title: "Cart", subtitle: "\(cart.totalItems) items", icon: "cart.fill") {
            if cart.isEmpty {
                VStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        Circle()
                            .fill(AppTheme.accentSoft.opacity(0.3))
                            .frame(width: 56, height: 56)
                        Image(systemName: "cart")
                            .font(.system(size: 22))
                            .foregroundStyle(AppTheme.accent.opacity(0.5))
                    }
                    Text("Your cart is empty")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, AppTheme.spacingXL)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(cart.items.enumerated()), id: \.element.id) { index, item in
                        if index > 0 {
                            Rectangle()
                                .fill(AppTheme.separator.opacity(0.3))
                                .frame(height: AppTheme.separatorHeight)
                                .padding(.horizontal, AppTheme.spacingXS)
                        }
                        cartItemRow(item)
                    }
                }
            }
        }
    }

    private func cartItemRow(_ item: CartItem) -> some View {
        HStack(spacing: AppTheme.spacingMD) {
            // Product icon
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(AppTheme.accentSoft.opacity(0.3))
                    .frame(width: 40, height: 40)
                Image(systemName: "leaf.fill")
                    .font(.system(size: 16))
                    .foregroundStyle(AppTheme.accent.opacity(0.6))
            }

            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: AppTheme.spacingSM) {
                    Text(item.product.name)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                        .lineLimit(1)
                    if isOos(item) {
                        Text("OOS")
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.destructive)
                            .padding(.horizontal, 6).padding(.vertical, 2)
                            .background(AppTheme.destructive.opacity(0.12))
                            .clipShape(.capsule)
                    }
                }
                Text(item.variant.size)
                    .font(.system(.caption2, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Spacer()

            QuantityStepper(
                quantity: Binding(
                    get: { item.quantity },
                    set: { cart.updateQuantity(itemId: item.id, quantity: $0) }
                ),
                maximum: cart.maxQuantity(for: item, preview: preview),
                compact: true
            )

            Text("\(Int(item.totalPrice).formatted())")
                .font(.system(.caption, design: .rounded, weight: .bold))
                .monospacedDigit()
                .foregroundStyle(AppTheme.accent)
                .frame(width: 70, alignment: .trailing)
        }
        .padding(.vertical, AppTheme.spacingSM)
    }

    // MARK: - Payment

    private var paymentSection: some View {
        LabCardWithHeader(title: "Payment", icon: "creditcard.fill") {
            Button {
                showPaymentPicker = true
            } label: {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(AppTheme.infoSoft.opacity(0.5))
                            .frame(width: 36, height: 36)
                        Image(systemName: "creditcard")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(AppTheme.info)
                    }

                    Text(selectedPaymentLabel)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)

                    Spacer()

                    Text("Change")
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.accent)
                }
            }
        }
    }

    // MARK: - Summary

    private var summarySection: some View {
        LabCard {
            VStack(spacing: AppTheme.spacingMD) {
                summaryRow("Subtotal", value: "\(Int(cart.checkoutSubtotal).formatted())")
                if cart.checkoutDiscount > 0 {
                    summaryRow(
                        "Promotion",
                        value: "-\(Int(cart.checkoutDiscount).formatted())",
                        valueColor: AppTheme.success
                    )
                }
                summaryRow("Delivery", value: "Free", valueColor: AppTheme.success)

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.5))
                    .frame(height: AppTheme.separatorHeight)

                HStack {
                    Text("Total")
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)
                    Spacer()
                    AnimatedCurrencyText(value: cart.checkoutTotal, font: .system(.title3, design: .rounded, weight: .bold))
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func summaryRow(_ title: String, value: String, valueColor: Color = AppTheme.textPrimary) -> some View {
        HStack {
            Text(title)
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
            Text(value)
                .font(.system(.subheadline, design: .rounded, weight: .medium))
                .foregroundStyle(valueColor)
        }
    }

    // MARK: - Submit Button

    private var submitButton: some View {
        VStack(spacing: 0) {
            Rectangle()
                .fill(AppTheme.separator.opacity(0.3))
                .frame(height: AppTheme.separatorHeight)

            LabButton("Place Order", icon: "checkmark.circle", fullWidth: true) {
                if !supplierIsActive {
                    showSupplierClosedWarning = true
                } else {
                    Task { await submitOrder() }
                }
            }
            .padding(AppTheme.spacingLG)
            .opacity(cart.isEmpty || isSubmitting || previewLoading ? 0.5 : 1.0)
            .disabled(cart.isEmpty || isSubmitting || previewLoading)
        }
        .background(.ultraThinMaterial)
    }

    // MARK: - Payment Picker

    private func fetchCards() async {
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
            paymentOptions = tokenOptions + [
                CheckoutPaymentOption(id: "Cash", label: "Cash on Delivery", isToken: false),
                CheckoutPaymentOption(id: "GlobalPay", label: "GlobalPay", isToken: false)
            ]
        } catch {
            print("Failed to fetch cards")
        }
    }

    private var paymentPickerSheet: some View {
        NavigationStack {
            List(paymentOptions, id: \.id) { option in
                Button {
                    withAnimation(AnimationConstants.express) {
                        selectedPaymentId = option.id
                    }
                    showPaymentPicker = false
                } label: {
                    HStack(spacing: AppTheme.spacingMD) {
                        Image(systemName: paymentIcon(option))
                            .font(.system(size: 16, weight: .medium))
                            .foregroundStyle(AppTheme.accent)
                            .frame(width: 24)

                        Text(option.label)
                            .font(.system(.body, design: .rounded))
                            .foregroundStyle(AppTheme.textPrimary)

                        Spacer()

                        if option.id == selectedPaymentId {
                            Image(systemName: "checkmark.circle.fill")
                                .foregroundStyle(AppTheme.accent)
                        }
                    }
                }
            }
            .navigationTitle("Payment Method")
            .navigationBarTitleDisplayMode(.inline)
        }
    }

    private func paymentIcon(_ option: CheckoutPaymentOption) -> String {
        if option.isToken { return "creditcard.fill" }
        switch option.id {
        case "GlobalPay": return "wallet.pass"
        case "Adyen": return "creditcard.and.123"
        case "Cash": return "banknote"
        default: return "creditcard"
        }
    }

    // MARK: - Success View

    private var successView: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()

            ZStack {
                Circle()
                    .fill(AppTheme.successSoft.opacity(0.3))
                    .frame(width: 120, height: 120)
                Circle()
                    .fill(AppTheme.successSoft.opacity(0.5))
                    .frame(width: 90, height: 90)
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 52))
                    .foregroundStyle(AppTheme.success)
                    .symbolEffect(.bounce, value: showSuccess)
            }

            VStack(spacing: AppTheme.spacingSM) {
                Text("Order Placed! 🎉")
                    .font(.system(.title2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text(stockWarnings.isEmpty
                     ? "Your order has been submitted successfully.\nYou'll receive updates in your inbox."
                     : "Your order was placed with partial backorder items.\nFulfillment will continue when stock arrives.")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .multilineTextAlignment(.center)
            }

            Spacer()

            LabButton("Done", icon: "checkmark", fullWidth: true) {
                dismiss()
            }
            .padding(.horizontal, AppTheme.spacingXL)
            .padding(.bottom, AppTheme.spacingXL)
        }
    }

    // MARK: - API

    private func submitOrder() async {
        isSubmitting = true
        let rid = AuthManager.shared.currentUser?.id ?? ""
        
        let option = paymentOptions.first(where: { $0.id == selectedPaymentId })
        var finalGateway = gatewayCode(for: selectedPaymentId)
        
        if option?.isToken == true {
            do {
                try await api.setDefaultCard(tokenId: selectedPaymentId)
                finalGateway = "GLOBAL_PAY"
            } catch {
                errorMessage = RetailerErrorSupport.message(
                    for: error,
                    restricted: "Payment method selection is restricted for this account.",
                    offline: "Offline mode active. Reconnect and retry payment method selection.",
                    fallback: "Failed to select payment method. Please try again.",
                )
                showError = true
                isSubmitting = false
                return
            }
        }

        await fetchPreview()
        if let preview, preview.code == "order_acceptance_closed" {
            errorMessage = preview.message ?? "This supplier is not accepting orders at this time."
            showError = true
            isSubmitting = false
            return
        }
        if let preview, preview.blocked == true {
            oosItems = preview.oosItems ?? preview.rejectedSkus ?? []
            errorMessage = preview.message ?? "Checkout blocked by stock policy"
            showError = true
            isSubmitting = false
            return
        }
        if !skipBackorderConfirm, let preview, !preview.stockWarnings.isEmpty {
            pendingBackorderPreview = preview
            stockWarnings = preview.stockWarnings
            showBackorderConfirm = true
            isSubmitting = false
            return
        }
        skipBackorderConfirm = false
        
        stockWarnings = preview?.stockWarnings ?? []
        oosItems = preview?.oosItems ?? preview?.rejectedSkus ?? []
        
        let payload = buildPayload(retailerId: rid, gateway: finalGateway)
        let idempotencyKey = checkoutIdempotencyKey(payload: payload, gateway: finalGateway)
        do {
            let response: CheckoutResponse = try await api.post(
                path: "/v1/checkout/unified",
                body: payload,
                headers: ["Idempotency-Key": idempotencyKey]
            )
            stockWarnings = response.stockWarnings ?? []
            cart.clear()
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
            await fetchPreview()
        } catch let apiError as APIError {
            if case .serverError(let statusCode, let message) = apiError, statusCode == 409 {
                if let jsonData = message.data(using: .utf8),
                   let json = try? JSONSerialization.jsonObject(with: jsonData) as? [String: Any] {
                    let code = json["code"] as? String ?? ""
                    if let items = json["oos_items"] as? [String] {
                        oosItems = items
                    }
                    switch code {
                    case "inventory_exhausted":
                        errorMessage = "Stock changed while checking out. Review your cart and try again."
                    case "ALL_ITEMS_OUT_OF_STOCK":
                        errorMessage = "All items are out of stock. Please update your cart."
                    case "PARTIAL_OUT_OF_STOCK_REJECTED":
                        errorMessage = "Some items are out of stock and cannot be backordered. Please update your cart."
                    default:
                        errorMessage = "Items are out of stock — please refresh"
                    }
                } else {
                    errorMessage = "Items are out of stock — please refresh"
                }
                Haptics.error()
                showError = true
                await fetchPreview()
            } else {
                // Queue for offline retry
                if let data = try? JSONEncoder().encode(payload) {
                    let pending = PendingOrder(
                        payloadJson: String(data: data, encoding: .utf8) ?? "",
                        idempotencyKey: idempotencyKey
                    )
                    modelContext.insert(pending)
                    try? modelContext.save()
                }
                Haptics.error()
                errorMessage = RetailerErrorSupport.retryQueuedMessage(
                    for: apiError,
                    fallback: "Saved for retry. Payment status is degraded.",
                )
                showError = true
            }
        } catch {
            // Queue for offline retry
            if let data = try? JSONEncoder().encode(payload) {
                let pending = PendingOrder(
                    payloadJson: String(data: data, encoding: .utf8) ?? "",
                    idempotencyKey: idempotencyKey
                )
                modelContext.insert(pending)
                try? modelContext.save()
            }
            Haptics.error()
            errorMessage = RetailerErrorSupport.retryQueuedMessage(
                for: error,
                fallback: "Saved for retry. Payment status is degraded.",
            )
            showError = true
        }
        isSubmitting = false
    }

    private func checkoutIdempotencyKey(payload: UnifiedCheckoutPayload, gateway: String) -> String {
        let itemKey = payload.items
            .map { "\($0.skuId):\($0.quantity):\($0.unitPriceUzs)" }
            .sorted()
            .joined(separator: "|")
        return "retailer-checkout:\(gateway):\(itemKey)"
    }
}

#Preview {
    CheckoutView()
        .environment(CartManager())
}

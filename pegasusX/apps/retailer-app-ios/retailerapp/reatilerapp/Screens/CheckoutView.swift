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
    @State private var showSupplierClosedWarning = false

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
        }
    }

    // MARK: - Checkout Content

    private var checkoutContent: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
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
                Text(item.product.name)
                    .font(.system(.subheadline, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
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
                summaryRow("Subtotal", value: cart.displayTotal)
                summaryRow("Delivery", value: "Free", valueColor: AppTheme.success)

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.5))
                    .frame(height: AppTheme.separatorHeight)

                HStack {
                    Text("Total")
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)
                    Spacer()
                    AnimatedCurrencyText(value: cart.totalPrice, font: .system(.title3, design: .rounded, weight: .bold))
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
            .opacity(cart.isEmpty || isSubmitting ? 0.5 : 1.0)
            .disabled(cart.isEmpty || isSubmitting)
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

                Text("Your order has been submitted successfully.\nYou'll receive updates in your inbox.")
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
        
        let payload = cart.buildCheckoutPayload(retailerId: rid, paymentGateway: finalGateway)
        let idempotencyKey = checkoutIdempotencyKey(payload: payload, gateway: finalGateway)
        do {
            _ = try await RetailerCheckoutService.completeCheckout(
                api: api,
                payload: payload,
                gateway: finalGateway,
                idempotencyKey: idempotencyKey
            )
            cart.clear()
            Haptics.success()
            withAnimation(AnimationConstants.fluid) { showSuccess = true }
        } catch let apiError as APIError {
            if case .serverError(let statusCode, let message) = apiError, statusCode == 409 {
                // Parse structured OOS response from body string
                if let jsonData = message.data(using: .utf8),
                   let json = try? JSONSerialization.jsonObject(with: jsonData) as? [String: Any] {
                    let code = json["code"] as? String ?? ""
                    if code == "ALL_ITEMS_OUT_OF_STOCK",
                       let items = json["oos_items"] as? [String] {
                        oosItems = items
                    }
                    errorMessage = "All items are out of stock. Please update your cart."
                } else {
                    errorMessage = "Items are out of stock — please refresh"
                }
                Haptics.error()
                showError = true
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

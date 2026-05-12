import re

target = "pegasus/apps/retailer-app-ios/retailerapp/reatilerapp/Screens/CheckoutView.swift"
with open(target, "r") as f:
    text = f.read()

# Add CheckoutPaymentOption struct at the top or inside CheckoutView
struct_code = """
struct CheckoutPaymentOption: Identifiable, Hashable {
    let id: String
    let label: String
    let isToken: Bool
}
"""

if "struct CheckoutPaymentOption" not in text:
    text = text.replace("struct CheckoutView: View {", struct_code + "\nstruct CheckoutView: View {")

# Replace selectedPayment and paymentMethods
old_state = """    @State private var selectedPayment = "GlobalPay"
    @State private var showPaymentPicker = false
    @State private var isSubmitting = false
    @State private var showSuccess = false
    @State private var showError = false
    @State private var errorMessage = ""
    @State private var oosItems: [String] = []
    @State private var showSupplierClosedWarning = false

    private let api = APIClient.shared
    private let paymentMethods = ["GlobalPay", "Cash on Delivery"]

    /// Map UI labels to backend gateway codes expected by /v1/checkout/unified
    private func gatewayCode(for method: String) -> String {
        switch method {
        case "GlobalPay": return "GLOBAL_PAY"
        case "Cash on Delivery": return "CASH"
        default: return "GLOBAL_PAY"
        }
    }"""

new_state = """    @State private var selectedPaymentId = "GlobalPay"
    @State private var paymentOptions: [CheckoutPaymentOption] = [
        CheckoutPaymentOption(id: "GlobalPay", label: "GlobalPay", isToken: false),
        CheckoutPaymentOption(id: "Cash", label: "Cash on Delivery", isToken: false)
    ]
    @State private var showPaymentPicker = false
    @State private var isSubmitting = false
    @State private var showSuccess = false
    @State private var showError = false
    @State private var errorMessage = ""
    @State private var oosItems: [String] = []
    @State private var showSupplierClosedWarning = false

    private let api = APIClient.shared

    /// Map UI labels to backend gateway codes expected by /v1/checkout/unified
    private func gatewayCode(for methodId: String) -> String {
        switch methodId {
        case "GlobalPay": return "GLOBAL_PAY"
        case "Cash": return "CASH"
        default: return "GLOBAL_PAY"
        }
    }"""

text = text.replace(old_state, new_state)

# Add .task to fetch cards
old_task = """            .sheet(isPresented: $showPaymentPicker) {"""
new_task = """            .task {
                await fetchCards()
            }
            .sheet(isPresented: $showPaymentPicker) {"""
if ".task {" not in text:
    text = text.replace(old_task, new_task)

# Make sure we don't break payment selection rendering
old_selected_display = """                VStack(alignment: .leading, spacing: 2) {
                    Text("Payment Method")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    Text(selectedPayment)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)"""

new_selected_display = """                VStack(alignment: .leading, spacing: 2) {
                    Text("Payment Method")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    Text(paymentOptions.first(where: { $0.id == selectedPaymentId })?.label ?? "GlobalPay")
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)"""

text = text.replace(old_selected_display, new_selected_display)

# Replace Payment Picker Sheet
old_picker = """    private var paymentPickerSheet: some View {
        NavigationStack {
            List(paymentMethods, id: \.self) { method in
                Button {
                    withAnimation(AnimationConstants.express) {
                        selectedPayment = method
                    }
                    showPaymentPicker = false
                } label: {
                    HStack(spacing: AppTheme.spacingMD) {
                        Image(systemName: paymentIcon(method))
                            .font(.system(size: 16, weight: .medium))
                            .foregroundStyle(AppTheme.accent)
                            .frame(width: 24)

                        Text(method)
                            .font(.system(.body, design: .rounded))
                            .foregroundStyle(AppTheme.textPrimary)

                        Spacer()

                        if method == selectedPayment {
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

    private func paymentIcon(_ method: String) -> String {
        switch method {
        case "GlobalPay": "wallet.pass"
        case "Cash on Delivery": "banknote"
        default: "creditcard"
        }
    }"""

new_picker = """    private func fetchCards() async {
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
                CheckoutPaymentOption(id: "GlobalPay", label: "GlobalPay", isToken: false),
                CheckoutPaymentOption(id: "Cash", label: "Cash on Delivery", isToken: false)
            ]
        } catch {
            print("Failed to fetch cards: \(error.localizedDescription)")
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
        case "Cash": return "banknote"
        default: return "creditcard"
        }
    }"""

text = text.replace(old_picker, new_picker)

# Replace submitOrder
old_submit = """    private func submitOrder() async {
        isSubmitting = true
        let rid = AuthManager.shared.currentUser?.id ?? ""
        let gateway = gatewayCode(for: selectedPayment)
        let payload = cart.buildCheckoutPayload(retailerId: rid, paymentGateway: gateway)
        let idempotencyKey = checkoutIdempotencyKey(payload: payload, gateway: gateway)"""
new_submit = """    private func submitOrder() async {
        isSubmitting = true
        let rid = AuthManager.shared.currentUser?.id ?? ""
        
        let option = paymentOptions.first(where: { $0.id == selectedPaymentId })
        var finalGateway = gatewayCode(for: selectedPaymentId)
        
        if option?.isToken == true {
            do {
                try await api.setDefaultCard(tokenId: selectedPaymentId)
                finalGateway = "GLOBAL_PAY"
            } catch {
                errorMessage = "Failed to select payment method: \(error.localizedDescription)"
                showError = true
                isSubmitting = false
                return
            }
        }
        
        let payload = cart.buildCheckoutPayload(retailerId: rid, paymentGateway: finalGateway)
        let idempotencyKey = checkoutIdempotencyKey(payload: payload, gateway: finalGateway)"""

text = text.replace(old_submit, new_submit)

with open(target, "w") as f:
    f.write(text)

print("iOS Checkout view patched\!")

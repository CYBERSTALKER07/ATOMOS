import SwiftUI

// MARK: - Delivery Payment Sheet

struct DeliveryPaymentSheetView: View {
    let event: PaymentRequiredEvent
    let onDismiss: () -> Void

    @State private var phase: PaymentPhase = .choose
    @State private var errorMessage: String?
    @State private var showSavedCards = false
    @State private var catalogCodes: [String] = []
    @State private var packDisplayCurrency = packCurrency(MarketPackStore.pack)

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    enum PaymentPhase {
        case choose
        case cashConfirm
        case processing
        case cashPending
        case fiscalizing
        case success
        case failed
    }

    private struct CardGatewayOption: Identifiable {
        let gateway: String
        let label: String
        let description: String

        var id: String { gateway }
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                switch phase {
                case .choose:
                    chooseContent
                case .cashConfirm:
                    cashConfirmContent
                case .processing:
                    processingContent
                case .cashPending:
                    cashPendingContent
                case .fiscalizing:
                    fiscalizingContent
                case .success:
                    successContent
                case .failed:
                    failedContent
                }
            }
            .background(AppTheme.background)
            .navigationTitle("mobile_retailer.ui.payment_required")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    if phase == .choose || phase == .failed {
                        Button("common.action.close") { onDismiss() }
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    }
                }
            }
        }
        .sheet(isPresented: $showSavedCards) {
            NavigationStack {
                SavedCardsView(
                    returnTo: "delivery_payment",
                    orderId: event.orderId,
                    sessionId: event.sessionId,
                    onReturnToPayment: {
                        showSavedCards = false
                        phase = .choose
                        errorMessage = nil
                    }
                )
            }
            .presentationDetents([.large])
            .presentationDragIndicator(.visible)
        }
        .task { await listenForCompletion() }
        .task { await fetchPaymentCatalog() }
    }

    // MARK: - Choose

    private var chooseContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer(minLength: AppTheme.spacingXL)

            ZStack {
                Circle()
                    .fill(AppTheme.warningSoft)
                    .frame(width: 80, height: 80)
                Image(systemName: "banknote.fill")
                    .font(.system(size: 32, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)
            }

            VStack(spacing: AppTheme.spacingSM) {
                Text("mobile_retailer.ui.amount_due")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                // Show crossed-out original amount if items were rejected during offload
                if event.originalAmountUzs > 0 && event.originalAmountUzs != event.amountUzs {
                    Text(formattedOriginalAmount)
                        .font(.system(.title3, design: .rounded, weight: .medium))
                        .strikethrough(true, color: AppTheme.textTertiary)
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Text(formattedAmount)
                    .font(.system(size: 40, weight: .bold, design: .rounded))
                    .foregroundStyle(
                        event.originalAmountUzs > 0 && event.originalAmountUzs != event.amountUzs
                            ? AppTheme.warning
                            : AppTheme.textPrimary
                    )
                if !packDisplayCurrency.isEmpty {
                    Text(packDisplayCurrency)
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                Text(L10n.format("mobile_retailer.ui.order_suffix", "\(String(event.orderId.suffix(6)))"))
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(AppTheme.textTertiary)
            }

            Text("mobile_retailer.ui.choose_payment_method")
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textSecondary)

            VStack(spacing: AppTheme.spacingSM) {
                PaymentOptionButton(
                    icon: "banknote",
                    label: "Cash on Delivery",
                    description: "Pay the driver in cash"
                ) {
                    phase = .cashConfirm
                }

                ForEach(cardGatewayOptions) { option in
                    PaymentOptionButton(
                        icon: "creditcard.fill",
                        label: option.label,
                        description: option.description
                    ) {
                        phase = .processing
                        Task { await initiateCardCheckout(gateway: option.gateway) }
                    }
                }

                Button("mobile_retailer.ui.add_payment_method") {
                    showSavedCards = true
                }
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.accent)
                .padding(.top, AppTheme.spacingSM)
            }
            .padding(.horizontal, AppTheme.spacingLG)

            Spacer()
        }
    }

    // MARK: - Cash Confirm

    private var cashConfirmContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer(minLength: AppTheme.spacingXL)

            ZStack {
                Circle()
                    .fill(AppTheme.warningSoft)
                    .frame(width: 80, height: 80)
                Image(systemName: "banknote.fill")
                    .font(.system(size: 32, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)
            }

            VStack(spacing: AppTheme.spacingSM) {
                Text("mobile_retailer.ui.confirm_cash_handoff")
                    .font(.system(.title2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text(formattedAmount)
                    .font(.system(.title3, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)

                Text("mobile_retailer.ui.only_confirm_after_the_driver_has_physically_received_the_cash")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                    .multilineTextAlignment(.center)
            }

            HStack(spacing: AppTheme.spacingMD) {
                Button {
                    phase = .choose
                } label: {
                    Text("common.action.back")
                        .font(.system(.body, design: .rounded, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(AppTheme.surfaceElevated)
                        .foregroundStyle(AppTheme.textPrimary)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                }

                Button {
                    phase = .processing
                    Task { await confirmCash() }
                } label: {
                    Text("mobile_retailer.ui.confirm_cash_received")
                        .font(.system(.body, design: .rounded, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(AppTheme.warning)
                        .foregroundStyle(.white)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)

            Spacer()
        }
    }

    // MARK: - Processing

    private var processingContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()
            ProgressView()
                .scaleEffect(1.5)
                .tint(AppTheme.accent)
            Text("mobile_retailer.ui.processing")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("mobile_retailer.ui.connecting_to_payment_service")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            Spacer()
        }
    }

    // MARK: - Cash Pending

    private var cashPendingContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()

            ZStack {
                Circle()
                    .fill(AppTheme.warningSoft)
                    .frame(width: 80, height: 80)
                Image(systemName: "banknote.fill")
                    .font(.system(size: 40, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)
            }

            Text("mobile_retailer.ui.cash_collection_pending")
                .font(.system(.title2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)

            Text(formattedAmount)
                .font(.system(.title3, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.warning)

            Text("mobile_retailer.ui.please_have_the_cash_ready_nthe_driver_will_collect_it_shortly")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
                .multilineTextAlignment(.center)

            HStack(spacing: AppTheme.spacingSM) {
                ProgressView()
                    .scaleEffect(0.7)
                Text("mobile_retailer.ui.waiting_for_driver_confirmation")
                    .font(.system(.caption, design: .rounded, weight: .medium))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.vertical, AppTheme.spacingSM)
            .background(AppTheme.surfaceElevated)
            .clipShape(.capsule)

            Spacer()
        }
    }

    // MARK: - Fiscalizing (ADR-009)

    private var fiscalizingContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()
            ProgressView()
                .scaleEffect(1.5)
                .tint(AppTheme.warning)
            Text("mobile_retailer.ui.pending_fiscal_receipt")
                .font(.system(.title2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(formattedAmount)
                .font(.system(.title3, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.warning)
            Text("mobile_retailer.ui.payment_is_captured_nofficial_fiscal_document_is_being_issued")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
                .multilineTextAlignment(.center)
            Spacer()
        }
    }

    // MARK: - Success

    private var successContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()

            ZStack {
                Circle()
                    .fill(AppTheme.successSoft)
                    .frame(width: 80, height: 80)
                Image(systemName: "checkmark.circle.fill")
                    .font(.system(size: 40, weight: .semibold))
                    .foregroundStyle(AppTheme.success)
            }

            Text("mobile_retailer.ui.paid_and_fiscalized")
                .font(.system(.title2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)

            Text(formattedAmount)
                .font(.system(.title3, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.success)

            Spacer()

            Button {
                onDismiss()
            } label: {
                Text("warehouse_portal.kpi_stat_card.text.done")
                    .font(.system(.body, design: .rounded, weight: .bold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(AppTheme.success)
                    .foregroundStyle(.white)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingXXL)
        }
    }

    // MARK: - Failed

    private var failedContent: some View {
        VStack(spacing: AppTheme.spacingXL) {
            Spacer()

            ZStack {
                Circle()
                    .fill(AppTheme.destructiveSoft)
                    .frame(width: 80, height: 80)
                Image(systemName: "xmark.circle.fill")
                    .font(.system(size: 40, weight: .semibold))
                    .foregroundStyle(AppTheme.destructive)
            }

            Text("notification.payment_failed.title")
                .font(.system(.title2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)

            if let errorMessage {
                Text(errorMessage)
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, AppTheme.spacingXL)
            }

            Spacer()

            VStack(spacing: AppTheme.spacingMD) {
                Button {
                    phase = .choose
                    errorMessage = nil
                } label: {
                    Text("mobile_retailer.ui.try_again")
                        .font(.system(.body, design: .rounded, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(AppTheme.accent)
                        .foregroundStyle(AppTheme.cardBackground)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                }

                Button {
                    onDismiss()
                } label: {
                    Text("common.action.cancel")
                        .font(.system(.body, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingXXL)
        }
    }

    // MARK: - API

    private func initiateCashCheckout() async {
        do {
            let body = ["order_id": event.orderId]
            let _: [String: DiscardableCodable] = try await api.post(
                path: "/v1/order/cash-checkout",
                body: body,
                headers: ["Idempotency-Key": "retailer-cash-checkout:\(event.orderId)"]
            )
            phase = .cashPending
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Payment access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry payment.",
                fallback: "Payment could not be completed. Please try again.",
            )
            phase = .failed
        }
    }

    private func confirmCash() async {
        do {
            let body = ["order_id": event.orderId]
            let _: [String: DiscardableCodable] = try await api.post(
                path: "/v1/delivery/confirm-cash",
                body: body,
                headers: ["Idempotency-Key": RetailerIdempotency.confirmCash(orderId: event.orderId)]
            )
            phase = .cashPending
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Payment access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry cash confirmation.",
                fallback: "Cash confirmation could not be completed. Please try again.",
            )
            phase = .failed
        }
    }

    private func initiateCardCheckout(gateway: String) async {
        do {
            let body = ["order_id": event.orderId, "gateway": gateway]
            let resp: CardCheckoutResponse = try await api.post(
                path: "/v1/order/card-checkout",
                body: body,
                headers: ["Idempotency-Key": "retailer-card-checkout:\(event.orderId):\(gateway)"]
            )
            if let url = URL(string: resp.paymentUrl), !resp.paymentUrl.isEmpty {
                await MainActor.run {
                    UIApplication.shared.open(url)
                }
            } else {
                errorMessage = "Payment gateway is not configured for this supplier."
                phase = .failed
                return
            }
            // Stay on processing — webhook settlement triggers ORDER_COMPLETED via WS
        } catch {
            errorMessage = RetailerErrorSupport.message(
                for: error,
                restricted: "Payment access is restricted for this account.",
                offline: "Offline mode active. Reconnect and retry payment.",
                fallback: "Payment could not be completed. Please try again.",
            )
            phase = .failed
        }
    }

    private func listenForCompletion() async {
        for await event in ws.eventStream() {
            switch event {
            case .orderCompleted(let completed) where completed.orderId == self.event.orderId:
                phase = .success
                return
            case .fiscalSucceeded(let succeeded) where succeeded.orderId == self.event.orderId:
                phase = .success
                return
            case .paymentSettled(let settled) where settled.orderId == self.event.orderId:
                // Money cleared — still wait for fiscal (ADR-009).
                phase = .fiscalizing
            case .fiscalizing(let orderId) where orderId == self.event.orderId:
                phase = .fiscalizing
            case .orderStatusChanged(let orderId, let state) where orderId == self.event.orderId:
                let st = state.uppercased()
                if st == "FISCALIZING" {
                    phase = .fiscalizing
                } else if st == "COMPLETED" {
                    phase = .success
                    return
                }
            default:
                continue
            }
        }
    }

    // MARK: - Helpers

    private var formattedAmount: String {
        let uzs = Double(event.amountUzs)
        if uzs >= 100 {
            return String(format: "%.0f", uzs)
        }
        return "\(event.amountUzs)"
    }

    private var formattedOriginalAmount: String {
        let uzs = Double(event.originalAmountUzs)
        if uzs >= 100 {
            return String(format: "%.0f", uzs)
        }
        return "\(event.originalAmountUzs)"
    }

    private var cardGatewayOptions: [CardGatewayOption] {
        let gateways = filterRetailerCardGateways(
            incoming: event.availableCardGateways,
            catalog: catalogCodes
        )

        return gateways.compactMap { gateway in
            switch gateway {
            case "GLOBAL_PAY":
                CardGatewayOption(gateway: gateway, label: "GlobalPay", description: "Pay via GlobalPay checkout")
            default:
                nil
            }
        }
    }

    private func fetchPaymentCatalog() async {
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
    }
}

// MARK: - Payment Option Button

private struct PaymentOptionButton: View {
    let icon: String
    let label: String
    let description: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 12) {
                ZStack {
                    Circle()
                        .fill(AppTheme.surfaceElevated)
                        .frame(width: 40, height: 40)
                    Image(systemName: icon)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(label)
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text(description)
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                Image(systemName: "chevron.right")
                    .font(.system(size: 12, weight: .semibold))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(12)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(AppTheme.separator, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Card Checkout Response

private struct CardCheckoutResponse: Decodable {
    let orderId: String
    let state: String
    let amountUzs: Int
    let gateway: String
    let resolvedGateway: String?
    let policySource: String?
    let allowedGateways: [String]?
    let policyReason: String?
    let paymentUrl: String
    let invoiceId: String
    let sessionId: String?
    let attemptId: String?
    let attemptNo: Int?
    let message: String

    enum CodingKeys: String, CodingKey {
        case orderId = "order_id"
        case state
        case amountUzs = "amount"
        case gateway
        case resolvedGateway = "resolved_gateway"
        case policySource = "policy_source"
        case allowedGateways = "allowed_gateways"
        case policyReason = "policy_reason"
        case paymentUrl = "payment_url"
        case invoiceId = "invoice_id"
        case sessionId = "session_id"
        case attemptId = "attempt_id"
        case attemptNo = "attempt_no"
        case message
    }
}

// MARK: - DiscardableCodable Helper

private struct DiscardableCodable: Decodable {
    init(from decoder: Decoder) throws {
        // Accept any JSON value
        _ = try decoder.singleValueContainer()
    }
}

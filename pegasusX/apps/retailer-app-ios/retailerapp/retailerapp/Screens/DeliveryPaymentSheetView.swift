import SwiftUI

// MARK: - Delivery Payment Sheet

struct DeliveryPaymentSheetView: View {
    let event: PaymentRequiredEvent
    let onDismiss: () -> Void

    @State private var vm = DeliveryPaymentViewModel()

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                switch vm.phase {
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
                    if vm.phase == .choose || vm.phase == .failed {
                        Button("common.action.close") { onDismiss() }
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .pegasusMinimumHitTarget()
                    }
                }
            }
        }
        .task { await vm.listenForCompletion(orderId: event.orderId) }
        .task { await vm.fetchPaymentCatalog() }
    }

    // MARK: - Choose

    private var chooseContent: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingXL) {
                Spacer(minLength: AppTheme.spacingXL)

                ZStack {
                    Circle()
                        .fill(AppTheme.warningSoft)
                        .frame(width: 80, height: 80)
                    Image(systemName: "banknote.fill")
                        .font(.title.weight(.semibold))
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
                        .font(.system(.largeTitle, design: .rounded, weight: .bold))
                        .foregroundStyle(
                            event.originalAmountUzs > 0 && event.originalAmountUzs != event.amountUzs
                                ? AppTheme.warning
                                : AppTheme.textPrimary
                        )
                    if !vm.packDisplayCurrency.isEmpty {
                        Text(vm.packDisplayCurrency)
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(String(event.orderId.suffix(6)))"))
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Amount due \(formattedAmount) \(vm.packDisplayCurrency) for order \(event.orderId.suffix(6))")

                Text("mobile_retailer.ui.choose_payment_method")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.textSecondary)

                VStack(spacing: AppTheme.spacingSM) {
                    PaymentOptionButton(
                        icon: "banknote",
                        label: "Cash on Delivery",
                        description: "Pay the driver in cash"
                    ) {
                        Haptics.selection()
                        vm.phase = .cashConfirm
                    }

                    ForEach(vm.cardGatewayOptions(availableCardGateways: event.availableCardGateways)) { option in
                        PaymentOptionButton(
                            icon: "creditcard.fill",
                            label: option.label,
                            description: option.description
                        ) {
                            Task { await vm.initiateCardCheckout(orderId: event.orderId, gateway: option.gateway) }
                        }
                    }
                }
                .padding(.horizontal, AppTheme.spacingLG)

                Spacer()
            }
        }
        .scrollDismissesKeyboard(.interactively)
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
                    .font(.title.weight(.semibold))
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
            .padding(.horizontal, AppTheme.spacingLG)
            .accessibilityElement(children: .combine)

            HStack(spacing: AppTheme.spacingMD) {
                Button {
                    Haptics.light()
                    vm.phase = .choose
                } label: {
                    Text("common.action.back")
                        .font(.system(.body, design: .rounded, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(AppTheme.surfaceElevated)
                        .foregroundStyle(AppTheme.textPrimary)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                        .pegasusMinimumHitTarget()
                }

                Button {
                    Task { await vm.confirmCash(orderId: event.orderId) }
                } label: {
                    Text("mobile_retailer.ui.confirm_cash_received")
                        .font(.system(.body, design: .rounded, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(AppTheme.warning)
                        .foregroundStyle(.white)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                        .pegasusMinimumHitTarget()
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
        .accessibilityElement(children: .combine)
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
                    .font(.largeTitle.weight(.semibold))
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
        .accessibilityElement(children: .combine)
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
        .accessibilityElement(children: .combine)
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
                    .font(.largeTitle.weight(.semibold))
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
                Haptics.light()
                onDismiss()
            } label: {
                Text("warehouse_portal.kpi_stat_card.text.done")
                    .font(.system(.body, design: .rounded, weight: .bold))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 16)
                    .background(AppTheme.success)
                    .foregroundStyle(.white)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                    .pegasusMinimumHitTarget()
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
                    .font(.largeTitle.weight(.semibold))
                    .foregroundStyle(AppTheme.destructive)
            }

            Text("notification.payment_failed.title")
                .font(.system(.title2, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)

            if let errorMessage = vm.errorMessage {
                Text(errorMessage)
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, AppTheme.spacingXL)
            }

            Spacer()

            VStack(spacing: AppTheme.spacingMD) {
                Button {
                    vm.resetToChoose()
                } label: {
                    Text("mobile_retailer.ui.try_again")
                        .font(.system(.body, design: .rounded, weight: .bold))
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(AppTheme.accent)
                        .foregroundStyle(AppTheme.cardBackground)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                        .pegasusMinimumHitTarget()
                }

                Button {
                    Haptics.light()
                    onDismiss()
                } label: {
                    Text("common.action.cancel")
                        .font(.system(.body, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                        .pegasusMinimumHitTarget()
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingXXL)
        }
    }

    // MARK: - Helpers (Pure Int64 / Integer Formatting)

    private var formattedAmount: String {
        "\(Int64(event.amountUzs))"
    }

    private var formattedOriginalAmount: String {
        "\(Int64(event.originalAmountUzs))"
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
                        .font(.body.weight(.semibold))
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
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(12)
            .background(AppTheme.cardBackground)
            .clipShape(.rect(cornerRadius: 12))
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(AppTheme.separator, lineWidth: 1)
            )
            .pegasusMinimumHitTarget()
        }
        .buttonStyle(.plain)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label), \(description)")
    }
}

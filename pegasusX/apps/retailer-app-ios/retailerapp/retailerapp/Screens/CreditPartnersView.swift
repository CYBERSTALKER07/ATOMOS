import SwiftUI

private enum CreditPaySheetMode: String, CaseIterable, Identifiable {
    case fifo
    case earmark

    var id: String { rawValue }
}

private let creditPaymentMethods = ["WALLET", "CARD", "BANK_TRANSFER", "CASH"]

struct CreditPartnersView: View {
    @State private var vm = CreditPartnersViewModel()
    @State private var showPaySheet = false
    @State private var payMode: CreditPaySheetMode = .fifo
    @State private var selectedInvoiceId: String = ""
    @State private var fifoEarmarkInvoiceId: String = ""
    @State private var amountSomInput: String = ""
    @State private var selectedMethod: String = "BANK_TRANSFER"
    @State private var paymentReference: String = ""
    @State private var amountValidationError: String?

    @State private var extensionDaysInput: String = "7"
    @State private var disputingOpeningInvoice: MigrationOpeningInvoiceWire?
    @State private var disputeReasonInput: String = ""

    var body: some View {
        List {
            if let loadError = vm.loadError {
                Section {
                    Label(loadError, systemImage: "exclamationmark.triangle.fill")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.error)
                }
            }

            Section("mobile_retailer.nasiya_opening.title") {
                if let err = vm.openingError, !err.isEmpty {
                    Label(err, systemImage: "exclamationmark.triangle.fill")
                        .font(.caption)
                        .foregroundStyle(AppTheme.error)
                }
                if let notice = vm.openingNotice, !notice.isEmpty {
                    Label(notice, systemImage: "checkmark.circle.fill")
                        .font(.caption)
                        .foregroundStyle(AppTheme.success)
                }

                if vm.openingBalances.isEmpty {
                    Text("mobile_retailer.nasiya_opening.empty")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    Label("mobile_retailer.nasiya_opening.frozen_notice", systemImage: "snowflake")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(AppTheme.warning)
                    Text("retailer_desktop.nasiya_opening.dunning_frozen_banner")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)

                    ForEach(vm.openingBalances) { inv in
                        VStack(alignment: .leading, spacing: 6) {
                            HStack(spacing: 8) {
                                Text(inv.supplierName ?? inv.supplierId)
                                    .font(.subheadline.weight(.semibold))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Text("mobile_retailer.nasiya_opening.migration_badge")
                                    .font(.caption2.weight(.bold))
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(AppTheme.warning.opacity(0.15))
                                    .foregroundStyle(AppTheme.warning)
                                    .clipShape(Capsule())
                                Spacer()
                                openingStatusBadge(for: inv)
                            }

                            HStack {
                                Text(RetailerMoney.formatWholeCurrency(inv.balanceMinor, currency: inv.effectiveCurrency))
                                    .font(.headline.monospacedDigit())
                                Spacer()
                                if let dueAt = inv.dueAt, !dueAt.isEmpty {
                                    Text(dueAt.prefix(10))
                                        .font(.caption.monospaced())
                                        .foregroundStyle(AppTheme.textSecondary)
                                }
                            }

                            if let extId = inv.externalId ?? inv.orderId, !extId.isEmpty {
                                Text("Ref: \(extId)")
                                    .font(.caption.monospaced())
                                    .foregroundStyle(AppTheme.textSecondary)
                            }

                            if inv.dunningFrozen {
                                Label("mobile_retailer.nasiya_opening.frozen_notice", systemImage: "snowflake")
                                    .font(.caption2)
                                    .foregroundStyle(AppTheme.warning)
                            }

                            if let reason = inv.disputeReason, !reason.isEmpty {
                                Text(reason)
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.error)
                            }

                            if inv.isPendingConfirmation {
                                HStack(spacing: 10) {
                                    Button {
                                        Task { await vm.confirmOpeningBalance(invoiceId: inv.invoiceId) }
                                    } label: {
                                        if vm.openingBusyInvoiceId == inv.invoiceId {
                                            ProgressView()
                                        } else {
                                            Text("mobile_retailer.nasiya_opening.confirm_balance")
                                                .font(.caption.weight(.semibold))
                                        }
                                    }
                                    .buttonStyle(.borderedProminent)
                                    .disabled(vm.openingBusyInvoiceId != nil)

                                    Button {
                                        disputeReasonInput = ""
                                        disputingOpeningInvoice = inv
                                    } label: {
                                        Text("mobile_retailer.nasiya_opening.dispute_balance")
                                            .font(.caption.weight(.semibold))
                                    }
                                    .buttonStyle(.bordered)
                                    .tint(AppTheme.error)
                                    .disabled(vm.openingBusyInvoiceId != nil)
                                }
                                .padding(.top, 2)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }

            Section("mobile_retailer.credit.suppliers_heading") {
                if vm.isLoading && vm.relationships.isEmpty {
                    HStack(spacing: 10) {
                        ProgressView()
                        Text("mobile_retailer.credit.loading")
                            .font(.subheadline)
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                } else if vm.relationships.isEmpty {
                    Text("mobile_retailer.credit.empty_suppliers")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(vm.relationships) { rel in
                        let isSelected = rel.supplierId == vm.selectedSupplierId
                        Button {
                            Task { await vm.selectSupplier(rel.supplierId) }
                        } label: {
                            VStack(alignment: .leading, spacing: 6) {
                                HStack {
                                    Text(rel.supplierName?.isEmpty == false ? (rel.supplierName ?? rel.supplierId) : rel.supplierId)
                                        .font(.headline)
                                        .foregroundStyle(AppTheme.textPrimary)
                                    Spacer()
                                    statusBadge(for: rel)
                                }

                                HStack {
                                    metricColumn(
                                        titleKey: "mobile_retailer.credit.available",
                                        value: RetailerMoney.formatWholeCurrency(rel.availableCreditMinor)
                                    )
                                    Spacer()
                                    metricColumn(
                                        titleKey: "mobile_retailer.credit.balance_due",
                                        value: RetailerMoney.formatWholeCurrency(rel.currentBalanceMinor)
                                    )
                                    Spacer()
                                    metricColumn(
                                        titleKey: "mobile_retailer.credit.limit",
                                        value: RetailerMoney.formatWholeCurrency(rel.creditLimitMinor)
                                    )
                                }

                                Text(L10n.format("mobile_retailer.credit.terms_fmt", "\(rel.effectiveTermsDays)", "\(rel.effectiveGraceDays)"))
                                    .font(.caption)
                                    .foregroundStyle(AppTheme.textTertiary)
                            }
                            .padding(.vertical, 4)
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .listRowBackground(isSelected ? AppTheme.accent.opacity(0.08) : Color.clear)
                    }
                }
            }

            if !vm.selectedSupplierId.isEmpty {
                Section("mobile_retailer.credit.statement_heading") {
                    if !vm.currentQuarterKey.isEmpty {
                        Text(L10n.format("mobile_retailer.credit.quarter_fmt", vm.currentQuarterKey))
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    HStack {
                        metricColumn(
                            titleKey: "mobile_retailer.credit.total_open",
                            value: RetailerMoney.formatWholeCurrency(vm.totalOpenMinor, currency: vm.statement?.currency)
                        )
                        Spacer()
                        metricColumn(
                            titleKey: "mobile_retailer.credit.undisputed",
                            value: RetailerMoney.formatWholeCurrency(vm.totalUndisputedMinor, currency: vm.statement?.currency)
                        )
                    }

                    HStack {
                        metricColumn(
                            titleKey: "mobile_retailer.credit.disputed",
                            value: RetailerMoney.formatWholeCurrency(vm.totalDisputedMinor, currency: vm.statement?.currency)
                        )
                        Spacer()
                        metricColumn(
                            titleKey: "mobile_retailer.credit.advance_balance",
                            value: RetailerMoney.formatWholeCurrency(vm.advanceBalanceMinor, currency: vm.statement?.currency)
                        )
                    }

                    if let paidMinor = vm.paySuccessPaidMinor {
                        Text(
                            L10n.format(
                                "mobile_retailer.credit.pay_success_fmt",
                                RetailerMoney.formatWholeCurrency(paidMinor, currency: vm.statement?.currency),
                                RetailerMoney.formatWholeCurrency(vm.paySuccessAdvanceMinor, currency: vm.statement?.currency)
                            )
                        )
                        .font(.footnote)
                        .foregroundStyle(AppTheme.success)
                    }

                    if let payError = vm.payError {
                        Text(payError)
                            .font(.footnote)
                            .foregroundStyle(AppTheme.error)
                    }

                    Button {
                        payMode = .fifo
                        fifoEarmarkInvoiceId = ""
                        selectedInvoiceId = vm.invoices.first?.invoiceId ?? ""
                        amountSomInput = vm.totalUndisputedMinor > 0
                            ? RetailerMoney.minorToSomInput(vm.totalUndisputedMinor)
                            : ""
                        amountValidationError = nil
                        showPaySheet = true
                    } label: {
                        HStack {
                            Spacer()
                            Text("mobile_retailer.credit.pay_fifo_cta")
                                .fontWeight(.semibold)
                            Spacer()
                        }
                    }
                    .buttonStyle(.borderedProminent)
                }

                Section("mobile_retailer.credit.promises_heading") {
                    let prepayLocked = vm.activeRelationship?.prepayOnlyLocked == true
                    HStack {
                        Text(vm.selfServePromiseUsed ? "mobile_retailer.credit.promise_used" : "mobile_retailer.credit.promise_available")
                            .font(.caption.weight(.semibold))
                            .foregroundStyle(vm.selfServePromiseUsed ? AppTheme.error : AppTheme.success)
                        Spacer()
                    }

                    if let limitErr = vm.quarterlyLimitError {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("mobile_retailer.credit.quarterly_limit_title")
                                .font(.footnote.weight(.semibold))
                                .foregroundStyle(AppTheme.error)
                            Text(
                                L10n.format(
                                    "mobile_retailer.credit.quarterly_limit_detail_fmt",
                                    limitErr.quarterKey ?? (vm.currentQuarterKey.isEmpty ? "—" : vm.currentQuarterKey)
                                )
                            )
                            .font(.caption)
                            .foregroundStyle(AppTheme.error)
                        }
                    }

                    if prepayLocked {
                        Text("mobile_retailer.credit.promise_prepay_blocked")
                            .font(.caption)
                            .foregroundStyle(AppTheme.error)
                    }

                    if let promErr = vm.promiseError {
                        Text(promErr)
                            .font(.caption)
                            .foregroundStyle(AppTheme.error)
                    }

                    if vm.promiseCreated {
                        Text("mobile_retailer.credit.promise_created")
                            .font(.caption)
                            .foregroundStyle(AppTheme.success)
                    }

                    HStack {
                        TextField("mobile_retailer.credit.extension_days", text: $extensionDaysInput)
                            .keyboardType(.numberPad)
                            .textFieldStyle(.roundedBorder)
                        Button {
                            let maxDays = max(1, vm.maxExtensionDays)
                            let days = min(maxDays, max(1, Int(extensionDaysInput.trimmingCharacters(in: .whitespaces)) ?? 7))
                            Task {
                                _ = await vm.createPromiseExtension(
                                    supplierId: vm.selectedSupplierId,
                                    extensionDays: days
                                )
                            }
                        } label: {
                            Text("mobile_retailer.credit.request_extension")
                        }
                        .disabled(vm.isCreatingPromise || prepayLocked)
                    }

                    if vm.promises.isEmpty {
                        Text("mobile_retailer.credit.no_promises")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)
                    } else {
                        ForEach(vm.promises) { prom in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text("\(prom.quarterKey) · \(prom.dueAt)")
                                        .font(.caption.weight(.medium))
                                    if let inv = prom.invoiceId, !inv.isEmpty {
                                        Text(inv)
                                            .font(.caption2)
                                            .foregroundStyle(AppTheme.textTertiary)
                                    }
                                }
                                Spacer()
                                Text(RetailerMoney.formatWholeCurrency(prom.promisedMinor, currency: vm.statement?.currency))
                                    .font(.caption.monospacedDigit().weight(.semibold))
                            }
                        }
                    }
                }
            }

            Section("mobile_retailer.credit.invoices_heading") {
                if vm.invoices.isEmpty {
                    Text("mobile_retailer.credit.no_invoices")
                        .font(.subheadline)
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    ForEach(vm.invoices) { inv in
                        let undisputed = inv.effectiveUndisputedMinor
                        VStack(alignment: .leading, spacing: 6) {
                            HStack {
                                Text("\(inv.orderId.map { "#\($0)" } ?? inv.invoiceId) · \(inv.status)")
                                    .font(.headline)
                                Spacer()
                                Text(RetailerMoney.formatWholeCurrency(inv.balanceMinor, currency: inv.currency))
                                    .font(.subheadline.monospacedDigit().weight(.semibold))
                            }

                            HStack(spacing: 8) {
                                Text(L10n.format("mobile_retailer.credit.due_fmt", String(inv.dueAt.prefix(10))))
                                if let aging = inv.agingBucket, !aging.isEmpty {
                                    Text(L10n.format("mobile_retailer.credit.aging_fmt", aging))
                                }
                            }
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)

                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    Text(
                                        L10n.format(
                                            "mobile_retailer.credit.undisputed_fmt",
                                            RetailerMoney.formatWholeCurrency(undisputed, currency: inv.currency)
                                        )
                                    )
                                    .font(.caption2)
                                    if inv.disputedMinor > 0 {
                                        Text(
                                            L10n.format(
                                                "mobile_retailer.credit.disputed_fmt",
                                                RetailerMoney.formatWholeCurrency(inv.disputedMinor, currency: inv.currency)
                                            )
                                        )
                                        .font(.caption2)
                                        .foregroundStyle(AppTheme.error)
                                    }
                                }
                                Spacer()
                                Button("mobile_retailer.credit.pay_invoice_cta") {
                                    payMode = .earmark
                                    selectedInvoiceId = inv.invoiceId
                                    let suggested = undisputed > 0 ? undisputed : inv.balanceMinor
                                    amountSomInput = suggested > 0 ? RetailerMoney.minorToSomInput(suggested) : ""
                                    amountValidationError = nil
                                    showPaySheet = true
                                }
                                .buttonStyle(.bordered)
                            }
                        }
                        .padding(.vertical, 4)
                    }
                }
            }
        }
        .navigationTitle("mobile_retailer.credit.title")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("mobile_retailer.credit.refresh") {
                    Task { await vm.load() }
                }
                .disabled(vm.isLoading || vm.isRefreshingDetail)
            }
        }
        .sheet(isPresented: $showPaySheet) {
            NavigationStack {
                Form {
                    Picker("", selection: $payMode) {
                        Text("mobile_retailer.credit.pay_mode_fifo").tag(CreditPaySheetMode.fifo)
                        Text("mobile_retailer.credit.pay_mode_earmark").tag(CreditPaySheetMode.earmark)
                    }
                    .pickerStyle(.segmented)

                    if payMode == .earmark {
                        Picker("mobile_retailer.credit.target_invoice", selection: $selectedInvoiceId) {
                            ForEach(vm.invoices) { inv in
                                Text("\(inv.orderId ?? inv.invoiceId) — \(RetailerMoney.formatWholeCurrency(inv.balanceMinor, currency: inv.currency))")
                                    .tag(inv.invoiceId)
                            }
                        }
                    } else if !vm.invoices.isEmpty {
                        Picker("mobile_retailer.credit.earmark_optional", selection: $fifoEarmarkInvoiceId) {
                            Text("mobile_retailer.credit.fifo_strict").tag("")
                            ForEach(vm.invoices) { inv in
                                Text("\(inv.orderId ?? inv.invoiceId) (\(RetailerMoney.formatWholeCurrency(inv.balanceMinor, currency: inv.currency)))")
                                    .tag(inv.invoiceId)
                            }
                        }
                    }

                    Section("mobile_retailer.credit.amount_som") {
                        TextField("500000", text: $amountSomInput)
                            .keyboardType(.numberPad)
                        if let err = amountValidationError {
                            Text(err)
                                .font(.caption)
                                .foregroundStyle(AppTheme.error)
                        }
                    }

                    Section {
                        Picker("mobile_retailer.credit.method", selection: $selectedMethod) {
                            ForEach(creditPaymentMethods, id: \.self) { method in
                                Text(methodLocalizedTitle(method)).tag(method)
                            }
                        }
                        TextField("mobile_retailer.credit.reference_optional", text: $paymentReference)
                    }
                }
                .navigationTitle("mobile_retailer.credit.pay_sheet_title")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("mobile_retailer.credit.cancel") {
                            showPaySheet = false
                        }
                        .disabled(vm.isSubmittingPayment)
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("mobile_retailer.credit.submit_pay") {
                            guard let minor = RetailerMoney.somInputToMinor(amountSomInput), minor > 0 else {
                                amountValidationError = String(localized: "mobile_retailer.credit.invalid_amount")
                                return
                            }
                            Task {
                                let ok: Bool
                                if payMode == .earmark && !selectedInvoiceId.isEmpty {
                                    ok = await vm.payInvoice(
                                        invoiceId: selectedInvoiceId,
                                        amountMinor: minor,
                                        paymentMethod: selectedMethod,
                                        paymentReference: paymentReference
                                    )
                                } else {
                                    ok = await vm.payFifo(
                                        supplierId: vm.selectedSupplierId,
                                        amountMinor: minor,
                                        paymentMethod: selectedMethod,
                                        paymentReference: paymentReference,
                                        earmarkInvoiceId: fifoEarmarkInvoiceId.isEmpty ? nil : fifoEarmarkInvoiceId
                                    )
                                }
                                if ok {
                                    showPaySheet = false
                                    paymentReference = ""
                                }
                            }
                        }
                        .disabled(vm.isSubmittingPayment || (payMode == .earmark && vm.invoices.isEmpty))
                    }
                }
            }
        }
        .sheet(item: $disputingOpeningInvoice) { inv in
            NavigationStack {
                Form {
                    Section("mobile_retailer.nasiya_opening.title") {
                        Text(inv.supplierName ?? inv.supplierId)
                            .font(.subheadline.weight(.semibold))
                        Text(RetailerMoney.formatWholeCurrency(inv.balanceMinor, currency: inv.effectiveCurrency))
                            .font(.headline.monospacedDigit())
                    }
                    Section("retailer_desktop.nasiya_opening.dispute_reason_label") {
                        TextField(
                            String(localized: "mobile_retailer.nasiya_opening.dispute_hint"),
                            text: $disputeReasonInput,
                            axis: .vertical
                        )
                        .lineLimit(3...6)
                    }
                }
                .navigationTitle("mobile_retailer.nasiya_opening.dispute_balance")
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("mobile_retailer.credit.cancel") {
                            disputingOpeningInvoice = nil
                        }
                        .disabled(vm.openingBusyInvoiceId != nil)
                    }
                    ToolbarItem(placement: .confirmationAction) {
                        Button("retailer_desktop.nasiya_opening.submit_dispute") {
                            Task {
                                let ok = await vm.disputeOpeningBalance(
                                    invoiceId: inv.invoiceId,
                                    reason: disputeReasonInput
                                )
                                if ok {
                                    disputingOpeningInvoice = nil
                                    disputeReasonInput = ""
                                }
                            }
                        }
                        .disabled(
                            vm.openingBusyInvoiceId != nil
                                || disputeReasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty
                        )
                    }
                }
            }
        }
        .task {
            await vm.load()
        }
    }

    @ViewBuilder
    private func openingStatusBadge(for inv: MigrationOpeningInvoiceWire) -> some View {
        if inv.isAcknowledged {
            Text("retailer_desktop.nasiya_opening.acknowledged_status")
                .font(.caption2.weight(.bold))
                .foregroundStyle(AppTheme.success)
        } else if inv.isDisputed {
            Text("retailer_desktop.nasiya_opening.disputed_status")
                .font(.caption2.weight(.bold))
                .foregroundStyle(AppTheme.error)
        } else {
            Text("retailer_desktop.nasiya_opening.pending_status")
                .font(.caption2.weight(.bold))
                .foregroundStyle(AppTheme.warning)
        }
    }

    @ViewBuilder
    private func metricColumn(titleKey: LocalizedStringKey, value: String) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(titleKey)
                .font(.caption2)
                .foregroundStyle(AppTheme.textTertiary)
            Text(value)
                .font(.subheadline.monospacedDigit().weight(.semibold))
                .foregroundStyle(AppTheme.textPrimary)
        }
    }

    @ViewBuilder
    private func statusBadge(for rel: CreditRelationshipWire) -> some View {
        if rel.onHold {
            Text("mobile_retailer.credit.on_hold")
                .font(.caption2.weight(.bold))
                .foregroundStyle(AppTheme.error)
        } else if rel.prepayOnlyLocked {
            Text("mobile_retailer.credit.prepay_only")
                .font(.caption2.weight(.bold))
                .foregroundStyle(AppTheme.warning)
        } else {
            Text("mobile_retailer.credit.active")
                .font(.caption2.weight(.semibold))
                .foregroundStyle(AppTheme.success)
        }
    }

    private func methodLocalizedTitle(_ method: String) -> LocalizedStringKey {
        switch method {
        case "WALLET": return "mobile_retailer.credit.method_wallet"
        case "CARD": return "mobile_retailer.credit.method_card"
        case "BANK_TRANSFER": return "mobile_retailer.credit.method_bank"
        default: return "mobile_retailer.credit.method_cash"
        }
    }
}

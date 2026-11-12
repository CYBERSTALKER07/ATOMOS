import SwiftData
import SwiftUI

struct CheckoutView: View {
    var supplierIsActive: Bool = true

    @Environment(CartManager.self) private var cart
    @Environment(\.dismiss) private var dismiss
    @Environment(\.modelContext) private var modelContext

    @State private var vm = CheckoutViewModel()

    var body: some View {
        @Bindable var vm = vm

        ZStack {
            AppTheme.background.ignoresSafeArea()

            if vm.showSuccess {
                successView
                    .transition(.opacity.combined(with: .scale(scale: 0.9)))
            } else {
                checkoutContent(deliveryDateBinding: $vm.deliveryDate, expressPriorityBinding: $vm.expressPriority)
                    .transition(.opacity)
            }
        }
        .animation(AnimationConstants.fluid, value: vm.showSuccess)
        .navigationTitle("mobile_retailer.ui.checkout")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .cancellationAction) {
                Button {
                    Haptics.light()
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                        .frame(width: 30, height: 30)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.circle)
                        .pegasusMinimumHitTarget()
                }
                .accessibilityLabel("Close")
            }
        }
        .task {
            await vm.loadPaymentMethods()
        }
        .sheet(isPresented: $vm.showPaymentPicker) {
            NavigationStack {
                CheckoutPaymentPickerSheet(
                    paymentOptions: vm.paymentOptions,
                    selectedPaymentId: vm.selectedPaymentId,
                    onSelect: { option in
                        vm.selectPaymentOption(option)
                    }
                )
            }
            .presentationDetents([.medium])
            .presentationDragIndicator(.visible)
        }
        .alert("Order Failed", isPresented: $vm.showError) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(vm.errorMessage)
        }
        .confirmationDialog(
            "Supplier is Currently Closed",
            isPresented: $vm.showSupplierClosedWarning,
            titleVisibility: .visible
        ) {
            Button("mobile_retailer.ui.i_understand_place_order") {
                Task { await vm.submitOrder(cart: cart, modelContext: modelContext) }
            }
            Button("common.action.cancel", role: .cancel) {}
        } message: {
            Text("mobile_retailer.ui.this_supplier_is_off_shift_or_outside_business_hours_your_order_")
        }
        .confirmationDialog(
            "Partial backorder",
            isPresented: $vm.showBackorderConfirm,
            titleVisibility: .visible
        ) {
            Button("mobile_retailer.ui.proceed") {
                vm.skipBackorderConfirm = true
                Task { await vm.submitOrder(cart: cart, modelContext: modelContext) }
            }
            Button("common.action.cancel", role: .cancel) {
                vm.pendingBackorderPreview = nil
            }
        } message: {
            Text("mobile_retailer.ui.some_items_will_be_backordered_delivery_may_be_delayed_proceed")
        }
        .task {
            await vm.fetchPreview(cart: cart)
        }
        .onChange(of: cart.totalItems) { _, _ in
            Task { await vm.fetchPreview(cart: cart) }
        }
    }

    // MARK: - Checkout Content

    private func checkoutContent(
        deliveryDateBinding: Binding<String>,
        expressPriorityBinding: Binding<Bool>
    ) -> some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
                    if vm.previewLoading {
                        HStack(spacing: AppTheme.spacingSM) {
                            ProgressView()
                            Text("mobile_retailer.ui.refreshing_stock_availability")
                                .font(.caption)
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        .accessibilityElement(children: .combine)
                    }
                    if !vm.stockWarnings.isEmpty {
                        stockWarningsSection
                    }
                    if !vm.oosItems.isEmpty {
                        oosItemsSection
                    }
                    deliverySection(
                        deliveryDateBinding: deliveryDateBinding,
                        expressPriorityBinding: expressPriorityBinding
                    ).slideIn(delay: 0)
                    cartItemsSection.slideIn(delay: 0)
                    paymentSection.slideIn(delay: 0.05)
                    summarySection.slideIn(delay: 0.1)
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, 100)
            }
            .scrollDismissesKeyboard(.interactively)
            .scrollIndicators(.hidden)

            submitButton
        }
    }

    // MARK: - Stock Warnings

    private var stockWarningsSection: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Label("mobile_retailer.ui.partial_backorder", systemImage: "exclamationmark.triangle.fill")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.warning)
                Text("mobile_retailer.ui.some_items_are_out_of_stock_but_your_warehouse_accepts_backorder")
                    .font(.caption)
                    .foregroundStyle(AppTheme.textSecondary)
                ForEach(vm.stockWarnings, id: \.sku) { warning in
                    Text(L10n.format("mobile_retailer.ui.sku_backorderqty_of_requested_backordered", "\(warning.sku)", "\(warning.backorderQty)", "\(warning.requested)"))
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(AppTheme.textPrimary)
                }
            }
            .padding(AppTheme.spacingLG)
            .accessibilityElement(children: .combine)
        }
    }

    private var oosItemsSection: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                Label("mobile_retailer.ui.out_of_stock", systemImage: "xmark.circle.fill")
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .foregroundStyle(AppTheme.destructive)
                ForEach(vm.oosItems, id: \.self) { sku in
                    Text(sku)
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(AppTheme.textPrimary)
                }
            }
            .padding(AppTheme.spacingLG)
            .accessibilityElement(children: .combine)
        }
    }

    private func deliverySection(
        deliveryDateBinding: Binding<String>,
        expressPriorityBinding: Binding<Bool>
    ) -> some View {
        LabCardWithHeader(title: "Delivery", icon: "truck.box.fill") {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(spacing: AppTheme.spacingSM) {
                    deliveryModeButton("Standard", mode: "STANDARD", subtitle: "ASAP")
                    deliveryModeButton("Scheduled", mode: "SCHEDULED", subtitle: "T+\(vm.preview?.preorderMinLeadDays ?? 3)")
                }
                if vm.deliveryMode == "SCHEDULED" {
                    TextField("mobile_retailer.ui.yyyy_mm_dd", text: deliveryDateBinding)
                        .textFieldStyle(.roundedBorder)
                        .font(.system(.caption, design: .monospaced))
                        .pegasusMinimumHitTarget()
                        .accessibilityLabel("Scheduled delivery date")
                    Text("Choose \(vm.scheduledMinDate)\(vm.scheduledMaxDate.map { " to \($0)" } ?? " or later")")
                        .font(.caption2)
                        .foregroundStyle(AppTheme.textTertiary)
                }
                Toggle("Express priority (+fee)", isOn: expressPriorityBinding)
                    .font(.system(.caption, design: .rounded))
                    .onChange(of: vm.expressPriority) { _, _ in
                        Haptics.selection()
                        Task { await vm.fetchPreview(cart: cart) }
                    }
                if let deliveryFee = vm.preview?.deliveryFeeMinor, deliveryFee > 0 {
                    Text("Delivery fee: \(deliveryFee.formatted()) \(vm.packDisplayCurrency)\(vm.preview?.deliveryDistanceKm.map { " · \(String(format: "%.1f", $0)) km" } ?? "")")
                        .font(.caption)
                        .foregroundStyle(AppTheme.textSecondary)
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func deliveryModeButton(_ title: String, mode: String, subtitle: String) -> some View {
        Button {
            vm.selectDeliveryMode(mode)
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
            .pegasusMinimumHitTarget()
            .background(vm.deliveryMode == mode ? AppTheme.accentSoft.opacity(0.45) : AppTheme.surfaceElevated)
            .clipShape(RoundedRectangle(cornerRadius: AppTheme.radiusSM))
            .overlay(
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .stroke(vm.deliveryMode == mode ? AppTheme.accent : AppTheme.separator.opacity(0.4), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title) delivery, \(subtitle)\(vm.deliveryMode == mode ? ", selected" : "")")
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
                            .font(.title2)
                            .foregroundStyle(AppTheme.accent.opacity(0.5))
                    }
                    Text("mobile_retailer.ui.your_cart_is_empty")
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, AppTheme.spacingXL)
                .accessibilityElement(children: .combine)
            } else {
                VStack(spacing: 0) {
                    ForEach(Array(cart.items.enumerated()), id: \.element.id) { index, item in
                        if index > 0 {
                            Rectangle()
                                .fill(AppTheme.separator.opacity(0.3))
                                .frame(height: AppTheme.separatorHeight)
                                .padding(.horizontal, AppTheme.spacingXS)
                        }
                        CheckoutCartItemRowView(
                            item: item,
                            isOos: vm.isOos(item),
                            maximumQuantity: cart.maxQuantity(for: item, preview: vm.preview),
                            onUpdateQuantity: { newQty in
                                Haptics.selection()
                                cart.updateQuantity(itemId: item.id, quantity: newQty)
                            }
                        )
                    }
                }
            }
        }
    }

    // MARK: - Payment

    private var paymentSection: some View {
        LabCardWithHeader(title: "Payment", icon: "creditcard.fill") {
            Button {
                Haptics.light()
                vm.showPaymentPicker = true
            } label: {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                            .fill(AppTheme.infoSoft.opacity(0.5))
                            .frame(width: 36, height: 36)
                        Image(systemName: "creditcard")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(AppTheme.info)
                    }

                    Text(vm.selectedPaymentLabel)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)

                    Spacer()

                    Text("mobile_retailer.ui.change")
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.accent)
                }
                .pegasusMinimumHitTarget()
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("Payment method: \(vm.selectedPaymentLabel). Activate to change.")
        }
    }

    // MARK: - Summary

    private var summarySection: some View {
        LabCard {
            VStack(spacing: AppTheme.spacingMD) {
                summaryRow("Subtotal", value: "\(cart.checkoutSubtotalMinor.formatted())")
                if cart.checkoutDiscountMinor > 0 {
                    summaryRow(
                        "Promotion",
                        value: "-\(cart.checkoutDiscountMinor.formatted())",
                        valueColor: AppTheme.success
                    )
                }
                summaryRow("Delivery", value: "Free", valueColor: AppTheme.success)

                Rectangle()
                    .fill(AppTheme.separator.opacity(0.5))
                    .frame(height: AppTheme.separatorHeight)

                HStack {
                    Text("retailer_desktop.pos.text.total")
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)
                    Spacer()
                    AnimatedCurrencyText(
                        value: cart.checkoutTotalMinor,
                        font: .system(.title3, design: .rounded, weight: .bold)
                    )
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Total \(cart.checkoutTotalMinor.formatted())")
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
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value)")
    }

    // MARK: - Submit Button

    private var submitButton: some View {
        VStack(spacing: 0) {
            Rectangle()
                .fill(AppTheme.separator.opacity(0.3))
                .frame(height: AppTheme.separatorHeight)

            LabButton("Place Order", icon: "checkmark.circle", fullWidth: true) {
                Haptics.medium()
                if !supplierIsActive {
                    vm.showSupplierClosedWarning = true
                } else {
                    Task { await vm.submitOrder(cart: cart, modelContext: modelContext) }
                }
            }
            .padding(AppTheme.spacingLG)
            .opacity(cart.isEmpty || vm.isSubmitting || vm.previewLoading ? 0.5 : 1.0)
            .disabled(cart.isEmpty || vm.isSubmitting || vm.previewLoading)
            .accessibilityLabel("Place order")
        }
        .background(.ultraThinMaterial)
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
                    .font(.largeTitle)
                    .imageScale(.large)
                    .foregroundStyle(AppTheme.success)
                    .symbolEffect(.bounce, value: vm.showSuccess)
            }
            .accessibilityHidden(true)

            VStack(spacing: AppTheme.spacingSM) {
                Text("mobile_retailer.ui.order_placed")
                    .font(.system(.title2, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)

                Text(vm.stockWarnings.isEmpty
                     ? "Your order has been submitted successfully.\nYou'll receive updates in your inbox."
                     : "Your order was placed with partial backorder items.\nFulfillment will continue when stock arrives.")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .multilineTextAlignment(.center)
            }
            .accessibilityElement(children: .combine)

            Spacer()

            LabButton("Done", icon: "checkmark", fullWidth: true) {
                Haptics.light()
                dismiss()
            }
            .padding(.horizontal, AppTheme.spacingXL)
            .padding(.bottom, AppTheme.spacingXL)
            .accessibilityLabel("Done")
        }
    }
}

private struct CheckoutCartItemRowView: View {
    let item: CartItem
    let isOos: Bool
    let maximumQuantity: Int
    let onUpdateQuantity: (Int) -> Void

    @State private var quantity: Int

    init(
        item: CartItem,
        isOos: Bool,
        maximumQuantity: Int,
        onUpdateQuantity: @escaping (Int) -> Void
    ) {
        self.item = item
        self.isOos = isOos
        self.maximumQuantity = maximumQuantity
        self.onUpdateQuantity = onUpdateQuantity
        self._quantity = State(initialValue: item.quantity)
    }

    var body: some View {
        HStack(spacing: AppTheme.spacingMD) {
            ZStack {
                RoundedRectangle(cornerRadius: AppTheme.radiusSM)
                    .fill(AppTheme.accentSoft.opacity(0.3))
                    .frame(width: 40, height: 40)
                Image(systemName: "leaf.fill")
                    .font(.callout)
                    .foregroundStyle(AppTheme.accent.opacity(0.6))
            }
            .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 2) {
                HStack(spacing: AppTheme.spacingSM) {
                    Text(item.product.name)
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                        .lineLimit(1)
                    if isOos {
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
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(item.product.name), \(item.variant.size)\(isOos ? ", out of stock" : "")")

            Spacer()

            QuantityStepper(
                quantity: $quantity,
                maximum: maximumQuantity,
                compact: true
            )
            .onChange(of: quantity) { _, newVal in
                guard newVal != item.quantity else { return }
                onUpdateQuantity(newVal)
            }
            .onChange(of: item.quantity) { _, updated in
                if quantity != updated {
                    quantity = updated
                }
            }

            Text("\(item.totalPriceMinor.formatted())")
                .font(.system(.caption, design: .rounded, weight: .bold))
                .monospacedDigit()
                .foregroundStyle(AppTheme.accent)
                .frame(width: 70, alignment: .trailing)
        }
        .padding(.vertical, AppTheme.spacingSM)
    }
}

#Preview {
    NavigationStack {
        CheckoutView()
            .environment(CartManager())
    }
}

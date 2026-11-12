import SwiftData
import SwiftUI

struct ProcurementView: View {
    @Environment(CartManager.self) private var cart
    @Environment(\.modelContext) private var modelContext
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = ProcurementViewModel()

    var body: some View {
        @Bindable var vm = vm

        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
                    headerSection.slideIn(delay: 0)
                    suggestionsSection.slideIn(delay: 0.05)

                    if !vm.selectedItems.isEmpty {
                        selectedSummary.slideIn(delay: 0.1)
                    }
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, vm.selectedItems.isEmpty ? AppTheme.spacingXXL : 100)
            }
            .scrollDismissesKeyboard(.interactively)
            .scrollIndicators(.hidden)

            if !vm.selectedItems.isEmpty {
                actionBar
            }
        }
        .background(AppTheme.background)
        .task { await vm.loadPredictions() }
        .task(id: refreshCenter.refreshToken) { await vm.loadPredictions() }
        .alert("Order Created", isPresented: $vm.showSuccess) {
            Button("OK") { vm.acknowledgeSuccess() }
        } message: {
            Text("mobile_retailer.ui.your_procurement_order_has_been_submitted_successfully")
        }
        .alert("Order Failed", isPresented: $vm.showError) {
            Button("common.action.retry") {
                Task { await vm.createOrder(modelContext: modelContext, refreshCenter: refreshCenter) }
            }
            Button("common.action.cancel", role: .cancel) {}
        } message: {
            Text(vm.errorMessage)
        }
    }

    // MARK: - Header

    private var headerSection: some View {
        GradientHeaderCard(title: "AI Procurement", subtitle: "Smart suggestions based on demand analysis", icon: "sparkles") {
            HStack(spacing: AppTheme.spacingXL) {
                VStack(spacing: 3) {
                    Text("\(vm.forecasts.count)")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("mobile_retailer.ui.suggestions")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .frame(maxWidth: .infinity)
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Suggestions: \(vm.forecasts.count)")

                VStack(spacing: 3) {
                    Text("\(vm.selectedItems.count)")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.accent)
                        .contentTransition(.numericText())
                    Text("common.action.selected")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }
                .frame(maxWidth: .infinity)
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Selected: \(vm.selectedItems.count)")
            }
        }
    }

    // MARK: - Suggestions

    private var suggestionsSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "lightbulb.fill")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(AppTheme.accent)
                    Text("mobile_retailer.ui.suggestions")
                        .font(.system(.headline, design: .rounded))
                        .foregroundStyle(AppTheme.textPrimary)
                }
                Spacer()
                if !vm.forecasts.isEmpty {
                    Button {
                        vm.toggleSelectAll()
                    } label: {
                        Text(vm.selectedItems.count == vm.forecasts.count ? "Deselect All" : "Select All")
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                            .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel(vm.selectedItems.count == vm.forecasts.count ? "Deselect all suggestions" : "Select all suggestions")
                }
            }

            ForEach(Array(vm.forecasts.enumerated()), id: \.element.id) { index, forecast in
                ProcurementSuggestionCardView(
                    forecast: forecast,
                    isSelected: vm.selectedItems.contains(forecast.id),
                    externalQuantity: vm.quantities[forecast.id] ?? forecast.predictedQuantity,
                    onToggleSelection: {
                        vm.toggleSelection(for: forecast)
                    },
                    onUpdateQuantity: { newQty in
                        vm.updateQuantity(for: forecast.id, quantity: newQty)
                    }
                )
                .staggeredSlideIn(index: index)
            }
        }
    }

    // MARK: - Summary

    private var selectedSummary: some View {
        LabCard {
            HStack {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundStyle(AppTheme.accent)
                    Text(L10n.format("mobile_retailer.ui.count_items_selected", "\(vm.selectedItems.count)"))
                        .font(.system(.subheadline, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textPrimary)
                }
                Spacer()
                AnimatedNumberText(
                    value: vm.selectedTotalUnits,
                    font: .system(.subheadline, design: .rounded, weight: .bold),
                    color: AppTheme.accent
                )
                Text("retailer_desktop.auto_order.auto_order_list.text.units")
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
            }
            .padding(AppTheme.spacingLG)
            .accessibilityElement(children: .combine)
            .accessibilityLabel("\(vm.selectedItems.count) items selected, \(vm.selectedTotalUnits) units")
        }
    }

    // MARK: - Action Bar

    private var actionBar: some View {
        VStack(spacing: 0) {
            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)
            HStack(spacing: AppTheme.spacingMD) {
                LabButton("Create Order", icon: "cart.badge.plus") {
                    Haptics.medium()
                    Task { await vm.createOrder(modelContext: modelContext, refreshCenter: refreshCenter) }
                }
                .opacity(vm.isSubmitting ? 0.5 : 1)
                .disabled(vm.isSubmitting)
                .accessibilityLabel("Create procurement order")

                LabButton("Add to Cart", variant: .secondary, icon: "cart") {
                    vm.addToCart(cart: cart)
                }
                .accessibilityLabel("Add selected suggestions to cart")
            }
            .padding(AppTheme.spacingLG)
            .background(.ultraThinMaterial)
        }
        .transition(.move(edge: .bottom).combined(with: .opacity))
    }
}

private struct ProcurementSuggestionCardView: View {
    let forecast: DemandForecast
    let isSelected: Bool
    let externalQuantity: Int
    let onToggleSelection: () -> Void
    let onUpdateQuantity: (Int) -> Void

    @State private var localQuantity: Int

    init(
        forecast: DemandForecast,
        isSelected: Bool,
        externalQuantity: Int,
        onToggleSelection: @escaping () -> Void,
        onUpdateQuantity: @escaping (Int) -> Void
    ) {
        self.forecast = forecast
        self.isSelected = isSelected
        self.externalQuantity = externalQuantity
        self.onToggleSelection = onToggleSelection
        self.onUpdateQuantity = onUpdateQuantity
        self._localQuantity = State(initialValue: externalQuantity)
    }

    var body: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack {
                    Button(action: onToggleSelection) {
                        ZStack {
                            RoundedRectangle(cornerRadius: 6)
                                .fill(isSelected ? AppTheme.accent : AppTheme.separator.opacity(0.3))
                                .frame(width: 24, height: 24)
                            if isSelected {
                                Image(systemName: "checkmark")
                                    .font(.caption.weight(.bold))
                                    .foregroundStyle(.white)
                            }
                        }
                        .pegasusMinimumHitTarget()
                    }
                    .accessibilityLabel(isSelected ? "Deselect \(forecast.productName)" : "Select \(forecast.productName)")

                    VStack(alignment: .leading, spacing: 2) {
                        Text(forecast.productName)
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text(L10n.format("mobile_retailer.ui.confidence_confidencepercent", "\(forecast.confidencePercent)"))
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .accessibilityElement(children: .combine)

                    Spacer()

                    if isSelected {
                        QuantityStepper(quantity: $localQuantity, compact: true)
                            .onChange(of: localQuantity) { _, newVal in
                                guard newVal != externalQuantity else { return }
                                onUpdateQuantity(newVal)
                            }
                            .onChange(of: externalQuantity) { _, updated in
                                if localQuantity != updated {
                                    localQuantity = updated
                                }
                            }
                    } else {
                        Text(L10n.format("mobile_retailer.ui.predictedquantity_units", "\(forecast.predictedQuantity)"))
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                    }
                }

                Text(forecast.reasoning)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .lineLimit(2)
            }
            .padding(AppTheme.spacingLG)
        }
        .overlay {
            if isSelected {
                RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                    .strokeBorder(AppTheme.accent.opacity(0.4), lineWidth: 2)
            }
        }
    }
}

#Preview {
    NavigationStack { ProcurementView().environment(CartManager()) }
}

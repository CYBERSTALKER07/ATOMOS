import SwiftUI

struct FutureDemandView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = FutureDemandViewModel()

    var body: some View {
        ScrollView {
            VStack(spacing: AppTheme.spacingLG) {
                headerCard.slideIn(delay: 0)

                ForEach(Array(vm.items.enumerated()), id: \.element.id) { index, item in
                    predictionCard(item)
                        .staggeredSlideIn(index: index, baseDelay: 0.06)
                }
            }
            .padding(AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingXXL)
        }
        .scrollDismissesKeyboard(.interactively)
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .navigationTitle("mobile_retailer.ui.ai_demand_forecast")
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
        .task { await vm.loadPredictions() }
        .task(id: refreshCenter.refreshToken) { await vm.loadPredictions() }
        .refreshable { await vm.loadPredictions() }
    }

    private var headerCard: some View {
        GradientHeaderCard(title: "Pending AI preorders", subtitle: "Confirm or reject restock drafts", icon: "sparkles") {
            HStack(spacing: AppTheme.spacingXL) {
                miniStat(value: "\(vm.items.count)", label: "Pending")
                miniStat(value: "\(vm.totalUnits)", label: "Total Units")
            }
        }
    }

    private func miniStat(value: String, label: String) -> some View {
        VStack(spacing: 3) {
            Text(value)
                .font(.system(.headline, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(label)
                .font(.system(.caption2, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }

    private func predictionCard(_ item: RetailerAIPrediction) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        Circle()
                            .stroke(AppTheme.separator.opacity(0.3), lineWidth: 3)
                            .frame(width: 48, height: 48)
                        Text(String(item.statusLabel.prefix(7)))
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.warning)
                    }

                    VStack(alignment: .leading, spacing: 3) {
                        Text(item.title)
                            .font(.system(.headline, design: .rounded))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("\(item.deliveryLabel) · \(item.statusLabel)")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    VStack(spacing: 2) {
                        Text(item.formattedTotal)
                            .font(.system(.title3, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                        Text("\(item.quantity) units")
                            .font(.system(.caption2, design: .rounded, weight: .medium))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }
                .accessibilityElement(children: .combine)
                .accessibilityLabel("\(item.title), \(item.deliveryLabel), \(item.statusLabel), \(item.formattedTotal), \(item.quantity) units")

                Text(item.orderId)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .lineLimit(1)

                LabButton("Confirm", variant: .secondary, icon: "checkmark", fullWidth: true) {
                    Haptics.medium()
                    Task { await vm.confirm(item) }
                }
                .opacity(vm.actingId == item.orderId ? 0.5 : 1)
                .disabled(vm.actingId == item.orderId)
                .accessibilityLabel("Confirm AI preorder \(item.title)")

                HStack(spacing: AppTheme.spacingSM) {
                    Button {
                        Haptics.medium()
                        Task { await vm.reject(item) }
                    } label: {
                        Label("mobile_retailer.ui.reject", systemImage: "xmark")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.destructive)
                            .padding(.horizontal, AppTheme.spacingMD)
                            .padding(.vertical, AppTheme.spacingSM)
                            .background(AppTheme.destructive.opacity(0.1))
                            .clipShape(.capsule)
                            .pegasusMinimumHitTarget()
                    }
                    .disabled(vm.actingId == item.orderId)
                    .accessibilityLabel("Reject AI preorder \(item.title)")

                    Spacer()
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }
}

#Preview { FutureDemandView() }

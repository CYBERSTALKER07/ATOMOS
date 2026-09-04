import SwiftUI

struct FutureDemandView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var items: [RetailerAIPrediction] = []
    @State private var isLoading = false
    @State private var actingId: String?

    private let api = APIClient.shared

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
                    headerCard.slideIn(delay: 0)

                    ForEach(Array(items.enumerated()), id: \.element.id) { index, item in
                        predictionCard(item)
                            .staggeredSlideIn(index: index, baseDelay: 0.06)
                    }
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingXXL)
            }
            .scrollIndicators(.hidden)
            .background(AppTheme.background)
            .navigationTitle("mobile_retailer.ui.ai_demand_forecast")
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
                    .accessibilityLabel("Close")
                }
            }
            .task { await loadPredictions() }
            .task(id: refreshCenter.refreshToken) { await loadPredictions() }
            .refreshable { await loadPredictions() }
        }
    }

    private var headerCard: some View {
        GradientHeaderCard(title: "Pending AI preorders", subtitle: "Confirm or reject restock drafts", icon: "sparkles") {
            HStack(spacing: AppTheme.spacingXL) {
                miniStat(value: "\(items.count)", label: "Pending")
                miniStat(value: "\(totalUnits)", label: "Total Units")
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
    }

    private var totalUnits: Int64 {
        items.reduce(0) { $0 + $1.quantity }
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
                            .font(.system(size: 10, weight: .bold, design: .rounded))
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
                            .font(.system(size: 9, weight: .medium, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }

                Text(item.orderId)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .lineLimit(1)

                LabButton("Confirm", variant: .secondary, icon: "checkmark", fullWidth: true) {
                    Task { await confirm(item) }
                }
                .opacity(actingId == item.orderId ? 0.5 : 1)
                .disabled(actingId == item.orderId)

                HStack(spacing: AppTheme.spacingSM) {
                    Button {
                        Task { await reject(item) }
                    } label: {
                        Label("mobile_retailer.ui.reject", systemImage: "xmark")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.destructive)
                            .padding(.horizontal, AppTheme.spacingMD)
                            .padding(.vertical, AppTheme.spacingSM)
                            .background(AppTheme.destructive.opacity(0.1))
                            .clipShape(.capsule)
                    }

                    Spacer()
                }
            }
            .padding(AppTheme.spacingLG)
        }
    }

    private func loadPredictions() async {
        isLoading = true
        do { items = try await api.getRetailerAIPredictions() }
        catch { items = [] }
        isLoading = false
    }

    private func confirm(_ item: RetailerAIPrediction) async {
        actingId = item.orderId
        do {
            try await api.confirmAiOrder(orderId: item.orderId)
            Haptics.success()
            await loadPredictions()
        } catch {
            Haptics.error()
        }
        actingId = nil
    }

    private func reject(_ item: RetailerAIPrediction) async {
        actingId = item.orderId
        do {
            try await api.rejectAiOrder(orderId: item.orderId, reason: "Retailer rejected")
            Haptics.success()
            await loadPredictions()
        } catch {
            Haptics.error()
        }
        actingId = nil
    }
}

#Preview { FutureDemandView() }

import SwiftUI

struct FutureDemandView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var forecasts: [DemandForecast] = []
    @State private var isLoading = false
    @State private var preorderingId: String?
    @State private var correctionForecast: DemandForecast?
    @State private var correctionAmount: String = ""

    private let api = APIClient.shared

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: AppTheme.spacingLG) {
                    headerCard.slideIn(delay: 0)

                    ForEach(Array(forecasts.enumerated()), id: \.element.id) { index, forecast in
                        forecastCard(forecast)
                            .staggeredSlideIn(index: index, baseDelay: 0.06)
                    }
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingXXL)
            }
            .scrollIndicators(.hidden)
            .background(AppTheme.background)
            .navigationTitle("AI Demand Forecast")
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
            .task { await loadForecasts() }
            .alert("Correct Prediction", isPresented: .init(
                get: { correctionForecast != nil },
                set: { if !$0 { correctionForecast = nil; correctionAmount = "" } }
            )) {
                TextField("Correct amount", text: $correctionAmount)
                    .keyboardType(.numberPad)
                Button("Submit") {
                    if let forecast = correctionForecast, let amt = Int64(correctionAmount) {
                        Task { await correctPrediction(forecast, amount: amt) }
                    }
                    correctionForecast = nil; correctionAmount = ""
                }
                Button("Reject", role: .destructive) {
                    if let forecast = correctionForecast {
                        Task { await rejectPrediction(forecast) }
                    }
                    correctionForecast = nil; correctionAmount = ""
                }
                Button("Cancel", role: .cancel) {
                    correctionForecast = nil; correctionAmount = ""
                }
            } message: {
                if let forecast = correctionForecast {
                    Text("\(forecast.productName) — AI predicted \(forecast.predictedQuantity) units")
                }
            }
        }
    }

    // MARK: - Header

    private var headerCard: some View {
        GradientHeaderCard(title: "Smart Predictions", subtitle: "Based on sales history and market trends", icon: "sparkles") {
            HStack(spacing: AppTheme.spacingXL) {
                miniStat(value: "\(forecasts.count)", label: "Predictions")
                miniStat(value: avgConfidence, label: "Avg Confidence")
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

    private var avgConfidence: String {
        guard !forecasts.isEmpty else { return "—" }
        let avg = forecasts.map(\.confidence).reduce(0, +) / Double(forecasts.count)
        return String(format: "%.0f%%", avg * 100)
    }

    private var totalUnits: Int {
        forecasts.reduce(0) { $0 + $1.predictedQuantity }
    }

    // MARK: - Forecast Card

    private func forecastCard(_ forecast: DemandForecast) -> some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                // Header with confidence ring
                HStack(spacing: AppTheme.spacingMD) {
                    ZStack {
                        Circle()
                            .stroke(AppTheme.separator.opacity(0.3), lineWidth: 3)
                            .frame(width: 48, height: 48)
                        Circle()
                            .trim(from: 0, to: forecast.confidence)
                            .stroke(confidenceColor(forecast.confidence), style: StrokeStyle(lineWidth: 3, lineCap: .round))
                            .frame(width: 48, height: 48)
                            .rotationEffect(.degrees(-90))
                        Text(forecast.confidencePercent)
                            .font(.system(size: 11, weight: .bold, design: .rounded))
                            .foregroundStyle(confidenceColor(forecast.confidence))
                    }

                    VStack(alignment: .leading, spacing: 3) {
                        Text(forecast.productName)
                            .font(.system(.headline, design: .rounded))
                            .foregroundStyle(AppTheme.textPrimary)
                        Text("Order by \(forecast.suggestedOrderDate)")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }

                    Spacer()

                    VStack(spacing: 2) {
                        Text("\(forecast.predictedQuantity)")
                            .font(.system(.title3, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.accent)
                        Text("units")
                            .font(.system(size: 9, weight: .medium, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                }

                // Confidence bar
                GeometryReader { geo in
                    ZStack(alignment: .leading) {
                        Capsule().fill(AppTheme.separator.opacity(0.2)).frame(height: 5)
                        Capsule()
                            .fill(LinearGradient(colors: [confidenceColor(forecast.confidence).opacity(0.6), confidenceColor(forecast.confidence)], startPoint: .leading, endPoint: .trailing))
                            .frame(width: geo.size.width * forecast.confidence, height: 5)
                    }
                }
                .frame(height: 5)

                Text(forecast.reasoning)
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textTertiary)
                    .lineLimit(3)

                LabButton("Preorder \(forecast.predictedQuantity) Units", variant: .secondary, icon: "cart.badge.plus", fullWidth: true) {
                    Task { await preorder(forecast) }
                }
                .opacity(preorderingId == forecast.id ? 0.5 : 1)
                .disabled(preorderingId == forecast.id)

                HStack(spacing: AppTheme.spacingSM) {
                    Button {
                        correctionForecast = forecast
                    } label: {
                        Label("Correct", systemImage: "pencil.line")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textSecondary)
                            .padding(.horizontal, AppTheme.spacingMD)
                            .padding(.vertical, AppTheme.spacingSM)
                            .background(AppTheme.surfaceElevated)
                            .clipShape(.capsule)
                    }

                    Button {
                        Task { await rejectPrediction(forecast) }
                    } label: {
                        Label("Reject", systemImage: "xmark")
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

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }

    // MARK: - API

    private func loadForecasts() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do { let r: [DemandForecast] = try await api.get(path: "/v1/ai/predictions?retailer_id=\(rid)"); forecasts = r }
        catch { forecasts = [] }
        isLoading = false
    }

    private func preorder(_ forecast: DemandForecast) async {
        preorderingId = forecast.id
        do {
            let body: [String: String] = ["product_id": forecast.productId, "quantity": "\(forecast.predictedQuantity)"]
            let _: [String: String] = try await api.post(path: "/v1/ai/preorder", body: body)
            Haptics.success()
        } catch {
            Haptics.error()
        }
        preorderingId = nil
    }

    private func correctPrediction(_ forecast: DemandForecast, amount: Int64) async {
        do {
            let body: [String: Int64] = ["amount": amount]
            let _: [String: String] = try await api.patch(
                path: "/v1/ai/predictions/correct?prediction_id=\(forecast.id)",
                body: body,
                headers: ["Idempotency-Key": "retailer-prediction-correct:\(forecast.id):amount:\(amount)"]
            )
            Haptics.success()
            await loadForecasts()
        } catch {
            Haptics.error()
        }
    }

    private func rejectPrediction(_ forecast: DemandForecast) async {
        do {
            let body: [String: String] = ["status": "REJECTED"]
            let _: [String: String] = try await api.patch(
                path: "/v1/ai/predictions/correct?prediction_id=\(forecast.id)",
                body: body,
                headers: ["Idempotency-Key": "retailer-prediction-correct:\(forecast.id):rejected"]
            )
            Haptics.success()
            await loadForecasts()
        } catch {
            Haptics.error()
        }
    }
}

#Preview { FutureDemandView() }

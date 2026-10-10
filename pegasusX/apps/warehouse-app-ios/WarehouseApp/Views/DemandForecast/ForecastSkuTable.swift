import SwiftUI

struct ForecastSkuTable: View {
    let products: [DemandForecastProduct]
    let generatedAt: String?
    let forecastDays: Int

    var body: some View {
        VStack(spacing: LabTheme.spacingLG) {
            ForEach(Array(products.enumerated()), id: \.element.id) { index, product in
                ForecastProductRow(product: product)
                    .staggeredAppear(index: index + 3)
            }

            if let generatedAt = generatedAt, !generatedAt.isEmpty {
                Text(L10n.format("mobile_warehouse.ui.generated_formattedgeneratedat_forecastdays_day_window", "\(formattedGeneratedAt(generatedAt))", "\(forecastDays)"))
                    .font(.caption2)
                    .foregroundStyle(.tertiary)
            }
        }
    }

    private func formattedGeneratedAt(_ raw: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: raw) ?? ISO8601DateFormatter().date(from: raw) {
            return date.formatted(date: .abbreviated, time: .shortened)
        }
        return raw
    }
}

private struct ForecastProductRow: View {
    let product: DemandForecastProduct

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            HStack(alignment: .firstTextBaseline) {
                Text(product.productName.isEmpty ? String(product.productId.prefix(8)) : product.productName)
                    .font(.headline)
                Spacer()
                Text(product.priority)
                    .font(.caption.bold())
                    .foregroundStyle(priorityColor(product.priority))
            }

            HStack {
                metricColumn(title: "Stock", value: "\(product.currentStock)")
                metricColumn(title: "Rec.", value: "\(product.recommendedQty)")
                metricColumn(
                    title: "Stockout",
                    value: String(format: "%.1fd", product.daysUntilStockout),
                    valueColor: stockoutColor(product.daysUntilStockout)
                )
            }

            HStack(spacing: LabTheme.spacingMD) {
                sourceChip(label: "In", value: "\(product.sources.incomingOrders)")
                sourceChip(label: "AI", value: "\(product.sources.aiPrediction)")
                sourceChip(label: "Pre", value: "\(product.sources.preOrders)")
                sourceChip(label: "Burn", value: String(format: "%.1f", product.sources.burnRate))
            }

            if let confidence = parseForecastConfidence(product.demandBreakdown) {
                ForecastConfidenceView(confidence: confidence, compact: true)
            }
        }
        .labCard()
    }

    private func metricColumn(title: String, value: String, valueColor: Color = LabTheme.label) -> some View {
        VStack(alignment: .leading, spacing: 2) {
            Text(title)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.subheadline.monospacedDigit())
                .foregroundStyle(valueColor)
        }
        .frame(maxWidth: .infinity, alignment: .leading)
    }

    private func sourceChip(label: String, value: String) -> some View {
        VStack(spacing: 2) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(.secondary)
            Text(value)
                .font(.caption.monospacedDigit())
        }
        .frame(maxWidth: .infinity)
    }

    private func priorityColor(_ priority: String) -> Color {
        switch priority.uppercased() {
        case "CRITICAL": return LabTheme.destructive
        case "URGENT": return LabTheme.warning
        default: return LabTheme.secondaryLabel
        }
    }

    private func stockoutColor(_ days: Double) -> Color {
        if days < 2 { return LabTheme.destructive }
        if days < 5 { return LabTheme.warning }
        return LabTheme.label
    }
}

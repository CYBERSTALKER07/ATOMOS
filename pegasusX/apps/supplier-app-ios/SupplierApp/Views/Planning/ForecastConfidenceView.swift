import SwiftUI

struct ForecastConfidenceView: View {
    let confidence: ForecastConfidence
    var updatedAt: String?
    var stale: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            HStack {
                Text("supplier_portal.forecast_confidence_card.text.forecast_confidence")
                    .font(.subheadline.bold())
                Spacer()
                if let baseline = confidence.baselineSource {
                    Text(ForecastConfidenceSupport.formatBaselineSourceLabel(baseline))
                        .font(.caption2.weight(.semibold))
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(SupplierTheme.secondaryBackground)
                        .clipShape(Capsule())
                }
            }
            if confidence.isBlocked {
                Text("warehouse_portal.forecast_confidence_view.text.insufficient_history_predictive_forecast_blocked")
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.warning)
            } else {
                Text(L10n.format("mobile_supplier.ui.lowunits_0_highunits_0_units", "\(confidence.lowUnits ?? 0)", "\(confidence.highUnits ?? 0)"))
                    .font(.title3.bold())
                if let pct = confidence.confidencePct {
                    Text(L10n.format("mobile_supplier.ui.pct_confidence_2", "\(pct)"))
                        .font(.caption)
                        .foregroundStyle(confidence.confidenceTint)
                }
            }
            if let updatedAt {
                Text("\(stale ? "Stale · " : "")Updated \(updatedAt)")
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }
}

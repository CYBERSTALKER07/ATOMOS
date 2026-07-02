import SwiftUI

struct ForecastConfidenceView: View {
    let confidence: ForecastConfidence
    var updatedAt: String?
    var stale: Bool = false

    var body: some View {
        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            HStack {
                Text("Forecast confidence")
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
                Text("Insufficient history — predictive forecast blocked")
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.warning)
            } else {
                Text("\(confidence.lowUnits ?? 0, format: .number) – \(confidence.highUnits ?? 0, format: .number) units")
                    .font(.title3.bold())
                if let pct = confidence.confidencePct {
                    Text("\(pct)% confidence")
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

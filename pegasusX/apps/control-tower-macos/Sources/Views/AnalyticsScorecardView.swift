import SwiftUI

public struct AnalyticsScorecardView: View {
    @ObservedObject var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                VStack(alignment: .leading, spacing: 4) {
                    Text("Multi-Tier Operational Scorecards")
                        .font(.title2)
                        .fontWeight(.bold)
                    Text("Cross-calibrated SLAs and compliance thresholds across Supplier, Warehouse, Fleet, and Retailer echelons.")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }

                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 16) {
                    ForEach(viewModel.scorecards) { card in
                        VStack(alignment: .leading, spacing: 14) {
                            HStack {
                                Text(card.role)
                                    .font(.headline)
                                    .fontWeight(.bold)
                                Spacer()
                                HStack(spacing: 5) {
                                    Circle()
                                        .fill(card.complianceRatePercent >= 98.0 ? Color.green : Color.orange)
                                        .frame(width: 7, height: 7)
                                    Text("\(String(format: "%.1f", card.complianceRatePercent))% SLA")
                                        .font(.system(.caption, design: .monospaced))
                                        .fontWeight(.bold)
                                        .foregroundColor(card.complianceRatePercent >= 98.0 ? .green : .orange)
                                }
                                .padding(.horizontal, 8)
                                .padding(.vertical, 3)
                                .background(card.complianceRatePercent >= 98.0 ? Color.green.opacity(0.12) : Color.orange.opacity(0.12))
                                .clipShape(Capsule())
                            }

                            Divider()

                            HStack(spacing: 24) {
                                VStack(alignment: .leading, spacing: 4) {
                                    Text(card.primaryMetricName)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    Text(card.primaryMetricValue)
                                        .font(.system(.title2, design: .rounded))
                                        .fontWeight(.bold)
                                        .monospacedDigit()
                                    Text(card.primaryMetricSubtext)
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }
                                Spacer()
                                VStack(alignment: .trailing, spacing: 4) {
                                    Text(card.secondaryMetricName)
                                        .font(.caption)
                                        .foregroundColor(.secondary)
                                    Text(card.secondaryMetricValue)
                                        .font(.system(.title3, design: .monospaced))
                                        .fontWeight(.bold)
                                        .foregroundColor(.blue)
                                        .monospacedDigit()
                                    Text("Statutory Target")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                }
                            }

                            VStack(alignment: .leading, spacing: 4) {
                                HStack {
                                    Text("SLA Delivery Threshold")
                                        .font(.caption2)
                                        .foregroundColor(.secondary)
                                    Spacer()
                                    Text("\(String(format: "%.1f", card.complianceRatePercent))%")
                                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                                        .foregroundColor(.secondary)
                                }
                                ProgressView(value: card.complianceRatePercent, total: 100.0)
                                    .tint(card.complianceRatePercent >= 98.0 ? .green : .orange)
                            }
                        }
                        .padding(18)
                        .background(.ultraThinMaterial)
                        .overlay(
                            RoundedRectangle(cornerRadius: 14, style: .continuous)
                                .stroke(
                                    LinearGradient(
                                        colors: [Color.primary.opacity(0.12), Color.primary.opacity(0.02)],
                                        startPoint: .topLeading,
                                        endPoint: .bottomTrailing
                                    ),
                                    lineWidth: 1
                                )
                        )
                        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                        .shadow(color: Color.black.opacity(0.04), radius: 6, x: 0, y: 2)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("\(card.role) Operational Scorecard: \(String(format: "%.1f", card.complianceRatePercent)) percent compliance")
                        .accessibilityValue("\(card.primaryMetricName): \(card.primaryMetricValue), \(card.secondaryMetricName): \(card.secondaryMetricValue)")
                    }
                }

                Spacer()
            }
            .padding(24)
        }
    }
}

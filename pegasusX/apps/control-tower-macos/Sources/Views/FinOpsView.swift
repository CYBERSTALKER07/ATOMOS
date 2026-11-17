import Charts
import SwiftUI

public struct FinOpsView: View {
    @Bindable var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 22) {
                // Header with Export button
                HStack(alignment: .top) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("FinOps, Cost-to-Serve & Attributed Billing")
                            .font(.title2)
                            .fontWeight(.bold)
                        Text("Enterprise multidimensional financial ledger matching SAP S/4HANA TM and o9 Solutions Digital Brain.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    Button {
                        viewModel.exportWaterfallCSV()
                    } label: {
                        Label("Export Waterfall CSV", systemImage: "square.and.arrow.up")
                    }
                    .buttonStyle(.bordered)
                    .accessibilityLabel("Export Tender Waterfall CSV")
                }

                // Role Perspective Picker
                Picker("Role Perspective", selection: $viewModel.selectedRole) {
                    ForEach(RolePerspective.allCases) { role in
                        HStack {
                            Image(systemName: role.systemIcon)
                            Text(role.rawValue)
                        }
                        .tag(role)
                    }
                }
                .pickerStyle(.segmented)
                .accessibilityLabel("Role Perspective")

                // Visual Tender Settlement Waterfall
                VStack(alignment: .leading, spacing: 14) {
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text("Tender Settlement Waterfall — \(viewModel.selectedRole.rawValue)")
                                .font(.headline)
                            Text("Multi-tender breakdown with mathematical balance invariants and tax tracking.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                    }

                    // Swift Charts Bar Flow with Rounded Gradient Marks
                    Chart {
                        ForEach(viewModel.waterfallSlices) { slice in
                            BarMark(
                                x: .value("Amount", slice.amountTiyin / 100),
                                y: .value("Stage", slice.label)
                            )
                            .foregroundStyle(
                                slice.isDeduction
                                    ? LinearGradient(colors: [Color.red.opacity(0.9), Color.orange.opacity(0.8)], startPoint: .leading, endPoint: .trailing)
                                    : LinearGradient(colors: [Color.blue.opacity(0.9), Color.teal.opacity(0.8)], startPoint: .leading, endPoint: .trailing)
                            )
                            .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                            .annotation(position: .trailing) {
                                Text(slice.amountUzsFormatted)
                                    .font(.system(.caption2, design: .monospaced).weight(.bold))
                                    .foregroundStyle(.primary)
                                    .padding(.leading, 4)
                            }
                            .accessibilityLabel(slice.label)
                            .accessibilityValue("\(slice.amountUzsFormatted), \(slice.isDeduction ? "Deduction" : "Disbursable")")
                        }
                    }
                    .accessibilityLabel("Tender settlement waterfall chart for \(viewModel.selectedRole.rawValue)")
                    .frame(height: CGFloat(max(190, viewModel.waterfallSlices.count * 40)))
                    .padding(.vertical, 6)

                    Divider()

                    // Itemized Breakdown Table
                    VStack(spacing: 8) {
                        ForEach(viewModel.waterfallSlices) { slice in
                            HStack(alignment: .center) {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack(spacing: 6) {
                                        Text(slice.label)
                                            .font(.subheadline)
                                            .fontWeight(.semibold)
                                        Text(slice.isDeduction ? "DEDUCTION" : "DISBURSABLE")
                                            .font(.system(.caption2, design: .monospaced).weight(.bold))
                                            .padding(.horizontal, 5)
                                            .padding(.vertical, 2)
                                            .background(slice.isDeduction ? Color.red.opacity(0.12) : Color.blue.opacity(0.12))
                                            .foregroundStyle(slice.isDeduction ? Color.red : Color.blue)
                                            .clipShape(Capsule())
                                    }
                                    Text(slice.detail)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Spacer()
                                VStack(alignment: .trailing, spacing: 2) {
                                    Text(slice.amountUzsFormatted)
                                        .font(.system(.subheadline, design: .monospaced))
                                        .fontWeight(.bold)
                                        .foregroundStyle(slice.isDeduction ? Color.red : Color.primary)
                                        .monospacedDigit()
                                    Text("\(String(format: "%.2f", slice.percentage))%")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                        .monospacedDigit()
                                }
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .background(Color.primary.opacity(0.02))
                            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(slice.label), \(slice.isDeduction ? "Deduction" : "Disbursable"): \(slice.amountUzsFormatted)")
                        }
                    }
                }
                .padding(18)
                .background(.ultraThinMaterial)
                .overlay(
                    RoundedRectangle(cornerRadius: 16, style: .continuous)
                        .stroke(
                            LinearGradient(
                                colors: [Color.primary.opacity(0.12), Color.primary.opacity(0.02)],
                                startPoint: .topLeading,
                                endPoint: .bottomTrailing
                            ),
                            lineWidth: 1
                        )
                )
                .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 3)

                // Attributed Infrastructure Cost-to-Serve Table
                VStack(alignment: .leading, spacing: 10) {
                    Text("Attributed Infrastructure Consumption")
                        .font(.headline)

                    VStack(spacing: 8) {
                        ForEach(viewModel.costs) { cost in
                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack(spacing: 6) {
                                        Text(cost.resourceName)
                                            .font(.subheadline)
                                            .fontWeight(.semibold)
                                        Text("• \(cost.role)")
                                            .font(.caption)
                                            .foregroundStyle(.secondary)
                                    }
                                    HStack(spacing: 6) {
                                        Text("Entity:")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                        Text(cost.entityId)
                                            .font(.system(.caption2, design: .monospaced))
                                            .foregroundStyle(.blue)
                                        Button {
                                            viewModel.copyToClipboard(cost.entityId, label: "Entity ID")
                                        } label: {
                                            Image(systemName: "doc.on.doc")
                                                .font(.caption2)
                                                .foregroundStyle(.secondary)
                                                .frame(minWidth: 32, minHeight: 32)
                                                .contentShape(Rectangle())
                                        }
                                        .buttonStyle(.plain)
                                        .help("Copy Entity ID")
                                        .accessibilityLabel("Copy Entity ID \(cost.entityId)")
                                    }
                                }
                                Spacer()
                                HStack(spacing: 20) {
                                    VStack(alignment: .trailing, spacing: 2) {
                                        Text(cost.usageMetric)
                                            .font(.system(.caption, design: .monospaced))
                                            .foregroundStyle(.secondary)
                                        Text("Quota Volume")
                                            .font(.caption2)
                                            .foregroundStyle(.secondary)
                                    }
                                    VStack(alignment: .trailing, spacing: 2) {
                                        Text(cost.monthlyCostUsdFormatted)
                                            .font(.system(.subheadline, design: .monospaced))
                                            .fontWeight(.bold)
                                            .monospacedDigit()
                                        HStack(spacing: 2) {
                                            Image(systemName: cost.trendPercent >= 0 ? "arrow.up.right" : "arrow.down.right")
                                            Text("\(String(format: "%+.1f", cost.trendPercent))%")
                                        }
                                        .font(.caption2)
                                        .foregroundStyle(cost.trendPercent > 0 ? Color.orange : Color.green)
                                    }
                                }
                            }
                            .padding(10)
                            .background(Color(NSColor.controlBackgroundColor))
                            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("\(cost.resourceName), Role \(cost.role), Entity \(cost.entityId)")
                            .accessibilityValue("\(cost.monthlyCostUsdFormatted) per month, Quota \(cost.usageMetric), Trend \(String(format: "%+.1f", cost.trendPercent)) percent")
                        }
                    }
                }

                Spacer()
            }
            .padding(24)
        }
    }
}

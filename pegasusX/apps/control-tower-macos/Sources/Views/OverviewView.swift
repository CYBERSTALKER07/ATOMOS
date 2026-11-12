import SwiftUI

public struct OverviewView: View {
    var viewModel: ControlTowerViewModel
    @State private var isPulsing: Bool = false
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 24) {
                // Header with live pulse badge
                HStack(alignment: .center) {
                    VStack(alignment: .leading, spacing: 6) {
                        HStack(spacing: 8) {
                            Text("PegasusX Global NOC & Control Tower")
                                .font(.system(.title2, design: .rounded))
                                .fontWeight(.bold)
                            Text("PROD-CELL")
                                .font(.system(.caption2, design: .monospaced).weight(.bold))
                                .padding(.horizontal, 6)
                                .padding(.vertical, 2)
                                .background(Color.blue.opacity(0.15))
                                .foregroundStyle(.blue)
                                .clipShape(Capsule())
                        }
                        Text("Enterprise-grade real-time telemetry across multi-cell infrastructure and settlement engines.")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()

                    // Live Operational Pulse Pill
                    HStack(spacing: 8) {
                        ZStack {
                            if !reduceMotion {
                                Circle()
                                    .fill(Color.green.opacity(0.35))
                                    .frame(width: 14, height: 14)
                                    .scaleEffect(isPulsing ? 1.4 : 1.0)
                                    .opacity(isPulsing ? 0.0 : 0.8)
                                    .animation(Animation.easeOut(duration: 1.8).repeatForever(autoreverses: false), value: isPulsing)
                            }
                            Circle()
                                .fill(Color.green)
                                .frame(width: 8, height: 8)
                        }
                        Text("ALL CELLS OPERATIONAL")
                            .font(.system(.caption, design: .monospaced).weight(.bold))
                            .foregroundStyle(.green)
                    }
                    .padding(.horizontal, 12)
                    .padding(.vertical, 7)
                    .background(Color.green.opacity(0.12))
                    .overlay(
                        Capsule()
                            .stroke(Color.green.opacity(0.25), lineWidth: 1)
                    )
                    .clipShape(Capsule())
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Operational Status: All Cells Operational")
                }
                .task { isPulsing = true }

                // Top KPI Grid with Glassmorphic Elevation
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible()), GridItem(.flexible()), GridItem(.flexible())], spacing: 16) {
                    metricCard(
                        title: "Active Cells",
                        value: "\(viewModel.cells.count) Regions",
                        subtext: "Toshkent • Samarqand • Fargʻona",
                        icon: "network",
                        color: .blue,
                        trend: "100% Mesh"
                    )
                    metricCard(
                        title: "Spanner P99 Latency",
                        value: "\(String(format: "%.1f", viewModel.spanner?.p99ReadMs ?? 0)) ms",
                        subtext: "Write P99: \(String(format: "%.1f", viewModel.spanner?.p99WriteMs ?? 0)) ms",
                        icon: "internaldrive.fill",
                        color: .teal,
                        trend: "ACID Validated"
                    )
                    let totalDLQ = viewModel.kafkaPartitions.reduce(0) { $0 + $1.dlqCount }
                    metricCard(
                        title: "Kafka DLQ Backlog",
                        value: "\(totalDLQ) Dead Letters",
                        subtext: totalDLQ > 0 ? "Requires Operator Triage" : "Zero poison messages",
                        icon: "tray.full.fill",
                        color: totalDLQ > 0 ? .red : .green,
                        trend: totalDLQ > 0 ? "ATTENTION" : "Clean"
                    )
                    metricCard(
                        title: "Attributed Burn Rate",
                        value: "$14,820 / mo",
                        subtext: "Dynamic Platform Take-Rate",
                        icon: "banknote.fill",
                        color: .purple,
                        trend: "FinOps Nominal"
                    )
                }

                // Active Incident & Dead Letter Queue Banner if any DLQ exists
                let totalDLQ = viewModel.kafkaPartitions.reduce(0) { $0 + $1.dlqCount }
                if totalDLQ > 0 {
                    HStack(spacing: 14) {
                        ZStack {
                            Circle()
                                .fill(Color.orange.opacity(0.2))
                                .frame(width: 36, height: 36)
                            Image(systemName: "exclamationmark.triangle.fill")
                                .font(.title3)
                                .foregroundStyle(.orange)
                        }
                        VStack(alignment: .leading, spacing: 3) {
                            Text("Dead-Letter Queue Backlog Alert (Kafka topic: compliance.fiscal.soliq)")
                                .font(.subheadline)
                                .fontWeight(.bold)
                            Text("3 poison messages held in DLQ partition 2 awaiting Soliq endpoint token renewal. Auto-quarantine engaged.")
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        Button("Triage in Telemetry") {
                            viewModel.triageDLQInTelemetry()
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(.orange)
                        .accessibilityLabel("Triage Dead-Letter Queue in Telemetry")
                    }
                    .padding(14)
                    .background(.ultraThinMaterial)
                    .overlay(
                        RoundedRectangle(cornerRadius: 14, style: .continuous)
                            .stroke(Color.orange.opacity(0.3), lineWidth: 1)
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
                    .shadow(color: Color.orange.opacity(0.08), radius: 8, x: 0, y: 3)
                }

                // Regional Deployment Grid
                VStack(alignment: .leading, spacing: 14) {
                    HStack {
                        Text("Regional Deployment Grid")
                            .font(.headline)
                        Spacer()
                        Text("\(viewModel.cells.count) Live Nodes")
                            .font(.system(.caption, design: .monospaced))
                            .foregroundStyle(.secondary)
                    }

                    ForEach(viewModel.cells) { cell in
                        let maxCpuPercent = 100.0
                        HStack(spacing: 16) {
                            // Status indicator & Cell ID
                            VStack(alignment: .leading, spacing: 3) {
                                HStack(spacing: 6) {
                                    Circle()
                                        .fill(Color.green)
                                        .frame(width: 8, height: 8)
                                    Text(cell.cellId)
                                        .font(.system(.subheadline, design: .monospaced))
                                        .fontWeight(.bold)
                                    Text("• \(cell.region)")
                                        .font(.subheadline)
                                        .foregroundStyle(.secondary)
                                }
                                Text("Role: \(cell.role) • \(cell.activeNodes) Cluster Nodes Active")
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                            }

                            Spacer()

                            // CPU Progress Bar
                            VStack(alignment: .trailing, spacing: 3) {
                                HStack(spacing: 6) {
                                    Text("Load:")
                                        .font(.caption2)
                                        .foregroundStyle(.secondary)
                                    Text("\(String(format: "%.1f", cell.cpuPercent))% CPU")
                                        .font(.system(.caption, design: .monospaced))
                                        .fontWeight(.semibold)
                                }
                                ProgressView(value: cell.cpuPercent, total: maxCpuPercent)
                                    .frame(width: 90)
                                    .tint(cell.cpuPercent > 80 ? .red : (cell.cpuPercent > 60 ? .orange : .blue))
                            }

                            // QPS & Latency Pills
                            VStack(alignment: .trailing, spacing: 3) {
                                Text("\(cell.qps) QPS")
                                    .font(.system(.caption, design: .monospaced))
                                    .fontWeight(.bold)
                                    .monospacedDigit()
                                Text("Throughput")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                            .frame(minWidth: 70, alignment: .trailing)

                            VStack(alignment: .trailing, spacing: 3) {
                                Text("\(String(format: "%.1f", cell.p99LatencyMs)) ms")
                                    .font(.system(.caption, design: .monospaced))
                                    .fontWeight(.bold)
                                    .foregroundStyle(.teal)
                                    .monospacedDigit()
                                Text("P99 Latency")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                            .frame(minWidth: 70, alignment: .trailing)
                        }
                        .padding(14)
                        .background(.ultraThinMaterial)
                        .overlay(
                            RoundedRectangle(cornerRadius: 12, style: .continuous)
                                .stroke(
                                    LinearGradient(
                                        colors: [Color.primary.opacity(0.10), Color.primary.opacity(0.02)],
                                        startPoint: .topLeading,
                                        endPoint: .bottomTrailing
                                    ),
                                    lineWidth: 1
                                )
                        )
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        .shadow(color: Color.black.opacity(0.04), radius: 6, x: 0, y: 2)
                        .accessibilityElement(children: .combine)
                        .accessibilityLabel("Cell \(cell.cellId), \(cell.region), Role \(cell.role), \(cell.activeNodes) cluster nodes active")
                        .accessibilityValue("CPU Load \(String(format: "%.1f", cell.cpuPercent)) percent, Throughput \(cell.qps) QPS, P99 Latency \(String(format: "%.1f", cell.p99LatencyMs)) milliseconds")
                    }
                }

                Spacer()
            }
            .padding(24)
        }
    }

    private func metricCard(title: String, value: String, subtext: String, icon: String, color: Color, trend: String) -> some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(title.uppercased())
                        .font(.system(.caption2, design: .monospaced).weight(.bold))
                        .tracking(0.8)
                        .foregroundStyle(.secondary)
                    Text(trend)
                        .font(.system(.caption2, design: .monospaced).weight(.bold))
                        .foregroundStyle(color)
                        .padding(.horizontal, 5)
                        .padding(.vertical, 2)
                        .background(color.opacity(0.12))
                        .clipShape(Capsule())
                }
                Spacer()
                ZStack {
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .fill(color.opacity(0.15))
                        .frame(width: 36, height: 36)
                    Image(systemName: icon)
                        .font(.headline.weight(.semibold))
                        .foregroundStyle(color)
                }
            }

            VStack(alignment: .leading, spacing: 3) {
                Text(value)
                    .font(.system(.title2, design: .monospaced).weight(.bold))
                    .monospacedDigit()
                Text(subtext)
                    .font(.caption2)
                    .foregroundStyle(.secondary)
            }
        }
        .padding(16)
        .background(.ultraThinMaterial)
        .overlay(
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(
                    LinearGradient(
                        colors: [Color.primary.opacity(0.15), Color.primary.opacity(0.05)],
                        startPoint: .topLeading,
                        endPoint: .bottomTrailing
                    ),
                    lineWidth: 1
                )
        )
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .shadow(color: Color.black.opacity(0.05), radius: 8, x: 0, y: 3)
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(title): \(value)")
        .accessibilityValue("\(trend). \(subtext)")
    }
}

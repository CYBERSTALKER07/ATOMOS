import SwiftUI

public struct InspectorView: View {
    @ObservedObject var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 16) {
                Text("Ecosystem Inspector")
                    .font(.headline)
                    .padding(.bottom, 4)

                // Current Perspective Card
                GroupBox("Active Filter Scope") {
                    VStack(alignment: .leading, spacing: 8) {
                        HStack {
                            Image(systemName: viewModel.selectedRole.systemIcon)
                                .foregroundColor(.accentColor)
                            Text(viewModel.selectedRole.rawValue)
                                .font(.subheadline)
                                .fontWeight(.medium)
                        }
                        Text("All telemetry, FinOps cost-to-serve, and scorecards are scoped to this echelon.")
                            .font(.caption)
                            .foregroundColor(.secondary)
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(4)
                }

                // Cloud Engine Summary
                GroupBox("Core Engines") {
                    VStack(alignment: .leading, spacing: 10) {
                        inspectorRow(label: "Spanner Read P99", value: "\(String(format: "%.1f", viewModel.spanner?.p99ReadMs ?? 0))ms")
                        inspectorRow(label: "Spanner Write P99", value: "\(String(format: "%.1f", viewModel.spanner?.p99WriteMs ?? 0))ms")
                        inspectorRow(label: "Kafka Max Lag", value: "\(viewModel.kafkaPartitions.map(\.maxLagMs).max() ?? 0)ms")
                        let totalDLQ = viewModel.kafkaPartitions.reduce(0) { $0 + $1.dlqCount }
                        inspectorRow(label: "Poison DLQ Total", value: "\(totalDLQ)", isAlert: totalDLQ > 0)
                        inspectorRow(label: "Active Mutex Sessions", value: "\(viewModel.spanner?.activeSessions ?? 0) / \(viewModel.spanner?.maxSessions ?? 0)")
                    }
                    .frame(maxWidth: .infinity, alignment: .leading)
                    .padding(4)
                }

                // Quick Actions
                GroupBox("Operator Tools") {
                    VStack(spacing: 8) {
                        Button {
                            viewModel.exportDiagnosticBundle()
                        } label: {
                            Label("Export Diagnostic Bundle", systemImage: "arrow.down.doc.fill")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.borderedProminent)

                        Button {
                            viewModel.exportWaterfallCSV()
                        } label: {
                            Label("Export Waterfall CSV", systemImage: "tablecells.badge.ellipsis")
                                .frame(maxWidth: .infinity)
                        }
                        .buttonStyle(.bordered)
                    }
                    .padding(4)
                }

                Spacer()
            }
            .padding()
        }
        .frame(minWidth: 260, idealWidth: 280)
    }

    private func inspectorRow(label: String, value: String, isAlert: Bool = false) -> some View {
        HStack {
            Text(label)
                .font(.caption)
                .foregroundColor(.secondary)
            Spacer()
            Text(value)
                .font(.system(.caption, design: .monospaced))
                .fontWeight(.semibold)
                .foregroundColor(isAlert ? .red : .primary)
                .monospacedDigit()
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }
}

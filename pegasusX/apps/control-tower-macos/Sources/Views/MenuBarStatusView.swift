import SwiftUI

public struct MenuBarStatusView: View {
    @ObservedObject var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // Header
            HStack {
                Circle()
                    .fill(Color.green)
                    .frame(width: 8, height: 8)
                Text("PegasusX Control Tower")
                    .font(.headline)
                Spacer()
                Text("3 Cells")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundColor(.secondary)
            }

            Divider()

            // Metrics
            VStack(alignment: .leading, spacing: 8) {
                HStack {
                    Text("Cloud Spanner Read P99:")
                        .font(.caption)
                    Spacer()
                    Text("\(String(format: "%.1f", viewModel.spanner?.p99ReadMs ?? 0)) ms")
                        .font(.system(.caption, design: .monospaced))
                        .fontWeight(.bold)
                }

                HStack {
                    Text("Kafka DLQ Backlog:")
                        .font(.caption)
                    Spacer()
                    let totalDLQ = viewModel.kafkaPartitions.reduce(0) { $0 + $1.dlqCount }
                    Text("\(totalDLQ) messages")
                        .font(.system(.caption, design: .monospaced))
                        .fontWeight(.bold)
                        .foregroundColor(totalDLQ > 0 ? .red : .green)
                }

                HStack {
                    Text("Active Perspective:")
                        .font(.caption)
                    Spacer()
                    Text(viewModel.selectedRole.rawValue)
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundColor(.blue)
                }
            }

            Divider()

            // Quick App Controls
            HStack(spacing: 8) {
                Button("Open Window") {
                    NSApp.activate(ignoringOtherApps: true)
                    if let window = NSApp.windows.first(where: { $0.canBecomeMain }) {
                        window.makeKeyAndOrderFront(nil)
                    }
                }
                .buttonStyle(.borderedProminent)
                .font(.caption)

                Button("Refresh") {
                    Task { await viewModel.initialLoad() }
                }
                .buttonStyle(.bordered)
                .font(.caption)

                Spacer()

                Button("Quit") {
                    NSApplication.shared.terminate(nil)
                }
                .buttonStyle(.plain)
                .font(.caption)
                .foregroundColor(.secondary)
            }
        }
        .padding(14)
        .frame(width: 280)
    }
}

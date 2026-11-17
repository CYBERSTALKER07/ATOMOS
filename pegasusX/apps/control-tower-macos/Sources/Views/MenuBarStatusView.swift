import SwiftUI

public struct MenuBarStatusView: View {
    var viewModel: ControlTowerViewModel

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
                Text("\(viewModel.cells.isEmpty ? 3 : viewModel.cells.count) Cells")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(.secondary)
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("PegasusX Control Tower, \(viewModel.cells.isEmpty ? 3 : viewModel.cells.count) active cells")

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
                .accessibilityElement(children: .combine)

                let totalDLQ = viewModel.kafkaPartitions.reduce(0) { $0 + $1.dlqCount }
                HStack {
                    Text("Kafka DLQ Backlog:")
                        .font(.caption)
                    Spacer()
                    Text("\(totalDLQ) messages")
                        .font(.system(.caption, design: .monospaced))
                        .fontWeight(.bold)
                        .foregroundStyle(totalDLQ > 0 ? Color.red : Color.green)
                }
                .accessibilityElement(children: .combine)

                HStack {
                    Text("Active Perspective:")
                    .font(.caption)
                    Spacer()
                    Text(viewModel.selectedRole.rawValue)
                        .font(.caption)
                        .fontWeight(.semibold)
                        .foregroundStyle(.blue)
                }
                .accessibilityElement(children: .combine)
            }

            Divider()

            // Quick App Controls
            HStack(spacing: 8) {
                Button("Open Window") {
                    ControlTowerHaptics.alignment()
                    NSApp.activate(ignoringOtherApps: true)
                    if let window = NSApp.windows.first(where: { $0.canBecomeMain }) {
                        window.makeKeyAndOrderFront(nil)
                    }
                }
                .buttonStyle(.borderedProminent)
                .font(.caption)
                .accessibilityLabel("Open Control Tower Main Window")

                Button("Refresh") {
                    viewModel.refresh()
                }
                .buttonStyle(.bordered)
                .font(.caption)
                .accessibilityLabel("Refresh Telemetry")

                Spacer()

                Button("Quit") {
                    NSApplication.shared.terminate(nil)
                }
                .buttonStyle(.plain)
                .font(.caption)
                .foregroundStyle(.secondary)
                .frame(minWidth: 36, minHeight: 28)
                .contentShape(Rectangle())
                .accessibilityLabel("Quit Pegasus Control Tower")
            }
        }
        .padding(14)
        .frame(width: 280)
    }
}

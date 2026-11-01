import SwiftUI

public struct TelemetryStreamView: View {
    @ObservedObject var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        VStack(spacing: 0) {
            // Top Controls Bar
            HStack(spacing: 12) {
                // Connection indicator
                HStack(spacing: 6) {
                    Circle()
                        .fill(viewModel.streamClient.isConnected ? Color.green : Color.orange)
                        .frame(width: 8, height: 8)
                    Text(viewModel.streamClient.isConnected ? "LIVE WS" : "RECONNECTING")
                        .font(.system(.caption2, design: .monospaced))
                        .fontWeight(.bold)
                }
                .padding(.horizontal, 8)
                .padding(.vertical, 4)
                .background(Color(NSColor.controlBackgroundColor))
                .cornerRadius(4)

                // Search field
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("Filter message, entity ID, or service...", text: $viewModel.searchQuery)
                        .textFieldStyle(.plain)
                        .accessibilityLabel("Filter telemetry stream")
                    if !viewModel.searchQuery.isEmpty {
                        Button {
                            viewModel.searchQuery = ""
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                        .buttonStyle(.plain)
                        .accessibilityLabel("Clear search query")
                    }
                }
                .padding(6)
                .background(Color(NSColor.controlBackgroundColor))
                .cornerRadius(6)

                // Severity Filter Picker
                Picker("Severity", selection: $viewModel.selectedSeverity) {
                    Text("All Severities").tag(LogSeverity?.none)
                    ForEach(LogSeverity.allCases) { sev in
                        Text(sev.rawValue).tag(LogSeverity?.some(sev))
                    }
                }
                .frame(width: 130)

                // Pause / Resume Toggle
                Button {
                    viewModel.streamClient.togglePause()
                } label: {
                    Label(
                        viewModel.streamClient.isPaused ? "Resume" : "Pause",
                        systemImage: viewModel.streamClient.isPaused ? "play.fill" : "pause.fill"
                    )
                }
                .buttonStyle(.bordered)

                // Export Diagnostic Bundle
                Button {
                    viewModel.exportDiagnosticBundle()
                } label: {
                    Label("Export Bundle", systemImage: "arrow.down.doc")
                }
                .buttonStyle(.bordered)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(Color(NSColor.windowBackgroundColor))

            Divider()

            // Event List
            if viewModel.filteredTelemetryEvents.isEmpty {
                VStack(spacing: 12) {
                    Spacer()
                    Image(systemName: "waveform.path.ecg")
                        .font(.system(size: 40))
                        .foregroundColor(.secondary)
                    Text("No telemetry events matching current filter")
                        .font(.headline)
                        .foregroundColor(.secondary)
                    Text("Adjust role perspective, severity selector, or search query.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                    Spacer()
                }
            } else {
                List(viewModel.filteredTelemetryEvents) { event in
                    VStack(alignment: .leading, spacing: 6) {
                        HStack(spacing: 10) {
                            severityBadge(event.severity)
                            Text(event.timestampIso)
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundColor(.secondary)
                            Text("• \(event.role)")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                            Text("[\(event.service)]")
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundColor(.teal)
                            Spacer()

                            // Copyable Trace ID
                            HStack(spacing: 4) {
                                Text(event.traceId)
                                    .font(.system(size: 10, design: .monospaced))
                                    .foregroundColor(.blue)
                                Button {
                                    viewModel.copyToClipboard(event.traceId, label: "Trace ID")
                                } label: {
                                    Image(systemName: "doc.on.doc")
                                        .font(.system(size: 9))
                                        .foregroundColor(.secondary)
                                }
                                .buttonStyle(.plain)
                            }
                        }

                        Text(event.message)
                            .font(.system(.body, design: .default))
                            .foregroundColor(.primary)

                        HStack(spacing: 8) {
                            Text("Entity:")
                                .font(.caption2)
                                .foregroundColor(.secondary)
                            Text(event.entityId)
                                .font(.system(.caption2, design: .monospaced))
                                .fontWeight(.semibold)
                            Button {
                                viewModel.copyToClipboard(event.entityId, label: "Entity ID")
                            } label: {
                                Image(systemName: "doc.on.doc")
                                    .font(.system(size: 9))
                                    .foregroundColor(.secondary)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(.vertical, 4)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Severity \(event.severity.rawValue), Service \(event.service), Role \(event.role), Time \(event.timestampIso)")
                    .accessibilityValue("\(event.message). Entity ID: \(event.entityId), Trace ID: \(event.traceId)")
                }
                .listStyle(.inset)
            }
        }
    }

    private func severityBadge(_ severity: LogSeverity) -> some View {
        let color: Color
        let icon: String
        switch severity {
        case .critical:
            color = .red
            icon = "exclamationmark.octagon.fill"
        case .error:
            color = .orange
            icon = "xmark.circle.fill"
        case .warn:
            color = .yellow
            icon = "exclamationmark.triangle.fill"
        case .info:
            color = .blue
            icon = "info.circle.fill"
        }

        return HStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 8, weight: .bold))
            Text(severity.rawValue)
                .font(.system(size: 9, weight: .bold, design: .monospaced))
        }
        .padding(.horizontal, 7)
        .padding(.vertical, 3)
        .background(color.opacity(0.15))
        .foregroundColor(color)
        .overlay(
            Capsule()
                .stroke(color.opacity(0.3), lineWidth: 1)
        )
        .clipShape(Capsule())
    }
}

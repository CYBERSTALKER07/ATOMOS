import SwiftUI

public struct SidebarView: View {
    @Bindable var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        List(selection: $viewModel.selectedSection) {
            Section("Ecosystem Operations") {
                ForEach(NavigationSection.allCases) { section in
                    NavigationLink(value: section) {
                        Label {
                            HStack {
                                Text(section.rawValue)
                                    .font(.system(.body, design: .default))
                                Spacer()
                                badgeForSection(section)
                            }
                        } icon: {
                            Image(systemName: section.icon)
                                .foregroundStyle(colorForSection(section))
                        }
                    }
                    .tag(section)
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Navigate to \(section.rawValue)")
                }
            }

            Section("Active Cloud Cell") {
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Circle()
                            .fill(Color.green)
                            .frame(width: 8, height: 8)
                        Text("cell-uz-tashkent")
                            .font(.system(.caption, design: .monospaced))
                            .fontWeight(.semibold)
                    }
                    Text("P99 Latency: 4.2ms | 16 Nodes")
                        .font(.caption2)
                        .foregroundStyle(.secondary)
                }
                .padding(.vertical, 4)
                .accessibilityElement(children: .combine)
                .accessibilityLabel("Active Cloud Cell cell-uz-tashkent, P99 Latency 4.2 milliseconds, 16 Nodes")
            }
        }
        .listStyle(.sidebar)
        .frame(minWidth: 220, idealWidth: 240)
    }

    @ViewBuilder
    private func badgeForSection(_ section: NavigationSection) -> some View {
        switch section {
        case .infra:
            Text("\(viewModel.cells.count) Cells")
                .font(.system(.caption2, design: .monospaced).weight(.semibold))
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.blue.opacity(0.15))
                .foregroundStyle(.blue)
                .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))
        case .telemetry:
            let count = viewModel.filteredTelemetryEvents.count
            Text("\(count)")
                .font(.system(.caption2, design: .monospaced).weight(.semibold))
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.green.opacity(0.15))
                .foregroundStyle(.green)
                .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))
        case .finops:
            Text("FinOps")
                .font(.caption2.weight(.semibold))
                .padding(.horizontal, 6)
                .padding(.vertical, 2)
                .background(Color.purple.opacity(0.15))
                .foregroundStyle(.purple)
                .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))
        default:
            EmptyView()
        }
    }

    private func colorForSection(_ section: NavigationSection) -> Color {
        switch section {
        case .overview: return .blue
        case .infra: return .teal
        case .finops: return .purple
        case .telemetry: return .green
        case .analytics: return .orange
        }
    }
}

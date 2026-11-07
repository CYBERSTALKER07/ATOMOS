import SwiftUI

public struct MainSplitView: View {
    @ObservedObject var viewModel: ControlTowerViewModel

    public init(viewModel: ControlTowerViewModel) {
        self.viewModel = viewModel
    }

    public var body: some View {
        NavigationSplitView {
            SidebarView(viewModel: viewModel)
        } detail: {
            ZStack(alignment: .bottom) {
                switch viewModel.selectedSection {
                case .overview:
                    OverviewView(viewModel: viewModel)
                case .infra:
                    InfraHealthView(viewModel: viewModel)
                case .finops:
                    FinOpsView(viewModel: viewModel)
                case .telemetry:
                    TelemetryStreamView(viewModel: viewModel)
                case .analytics:
                    AnalyticsScorecardView(viewModel: viewModel)
                }

                // Copy Toast Notification
                if let toast = viewModel.copiedToastMessage {
                    HStack(spacing: 8) {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundColor(.green)
                        Text(toast)
                            .font(.system(.caption, design: .monospaced))
                            .fontWeight(.medium)
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(.ultraThinMaterial)
                    .cornerRadius(8)
                    .shadow(radius: 6)
                    .padding(.bottom, 20)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                }
            }
        }
        .inspector(isPresented: .constant(true)) {
            InspectorView(viewModel: viewModel)
        }
        .toolbar {
            ToolbarItem(placement: .automatic) {
                Picker("Role", selection: $viewModel.selectedRole) {
                    ForEach(RolePerspective.allCases) { role in
                        HStack {
                            Image(systemName: role.systemIcon)
                            Text(role.rawValue)
                        }
                        .tag(role)
                    }
                }
            }

            ToolbarItem(placement: .automatic) {
                Button {
                    Task { await viewModel.initialLoad() }
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .keyboardShortcut("r", modifiers: .command)
            }
        }
        .task {
            await viewModel.initialLoad()
            viewModel.streamClient.connect()
        }
    }
}

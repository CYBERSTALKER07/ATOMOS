import SwiftUI

public struct MainSplitView: View {
    @Bindable var viewModel: ControlTowerViewModel

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
                            .foregroundStyle(.green)
                        Text(toast)
                            .font(.system(.caption, design: .monospaced))
                            .fontWeight(.medium)
                    }
                    .padding(.horizontal, 14)
                    .padding(.vertical, 8)
                    .background(.ultraThinMaterial)
                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                    .shadow(radius: 6)
                    .padding(.bottom, 20)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel(toast)
                }
            }
        }
        .inspector(isPresented: $viewModel.isInspectorPresented) {
            InspectorView(viewModel: viewModel)
        }
        .sheet(isPresented: $viewModel.isCheatsheetPresented) {
            DesktopKeyboardCheatsheetView(isPresented: $viewModel.isCheatsheetPresented)
        }
        .toolbar {
            ToolbarItem(placement: .status) {
                HStack(spacing: 8) {
                    HStack(spacing: 6) {
                        Circle()
                            .fill(viewModel.streamClient.isConnected ? Color.green : Color.orange)
                            .frame(width: 7, height: 7)
                        Text(viewModel.telemetryHUDString)
                            .font(.system(.caption, design: .monospaced).weight(.semibold))
                            .foregroundStyle(.secondary)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 3)
                    .background(Color.primary.opacity(0.05))
                    .overlay(
                        RoundedRectangle(cornerRadius: 6, style: .continuous)
                            .stroke(Color.primary.opacity(0.10), lineWidth: 1)
                    )
                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                    .accessibilityElement(children: .combine)
                    .accessibilityLabel("Telemetry HUD: \(viewModel.telemetryHUDString)")

                    Button {
                        ControlTowerHaptics.alignment()
                        viewModel.isCheatsheetPresented = true
                    } label: {
                        Image(systemName: "keyboard")
                            .foregroundStyle(.secondary)
                            .frame(minWidth: 32, minHeight: 32)
                            .contentShape(Rectangle())
                    }
                    .help("Keyboard Shortcuts (⌘/)")
                    .accessibilityLabel("Keyboard Shortcuts Cheatsheet")
                    .buttonStyle(.plain)
                }
            }

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
                .accessibilityLabel("Role Perspective")
            }

            ToolbarItem(placement: .automatic) {
                Button {
                    viewModel.refresh()
                } label: {
                    Label("Refresh", systemImage: "arrow.clockwise")
                }
                .accessibilityLabel("Refresh Telemetry")
                .keyboardShortcut("r", modifiers: .command)
            }
        }
        .task {
            await viewModel.initialLoad()
            viewModel.streamClient.connect()
        }
    }
}

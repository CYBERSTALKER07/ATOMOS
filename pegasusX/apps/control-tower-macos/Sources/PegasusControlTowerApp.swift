import AppKit
import SwiftUI

@main
struct PegasusControlTowerApp: App {
    @StateObject private var viewModel = ControlTowerViewModel()

    var body: some Scene {
        WindowGroup {
            MainSplitView(viewModel: viewModel)
        }
        .windowStyle(.titleBar)
        .windowToolbarStyle(.unified(showsTitle: true))
        .defaultSize(width: 1440, height: 900)
        .commands {
            CommandGroup(replacing: .newItem) {}

            CommandMenu("Navigate") {
                Button("Overview") { viewModel.selectedSection = .overview }
                    .keyboardShortcut("1", modifiers: .command)
                Button("Infrastructure") { viewModel.selectedSection = .infra }
                    .keyboardShortcut("2", modifiers: .command)
                Button("FinOps & Billing") { viewModel.selectedSection = .finops }
                    .keyboardShortcut("3", modifiers: .command)
                Button("Live Telemetry") { viewModel.selectedSection = .telemetry }
                    .keyboardShortcut("4", modifiers: .command)
                Button("Scorecards") { viewModel.selectedSection = .analytics }
                    .keyboardShortcut("5", modifiers: .command)

                Divider()

                Button("Refresh Telemetry") {
                    Task { await viewModel.initialLoad() }
                }
                .keyboardShortcut("r", modifiers: .command)
            }

            CommandMenu("FinOps") {
                Button("Export Tender Waterfall CSV…") {
                    viewModel.exportWaterfallCSV()
                }
                .keyboardShortcut("e", modifiers: [.command, .shift])

                Button("Export Diagnostic Bundle…") {
                    viewModel.exportDiagnosticBundle()
                }
                .keyboardShortcut("d", modifiers: [.command, .shift])
            }
        }

        // Native macOS Menu Bar Extra (Top System Status Bar)
        MenuBarExtra("Pegasus Control Tower", systemImage: "antenna.radiowaves.left.and.right") {
            MenuBarStatusView(viewModel: viewModel)
        }
        .menuBarExtraStyle(.window)
    }
}

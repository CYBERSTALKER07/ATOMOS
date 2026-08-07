import SwiftUI

struct ManifestExceptionsSheet: View {
    @Bindable var viewModel: HomeViewModel

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.loadingExceptions && viewModel.manifestExceptions.isEmpty {
                    PayloadLoadingView(
                        title: "LOADING EXCEPTIONS",
                        message: "Fetching overflow, damaged, and manual removals."
                    )
                } else if viewModel.manifestExceptions.isEmpty {
                    PayloadStateView(
                        variant: .warning,
                        title: "NO_EXCEPTIONS",
                        message: "Overflow, damaged, and manual removals appear here.",
                        compact: false
                    )
                } else {
                    ScrollView {
                        VStack(spacing: 16) {
                            ForEach(viewModel.manifestExceptions) { row in
                                VStack(alignment: .leading, spacing: TermTheme.s8) {
                                    HStack(spacing: TermTheme.s8) {
                                        PayloadStatusBadge(text: row.reason)
                                        if row.escalated {
                                            PayloadStatusBadge(text: "ESCALATED", tint: TermTheme.alert)
                                        }
                                    }
                                    Text(L10n.format("mobile_payload.ui.order_prefix_manifest_prefix_2", "\(row.orderId.prefix(8))", "\(row.manifestId.prefix(8))"))
                                        .font(.system(size: 13, weight: .medium, design: .monospaced))
                                        .foregroundStyle(TermTheme.secondary)
                                    Text(L10n.format("mobile_payload.ui.attempts_attemptcount", "\(row.attemptCount)"))
                                        .font(.system(size: 11, weight: .bold, design: .monospaced))
                                        .foregroundStyle(TermTheme.tertiary)
                                }
                                .padding(.vertical, TermTheme.s4)
                            }
                        }
                        .padding()
                    }
                }
            }
            .navigationTitle("mobile_payload.ui.manifest_exceptions")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close") { viewModel.toggleExceptionsPanel() }
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        Task { await viewModel.loadManifestExceptions() }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                    }
                }
            }
        }
    }
}

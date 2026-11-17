import SwiftUI

struct ManifestExceptionsSheet: View {
    @Bindable var viewModel: HomeViewModel
    @State private var exceptionsViewModel = ExceptionsViewModel()

    var body: some View {
        NavigationStack {
            VStack {
                VStack(alignment: .leading, spacing: 8) {
                    Text("Report Dock Damage")
                        .font(.headline)
                        .foregroundStyle(Color(red: 0.6, green: 0.1, blue: 0.1))

                    Button(action: {
                        if let selectedOrder = viewModel.selectedOrderId,
                           let manifestId = viewModel.manifest?.manifestId {
                            PegasusHaptics.warning()
                            Task {
                                await viewModel.reportException(orderId: selectedOrder, reason: "DAMAGED")
                                await exceptionsViewModel.reportDockDamage(orderId: selectedOrder, manifestId: manifestId)
                            }
                        }
                    }) {
                        Text("MARK DAMAGED")
                            .font(.headline)
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity, minHeight: 44)
                            .padding()
                            .background(Color.red)
                            .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                    }
                    .buttonStyle(.tactical)
                    .disabled(viewModel.selectedOrderId == nil)
                    .accessibilityLabel("Mark selected order as dock damaged")
                }
                .padding()
                .background(Color(red: 1.0, green: 0.9, blue: 0.9))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                .padding(.horizontal)
                .padding(.top, 8)

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
                                            .font(.system(.footnote, design: .monospaced, weight: .medium))
                                            .foregroundStyle(TermTheme.secondary)
                                        Text(L10n.format("mobile_payload.ui.attempts_attemptcount", "\(row.attemptCount)"))
                                            .font(.system(.caption, design: .monospaced, weight: .bold))
                                            .foregroundStyle(TermTheme.tertiary)
                                    }
                                    .padding(.vertical, TermTheme.s4)
                                    .frame(maxWidth: .infinity, alignment: .leading)
                                    .accessibilityElement(children: .combine)
                                    .accessibilityLabel("\(row.reason) exception on order \(row.orderId.prefix(8)), attempts: \(row.attemptCount)\(row.escalated ? ", escalated" : "")")
                                }
                            }
                            .padding()
                        }
                        .scrollDismissesKeyboard(.interactively)
                    }
                }
            }
            .navigationTitle("mobile_payload.ui.manifest_exceptions")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("common.action.close") {
                        PegasusHaptics.light()
                        viewModel.toggleExceptionsPanel()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        PegasusHaptics.light()
                        Task {
                            await viewModel.loadManifestExceptions()
                            await exceptionsViewModel.loadExceptions()
                        }
                    } label: {
                        Image(systemName: "arrow.clockwise")
                            .frame(minWidth: 44, minHeight: 44)
                    }
                    .accessibilityLabel("Refresh exceptions")
                }
            }
        }
    }
}

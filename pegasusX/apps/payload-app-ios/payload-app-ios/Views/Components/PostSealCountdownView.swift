import SwiftUI

struct PostSealCountdownView: View {
    let orderId: String
    let dispatchCode: String
    let secondsLeft: Int
    let reportingMissingItems: Bool
    let onDismiss: () -> Void
    let onReportMissingItems: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            PayloadStateView(
                variant: .warning,
                title: "ORDER_SEALED",
                message: "Double-check the load before dispatch.",
                compact: true,
                tone: .warning
            )

            HStack {
                Image(systemName: "lock.shield.fill")
                    .foregroundStyle(TermTheme.live)
                Text("ORDER_SEALED")
                    .font(.system(.caption, design: .monospaced, weight: .black))
                    .foregroundStyle(TermTheme.secondary)
                Spacer()
                Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(orderId.suffix(6).uppercased())"))
                    .font(.system(.caption, design: .monospaced, weight: .black))
                    .foregroundStyle(TermTheme.accent)
            }
            .accessibilityElement(children: .combine)

            VStack(alignment: .leading, spacing: 4) {
                Text("DISPATCH_CODE")
                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                    .foregroundStyle(TermTheme.secondary)

                Text(dispatchCode)
                    .font(.system(.largeTitle, design: .monospaced, weight: .black))
                    .foregroundStyle(TermTheme.live)
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel("Dispatch code \(dispatchCode)")

            HStack {
                Text(L10n.format("mobile_payload.ui.double_check_window_secondslefts_2", "\(secondsLeft)"))
                    .font(.system(.caption, design: .monospaced, weight: .bold))
                    .foregroundStyle(TermTheme.accent)
                Spacer()
            }

            ProgressView(value: min(max(Double(secondsLeft) / 60.0, 0), 1))
                .tint(TermTheme.live)

            Button {
                PegasusHaptics.warning()
                onReportMissingItems()
            } label: {
                HStack {
                    if reportingMissingItems {
                        ProgressView().controlSize(.small)
                    }
                    Text("REPORT_MISSING_ITEMS")
                        .font(.system(.subheadline, design: .monospaced, weight: .black))
                }
                .frame(maxWidth: .infinity, minHeight: 44)
                .padding()
                .background(TermTheme.card)
                .foregroundStyle(TermTheme.accent)
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
            .buttonStyle(.tactical)
            .disabled(reportingMissingItems)
            .accessibilityLabel("Report missing items")

            Button {
                PegasusHaptics.light()
                onDismiss()
            } label: {
                Text("CONTINUE_TO_NEXT")
                    .font(.system(.subheadline, design: .monospaced, weight: .black))
                    .frame(maxWidth: .infinity, minHeight: 44)
                    .padding()
                    .background(TermTheme.accent)
                    .foregroundStyle(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            }
            .buttonStyle(.tactical)
            .accessibilityLabel("Continue to next order")
        }
        .padding(20)
        .background(TermTheme.live.opacity(0.05))
        .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 24, style: .continuous)
                .stroke(TermTheme.live.opacity(0.2), lineWidth: 1)
        }
    }
}

struct AllSealedSuccessView: View {
    let dispatchCodes: [String: String]
    let onStartNew: () -> Void

    var body: some View {
        ScrollView {
            VStack(spacing: 32) {
                PayloadStateView(
                    variant: .success,
                    title: "MANIFEST_LOCKED",
                    message: "All items verified and sealed for transport.",
                    tone: .success
                )
                .padding(.top, 40)

                VStack(alignment: .leading, spacing: 16) {
                    Text("DISPATCH_MANIFEST_SUMMARY")
                        .font(.system(.caption, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.tertiary)

                    VStack(spacing: 8) {
                        ForEach(dispatchCodes.sorted(by: { $0.key < $1.key }), id: \.key) { id, code in
                            HStack {
                                Text(L10n.format("mobile_payload.ui.ord_uppercased", "\(id.suffix(6).uppercased())"))
                                    .font(.system(.subheadline, design: .monospaced, weight: .black))
                                    .foregroundStyle(TermTheme.accent)
                                Spacer()
                                Text(code)
                                    .font(.system(.headline, design: .monospaced, weight: .black))
                                    .foregroundStyle(TermTheme.live)
                            }
                            .padding(16)
                            .frame(minHeight: 48)
                            .background(TermTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                            .overlay {
                                RoundedRectangle(cornerRadius: 12, style: .continuous)
                                    .stroke(TermTheme.separator.opacity(0.1), lineWidth: 1)
                            }
                            .accessibilityElement(children: .combine)
                            .accessibilityLabel("Order \(id.suffix(6)), dispatch code \(code)")
                        }
                    }
                }
                .padding(24)
                .background(TermTheme.bg)
                .clipShape(RoundedRectangle(cornerRadius: 24, style: .continuous))
                .overlay {
                    RoundedRectangle(cornerRadius: 24, style: .continuous)
                        .stroke(TermTheme.separator.opacity(0.1), lineWidth: 1)
                }

                Button {
                    PegasusHaptics.medium()
                    onStartNew()
                } label: {
                    HStack {
                        Image(systemName: "plus.circle.fill")
                        Text("START_NEXT_LOAD")
                            .font(.system(.headline, design: .monospaced, weight: .black))
                    }
                    .frame(maxWidth: .infinity, minHeight: 52)
                    .padding(.vertical, 12)
                    .background(TermTheme.accent)
                    .foregroundStyle(TermTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                }
                .buttonStyle(.tactical)
                .accessibilityLabel("Start next manifest load")
            }
            .padding(24)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(TermTheme.bg)
    }
}

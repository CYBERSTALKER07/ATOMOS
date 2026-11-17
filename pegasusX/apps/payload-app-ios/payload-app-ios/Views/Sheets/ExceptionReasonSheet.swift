import SwiftUI

struct ExceptionReasonSheet: View {
    let orderId: String
    let inFlight: Bool
    let onCancel: () -> Void
    let onSelect: (String) -> Void

    private let reasons: [(code: String, label: String, icon: String)] = [
        ("OVERFLOW", "OVERFLOW - NO CAPACITY", "shippingbox.fill"),
        ("DAMAGED", "DAMAGED - QUALITY FAIL", "exclamationmark.shield.fill"),
        ("MANUAL", "MANUAL - OPERATOR VOID", "hand.raised.fill"),
    ]

    var body: some View {
        ZStack {
            TermTheme.bg.ignoresSafeArea()

            ScrollView {
                VStack(spacing: 24) {
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text("EXCEPTION_REPORT")
                                .font(.system(.caption, design: .monospaced, weight: .black))
                                .foregroundStyle(TermTheme.secondary)
                            Text(L10n.format("mobile_payload.ui.remove_ord_uppercased", "\(orderId.suffix(6).uppercased())"))
                                .font(.system(.title3, design: .monospaced, weight: .black))
                                .foregroundStyle(TermTheme.warn)
                        }
                        Spacer()
                        Button {
                            PegasusHaptics.light()
                            onCancel()
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .font(.system(.title))
                                .frame(minWidth: 44, minHeight: 44)
                                .foregroundStyle(TermTheme.tertiary)
                        }
                        .accessibilityLabel("Cancel exception")
                    }
                    .padding(.horizontal, 4)

                    VStack(alignment: .leading, spacing: 16) {
                        Text("SELECT_EXCEPTION_REASON")
                            .font(.system(.caption2, design: .monospaced, weight: .bold))
                            .foregroundStyle(TermTheme.secondary)

                        VStack(spacing: 12) {
                            ForEach(reasons, id: \.code) { reason in
                                Button {
                                    PegasusHaptics.warning()
                                    onSelect(reason.code)
                                } label: {
                                    HStack(spacing: 16) {
                                        Image(systemName: reason.icon)
                                            .font(.system(.title3))
                                            .foregroundStyle(TermTheme.warn)
                                            .frame(width: 44, height: 44)
                                            .background(TermTheme.warn.opacity(0.1), in: RoundedRectangle(cornerRadius: 12, style: .continuous))

                                        Text(reason.label)
                                            .font(.system(.subheadline, design: .monospaced, weight: .black))
                                            .foregroundStyle(TermTheme.accent)

                                        Spacer()

                                        if inFlight {
                                            ProgressView().tint(TermTheme.accent)
                                        } else {
                                            Image(systemName: "chevron.right")
                                                .font(.system(.subheadline, weight: .bold))
                                                .foregroundStyle(TermTheme.tertiary)
                                        }
                                    }
                                    .padding(12)
                                    .frame(minWidth: 44, minHeight: 48)
                                    .background(TermTheme.card)
                                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                                    .tacticalCard()
                                }
                                .disabled(inFlight)
                                .buttonStyle(.tactical)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel(reason.label)
                            }
                        }

                        Text("mobile_payload.ui.3_overflow_attempts_on_this_manifest_will_escalate_to_admin_dlq")
                            .font(.system(.caption, design: .monospaced, weight: .medium))
                            .foregroundStyle(TermTheme.tertiary)
                            .padding(.horizontal, 4)
                            .padding(.top, 8)
                    }
                }
                .padding(24)
            }
            .scrollDismissesKeyboard(.interactively)
        }
    }
}

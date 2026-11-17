import SwiftUI

struct ReDispatchSheet: View {
    let orderId: String
    let loading: Bool
    let response: RecommendReassignResponse?
    let reassigning: Bool
    let onClose: () -> Void
    let onPick: (String, Bool) -> Void

    var body: some View {
        ZStack {
            TermTheme.bg.ignoresSafeArea()

            VStack(spacing: 24) {
                HStack {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("LOGISTICS_OPTIMIZER")
                            .font(.system(.caption, design: .monospaced, weight: .black))
                            .foregroundStyle(TermTheme.secondary)
                        Text(L10n.format("mobile_payload.ui.re_dispatch_ord_uppercased", "\(orderId.suffix(6).uppercased())"))
                            .font(.system(.title3, design: .monospaced, weight: .black))
                            .foregroundStyle(TermTheme.accent)
                    }
                    Spacer()
                    Button {
                        PegasusHaptics.light()
                        onClose()
                    } label: {
                        Image(systemName: "xmark.circle.fill")
                            .font(.system(.title))
                            .frame(minWidth: 44, minHeight: 44)
                            .foregroundStyle(TermTheme.tertiary)
                    }
                    .accessibilityLabel("Close re-dispatch")
                }
                .padding(.horizontal, 4)

                Group {
                    if loading {
                        PayloadStateView(
                            variant: .dispatch,
                            title: "SOLVING_CONSTRAINTS...",
                            message: "Scoring nearby trucks for the best reassignment path.",
                            compact: true
                        )
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    } else if let resp = response {
                        ScrollView {
                            VStack(spacing: 20) {
                                VStack(alignment: .leading, spacing: 8) {
                                    Text("TARGET_OBJECT")
                                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                                        .foregroundStyle(TermTheme.secondary)

                                    HStack {
                                        Text(resp.retailerName?.uppercased() ?? "OFFLINE_RETAILER")
                                            .font(.system(.callout, design: .monospaced, weight: .black))
                                            .foregroundStyle(TermTheme.accent)
                                        Spacer()
                                        Text(String(format: "%.1f VU", resp.orderVolumeVu ?? 0))
                                            .font(.system(.callout, design: .monospaced, weight: .black))
                                            .foregroundStyle(TermTheme.accent)
                                    }
                                }
                                .padding(16)
                                .background(TermTheme.card)
                                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                                .tacticalCard()
                                .accessibilityElement(children: .combine)

                                if resp.recommendations.isEmpty {
                                    PayloadStateView(
                                        variant: .dispatch,
                                        title: "NO_SUITABLE_CARRIERS_FOUND",
                                        message: "No nearby fleet target can accept this order right now.",
                                        compact: true,
                                        tone: .warning
                                    )
                                    .padding(40)
                                    .frame(maxWidth: .infinity)
                                    .background(TermTheme.card)
                                    .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                                    .tacticalCard()
                                } else {
                                    VStack(alignment: .leading, spacing: 12) {
                                        Text("AI_RECOMMENDATIONS")
                                            .font(.system(.caption, design: .monospaced, weight: .black))
                                            .foregroundStyle(TermTheme.secondary)
                                            .padding(.horizontal, 4)

                                        VStack(spacing: 12) {
                                            ForEach(resp.recommendations) { rec in
                                                RecommendationRow(
                                                    rec: rec,
                                                    onPickComplete: { onPick(rec.driverId, false) },
                                                    onPickPartial: { onPick(rec.driverId, true) }
                                                )
                                                .disabled(reassigning)
                                            }
                                        }
                                    }
                                }
                            }
                        }
                        .scrollDismissesKeyboard(.interactively)
                    } else {
                        PayloadStateView(
                            variant: .dispatch,
                            title: "NO_DATA_AVAILABLE",
                            message: "The optimizer response is not available yet.",
                            compact: true,
                            tone: .warning
                        )
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                    }
                }
            }
            .padding(24)

            if reassigning {
                TermTheme.bg.opacity(0.8).ignoresSafeArea()
                VStack(spacing: 16) {
                    ProgressView().tint(TermTheme.accent)
                    Text("mobile_payload.ui.reassigning_order")
                        .font(.system(.caption, design: .monospaced, weight: .black))
                        .foregroundStyle(TermTheme.accent)
                }
            }
        }
    }
}

import SwiftUI

struct TacticalWorkbenchView: View {
    let section: WarehouseSection
    var onInspect: (() -> Void)? = nil

    private let stages = ["Inbound ASN", "Dock Receiving", "Pick Waves", "Staging Lanes", "Gate Dispatch"]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                HStack(alignment: .center) {
                    VStack(alignment: .leading, spacing: 4) {
                        HStack(spacing: 8) {
                            Text("ACTIVE WORKBENCH")
                                .tacticalMicroLabel()
                            TacticalStatusBadge(status: "OPERATIONAL", showPulse: true)
                        }

                        Text("\(section.rawValue) Workbench")
                            .font(.system(.title2, weight: .bold))
                            .foregroundStyle(TacticalTheme.textPrimary)

                        Text("ZONE TASHKENT-NORTH-01 · LIVE DISPATCH HUB")
                            .font(.system(.caption2, design: .monospaced, weight: .semibold))
                            .foregroundStyle(TacticalTheme.textTertiary)
                    }

                    Spacer()

                    Button {
                        PegasusHaptics.light()
                        onInspect?()
                    } label: {
                        HStack(spacing: 6) {
                            Image(systemName: "sidebar.right")
                            Text("TELEMETRY INSPECTOR")
                                .font(.system(.caption2, weight: .bold))
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 8)
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                        .foregroundStyle(TacticalTheme.accentCobalt)
                        .background(TacticalTheme.accentCobalt.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(
                            RoundedRectangle(cornerRadius: 8, style: .continuous)
                                .stroke(TacticalTheme.accentCobalt.opacity(0.4), lineWidth: 1)
                        )
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Toggle Telemetry Inspector")
                }
                .tacticalCard()

                VStack(alignment: .leading, spacing: 10) {
                    Text("FULFILLMENT PIPELINE STAGE")
                        .tacticalMicroLabel()

                    TacticalStageStepper(stages: stages, currentStageIndex: 2)
                }
                .tacticalCard()

                LazyVGrid(
                    columns: [
                        GridItem(.flexible(minimum: 180), spacing: 14),
                        GridItem(.flexible(minimum: 180), spacing: 14),
                        GridItem(.flexible(minimum: 180), spacing: 14)
                    ],
                    spacing: 14
                ) {
                    TacticalGaugeCard(
                        title: "DOCK BAY CAPACITY",
                        percentage: 0.75,
                        subtitle: "6 OF 8 BAYS OCCUPIED",
                        tint: TacticalTheme.accentCobalt
                    )

                    TacticalGaugeCard(
                        title: "STAGED VOLUME (VU)",
                        percentage: 0.62,
                        subtitle: "18,400 / 30,000 VU",
                        tint: TacticalTheme.accentLime
                    )

                    TacticalGaugeCard(
                        title: "COLD CHAIN COMPLIANCE",
                        percentage: 0.992,
                        subtitle: "ALL ZONES WITHIN SLA",
                        tint: TacticalTheme.accentCyan
                    )
                }

                VStack(alignment: .leading, spacing: 14) {
                    HStack {
                        Text("ACTIVE STAGING LANES")
                            .tacticalMicroLabel()
                        Spacer()
                        Text("AUTO-REFRESH: 300MS")
                            .font(.system(.caption2, design: .monospaced, weight: .bold))
                            .foregroundStyle(TacticalTheme.accentCyan)
                    }

                    VStack(spacing: 10) {
                        stagingLaneRow(
                            lane: "LANE 01 · TASHKENT NORTH",
                            truck: "ISUZU NPR 75 (01 772 AAA)",
                            driver: "F. KASIMOV",
                            ordersCount: 14,
                            progress: (current: 11, total: 14),
                            eta: "DEPARTURE: 14:35",
                            status: "LOADING"
                        )

                        stagingLaneRow(
                            lane: "LANE 02 · SAMARKAND EXPRESS",
                            truck: "KAMAZ REEFER (01 909 BBB)",
                            driver: "S. AKHMEDOV",
                            ordersCount: 8,
                            progress: (current: 8, total: 8),
                            eta: "DEPARTURE: 15:00",
                            status: "SEALED"
                        )

                        stagingLaneRow(
                            lane: "LANE 03 · CHILANZAR CROSS-DOCK",
                            truck: "GAZELLE NEXT (01 341 CCC)",
                            driver: "A. RAKHIMOV",
                            ordersCount: 22,
                            progress: (current: 16, total: 22),
                            eta: "DEPARTURE: 15:20",
                            status: "STAGED"
                        )
                    }
                }
                .tacticalCard()
            }
            .padding(18)
        }
        .background(TacticalTheme.canvas)
    }

    @ViewBuilder
    private func stagingLaneRow(
        lane: String,
        truck: String,
        driver: String,
        ordersCount: Int,
        progress: (current: Int, total: Int),
        eta: String,
        status: String
    ) -> some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack {
                VStack(alignment: .leading, spacing: 2) {
                    Text(lane)
                        .font(.system(.footnote, weight: .bold))
                        .foregroundStyle(TacticalTheme.textPrimary)

                    HStack(spacing: 8) {
                        Text(truck)
                            .font(.system(.caption2, design: .monospaced, weight: .semibold))
                            .foregroundStyle(TacticalTheme.textSecondary)

                        Text("·")
                            .foregroundStyle(TacticalTheme.textTertiary)

                        Text(driver)
                            .font(.system(.caption2, weight: .medium))
                            .foregroundStyle(TacticalTheme.accentCobalt)
                    }
                }

                Spacer()

                VStack(alignment: .trailing, spacing: 4) {
                    TacticalStatusBadge(status: status)
                    Text(eta)
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .foregroundStyle(TacticalTheme.textTertiary)
                }
            }

            TacticalLedProgressBar(
                title: "PALLET LOAD VERIFICATION",
                current: progress.current,
                total: progress.total,
                tint: status == "SEALED" ? TacticalTheme.statusSuccess : TacticalTheme.accentCobalt
            )
        }
        .padding(12)
        .background(TacticalTheme.surfaceSubtle)
        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
        .overlay(
            RoundedRectangle(cornerRadius: 10, style: .continuous)
                .stroke(TacticalTheme.border, lineWidth: 1)
        )
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(lane), truck \(truck), driver \(driver), status \(status), \(progress.current) of \(progress.total) pallets loaded, \(eta)")
    }
}

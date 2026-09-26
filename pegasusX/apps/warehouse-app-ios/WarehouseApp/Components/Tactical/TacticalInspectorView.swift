import SwiftUI

struct TacticalInspectorView: View {
    let title: String
    let subtitle: String
    var status: String = "ACTIVE"
    var temperature: Double? = -18.4
    var humidity: Int? = 82
    var battery: Int? = 94
    var boltSealCode: String? = "UZ-SEAL-998241"
    var onAction: ((String) -> Void)? = nil

    private let sampleTimeline: [TacticalTimelineItem] = [
        TacticalTimelineItem(
            id: "1",
            timestamp: "14:45:12",
            title: "Staging Scan Confirmed",
            description: "Pallet 4 of 6 passed barcode verification at Bay 3.",
            actor: "J.DOE (FORKLIFT-02)",
            tint: TacticalTheme.statusSuccess
        ),
        TacticalTimelineItem(
            id: "2",
            timestamp: "14:32:00",
            title: "Cold-Chain Integrity Check",
            description: "Reefer compartment pre-cooled to -18.4°C. Threshold OK.",
            actor: "IOT-PROBE-TH-09",
            tint: TacticalTheme.accentCyan
        ),
        TacticalTimelineItem(
            id: "3",
            timestamp: "14:10:45",
            title: "Pick Wave Dispatched",
            description: "Route TASHKENT-NORTH-WAVE-2 generated and assigned.",
            actor: "AI-OPTIMIZER",
            tint: TacticalTheme.accentCobalt
        )
    ]

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                // Header Capsule
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Text("CONTROL TOWER INSPECTOR")
                            .tacticalMicroLabel()
                        Spacer()
                        TacticalStatusBadge(status: status)
                    }

                    Text(title)
                        .font(.system(size: 18, weight: .bold))
                        .foregroundStyle(TacticalTheme.textPrimary)

                    Text(subtitle)
                        .font(.system(size: 12, weight: .semibold, design: .monospaced))
                        .foregroundStyle(TacticalTheme.textTertiary)
                }
                .tacticalCard()

                // Cold Chain & Telematics IoT Sensor Card
                VStack(alignment: .leading, spacing: 12) {
                    Text("LIVE SENSOR TELEMETRY")
                        .tacticalMicroLabel()

                    HStack(spacing: 12) {
                        if let temp = temperature {
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(spacing: 4) {
                                    Image(systemName: "thermometer.snowflake")
                                        .foregroundStyle(TacticalTheme.accentCyan)
                                    Text("REEFER")
                                        .tacticalMicroLabel()
                                }
                                Text(String(format: "%.1f°C", temp))
                                    .font(.system(size: 18, weight: .bold, design: .monospaced))
                                    .foregroundStyle(TacticalTheme.accentCyan)
                                Text("THRESHOLD: -22°C to -15°C")
                                    .font(.system(size: 9, weight: .bold))
                                    .foregroundStyle(TacticalTheme.statusSuccess)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        if let hum = humidity {
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(spacing: 4) {
                                    Image(systemName: "humidity")
                                        .foregroundStyle(TacticalTheme.accentCobalt)
                                    Text("HUMIDITY")
                                        .tacticalMicroLabel()
                                }
                                Text("\(hum)% RH")
                                    .font(.system(size: 18, weight: .bold, design: .monospaced))
                                    .foregroundStyle(TacticalTheme.textPrimary)
                                Text("CONDENSATION SAFE")
                                    .font(.system(size: 9, weight: .bold))
                                    .foregroundStyle(TacticalTheme.statusSuccess)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }

                        if let bat = battery {
                            VStack(alignment: .leading, spacing: 4) {
                                HStack(spacing: 4) {
                                    Image(systemName: "battery.75")
                                        .foregroundStyle(TacticalTheme.accentLime)
                                    Text("BATTERY")
                                        .tacticalMicroLabel()
                                }
                                Text("\(bat)%")
                                    .font(.system(size: 18, weight: .bold, design: .monospaced))
                                    .foregroundStyle(TacticalTheme.textPrimary)
                                Text("18.4 HOURS EST.")
                                    .font(.system(size: 9, weight: .bold))
                                    .foregroundStyle(TacticalTheme.accentLime)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                        }
                    }
                }
                .tacticalCard()

                // Security & Bolt Seal Verification
                if let seal = boltSealCode {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("SECURITY & BOLT SEAL")
                            .tacticalMicroLabel()

                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(seal)
                                    .font(.system(size: 14, weight: .bold, design: .monospaced))
                                    .foregroundStyle(TacticalTheme.accentOrange)
                                Text("TAMPER-EVIDENT RFID BOLT")
                                    .font(.system(size: 10, weight: .medium))
                                    .foregroundStyle(TacticalTheme.textTertiary)
                            }
                            Spacer()
                            Image(systemName: "checkmark.shield.fill")
                                .font(.system(size: 20))
                                .foregroundStyle(TacticalTheme.statusSuccess)
                        }
                    }
                    .tacticalCard()
                }

                // Rapid Operational Action Triggers
                VStack(alignment: .leading, spacing: 10) {
                    Text("SUPERVISOR ACTIONS")
                        .tacticalMicroLabel()

                    Button {
                        onAction?("print_zpl")
                    } label: {
                        HStack {
                            Image(systemName: "printer.fill")
                            Text("PRINT ZPL THERMAL LABELS")
                                .font(.system(size: 11, weight: .bold))
                            Spacer()
                            Text("DOCK PRINTER 2")
                                .font(.system(size: 10, weight: .bold, design: .monospaced))
                                .foregroundStyle(TacticalTheme.accentCobalt)
                        }
                        .padding(12)
                        .background(TacticalTheme.surfaceSubtle)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).stroke(TacticalTheme.border, lineWidth: 1))
                    }
                    .buttonStyle(.plain)

                    Button {
                        onAction?("hot_swap")
                    } label: {
                        HStack {
                            Image(systemName: "arrow.triangle.2.circlepath")
                            Text("TRIGGER MID-SHIFT HOT-SWAP")
                                .font(.system(size: 11, weight: .bold))
                            Spacer()
                            Image(systemName: "chevron.right")
                                .font(.system(size: 10, weight: .bold))
                        }
                        .padding(12)
                        .background(TacticalTheme.surfaceSubtle)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).stroke(TacticalTheme.border, lineWidth: 1))
                    }
                    .buttonStyle(.plain)

                    Button {
                        onAction?("emergency_hold")
                    } label: {
                        HStack {
                            Image(systemName: "hand.raised.fill")
                            Text("EMERGENCY QUALITY HOLD")
                                .font(.system(size: 11, weight: .bold))
                            Spacer()
                        }
                        .padding(12)
                        .foregroundStyle(TacticalTheme.statusDanger)
                        .background(TacticalTheme.statusDanger.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).stroke(TacticalTheme.statusDanger.opacity(0.4), lineWidth: 1))
                    }
                    .buttonStyle(.plain)
                }

                // Live Activity Timeline
                VStack(alignment: .leading, spacing: 12) {
                    Text("LIVE AUDIT TIMELINE")
                        .tacticalMicroLabel()

                    TacticalActivityTimeline(items: sampleTimeline)
                }
                .tacticalCard()
            }
            .padding()
        }
        .background(TacticalTheme.canvas)
    }
}

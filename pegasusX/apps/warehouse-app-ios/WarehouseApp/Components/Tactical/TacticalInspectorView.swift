import SwiftUI

struct TacticalInspectorView: View {
    let title: String
    let subtitle: String
    var status: String = "ACTIVE"
    var temperature: Double? = nil
    var humidity: Int? = nil
    var battery: Int? = nil
    var boltSealCode: String? = nil
    var timelineItems: [TacticalTimelineItem] = []
    var onAction: ((String) -> Void)? = nil

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 20) {
                VStack(alignment: .leading, spacing: 6) {
                    HStack {
                        Text("CONTROL TOWER INSPECTOR")
                            .tacticalMicroLabel()
                        Spacer()
                        TacticalStatusBadge(status: status)
                    }

                    Text(title)
                        .font(.system(.headline, weight: .bold))
                        .foregroundStyle(TacticalTheme.textPrimary)

                    Text(subtitle)
                        .font(.system(.caption, design: .monospaced, weight: .semibold))
                        .foregroundStyle(TacticalTheme.textTertiary)
                }
                .tacticalCard()

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
                                    .font(.system(.headline, design: .monospaced, weight: .bold))
                                    .foregroundStyle(TacticalTheme.accentCyan)
                                Text("THRESHOLD: -22°C to -15°C")
                                    .font(.system(.caption2, weight: .bold))
                                    .foregroundStyle(TacticalTheme.statusSuccess)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .accessibilityElement(children: .combine)
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
                                    .font(.system(.headline, design: .monospaced, weight: .bold))
                                    .foregroundStyle(TacticalTheme.textPrimary)
                                Text("CONDENSATION SAFE")
                                    .font(.system(.caption2, weight: .bold))
                                    .foregroundStyle(TacticalTheme.statusSuccess)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .accessibilityElement(children: .combine)
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
                                    .font(.system(.headline, design: .monospaced, weight: .bold))
                                    .foregroundStyle(TacticalTheme.textPrimary)
                                Text("18.4 HOURS EST.")
                                    .font(.system(.caption2, weight: .bold))
                                    .foregroundStyle(TacticalTheme.accentLime)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .accessibilityElement(children: .combine)
                        }
                    }
                }
                .tacticalCard()

                if let seal = boltSealCode {
                    VStack(alignment: .leading, spacing: 10) {
                        Text("SECURITY & BOLT SEAL")
                            .tacticalMicroLabel()

                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text(seal)
                                    .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                    .foregroundStyle(TacticalTheme.accentOrange)
                                Text("TAMPER-EVIDENT RFID BOLT")
                                    .font(.system(.caption2, weight: .medium))
                                    .foregroundStyle(TacticalTheme.textTertiary)
                            }
                            Spacer()
                            Image(systemName: "checkmark.shield.fill")
                                .font(.title3)
                                .foregroundStyle(TacticalTheme.statusSuccess)
                        }
                    }
                    .tacticalCard()
                    .accessibilityElement(children: .combine)
                }

                VStack(alignment: .leading, spacing: 10) {
                    Text("SUPERVISOR ACTIONS")
                    .tacticalMicroLabel()

                    Button {
                        PegasusHaptics.light()
                        onAction?("print_zpl")
                    } label: {
                        HStack {
                            Image(systemName: "printer.fill")
                            Text("PRINT ZPL THERMAL LABELS")
                                .font(.system(.caption2, weight: .bold))
                            Spacer()
                            Text("DOCK PRINTER 2")
                                .font(.system(.caption2, design: .monospaced, weight: .bold))
                                .foregroundStyle(TacticalTheme.accentCobalt)
                        }
                        .padding(12)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .background(TacticalTheme.surfaceSubtle)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).stroke(TacticalTheme.border, lineWidth: 1))
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Print ZPL thermal labels on Dock Printer 2")

                    Button {
                        PegasusHaptics.medium()
                        onAction?("hot_swap")
                    } label: {
                        HStack {
                            Image(systemName: "arrow.triangle.2.circlepath")
                            Text("TRIGGER MID-SHIFT HOT-SWAP")
                                .font(.system(.caption2, weight: .bold))
                            Spacer()
                            Image(systemName: "chevron.right")
                                .font(.system(.caption2, weight: .bold))
                        }
                        .padding(12)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .background(TacticalTheme.surfaceSubtle)
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).stroke(TacticalTheme.border, lineWidth: 1))
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Trigger mid-shift hot-swap")

                    Button {
                        PegasusHaptics.warning()
                        onAction?("emergency_hold")
                    } label: {
                        HStack {
                            Image(systemName: "hand.raised.fill")
                            Text("EMERGENCY QUALITY HOLD")
                                .font(.system(.caption2, weight: .bold))
                            Spacer()
                        }
                        .padding(12)
                        .frame(minHeight: 44)
                        .contentShape(Rectangle())
                        .foregroundStyle(TacticalTheme.statusDanger)
                        .background(TacticalTheme.statusDanger.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        .overlay(RoundedRectangle(cornerRadius: 8, style: .continuous).stroke(TacticalTheme.statusDanger.opacity(0.4), lineWidth: 1))
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Emergency quality hold")
                }

                if !timelineItems.isEmpty {
                    VStack(alignment: .leading, spacing: 12) {
                        Text("LIVE AUDIT TIMELINE")
                            .tacticalMicroLabel()

                        TacticalActivityTimeline(items: timelineItems)
                    }
                    .tacticalCard()
                }
            }
            .padding()
        }
        .background(TacticalTheme.canvas)
    }
}

//
//  FleetDVIRView.swift
//  driverappios
//

import SwiftUI

struct FleetDVIRView: View {

    @State private var vm = FleetDVIRViewModel()
    let onBack: () -> Void

    var body: some View {
        NavigationStack {
            ZStack(alignment: .bottom) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 20) {
                        // Header info
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Pre-Trip DVIR Walkaround")
                                .font(.system(size: 24, weight: .bold))
                                .foregroundStyle(LabTheme.fg)
                            Text("8-Point Mandatory Safety Inspection before Route Dispatch")
                                .font(.system(size: 14))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }
                        .padding(.top, 12)

                        // Red Tag Quarantine Gate Banner
                        if vm.isRedTagQuarantined {
                            VStack(alignment: .leading, spacing: 12) {
                                HStack(spacing: 12) {
                                    Image(systemName: "exclamationmark.octagon.fill")
                                        .font(.system(size: 28))
                                        .foregroundStyle(.white)
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text("RED-TAG GROUNDING ENFORCED")
                                            .font(.system(size: 16, weight: .bold))
                                            .foregroundStyle(.white)
                                        Text("Vehicle Quarantine Active (HTTP 409 Gate)")
                                            .font(.system(size: 12))
                                            .foregroundStyle(.white.opacity(0.85))
                                    }
                                }

                                Divider().background(.white.opacity(0.4))

                                if let reason = vm.redTagReason {
                                    Text(reason)
                                        .font(.system(size: 13, weight: .medium))
                                        .foregroundStyle(.white)
                                }

                                if let wo = vm.workOrderId {
                                    HStack {
                                        Text("Maintenance WO:")
                                            .font(.system(size: 12))
                                            .foregroundStyle(.white.opacity(0.8))
                                        Text(wo)
                                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                                            .foregroundStyle(.white)
                                    }
                                }

                                Text("Driver route pairing uncoupled. Vehicle dispatch is physically locked.")
                                    .font(.system(size: 11, weight: .semibold))
                                    .foregroundStyle(.white.opacity(0.9))
                            }
                            .padding(16)
                            .background(Color.red)
                            .clipShape(RoundedRectangle(cornerRadius: 16))
                        }

                        // Vehicle Details Card
                        VStack(alignment: .leading, spacing: 12) {
                            Text("Vehicle Metadata")
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(LabTheme.fgSecondary)

                            HStack {
                                Text("Vehicle ID:")
                                    .font(.system(size: 14))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Spacer()
                                Text(vm.vehicleId)
                                    .font(.system(size: 14, weight: .bold, design: .monospaced))
                            }

                            HStack {
                                Text("Odometer:")
                                    .font(.system(size: 14))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Spacer()
                                Text("\(vm.odometerKm) km")
                                    .font(.system(size: 14, weight: .bold))
                            }

                            HStack {
                                Text("Cold Chain Required:")
                                    .font(.system(size: 14))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Spacer()
                                Text(vm.requiresColdChain ? "YES" : "NO")
                                    .font(.system(size: 13, weight: .bold))
                                    .foregroundStyle(vm.requiresColdChain ? Color.blue : LabTheme.fgSecondary)
                            }

                            if vm.requiresColdChain {
                                HStack {
                                    Text("Reefer Temperature:")
                                        .font(.system(size: 14))
                                        .foregroundStyle(LabTheme.fgSecondary)
                                    Spacer()
                                    Text(String(format: "%.1f °C", vm.reeferTempCelsius))
                                        .font(.system(size: 14, weight: .bold, design: .monospaced))
                                        .foregroundStyle(vm.reeferTempCelsius <= 4.0 ? Color.green : Color.orange)
                                }
                            }
                        }
                        .padding(16)
                        .background(LabTheme.card)
                        .clipShape(RoundedRectangle(cornerRadius: 16))

                        // 8-Point Checklist Cards
                        VStack(alignment: .leading, spacing: 10) {
                            Text("8-Point Walkaround Checklist")
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(LabTheme.fgSecondary)

                            ChecklistRow(
                                number: 1,
                                title: "Steering & Horn",
                                description: "Clear motion, no excessive play, audible horn warning",
                                isOk: vm.checklist.steeringAndHornOk,
                                onToggle: vm.toggleSteeringAndHorn
                            )

                            ChecklistRow(
                                number: 2,
                                title: "Service & Parking Brakes",
                                description: "Pressure hold, air tank drainage, firm pedal resistance",
                                isOk: vm.checklist.serviceAndParkingBrakesOk,
                                onToggle: vm.toggleBrakes
                            )

                            ChecklistRow(
                                number: 3,
                                title: "Tires Tread & Pressure",
                                description: "Depth ≥3mm, sidewall integrity, lug nut torque verified",
                                isOk: vm.checklist.tiresTreadAndPressureOk,
                                onToggle: vm.toggleTires
                            )

                            ChecklistRow(
                                number: 4,
                                title: "Lights & Hazards",
                                description: "High/low beams, indicators, brake lamps, reverse beacon",
                                isOk: vm.checklist.lightsAndHazardsOk,
                                onToggle: vm.toggleLights
                            )

                            ChecklistRow(
                                number: 5,
                                title: "Wipers & Fluids",
                                description: "Oil, coolant, washer fluid filled, blades unperished",
                                isOk: vm.checklist.wipersAndFluidsOk,
                                onToggle: vm.toggleWipers
                            )

                            ChecklistRow(
                                number: 6,
                                title: "Emergency Kit & Fire Extinguisher",
                                description: "First aid pack intact, gauge in green zone, warning triangle",
                                isOk: vm.checklist.emergencyKitAndExtinguisherOk,
                                onToggle: vm.toggleEmergencyKit
                            )

                            ChecklistRow(
                                number: 7,
                                title: "Cargo Box Sanitation",
                                description: "Floor sweeps clean, zero foreign odor, pest free",
                                isOk: vm.checklist.cargoBoxSanitationOk,
                                onToggle: vm.toggleCargoSanitation
                            )

                            if vm.requiresColdChain {
                                ChecklistRow(
                                    number: 8,
                                    title: "Reefer Temperature Calibrated",
                                    description: "Telemetry sensor matches thermometer (±0.5°C tolerance)",
                                    isOk: vm.checklist.reeferTempCalibratedOk,
                                    onToggle: vm.toggleReeferCalibration
                                )
                            }
                        }

                        // Success / Error messages
                        if let success = vm.successMessage {
                            Text(success)
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(Color.green)
                                .padding(12)
                        }

                        if let err = vm.errorMessage {
                            Text(err)
                                .font(.system(size: 14, weight: .semibold))
                                .foregroundStyle(Color.red)
                                .padding(12)
                        }

                        Spacer(minLength: 80)
                    }
                    .padding(.horizontal, 20)
                }

                // Bottom Action Button
                VStack {
                    Button {
                        Haptics.heavy()
                        Task {
                            await vm.submitInspection()
                            if vm.isRedTagQuarantined {
                                Haptics.error()
                            } else if vm.successMessage != nil {
                                Haptics.success()
                            }
                        }
                    } label: {
                        HStack {
                            if vm.isSubmitting {
                                ProgressView()
                                    .tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text(vm.hasCriticalDefect ? "Report Defect & Red-Tag Ground" : "Certify Walkaround & Dispatch")
                                .font(.system(size: 16, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                        .background(vm.hasCriticalDefect ? Color.red : Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                        .shadow(color: (vm.hasCriticalDefect ? Color.red : Color.blue).opacity(0.3), radius: 8, x: 0, y: 4)
                    }
                    .disabled(vm.isSubmitting)
                }
                .padding(.horizontal, 20)
                .padding(.bottom, 16)
                .background(.ultraThinMaterial)
            }
            .animation(Anim.snappy, value: vm.hasCriticalDefect)
            .animation(Anim.snappy, value: vm.isRedTagQuarantined)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(action: onBack) {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 16, weight: .semibold))
                    }
                }
            }
        }
    }
}

private struct ChecklistRow: View {
    let number: Int
    let title: String
    let description: String
    let isOk: Bool
    let onToggle: () -> Void

    var body: some View {
        Button(action: {
            Haptics.light()
            onToggle()
        }) {
            HStack(alignment: .top, spacing: 14) {
                Image(systemName: isOk ? "checkmark.circle.fill" : "xmark.octagon.fill")
                    .font(.system(size: 24))
                    .foregroundStyle(isOk ? Color.green : Color.red)
                    .contentTransition(.symbolEffect(.replace))

                VStack(alignment: .leading, spacing: 4) {
                    Text("\(number). \(title)")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                    Text(description)
                        .font(.system(size: 12))
                        .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                Text(isOk ? "PASS" : "DEFECT")
                    .font(.system(size: 11, weight: .heavy, design: .monospaced))
                    .foregroundStyle(isOk ? Color.green : Color.red)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background((isOk ? Color.green : Color.red).opacity(0.12))
                    .clipShape(Capsule())
            }
            .padding(14)
            .background(LabTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: 14))
            .overlay(
                RoundedRectangle(cornerRadius: 14)
                    .stroke((isOk ? Color.clear : Color.red.opacity(0.4)), lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
        .contentShape(Rectangle())
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(number). \(title)")
        .accessibilityValue(isOk ? "Status: Passed" : "Status: Defect reported")
        .accessibilityHint("Double tap to toggle defect status")
    }
}

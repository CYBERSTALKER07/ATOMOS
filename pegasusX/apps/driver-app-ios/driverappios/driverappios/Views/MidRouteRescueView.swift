//
//  MidRouteRescueView.swift
//  driverappios
//

import SwiftUI

struct MidRouteRescueView: View {

    @State private var vm = MidRouteRescueViewModel()
    let onBack: () -> Void

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Mid-Route Rescue Handoff")
                            .font(.system(.title2, weight: .bold))
                        Text("Breakdown Cargo & Cash Liability Transfer Protocol")
                            .font(.system(.subheadline))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.top, 12)

                    // Liability Boundary Summary Card
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Cash Liability Boundary (Stop K = \(vm.lastCompletedStopSeq))")
                            .font(.system(.subheadline, weight: .bold))

                        HStack {
                            VStack(alignment: .leading) {
                                Text("Original Driver Liability")
                                    .font(.system(.caption))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Text("\(vm.originalDriverLiability / 100) UZS")
                                    .font(.system(.callout, weight: .bold))
                                    .foregroundStyle(Color.blue)
                            }
                            Spacer()
                            VStack(alignment: .trailing) {
                                Text("Rescue Driver Liability")
                                    .font(.system(.caption))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Text("\(vm.rescueDriverLiability / 100) UZS")
                                    .font(.system(.callout, weight: .bold))
                                    .foregroundStyle(Color.green)
                            }
                        }

                        if !vm.spoiledTotes.isEmpty {
                            HStack {
                                Image(systemName: "thermometer.snowflake")
                                    .foregroundStyle(Color.red)
                                Text("Thermal Excursion Quarantined:")
                                    .font(.system(.caption, weight: .semibold))
                                    .foregroundStyle(Color.red)
                                Spacer()
                                Text("\(vm.spoiledTotes.count) totes spoiled")
                                    .font(.system(.caption, weight: .bold))
                                    .foregroundStyle(Color.red)
                            }
                            .padding(10)
                            .background(Color.red.opacity(0.1))
                            .clipShape(RoundedRectangle(cornerRadius: 8))
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .accessibilityElement(children: .combine)

                    // Vehicle & Seal Inputs
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Handoff Telemetry")
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        HStack {
                            Text("Broken Vehicle:")
                                .font(.system(.subheadline))
                            Spacer()
                            Text(vm.brokenVehicleId)
                                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        }

                        HStack {
                            Text("Rescue Vehicle:")
                                .font(.system(.subheadline))
                            Spacer()
                            Text(vm.rescueVehicleId)
                                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        }

                        VStack(alignment: .leading, spacing: 4) {
                            Text("New Rescue Bolt Seal Barcode:")
                                .font(.system(.caption))
                                .foregroundStyle(LabTheme.fgSecondary)
                            TextField("Enter or scan seal barcode", text: $vm.newRescueBoltSeal)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(.subheadline, design: .monospaced))
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .accessibilityElement(children: .combine)

                    // Totes List
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Cargo Totes Handoff List")
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        ForEach(vm.totes) { tote in
                            let isCompleted = tote.stopSequence <= vm.lastCompletedStopSeq
                            let isSpoiled = tote.isColdChain && tote.thermalExcursionMinutes > tote.maxAllowedExcursionMins

                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack {
                                        Text(tote.toteId)
                                            .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                        if tote.isColdChain {
                                            Image(systemName: "snowflake")
                                                .font(.system(.caption))
                                                .foregroundStyle(Color.blue)
                                        }
                                    }
                                    Text("Stop #\(tote.stopSequence) · \(tote.expectedCashMinor / 100) UZS")
                                        .font(.system(.caption))
                                        .foregroundStyle(LabTheme.fgSecondary)
                                }

                                Spacer()

                                if isCompleted {
                                    Text("DELIVERED (STOP ≤ K)")
                                        .font(.system(.caption2, weight: .bold))
                                        .foregroundStyle(Color.blue)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 3)
                                        .background(Color.blue.opacity(0.12))
                                        .clipShape(Capsule())
                                } else if isSpoiled {
                                    Text("SPOILED (EXCURSION)")
                                        .font(.system(.caption2, weight: .bold))
                                        .foregroundStyle(Color.red)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 3)
                                        .background(Color.red.opacity(0.12))
                                        .clipShape(Capsule())
                                } else {
                                    Text("TRANSFERRED")
                                        .font(.system(.caption2, weight: .bold))
                                        .foregroundStyle(Color.green)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 3)
                                        .background(Color.green.opacity(0.12))
                                        .clipShape(Capsule())
                                }
                            }
                            .padding(12)
                            .background(LabTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                            .accessibilityElement(children: .combine)
                        }
                    }

                    if let msg = vm.statusMessage {
                        Text(msg)
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(Color.green)
                            .padding(12)
                    }

                    if let err = vm.errorMessage {
                        Text(err)
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(Color.red)
                            .padding(12)
                    }

                    Button {
                        Task {
                            await vm.executeRescue()
                        }
                    } label: {
                        HStack {
                            if vm.isExecuting {
                                ProgressView().tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text("Confirm Cargo & Liability Transfer")
                                .font(.system(.callout, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }
                    .disabled(vm.isExecuting)

                    Spacer(minLength: 40)
                }
                .padding(.horizontal, 20)
            }
            .scrollDismissesKeyboard(.interactively)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(action: onBack) {
                        Image(systemName: "chevron.left")
                            .font(.system(.callout, weight: .semibold))
                            .frame(width: 44, height: 44)
                            .contentShape(Rectangle())
                    }
                    .accessibilityLabel("Back")
                }
            }
        }
    }
}

//
//  YardGateView.swift
//  driverappios
//

import SwiftUI

struct YardGateView: View {

    @State private var vm = YardGateViewModel()
    let onBack: () -> Void

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Yard Gate Check-In/Out")
                            .font(.system(.title2, weight: .bold))
                        Text("ALPR Vehicle & Bolt Seal Security Verification")
                            .font(.system(.subheadline))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.top, 12)

                    // Gate Direction Picker
                    Picker("Direction", selection: $vm.direction) {
                        Text("Gate Out (Departure)").tag("GATE_OUT")
                        Text("Gate In (Arrival)").tag("GATE_IN")
                    }
                    .pickerStyle(.segmented)
                    .onChange(of: vm.direction) { _, _ in
                        vm.updateBaySlot()
                    }

                    // Vehicle & Plate Card
                    VStack(alignment: .leading, spacing: 14) {
                        Text("ALPR Optical Recognition")
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        HStack {
                            Text("License Plate:")
                                .font(.system(.subheadline))
                            Spacer()
                            Text(vm.licensePlate)
                                .font(.system(.callout, design: .monospaced, weight: .heavy))
                                .padding(.horizontal, 10)
                                .padding(.vertical, 4)
                                .background(Color.yellow.opacity(0.2))
                                .clipShape(RoundedRectangle(cornerRadius: 6))
                        }

                        HStack {
                            Text("Vehicle Unit ID:")
                                .font(.system(.subheadline))
                            Spacer()
                            Text(vm.vehicleId)
                                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                        }

                        HStack {
                            Text("Purpose:")
                                .font(.system(.subheadline))
                            Spacer()
                            Text(vm.vehiclePurpose)
                                .font(.system(.footnote, weight: .bold))
                                .foregroundStyle(Color.blue)
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .accessibilityElement(children: .combine)

                    // Bolt Seal Card
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Tamper-Evident Bolt Seal")
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        VStack(alignment: .leading, spacing: 4) {
                            Text("Seal Barcode:")
                                .font(.system(.caption))
                                .foregroundStyle(LabTheme.fgSecondary)
                            TextField("Enter or scan seal barcode", text: $vm.boltSealBarcode)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(.subheadline, design: .monospaced))
                        }

                        Toggle("Physical Bolt Seal Intact", isOn: $vm.sealIntact)
                            .tint(.green)
                            .onChange(of: vm.sealIntact) { _, _ in
                                vm.updateBaySlot()
                            }

                        HStack {
                            Text("Assigned Bay Slot:")
                                .font(.system(.subheadline))
                            Spacer()
                            Text(vm.assignedBaySlot)
                                .font(.system(.subheadline, design: .monospaced, weight: .heavy))
                                .foregroundStyle(vm.sealIntact ? Color.primary : Color.red)
                        }

                        if !vm.sealIntact {
                            Text("CRITICAL: Broken or mismatched seal triggers automatic quarantine bay redirection.")
                                .font(.system(.caption, weight: .semibold))
                                .foregroundStyle(Color.red)
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                    .accessibilityElement(children: .combine)

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
                            await vm.submitGateCheck()
                        }
                    } label: {
                        HStack {
                            if vm.isSubmitting {
                                ProgressView().tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text(vm.sealIntact ? "Authorize Gate Passage" : "Quarantine & Alert Yard Marshall")
                                .font(.system(.callout, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                        .background(vm.sealIntact ? Color.blue : Color.red)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }
                    .disabled(vm.isSubmitting)

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

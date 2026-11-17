//
//  YardGateView.swift
//  WarehouseAppIOS
//
//  Yard Gate Control: Inbound / Outbound ALPR & Tamper-Evident Bolt Seal Verification
//

import SwiftUI

struct YardGateView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = YardGateViewModel()

    var onBack: (() -> Void)? = nil

    init(onBack: (() -> Void)? = nil) {
        self.onBack = onBack
    }

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(alignment: .leading, spacing: PegasusMonochromeTheme.spacingLG) {
                VStack(alignment: .leading, spacing: PegasusMonochromeTheme.spacingXS) {
                    Text("Yard Gate Control")
                        .font(.system(.title2, weight: .bold))
                        .foregroundStyle(PegasusMonochromeTheme.label)
                    Text("Inbound / Outbound ALPR & Tamper-Evident Bolt Seal Verification")
                        .font(.subheadline)
                        .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                }
                .padding(.top, PegasusMonochromeTheme.spacingSM)

                Picker("Direction", selection: $vm.direction) {
                    Text("Inbound Entry").tag("GATE_IN")
                    Text("Outbound Exit").tag("GATE_OUT")
                }
                .pickerStyle(.segmented)
                .onChange(of: vm.direction) { _, _ in
                    PegasusHaptics.selection()
                }

                // Quick ALPR presets chips
                VStack(alignment: .leading, spacing: 6) {
                    Text("ALPR Quick Recognition Presets:")
                        .font(.system(.caption2, design: .monospaced, weight: .bold))
                        .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                    ScrollView(.horizontal, showsIndicators: false) {
                        HStack(spacing: 8) {
                            ForEach(vm.vehiclePresets) { preset in
                                Button {
                                    PegasusHaptics.light()
                                    vm.applyPreset(preset)
                                } label: {
                                    HStack(spacing: 4) {
                                        Image(systemName: "camera.viewfinder")
                                            .font(.caption2)
                                        Text(preset.label)
                                            .font(.system(.caption2, weight: .semibold))
                                    }
                                    .padding(.horizontal, 10)
                                    .padding(.vertical, 6)
                                    .frame(minHeight: 44)
                                    .contentShape(Rectangle())
                                    .background(vm.licensePlate == preset.plate ? PegasusMonochromeTheme.primaryAction : PegasusMonochromeTheme.secondaryBackground)
                                    .foregroundStyle(vm.licensePlate == preset.plate ? PegasusMonochromeTheme.onPrimaryAction : PegasusMonochromeTheme.label)
                                    .clipShape(Capsule())
                                }
                                .buttonStyle(.plain)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("Preset vehicle \(preset.label), plate \(preset.plate)")
                            }
                        }
                    }
                }

                // Vehicle Info
                VStack(alignment: .leading, spacing: PegasusMonochromeTheme.spacingMD) {
                    HStack {
                        Text("Optical Vehicle Recognition (ALPR)")
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(PegasusMonochromeTheme.label)
                        Spacer()
                        Text(vm.direction)
                            .font(.system(.caption2, design: .monospaced, weight: .bold))
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(PegasusMonochromeTheme.tertiaryBackground)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                            .clipShape(RoundedRectangle(cornerRadius: 4, style: .continuous))
                    }

                    HStack {
                        Text("License Plate:")
                            .font(.footnote)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        Spacer()
                        TextField("License plate", text: $vm.licensePlate)
                            .textFieldStyle(.plain)
                            .multilineTextAlignment(.trailing)
                            .font(.system(.subheadline, design: .monospaced, weight: .heavy))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(PegasusMonochromeTheme.tertiaryBackground)
                            .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                            .frame(maxWidth: 160)
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.characters)
                    }

                    HStack {
                        Text("Vehicle Unit ID:")
                            .font(.footnote)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        Spacer()
                        TextField("Vehicle ID", text: $vm.vehicleId)
                            .textFieldStyle(.plain)
                            .multilineTextAlignment(.trailing)
                            .font(.system(.subheadline, design: .monospaced, weight: .bold))
                            .foregroundStyle(PegasusMonochromeTheme.label)
                            .frame(maxWidth: 160)
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.characters)
                    }

                    HStack {
                        Text("Purpose:")
                            .font(.footnote)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        Spacer()
                        Text(vm.vehiclePurpose)
                            .font(.system(.footnote, design: .monospaced, weight: .semibold))
                            .foregroundStyle(PegasusMonochromeTheme.label)
                    }
                    .accessibilityElement(children: .combine)
                }
                .padding(PegasusMonochromeTheme.spacingMD)
                .background(PegasusMonochromeTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .stroke(PegasusMonochromeTheme.cardBorder, lineWidth: 1)
                )

                // Seal Verification
                VStack(alignment: .leading, spacing: PegasusMonochromeTheme.spacingMD) {
                    HStack {
                        Text("Bolt Seal Verification")
                            .font(.system(.subheadline, weight: .semibold))
                            .foregroundStyle(PegasusMonochromeTheme.label)
                        Spacer()
                        Menu {
                            ForEach(vm.sealPresets, id: \.self) { preset in
                                Button(preset) {
                                    vm.boltSealBarcode = preset
                                    PegasusHaptics.light()
                                }
                            }
                        } label: {
                            Text("Presets")
                                .font(.system(.caption2, weight: .bold))
                                .foregroundStyle(PegasusMonochromeTheme.accentBronze)
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                    }

                    VStack(alignment: .leading, spacing: PegasusMonochromeTheme.spacingXS) {
                        Text("Bolt Seal Barcode:")
                            .font(.caption)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        TextField("Seal barcode", text: $vm.boltSealBarcode)
                            .textFieldStyle(.plain)
                            .font(.system(.footnote, design: .monospaced))
                            .padding(8)
                            .background(PegasusMonochromeTheme.secondaryBackground)
                            .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                            .overlay(
                                RoundedRectangle(cornerRadius: 6, style: .continuous)
                                    .stroke(PegasusMonochromeTheme.separator, lineWidth: 1)
                            )
                            .autocorrectionDisabled()
                            .textInputAutocapitalization(.characters)
                    }

                    Toggle("Bolt Seal Verified Intact", isOn: $vm.sealIntact)
                        .tint(PegasusMonochromeTheme.success)
                        .frame(minHeight: 44)
                        .onChange(of: vm.sealIntact) { _, intact in
                            PegasusHaptics.light()
                            vm.updateSealIntegrity(intact)
                        }

                    HStack {
                        Text("Assigned Bay Slot:")
                            .font(.footnote)
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        Spacer()
                        Text(vm.assignedBaySlot)
                            .font(.system(.subheadline, design: .monospaced, weight: .heavy))
                            .foregroundStyle(vm.sealIntact ? PegasusMonochromeTheme.label : PegasusMonochromeTheme.destructive)
                    }
                    .accessibilityElement(children: .combine)

                    if !vm.sealIntact {
                        HStack(spacing: 6) {
                            Image(systemName: "exclamationmark.octagon.fill")
                                .font(.footnote)
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                            Text(vm.direction == "GATE_OUT" ? "CRITICAL: Outbound exit hard-blocked! Tampered bolt seal." : "Tampered seal detected: Vehicle routed to security quarantine.")
                                .font(.system(.caption, weight: .semibold))
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                        }
                        .accessibilityElement(children: .combine)
                    }
                }
                .padding(PegasusMonochromeTheme.spacingMD)
                .background(PegasusMonochromeTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                .overlay(
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .stroke(PegasusMonochromeTheme.cardBorder, lineWidth: 1)
                )

                if let msg = vm.statusMessage {
                    HStack(spacing: 6) {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(PegasusMonochromeTheme.success)
                        Text(msg)
                            .font(.system(.footnote, weight: .semibold))
                            .foregroundStyle(PegasusMonochromeTheme.success)
                    }
                    .padding(.horizontal, PegasusMonochromeTheme.spacingXS)
                    .accessibilityElement(children: .combine)
                }

                if let err = vm.errorMessage {
                    HStack(spacing: 6) {
                        Image(systemName: "xmark.circle.fill")
                            .foregroundStyle(PegasusMonochromeTheme.destructive)
                        Text(err)
                            .font(.system(.footnote, weight: .semibold))
                            .foregroundStyle(PegasusMonochromeTheme.destructive)
                    }
                    .padding(.horizontal, PegasusMonochromeTheme.spacingXS)
                    .accessibilityElement(children: .combine)
                }

                Button {
                    PegasusHaptics.heavy()
                    Task { await vm.submitGateEvent() }
                } label: {
                    HStack {
                        if vm.isSubmitting {
                            ProgressView().tint(.white)
                            Spacer().frame(width: 8)
                        }
                        if vm.isOutboundGateBlocked {
                            Image(systemName: "hand.raised.fill")
                            Text("Gate Blocked: Broken Seal")
                                .font(.system(.subheadline, weight: .bold))
                        } else {
                            Text(vm.sealIntact ? "Confirm Gate Clearance" : "Quarantine Vehicle Entry")
                                .font(.system(.subheadline, weight: .bold))
                        }
                    }
                    .frame(maxWidth: .infinity, minHeight: 50)
                    .contentShape(Rectangle())
                    .background(vm.isOutboundGateBlocked ? PegasusMonochromeTheme.destructive.opacity(0.5) : (vm.sealIntact ? PegasusMonochromeTheme.primaryAction : PegasusMonochromeTheme.destructive))
                    .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(vm.isSubmitting || vm.isOutboundGateBlocked)

                Spacer(minLength: 40)
            }
            .padding(.horizontal, PegasusMonochromeTheme.spacingLG)
        }
        .scrollDismissesKeyboard(.interactively)
        .background(PegasusMonochromeTheme.background)
        .navigationTitle("Yard Gate")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button {
                    PegasusHaptics.light()
                    if let onBack = onBack {
                        onBack()
                    } else {
                        dismiss()
                    }
                } label: {
                    Image(systemName: "chevron.left")
                        .font(.system(.callout, weight: .semibold))
                        .foregroundStyle(PegasusMonochromeTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Back")
            }
        }
    }
}

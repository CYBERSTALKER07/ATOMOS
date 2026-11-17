//
//  CashierBlindCountView.swift
//  WarehouseAppIOS
//
//

import SwiftUI

struct CashierBlindCountView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var vm = CashierViewModel()
    var onBack: (() -> Void)? = nil

    init(onBack: (() -> Void)? = nil) {
        self.onBack = onBack
    }

    private let banknoteDenominations: [Int64] = CashierViewModel.supportedDenominations

    private static func parseWholeSomToMinor(_ raw: String) -> Int64 {
        let cleaned = raw
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: " ", with: "")
            .replacingOccurrences(of: ",", with: "")
        guard let som = Int64(cleaned), som > 0 else { return 0 }
        let (minor, overflow) = som.multipliedReportingOverflow(by: 100)
        return overflow ? 0 : minor
    }

    var body: some View {
        @Bindable var vm = vm
        ZStack(alignment: .bottom) {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Cashier Blind Count & Vault")
                            .font(.system(.title2, weight: .bold))
                        Text("Dual-Blind Cash Reconciliation (so'm)")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.top, 10)

                    if let err = vm.errorMessage {
                        HStack {
                            Image(systemName: "exclamationmark.triangle.fill")
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                            Text(err)
                                .font(.system(.footnote, weight: .medium))
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                        }
                        .padding(12)
                        .background(PegasusMonochromeTheme.destructive.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        .accessibilityElement(children: .combine)
                    }

                    if vm.isRecountRequired {
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Image(systemName: "lock.rotation")
                                    .font(.system(.title3, weight: .bold))
                                    .foregroundStyle(PegasusMonochromeTheme.warning)
                                Text("BLIND RECOUNT #2 REQUIRED")
                                    .font(.system(.subheadline, weight: .heavy))
                                    .foregroundStyle(PegasusMonochromeTheme.warning)
                            }
                            Text("Physical count does not match route ledger. Expected cash remains strictly masked.")
                                .font(.caption)
                                .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        }
                        .padding(14)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(PegasusMonochromeTheme.warning.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        .accessibilityElement(children: .combine)
                    }

                    if vm.isBalanced, let hash = vm.clearanceTokenHash {
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Image(systemName: "checkmark.seal.fill")
                                    .font(.title3)
                                    .foregroundStyle(PegasusMonochromeTheme.success)
                                Text("COUNT BALANCED · CLEARANCE ISSUED")
                                    .font(.system(.subheadline, weight: .bold))
                                    .foregroundStyle(PegasusMonochromeTheme.success)
                            }
                            Text("Token Hash: \(hash)")
                                .font(.system(.caption2, design: .monospaced))
                                .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                        }
                        .padding(12)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(PegasusMonochromeTheme.success.opacity(0.12))
                        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                        .accessibilityElement(children: .combine)
                    } else if let status = vm.statusMessage, !vm.isRecountRequired {
                        Text(status)
                            .font(.system(.caption, weight: .medium))
                            .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                            .padding(12)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color(.secondarySystemGroupedBackground))
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    }

                    // Session Setup Card
                    VStack(alignment: .leading, spacing: 10) {
                        HStack {
                            Text("Route Deposit Bag Verification")
                                .font(.system(.subheadline, weight: .semibold))
                            Spacer()
                            Button {
                                vm.manualEntryMode.toggle()
                            } label: {
                                Text(vm.manualEntryMode ? "Use Pickers" : "Manual IDs")
                                    .font(.system(.caption, weight: .semibold))
                                    .foregroundStyle(PegasusMonochromeTheme.accentBronze)
                            }
                            .pegasusMinimumHitTarget(44)
                        }

                        HStack {
                            Text("Depot ID:")
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                                .frame(width: 84, alignment: .leading)
                            TextField("Warehouse Depot ID", text: $vm.depotId)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(.footnote, design: .monospaced))
                                .autocorrectionDisabled()
                                .textInputAutocapitalization(.never)
                        }

                        HStack {
                            Text("Supplier ID:")
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                                .frame(width: 84, alignment: .leading)
                            TextField("Supplier Scope ID", text: $vm.supplierId)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(.footnote, design: .monospaced))
                                .autocorrectionDisabled()
                                .textInputAutocapitalization(.never)
                        }

                        if !vm.manualEntryMode && !vm.driverOptions.isEmpty {
                            HStack {
                                Text("Driver:")
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 84, alignment: .leading)
                                Picker(
                                    "Driver",
                                    selection: Binding(
                                        get: { vm.driverId },
                                        set: { vm.selectDriver($0) }
                                    )
                                ) {
                                    Text("Select driver...").tag("")
                                    ForEach(vm.driverOptions) { option in
                                        Text(option.label).tag(option.driverId)
                                    }
                                }
                                .pickerStyle(.menu)
                                .frame(maxWidth: .infinity, alignment: .leading)
                            }
                        } else {
                            HStack {
                                Text("Driver ID:")
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 84, alignment: .leading)
                                TextField("Driver ID", text: $vm.driverId)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.footnote)
                                    .autocorrectionDisabled()
                                    .textInputAutocapitalization(.never)
                            }
                        }

                        if !vm.manualEntryMode && !vm.routeOptions.isEmpty {
                            HStack {
                                Text("Route:")
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 84, alignment: .leading)
                                Picker(
                                    "Route",
                                    selection: Binding(
                                        get: { vm.routeId },
                                        set: { vm.selectRoute($0) }
                                    )
                                ) {
                                    Text("Select route / manifest...").tag("")
                                    ForEach(vm.routeOptions) { option in
                                        Text(option.label).tag(option.routeId)
                                    }
                                }
                                .pickerStyle(.menu)
                                .frame(maxWidth: .infinity, alignment: .leading)
                            }
                        } else {
                            HStack {
                                Text("Route ID:")
                                    .font(.footnote)
                                    .foregroundStyle(.secondary)
                                    .frame(width: 84, alignment: .leading)
                                TextField("Route or Manifest ID", text: $vm.routeId)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.footnote)
                                    .autocorrectionDisabled()
                                    .textInputAutocapitalization(.never)
                            }
                        }

                        HStack {
                            Text("Bag Seal:")
                                .font(.footnote)
                                .foregroundStyle(.secondary)
                                .frame(width: 84, alignment: .leading)
                            TextField("Bag Seal Barcode", text: $vm.bagSealBarcode)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(.footnote, design: .monospaced))
                                .autocorrectionDisabled()
                                .textInputAutocapitalization(.characters)
                        }

                        if vm.isLoadingSelectors {
                            HStack(spacing: 8) {
                                ProgressView()
                                    .scaleEffect(0.8)
                                Text("Loading drivers and route manifests...")
                                    .font(.caption2)
                                    .foregroundStyle(.secondary)
                            }
                        }

                        if vm.session == nil {
                            Button {
                                PegasusHaptics.medium()
                                Task { await vm.startSession() }
                            } label: {
                                Text("Start Cashier Session")
                                    .font(.system(.subheadline, weight: .bold))
                                    .frame(maxWidth: .infinity, minHeight: 44)
                                    .contentShape(Rectangle())
                                    .background(PegasusMonochromeTheme.primaryAction)
                                    .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                                    .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(14)
                    .background(Color(.secondarySystemGroupedBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))

                    // Blind Masking Verification Card
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Reconciliation State (Attempt #\(vm.attemptNumber))")
                            .font(.system(.subheadline, weight: .semibold))

                        HStack {
                            Text("Physical Counted Cash:")
                                .font(.footnote)
                            Spacer()
                            Text(CashierViewModel.formatWholeSom(minor: vm.countedCashMinor))
                                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                .foregroundStyle(Color.primary)
                        }
                        .accessibilityElement(children: .combine)

                        HStack {
                            Text("Expected Route Cash:")
                                .font(.footnote)
                            Spacer()
                            if vm.isBlindCountMasked {
                                HStack(spacing: 4) {
                                    Image(systemName: "lock.fill")
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Text("•••••••• so'm [MASKED]")
                                        .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                        .foregroundStyle(.secondary)
                                }
                            } else {
                                Text(CashierViewModel.formatWholeSom(minor: vm.session?.expectedCashMinor ?? 0))
                                    .font(.system(.subheadline, design: .monospaced, weight: .bold))
                            }
                        }
                        .accessibilityElement(children: .combine)

                        if vm.isBlindCountMasked {
                            Text("Dual-Blind Rule: Expected amount strictly masked during counts #1 and #2 to prevent cashier confirmation bias.")
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                        }
                    }
                    .padding(14)
                    .background(Color(.secondarySystemGroupedBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))

                    // Banknote Denomination Grid with Quick Strap Bundles
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            VStack(alignment: .leading, spacing: 2) {
                                Text("Banknote Physical Counter (so'm)")
                                    .font(.system(.subheadline, weight: .semibold))
                                    .foregroundStyle(PegasusMonochromeTheme.label)
                                Text("Total Notes: \(vm.totalBanknotesCount)")
                                    .font(.system(.caption, design: .monospaced, weight: .medium))
                                    .foregroundStyle(PegasusMonochromeTheme.secondaryLabel)
                            }
                            Spacer()

                            Button {
                                PegasusHaptics.medium()
                                vm.clearCounts()
                            } label: {
                                Text("Reset")
                                    .font(.system(.caption, weight: .bold))
                                    .foregroundStyle(PegasusMonochromeTheme.destructive)
                            }
                            .padding(.horizontal, 6)
                            .pegasusMinimumHitTarget(44)
                        }

                        ForEach(banknoteDenominations, id: \.self) { denom in
                            let count = vm.denominations[denom] ?? 0
                            let subtotal = denom * count
                            VStack(alignment: .leading, spacing: 6) {
                                HStack {
                                    Text(CashierViewModel.formatSom(amount: denom))
                                        .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                        .foregroundStyle(PegasusMonochromeTheme.label)
                                    Spacer()
                                    Text(CashierViewModel.formatSom(amount: subtotal))
                                        .font(.system(.footnote, design: .monospaced, weight: .bold))
                                        .monospacedDigit()
                                        .foregroundStyle(count > 0 ? PegasusMonochromeTheme.label : PegasusMonochromeTheme.secondaryLabel)
                                }

                                HStack(spacing: 8) {
                                    HStack(spacing: 4) {
                                        ForEach([10, 50, 100], id: \.self) { bundle in
                                            Button {
                                                PegasusHaptics.light()
                                                vm.addStrap(noteSom: denom, strapCount: Int64(bundle))
                                            } label: {
                                                Text("+\(bundle)")
                                                    .font(.system(.caption2, design: .monospaced, weight: .bold))
                                                    .padding(.horizontal, 8)
                                                    .padding(.vertical, 6)
                                                    .frame(minWidth: 44, minHeight: 44)
                                                    .contentShape(Rectangle())
                                                    .background(PegasusMonochromeTheme.tertiaryBackground)
                                                    .foregroundStyle(PegasusMonochromeTheme.label)
                                                    .clipShape(RoundedRectangle(cornerRadius: 6, style: .continuous))
                                            }
                                            .buttonStyle(.plain)
                                            .accessibilityLabel("Add strap of \(bundle) notes (\(CashierViewModel.formatSom(amount: denom * Int64(bundle))))")
                                        }
                                    }

                                    Spacer()

                                    HStack(spacing: 6) {
                                        Button {
                                            PegasusHaptics.light()
                                            vm.adjustCount(noteSom: denom, delta: -1)
                                        } label: {
                                            Image(systemName: "minus.circle.fill")
                                                .font(.title2)
                                                .foregroundStyle(count > 0 ? PegasusMonochromeTheme.secondaryLabel : PegasusMonochromeTheme.tertiaryLabel)
                                                .frame(minWidth: 44, minHeight: 44)
                                                .contentShape(Rectangle())
                                        }
                                        .disabled(count <= 0)
                                        .accessibilityLabel("Decrease \(denom) so'm note count")

                                        Text("\(count)")
                                            .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                            .monospacedDigit()
                                            .frame(minWidth: 36, alignment: .center)
                                            .accessibilityLabel("\(count) notes of \(denom) so'm")

                                        Button {
                                            PegasusHaptics.light()
                                            vm.adjustCount(noteSom: denom, delta: 1)
                                        } label: {
                                            Image(systemName: "plus.circle.fill")
                                                .font(.title2)
                                                .foregroundStyle(PegasusMonochromeTheme.label)
                                                .frame(minWidth: 44, minHeight: 44)
                                                .contentShape(Rectangle())
                                        }
                                        .accessibilityLabel("Increase \(denom) so'm note count")
                                    }
                                }
                            }
                            .padding(.vertical, 4)
                            .accessibilityElement(children: .contain)
                            Divider()
                        }
                    }
                    .padding(14)
                    .background(Color(.secondarySystemGroupedBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))

                    // Ops Action Buttons
                    HStack(spacing: 10) {
                        Button {
                            PegasusHaptics.selection()
                            vm.showRRNSheet = true
                        } label: {
                            Text("Reclassify RRN")
                                .font(.system(.caption, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .contentShape(Rectangle())
                                .background(Color(.tertiarySystemGroupedBackground))
                                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        }

                        Button {
                            PegasusHaptics.selection()
                            vm.showCITSheet = true
                        } label: {
                            Text("CIT Handover")
                                .font(.system(.caption, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .contentShape(Rectangle())
                                .background(Color(.tertiarySystemGroupedBackground))
                                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        }

                        Button {
                            PegasusHaptics.selection()
                            vm.showAMLSheet = true
                        } label: {
                            Text("AML Exposure")
                                .font(.system(.caption, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: 44)
                                .contentShape(Rectangle())
                                .background(Color(.tertiarySystemGroupedBackground))
                                .clipShape(RoundedRectangle(cornerRadius: 8, style: .continuous))
                        }
                    }

                    Spacer(minLength: 80)
                }
                .padding(.horizontal, 16)
            }
            .scrollDismissesKeyboard(.interactively)

            // Submit Button
            VStack {
                Button {
                    PegasusHaptics.heavy()
                    Task {
                        await vm.submitBlindCount()
                        if vm.isBalanced {
                            PegasusHaptics.success()
                        } else if vm.isRecountRequired || vm.errorMessage != nil {
                            PegasusHaptics.error()
                        }
                    }
                } label: {
                    HStack {
                        if vm.isLoading {
                            ProgressView().tint(.white)
                            Spacer().frame(width: 8)
                        }
                        Text(vm.isRecountRequired ? "Submit Blind Recount #2" : "Submit Blind Count #1")
                            .font(.system(.callout, weight: .bold))
                    }
                    .frame(maxWidth: .infinity, minHeight: 50)
                    .contentShape(Rectangle())
                    .background(vm.isRecountRequired ? PegasusMonochromeTheme.warning : PegasusMonochromeTheme.primaryAction)
                    .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                    .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                }
                .buttonStyle(.plain)
                .disabled(vm.isLoading || vm.countedCashMinor == 0)
            }
            .padding(.horizontal, 16)
            .padding(.bottom, 16)
            .background(.ultraThinMaterial)
        }
        .scrollDismissesKeyboard(.interactively)
        .animation(.snappy, value: vm.isRecountRequired)
        .animation(.snappy, value: vm.isBalanced)
        .navigationTitle("Cashier Vault")
        .navigationBarTitleDisplayMode(.inline)
        .task {
            await vm.loadSelectors()
        }
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
        .sheet(isPresented: $vm.showRRNSheet) {
            NavigationStack {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Reclassify card or online gateway transaction to cash ledger (so'm).")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        TextField("Order ID", text: $vm.rrnOrderId)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                        TextField("POS Terminal Gateway RRN", text: $vm.rrnGateway)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                        TextField("Disputed Cash Amount (so'm)", text: $vm.rrnDisputedText)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        TextField("Terminal ID (optional)", text: $vm.rrnTerminalId)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                        TextField("Card Last 4 (optional)", text: $vm.rrnCardLastFour)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        TextField("Authorization Code (optional)", text: $vm.rrnAuthCode)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                        if let err = vm.errorMessage {
                            Text(err)
                                .font(.caption)
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                        }
                        Button("Apply Compensating Ledger") {
                            PegasusHaptics.medium()
                            let minor = Self.parseWholeSomToMinor(vm.rrnDisputedText)
                            Task {
                                let ok = await vm.reclassifyTender(
                                    orderId: vm.rrnOrderId,
                                    gatewayRrn: vm.rrnGateway,
                                    disputedCashMinor: minor
                                )
                                if ok {
                                    vm.showRRNSheet = false
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .contentShape(Rectangle())
                        .background(PegasusMonochromeTheme.primaryAction)
                        .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                        Spacer()
                    }
                    .padding(20)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("RRN Reclassification")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { vm.showRRNSheet = false }
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                }
            }
        }
        .sheet(isPresented: $vm.showCITSheet) {
            NavigationStack {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Log physical cash handoff to authorized CIT armored courier (so'm).")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        TextField("Courier Badge ID", text: $vm.citBadgeId)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                        TextField("CIT Security Bag Seal Barcode", text: $vm.citSealBarcode)
                            .textFieldStyle(.roundedBorder)
                            .autocorrectionDisabled()
                        TextField("Cash Handover (so'm)", text: $vm.citHandoverText)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        if let err = vm.errorMessage {
                            Text(err)
                                .font(.caption)
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                        }
                        Button("Confirm Armored Handover") {
                            PegasusHaptics.medium()
                            let minor = Self.parseWholeSomToMinor(vm.citHandoverText)
                            Task {
                                let ok = await vm.executeCITHandover(
                                    courierBadgeId: vm.citBadgeId,
                                    citSealBarcode: vm.citSealBarcode,
                                    amountMinor: minor
                                )
                                if ok {
                                    vm.showCITSheet = false
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .contentShape(Rectangle())
                        .background(PegasusMonochromeTheme.primaryAction)
                        .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                        Spacer()
                    }
                    .padding(20)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("Armored Handover")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { vm.showCITSheet = false }
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                }
            }
        }
        .sheet(isPresented: $vm.showAMLSheet) {
            NavigationStack {
                ScrollView {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Audit legal entity multi-tender cash exposure (so'm).")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        TextField("Legal Entity TIN", text: $vm.amlTin)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        TextField("Amount (so'm)", text: $vm.amlExposureText)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        if let err = vm.errorMessage {
                            Text(err)
                                .font(.caption)
                                .foregroundStyle(PegasusMonochromeTheme.destructive)
                        }
                        Button("Audit Cash Exposure") {
                            PegasusHaptics.medium()
                            let minor = Self.parseWholeSomToMinor(vm.amlExposureText)
                            Task {
                                let ok = await vm.checkAMLExposure(tin: vm.amlTin, cashMinor: minor)
                                if ok {
                                    vm.showAMLSheet = false
                                }
                            }
                        }
                        .frame(maxWidth: .infinity, minHeight: 44)
                        .contentShape(Rectangle())
                        .background(PegasusMonochromeTheme.primaryAction)
                        .foregroundStyle(PegasusMonochromeTheme.onPrimaryAction)
                        .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                        Spacer()
                    }
                    .padding(20)
                }
                .scrollDismissesKeyboard(.interactively)
                .navigationTitle("Exposure Audit")
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("Cancel") { vm.showAMLSheet = false }
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                    }
                }
            }
        }
    }
}

//
//  CashierBlindCountView.swift
//  WarehouseAppIOS
//

import SwiftUI

struct CashierBlindCountView: View {

    @State private var vm = CashierViewModel()
    let onBack: () -> Void

    @State private var showRRNSheet: Bool = false
    @State private var showCITSheet: Bool = false
    @State private var showAMLSheet: Bool = false

    @State private var rrnOrderId: String = "order-uz-8812"
    @State private var rrnGateway: String = "123456789012"
    @State private var rrnDisputedCash: String = "500000"

    @State private var citBadgeId: String = "INKAS-9921"
    @State private var citSealBarcode: String = "CIT-SEAL-7712"
    @State private var citAmountUzs: String = "1425000"

    @State private var amlTin: String = "123456789"
    @State private var amlAmountUzs: String = "15000000"

    private let banknoteDenominations: [Int64] = [200000, 100000, 50000, 20000, 10000, 5000, 2000, 1000]

    var body: some View {
        NavigationStack {
            ZStack(alignment: .bottom) {
                ScrollView {
                    VStack(alignment: .leading, spacing: 18) {
                        // Title
                        VStack(alignment: .leading, spacing: 4) {
                            Text("Cashier Blind Count & Vault")
                                .font(.system(size: 24, weight: .bold))
                            Text("Dual-Blind Reconciliation with UZS Integer Tiyin Arithmetic")
                                .font(.system(size: 13))
                                .foregroundStyle(.secondary)
                        }
                        .padding(.top, 10)

                        // Status / Alert Banners
                        if let err = vm.errorMessage {
                            HStack {
                                Image(systemName: "exclamationmark.triangle.fill")
                                    .foregroundStyle(.red)
                                Text(err)
                                    .font(.system(size: 13, weight: .medium))
                                    .foregroundStyle(.red)
                            }
                            .padding(12)
                            .background(Color.red.opacity(0.12))
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        }

                        if vm.isRecountRequired {
                            VStack(alignment: .leading, spacing: 8) {
                                HStack {
                                    Image(systemName: "lock.rotation")
                                        .font(.system(size: 20, weight: .bold))
                                        .foregroundStyle(.orange)
                                    Text("BLIND RECOUNT #2 REQUIRED")
                                        .font(.system(size: 15, weight: .heavy))
                                        .foregroundStyle(.orange)
                                }
                                Text("Physical count does not match route ledger. Expected cash remains strictly masked.")
                                    .font(.system(size: 12))
                                    .foregroundStyle(.secondary)
                            }
                            .padding(14)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.orange.opacity(0.12))
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        }

                        if vm.isBalanced, let hash = vm.clearanceTokenHash {
                            VStack(alignment: .leading, spacing: 8) {
                                HStack {
                                    Image(systemName: "checkmark.seal.fill")
                                        .font(.system(size: 20))
                                        .foregroundStyle(.green)
                                    Text("COUNT BALANCED · CLEARANCE ISSUED")
                                        .font(.system(size: 14, weight: .bold))
                                        .foregroundStyle(.green)
                                }
                                Text("Token Hash: \(hash)")
                                    .font(.system(size: 10, design: .monospaced))
                                    .foregroundStyle(.secondary)
                            }
                            .padding(12)
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .background(Color.green.opacity(0.12))
                            .clipShape(RoundedRectangle(cornerRadius: 12))
                        }

                        // Session Setup Card
                        VStack(alignment: .leading, spacing: 10) {
                            Text("Route Deposit Bag Verification")
                                .font(.system(size: 14, weight: .semibold))

                            HStack {
                                Text("Bag Seal:")
                                    .font(.system(size: 13))
                                    .foregroundStyle(.secondary)
                                TextField("Bag Seal Barcode", text: $vm.bagSealBarcode)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.system(size: 13, design: .monospaced))
                            }

                            HStack {
                                Text("Driver ID:")
                                    .font(.system(size: 13))
                                    .foregroundStyle(.secondary)
                                TextField("Driver ID", text: $vm.driverId)
                                    .textFieldStyle(.roundedBorder)
                                    .font(.system(size: 13))
                            }

                            if vm.session == nil {
                                Button {
                                    Task { await vm.startSession() }
                                } label: {
                                    Text("Start Cashier Session")
                                        .font(.system(size: 14, weight: .bold))
                                        .frame(maxWidth: .infinity)
                                        .frame(height: 40)
                                        .background(Color.blue)
                                        .foregroundStyle(.white)
                                        .clipShape(RoundedRectangle(cornerRadius: 8))
                                }
                            }
                        }
                        .padding(14)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 14))

                        // Blind Masking Verification Card
                        VStack(alignment: .leading, spacing: 10) {
                            Text("Reconciliation State (Attempt #\(vm.attemptNumber))")
                                .font(.system(size: 14, weight: .semibold))

                            HStack {
                                Text("Physical Counted Cash:")
                                    .font(.system(size: 13))
                                Spacer()
                                Text("\(vm.countedCashMinor / 100) UZS")
                                    .font(.system(size: 15, weight: .bold, design: .monospaced))
                                    .foregroundStyle(Color.primary)
                            }

                            HStack {
                                Text("Expected Route Cash:")
                                    .font(.system(size: 13))
                                Spacer()
                                if vm.isBlindCountMasked {
                                    HStack(spacing: 4) {
                                        Image(systemName: "lock.fill")
                                            .font(.system(size: 12))
                                            .foregroundStyle(.secondary)
                                        Text("•••••••• UZS [MASKED]")
                                            .font(.system(size: 14, weight: .bold, design: .monospaced))
                                            .foregroundStyle(.secondary)
                                    }
                                } else {
                                    Text("\((vm.session?.expectedCashMinor ?? 0) / 100) UZS")
                                        .font(.system(size: 15, weight: .bold, design: .monospaced))
                                }
                            }

                            if vm.isBlindCountMasked {
                                Text("Dual-Blind Rule: Expected amount strictly masked during counts #1 and #2 to prevent cashier confirmation bias.")
                                    .font(.system(size: 11))
                                    .foregroundStyle(.secondary)
                            }
                        }
                        .padding(14)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 14))

                        // Banknote Denomination Grid
                        VStack(alignment: .leading, spacing: 10) {
                            HStack {
                                Text("UZS Banknote Physical Counter")
                                    .font(.system(size: 14, weight: .semibold))
                                Spacer()
                                Button {
                                    PegasusHaptics.medium()
                                    vm.clearCounts()
                                } label: {
                                    Text("Reset")
                                        .font(.system(size: 12, weight: .bold))
                                        .foregroundStyle(.red)
                                        .frame(minHeight: 32)
                                }
                            }

                            ForEach(banknoteDenominations, id: \.self) { denom in
                                let count = vm.denominations[denom] ?? 0
                                let subtotal = denom * count
                                HStack {
                                    Text("\(denom) UZS")
                                        .font(.system(size: 14, weight: .bold, design: .monospaced))
                                        .frame(width: 100, alignment: .leading)

                                    Spacer()

                                    Button {
                                        PegasusHaptics.light()
                                        vm.adjustCount(noteUzs: denom, delta: -1)
                                    } label: {
                                        Image(systemName: "minus.circle.fill")
                                            .font(.system(size: 24))
                                            .foregroundStyle(.secondary)
                                            .frame(minWidth: 44, minHeight: 44)
                                            .contentShape(Rectangle())
                                    }
                                    .accessibilityLabel("Decrease \(denom) UZS note count")

                                    Text("\(count)")
                                        .font(.system(size: 15, weight: .bold))
                                        .monospacedDigit()
                                        .frame(width: 44, alignment: .center)
                                        .accessibilityLabel("\(count) notes")

                                    Button {
                                        PegasusHaptics.light()
                                        vm.adjustCount(noteUzs: denom, delta: 1)
                                    } label: {
                                        Image(systemName: "plus.circle.fill")
                                            .font(.system(size: 24))
                                            .foregroundStyle(Color.blue)
                                            .frame(minWidth: 44, minHeight: 44)
                                            .contentShape(Rectangle())
                                    }
                                    .accessibilityLabel("Increase \(denom) UZS note count")

                                    Text("\(subtotal) UZS")
                                        .font(.system(size: 13, design: .monospaced))
                                        .monospacedDigit()
                                        .foregroundStyle(.secondary)
                                        .frame(width: 90, alignment: .trailing)
                                }
                                .padding(.vertical, 2)
                                .accessibilityElement(children: .contain)
                                Divider()
                            }
                        }
                        .padding(14)
                        .background(Color(.secondarySystemGroupedBackground))
                        .clipShape(RoundedRectangle(cornerRadius: 14))

                        // Ops Action Buttons
                        HStack(spacing: 10) {
                            Button {
                                showRRNSheet = true
                            } label: {
                                Text("Reclassify RRN")
                                    .font(.system(size: 12, weight: .bold))
                                    .frame(maxWidth: .infinity)
                                    .frame(height: 38)
                                    .background(Color(.tertiarySystemGroupedBackground))
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }

                            Button {
                                showCITSheet = true
                            } label: {
                                Text("CIT Handover")
                                    .font(.system(size: 12, weight: .bold))
                                    .frame(maxWidth: .infinity)
                                    .frame(height: 38)
                                    .background(Color(.tertiarySystemGroupedBackground))
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }

                            Button {
                                showAMLSheet = true
                            } label: {
                                Text("AML Exposure")
                                    .font(.system(size: 12, weight: .bold))
                                    .frame(maxWidth: .infinity)
                                    .frame(height: 38)
                                    .background(Color(.tertiarySystemGroupedBackground))
                                    .clipShape(RoundedRectangle(cornerRadius: 8))
                            }
                        }

                        Spacer(minLength: 80)
                    }
                    .padding(.horizontal, 16)
                }

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
                                .font(.system(size: 16, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 50)
                        .background(vm.isRecountRequired ? Color.orange : Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                        .shadow(color: (vm.isRecountRequired ? Color.orange : Color.blue).opacity(0.3), radius: 8, x: 0, y: 4)
                    }
                    .disabled(vm.isLoading || vm.countedCashMinor == 0)
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 16)
                .background(.ultraThinMaterial)
            }
            .animation(.snappy, value: vm.isRecountRequired)
            .animation(.snappy, value: vm.isBalanced)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button(action: onBack) {
                        Image(systemName: "chevron.left")
                            .font(.system(size: 16, weight: .semibold))
                    }
                }
            }
            .sheet(isPresented: $showRRNSheet) {
                NavigationStack {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Reclassify card or online gateway transaction to cash ledger.")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                        TextField("Order ID", text: $rrnOrderId)
                            .textFieldStyle(.roundedBorder)
                        TextField("POS Terminal Gateway RRN", text: $rrnGateway)
                            .textFieldStyle(.roundedBorder)
                        TextField("Disputed Cash Amount (UZS)", text: $rrnDisputedCash)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        Button("Apply Compensating Ledger") {
                            let minor = (Int64(rrnDisputedCash) ?? 0) * 100
                            Task {
                                await vm.reclassifyTender(orderId: rrnOrderId, gatewayRrn: rrnGateway, disputedCashMinor: minor)
                                showRRNSheet = false
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 44)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                        Spacer()
                    }
                    .padding(20)
                    .navigationTitle("RRN Reclassification")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { showRRNSheet = false }
                        }
                    }
                }
            }
            .sheet(isPresented: $showCITSheet) {
                NavigationStack {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Log physical cash handoff to authorized CIT armored courier.")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                        TextField("Courier Badge ID", text: $citBadgeId)
                            .textFieldStyle(.roundedBorder)
                        TextField("CIT Security Bag Seal Barcode", text: $citSealBarcode)
                            .textFieldStyle(.roundedBorder)
                        TextField("Cash Handover (UZS)", text: $citAmountUzs)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        Button("Confirm Armored Handover") {
                            let minor = (Int64(citAmountUzs) ?? 0) * 100
                            Task {
                                await vm.executeCITHandover(courierBadgeId: citBadgeId, citSealBarcode: citSealBarcode, amountMinor: minor)
                                showCITSheet = false
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 44)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                        Spacer()
                    }
                    .padding(20)
                    .navigationTitle("Armored Handover")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { showCITSheet = false }
                        }
                    }
                }
            }
            .sheet(isPresented: $showAMLSheet) {
                NavigationStack {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Audit legal entity multi-tender cash exposure.")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                        TextField("Legal Entity TIN", text: $amlTin)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        TextField("Amount (UZS)", text: $amlAmountUzs)
                            .textFieldStyle(.roundedBorder)
                            .keyboardType(.numberPad)
                        Button("Audit Cash Exposure") {
                            let minor = (Int64(amlAmountUzs) ?? 0) * 100
                            Task {
                                await vm.checkAMLExposure(tin: amlTin, cashMinor: minor)
                                showAMLSheet = false
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 44)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 10))
                        Spacer()
                    }
                    .padding(20)
                    .navigationTitle("Exposure Audit")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { showAMLSheet = false }
                        }
                    }
                }
            }
        }
    }
}

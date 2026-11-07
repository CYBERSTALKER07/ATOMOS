//
//  MidRouteRescueView.swift
//  driverappios
//

import SwiftUI

struct MidRouteRescueView: View {

    @State private var originalRouteId: String = "route-tashkent-042"
    @State private var brokenVehicleId: String = "vh-isuzu-01"
    @State private var originalDriverId: String = "driver-uz-09"
    @State private var rescueVehicleId: String = "vh-isuzu-04"
    @State private var rescueDriverId: String = "driver-uz-12"
    @State private var lastCompletedStopSeq: Int64 = 2
    @State private var newRescueBoltSeal: String = "SEAL-RESCUE-8812"

    @State private var totes: [RescueToteItem] = [
        RescueToteItem(toteId: "tote-01", stopSequence: 1, isColdChain: false, expectedCashMinor: 50000000),
        RescueToteItem(toteId: "tote-02", stopSequence: 2, isColdChain: false, expectedCashMinor: 75000000),
        RescueToteItem(toteId: "tote-03", stopSequence: 3, isColdChain: true, expectedCashMinor: 120000000, thermalExcursionMinutes: 45, maxAllowedExcursionMins: 30),
        RescueToteItem(toteId: "tote-04", stopSequence: 4, isColdChain: false, expectedCashMinor: 60000000),
        RescueToteItem(toteId: "tote-05", stopSequence: 5, isColdChain: true, expectedCashMinor: 90000000, thermalExcursionMinutes: 10, maxAllowedExcursionMins: 30),
    ]

    @State private var isExecuting: Bool = false
    @State private var rescueResult: MidRouteRescueResult? = nil
    @State private var statusMessage: String? = nil
    @State private var errorMessage: String? = nil

    let onBack: () -> Void

    var originalDriverLiability: Int64 {
        totes.filter { $0.stopSequence <= lastCompletedStopSeq }.reduce(0) { $0 + $1.expectedCashMinor }
    }

    var remainingTotes: [RescueToteItem] {
        totes.filter { $0.stopSequence > lastCompletedStopSeq }
    }

    var spoiledTotes: [RescueToteItem] {
        remainingTotes.filter { $0.isColdChain && $0.thermalExcursionMinutes > $0.maxAllowedExcursionMins }
    }

    var transferredTotes: [RescueToteItem] {
        remainingTotes.filter { !($0.isColdChain && $0.thermalExcursionMinutes > $0.maxAllowedExcursionMins) }
    }

    var rescueDriverLiability: Int64 {
        transferredTotes.reduce(0) { $0 + $1.expectedCashMinor }
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Mid-Route Rescue Handoff")
                            .font(.system(size: 24, weight: .bold))
                        Text("Breakdown Cargo & Cash Liability Transfer Protocol")
                            .font(.system(size: 14))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.top, 12)

                    // Liability Boundary Summary Card
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Cash Liability Boundary (Stop K = \(lastCompletedStopSeq))")
                            .font(.system(size: 15, weight: .bold))

                        HStack {
                            VStack(alignment: .leading) {
                                Text("Original Driver Liability")
                                    .font(.system(size: 12))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Text("\(originalDriverLiability / 100) UZS")
                                    .font(.system(size: 16, weight: .bold))
                                    .foregroundStyle(Color.blue)
                            }
                            Spacer()
                            VStack(alignment: .trailing) {
                                Text("Rescue Driver Liability")
                                    .font(.system(size: 12))
                                    .foregroundStyle(LabTheme.fgSecondary)
                                Text("\(rescueDriverLiability / 100) UZS")
                                    .font(.system(size: 16, weight: .bold))
                                    .foregroundStyle(Color.green)
                            }
                        }

                        if !spoiledTotes.isEmpty {
                            HStack {
                                Image(systemName: "thermometer.snowflake")
                                    .foregroundStyle(Color.red)
                                Text("Thermal Excursion Quarantined:")
                                    .font(.system(size: 12, weight: .semibold))
                                    .foregroundStyle(Color.red)
                                Spacer()
                                Text("\(spoiledTotes.count) totes spoiled")
                                    .font(.system(size: 12, weight: .bold))
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

                    // Vehicle & Seal Inputs
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Handoff Telemetry")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        HStack {
                            Text("Broken Vehicle:")
                                .font(.system(size: 14))
                            Spacer()
                            Text(brokenVehicleId)
                                .font(.system(size: 14, weight: .bold, design: .monospaced))
                        }

                        HStack {
                            Text("Rescue Vehicle:")
                                .font(.system(size: 14))
                            Spacer()
                            Text(rescueVehicleId)
                                .font(.system(size: 14, weight: .bold, design: .monospaced))
                        }

                        VStack(alignment: .leading, spacing: 4) {
                            Text("New Rescue Bolt Seal Barcode:")
                                .font(.system(size: 12))
                                .foregroundStyle(LabTheme.fgSecondary)
                            TextField("Enter or scan seal barcode", text: $newRescueBoltSeal)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(size: 14, design: .monospaced))
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))

                    // Totes List
                    VStack(alignment: .leading, spacing: 10) {
                        Text("Cargo Totes Handoff List")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        ForEach(totes) { tote in
                            let isCompleted = tote.stopSequence <= lastCompletedStopSeq
                            let isSpoiled = tote.isColdChain && tote.thermalExcursionMinutes > tote.maxAllowedExcursionMins

                            HStack {
                                VStack(alignment: .leading, spacing: 2) {
                                    HStack {
                                        Text(tote.toteId)
                                            .font(.system(size: 14, weight: .bold, design: .monospaced))
                                        if tote.isColdChain {
                                            Image(systemName: "snowflake")
                                                .font(.system(size: 12))
                                                .foregroundStyle(Color.blue)
                                        }
                                    }
                                    Text("Stop #\(tote.stopSequence) · \(tote.expectedCashMinor / 100) UZS")
                                        .font(.system(size: 12))
                                        .foregroundStyle(LabTheme.fgSecondary)
                                }

                                Spacer()

                                if isCompleted {
                                    Text("DELIVERED (STOP ≤ K)")
                                        .font(.system(size: 10, weight: .bold))
                                        .foregroundStyle(Color.blue)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 3)
                                        .background(Color.blue.opacity(0.12))
                                        .clipShape(Capsule())
                                } else if isSpoiled {
                                    Text("SPOILED (EXCURSION)")
                                        .font(.system(size: 10, weight: .bold))
                                        .foregroundStyle(Color.red)
                                        .padding(.horizontal, 6)
                                        .padding(.vertical, 3)
                                        .background(Color.red.opacity(0.12))
                                        .clipShape(Capsule())
                                } else {
                                    Text("TRANSFERRED")
                                        .font(.system(size: 10, weight: .bold))
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
                        }
                    }

                    if let msg = statusMessage {
                        Text(msg)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(Color.green)
                            .padding(12)
                    }

                    if let err = errorMessage {
                        Text(err)
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(Color.red)
                            .padding(12)
                    }

                    Button {
                        Task {
                            await executeRescue()
                        }
                    } label: {
                        HStack {
                            if isExecuting {
                                ProgressView().tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text("Confirm Cargo & Liability Transfer")
                                .font(.system(size: 16, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                        .background(Color.blue)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }
                    .disabled(isExecuting)

                    Spacer(minLength: 40)
                }
                .padding(.horizontal, 20)
            }
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

    @MainActor
    private func executeRescue() async {
        isExecuting = true
        errorMessage = nil
        statusMessage = nil

        let req = MidRouteRescueRequest(
            cellId: "cell-uz",
            supplierId: "supplier-tashkent-1",
            originalRouteId: originalRouteId,
            brokenVehicleId: brokenVehicleId,
            originalDriverId: originalDriverId,
            rescueVehicleId: rescueVehicleId,
            rescueDriverId: rescueDriverId,
            lastCompletedStopSeq: lastCompletedStopSeq,
            newRescueBoltSeal: newRescueBoltSeal,
            totes: totes
        )

        do {
            let res = try await APIClient.shared.executeMidRouteRescue(request: req)
            isExecuting = false
            rescueResult = res
            statusMessage = "Rescue Route \(res.rescueRouteId) Created! Transferred \(res.transferredToteIds.count) totes. Liability sealed."
        } catch {
            isExecuting = false
            errorMessage = error.localizedDescription
        }
    }
}

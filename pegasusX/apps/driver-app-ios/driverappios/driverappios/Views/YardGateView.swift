//
//  YardGateView.swift
//  driverappios
//

import SwiftUI

struct YardGateView: View {

    @State private var depotId: String = "depot-tashkent-south"
    @State private var supplierId: String = "supplier-tashkent-1"
    @State private var vehicleId: String = "vh-isuzu-01"
    @State private var licensePlate: String = "01 777 AAA"
    @State private var direction: String = "GATE_OUT"
    @State private var vehiclePurpose: String = "OUTBOUND_ROUTE"
    @State private var boltSealBarcode: String = "SEAL-UZ-44912"
    @State private var sealIntact: Bool = true
    @State private var assignedBaySlot: String = "OUTBOUND_LOADING"

    @State private var isSubmitting: Bool = false
    @State private var recordedEvent: YardGateCheckEvent? = nil
    @State private var statusMessage: String? = nil
    @State private var errorMessage: String? = nil

    let onBack: () -> Void

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Yard Gate Check-In/Out")
                            .font(.system(size: 24, weight: .bold))
                        Text("ALPR Vehicle & Bolt Seal Security Verification")
                            .font(.system(size: 14))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    .padding(.top, 12)

                    // Gate Direction Picker
                    Picker("Direction", selection: $direction) {
                        Text("Gate Out (Departure)").tag("GATE_OUT")
                        Text("Gate In (Arrival)").tag("GATE_IN")
                    }
                    .pickerStyle(.segmented)

                    // Vehicle & Plate Card
                    VStack(alignment: .leading, spacing: 14) {
                        Text("ALPR Optical Recognition")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        HStack {
                            Text("License Plate:")
                                .font(.system(size: 14))
                            Spacer()
                            Text(licensePlate)
                                .font(.system(size: 16, weight: .heavy, design: .monospaced))
                                .padding(.horizontal, 10)
                                .padding(.vertical, 4)
                                .background(Color.yellow.opacity(0.2))
                                .clipShape(RoundedRectangle(cornerRadius: 6))
                        }

                        HStack {
                            Text("Vehicle Unit ID:")
                                .font(.system(size: 14))
                            Spacer()
                            Text(vehicleId)
                                .font(.system(size: 14, weight: .bold, design: .monospaced))
                        }

                        HStack {
                            Text("Purpose:")
                                .font(.system(size: 14))
                            Spacer()
                            Text(vehiclePurpose)
                                .font(.system(size: 13, weight: .bold))
                                .foregroundStyle(Color.blue)
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))

                    // Bolt Seal Card
                    VStack(alignment: .leading, spacing: 14) {
                        Text("Tamper-Evident Bolt Seal")
                            .font(.system(size: 14, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)

                        VStack(alignment: .leading, spacing: 4) {
                            Text("Seal Barcode:")
                                .font(.system(size: 12))
                                .foregroundStyle(LabTheme.fgSecondary)
                            TextField("Enter or scan seal barcode", text: $boltSealBarcode)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(size: 14, design: .monospaced))
                        }

                        Toggle("Physical Bolt Seal Intact", isOn: $sealIntact)
                            .tint(.green)
                            .onChange(of: sealIntact) { _, intact in
                                assignedBaySlot = intact ? (direction == "GATE_OUT" ? "OUTBOUND_LOADING" : "BAY_3_INSPECTION") : "SECURITY_QUARANTINE_BAY"
                            }

                        HStack {
                            Text("Assigned Bay Slot:")
                                .font(.system(size: 14))
                            Spacer()
                            Text(assignedBaySlot)
                                .font(.system(size: 14, weight: .heavy, design: .monospaced))
                                .foregroundStyle(sealIntact ? Color.primary : Color.red)
                        }

                        if !sealIntact {
                            Text("CRITICAL: Broken or mismatched seal triggers automatic quarantine bay redirection.")
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(Color.red)
                        }
                    }
                    .padding(16)
                    .background(LabTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: 16))

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
                            await submitGateCheck()
                        }
                    } label: {
                        HStack {
                            if isSubmitting {
                                ProgressView().tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text(sealIntact ? "Authorize Gate Passage" : "Quarantine & Alert Yard Marshall")
                                .font(.system(size: 16, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 52)
                        .background(sealIntact ? Color.blue : Color.red)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 14))
                    }
                    .disabled(isSubmitting)

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
    private func submitGateCheck() async {
        isSubmitting = true
        errorMessage = nil
        statusMessage = nil

        let event = YardGateCheckEvent(
            depotId: depotId,
            supplierId: supplierId,
            vehicleId: vehicleId,
            licensePlate: licensePlate,
            direction: direction,
            vehiclePurpose: vehiclePurpose,
            boltSealBarcode: boltSealBarcode,
            sealIntact: sealIntact,
            assignedBaySlot: assignedBaySlot
        )

        do {
            let res = try await APIClient.shared.recordGateEvent(event: event)
            isSubmitting = false
            recordedEvent = res
            statusMessage = "Gate Event Logged: \(direction) for \(licensePlate) -> Bay: \(assignedBaySlot)"
        } catch {
            isSubmitting = false
            errorMessage = error.localizedDescription
        }
    }
}

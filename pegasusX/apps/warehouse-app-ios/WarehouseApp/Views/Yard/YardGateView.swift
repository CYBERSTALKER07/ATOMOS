//
//  YardGateView.swift
//  WarehouseAppIOS
//

import SwiftUI

struct YardGateView: View {

    @State private var depotId: String = "depot-tashkent-south"
    @State private var supplierId: String = "supplier-tashkent-1"
    @State private var vehicleId: String = "vh-isuzu-01"
    @State private var licensePlate: String = "01 777 AAA"
    @State private var direction: String = "GATE_IN"
    @State private var vehiclePurpose: String = "INBOUND_SUPPLIER_DROP"
    @State private var boltSealBarcode: String = "SEAL-UZ-11029"
    @State private var sealIntact: Bool = true
    @State private var assignedBaySlot: String = "BAY_3_INSPECTION"

    @State private var isSubmitting: Bool = false
    @State private var recordedEvent: YardGateCheckEvent? = nil
    @State private var statusMessage: String? = nil
    @State private var errorMessage: String? = nil

    let onBack: () -> Void

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 18) {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("Yard Gate Control (Role 9)")
                            .font(.system(size: 24, weight: .bold))
                        Text("Inbound / Outbound ALPR & Tamper-Evident Bolt Seal Verification")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                    .padding(.top, 10)

                    Picker("Direction", selection: $direction) {
                        Text("Inbound Entry").tag("GATE_IN")
                        Text("Outbound Exit").tag("GATE_OUT")
                    }
                    .pickerStyle(.segmented)

                    // Vehicle Info
                    VStack(alignment: .leading, spacing: 12) {
                        Text("ALPR Optical Recognition")
                            .font(.system(size: 14, weight: .semibold))

                        HStack {
                            Text("License Plate:")
                                .font(.system(size: 13))
                                .foregroundStyle(.secondary)
                            Spacer()
                            Text(licensePlate)
                                .font(.system(size: 15, weight: .heavy, design: .monospaced))
                                .padding(.horizontal, 8)
                                .padding(.vertical, 4)
                                .background(Color.yellow.opacity(0.2))
                                .clipShape(RoundedRectangle(cornerRadius: 6))
                        }

                        HStack {
                            Text("Vehicle Unit:")
                                .font(.system(size: 13))
                                .foregroundStyle(.secondary)
                            Spacer()
                            Text(vehicleId)
                                .font(.system(size: 14, weight: .bold, design: .monospaced))
                        }

                        HStack {
                            Text("Purpose:")
                                .font(.system(size: 13))
                                .foregroundStyle(.secondary)
                            Spacer()
                            Text(vehiclePurpose)
                                .font(.system(size: 13, weight: .semibold))
                                .foregroundStyle(Color.blue)
                        }
                    }
                    .padding(14)
                    .background(Color(.secondarySystemGroupedBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 14))

                    // Seal Verification
                    VStack(alignment: .leading, spacing: 12) {
                        Text("Bolt Seal Verification")
                            .font(.system(size: 14, weight: .semibold))

                        VStack(alignment: .leading, spacing: 4) {
                            Text("Bolt Seal Barcode:")
                                .font(.system(size: 12))
                                .foregroundStyle(.secondary)
                            TextField("Seal barcode", text: $boltSealBarcode)
                                .textFieldStyle(.roundedBorder)
                                .font(.system(size: 13, design: .monospaced))
                        }

                        Toggle("Bolt Seal Verified Intact", isOn: $sealIntact)
                            .tint(.green)
                            .onChange(of: sealIntact) { _, intact in
                                assignedBaySlot = intact ? "BAY_3_INSPECTION" : "SECURITY_QUARANTINE_BAY"
                            }

                        HStack {
                            Text("Assigned Bay Slot:")
                                .font(.system(size: 13))
                            Spacer()
                            Text(assignedBaySlot)
                                .font(.system(size: 14, weight: .heavy, design: .monospaced))
                                .foregroundStyle(sealIntact ? Color.primary : Color.red)
                        }

                        if !sealIntact {
                            Text("Tampered seal detected: Vehicle routed directly to security quarantine bay.")
                                .font(.system(size: 11, weight: .semibold))
                                .foregroundStyle(Color.red)
                        }
                    }
                    .padding(14)
                    .background(Color(.secondarySystemGroupedBackground))
                    .clipShape(RoundedRectangle(cornerRadius: 14))

                    if let msg = statusMessage {
                        Text(msg)
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(Color.green)
                    }

                    if let err = errorMessage {
                        Text(err)
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(Color.red)
                    }

                    Button {
                        Task { await submitGateEvent() }
                    } label: {
                        HStack {
                            if isSubmitting {
                                ProgressView().tint(.white)
                                Spacer().frame(width: 8)
                            }
                            Text(sealIntact ? "Confirm Gate Clearance" : "Quarantine Vehicle Entry")
                                .font(.system(size: 15, weight: .bold))
                        }
                        .frame(maxWidth: .infinity)
                        .frame(height: 48)
                        .background(sealIntact ? Color.blue : Color.red)
                        .foregroundStyle(.white)
                        .clipShape(RoundedRectangle(cornerRadius: 12))
                    }
                    .disabled(isSubmitting)

                    Spacer(minLength: 40)
                }
                .padding(.horizontal, 16)
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
    private func submitGateEvent() async {
        isSubmitting = true
        errorMessage = nil
        statusMessage = nil

        let evt = YardGateCheckEvent(
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
            let res = try await WarehouseService.recordGateEvent(event: evt)
            isSubmitting = false
            recordedEvent = res
            statusMessage = "Gate Event Logged: \(direction) -> \(assignedBaySlot)"
        } catch {
            isSubmitting = false
            errorMessage = error.localizedDescription
        }
    }
}

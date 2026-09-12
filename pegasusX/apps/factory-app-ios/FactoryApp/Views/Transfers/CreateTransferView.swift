import SwiftUI

struct CreateTransferView: View {
    @Environment(\.dismiss) private var dismiss
    let onCreated: (String) -> Void

    @State private var realtimeClient = FactoryRealtimeClient()
    @State private var loadingFleet = true
    @State private var submitting = false
    @State private var error: String?
    @State private var drivers: [FactoryFleetDriverRow] = []
    @State private var vehicles: [FactoryFleetVehicleRow] = []
    @State private var orderId = ""
    @State private var totalVu = "25"
    @State private var driverId = ""
    @State private var vehicleId = ""

    var body: some View {
        NavigationStack {
            Group {
                if loadingFleet {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") { Task { await loadFleet() } }
                    }
                } else {
                    Form {
                        Section {
                            TextField("mobile_factory.ui.order_id_optional", text: $orderId)
                            TextField("mobile_factory.ui.total_vu", text: $totalVu)
                                .keyboardType(.numberPad)
                        } footer: {
                            Text("mobile_factory.ui.stage_a_factory_to_warehouse_movement_volume_is_captured_in_vu")
                        }

                        Section("Fleet assignment") {
                            Picker("Driver", selection: $driverId) {
                                Text("factory_portal.transfers._id_.text.unassigned").tag("")
                                ForEach(drivers) { driver in
                                    Text("\(driver.name)\(driver.onShift ? " (on shift)" : "")").tag(driver.driverId)
                                }
                            }
                            Picker("Vehicle", selection: $vehicleId) {
                                Text("factory_portal.transfers._id_.text.unassigned").tag("")
                                ForEach(vehicles) { vehicle in
                                    Text(L10n.format("mobile_factory.ui.plateno_state", "\(vehicle.plateNo)", "\(vehicle.state)")).tag(vehicle.vehicleId)
                                }
                            }
                        }
                    }
                }
            }
            .navigationTitle("mobile_factory.ui.create_transfer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(submitting ? "Creating…" : "Create") {
                        Task { await submit() }
                    }
                    .disabled(submitting || loadingFleet)
                }
            }
            .task { await loadFleet() }
            .onAppear {
                realtimeClient.connect(
                    onStateChange: { _ in },
                    onEvent: { _ in },
                    onReconnect: {
                        if submitting {
                            submitting = false
                            error = "Connection restored — verify transfer was created before retrying."
                        }
                    }
                )
            }
            .onDisappear {
                realtimeClient.disconnect()
            }
        }
    }

    @MainActor
    private func loadFleet() async {
        loadingFleet = true
        error = nil
        do {
            async let driverRows = FactoryService.fleetDrivers()
            async let vehicleRows = FactoryService.fleetVehicles()
            drivers = try await driverRows
            vehicles = try await vehicleRows
        } catch {
            self.error = error.localizedDescription
        }
        loadingFleet = false
    }

    @MainActor
    private func submit() async {
        guard let parsedVu = Int(totalVu), parsedVu > 0 else {
            error = "Total VU must be a positive number."
            return
        }
        submitting = true
        error = nil
        do {
            let response = try await FactoryService.createTransfer(
                FactoryCreateTransferRequest(
                    orderId: orderId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty ? nil : orderId,
                    totalVu: parsedVu,
                    driverId: driverId.isEmpty ? nil : driverId,
                    vehicleId: vehicleId.isEmpty ? nil : vehicleId
                )
            )
            onCreated(response.transferId)
            dismiss()
        } catch {
            self.error = error.localizedDescription
        }
        submitting = false
    }
}

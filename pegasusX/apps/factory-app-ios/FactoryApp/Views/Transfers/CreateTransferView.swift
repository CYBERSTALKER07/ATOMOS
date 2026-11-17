import SwiftUI

struct CreateTransferView: View {
    @Environment(\.dismiss) private var dismiss
    let onCreated: (String) -> Void

    @State private var vm = CreateTransferViewModel()

    var body: some View {
        @Bindable var vm = vm

        NavigationStack {
            Group {
                if vm.loadingFleet {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error = vm.error, vm.drivers.isEmpty, vm.vehicles.isEmpty {
                    ContentUnavailableView {
                        Label("mobile_factory.ui.error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("common.action.retry") {
                            PegasusHaptics.selection()
                            Task { await vm.loadFleet() }
                        }
                        .frame(minWidth: 44, minHeight: 44)
                    }
                } else {
                    Form {
                        if let error = vm.error {
                            Section {
                                Text(error)
                                    .foregroundStyle(.red)
                                    .font(.caption)
                                    .accessibilityLabel("Error: \(error)")
                            }
                        }

                        Section {
                            TextField("mobile_factory.ui.order_id_optional", text: $vm.orderId)
                                .frame(minHeight: 44)
                                .accessibilityLabel("Optional order ID")
                            TextField("mobile_factory.ui.total_vu", text: $vm.totalVu)
                                .keyboardType(.numberPad)
                                .frame(minHeight: 44)
                                .accessibilityLabel("Total volume units")
                        } footer: {
                            Text("mobile_factory.ui.stage_a_factory_to_warehouse_movement_volume_is_captured_in_vu")
                        }

                        Section("Fleet assignment") {
                            Picker("Driver", selection: $vm.driverId) {
                                Text("factory_portal.transfers._id_.text.unassigned").tag("")
                                ForEach(vm.drivers) { driver in
                                    Text("\(driver.name)\(driver.onShift ? " (on shift)" : "")").tag(driver.driverId)
                                }
                            }
                            Picker("Vehicle", selection: $vm.vehicleId) {
                                Text("factory_portal.transfers._id_.text.unassigned").tag("")
                                ForEach(vm.vehicles) { vehicle in
                                    Text(L10n.format("mobile_factory.ui.plateno_state", "\(vehicle.plateNo)", "\(vehicle.state)")).tag(vehicle.vehicleId)
                                }
                            }
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                }
            }
            .navigationTitle("mobile_factory.ui.create_transfer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("common.action.cancel") {
                        PegasusHaptics.selection()
                        dismiss()
                    }
                    .frame(minWidth: 44, minHeight: 44)
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(vm.submitting ? "Creating…" : "Create") {
                        Task {
                            if let transferId = await vm.submit() {
                                onCreated(transferId)
                                dismiss()
                            }
                        }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .disabled(vm.submitting || vm.loadingFleet)
                }
            }
            .task {
                vm.startRealtime()
                await vm.loadFleet()
            }
            .onDisappear {
                vm.stopRealtime()
            }
        }
    }
}

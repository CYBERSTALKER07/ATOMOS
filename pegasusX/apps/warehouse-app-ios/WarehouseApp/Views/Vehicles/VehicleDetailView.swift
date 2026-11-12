import SwiftUI

struct VehicleDetailView: View {
    let vehicleId: String

    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @State private var vm = VehicleDetailViewModel()

    var body: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.vehicle == nil {
                ProgressView("Loading truck…")
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else if let error = vm.error, vm.vehicle == nil {
                ContentUnavailableView {
                    Label("mobile_warehouse.ui.error", systemImage: "exclamationmark.triangle")
                } description: {
                    Text(error)
                } actions: {
                    Button("common.action.retry") {
                        PegasusHaptics.light()
                        Task { await vm.load(vehicleId: vehicleId) }
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                }
            } else if let vehicle = vm.vehicle {
                ScrollView {
                    VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label)
                                .font(.title2.bold())
                            Text(L10n.format("mobile_warehouse.ui.licenseplate_vehicleclass_capacityvu_vu", "\(vehicle.licensePlate)", "\(vehicle.vehicleClass)", "\(vehicle.capacityVu)"))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                            Text(vehicle.isActive ? (vehicle.status.isEmpty ? "ACTIVE" : vehicle.status) : "UNAVAILABLE")
                                .font(.caption.bold())
                                .padding(.horizontal, LabTheme.spacingSM)
                                .padding(.vertical, LabTheme.spacingXS)
                                .background(.quaternary, in: Capsule())
                        }
                        .accessibilityElement(children: .combine)

                        WarehouseSectionHeader(title: "Assignment", subtitle: nil)
                        Text(vehicle.assignedDriverName ?? "Unassigned")
                            .font(.body)

                        WarehouseSectionHeader(title: "Dispatch impact", subtitle: nil)
                        Text(
                            vehicle.isActive
                                ? "This truck is eligible for manual and smart dispatch."
                                : "Excluded from dispatch: \(formatUnavailableReason(vehicle.unavailableReason, note: vehicle.unavailableNote))"
                        )
                        .font(.subheadline)
                        .foregroundStyle(.secondary)

                        VehicleAvailabilityPanel(
                            vehicle: vehicle,
                            mutating: vm.mutating,
                            selectedReason: $vm.selectedReason,
                            customNote: $vm.customNote,
                            onMarkUnavailable: { reason, note in
                                Task {
                                    await vm.updateAvailability(
                                        vehicleId: vehicleId,
                                        isActive: false,
                                        reason: reason,
                                        note: note
                                    )
                                }
                            },
                            onRestore: {
                                Task {
                                    await vm.updateAvailability(vehicleId: vehicleId, isActive: true)
                                }
                            }
                        )
                    }
                    .padding(LabTheme.spacingLG)
                }
                .scrollDismissesKeyboard(.interactively)
            }
        }
        .background(LabTheme.background)
        .navigationTitle((vm.vehicle?.label.isEmpty == false ? vm.vehicle?.label : nil) ?? "Truck")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load(vehicleId: vehicleId) }
                } label: {
                    Label("portal.page.orders.action.refresh", systemImage: "arrow.clockwise")
                        .frame(minWidth: 44, minHeight: 44)
                        .contentShape(Rectangle())
                }
                .accessibilityLabel("Refresh truck details")
            }
        }
        .task(id: vehicleId) {
            await vm.load(vehicleId: vehicleId)
        }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            Task { await vm.load(vehicleId: vehicleId, silent: silent) }
        }
    }
}

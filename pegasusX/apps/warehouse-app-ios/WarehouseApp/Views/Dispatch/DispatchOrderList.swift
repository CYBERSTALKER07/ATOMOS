import SwiftUI

/// Reusable orders segment for the Dispatch screen.
///
/// Renders fleet truck cards, dispatch-mode picker (Smart / Manual),
/// driver picker, undispatched order list with selection checkboxes,
/// smart-suggest preview warnings, and proposed routes.
struct DispatchOrderList: View {
    let preview: DispatchPreview
    let fleetVehicles: [Vehicle]
    @Binding var vehicleReasons: [String: String]
    @Binding var vehicleNotes: [String: String]
    let mutatingFleetVehicleId: String?
    @Binding var dispatchMode: DispatchAssignmentMode
    @Binding var selectedDriverId: String
    @Binding var selectedOrderIds: Set<String>
    let executing: Bool

    // Callbacks
    var onManualDispatch: () -> Void
    var onSmartDispatch: () -> Void
    var onProposeDate: (_ orderId: String) -> Void
    var onReject: (_ orderId: String) -> Void
    var onOrderDoubleTap: (_ orderId: String) -> Void
    var onMarkVehicleUnavailable: (_ vehicle: Vehicle) -> Void
    var onRestoreVehicle: (_ vehicle: Vehicle) -> Void

    private let dispatchTetrisBuffer: Double = 0.95

    var body: some View {
        let selectedDriver = preview.availableDrivers.first(where: { $0.driverId == selectedDriverId })
        let selectedVolume = preview.undispatchedOrders
            .filter { selectedOrderIds.contains($0.orderId) }
            .reduce(0.0) { $0 + $1.volumeVu }
        let effectiveMax: Double = {
            guard let driver = selectedDriver else { return 0 }
            if let free = driver.freeVolumeVu, free > 0 {
                return free * dispatchTetrisBuffer
            }
            return driver.maxVolumeVu * dispatchTetrisBuffer
        }()

        ResponsiveGridContentWrapper {
            // ── Fleet trucks ──
            if !fleetVehicles.isEmpty {
                Section(L10n.format("mobile_warehouse.ui.fleet_trucks_count", "\(fleetVehicles.count)")) {
                    ForEach(fleetVehicles) { vehicle in
                        let reasonBinding = Binding<String>(
                            get: { vehicleReasons[vehicle.vehicleId] ?? vehicle.unavailableReason ?? "MANUAL_HOLD" },
                            set: { vehicleReasons[vehicle.vehicleId] = $0 }
                        )
                        let noteBinding = Binding<String>(
                            get: { vehicleNotes[vehicle.vehicleId] ?? vehicle.unavailableNote ?? "" },
                            set: { vehicleNotes[vehicle.vehicleId] = $0 }
                        )
                        FleetTruckDispatchCard(
                            vehicle: vehicle,
                            selectedReason: reasonBinding,
                            customNote: noteBinding,
                            mutating: mutatingFleetVehicleId == vehicle.vehicleId,
                            onMarkUnavailable: { onMarkVehicleUnavailable(vehicle) },
                            onRestore: { onRestoreVehicle(vehicle) }
                        )
                    }
                }
            }

            // ── Empty state ──
            if preview.undispatchedOrders.isEmpty {
                Section {
                    ContentUnavailableView("All Dispatched", systemImage: "checkmark.circle", description: Text("mobile_warehouse.ui.no_pending_orders"))
                }
            } else {
                // ── Dispatch mode selector ──
                Section {
                    Picker("Dispatch mode", selection: $dispatchMode) {
                        ForEach(DispatchAssignmentMode.allCases) { mode in
                            Text(mode.label).tag(mode)
                        }
                    }
                    .pickerStyle(.segmented)
                    if dispatchMode == .manual {
                    Picker("Truck / driver", selection: $selectedDriverId) {
                        Text("mobile_warehouse.ui.select_truck_driver").tag("")
                        ForEach(preview.availableDrivers) { driver in
                            Text(driverPickerLabel(driver)).tag(driver.driverId)
                        }
                    }
                    if selectedDriver != nil {
                        Text(L10n.format("mobile_warehouse.ui.loaded_n_1f_n_1f_2_vu_effective", String(format: "%.1f", selectedVolume), String(format: "%.1f", effectiveMax)))
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    Button {
                        onManualDispatch()
                    } label: {
                        Text(executing ? "Dispatching…" : "Manual (\(selectedOrderIds.count))")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(executing || selectedDriverId.isEmpty || selectedOrderIds.isEmpty)
                    } else {
                    Text("mobile_warehouse.ui.trucks_are_assigned_automatically_across_the_fleet")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    Button {
                        onSmartDispatch()
                    } label: {
                        Text("mobile_warehouse.ui.smart_dispatch")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(executing || preview.undispatchedOrders.isEmpty || preview.availableDrivers.isEmpty)
                    }
                    if preview.fleetEffectiveCapacityVu > 0 {
                        Text(L10n.format("mobile_warehouse.ui.fleet_n_1f_vu_effective", String(format: "%.1f", preview.fleetEffectiveCapacityVu)))
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                }

                // ── Orders list ──
                Section("Orders") {
                    ForEach(preview.undispatchedOrders) { order in
                        HStack(alignment: .center, spacing: LabTheme.spacingSM) {
                            Button {
                                if selectedOrderIds.contains(order.orderId) {
                                    selectedOrderIds.remove(order.orderId)
                                } else {
                                    selectedOrderIds.insert(order.orderId)
                                }
                            } label: {
                                Image(systemName: selectedOrderIds.contains(order.orderId) ? "checkmark.circle.fill" : "circle")
                                    .foregroundStyle(selectedOrderIds.contains(order.orderId) ? Color.accentColor : .secondary)
                            }
                            .buttonStyle(.plain)

                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                Text(order.retailerName.isEmpty ? String(order.orderId.prefix(8)) : order.retailerName)
                                    .font(.headline)
                                Text(L10n.format("mobile_warehouse.ui.formatted_uzs_n_1f_vu", "\(order.totalUzs.formatted())", String(format: "%.1f", order.volumeVu ?? 0)))
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                HStack(spacing: LabTheme.spacingSM) {
                                    Button("mobile_warehouse.ui.propose_date") {
                                        onProposeDate(order.orderId)
                                    }
                                    .buttonStyle(.bordered)
                                    .controlSize(.small)
                                    Button("common.action.cancel", role: .destructive) {
                                        onReject(order.orderId)
                                    }
                                    .buttonStyle(.bordered)
                                    .controlSize(.small)
                                }
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .contentShape(Rectangle())
                            .onTapGesture(count: 2) {
                                onOrderDoubleTap(order.orderId)
                            }
                            .accessibilityHint("Double-tap to open order detail")
                        }
                    }
                }

                // ── Smart suggest preview ──
                if !preview.proposedRoutes.isEmpty || !preview.optimizerWarnings.isEmpty || preview.windowConstrainedCount > 0 {
                    Section("Smart suggest preview") {
                        if preview.windowConstrainedCount > 0 {
                            Text(L10n.format("mobile_warehouse.ui.windowconstrainedcount_order_s_constrained_by_receiving_window", "\(preview.windowConstrainedCount)"))
                                .font(.caption)
                                .foregroundStyle(.orange)
                        }
                        if let source = preview.optimizerSource, !source.isEmpty {
                            Text(L10n.format("mobile_warehouse.ui.source_source_2", "\(source)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                        ForEach(preview.optimizerWarnings, id: \.self) { warning in
                            Text(warning)
                                .font(.caption)
                                .foregroundStyle(.orange)
                        }
                    }
                }

                // ── Proposed routes ──
                if !preview.proposedRoutes.isEmpty {
                    Section(L10n.format("mobile_warehouse.ui.smart_suggest_routes_count", "\(preview.proposedRoutes.count)")) {
                        DispatchPreviewMapView(routes: preview.proposedRoutes)
                            .listRowInsets(EdgeInsets())
                        ForEach(preview.proposedRoutes) { route in
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                HStack {
                                    Text(route.driverName ?? route.driverId ?? "Driver")
                                        .font(.headline)
                                    Spacer()
                                    Text(L10n.format("mobile_warehouse.ui.count_stops_n_1f_vu", "\(route.stopCount ?? route.orderIds.count)", String(format: "%.1f", route.volumeVu ?? route.loadedVolume ?? 0)))
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                }
                                Text(route.orderIds.joined(separator: " → "))
                                    .font(.caption.monospaced())
                                    .lineLimit(2)
                            }
                        }
                    }
                }
            }
        }
    }

    private func driverPickerLabel(_ driver: AvailableDriver) -> String {
        let vehicle = driver.vehicleLabel.isEmpty ? driver.truckStatus : driver.vehicleLabel
        return "\(driver.name) · \(vehicle.isEmpty ? "No vehicle" : vehicle)"
    }
}

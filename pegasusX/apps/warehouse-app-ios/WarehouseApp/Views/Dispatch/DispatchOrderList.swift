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
                Section("Fleet trucks (\(fleetVehicles.count))") {
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
                    ContentUnavailableView("All Dispatched", systemImage: "checkmark.circle", description: Text("No pending orders"))
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
                        Text("Select truck / driver").tag("")
                        ForEach(preview.availableDrivers) { driver in
                            Text(driverPickerLabel(driver)).tag(driver.driverId)
                        }
                    }
                    if selectedDriver != nil {
                        Text("Loaded \(selectedVolume, specifier: "%.1f") / \(effectiveMax, specifier: "%.1f") VU effective")
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
                    Text("Trucks are assigned automatically across the fleet.")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    Button {
                        onSmartDispatch()
                    } label: {
                        Text("Smart Dispatch")
                            .frame(maxWidth: .infinity)
                    }
                    .buttonStyle(.borderedProminent)
                    .disabled(executing || preview.undispatchedOrders.isEmpty || preview.availableDrivers.isEmpty)
                    }
                    if preview.fleetEffectiveCapacityVu > 0 {
                        Text("Fleet \(preview.fleetEffectiveCapacityVu, specifier: "%.1f") VU effective")
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
                                Text("\(order.totalUzs.formatted()) UZS · \(order.volumeVu, specifier: "%.1f") VU")
                                    .font(.subheadline)
                                    .foregroundStyle(.secondary)
                                HStack(spacing: LabTheme.spacingSM) {
                                    Button("Propose date") {
                                        onProposeDate(order.orderId)
                                    }
                                    .buttonStyle(.bordered)
                                    .controlSize(.small)
                                    Button("Cancel", role: .destructive) {
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
                            Text("\(preview.windowConstrainedCount) order(s) constrained by receiving window")
                                .font(.caption)
                                .foregroundStyle(.orange)
                        }
                        if let source = preview.optimizerSource, !source.isEmpty {
                            Text("Source: \(source)")
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
                    Section("Smart suggest routes (\(preview.proposedRoutes.count))") {
                        DispatchPreviewMapView(routes: preview.proposedRoutes)
                            .listRowInsets(EdgeInsets())
                        ForEach(preview.proposedRoutes) { route in
                            VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                HStack {
                                    Text(route.driverName ?? route.driverId ?? "Driver")
                                        .font(.headline)
                                    Spacer()
                                    Text("\(route.stopCount ?? route.orderIds.count) stops · \((route.volumeVu ?? route.loadedVolume ?? 0), specifier: "%.1f") VU")
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

import SwiftUI

struct DispatchView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @Environment(\.scenePhase) private var scenePhase
    @State private var vm = DispatchViewModel()

    var body: some View {
        dispatchDialogs
    }

    @ViewBuilder
    private var dispatchBody: some View {
        @Bindable var vm = vm
        Group {
            if vm.loading && vm.preview == nil {
                WarehouseLoadingView(
                    title: "Loading dispatch",
                    message: "Fetching orders, drivers, supply requests, and locks."
                )
            } else if let error = vm.error, vm.preview == nil {
                WarehouseErrorView(message: error) {
                    PegasusHaptics.light()
                    vm.load()
                }
            } else if let preview = vm.preview {
                dispatchContent(preview: preview)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("portal.nav.dispatch")
        .navigationDestination(item: $vm.detailRoute) { route in
            OrderDetailView(orderId: route.id)
        }
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                if vm.selectedSegment == 2 {
                    Button("mobile_warehouse.ui.new_request_2", systemImage: "plus") {
                        PegasusHaptics.light()
                        vm.showCreateSupplyRequest = true
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                    .accessibilityLabel("Create supply request")
                }
                if vm.selectedSegment == 3 && !vm.hasActiveManualDispatchLock {
                    Button("mobile_warehouse.ui.lock", systemImage: "lock.fill") {
                        PegasusHaptics.warning()
                        vm.showAcquireDispatchLock = true
                    }
                    .frame(minWidth: 44, minHeight: 44)
                    .contentShape(Rectangle())
                    .accessibilityLabel("Acquire dispatch lock")
                }
                Button("Rescue", systemImage: "lifepreserver") {
                    PegasusHaptics.light()
                    vm.showRescueDialog = true
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Rescue driver")
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.light()
                    vm.load()
                }
                .frame(minWidth: 44, minHeight: 44)
                .contentShape(Rectangle())
                .accessibilityLabel("Refresh dispatch")
            }
        }
        .task {
            vm.load()
            vm.connectRealtime {
                realtimeHub.bumpReconnect()
            }
        }
        .refreshable { vm.load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            vm.load(silent: silent)
        }
        .onDisappear { vm.disconnectRealtime() }
        .onChange(of: scenePhase) { _, phase in
            switch phase {
            case .active:
                vm.connectRealtime {
                    realtimeHub.bumpReconnect()
                }
            case .inactive, .background:
                vm.disconnectRealtime()
            @unknown default:
                break
            }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if vm.executing {
                vm.executing = false
                vm.actionAlert = DispatchActionAlert(
                    title: "Connection restored",
                    message: "Verify dispatch status before retrying."
                )
            }
        }
    }

    @ViewBuilder
    private var dispatchDialogs: some View {
        @Bindable var vm = vm
        dispatchBody
            .confirmationDialog("Capacity exceeded", isPresented: $vm.showCapacityDialog, titleVisibility: .visible) {
                Button("mobile_warehouse.ui.force_dispatch", role: .destructive) {
                    PegasusHaptics.heavy()
                    Task {
                        if vm.capacityDialogAutoMode {
                            await vm.runSmartDispatch(forceCapacity: true)
                        } else {
                            await vm.runManualDispatch(forceCapacity: true)
                        }
                    }
                }
                if vm.capacityDialogAutoMode {
                    Button("mobile_warehouse.ui.accept_partial") {
                        PegasusHaptics.medium()
                        Task { await vm.runSmartDispatch(acceptPartial: true) }
                    }
                }
                if !vm.capacityDialogAutoMode && !vm.capacitySuggestedUnselect.isEmpty {
                    Button("mobile_warehouse.ui.apply_suggestion") {
                        PegasusHaptics.selection()
                        vm.selectedOrderIds.subtract(vm.capacitySuggestedUnselect)
                        vm.showCapacityDialog = false
                    }
                }
                Button("common.action.cancel", role: .cancel) {}
            } message: {
                Text(vm.capacityExceededMessage)
            }
            .confirmationDialog("Run smart dispatch?", isPresented: $vm.showSmartConfirm, titleVisibility: .visible) {
                Button("mobile_warehouse.ui.smart_dispatch") {
                    PegasusHaptics.medium()
                    Task { await vm.runSmartDispatch() }
                }
                Button("common.action.cancel", role: .cancel) {}
            } message: {
                Text("mobile_warehouse.ui.assign_orders_using_the_optimizer_across_available_trucks")
            }
            .sheet(isPresented: $vm.showCreateSupplyRequest) {
                CreateSupplyRequestSheet { form in
                    Task { await vm.createSupplyRequest(form: form) }
                }
            }
            .alert(item: $vm.actionAlert) { alert in
                Alert(title: Text(alert.title), message: Text(alert.message), dismissButton: .default(Text("OK")))
            }
            .alert(
                "Cancel Supply Request?",
                isPresented: $vm.showCancelSupplyRequestAlert,
                presenting: vm.requestPendingCancellation
            ) { request in
                Button("mobile_warehouse.ui.keep", role: .cancel) {
                    vm.requestPendingCancellation = nil
                }
                Button("notification.cancel_requested.title", role: .destructive) {
                    PegasusHaptics.warning()
                    Task { await vm.cancelSupplyRequest(request) }
                }
            } message: { request in
                Text(L10n.format("mobile_warehouse.ui.cancel_request_prefix_this_keeps_the_warehouse_and_factory_clients_in_sy", "\(request.requestId.prefix(8))"))
            }
            .alert(
                "Release Dispatch Lock?",
                isPresented: $vm.showReleaseDispatchLockAlert,
                presenting: vm.lockPendingRelease
            ) { lock in
                Button("mobile_warehouse.ui.keep", role: .cancel) {
                    vm.lockPendingRelease = nil
                }
                Button("mobile_warehouse.ui.release", role: .destructive) {
                    PegasusHaptics.warning()
                    Task { await vm.releaseDispatchLock(lock) }
                }
            } message: { lock in
                Text(L10n.format("mobile_warehouse.ui.release_locktype_for_this_warehouse_scope", "\(lock.lockType)"))
            }
            .confirmationDialog("Lock dispatch for manual override?", isPresented: $vm.showAcquireDispatchLock, titleVisibility: .visible) {
                Button("mobile_warehouse.ui.acquire_manual_dispatch") {
                    PegasusHaptics.medium()
                    Task { await vm.acquireDispatchLock() }
                }
                Button("common.action.cancel", role: .cancel) {}
            } message: {
                Text("mobile_warehouse.ui.this_freezes_auto_dispatch_changes_until_the_lock_is_released")
            }
            .sheet(item: $vm.proposeRoute) { target in
                NavigationStack {
                    Form {
                        DatePicker("Proposed delivery date", selection: $vm.proposeDate, displayedComponents: .date)
                        Section {
                            TextField("supplier_portal.admin.control_center.field.reason", text: $vm.opsReasonInput, axis: .vertical)
                                .lineLimit(3...5)
                        } footer: {
                            Text("mobile_warehouse.ui.the_retailer_is_notified_and_can_accept_or_reject_the_new_date")
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("mobile_warehouse.ui.propose_new_date")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.cancel") { vm.proposeTarget = nil }
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("mobile_warehouse.ui.send") {
                                PegasusHaptics.medium()
                                Task { await vm.submitDispatchPropose(orderId: target.id) }
                            }
                            .disabled(vm.opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                        }
                    }
                }
                .presentationDetents([.medium])
            }
            .sheet(item: $vm.rejectRoute) { target in
                NavigationStack {
                    Form {
                        Section {
                            TextField("supplier_portal.admin.control_center.field.reason", text: $vm.opsReasonInput, axis: .vertical)
                                .lineLimit(3...5)
                        } footer: {
                            Text("mobile_warehouse.ui.cancels_the_order_and_notifies_the_retailer")
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("warehouse_portal.dispatch.text.cancel_order")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("common.action.back") { vm.rejectRoute = nil }
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("warehouse_portal.dispatch.text.cancel_order", role: .destructive) {
                                PegasusHaptics.warning()
                                Task { await vm.submitDispatchReject(orderId: target.id) }
                            }
                            .disabled(vm.opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                            .frame(minWidth: 44, minHeight: 44)
                            .contentShape(Rectangle())
                        }
                    }
                }
                .presentationDetents([.medium])
            }
            .sheet(isPresented: $vm.showRescueDialog) {
                NavigationStack {
                    Form {
                        if vm.earlyCompleteReq == nil {
                            Section("Driver Lookup") {
                                TextField("Driver ID", text: $vm.lookupDriverId)
                                Button("Find Driver") {
                                    PegasusHaptics.light()
                                    Task { await vm.findRescueDriver() }
                                }
                                .disabled(vm.lookupDriverId.isEmpty || vm.lookupLoading)
                                .frame(minHeight: 44)
                                .contentShape(Rectangle())

                                if vm.lookupLoading { ProgressView() }
                                if let err = vm.lookupError { Text(err).foregroundStyle(.red) }
                            }
                        } else if let req = vm.earlyCompleteReq {
                            Section("Rescue Request") {
                                LabeledContent("Status", value: "\(req["status"] ?? "")")
                                LabeledContent("Reason", value: "\(req["reason"] ?? "")")
                                LabeledContent("Driver", value: "\(vm.lookupDriverId)")
                            }

                            if !vm.rescueMutating && vm.rescueAction == nil {
                                Section("Action") {
                                    Button("Cancel Route", role: .destructive) {
                                        PegasusHaptics.warning()
                                        vm.rescueAction = "CANCEL"
                                    }
                                    .frame(minHeight: 44)
                                    .contentShape(Rectangle())
                                    Button("Reschedule Orders") {
                                        PegasusHaptics.selection()
                                        vm.rescueAction = "RESCHEDULE"
                                    }
                                    .frame(minHeight: 44)
                                    .contentShape(Rectangle())
                                }
                            }

                            if vm.rescueAction == "RESCHEDULE" {
                                Section("Reschedule Details") {
                                    TextField("New Start (e.g. 2026-08-01T09:00:00Z)", text: $vm.rescueWindowStart)
                                    TextField("New End (e.g. 2026-08-01T17:00:00Z)", text: $vm.rescueWindowEnd)
                                }
                            }

                            if let action = vm.rescueAction {
                                Section {
                                    Button("Confirm \(action)") {
                                        PegasusHaptics.medium()
                                        Task { await vm.confirmRescue(action: action) }
                                    }
                                    .disabled(vm.rescueMutating)
                                    .frame(minHeight: 44)
                                    .contentShape(Rectangle())

                                    Button("Cancel") { vm.rescueAction = nil }
                                        .disabled(vm.rescueMutating)
                                        .frame(minHeight: 44)
                                        .contentShape(Rectangle())
                                }
                            }
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .navigationTitle("Driver Rescue")
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Close") { vm.resetRescue() }
                                .frame(minWidth: 44, minHeight: 44)
                                .contentShape(Rectangle())
                        }
                    }
                }
                .presentationDetents([.medium, .large])
            }
    }

    @ViewBuilder
    private func dispatchContent(preview: DispatchPreview) -> some View {
        @Bindable var vm = vm
        VStack(spacing: 0) {
            FleetLiveMapSection(mapHeight: 240, showsExpand: false)
                .padding(.horizontal)
                .padding(.top, LabTheme.spacingSM)

            HandoffTimelineSection(events: vm.handoffEvents, loading: vm.handoffLoading, error: vm.handoffError)
                .padding(.horizontal)
                .padding(.bottom, LabTheme.spacingSM)

            Picker("View", selection: $vm.selectedSegment) {
                Text(L10n.format("mobile_warehouse.ui.orders_count", "\(preview.undispatchedOrders.count)")).tag(0)
                Text(L10n.format("mobile_warehouse.ui.drivers_count", "\(preview.availableDrivers.count + preview.unavailableDrivers.count)")).tag(1)
                Text(L10n.format("mobile_warehouse.ui.supply_count", "\(vm.supplyRequests.count)")).tag(2)
                Text(L10n.format("mobile_warehouse.ui.locks_count", "\(vm.dispatchLocks.count)")).tag(3)
            }
            .pickerStyle(.segmented)
            .padding()
            .onChange(of: vm.selectedSegment) { _, _ in
                PegasusHaptics.selection()
            }

            if let banner = vm.realtimeBanner {
                DispatchStatusBanner(
                    systemImage: banner.systemImage,
                    title: banner.title,
                    tint: banner.tint
                )
                .padding(.horizontal)
                .padding(.bottom, LabTheme.spacingSM)
            }

            if let fleetAlert = vm.fleetAlert {
                Text(fleetAlert)
                    .font(.caption)
                    .foregroundStyle(.orange)
                    .padding(.horizontal)
                    .padding(.bottom, LabTheme.spacingSM)
            }

            switch vm.selectedSegment {
            case 0:
                ordersSegment(preview: preview)
            case 1:
                driversSegment(preview: preview)
            case 2:
                supplySegment
            default:
                locksSegment
            }
        }
    }

    @ViewBuilder
    private func ordersSegment(preview: DispatchPreview) -> some View {
        @Bindable var vm = vm
        DispatchOrderList(
            preview: preview,
            fleetVehicles: vm.fleetVehicles,
            vehicleReasons: $vm.vehicleReasons,
            vehicleNotes: $vm.vehicleNotes,
            mutatingFleetVehicleId: vm.mutatingFleetVehicleId,
            dispatchMode: $vm.dispatchMode,
            selectedDriverId: $vm.selectedDriverId,
            selectedOrderIds: $vm.selectedOrderIds,
            executing: vm.executing,
            onManualDispatch: {
                PegasusHaptics.medium()
                Task { await vm.runManualDispatch(forceCapacity: false) }
            },
            onSmartDispatch: {
                PegasusHaptics.light()
                vm.showSmartConfirm = true
            },
            onProposeDate: { orderId in
                PegasusHaptics.light()
                vm.proposeTarget = orderId
                vm.proposeDate = Date()
                vm.opsReasonInput = ""
            },
            onReject: { orderId in
                PegasusHaptics.warning()
                vm.rejectRoute = DispatchOrderDetailRoute(id: orderId)
                vm.opsReasonInput = ""
            },
            onOrderDoubleTap: { orderId in
                PegasusHaptics.light()
                vm.detailRoute = DispatchOrderDetailRoute(id: orderId)
            },
            onMarkVehicleUnavailable: { vehicle in
                let reason = vm.vehicleReasons[vehicle.vehicleId] ?? vehicle.unavailableReason ?? "MANUAL_HOLD"
                let note = vm.vehicleNotes[vehicle.vehicleId] ?? vehicle.unavailableNote ?? ""
                Task {
                    await vm.updateFleetVehicle(
                        vehicle,
                        isActive: false,
                        reason: reason,
                        note: reason == VehicleUnavailableReasonOption.other.rawValue ? note : nil
                    )
                }
            },
            onRestoreVehicle: { vehicle in
                Task { await vm.updateFleetVehicle(vehicle, isActive: true) }
            }
        )
    }

    @ViewBuilder
    private func driversSegment(preview: DispatchPreview) -> some View {
        DispatchDriverList(
            availableDrivers: preview.availableDrivers,
            unavailableDrivers: preview.unavailableDrivers
        )
    }

    @ViewBuilder
    private var supplySegment: some View {
        @Bindable var vm = vm
        if vm.supplyRequests.isEmpty {
            ContentUnavailableView("No Supply Requests", systemImage: "shippingbox", description: Text("mobile_warehouse.ui.no_active_supply_requests"))
        } else {
            ResponsiveGridContentWrapper {
                ForEach(vm.supplyRequests) { request in
                    HStack {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(String(request.requestId.prefix(8)))
                                .font(.headline)
                            Text(L10n.format("mobile_warehouse.ui.state_priority_totalvolumevu_vu", "\(request.state)", "\(request.priority)", "\(Int(request.totalVolumeVu))"))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        WarehouseStatusBadge(text: request.state)
                    }
                    .accessibilityElement(children: .combine)
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        if vm.cancellableSupplyStates.contains(request.state) {
                            Button("common.action.cancel", role: .destructive) {
                                vm.requestPendingCancellation = request
                            }
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var locksSegment: some View {
        @Bindable var vm = vm
        if vm.dispatchLocks.isEmpty {
            ContentUnavailableView("No Dispatch Locks", systemImage: "lock.open", description: Text("mobile_warehouse.ui.dispatch_is_currently_unlocked"))
        } else {
            ResponsiveGridContentWrapper {
                ForEach(vm.dispatchLocks) { lock in
                    HStack {
                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                            Text(lock.lockType)
                                .font(.headline)
                            Text(vm.dispatchLockSubtitle(lock))
                                .font(.subheadline)
                                .foregroundStyle(.secondary)
                        }
                        Spacer()
                        WarehouseStatusBadge(text: vm.dispatchLockScope(lock))
                    }
                    .accessibilityElement(children: .combine)
                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                        Button("mobile_warehouse.ui.release", role: .destructive) {
                            vm.lockPendingRelease = lock
                        }
                    }
                }
            }
        }
    }
}

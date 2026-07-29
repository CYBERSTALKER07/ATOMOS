import SwiftUI

private let dispatchTetrisBuffer = 0.95

private enum DispatchAssignmentMode: String, CaseIterable, Identifiable {
    case smart
    case manual

    var id: String { rawValue }
    var label: String {
        switch self {
        case .smart: "Smart fleet"
        case .manual: "Manual truck"
        }
    }
}

struct DispatchView: View {
    @Environment(WarehouseRealtimeHub.self) private var realtimeHub
    @Environment(\.scenePhase) private var scenePhase
    @State private var preview: DispatchPreview?
    @State private var fleetVehicles: [Vehicle] = []
    @State private var vehicleReasons: [String: String] = [:]
    @State private var vehicleNotes: [String: String] = [:]
    @State private var mutatingFleetVehicleId: String?
    @State private var fleetAlert: String?
    @State private var supplyRequests: [WarehouseSupplyRequest] = []
    @State private var dispatchLocks: [WarehouseDispatchLock] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0
    @State private var realtimeClient = WarehouseRealtimeClient()
    @State private var realtimeStatus: WarehouseRealtimeStatus = .idle
    @State private var showCreateSupplyRequest = false
    @State private var showAcquireDispatchLock = false
    @State private var requestPendingCancellation: WarehouseSupplyRequest?
    @State private var lockPendingRelease: WarehouseDispatchLock?
    @State private var actionAlert: DispatchActionAlert?
    @State private var executing = false
    @State private var selectedDriverId = ""
    @State private var selectedOrderIds: Set<String> = []
    @State private var showCapacityDialog = false
    @State private var capacityDialogAutoMode = false
    @State private var showSmartConfirm = false
    @State private var dispatchMode: DispatchAssignmentMode = .smart
    @State private var capacityWarnings: [DispatchCapacityWarning] = []
    @State private var proposeTarget: String?
    @State private var rejectRoute: DispatchOrderDetailRoute?
    @State private var opsReasonInput = ""
    @State private var proposeDate = Date()
    @State private var detailRoute: DispatchOrderDetailRoute?
    @State private var handoffEvents: [WarehousePulseEvent] = []
    @State private var handoffLoading = true

    private var capacitySuggestedUnselect: [String] {
        Array(Set(capacityWarnings.flatMap(\.suggestedUnselectOrderIds)))
    }

    var body: some View {
        NavigationStack {
            dispatchDialogs
        }
    }

    private var capacityExceededMessage: String {
        if capacityDialogAutoMode {
            return "Smart dispatch cannot fit all orders on available trucks."
        }
        return capacityWarnings.map { warning in
            var lines = [String(format: "%.1f VU loaded / %.1f VU effective max", warning.loadedVu, warning.effectiveMaxVu)]
            if !warning.suggestedUnselectOrderIds.isEmpty {
                lines.append("Suggested unselect: \(warning.suggestedUnselectOrderIds.map { String($0.prefix(8)) }.joined(separator: ", "))")
            }
            if !warning.suggestedDeferOrderIds.isEmpty {
                lines.append("Suggested defer: \(warning.suggestedDeferOrderIds.map { String($0.prefix(8)) }.joined(separator: ", "))")
            }
            return lines.joined(separator: "\n")
        }.joined(separator: "\n\n")
    }

    @ViewBuilder
    private var dispatchBody: some View {
        Group {
            if loading && preview == nil {
                WarehouseLoadingView(
                    title: "Loading dispatch",
                    message: "Fetching orders, drivers, supply requests, and locks."
                )
            } else if let error, preview == nil {
                WarehouseErrorView(message: error) { load() }
            } else if let preview {
                dispatchContent(preview: preview)
            }
        }
        .background(LabTheme.background)
        .navigationTitle("Dispatch")
        .navigationDestination(item: $detailRoute) { route in
            OrderDetailView(orderId: route.id)
        }
        .toolbar {
            ToolbarItemGroup(placement: .topBarTrailing) {
                if selectedSegment == 2 {
                    Button("New Request", systemImage: "plus") { showCreateSupplyRequest = true }
                }
                if selectedSegment == 3 && !hasActiveManualDispatchLock {
                    Button("Lock", systemImage: "lock.fill") { showAcquireDispatchLock = true }
                }
                Button("Refresh", systemImage: "arrow.clockwise") { load() }
            }
        }
        .task {
            load()
            connectRealtime()
        }
        .refreshable { load(silent: false) }
        .silentRealtimeRefresh(refreshEpoch: realtimeHub.refreshEpoch, reconnectEpoch: realtimeHub.reconnectEpoch) { silent in
            load(silent: silent)
        }
        .onDisappear { realtimeClient.disconnect() }
        .onChange(of: scenePhase) { phase in
            switch phase {
            case .active:
                connectRealtime()
            case .inactive, .background:
                realtimeClient.disconnect()
            @unknown default:
                break
            }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            if executing {
                executing = false
                actionAlert = DispatchActionAlert(
                    title: "Connection restored",
                    message: "Verify dispatch status before retrying."
                )
            }
        }
    }

    @ViewBuilder
    private var dispatchDialogs: some View {
        dispatchBody
            .confirmationDialog("Capacity exceeded", isPresented: $showCapacityDialog, titleVisibility: .visible) {
                Button("Force dispatch", role: .destructive) {
                    Task {
                        if capacityDialogAutoMode {
                            await runSmartDispatch(forceCapacity: true)
                        } else {
                            await runManualDispatch(forceCapacity: true)
                        }
                    }
                }
                if capacityDialogAutoMode {
                    Button("Accept partial") {
                        Task { await runSmartDispatch(acceptPartial: true) }
                    }
                }
                if !capacityDialogAutoMode && !capacitySuggestedUnselect.isEmpty {
                    Button("Apply suggestion") {
                        selectedOrderIds.subtract(capacitySuggestedUnselect)
                        showCapacityDialog = false
                    }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text(capacityExceededMessage)
            }
            .confirmationDialog("Run smart dispatch?", isPresented: $showSmartConfirm, titleVisibility: .visible) {
                Button("Smart Dispatch") {
                    Task { await runSmartDispatch() }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("Assign orders using the optimizer across available trucks.")
            }
            .sheet(isPresented: $showCreateSupplyRequest) {
                CreateSupplyRequestSheet { form in
                    Task { await createSupplyRequest(form: form) }
                }
            }
            .alert(item: $actionAlert) { alert in
                Alert(title: Text(alert.title), message: Text(alert.message), dismissButton: .default(Text("OK")))
            }
            .alert(
                "Cancel Supply Request?",
                isPresented: Binding(
                    get: { requestPendingCancellation != nil },
                    set: { if !$0 { requestPendingCancellation = nil } }
                ),
                presenting: requestPendingCancellation
            ) { request in
                Button("Keep", role: .cancel) {
                    requestPendingCancellation = nil
                }
                Button("Cancel Request", role: .destructive) {
                    Task { await cancelSupplyRequest(request) }
                }
            } message: { request in
                Text("Cancel request \(request.requestId.prefix(8))? This keeps the warehouse and factory clients in sync.")
            }
            .alert(
                "Release Dispatch Lock?",
                isPresented: Binding(
                    get: { lockPendingRelease != nil },
                    set: { if !$0 { lockPendingRelease = nil } }
                ),
                presenting: lockPendingRelease
            ) { lock in
                Button("Keep", role: .cancel) {
                    lockPendingRelease = nil
                }
                Button("Release", role: .destructive) {
                    Task { await releaseDispatchLock(lock) }
                }
            } message: { lock in
                Text("Release \(lock.lockType) for this warehouse scope?")
            }
            .confirmationDialog("Lock dispatch for manual override?", isPresented: $showAcquireDispatchLock, titleVisibility: .visible) {
                Button("Acquire MANUAL_DISPATCH") {
                    Task { await acquireDispatchLock() }
                }
                Button("Cancel", role: .cancel) {}
            } message: {
                Text("This freezes auto-dispatch changes until the lock is released.")
            }
            .sheet(item: Binding(
                get: { proposeTarget.map { DispatchOrderDetailRoute(id: $0) } },
                set: { proposeTarget = $0?.id },
            )) { target in
                NavigationStack {
                    Form {
                        DatePicker("Proposed delivery date", selection: $proposeDate, displayedComponents: .date)
                        Section {
                            TextField("Reason", text: $opsReasonInput, axis: .vertical)
                                .lineLimit(3...5)
                        } footer: {
                            Text("The retailer is notified and can accept or reject the new date.")
                        }
                    }
                    .navigationTitle("Propose new date")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Cancel") { proposeTarget = nil }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Send") { Task { await submitDispatchPropose(orderId: target.id) } }
                                .disabled(opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        }
                    }
                }
                .presentationDetents([.medium])
            }
            .sheet(item: $rejectRoute) { target in
                NavigationStack {
                    Form {
                        Section {
                            TextField("Reason", text: $opsReasonInput, axis: .vertical)
                                .lineLimit(3...5)
                        } footer: {
                            Text("Cancels the order and notifies the retailer.")
                        }
                    }
                    .navigationTitle("Cancel order")
                    .navigationBarTitleDisplayMode(.inline)
                    .toolbar {
                        ToolbarItem(placement: .cancellationAction) {
                            Button("Back") { rejectRoute = nil }
                        }
                        ToolbarItem(placement: .confirmationAction) {
                            Button("Cancel order", role: .destructive) {
                                Task { await submitDispatchReject(orderId: target.id) }
                            }
                            .disabled(opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                        }
                    }
                }
                .presentationDetents([.medium])
            }
    }

    @ViewBuilder
    private func dispatchContent(preview: DispatchPreview) -> some View {
        VStack(spacing: 0) {
            FleetLiveMapSection(mapHeight: 240, showsExpand: false)
                .padding(.horizontal)
                .padding(.top, LabTheme.spacingSM)

            HandoffTimelineSection(events: handoffEvents, loading: handoffLoading)
                .padding(.horizontal)
                .padding(.bottom, LabTheme.spacingSM)

            Picker("View", selection: $selectedSegment) {
                Text("Orders (\(preview.undispatchedOrders.count))").tag(0)
                Text("Drivers (\(preview.availableDrivers.count + preview.unavailableDrivers.count))").tag(1)
                Text("Supply (\(supplyRequests.count))").tag(2)
                Text("Locks (\(dispatchLocks.count))").tag(3)
            }
            .pickerStyle(.segmented)
            .padding()

            if let banner = realtimeBanner {
                DispatchStatusBanner(
                    systemImage: banner.systemImage,
                    title: banner.title,
                    tint: banner.tint
                )
                .padding(.horizontal)
                .padding(.bottom, LabTheme.spacingSM)
            }

            if let fleetAlert {
                Text(fleetAlert)
                    .font(.caption)
                    .foregroundStyle(.orange)
                    .padding(.horizontal)
                    .padding(.bottom, LabTheme.spacingSM)
            }

            switch selectedSegment {
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
        DispatchOrderList(
            preview: preview,
            fleetVehicles: fleetVehicles,
            vehicleReasons: $vehicleReasons,
            vehicleNotes: $vehicleNotes,
            mutatingFleetVehicleId: mutatingFleetVehicleId,
            dispatchMode: $dispatchMode,
            selectedDriverId: $selectedDriverId,
            selectedOrderIds: $selectedOrderIds,
            executing: executing,
            onManualDispatch: {
                Task { await runManualDispatch(forceCapacity: false) }
            },
            onSmartDispatch: {
                showSmartConfirm = true
            },
            onProposeDate: { orderId in
                proposeTarget = orderId
                proposeDate = Date()
                opsReasonInput = ""
            },
            onReject: { orderId in
                rejectRoute = DispatchOrderDetailRoute(id: orderId)
                opsReasonInput = ""
            },
            onOrderDoubleTap: { orderId in
                detailRoute = DispatchOrderDetailRoute(id: orderId)
            },
            onMarkVehicleUnavailable: { vehicle in
                let reason = vehicleReasons[vehicle.vehicleId] ?? vehicle.unavailableReason ?? "MANUAL_HOLD"
                let note = vehicleNotes[vehicle.vehicleId] ?? vehicle.unavailableNote ?? ""
                Task {
                    await updateFleetVehicle(
                        vehicle,
                        isActive: false,
                        reason: reason,
                        note: reason == VehicleUnavailableReasonOption.other.rawValue ? note : nil
                    )
                }
            },
            onRestoreVehicle: { vehicle in
                Task { await updateFleetVehicle(vehicle, isActive: true) }
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
        if supplyRequests.isEmpty {
            ContentUnavailableView("No Supply Requests", systemImage: "shippingbox", description: Text("No active supply requests"))
        } else {
            ResponsiveGridContentWrapper {
                ForEach(supplyRequests) { request in
                HStack {
                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                        Text(String(request.requestId.prefix(8)))
                            .font(.headline)
                        Text("\(request.state) · \(request.priority) · \(Int(request.totalVolumeVu)) VU")
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    WarehouseStatusBadge(text: request.state)
                }
                .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                    if cancellableSupplyStates.contains(request.state) {
                        Button("Cancel", role: .destructive) {
                            requestPendingCancellation = request
                        }
                    }
                }
            }
        }
    }

    @ViewBuilder
    private var locksSegment: some View {
        if dispatchLocks.isEmpty {
            ContentUnavailableView("No Dispatch Locks", systemImage: "lock.open", description: Text("Dispatch is currently unlocked"))
        } else {
            ResponsiveGridContentWrapper {
                ForEach(dispatchLocks) { lock in
                HStack {
                    VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                        Text(lock.lockType)
                            .font(.headline)
                        Text(dispatchLockSubtitle(lock))
                            .font(.subheadline)
                            .foregroundStyle(.secondary)
                    }
                    Spacer()
                    WarehouseStatusBadge(text: dispatchLockScope(lock))
                }
                .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                    Button("Release", role: .destructive) {
                        lockPendingRelease = lock
                    }
                }
            }
        }
    }

    private func driverSubtitle(_ driver: AvailableDriver, unavailableDetail: Bool) -> String {
        var parts: [String] = []
        if !driver.vehicleLabel.isEmpty {
            parts.append(driver.vehicleLabel)
        } else if driver.phone.isEmpty {
            parts.append(unavailableDetail ? "Assigned vehicle unavailable" : "Assigned vehicle")
        } else {
            parts.append(driver.phone)
        }
        if !unavailableDetail, let free = driver.freeVolumeVu, free > 0 {
            parts.append(String(format: "%.1f VU free", free))
        }
        return parts.joined(separator: " · ")
    }

    private func dispatchLockSubtitle(_ lock: WarehouseDispatchLock) -> String {
        if lock.lockedBy.isEmpty {
            return String(lock.lockId.prefix(8))
        }
        return String(lock.lockedBy.prefix(8))
    }

    private func dispatchLockScope(_ lock: WarehouseDispatchLock) -> String {
        if lock.warehouseId.isEmpty {
            return "Global"
        }
        return String(lock.warehouseId.prefix(8))
    }

    private func connectRealtime() {
        realtimeClient.connect(onStateChange: { status in
            realtimeStatus = status
        }, onEvent: { event in
            switch event.type {
            case "SUPPLY_REQUEST_UPDATE":
                Task { await reloadSupplyRequests() }
            case "DRIVER_AVAILABILITY_CHANGED", "VEHICLE_AVAILABILITY_CHANGED":
                Task {
                    fleetAlert = "Fleet availability updated"
                    await reloadFleetVehicles()
                    await reloadDispatchPreview()
                }
            case "DISPATCH_LOCK_CHANGE", "DISPATCH_COMMITTED", "DISPATCH_PLAN_UPDATED":
                Task {
                    await reloadDispatchLocks()
                    await reloadDispatchPreview()
                }
            default:
                break
            }
        }, onReconnect: {
            Task {
                await WarehouseSessionReconcile.run()
                realtimeHub.bumpReconnect()
            }
        })
    }

    private func load(silent: Bool = false) {
        if !silent { loading = true }
        error = nil
        Task {
            do {
                async let previewData = WarehouseService.dispatchPreview()
                async let supplyData = WarehouseService.supplyRequests()
                async let lockData = WarehouseService.dispatchLocks()
                async let fleetData = WarehouseService.vehicles()
                preview = try await previewData
                supplyRequests = try await supplyData
                dispatchLocks = try await lockData
                fleetVehicles = try await fleetData.vehicles
                handoffLoading = true
                if let pulse = try? await WarehouseService.pulse() {
                    handoffEvents = HandoffPulseSupport.filter(pulse.events)
                } else {
                    handoffEvents = []
                }
                handoffLoading = false
            } catch {
                if !silent { self.error = describe(error, fallback: "Failed to load dispatch data") }
                handoffEvents = []
                handoffLoading = false
            }
            if !silent { loading = false }
        }
    }

    private var cancellableSupplyStates: Set<String> {
        ["DRAFT", "SUBMITTED", "ACKNOWLEDGED"]
    }

    private var hasActiveManualDispatchLock: Bool {
        dispatchLocks.contains { $0.lockType == "MANUAL_DISPATCH" }
    }

    private var realtimeBanner: (systemImage: String, title: String, tint: Color)? {
        switch realtimeStatus {
        case .idle, .live:
            return nil
        case .connecting:
            return ("dot.radiowaves.left.and.right", "Connecting live warehouse updates…", .blue)
        case .reconnecting:
            return ("arrow.triangle.2.circlepath", "Live updates reconnecting. Current data may be stale.", .orange)
        case .offline:
            return ("wifi.slash", "Offline. Live updates are paused until the connection returns.", .red)
        }
    }

    private func driverPickerLabel(_ driver: AvailableDriver) -> String {
        if (driver.freeVolumeVu ?? 0) > 0 {
            return "\(driver.name) · \(Int(driver.freeVolumeVu ?? 0)) VU free"
        }
        return "\(driver.name) · \(Int(driver.maxVolumeVu)) VU max"
    }

    private func createSupplyRequest(form: SupplyRequestFormData) async {
        do {
            let response = try await WarehouseService.createSupplyRequest(form: form)
            showCreateSupplyRequest = false
            actionAlert = DispatchActionAlert(title: "Supply Request Submitted", message: "Request \(response.requestId.prefix(8)) is now \(response.state).")
            await reloadSupplyRequests()
        } catch {
            actionAlert = DispatchActionAlert(title: "Supply Request Failed", message: describe(error))
        }
    }

    private func cancelSupplyRequest(_ request: WarehouseSupplyRequest) async {
        defer { requestPendingCancellation = nil }
        do {
            let response = try await WarehouseService.cancelSupplyRequest(id: request.requestId)
            actionAlert = DispatchActionAlert(title: "Supply Request Cancelled", message: "Request \(response.requestId.prefix(8)) moved to \(response.state).")
            await reloadSupplyRequests()
        } catch {
            actionAlert = DispatchActionAlert(title: "Cancellation Failed", message: describe(error))
        }
    }

    private func runManualDispatch(forceCapacity: Bool) async {
        guard !executing, !selectedDriverId.isEmpty, !selectedOrderIds.isEmpty else { return }
        executing = true
        defer { executing = false }
        do {
            let sortedOrderIds = selectedOrderIds.sorted()
            let orderIdsJson = sortedOrderIds.map { "\"\($0)\"" }.joined(separator: ",")
            let routeFingerprint = """
            {"mode":"MANUAL","force_capacity":\(forceCapacity),"routes":[{"driver_id":"\(selectedDriverId)","order_ids":[\(orderIdsJson)]}]}
            """
            let idempotencyKey = WarehouseIdempotency.dispatch(
                actorId: selectedDriverId,
                routeFingerprint: routeFingerprint
            )
            let result = try await WarehouseService.executeDispatch(
                body: DispatchExecuteRequest(
                    mode: "MANUAL",
                    forceCapacity: forceCapacity,
                    acceptPartial: nil,
                    orderIds: sortedOrderIds,
                    planFingerprint: nil,
                    routes: [
                        DispatchExecuteRouteRequest(
                            driverId: selectedDriverId,
                            orderIds: sortedOrderIds,
                        ),
                    ],
                ),
                idempotencyKey: idempotencyKey
            )
            switch result.status {
            case "plan_stale":
                actionAlert = DispatchActionAlert(
                    title: "Plan stale",
                    message: "Refresh preview and try smart dispatch again."
                )
                await reloadDispatchPreview()
            case "capacity_exceeded":
                capacityWarnings = result.capacityWarnings
                capacityDialogAutoMode = false
                showCapacityDialog = true
            case "dispatched":
                actionAlert = DispatchActionAlert(
                    title: "Dispatch Committed",
                    message: "Assigned \(result.ordersAssigned) order(s). Payloader loading gate is active."
                )
                selectedOrderIds = []
                await reloadDispatchPreview()
            default:
                let warning = result.warnings.first ?? "Dispatch did not commit."
                actionAlert = DispatchActionAlert(title: "Dispatch Incomplete", message: warning)
            }
        } catch {
            actionAlert = DispatchActionAlert(title: "Dispatch Failed", message: describe(error))
        }
    }

    private func runSmartDispatch(forceCapacity: Bool = false, acceptPartial: Bool = false) async {
        guard !executing, let preview else { return }
        let orderIds = selectedOrderIds.isEmpty
            ? preview.undispatchedOrders.map(\.orderId)
            : selectedOrderIds.sorted()
        guard !orderIds.isEmpty else { return }
        executing = true
        defer { executing = false }
        do {
            let orderIdsJson = orderIds.map { "\"\($0)\"" }.joined(separator: ",")
            let routeFingerprint = """
            {"mode":"AUTO","order_ids":[\(orderIdsJson)],"force_capacity":\(forceCapacity),"accept_partial":\(acceptPartial)}
            """
            let idempotencyKey = WarehouseIdempotency.dispatch(
                actorId: "smart-dispatch",
                routeFingerprint: routeFingerprint
            )
            let result = try await WarehouseService.executeDispatch(
                body: DispatchExecuteRequest(
                    mode: "AUTO",
                    forceCapacity: forceCapacity,
                    acceptPartial: acceptPartial ? true : nil,
                    orderIds: orderIds,
                    planFingerprint: preview.planFingerprint,
                    routes: nil,
                ),
                idempotencyKey: idempotencyKey
            )
            switch result.status {
            case "plan_stale":
                actionAlert = DispatchActionAlert(
                    title: "Plan stale",
                    message: "Refresh preview and try smart dispatch again."
                )
                await reloadDispatchPreview()
            case "capacity_exceeded":
                capacityWarnings = result.capacityWarnings
                capacityDialogAutoMode = true
                showCapacityDialog = true
            case "dispatched":
                var message = "Assigned \(result.ordersAssigned) order(s)."
                if !result.orphanOrderIds.isEmpty {
                    message += " \(result.orphanOrderIds.count) order(s) could not be assigned."
                }
                actionAlert = DispatchActionAlert(title: "Smart Dispatch Committed", message: message)
                selectedOrderIds = []
                await reloadDispatchPreview()
            default:
                let warning = result.warnings.first ?? "Smart dispatch did not commit."
                actionAlert = DispatchActionAlert(title: "Smart Dispatch Incomplete", message: warning)
            }
        } catch {
            actionAlert = DispatchActionAlert(title: "Smart Dispatch Failed", message: describe(error))
        }
    }

    private func acquireDispatchLock() async {
        do {
            let response = try await WarehouseService.acquireDispatchLock()
            actionAlert = DispatchActionAlert(title: "Dispatch Locked", message: "\(response.lockType) is now active for this warehouse scope.")
            await reloadDispatchLocks()
            await reloadDispatchPreview()
        } catch {
            actionAlert = DispatchActionAlert(title: "Lock Failed", message: describe(error))
        }
    }

    private func releaseDispatchLock(_ lock: WarehouseDispatchLock) async {
        defer { lockPendingRelease = nil }
        do {
            let response = try await WarehouseService.releaseDispatchLock(lockId: lock.lockId)
            actionAlert = DispatchActionAlert(title: "Dispatch Lock Released", message: "Lock \(response.lockId.prefix(8)) is now \(response.status).")
            await reloadDispatchLocks()
            await reloadDispatchPreview()
        } catch {
            actionAlert = DispatchActionAlert(title: "Release Failed", message: describe(error))
        }
    }

    private func reloadDispatchPreview() async {
        do {
            preview = try await WarehouseService.dispatchPreview()
        } catch {
            self.error = describe(error, fallback: "Failed to refresh dispatch preview")
        }
    }

    private func reloadSupplyRequests() async {
        do {
            supplyRequests = try await WarehouseService.supplyRequests()
        } catch {
            self.error = describe(error, fallback: "Failed to refresh supply requests")
        }
    }

    private func reloadDispatchLocks() async {
        do {
            dispatchLocks = try await WarehouseService.dispatchLocks()
        } catch {
            self.error = describe(error, fallback: "Failed to refresh dispatch locks")
        }
    }

    private func reloadFleetVehicles() async {
        do {
            let response = try await WarehouseService.vehicles()
            fleetVehicles = response.vehicles
        } catch {
            self.error = describe(error, fallback: "Failed to refresh fleet trucks")
        }
    }

    private func updateFleetVehicle(_ vehicle: Vehicle, isActive: Bool, reason: String? = nil, note: String? = nil) async {
        mutatingFleetVehicleId = vehicle.vehicleId
        fleetAlert = nil
        defer { mutatingFleetVehicleId = nil }
        do {
            _ = try await WarehouseService.updateVehicleAvailability(
                vehicleId: vehicle.vehicleId,
                isActive: isActive,
                unavailableReason: reason,
                unavailableNote: note
            )
            fleetAlert = isActive
                ? "\(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label) restored to dispatch"
                : "\(vehicle.label.isEmpty ? vehicle.licensePlate : vehicle.label) excluded from dispatch"
            await reloadFleetVehicles()
            await reloadDispatchPreview()
        } catch {
            actionAlert = DispatchActionAlert(title: "Fleet Update Failed", message: describe(error))
        }
    }

    private func describe(_ error: Error, fallback: String = "Request failed") -> String {
        if let apiError = error as? APIError, case let .httpError(code) = apiError {
            if code == 403 {
                return "Permission denied for this warehouse scope."
            }
            return "\(fallback) (HTTP \(code))"
        }
        return error.localizedDescription
    }

    private func submitDispatchPropose(orderId: String) async {
        let reason = opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else { return }
        do {
            _ = try await WarehouseOperationsService.proposeOrderDelivery(
                orderId: orderId,
                proposedDeliveryDate: dispatchIsoDeliveryDate(from: proposeDate),
                reason: reason
            )
            actionAlert = DispatchActionAlert(title: "Date proposed", message: "Retailer notified — they can accept or reject.")
            proposeTarget = nil
            opsReasonInput = ""
            load(silent: true)
        } catch {
            actionAlert = DispatchActionAlert(title: "Propose failed", message: describe(error))
        }
    }

    private func submitDispatchReject(orderId: String) async {
        let reason = opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else {
            actionAlert = DispatchActionAlert(title: "Reason required", message: "Enter a cancellation reason before confirming.")
            return
        }
        do {
            _ = try await WarehouseOperationsService.rejectOrder(orderId: orderId, reason: reason)
            actionAlert = DispatchActionAlert(title: "Order cancelled", message: "Retailer notified.")
            rejectRoute = nil
            opsReasonInput = ""
            load(silent: true)
        } catch {
            actionAlert = DispatchActionAlert(title: "Cancel failed", message: describe(error))
        }
    }
}

private struct DispatchOrderDetailRoute: Identifiable, Hashable {
    let id: String
}

private func dispatchIsoDeliveryDate(from date: Date) -> String {
    var calendar = Calendar(identifier: .gregorian)
    calendar.timeZone = TimeZone(secondsFromGMT: 5 * 3600) ?? .current
    let components = calendar.dateComponents([.year, .month, .day], from: date)
    var merged = DateComponents()
    merged.year = components.year
    merged.month = components.month
    merged.day = components.day
    merged.hour = 12
    merged.minute = 0
    merged.second = 0
    merged.timeZone = TimeZone(secondsFromGMT: 5 * 3600)
    let normalized = calendar.date(from: merged) ?? date
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    return formatter.string(from: normalized)
}

private struct DispatchActionAlert: Identifiable {
    let id = UUID()
    let title: String
    let message: String
}

private struct DispatchStatusBanner: View {
    let systemImage: String
    let title: String
    let tint: Color

    var body: some View {
        Label(title, systemImage: systemImage)
            .font(.caption.weight(.semibold))
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, LabTheme.spacingSM)
            .padding(.vertical, LabTheme.spacingXS)
            .foregroundStyle(tint)
            .background(tint.opacity(0.12), in: RoundedRectangle(cornerRadius: 12, style: .continuous))
    }
}

private struct HandoffTimelineSection: View {
    let events: [WarehousePulseEvent]
    let loading: Bool

    private var subtitle: String {
        if loading && events.isEmpty { return "Loading handoff chain…" }
        if events.isEmpty { return "No preorder → dispatch → seal events in the recent pulse window." }
        return "\(events.count) handoff event(s) in recent pulse."
    }

    var body: some View {
        VStack(alignment: .leading, spacing: LabTheme.spacingSM) {
            Text("Handoff timeline")
                .font(.headline)
            Text(subtitle)
                .font(.caption)
                .foregroundStyle(.secondary)
            ForEach(events.prefix(8)) { event in
                VStack(alignment: .leading, spacing: 4) {
                    Text(event.title)
                        .font(.subheadline.bold())
                        .lineLimit(2)
                    if let description = event.description, !description.isEmpty {
                        Text(description)
                            .font(.caption)
                            .foregroundStyle(.secondary)
                            .lineLimit(3)
                    }
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                .padding()
                .background(LabTheme.card)
                .clipShape(RoundedRectangle(cornerRadius: LabTheme.radiusMD))
            }
        }
    }
}

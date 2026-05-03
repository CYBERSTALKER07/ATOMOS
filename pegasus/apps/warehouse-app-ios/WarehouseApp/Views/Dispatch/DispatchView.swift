import SwiftUI

struct DispatchView: View {
    @Environment(\.scenePhase) private var scenePhase
    @State private var preview: DispatchPreview?
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

    var body: some View {
        NavigationStack {
            Group {
                if loading {
                    ProgressView()
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let error {
                    ContentUnavailableView {
                        Label("Error", systemImage: "exclamationmark.triangle")
                    } description: {
                        Text(error)
                    } actions: {
                        Button("Retry") { load() }
                    }
                } else if let preview {
                    VStack(spacing: 0) {
                        Picker("View", selection: $selectedSegment) {
                            Text("Orders (\(preview.undispatchedOrders.count))").tag(0)
                            Text("Drivers (\(preview.availableDrivers.count))").tag(1)
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

                        if selectedSegment == 0 {
                            if preview.undispatchedOrders.isEmpty {
                                ContentUnavailableView("All Dispatched", systemImage: "checkmark.circle", description: Text("No pending orders"))
                            } else {
                                List(preview.undispatchedOrders) { order in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(order.retailerName.isEmpty ? String(order.orderId.prefix(8)) : order.retailerName)
                                                .font(.headline)
                                            Text("\(order.totalUzs.formatted()) UZS · \(order.itemCount) items")
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                }
                                .listStyle(.insetGrouped)
                            }
                        } else if selectedSegment == 1 {
                            if preview.availableDrivers.isEmpty {
                                ContentUnavailableView("No Drivers", systemImage: "person.badge.key", description: Text("No available drivers"))
                            } else {
                                List(preview.availableDrivers) { driver in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(driver.name)
                                                .font(.headline)
                                            Text(driver.vehicleLabel.isEmpty ? "No vehicle" : driver.vehicleLabel)
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                        }
                                    }
                                }
                                .listStyle(.insetGrouped)
                            }
                        } else if selectedSegment == 2 {
                            if supplyRequests.isEmpty {
                                ContentUnavailableView("No Supply Requests", systemImage: "shippingbox", description: Text("No active supply requests"))
                            } else {
                                List(supplyRequests) { request in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(String(request.requestId.prefix(8)))
                                                .font(.headline)
                                            Text("\(request.state) · \(request.priority) · \(Int(request.totalVolumeVu)) VU")
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        Text(request.state)
                                            .font(.caption.bold())
                                            .padding(.horizontal, LabTheme.spacingSM)
                                            .padding(.vertical, LabTheme.spacingXS)
                                            .background(.quaternary, in: Capsule())
                                    }
                                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                                        if cancellableSupplyStates.contains(request.state) {
                                            Button("Cancel", role: .destructive) {
                                                requestPendingCancellation = request
                                            }
                                        }
                                    }
                                }
                                .listStyle(.insetGrouped)
                            }
                        } else {
                            if dispatchLocks.isEmpty {
                                ContentUnavailableView("No Dispatch Locks", systemImage: "lock.open", description: Text("Dispatch is currently unlocked"))
                            } else {
                                List(dispatchLocks) { lock in
                                    HStack {
                                        VStack(alignment: .leading, spacing: LabTheme.spacingXS) {
                                            Text(lock.lockType)
                                                .font(.headline)
                                            Text(lock.lockedBy.isEmpty ? String(lock.lockId.prefix(8)) : String(lock.lockedBy.prefix(8)))
                                                .font(.subheadline)
                                                .foregroundStyle(.secondary)
                                        }
                                        Spacer()
                                        Text(lock.warehouseId.isEmpty ? "Global" : String(lock.warehouseId.prefix(8)))
                                            .font(.caption.bold())
                                            .padding(.horizontal, LabTheme.spacingSM)
                                            .padding(.vertical, LabTheme.spacingXS)
                                            .background(.quaternary, in: Capsule())
                                    }
                                    .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                                        Button("Release", role: .destructive) {
                                            lockPendingRelease = lock
                                        }
                                    }
                                }
                                .listStyle(.insetGrouped)
                            }
                        }
                    }
                }
            }
            .background(LabTheme.background)
            .navigationTitle("Dispatch")
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
            .refreshable { load() }
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
            .sheet(isPresented: $showCreateSupplyRequest) {
                CreateSupplyRequestSheet { factoryId, priority, notes in
                    Task { await createSupplyRequest(factoryId: factoryId, priority: priority, notes: notes) }
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
        }
    }

    private func connectRealtime() {
        realtimeClient.connect(onStateChange: { status in
            realtimeStatus = status
        }, onEvent: { event in
            switch event.type {
            case "SUPPLY_REQUEST_UPDATE":
                Task { await reloadSupplyRequests() }
            case "DISPATCH_LOCK_CHANGE":
                Task {
                    await reloadDispatchLocks()
                    await reloadDispatchPreview()
                }
            default:
                break
            }
        })
    }

    private func load() {
        loading = true
        error = nil
        Task {
            do {
                async let previewData = WarehouseService.dispatchPreview()
                async let supplyData = WarehouseService.supplyRequests()
                async let lockData = WarehouseService.dispatchLocks()
                preview = try await previewData
                supplyRequests = try await supplyData
                dispatchLocks = try await lockData
            } catch {
                self.error = error.localizedDescription
            }
            loading = false
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

    private func createSupplyRequest(factoryId: String, priority: String, notes: String) async {
        do {
            let response = try await WarehouseService.createSupplyRequest(factoryId: factoryId, priority: priority, notes: notes)
            showCreateSupplyRequest = false
            actionAlert = DispatchActionAlert(title: "Supply Request Submitted", message: "Request \(response.requestId.prefix(8)) is now \(response.state).")
            await reloadSupplyRequests()
        } catch {
            actionAlert = DispatchActionAlert(title: "Supply Request Failed", message: error.localizedDescription)
        }
    }

    private func cancelSupplyRequest(_ request: WarehouseSupplyRequest) async {
        defer { requestPendingCancellation = nil }
        do {
            let response = try await WarehouseService.cancelSupplyRequest(id: request.requestId)
            actionAlert = DispatchActionAlert(title: "Supply Request Cancelled", message: "Request \(response.requestId.prefix(8)) moved to \(response.state).")
            await reloadSupplyRequests()
        } catch {
            actionAlert = DispatchActionAlert(title: "Cancellation Failed", message: error.localizedDescription)
        }
    }

    private func acquireDispatchLock() async {
        do {
            let response = try await WarehouseService.acquireDispatchLock()
            actionAlert = DispatchActionAlert(title: "Dispatch Locked", message: "\(response.lockType) is now active for this warehouse scope.")
            await reloadDispatchLocks()
            await reloadDispatchPreview()
        } catch {
            actionAlert = DispatchActionAlert(title: "Lock Failed", message: error.localizedDescription)
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
            actionAlert = DispatchActionAlert(title: "Release Failed", message: error.localizedDescription)
        }
    }

    private func reloadDispatchPreview() async {
        do {
            preview = try await WarehouseService.dispatchPreview()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func reloadSupplyRequests() async {
        do {
            supplyRequests = try await WarehouseService.supplyRequests()
        } catch {
            self.error = error.localizedDescription
        }
    }

    private func reloadDispatchLocks() async {
        do {
            dispatchLocks = try await WarehouseService.dispatchLocks()
        } catch {
            self.error = error.localizedDescription
        }
    }
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

private struct CreateSupplyRequestSheet: View {
    let onCreate: (String, String, String) -> Void

    @Environment(\.dismiss) private var dismiss
    @State private var factoryId = ""
    @State private var priority = "NORMAL"
    @State private var notes = ""

    var body: some View {
        NavigationStack {
            Form {
                TextField("Factory ID", text: $factoryId)
                    .textInputAutocapitalization(.never)
                    .autocorrectionDisabled()

                Picker("Priority", selection: $priority) {
                    Text("Normal").tag("NORMAL")
                    Text("Urgent").tag("URGENT")
                    Text("Critical").tag("CRITICAL")
                }
                .pickerStyle(.segmented)

                TextField("Notes", text: $notes, axis: .vertical)
                    .lineLimit(3...5)

                Text("This submits a warehouse supply request using the backend demand forecast path.")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            .navigationTitle("New Supply Request")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Submit") {
                        onCreate(factoryId, priority, notes)
                    }
                    .disabled(factoryId.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
    }
}

import SwiftUI

struct DispatchView: View {
    @State private var preview: DispatchPreview?
    @State private var supplyRequests: [WarehouseSupplyRequest] = []
    @State private var dispatchLocks: [WarehouseDispatchLock] = []
    @State private var loading = true
    @State private var error: String?
    @State private var selectedSegment = 0
    @State private var realtimeClient = WarehouseRealtimeClient()

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
                ToolbarItem(placement: .topBarTrailing) {
                    Button("Refresh", systemImage: "arrow.clockwise") { load() }
                }
            }
            .task {
                load()
                connectRealtime()
            }
            .refreshable { load() }
            .onDisappear { realtimeClient.disconnect() }
        }
    }

    private func connectRealtime() {
        realtimeClient.connect { event in
            switch event.type {
            case "SUPPLY_REQUEST_UPDATE":
                Task { await reloadSupplyRequests() }
            case "DISPATCH_LOCK_CHANGE":
                Task { await reloadDispatchLocks() }
            default:
                break
            }
        }
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

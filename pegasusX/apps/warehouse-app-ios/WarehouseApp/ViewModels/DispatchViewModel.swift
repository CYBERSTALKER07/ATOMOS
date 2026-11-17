import Foundation
import SwiftUI
import Observation

enum DispatchAssignmentMode: String, CaseIterable, Identifiable, Sendable {
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

struct DispatchOrderDetailRoute: Identifiable, Hashable, Sendable {
    let id: String
    init(id: String) {
        self.id = id
    }
}

struct DispatchActionAlert: Identifiable, Sendable {
    let id = UUID()
    let title: String
    let message: String
    init(title: String, message: String) {
        self.title = title
        self.message = message
    }
}

@Observable
@MainActor
final class DispatchViewModel {
    var preview: DispatchPreview?
    var fleetVehicles: [Vehicle] = []
    var vehicleReasons: [String: String] = [:]
    var vehicleNotes: [String: String] = [:]
    var mutatingFleetVehicleId: String?
    var fleetAlert: String?
    var supplyRequests: [WarehouseSupplyRequest] = []
    var dispatchLocks: [WarehouseDispatchLock] = []
    var loading = true
    var error: String?
    var selectedSegment = 0
    var realtimeClient = WarehouseRealtimeClient()
    var realtimeStatus: WarehouseRealtimeStatus = .idle
    var showCreateSupplyRequest = false
    var showAcquireDispatchLock = false
    var requestPendingCancellation: WarehouseSupplyRequest?
    var lockPendingRelease: WarehouseDispatchLock?
    var actionAlert: DispatchActionAlert?
    var executing = false
    var selectedDriverId = ""
    var showRescueDialog = false
    var lookupDriverId = ""
    var lookupLoading = false
    var lookupError: String? = nil
    var earlyCompleteReq: [String: String]? = nil
    var rescueAction: String? = nil
    var rescueWindowStart = ""
    var rescueWindowEnd = ""
    var rescueMutating = false

    var selectedOrderIds: Set<String> = []
    var showCapacityDialog = false
    var capacityDialogAutoMode = false
    var showSmartConfirm = false
    var dispatchMode: DispatchAssignmentMode = .smart
    var capacityWarnings: [DispatchCapacityWarning] = []
    var proposeTarget: String?
    var rejectRoute: DispatchOrderDetailRoute?
    var opsReasonInput = ""
    var proposeDate = Date()
    var detailRoute: DispatchOrderDetailRoute?
    var handoffEvents: [WarehousePulseEvent] = []
    var handoffLoading = true
    var handoffError: String?

    var showCancelSupplyRequestAlert: Bool {
        get { requestPendingCancellation != nil }
        set {
            if !newValue {
                requestPendingCancellation = nil
            }
        }
    }

    var showReleaseDispatchLockAlert: Bool {
        get { lockPendingRelease != nil }
        set {
            if !newValue {
                lockPendingRelease = nil
            }
        }
    }

    var proposeRoute: DispatchOrderDetailRoute? {
        get { proposeTarget.map { DispatchOrderDetailRoute(id: $0) } }
        set { proposeTarget = newValue?.id }
    }

    static let dispatchIsoFractionalFormatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static func dispatchIsoDeliveryDate(from date: Date) -> String {
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
        return dispatchIsoFractionalFormatter.string(from: normalized)
    }

    var capacitySuggestedUnselect: [String] {
        Array(Set(capacityWarnings.flatMap(\.suggestedUnselectOrderIds)))
    }

    var capacityExceededMessage: String {
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

    var cancellableSupplyStates: Set<String> {
        ["DRAFT", "SUBMITTED", "ACKNOWLEDGED"]
    }

    var hasActiveManualDispatchLock: Bool {
        dispatchLocks.contains { $0.lockType == "MANUAL_DISPATCH" }
    }

    var realtimeBanner: (systemImage: String, title: String, tint: Color)? {
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

    init() {}

    func connectRealtime(onReconnect: (@Sendable () -> Void)? = nil) {
        realtimeClient.connect(onStateChange: { [weak self] status in
            Task { @MainActor [weak self] in
                self?.realtimeStatus = status
            }
        }, onEvent: { [weak self] event in
            Task { @MainActor [weak self] in
                guard let self else { return }
                switch event.type {
                case "SUPPLY_REQUEST_UPDATE":
                    await self.reloadSupplyRequests()
                case "DRIVER_AVAILABILITY_CHANGED", "VEHICLE_AVAILABILITY_CHANGED":
                    self.fleetAlert = "Fleet availability updated"
                    await self.reloadFleetVehicles()
                    await self.reloadDispatchPreview()
                case "DISPATCH_LOCK_CHANGE", "DISPATCH_COMMITTED", "DISPATCH_PLAN_UPDATED":
                    await self.reloadDispatchLocks()
                    await self.reloadDispatchPreview()
                default:
                    break
                }
            }
        }, onReconnect: {
            Task {
                await WarehouseSessionReconcile.run()
                onReconnect?()
            }
        })
    }

    func disconnectRealtime() {
        realtimeClient.disconnect()
    }

    func load(silent: Bool = false) {
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
            } catch {
                if !silent { self.error = describe(error, fallback: "Failed to load dispatch data") }
            }
            handoffLoading = true
            handoffError = nil
            do {
                let pulse = try await WarehouseService.pulse()
                let result = PulseHonesty.apply(
                    ok: true,
                    incoming: HandoffPulseSupport.filter(pulse.events),
                    previous: handoffEvents
                )
                handoffEvents = result.events
                handoffError = result.error
            } catch {
                let result = PulseHonesty.apply(ok: false, incoming: nil, previous: handoffEvents)
                handoffEvents = result.events
                handoffError = result.error
            }
            handoffLoading = false
            if !silent { loading = false }
        }
    }

    func driverPickerLabel(_ driver: AvailableDriver) -> String {
        if (driver.freeVolumeVu ?? 0) > 0 {
            return "\(driver.name) · \(Int(driver.freeVolumeVu ?? 0)) VU free"
        }
        return "\(driver.name) · \(Int(driver.maxVolumeVu)) VU max"
    }

    func driverSubtitle(_ driver: AvailableDriver, unavailableDetail: Bool) -> String {
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

    func dispatchLockSubtitle(_ lock: WarehouseDispatchLock) -> String {
        if lock.lockedBy.isEmpty {
            return String(lock.lockId.prefix(8))
        }
        return String(lock.lockedBy.prefix(8))
    }

    func dispatchLockScope(_ lock: WarehouseDispatchLock) -> String {
        if lock.warehouseId.isEmpty {
            return "Global"
        }
        return String(lock.warehouseId.prefix(8))
    }

    func createSupplyRequest(form: SupplyRequestFormData) async {
        do {
            let response = try await WarehouseService.createSupplyRequest(form: form)
            showCreateSupplyRequest = false
            actionAlert = DispatchActionAlert(title: "Supply Request Submitted", message: "Request \(response.requestId.prefix(8)) is now \(response.state).")
            await reloadSupplyRequests()
        } catch {
            actionAlert = DispatchActionAlert(title: "Supply Request Failed", message: describe(error))
        }
    }

    func cancelSupplyRequest(_ request: WarehouseSupplyRequest) async {
        defer { requestPendingCancellation = nil }
        do {
            let response = try await WarehouseService.cancelSupplyRequest(id: request.requestId)
            actionAlert = DispatchActionAlert(title: "Supply Request Cancelled", message: "Request \(response.requestId.prefix(8)) moved to \(response.state).")
            await reloadSupplyRequests()
        } catch {
            actionAlert = DispatchActionAlert(title: "Cancellation Failed", message: describe(error))
        }
    }

    func runManualDispatch(forceCapacity: Bool) async {
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

    func runSmartDispatch(forceCapacity: Bool = false, acceptPartial: Bool = false) async {
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

    func acquireDispatchLock() async {
        do {
            let response = try await WarehouseService.acquireDispatchLock()
            actionAlert = DispatchActionAlert(title: "Dispatch Locked", message: "\(response.lockType) is now active for this warehouse scope.")
            await reloadDispatchLocks()
            await reloadDispatchPreview()
        } catch {
            actionAlert = DispatchActionAlert(title: "Lock Failed", message: describe(error))
        }
    }

    func releaseDispatchLock(_ lock: WarehouseDispatchLock) async {
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

    func reloadDispatchPreview() async {
        do {
            preview = try await WarehouseService.dispatchPreview()
        } catch {
            self.error = describe(error, fallback: "Failed to refresh dispatch preview")
        }
    }

    func reloadSupplyRequests() async {
        do {
            supplyRequests = try await WarehouseService.supplyRequests()
        } catch {
            self.error = describe(error, fallback: "Failed to refresh supply requests")
        }
    }

    func reloadDispatchLocks() async {
        do {
            dispatchLocks = try await WarehouseService.dispatchLocks()
        } catch {
            self.error = describe(error, fallback: "Failed to refresh dispatch locks")
        }
    }

    func reloadFleetVehicles() async {
        do {
            let response = try await WarehouseService.vehicles()
            fleetVehicles = response.vehicles
        } catch {
            self.error = describe(error, fallback: "Failed to refresh fleet trucks")
        }
    }

    func updateFleetVehicle(_ vehicle: Vehicle, isActive: Bool, reason: String? = nil, note: String? = nil) async {
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

    func describe(_ error: Error, fallback: String = "Request failed") -> String {
        if let apiError = error as? APIError, case let .httpError(code) = apiError {
            if code == 403 {
                return "Permission denied for this warehouse scope."
            }
            return "\(fallback) (HTTP \(code))"
        }
        return error.localizedDescription
    }

    func submitDispatchPropose(orderId: String) async {
        let reason = opsReasonInput.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else { return }
        do {
            _ = try await WarehouseOperationsService.proposeOrderDelivery(
                orderId: orderId,
                proposedDeliveryDate: Self.dispatchIsoDeliveryDate(from: proposeDate),
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

    func submitDispatchReject(orderId: String) async {
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

    func findRescueDriver() async {
        let driverId = lookupDriverId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !driverId.isEmpty, !lookupLoading else { return }
        lookupLoading = true
        lookupError = nil
        defer { lookupLoading = false }
        do {
            let res = try await WarehouseOperationsService.getEarlyCompleteRequest(driverId: driverId)
            earlyCompleteReq = res
        } catch {
            lookupError = error.localizedDescription
        }
    }

    func confirmRescue(action: String) async {
        rescueMutating = true
        defer { rescueMutating = false }
        do {
            let s = rescueWindowStart.isEmpty ? nil : rescueWindowStart
            let e = rescueWindowEnd.isEmpty ? nil : rescueWindowEnd
            _ = try await WarehouseOperationsService.approveEarlyComplete(driverId: lookupDriverId, action: action, newWindowStart: s, newWindowEnd: e)
            showRescueDialog = false
            earlyCompleteReq = nil
            rescueAction = nil
            load()
        } catch {
            lookupError = error.localizedDescription
        }
    }

    func resetRescue() {
        showRescueDialog = false
        earlyCompleteReq = nil
        rescueAction = nil
        lookupDriverId = ""
        lookupError = nil
        rescueWindowStart = ""
        rescueWindowEnd = ""
    }
}

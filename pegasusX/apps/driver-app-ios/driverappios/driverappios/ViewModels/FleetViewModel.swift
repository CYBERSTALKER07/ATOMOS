//
//  FleetViewModel.swift
//  driverappios
//

import CoreLocation
import Combine
import SwiftUI

@Observable
@MainActor
final class FleetViewModel: NSObject, CLLocationManagerDelegate {

    // MARK: - Published State

    var orders: [Order] = []
    var missions: [Mission] = []          // legacy compat
    var location: CLLocationCoordinate2D?
    var course: CLLocationDirection?
    var speed: CLLocationSpeed?
    var activeMission: Mission?
    
    // Telemetry Sync Trigger
    var latestTransmitLocation: CLLocation?
    private var lastSentTime: Date?

    var activeOrder: Order?
    var completedIds: Set<String> = []
    var completedMissions: [Mission] = []
    var completedOrders: [Order] = []
    var showScanner = false
    var showCorrection = false
    var showOfflineVerifier = false
    var showMarkerSheet = false
    var isTelemetryLive = false
    var locationTrail: [CLLocationCoordinate2D] = []
    var plannedRouteCoordinates: [CLLocationCoordinate2D] = []
    var routeGeometryCoordinates: [CLLocationCoordinate2D] = []
    var routeSteps: [RouteStep] = []
    var navigationCue: NavigationCue?
    var displayLocation: CLLocationCoordinate2D?
    var displayBearing: CLLocationDirection = 0
    var displaySpeedMps: CLLocationSpeed = 0
    var lastRealtimeRefreshAt: Date?
    var gpsError: String?
    var isLoadingMissions = false
    var isTransitActive = false
    var truckStatus: String = "AVAILABLE"
    var isReturning = false

    /// Orders that have already been auto-transitioned to ARRIVED (one-shot guard)
    private var arrivedIds: Set<String> = []
    private var navigationStepIndex = 0
    private let navigationVoiceAnnouncer = NavigationVoiceAnnouncer()
    private let routeDeviationTracker = RouteDeviationTracker()
    private var isReroutingRoute = false
    private let maxTrailPoints = 60
    private let locationInterpolator = LocationInterpolator()
    private var interpolationTask: Task<Void, Never>?

    // MARK: - Services

    private let fleetService: FleetServiceProtocol
    private let locationManager = CLLocationManager()

    // MARK: - Computed

    var driverId: String { TokenStore.shared.userId ?? "DRV-UNKNOWN" }
    var truckId: String { TokenStore.shared.vehicleId ?? "—" }
    var driverName: String { TokenStore.shared.driverName ?? "Driver" }
    var licensePlate: String { TokenStore.shared.licensePlate ?? "—" }
    var vehicleClass: String { TokenStore.shared.vehicleClass ?? "—" }
    var maxVolumeVU: Double { TokenStore.shared.maxVolumeVU }

    static var warehouseCenter: CLLocationCoordinate2D {
        let lat = TokenStore.shared.warehouseLat != 0 ? TokenStore.shared.warehouseLat : 41.2995
        let lng = TokenStore.shared.warehouseLng != 0 ? TokenStore.shared.warehouseLng : 69.2401
        return CLLocationCoordinate2D(latitude: lat, longitude: lng)
    }
    /// Shared location for FleetServiceLive GPS injection
    static var lastKnownLocation: CLLocationCoordinate2D?
    let geofenceThreshold: Double = 100

    var pendingMissions: [Mission] {
        missions.filter { !completedIds.contains($0.id) }
    }

    var pendingOrders: [Order] {
        orders.filter { $0.state.isActive && !completedIds.contains($0.id) }
    }

    var loadedOrders: [Order] {
        orders.filter { $0.state == .LOADED }
    }

    var inTransitOrders: [Order] {
        orders.filter { $0.state == .IN_TRANSIT || $0.state == .ARRIVING || $0.state == .ARRIVED }
    }

    var hasActiveRoute: Bool { activeMission != nil || !inTransitOrders.isEmpty }

    var hasActiveOrders: Bool {
        orders.contains { $0.state.isActive }
    }

    var displayRouteCoordinates: [CLLocationCoordinate2D] {
        routeGeometryCoordinates.count >= 2 ? routeGeometryCoordinates : plannedRouteCoordinates
    }

    // MARK: - End Session State
    var isEndingSession = false
    var endSessionError: String?

    // MARK: - LEO: Ghost Stop Prevention
    var manifestId: String?
    var manifestSealed = false
    var manifestState: String?
    var awaitingSeal = false

    // MARK: - Delivery-edge feedback
    var deliveryEdgeError: String?
    var deliveryEdgeMessage: String?

    // MARK: - Init

    convenience override init() {
        self.init(fleetService: FleetServiceLive.shared)
    }

    init(fleetService: FleetServiceProtocol) {
        self.fleetService = fleetService
        super.init()
        locationManager.delegate = self
        locationManager.desiredAccuracy = kCLLocationAccuracyBestForNavigation
        locationManager.distanceFilter = 10
        ProfileService.shared.startPolling()
    }

    // MARK: - Actions

    func handleSocketEvent(_ event: DriverSocketState.DriverEvent?) {
        guard let event else { return }
        guard DriverWsRefresh.shouldRefreshManifest(eventType: event.type) else { return }
        Task {
            await loadMissions()
            lastRealtimeRefreshAt = Date()
        }
    }

    func loadMissions() async {
        isLoadingMissions = true
        defer { isLoadingMissions = false }

        // Try real API first
        do {
            let fetched = try await APIClient.shared.getAssignedOrders()
            orders = fetched
            // Bridge to legacy Mission format for existing views
            missions = fetched.map { order in
                Mission(
                    order_id: order.id,
                    state: order.state.rawValue,
                    target_lat: order.latitude,
                    target_lng: order.longitude,
                    amount: order.totalAmount,
                    gateway: "CASH",
                    estimated_arrival_at: order.estimatedArrivalAt,
                    route_id: order.routeId,
                    sequence_index: order.sequenceIndex
                )
            }
        } catch {
            missions = []
            orders = []
        }

        deriveTruckStatus()
        refreshPlannedRoute()
        await loadRouteGeometry()
    }

    func refreshPlannedRoute() {
        let routeId = activeOrder?.routeId ?? activeMission?.route_id ?? resolveActiveRouteId(orders: orders)
        plannedRouteCoordinates = buildPlannedRouteCoordinates(orders: orders, activeRouteId: routeId)
    }

    func loadRouteGeometry() async {
        let routeId = activeOrder?.routeId ?? activeMission?.route_id ?? resolveActiveRouteId(orders: orders)
        guard let routeId, !routeId.isEmpty else {
            navigationStepIndex = 0
            navigationVoiceAnnouncer.stop()
            routeDeviationTracker.reset()
            routeGeometryCoordinates = []
            routeSteps = []
            navigationCue = nil
            return
        }
        do {
            let response = try await APIClient.shared.getRouteGeometry(routeId: routeId)
            navigationStepIndex = 0
            navigationVoiceAnnouncer.stop()
            routeDeviationTracker.reset()
            routeGeometryCoordinates = response.coordinates.map {
                CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
            }
            routeSteps = response.steps
            navigationCue = nil
        } catch {
            navigationStepIndex = 0
            navigationVoiceAnnouncer.stop()
            routeDeviationTracker.reset()
            routeGeometryCoordinates = []
            routeSteps = []
            navigationCue = nil
        }
    }

    func checkRouteDeviation(coordinate: CLLocationCoordinate2D?) {
        guard isTransitActive, let coordinate, !isReroutingRoute else { return }
        let routeId = activeOrder?.routeId ?? activeMission?.route_id ?? resolveActiveRouteId(orders: orders)
        guard let routeId, !routeId.isEmpty else { return }
        let action = routeDeviationTracker.evaluate(
            now: Date(),
            coordinate: coordinate,
            polyline: routeGeometryCoordinates
        )
        guard action == .reroute else { return }
        Task { await rerouteRouteGeometry(routeId: routeId, from: coordinate) }
    }

    private func rerouteRouteGeometry(routeId: String, from coordinate: CLLocationCoordinate2D) async {
        isReroutingRoute = true
        defer { isReroutingRoute = false }
        do {
            let response = try await APIClient.shared.getRouteGeometry(
                routeId: routeId,
                includeSteps: true,
                from: coordinate,
                reroute: true
            )
            navigationStepIndex = 0
            navigationVoiceAnnouncer.stop()
            routeDeviationTracker.reset()
            routeGeometryCoordinates = response.coordinates.map {
                CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
            }
            routeSteps = response.steps
            navigationCue = nil
        } catch {
            routeDeviationTracker.reset()
        }
    }

    func updateNavigationCue(coordinate: CLLocationCoordinate2D?) {
        guard let coordinate, !routeSteps.isEmpty else {
            navigationCue = nil
            return
        }
        let previousIndex = navigationStepIndex
        navigationStepIndex = RouteNavigation.advanceStepIndex(
            currentIndex: navigationStepIndex,
            steps: routeSteps,
            coordinate: coordinate
        )
        navigationCue = RouteNavigation.resolveCue(
            steps: routeSteps,
            stepIndex: navigationStepIndex,
            coordinate: coordinate
        )
        if let cue = navigationCue, shouldAnnounceManeuverAdvance(previousIndex: previousIndex, nextIndex: navigationStepIndex) {
            navigationVoiceAnnouncer.announce(cue)
        }
    }

    func startLocationInterpolation() {
        interpolationTask?.cancel()
        interpolationTask = Task { @MainActor [weak self] in
            while !Task.isCancelled {
                guard let self else { return }
                self.locationInterpolator.tick()
                self.displayLocation = self.locationInterpolator.displayCoordinate
                self.displayBearing = self.locationInterpolator.displayBearing
                self.displaySpeedMps = self.locationInterpolator.displaySpeedMps
                self.updateNavigationCue(coordinate: self.displayLocation)
                self.checkRouteDeviation(coordinate: self.displayLocation)
                try? await Task.sleep(nanoseconds: MapCameraConfig.interpolationFrameNs)
            }
        }
    }

    func stopLocationInterpolation() {
        interpolationTask?.cancel()
        interpolationTask = nil
        locationInterpolator.clear()
        displayLocation = nil
        navigationVoiceAnnouncer.stop()
        routeDeviationTracker.reset()
    }

    func selectMission(_ mission: Mission) {
        Haptics.heavy()
        activeMission = mission
        activeOrder = orders.first { $0.id == mission.order_id }
        refreshPlannedRoute()
        Task { await loadRouteGeometry() }
        showMarkerSheet = true
    }

    func markCompleted(_ orderId: String) {
        completedIds.insert(orderId)
        if let m = missions.first(where: { $0.id == orderId }) {
            completedMissions.insert(m, at: 0)
        }
        if let o = orders.first(where: { $0.id == orderId }) {
            completedOrders.insert(o, at: 0)
        }
        if activeMission?.order_id == orderId {
            activeMission = nil
            activeOrder = nil
            showMarkerSheet = false
            refreshPlannedRoute()
        }
    }

    func dismissMarkerSheet() {
        showMarkerSheet = false
    }

    // MARK: - Transit Control (matches Android departRoute / returnComplete)

    func startTransit() async {
        await departRoute()
    }

    func departRoute() async {
        guard let vehicleId = TokenStore.shared.vehicleId, !vehicleId.isEmpty else {
            // Fallback: transition orders individually if no vehicle assigned
            await legacyStartTransit()
            return
        }

        // LEO: Ghost Stop Prevention — check manifest seal gate before depart
        if let mId = manifestId, !manifestSealed {
            do {
                let gate = try await APIClient.shared.checkManifestGate(manifestId: mId)
                if !gate.cleared {
                    manifestState = gate.state
                    awaitingSeal = true
                    return
                }
                manifestSealed = true
                awaitingSeal = false
            } catch {
                // Graceful degradation — allow depart if gate check fails
            }
        }

        Haptics.heavy()

        do {
            _ = try await APIClient.shared.depart(truckId: vehicleId)
            isTransitActive = true
            truckStatus = "IN_TRANSIT"
            isTelemetryLive = true
            await loadMissions()
        } catch {
            // Fallback to legacy per-order transition
            await legacyStartTransit()
        }
    }

    func returnComplete() async {
        guard let vehicleId = TokenStore.shared.vehicleId, !vehicleId.isEmpty else { return }

        Haptics.heavy()

        do {
            _ = try await APIClient.shared.returnComplete(truckId: vehicleId)
            truckStatus = "AVAILABLE"
            isReturning = false
            isTransitActive = false
            isTelemetryLive = false
            await loadMissions()
        } catch {
            // Silent — driver can retry
        }
    }

    func endSession(reason: String, note: String?) async {
        isEndingSession = true
        endSessionError = nil

        do {
            try await APIClient.shared.setAvailability(available: false, reason: reason, note: note)
            isEndingSession = false
            ProfileService.shared.stopPolling()
            TokenStore.shared.logout()
        } catch {
            isEndingSession = false
            endSessionError = "Failed to end session: \(error.localizedDescription)"
        }
    }

    // MARK: - Delivery-edge APIs

    func markCreditDelivery(orderId: String, photoProofUrl: String? = nil) async {
        deliveryEdgeError = nil
        deliveryEdgeMessage = nil
        do {
            _ = try await fleetService.markCreditDelivery(orderId: orderId, photoProofUrl: photoProofUrl)
            deliveryEdgeMessage = "Credit delivery recorded"
            await loadMissions()
        } catch {
            deliveryEdgeError = error.localizedDescription
        }
    }

    func recordSplitPayment(orderId: String, totalAmount: Int, cashMinor: Int? = nil, cardMinor: Int? = nil, currency: String? = nil) async {
        deliveryEdgeError = nil
        deliveryEdgeMessage = nil
        let cash = Int64(cashMinor ?? totalAmount / 2)
        let card = Int64(cardMinor ?? totalAmount - Int(cash))
        guard cash + card > 0 else {
            deliveryEdgeError = "Split amounts must be greater than zero"
            return
        }
        do {
            _ = try await fleetService.splitPayment(
                orderId: orderId,
                cashMinor: cash,
                cardMinor: card,
                currency: currency
            )
            deliveryEdgeMessage = "Split payment recorded"
        } catch {
            deliveryEdgeError = error.localizedDescription
        }
    }

    func updateOrderDuringDelivery(orderId: String) async {
        deliveryEdgeError = nil
        deliveryEdgeMessage = nil
        let location = await MainActor.run { Self.lastKnownLocation }
        guard let location, location.latitude != 0 || location.longitude != 0 else {
            deliveryEdgeError = "GPS unavailable for in-delivery update"
            return
        }
        do {
            let response = try await fleetService.updateOrderDuringDelivery(
                orderId: orderId,
                latitude: location.latitude,
                longitude: location.longitude
            )
            guard response.success else {
                deliveryEdgeError = response.message
                return
            }
            deliveryEdgeMessage = response.message
            await loadMissions()
        } catch {
            deliveryEdgeError = error.localizedDescription
        }
    }

    private func legacyStartTransit() async {
        isTransitActive = true
        Haptics.heavy()
        for order in loadedOrders {
            do {
                let updated = try await APIClient.shared.transitionState(
                    orderId: order.id,
                    newState: "IN_TRANSIT"
                )
                if let idx = orders.firstIndex(where: { $0.id == updated.id }) {
                    orders[idx] = updated
                }
            } catch { }
        }
        isTelemetryLive = true
        deriveTruckStatus()
    }

    /// Derive truck status from current order states
    private func deriveTruckStatus() {
        let activeStates: Set<OrderState> = [.IN_TRANSIT, .ARRIVING, .ARRIVED, .AWAITING_PAYMENT, .PENDING_CASH_COLLECTION]
        let hasActive = orders.contains { activeStates.contains($0.state) }
        let hasLoaded = orders.contains { $0.state == .LOADED || $0.state == .DISPATCHED }
        let allDone = !orders.isEmpty && orders.allSatisfy { $0.state == .COMPLETED || $0.state == .CANCELLED }

        if hasActive {
            truckStatus = "IN_TRANSIT"
            isTransitActive = true
            isReturning = false
        } else if allDone && isTransitActive {
            truckStatus = "RETURNING"
            isReturning = true
        } else if hasLoaded {
            truckStatus = "READY"
            isReturning = false
        } else {
            // Keep existing status if we have no orders
            if orders.isEmpty {
                isReturning = false
            }
        }
    }

    // MARK: - Reorder Stops

    func moveOrder(from fromIndex: Int, to toIndex: Int) {
        guard fromIndex != toIndex,
              fromIndex >= 0, fromIndex < missions.count,
              toIndex >= 0, toIndex < missions.count else { return }

        // Optimistic local reorder
        let moving = missions.remove(at: fromIndex)
        missions.insert(moving, at: toIndex)

        // Also reorder the orders array to stay in sync
        if fromIndex < orders.count && toIndex < orders.count {
            let movingOrder = orders.remove(at: fromIndex)
            orders.insert(movingOrder, at: toIndex)
        }

        Haptics.heavy()

        // Sync to backend
        guard let routeId = moving.route_id else { return }
        let sequence = missions.map { $0.order_id }

        Task {
            do {
                _ = try await APIClient.shared.reorderStops(routeId: routeId, orderSequence: sequence)
                await loadRouteGeometry()
                refreshPlannedRoute()
            } catch {
                // Revert on failure
                await loadMissions()
            }
        }
    }

    // MARK: - Location

    func requestLocationPermission() {
    locationManager.requestAlwaysAuthorization()
    locationManager.allowsBackgroundLocationUpdates = true
    locationManager.showsBackgroundLocationIndicator = true
    }

    func distanceToMission(_ mission: Mission) -> Double? {
        guard let loc = location else { return nil }
        let target = CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
        return haversineDistance(from: loc, to: target)
    }

    func isInRange(_ mission: Mission) -> Bool {
        guard let dist = distanceToMission(mission) else { return false }
        return dist <= geofenceThreshold
    }

    // MARK: - CLLocationManagerDelegate

    nonisolated func locationManager(_ manager: CLLocationManager, didUpdateLocations locations: [CLLocation]) {
        guard let loc = locations.last else { return }
        Task { @MainActor in
            self.location = loc.coordinate
            if loc.course >= 0 {
                self.course = loc.course
            }
            if loc.speed >= 0 {
                self.speed = loc.speed
            }
            self.locationInterpolator.onGps(loc)
            self.displayLocation = self.locationInterpolator.displayCoordinate ?? loc.coordinate
            self.displayBearing = self.locationInterpolator.displayBearing
            self.displaySpeedMps = self.locationInterpolator.displaySpeedMps
            self.updateNavigationCue(coordinate: self.displayLocation)
            self.checkRouteDeviation(coordinate: self.displayLocation)
            Self.lastKnownLocation = loc.coordinate
            self.gpsError = nil
            self.appendTrailPoint(loc.coordinate)

            // Auto-ARRIVED execution
            self.checkAutoArrive(driverLocation: loc.coordinate)
            
            // Adaptive Pipeline filter
            self.processAdaptiveTelemetrySync(loc)
        }
    }

    @MainActor
    private func processAdaptiveTelemetrySync(_ current: CLLocation) {
        let now = Date()
        var shouldTransmit = false
        
        if let lastT = lastSentTime, let lastL = latestTransmitLocation {
            let timeDelta = now.timeIntervalSince(lastT)
            if timeDelta > 15 {
                shouldTransmit = true
            } else {
                let distDelta = current.distance(from: lastL)
                let courseCurrent = current.course >= 0 ? current.course : 0
                let courseLast = lastL.course >= 0 ? lastL.course : 0
                let courseDelta = abs(courseCurrent - courseLast)
                
                if distDelta > 20 || courseDelta > 15 {
                    shouldTransmit = true
                }
            }
        } else {
            shouldTransmit = true
        }
        
        if shouldTransmit {
            latestTransmitLocation = current
            lastSentTime = now
            // TelemetryViewModel or ContentView observers will pick this up
        }
    }

    nonisolated func locationManager(_ manager: CLLocationManager, didFailWithError error: Error) {
        Task { @MainActor in
            self.gpsError = error.localizedDescription
        }
    }

    nonisolated func locationManagerDidChangeAuthorization(_ manager: CLLocationManager) {
        Task { @MainActor in
            switch manager.authorizationStatus {
            case .denied, .restricted:
                self.gpsError = "Location permission denied. Enable in Settings."
            case .authorizedWhenInUse, .authorizedAlways:
                self.gpsError = nil
                manager.startUpdatingLocation()
            default:
                break
            }
        }
    }

    private func appendTrailPoint(_ coordinate: CLLocationCoordinate2D) {
        locationTrail.append(coordinate)
        if locationTrail.count > maxTrailPoints {
            locationTrail.removeFirst(locationTrail.count - maxTrailPoints)
        }
    }

    // MARK: - Auto-ARRIVED Proximity Gate

    /// Checks all IN_TRANSIT orders and auto-transitions to ARRIVED when within geofence.
    /// Fires once per order (guarded by `arrivedIds`).
    private func checkAutoArrive(driverLocation: CLLocationCoordinate2D) {
        let candidates = orders.filter { $0.state == .IN_TRANSIT && !arrivedIds.contains($0.id) }
        for order in candidates {
            let target = CLLocationCoordinate2D(latitude: order.latitude, longitude: order.longitude)
            let dist = haversineDistance(from: driverLocation, to: target)
            guard dist <= geofenceThreshold else { continue }

            arrivedIds.insert(order.id)
            Task {
                do {
                    try await APIClient.shared.markArrived(orderId: order.id)
                    if let idx = orders.firstIndex(where: { $0.id == order.id }) {
                        // Rebuild with updated state (Order.state is let)
                        let o = orders[idx]
                        orders[idx] = Order(
                            id: o.id, retailerId: o.retailerId, retailerName: o.retailerName,
                            state: .ARRIVED, totalAmount: o.totalAmount,
                            deliveryAddress: o.deliveryAddress, latitude: o.latitude, longitude: o.longitude,
                            qrToken: o.qrToken, paymentGateway: o.paymentGateway,
                            createdAt: o.createdAt, updatedAt: o.updatedAt, items: o.items,
                            estimatedArrivalAt: o.estimatedArrivalAt, etaDurationSec: o.etaDurationSec,
                            etaDistanceM: o.etaDistanceM, routeId: o.routeId, sequenceIndex: o.sequenceIndex
                        )
                    }
                } catch {
                    // Silently degrade — driver can still manually proceed
                    arrivedIds.remove(order.id)
                }
            }
        }
    }
}

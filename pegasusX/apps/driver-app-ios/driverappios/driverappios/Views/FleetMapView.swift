//
//  FleetMapView.swift
//  driverappios
//

import MapKit
import SwiftUI
import SwiftData

// MARK: - Map Flow States

enum MapPhase: Equatable {
    case pickingOrder
    case previewingOrder
    case activeDelivery
}

enum ZoomFocus: Int {
    case me = 0
    case destination = 1
    case both = 2

    var next: ZoomFocus { ZoomFocus(rawValue: (rawValue + 1) % 3)! }

    var icon: String {
        switch self {
        case .me:          return "person.fill"
        case .destination: return "mappin.and.ellipse"
        case .both:        return "arrow.up.left.and.arrow.down.right"
        }
    }

    var label: String {
        switch self {
        case .me:          return "Me"
        case .destination: return "Target"
        case .both:        return "Both"
        }
    }
}

// MARK: - FleetMapView

struct FleetMapView: View {
    @Environment(\.colorScheme) private var cs
    @Environment(\.modelContext) private var modelContext
    @Bindable var vm: FleetViewModel
    var goBack: () -> Void = {}

    @State private var telemetryVM = TelemetryViewModel()
    @State private var driverSocketState = DriverSocketState.shared
    @State private var navPath = NavigationPath()
    @State private var cameraPosition: MapCameraPosition = .userLocation(followsHeading: true, fallback: .automatic)
    @State private var isCameraLocked: Bool = false
    @State private var userPannedAt: Date?

    @State private var phase: MapPhase = .pickingOrder
    @State private var selectedMission: Mission?
    @State private var zoomFocus: ZoomFocus = .both
    @State private var validatedQR: ValidateQRResponse?
    @State private var scannedQRToken: String = ""
    @State private var offloadResponse: ConfirmOffloadResponse?
    @State private var showRescueSheet = false

    var body: some View {
        NavigationStack(path: $navPath) {
            mapBody
                .toolbar(.hidden, for: .navigationBar)
                .navigationDestination(for: String.self) { route in
                    switch route {
                    case "scanner":
                        QRScannerView(
                            onValidated: { response, token in
                                validatedQR = response
                                scannedQRToken = token
                                navPath.append("offload-review")
                            },
                            onCancel: { navPath = NavigationPath() }
                        )
                        .toolbar(.hidden, for: .navigationBar)
                    case "offload-review":
                        if let qr = validatedQR {
                            OffloadReviewView(
                                response: qr,
                                scannedToken: scannedQRToken,
                                driverId: vm.driverId,
                                onConfirm: { result in
                                    offloadResponse = result
                                    navPath.append("payment-waiting")
                                },
                                onCancel: { navPath = NavigationPath() },
                                onShopClosed: { orderId in
                                    navPath.append("shop-closed-waiting")
                                },
                                onCreditDelivery: { orderId, photoUrl in
                                    Task {
                                        await vm.markCreditDelivery(
                                            orderId: orderId,
                                            photoProofUrl: photoUrl
                                        )
                                    }
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "payment-waiting":
                        if let offload = offloadResponse {
                            PaymentWaitingView(
                                orderId: offload.orderId,
                                amount: offload.amount,
                                driverId: vm.driverId,
                                onCompleted: {
                                    vm.markCompleted(offload.orderId)
                                    navPath = NavigationPath()
                                    withAnimation(Anim.snappy) { phase = .pickingOrder; selectedMission = nil }
                                },
                                onCashCollectionRequired: {
                                    navPath.append("cash-collection")
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "cash-collection":
                        if let offload = offloadResponse {
                            CashCollectionView(
                                orderId: offload.orderId,
                                amount: offload.amount,
                                onCompleted: {
                                    vm.markCompleted(offload.orderId)
                                    navPath = NavigationPath()
                                    withAnimation(Anim.snappy) { phase = .pickingOrder; selectedMission = nil }
                                },
                                onCancel: { navPath = NavigationPath() },
                                onSplitPayment: { orderId, amount in
                                    Task { await vm.recordSplitPayment(orderId: orderId, totalAmount: amount) }
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "correction":
                        if let m = vm.activeMission {
                            let order = vm.orders.first(where: { $0.id == m.order_id })
                            DeliveryCorrectionView(
                                orderId: m.order_id,
                                driverId: vm.driverId,
                                isPartial: order?.isPartial ?? false,
                                onClose: { navPath = NavigationPath() },
                                onAmended: {
                                    vm.markCompleted(m.order_id)
                                    navPath = NavigationPath()
                                    withAnimation(Anim.snappy) { phase = .pickingOrder; selectedMission = nil }
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "shop-closed-waiting":
                        ShopClosedWaitingView(
                            orderId: validatedQR?.orderId ?? "",
                            driverId: vm.driverId,
                            onResolved: {
                                if let oid = validatedQR?.orderId { vm.markCompleted(oid) }
                                navPath = NavigationPath()
                                withAnimation(Anim.snappy) { phase = .pickingOrder; selectedMission = nil }
                            },
                            onCancel: { navPath = NavigationPath() }
                        )
                        .toolbar(.hidden, for: .navigationBar)
                    default: EmptyView()
                    }
                }
        }
    }

    // MARK: - Map Body

    private var mapBody: some View {
        ZStack {
            Map(position: $cameraPosition) {
                if !(isCameraLocked && phase == .activeDelivery) {
                    UserAnnotation()
                }

                if isCameraLocked, phase == .activeDelivery, let coord = vm.displayLocation {
                    Annotation("You", coordinate: coord) {
                        Image(systemName: "location.north.fill")
                            .font(.system(size: 14, weight: .bold))
                            .foregroundStyle(LabTheme.buttonFg)
                            .padding(8)
                            .background(LabTheme.fg, in: Circle())
                            .rotationEffect(.degrees(vm.displayBearing))
                    }
                }

                ForEach(vm.pendingMissions) { mission in
                    Annotation(mission.order_id, coordinate: CLLocationCoordinate2D(
                        latitude: mission.target_lat, longitude: mission.target_lng
                    )) {
                        missionPin(for: mission)
                    }
                }

                if vm.locationTrail.count >= 2 {
                    MapPolyline(coordinates: vm.locationTrail)
                        .stroke(LabTheme.fg.opacity(0.25), lineWidth: 2)
                }

                if vm.displayRouteCoordinates.count >= 2 {
                    MapPolyline(coordinates: vm.displayRouteCoordinates)
                        .stroke(LabTheme.fg.opacity(0.45), style: StrokeStyle(lineWidth: 3, dash: [8, 6]))
                }

                if let loc = vm.location, let target = currentTarget {
                    MapPolyline(coordinates: [
                        loc,
                        CLLocationCoordinate2D(latitude: target.target_lat, longitude: target.target_lng)
                    ])
                    .stroke(LabTheme.fg.opacity(0.35), lineWidth: 2.5)
                }
            }
            .mapStyle(.standard(elevation: .realistic))
            .mapControls { MapCompass() }
            .onMapCameraChange(frequency: .onEnd) {
                isCameraLocked = false
                userPannedAt = Date()
            }
            .ignoresSafeArea()

            // GPS Error
            VStack {
                if let err = vm.gpsError { GPSErrorBanner(message: err) }
                Spacer()
            }
            .animation(Anim.snappy, value: vm.gpsError)

            // Top bar
            MapTopOverlay(
                phase: $phase,
                selectedMission: $selectedMission,
                isCameraLocked: $isCameraLocked,
                userPannedAt: $userPannedAt,
                cameraPosition: $cameraPosition,
                zoomFocus: $zoomFocus,
                currentTarget: currentTarget,
                vm: vm,
                isLive: telemetryVM.isLive,
                goBack: goBack,
                cycleZoom: cycleZoom
            )

            // Bottom sheet
            VStack {
                Spacer()
                bottomSheet
            }
            .animation(Anim.sheetReveal, value: phase)
        }
        .ignoresSafeArea(edges: .bottom)
        .task {
            vm.requestLocationPermission()
            vm.startLocationInterpolation()
            await vm.loadMissions()
            if let active = vm.activeMission {
                selectedMission = active
                phase = .activeDelivery
                isCameraLocked = true
                await telemetryVM.start()
            }
        }
        .onChange(of: phase) { _, newPhase in
            if newPhase == .activeDelivery {
                isCameraLocked = true
                userPannedAt = nil
                vm.refreshPlannedRoute()
            }
        }
        .task(id: userPannedAt) {
            guard let pannedAt = userPannedAt, phase == .activeDelivery else { return }
            try? await Task.sleep(nanoseconds: MapCameraConfig.idleRecenterMs)
            if userPannedAt == pannedAt {
                isCameraLocked = true
                userPannedAt = nil
            }
        }
        .onChange(of: vm.displayLocation) { _, coordinate in
            guard isCameraLocked, phase == .activeDelivery, let coordinate else { return }
            withAnimation(.easeInOut(duration: MapCameraConfig.cameraAnimationSeconds)) {
                cameraPosition = .camera(
                    MapCameraMath.trackingCamera(
                        coordinate: coordinate,
                        bearing: vm.displayBearing,
                        speedMps: vm.displaySpeedMps
                    )
                )
            }
        }
        .onChange(of: vm.displayBearing) { _, _ in
            guard isCameraLocked, phase == .activeDelivery, let coordinate = vm.displayLocation else { return }
            cameraPosition = .camera(
                MapCameraMath.trackingCamera(
                    coordinate: coordinate,
                    bearing: vm.displayBearing,
                    speedMps: vm.displaySpeedMps
                )
            )
        }
        .onDisappear {
            vm.stopLocationInterpolation()
        }
        .onChange(of: vm.latestTransmitLocation) { _, loc in
            // V.O.I.D. Adaptive Transmission Protocol Filtered Execution
            if let loc {
                telemetryVM.sendLocation(loc.coordinate, accuracy: loc.horizontalAccuracy)
            }
        }
        .onChange(of: driverSocketState.connectionState) { _, _ in
            telemetryVM.syncLiveFlags()
        }
        .onChange(of: driverSocketState.reconnectEpoch) { _, _ in
            Task {
                await DriverReconnectRecovery.recoverInFlight(wasInFlight: false)
                await vm.loadMissions(silent: true)
                await FleetServiceLive.shared.flushOfflineQueue()
                vm.lastRealtimeRefreshAt = Date()
            }
        }
        .onChange(of: driverSocketState.eventSequence) { _, _ in
            vm.handleSocketEvent(driverSocketState.lastEvent)
            telemetryVM.syncLiveFlags()
        }
        .sheet(isPresented: $vm.showOfflineVerifier) {
            OfflineVerifierView(modelContext: modelContext)
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
        .sheet(isPresented: $showRescueSheet) {
            RequestRescueSheet()
                .presentationDetents([.medium])
                .presentationDragIndicator(.visible)
        }
    }

    private var currentTarget: Mission? {
        selectedMission ?? vm.activeMission
    }

    // MARK: - Bottom Sheet Router

    @ViewBuilder
    private var bottomSheet: some View {
        switch phase {
        case .pickingOrder:
            OrderPickerSheet(
                vm: vm,
                selectedMission: $selectedMission,
                phase: $phase,
                zoomFocus: $zoomFocus,
                bottomInset: bottomInset,
                zoomTo: zoomTo
            ).transition(.move(edge: .bottom).combined(with: .opacity))
        case .previewingOrder:
            if let m = selectedMission {
                OrderPreviewSheet(
                    mission: m,
                    vm: vm,
                    bottomInset: bottomInset,
                    phase: $phase,
                    selectedMission: $selectedMission,
                    isCameraLocked: $isCameraLocked,
                    userPannedAt: $userPannedAt,
                    startTelemetry: { Task { await telemetryVM.start() } }
                ).transition(.move(edge: .bottom).combined(with: .opacity))
            }
        case .activeDelivery:
            if let m = vm.activeMission {
                ActiveDeliverySheet(
                    mission: m,
                    vm: vm,
                    bottomInset: bottomInset,
                    navPath: $navPath,
                    showRescueSheet: $showRescueSheet
                ).transition(.move(edge: .bottom).combined(with: .opacity))
            }
        }
    }

    // MARK: - 3-State Zoom

    private func cycleZoom() {
        zoomFocus = zoomFocus.next
        if let m = currentTarget { zoomTo(zoomFocus, mission: m) }
    }

    private func zoomTo(_ focus: ZoomFocus, mission: Mission) {
        let dest = CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
        let me = vm.location ?? FleetViewModel.warehouseCenter

        withAnimation(Anim.settle) {
            switch focus {
            case .me:
                cameraPosition = .region(MKCoordinateRegion(
                    center: me, span: MKCoordinateSpan(latitudeDelta: 0.008, longitudeDelta: 0.008)
                ))
            case .destination:
                cameraPosition = .region(MKCoordinateRegion(
                    center: dest, span: MKCoordinateSpan(latitudeDelta: 0.008, longitudeDelta: 0.008)
                ))
            case .both:
                let minLat = min(me.latitude, dest.latitude)
                let maxLat = max(me.latitude, dest.latitude)
                let minLng = min(me.longitude, dest.longitude)
                let maxLng = max(me.longitude, dest.longitude)
                let center = CLLocationCoordinate2D(latitude: (minLat + maxLat) / 2, longitude: (minLng + maxLng) / 2)
                let spanLat = max((maxLat - minLat) * 1.6, 0.01)
                let spanLng = max((maxLng - minLng) * 1.6, 0.01)
                cameraPosition = .region(MKCoordinateRegion(center: center, span: MKCoordinateSpan(latitudeDelta: spanLat, longitudeDelta: spanLng)))
            }
        }
    }

    // MARK: - Helpers

    private var bottomInset: Double {
        let scenes = UIApplication.shared.connectedScenes
        let windowScene = scenes.first as? UIWindowScene
        return Double(windowScene?.windows.first?.safeAreaInsets.bottom ?? 0)
    }

    @ViewBuilder
    private func missionPin(for mission: Mission) -> some View {
        let isSelected = selectedMission?.id == mission.id || vm.activeMission?.id == mission.id
        Button {
            Haptics.medium()
            if phase == .pickingOrder {
                withAnimation(Anim.sheetReveal) {
                    selectedMission = mission
                    phase = .previewingOrder
                    zoomFocus = .both
                }
                zoomTo(.both, mission: mission)
            }
        } label: {
            VStack(spacing: 0) {
                ZStack {
                    Circle()
                        .fill(isSelected ? LabTheme.fg : LabTheme.bg) // Toggle bg/fg
                        .frame(width: isSelected ? 42 : 36, height: isSelected ? 42 : 36)
                        .overlay {
                            Circle()
                                .stroke(isSelected ? LabTheme.fg : LabTheme.separator.opacity(0.12), lineWidth: 1)
                        }
                    Image(systemName: "shippingbox.fill")
                        .font(.system(size: isSelected ? 18 : 14, weight: .black)) // Tactical weights
                        .foregroundStyle(isSelected ? LabTheme.buttonFg : LabTheme.fg)
                }
                .shadow(color: .black.opacity(0.1), radius: 6, y: 3)
                .animation(Anim.snappy, value: isSelected)

                Image(systemName: "triangle.fill")
                    .font(.system(size: 7))
                    .foregroundStyle(isSelected ? LabTheme.fg : LabTheme.fg.opacity(0.5))
                    .rotationEffect(.degrees(180))
                    .offset(y: -3)
            }
        }
    }
}

#Preview {
    FleetMapView(vm: FleetViewModel())
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}

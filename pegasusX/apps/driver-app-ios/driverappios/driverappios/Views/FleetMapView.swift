//
//  FleetMapView.swift
//  driverappios
//

import MapKit
import SwiftData
import SwiftUI

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

    var next: ZoomFocus { ZoomFocus(rawValue: (rawValue + 1) % 3) ?? .me }

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
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @Environment(\.modelContext) private var modelContext
    @Bindable var vm: FleetViewModel
    var goBack: () -> Void = {}

    @State private var mapVM = FleetMapViewModel()
    @State private var telemetryVM = TelemetryViewModel()
    @State private var driverSocketState = DriverSocketState.shared

    var body: some View {
        @Bindable var mapVM = mapVM
        NavigationStack(path: $mapVM.navPath) {
            mapBody
                .toolbar(.hidden, for: .navigationBar)
                .navigationDestination(for: String.self) { route in
                    switch route {
                    case "scanner":
                        QRScannerView(
                            onValidated: { response, token in
                                mapVM.handleValidatedQR(response, token: token)
                            },
                            onCancel: { mapVM.navPath = NavigationPath() }
                        )
                        .toolbar(.hidden, for: .navigationBar)
                    case "offload-review":
                        if let qr = mapVM.validatedQR {
                            OffloadReviewView(
                                response: qr,
                                scannedToken: mapVM.scannedQRToken,
                                driverId: vm.driverId,
                                onConfirm: { result in
                                    mapVM.handleConfirmedOffload(result)
                                },
                                onCancel: { mapVM.navPath = NavigationPath() },
                                onShopClosed: { _ in
                                    mapVM.navPath.append("shop-closed-waiting")
                                },
                                onCreditDelivery: { orderId, photoUrl, signatureUrl, photoLocal, sigLocal in
                                    Task {
                                        await vm.markCreditDelivery(
                                            orderId: orderId,
                                            photoProofUrl: photoUrl.isEmpty ? nil : photoUrl,
                                            signatureUrl: signatureUrl.isEmpty ? nil : signatureUrl,
                                            photoLocalPath: photoLocal,
                                            signatureLocalPath: sigLocal
                                        )
                                    }
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "payment-waiting":
                        if let offload = mapVM.offloadResponse {
                            PaymentWaitingView(
                                orderId: offload.orderId,
                                amount: offload.amount,
                                driverId: vm.driverId,
                                onCompleted: {
                                    vm.markCompleted(offload.orderId)
                                    withAnimation(Anim.snappy) {
                                        mapVM.resetNavigationAndReturnToPicker()
                                    }
                                },
                                onCashCollectionRequired: {
                                    mapVM.navPath.append("cash-collection")
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "cash-collection":
                        if let offload = mapVM.offloadResponse {
                            CashCollectionView(
                                orderId: offload.orderId,
                                amount: offload.amount,
                                onCompleted: {
                                    vm.markCompleted(offload.orderId)
                                    withAnimation(Anim.snappy) {
                                        mapVM.resetNavigationAndReturnToPicker()
                                    }
                                },
                                onCancel: { mapVM.navPath = NavigationPath() },
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
                                onClose: { mapVM.navPath = NavigationPath() },
                                onAmended: {
                                    vm.markCompleted(m.order_id)
                                    withAnimation(Anim.snappy) {
                                        mapVM.resetNavigationAndReturnToPicker()
                                    }
                                }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "shop-closed-waiting":
                        ShopClosedWaitingView(
                            orderId: mapVM.validatedQR?.orderId ?? "",
                            driverId: vm.driverId,
                            onResolved: {
                                if let oid = mapVM.validatedQR?.orderId { vm.markCompleted(oid) }
                                withAnimation(Anim.snappy) {
                                    mapVM.resetNavigationAndReturnToPicker()
                                }
                            },
                            onCancel: { mapVM.navPath = NavigationPath() }
                        )
                        .toolbar(.hidden, for: .navigationBar)
                    default: EmptyView()
                    }
                }
        }
    }

    // MARK: - Map Body

    private var mapBody: some View {
        @Bindable var mapVM = mapVM
        return ZStack {
            Map(position: $mapVM.cameraPosition) {
                if !(mapVM.isCameraLocked && mapVM.phase == .activeDelivery) {
                    UserAnnotation()
                }

                if mapVM.isCameraLocked, mapVM.phase == .activeDelivery, let coord = vm.displayLocation {
                    Annotation("You", coordinate: coord) {
                        Image(systemName: "location.north.fill")
                            .font(.system(.subheadline, weight: .bold))
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
                mapVM.isCameraLocked = false
                mapVM.userPannedAt = Date()
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
                phase: $mapVM.phase,
                selectedMission: $mapVM.selectedMission,
                isCameraLocked: $mapVM.isCameraLocked,
                userPannedAt: $mapVM.userPannedAt,
                cameraPosition: $mapVM.cameraPosition,
                zoomFocus: $mapVM.zoomFocus,
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
                    .frame(maxWidth: horizontalSizeClass == .regular ? 500 : .infinity)
            }
            .padding(.horizontal, horizontalSizeClass == .regular ? LabTheme.s24 : 0)
            .padding(.bottom, horizontalSizeClass == .regular ? LabTheme.s20 : 0)
            .animation(Anim.sheetReveal, value: mapVM.phase)
        }
        .ignoresSafeArea(edges: .bottom)
        .task {
            vm.requestLocationPermission()
            vm.startLocationInterpolation()
            await vm.loadMissions()
            if let active = vm.activeMission {
                mapVM.selectedMission = active
                mapVM.phase = .activeDelivery
                mapVM.isCameraLocked = true
                await telemetryVM.start()
            }
        }
        .onChange(of: mapVM.phase) { _, newPhase in
            if newPhase == .activeDelivery {
                mapVM.isCameraLocked = true
                mapVM.userPannedAt = nil
                vm.refreshPlannedRoute()
            }
        }
        .task(id: mapVM.userPannedAt) {
            await mapVM.handleIdleRecenter(for: mapVM.userPannedAt)
        }
        .onChange(of: vm.displayLocation) { _, coordinate in
            guard mapVM.isCameraLocked, mapVM.phase == .activeDelivery, let coordinate else { return }
            withAnimation(.easeInOut(duration: MapCameraConfig.cameraAnimationSeconds)) {
                mapVM.cameraPosition = .camera(
                    MapCameraMath.trackingCamera(
                        coordinate: coordinate,
                        bearing: vm.displayBearing,
                        speedMps: vm.displaySpeedMps
                    )
                )
            }
        }
        .onChange(of: vm.displayBearing) { _, _ in
            guard mapVM.isCameraLocked, mapVM.phase == .activeDelivery, let coordinate = vm.displayLocation else { return }
            mapVM.cameraPosition = .camera(
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
        .sheet(isPresented: $mapVM.showRescueSheet) {
            RequestRescueSheet()
                .presentationDetents([.medium])
                .presentationDragIndicator(.visible)
        }
    }

    private var currentTarget: Mission? {
        mapVM.selectedMission ?? vm.activeMission
    }

    // MARK: - Bottom Sheet Router

    @ViewBuilder
    private var bottomSheet: some View {
        @Bindable var mapVM = mapVM
        switch mapVM.phase {
        case .pickingOrder:
            OrderPickerSheet(
                vm: vm,
                selectedMission: $mapVM.selectedMission,
                phase: $mapVM.phase,
                zoomFocus: $mapVM.zoomFocus,
                bottomInset: bottomInset,
                zoomTo: zoomTo
            ).transition(.move(edge: .bottom).combined(with: .opacity))
        case .previewingOrder:
            if let m = mapVM.selectedMission {
                OrderPreviewSheet(
                    mission: m,
                    vm: vm,
                    bottomInset: bottomInset,
                    phase: $mapVM.phase,
                    selectedMission: $mapVM.selectedMission,
                    isCameraLocked: $mapVM.isCameraLocked,
                    userPannedAt: $mapVM.userPannedAt,
                    startTelemetry: { Task { await telemetryVM.start() } }
                ).transition(.move(edge: .bottom).combined(with: .opacity))
            }
        case .activeDelivery:
            if let m = vm.activeMission {
                ActiveDeliverySheet(
                    mission: m,
                    vm: vm,
                    bottomInset: bottomInset,
                    navPath: $mapVM.navPath,
                    showRescueSheet: $mapVM.showRescueSheet
                ).transition(.move(edge: .bottom).combined(with: .opacity))
            }
        }
    }

    // MARK: - 3-State Zoom

    private func cycleZoom() {
        mapVM.zoomFocus = mapVM.zoomFocus.next
        if let m = currentTarget { zoomTo(mapVM.zoomFocus, mission: m) }
    }

    private func zoomTo(_ focus: ZoomFocus, mission: Mission) {
        let dest = CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng)
        let me = vm.location ?? FleetViewModel.warehouseCenter

        withAnimation(Anim.settle) {
            switch focus {
            case .me:
                mapVM.cameraPosition = .region(MKCoordinateRegion(
                    center: me, span: MKCoordinateSpan(latitudeDelta: 0.008, longitudeDelta: 0.008)
                ))
            case .destination:
                mapVM.cameraPosition = .region(MKCoordinateRegion(
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
                mapVM.cameraPosition = .region(MKCoordinateRegion(center: center, span: MKCoordinateSpan(latitudeDelta: spanLat, longitudeDelta: spanLng)))
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
        let isSelected = mapVM.selectedMission?.id == mission.id || vm.activeMission?.id == mission.id
        Button {
            Haptics.medium()
            if mapVM.phase == .pickingOrder {
                withAnimation(Anim.sheetReveal) {
                    mapVM.selectedMission = mission
                    mapVM.phase = .previewingOrder
                    mapVM.zoomFocus = .both
                }
                zoomTo(.both, mission: mission)
            }
        } label: {
            VStack(spacing: 0) {
                ZStack {
                    Circle()
                        .fill(isSelected ? LabTheme.fg : LabTheme.bg)
                        .frame(width: isSelected ? 44 : 40, height: isSelected ? 44 : 40)
                        .overlay {
                            Circle()
                                .stroke(isSelected ? LabTheme.fg : LabTheme.separator.opacity(0.12), lineWidth: 1)
                        }
                    Image(systemName: "shippingbox.fill")
                        .font(.system(isSelected ? .body : .subheadline, weight: .black))
                        .foregroundStyle(isSelected ? LabTheme.buttonFg : LabTheme.fg)
                }
                .shadow(color: .black.opacity(0.1), radius: 6, y: 3)
                .animation(Anim.snappy, value: isSelected)

                Image(systemName: "triangle.fill")
                    .font(.caption2)
                    .foregroundStyle(isSelected ? LabTheme.fg : LabTheme.fg.opacity(0.5))
                    .rotationEffect(.degrees(180))
                    .offset(y: -3)
            }
            .frame(minWidth: 44, minHeight: 44)
        }
        .accessibilityLabel("Mission \(mission.order_id)")
    }
}

#Preview {
    FleetMapView(vm: FleetViewModel())
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}

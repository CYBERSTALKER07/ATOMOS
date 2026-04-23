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
    @State private var navPath = NavigationPath()
    @State private var cameraPosition: MapCameraPosition = .userLocation(followsHeading: true, fallback: .automatic)
    @State private var isCameraLocked: Bool = false

    @State private var phase: MapPhase = .pickingOrder
    @State private var selectedMission: Mission?
    @State private var zoomFocus: ZoomFocus = .both
    @State private var validatedQR: ValidateQRResponse?
    @State private var offloadResponse: ConfirmOffloadResponse?

    var body: some View {
        NavigationStack(path: $navPath) {
            mapBody
                .toolbar(.hidden, for: .navigationBar)
                .navigationDestination(for: String.self) { route in
                    switch route {
                    case "scanner":
                        QRScannerView(
                            onValidated: { response in
                                validatedQR = response
                                navPath.append("offload-review")
                            },
                            onCancel: { navPath = NavigationPath() }
                        )
                        .toolbar(.hidden, for: .navigationBar)
                    case "offload-review":
                        if let qr = validatedQR {
                            OffloadReviewView(
                                response: qr,
                                driverId: vm.driverId,
                                onConfirm: { result in
                                    offloadResponse = result
                                    if result.paymentMethod.uppercased() == "CASH" {
                                        navPath.append("cash-collection")
                                    } else {
                                        navPath.append("payment-waiting")
                                    }
                                },
                                onCancel: { navPath = NavigationPath() },
                                onShopClosed: { orderId in
                                    navPath.append("shop-closed-waiting")
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
                                onCancel: { navPath = NavigationPath() }
                            )
                            .toolbar(.hidden, for: .navigationBar)
                        }
                    case "correction":
                        if let m = vm.activeMission {
                            DeliveryCorrectionView(
                                orderId: m.order_id,
                                driverId: vm.driverId,
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
                UserAnnotation()

                ForEach(vm.pendingMissions) { mission in
                    Annotation(mission.order_id, coordinate: CLLocationCoordinate2D(
                        latitude: mission.target_lat, longitude: mission.target_lng
                    )) {
                        missionPin(for: mission)
                    }
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
            .onMapCameraChange(frequency: .onEnd) { context in
                if isCameraLocked && !context.followsUserLocation {
                    isCameraLocked = false
                }
            }
            .ignoresSafeArea()

            // GPS Error
            VStack {
                if let err = vm.gpsError { GPSErrorBanner(message: err) }
                Spacer()
            }
            .animation(Anim.snappy, value: vm.gpsError)

            // Top bar
            topOverlay

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
            await vm.loadMissions()
            if let active = vm.activeMission {
                selectedMission = active
                phase = .activeDelivery
                await telemetryVM.start()
            }
        }
        .onChange(of: vm.latestTransmitLocation) { _, loc in
            // V.O.I.D. Adaptive Transmission Protocol Filtered Execution
            if let loc { 
                telemetryVM.sendLocation(loc.coordinate, accuracy: loc.horizontalAccuracy) 
            }
        }
        .sheet(isPresented: $vm.showOfflineVerifier) {
            OfflineVerifierView(modelContext: modelContext)
                .presentationDetents([.large])
                .presentationDragIndicator(.visible)
        }
    }

    private var currentTarget: Mission? {
        selectedMission ?? vm.activeMission
    }

    // MARK: - Top Overlay

    private var topOverlay: some View {
        VStack {
            HStack(spacing: 10) {
                Button {
                    Haptics.light()
                    if phase == .previewingOrder {
                        withAnimation(Anim.snappy) { selectedMission = nil; phase = .pickingOrder }
                    } else {
                        goBack()
                    }
                } label: {
                    Image(systemName: phase == .previewingOrder ? "chevron.left" : "xmark")
                        .font(.system(size: 13, weight: .bold))
                        .foregroundStyle(LabTheme.fg)
                        .frame(width: 38, height: 38)
                        .background(.ultraThinMaterial, in: Circle())
                        .overlay(Circle().stroke(LabTheme.separator, lineWidth: 0.5))
                        .shadow(color: .black.opacity(0.08), radius: 8, y: 4)
                }
                .accessibilityLabel(phase == .previewingOrder ? "Back" : "Close map")

                Spacer()

                if phase != .pickingOrder, currentTarget != nil {
                    HStack(spacing: 8) {
                        Button {
                            withAnimation(.easeInOut(duration: 1.0)) {
                                isCameraLocked = true
                                cameraPosition = .camera(
                                    MapCamera(
                                        centerCoordinate: vm.location ?? FleetViewModel.warehouseCenter,
                                        distance: max((vm.speed ?? 0.0) * 20.0, 400.0), // V.O.I.D. Dynamic Vault-Pitch Look-Ahead
                                        heading: vm.course ?? 0,
                                        pitch: 60
                                    )
                                )
                                DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                                    cameraPosition = .userLocation(followsHeading: true, fallback: .automatic)
                                }
                            }
                        } label: {
                            Image(systemName: isCameraLocked ? "location.north.line.fill" : "location.fill")
                                .font(.system(size: 13, weight: .bold))
                                .padding(12)
                                .background(isCameraLocked ? LabTheme.primary : .ultraThinMaterial)
                                .foregroundStyle(isCameraLocked ? LabTheme.onPrimary : LabTheme.fg)
                                .clipShape(Circle())
                                .overlay(Circle().stroke(LabTheme.separator, lineWidth: 0.5))
                                .shadow(color: .black.opacity(0.08), radius: 8, y: 4)
                        }

                        Button {
                            Haptics.light()
                            cycleZoom()
                        } label: {
                            HStack(spacing: 4) {
                                Image(systemName: zoomFocus.icon)
                                    .font(.system(size: 10, weight: .bold))
                                Text(zoomFocus.label)
                                    .font(.system(size: 10, weight: .bold))
                            }
                            .foregroundStyle(LabTheme.fg)
                            .padding(.horizontal, 11)
                            .padding(.vertical, 8)
                            .background(.ultraThinMaterial, in: Capsule())
                            .overlay(Capsule().stroke(LabTheme.separator, lineWidth: 0.5))
                            .shadow(color: .black.opacity(0.08), radius: 8, y: 4)
                        }
                        .accessibilityLabel("Zoom focus: \(zoomFocus.label)")
                        .transition(.fadeScale)
                    }
                }

                if vm.activeMission != nil {
                    TelemetryBadge(isLive: telemetryVM.isLive)
                        .transition(.fadeScale)
                }
            }
            .padding(.horizontal, LabTheme.s16)
            .padding(.top, 60)
            .animation(Anim.snappy, value: phase)
            .animation(Anim.snappy, value: zoomFocus)

            Spacer()
        }
    }

    // MARK: - Bottom Sheet Router

    @ViewBuilder
    private var bottomSheet: some View {
        switch phase {
        case .pickingOrder:
            orderPickerSheet.transition(.move(edge: .bottom).combined(with: .opacity))
        case .previewingOrder:
            if let m = selectedMission {
                orderPreviewSheet(m).transition(.move(edge: .bottom).combined(with: .opacity))
            }
        case .activeDelivery:
            if let m = vm.activeMission {
                activeSheet(m).transition(.move(edge: .bottom).combined(with: .opacity))
            }
        }
    }

    // MARK: - 1. Order Picker

    private var orderPickerSheet: some View {
        VStack(alignment: .leading, spacing: 0) {
            sheetHandle

            VStack(alignment: .leading, spacing: 4) {
                Text("SELECT ORDER")
                    .font(.system(size: 9, weight: .heavy, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                    .tracking(1)
                Text("Choose a delivery")
                    .font(.system(size: 20, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
            }
            .padding(.horizontal, LabTheme.s20)
            .padding(.bottom, LabTheme.s12)

            if vm.isLoadingMissions {
                loadingRow
            } else if vm.pendingMissions.isEmpty {
                emptyRow
            } else {
                ScrollView {
                    LazyVStack(spacing: 6) {
                        ForEach(Array(vm.pendingMissions.enumerated()), id: \.element.id) { i, m in
                            pickerRow(m, index: i)
                        }
                    }
                    .padding(.horizontal, LabTheme.s16)
                    .padding(.bottom, LabTheme.s8)
                }
                .scrollIndicators(.hidden)
                .frame(maxHeight: 260)
            }
        }
        .padding(.bottom, bottomInset + LabTheme.s4)
        .background(glassSheet)


    }

    private func pickerRow(_ mission: Mission, index: Int) -> some View {
        Button {
            Haptics.medium()
            withAnimation(Anim.sheetReveal) {
                selectedMission = mission
                phase = .previewingOrder
                zoomFocus = .both
            }
            zoomTo(.both, mission: mission)
        } label: {
            HStack(spacing: 12) {
                ZStack {
                    RoundedRectangle(cornerRadius: 10, style: .continuous)
                        .fill(LabTheme.fg.opacity(0.06))
                        .frame(width: 38, height: 38)
                    Image(systemName: "shippingbox.fill")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(LabTheme.fg)
                }

                VStack(alignment: .leading, spacing: 2) {
                    Text(mission.order_id)
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                    HStack(spacing: 3) {
                        Text(mission.gateway)
                        Text("·")
                        Text(mission.amount.formattedAmount)
                    }
                    .font(.system(size: 10, weight: .medium))
                    .foregroundStyle(LabTheme.fgSecondary)
                }

                Spacer()

                if let loc = vm.location {
                    let d = haversineDistance(from: loc, to: CLLocationCoordinate2D(latitude: mission.target_lat, longitude: mission.target_lng))
                    Text(formattedDistance(d))
                        .font(.system(size: 10, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fgTertiary)
                }

                Image(systemName: "chevron.right")
                    .font(.system(size: 9, weight: .bold))
                    .foregroundStyle(LabTheme.fgTertiary)
            }
            .padding(LabTheme.s12)
            .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: 14))
        }
        .buttonStyle(.pressable)
        .staggeredAppear(index: index)
    }

    // MARK: - 2. Order Preview

    private func orderPreviewSheet(_ mission: Mission) -> some View {
        let dist = vm.distanceToMission(mission)
        let inRange = vm.isInRange(mission)
        let order = vm.orders.first { $0.id == mission.order_id }

        return VStack(alignment: .leading, spacing: 16) {
            sheetHandle

            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(mission.order_id)
                        .font(.system(size: 18, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                    HStack(spacing: 6) {
                        Text(mission.gateway)
                            .font(.system(size: 10, weight: .bold))
                            .padding(.horizontal, 8).padding(.vertical, 4)
                            .background(LabTheme.fg.opacity(0.07), in: Capsule())
                        Text(mission.amount.formattedAmount)
                            .font(.system(size: 12, weight: .semibold))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                }
                Spacer()
                StatusPill(label: mission.state, color: LabTheme.fg)
            }
            .padding(.horizontal, LabTheme.s20)

            VStack(alignment: .leading, spacing: 7) {
                Text("DESTINATION")
                    .font(.system(size: 9, weight: .heavy, design: .monospaced))
                    .foregroundStyle(LabTheme.fgTertiary)
                Text(String(format: "%.4f, %.4f", mission.target_lat, mission.target_lng))
                    .font(.system(size: 14, weight: .semibold, design: .monospaced))
                    .foregroundStyle(LabTheme.fg)
                if let d = dist {
                    HStack(spacing: 5) {
                        Circle().fill(inRange ? LabTheme.success : LabTheme.warning).frame(width: 6, height: 6)
                        Text(formattedDistance(d))
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgSecondary)
                        Text(inRange ? "In range" : "Approaching")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(inRange ? LabTheme.success : LabTheme.warning)
                    }
                }

                // ETA Row
                if let order, let eta = order.estimatedArrivalAt {
                    HStack(spacing: 5) {
                        Image(systemName: "clock.fill")
                            .font(.system(size: 10, weight: .semibold))
                            .foregroundStyle(LabTheme.fgTertiary)
                        Text("ETA \(formatETATime(eta))")
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .foregroundStyle(LabTheme.fg)
                        if let dur = order.etaDurationSec {
                            Text("· \(formatDuration(dur))")
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }
                        if let distM = order.etaDistanceM {
                            Text("· \(formatETADistance(distM))")
                                .font(.system(size: 11, weight: .medium))
                                .foregroundStyle(LabTheme.fgSecondary)
                        }
                    }
                }
            }
            .padding(LabTheme.s16)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(LabTheme.fg.opacity(0.03), in: .rect(cornerRadius: 14))
            .overlay {
                RoundedRectangle(cornerRadius: 14, style: .continuous)
                    .stroke(LabTheme.separator, lineWidth: 0.5)
            }
            .padding(.horizontal, LabTheme.s20)

            VStack(spacing: 8) {
                Button {
                    Haptics.heavy()
                    vm.activeMission = mission
                    withAnimation(Anim.sheetReveal) { phase = .activeDelivery }
                    Task { await telemetryVM.start() }
                } label: {
                    Text("Start Delivery")
                        .font(.system(size: 15, weight: .bold))
                        .foregroundStyle(LabTheme.buttonFg)
                        .frame(maxWidth: .infinity).padding(.vertical, 15)
                        .background(LabTheme.fg, in: .rect(cornerRadius: 14))
                }
                .buttonStyle(.pressable)

                // Navigate in Apple Maps
                Button {
                    Haptics.light()
                    openDestinationInMaps(lat: mission.target_lat, lng: mission.target_lng, name: mission.order_id)
                } label: {
                    HStack(spacing: 6) {
                        Image(systemName: "arrow.triangle.turn.up.right.diamond.fill")
                            .font(.system(size: 12, weight: .semibold))
                        Text("Navigate")
                            .font(.system(size: 13, weight: .semibold))
                    }
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity).padding(.vertical, 11)
                }
                .buttonStyle(.pressable)

                Button {
                    Haptics.light()
                    withAnimation(Anim.snappy) { selectedMission = nil; phase = .pickingOrder }
                } label: {
                    Text("Choose Another")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundStyle(LabTheme.fgSecondary)
                        .frame(maxWidth: .infinity).padding(.vertical, 11)
                }
                .buttonStyle(.pressable)
            }
            .padding(.horizontal, LabTheme.s20)
        }
        .padding(.bottom, bottomInset + LabTheme.s4)
        .background(glassSheet)


    }

    // MARK: - 3. Active Delivery Sheet

    private func activeSheet(_ mission: Mission) -> some View {
        let dist = vm.distanceToMission(mission)
        let inRange = vm.isInRange(mission)
        let order = vm.orders.first { $0.id == mission.order_id }

        return VStack(alignment: .leading, spacing: 12) {
            sheetHandle

            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 6) {
                        Circle().fill(LabTheme.live).frame(width: 7, height: 7)
                        Text("ACTIVE")
                            .font(.system(size: 9, weight: .heavy, design: .monospaced))
                            .foregroundStyle(LabTheme.live)
                    }
                    Text(mission.order_id)
                        .font(.system(size: 17, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                }
                Spacer()
                Text(mission.gateway)
                    .font(.system(size: 10, weight: .bold))
                    .foregroundStyle(LabTheme.fg)
                    .padding(.horizontal, 10).padding(.vertical, 5)
                    .background(.ultraThinMaterial, in: Capsule())
                    .overlay(Capsule().stroke(LabTheme.separator, lineWidth: 0.5))
            }
            .padding(.horizontal, LabTheme.s20)

            Rectangle().fill(LabTheme.separator).frame(height: 0.5)
                .padding(.horizontal, LabTheme.s20)

            HStack {
                HStack(spacing: 4) {
                    Image(systemName: "banknote").font(.system(size: 10))
                    Text(mission.amount.formattedAmount).font(.system(size: 12, weight: .semibold))
                }
                .foregroundStyle(LabTheme.fgSecondary)
                Spacer()
                if let d = dist {
                    HStack(spacing: 4) {
                        Circle().fill(inRange ? LabTheme.success : LabTheme.warning).frame(width: 5, height: 5)
                        Text(formattedDistance(d))
                            .font(.system(size: 12, weight: .bold, design: .monospaced))
                            .foregroundStyle(inRange ? LabTheme.success : LabTheme.fgSecondary)
                    }
                }
            }
            .padding(.horizontal, LabTheme.s20)

            // ETA Row
            if let order, let eta = order.estimatedArrivalAt {
                HStack(spacing: 5) {
                    Image(systemName: "clock.fill")
                        .font(.system(size: 10, weight: .semibold))
                        .foregroundStyle(LabTheme.fgTertiary)
                    Text("ETA \(formatETATime(eta))")
                        .font(.system(size: 12, weight: .bold, design: .monospaced))
                        .foregroundStyle(LabTheme.fg)
                    if let dur = order.etaDurationSec {
                        Text("· \(formatDuration(dur))")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                    if let distM = order.etaDistanceM {
                        Text("· \(formatETADistance(distM))")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(LabTheme.fgSecondary)
                    }
                }
                .padding(.horizontal, LabTheme.s20)
            }

            VStack(spacing: 8) {
                // Navigate button
                Button {
                    Haptics.light()
                    openDestinationInMaps(lat: mission.target_lat, lng: mission.target_lng, name: mission.order_id)
                } label: {
                    HStack(spacing: 7) {
                        Image(systemName: "arrow.triangle.turn.up.right.diamond.fill")
                            .font(.system(size: 13, weight: .semibold))
                        Text("Navigate in Maps")
                            .font(.system(size: 14, weight: .bold))
                    }
                    .foregroundStyle(LabTheme.buttonFg)
                    .frame(maxWidth: .infinity).padding(.vertical, 14)
                    .background(LabTheme.fg, in: .rect(cornerRadius: 14))
                }
                .buttonStyle(.pressable)

                Button {
                    Haptics.medium()
                    if inRange { navPath.append("scanner") }
                    else { Haptics.warning() }
                } label: {
                    HStack(spacing: 7) {
                        Image(systemName: inRange ? "qrcode.viewfinder" : "location.north.fill")
                            .font(.system(size: 13, weight: .semibold))
                        Text(inRange ? "Scan Proof of Delivery" : "Approach Target")
                            .font(.system(size: 14, weight: .bold))
                    }
                    .foregroundStyle(inRange ? LabTheme.fg : LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity).padding(.vertical, 14)
                    .background(
                        inRange ? LabTheme.fg.opacity(0.08) : LabTheme.fg.opacity(0.04),
                        in: .rect(cornerRadius: 14)
                    )
                }
                .buttonStyle(.pressable)

                Button {
                    Haptics.light()
                    navPath.append("correction")
                } label: {
                    HStack(spacing: 4) {
                        Image(systemName: "pencil.and.list.clipboard").font(.system(size: 11))
                        Text("Delivery Correction").font(.system(size: 12, weight: .semibold))
                    }
                    .foregroundStyle(LabTheme.fgSecondary)
                    .frame(maxWidth: .infinity).padding(.vertical, 10)
                }
                .buttonStyle(.pressable)
            }
            .padding(.horizontal, LabTheme.s20)
        }
        .padding(.bottom, bottomInset + LabTheme.s4)
        .background(glassSheet)


    }

    // MARK: - Shared Components

    private var sheetHandle: some View {
        Capsule()
            .fill(LabTheme.fgTertiary.opacity(0.4))
            .frame(width: 32, height: 4)
            .frame(maxWidth: .infinity)
            .padding(.top, 10).padding(.bottom, 12)
    }

    private var glassSheet: some View {
        ZStack {
            UnevenRoundedRectangle(topLeadingRadius: LabTheme.cardRadius, topTrailingRadius: LabTheme.cardRadius, style: .continuous).fill(.ultraThinMaterial)
            UnevenRoundedRectangle(topLeadingRadius: LabTheme.cardRadius, topTrailingRadius: LabTheme.cardRadius, style: .continuous).fill(LabTheme.card.opacity(0.6))
            UnevenRoundedRectangle(topLeadingRadius: LabTheme.cardRadius, topTrailingRadius: LabTheme.cardRadius, style: .continuous).stroke(LabTheme.separator, lineWidth: 0.5)
        }
        .shadow(color: .black.opacity(cs == .dark ? 0.6 : 0.1), radius: 30, y: -8)
    }

    private var loadingRow: some View {
        HStack(spacing: 8) {
            ProgressView().tint(LabTheme.fg)
            Text("Loading...").font(.subheadline).foregroundStyle(LabTheme.fgSecondary)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 40)
    }

    private var emptyRow: some View {
        VStack(spacing: 8) {
            Image(systemName: "shippingbox").font(.system(size: 24)).foregroundStyle(LabTheme.fgTertiary)
            Text("No pending deliveries").font(.subheadline.weight(.medium)).foregroundStyle(LabTheme.fgSecondary)
        }
        .frame(maxWidth: .infinity).padding(.vertical, 40)
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
                        .fill(isSelected ? LabTheme.fg : LabTheme.fg.opacity(0.5))
                        .frame(width: isSelected ? 40 : 34, height: isSelected ? 40 : 34)
                    Image(systemName: "shippingbox.fill")
                        .font(.system(size: isSelected ? 16 : 13, weight: .semibold))
                        .foregroundStyle(LabTheme.buttonFg)
                }
                .shadow(color: .black.opacity(0.2), radius: 8, y: 4)
                .animation(Anim.snappy, value: isSelected)

                Image(systemName: "triangle.fill")
                    .font(.system(size: 7))
                    .foregroundStyle(isSelected ? LabTheme.fg : LabTheme.fg.opacity(0.5))
                    .rotationEffect(.degrees(180))
                    .offset(y: -3)
            }
        }
    }

    // MARK: - ETA Helpers

    private func formatETATime(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: iso) ?? ISO8601DateFormatter().date(from: iso) {
            let tf = DateFormatter()
            tf.dateFormat = "HH:mm"
            return tf.string(from: date)
        }
        return iso.suffix(8).prefix(5).description
    }

    private func formatDuration(_ totalSec: Int) -> String {
        let m = totalSec / 60
        if m < 60 { return "\(m)m" }
        let h = m / 60
        let rem = m % 60
        return rem > 0 ? "\(h)h \(rem)m" : "\(h)h"
    }

    private func formatETADistance(_ meters: Int) -> String {
        if meters < 1000 { return "\(meters)m" }
        let km = Double(meters) / 1000.0
        return String(format: "%.1f km", km)
    }

    // MARK: - Apple Maps Navigation

    private func openDestinationInMaps(lat: Double, lng: Double, name: String) {
        let coord = CLLocationCoordinate2D(latitude: lat, longitude: lng)
        let placemark = MKPlacemark(coordinate: coord)
        let mapItem = MKMapItem(placemark: placemark)
        mapItem.name = name
        mapItem.openInMaps(launchOptions: [
            MKLaunchOptionsDirectionsModeKey: MKLaunchOptionsDirectionsModeDriving
        ])
    }
}

#Preview {
    FleetMapView(vm: FleetViewModel())
        .modelContainer(for: OfflineDelivery.self, inMemory: true)
}

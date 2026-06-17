import SwiftUI
import MapKit

private enum TrackingLoadIssue {
    case restricted
    case offline
    case error
}

// MARK: - Delivery Map View

struct DeliveryMapView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var orders: [TrackingOrder] = []
    @State private var recentReceipts: [TrackingOrder] = []
    @State private var suppliers: [SupplierFilter] = []
    @State private var selectedSupplierIds: Set<String> = []
    @State private var isLoading = false
    @State private var activeFulfillmentCount = 0
    @State private var loadIssue: TrackingLoadIssue?
    @State private var selectedOrder: TrackingOrder?
    @State private var cameraPosition: MapCameraPosition = .region(
        MKCoordinateRegion(
            center: CLLocationCoordinate2D(latitude: 41.2995, longitude: 69.2401),
            span: MKCoordinateSpan(latitudeDelta: 0.1, longitudeDelta: 0.1)
        )
    )

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared
    private let pollingInterval: TimeInterval = 15

    private var visibleOrders: [TrackingOrder] {
        orders.filter { order in
            order.hasDriverLocation && (selectedSupplierIds.isEmpty || selectedSupplierIds.contains(order.supplierId))
        }
    }

    private var activeDeliveryCount: Int {
        if activeFulfillmentCount > 0 || orders.isEmpty {
            return activeFulfillmentCount
        }
        return orders.count
    }

    private var emptyStateMessage: String {
        switch loadIssue {
        case .restricted:
            return "Your account cannot view retailer tracking right now"
        case .offline:
            return "Live tracking is offline. Reconnect to refresh driver positions"
        case .error:
            return "Tracking data could not be loaded right now"
        case nil:
            return activeDeliveryCount > 0
                ? "Active deliveries exist, but live driver location is not available yet"
                : "No active deliveries with driver location"
        }
    }

    var body: some View {
        ZStack(alignment: .top) {
            // Map
            Map(position: $cameraPosition, selection: Binding<String?>(
                get: { selectedOrder?.orderId },
                set: { id in selectedOrder = visibleOrders.first { $0.orderId == id } }
            )) {
                ForEach(visibleOrders) { order in
                    if let lat = order.driverLatitude, let lng = order.driverLongitude {
                        Annotation(order.supplierName, coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lng)) {
                            Button {
                                Haptics.light()
                                selectedOrder = order
                            } label: {
                                DriverMarker(isGreen: order.isGreen)
                            }
                            .buttonStyle(.plain)
                            .pressable()
                            .accessibilityLabel("Driver marker for \(order.supplierName)")
                            .accessibilityHint("Opens order details")
                        }
                        .tag(order.orderId)
                    }
                }
            }
            .mapStyle(.standard)
            .mapControls {
                MapUserLocationButton()
                MapCompass()
            }
            .ignoresSafeArea(edges: .bottom)

            // Recent receipts
            if !recentReceipts.isEmpty {
                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Text("Recent receipts")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("Completed deliveries from the tracking feed.")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    ScrollView {
                        VStack(spacing: AppTheme.spacingSM) {
                            ForEach(recentReceipts.prefix(6)) { receipt in
                                HStack {
                                    VStack(alignment: .leading, spacing: 2) {
                                        Text(receipt.supplierName.isEmpty ? "Supplier" : receipt.supplierName)
                                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                        Text("#\(receipt.orderId.suffix(8))")
                                            .font(.system(.caption2, design: .monospaced))
                                            .foregroundStyle(AppTheme.textTertiary)
                                    }
                                    Spacer()
                                    Text(receipt.displayTotal)
                                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                                }
                                .padding(AppTheme.spacingMD)
                                .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: AppTheme.radiusSM))
                            }
                        }
                    }
                    .frame(maxHeight: 160)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .background(.ultraThinMaterial)
            }

            // Supplier filter chips
            if suppliers.count > 1 {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: AppTheme.spacingSM) {
                        ForEach(suppliers) { supplier in
                            SupplierChip(
                                name: supplier.name,
                                isSelected: selectedSupplierIds.isEmpty || selectedSupplierIds.contains(supplier.id),
                                onTap: { toggleSupplier(supplier.id) }
                            )
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.vertical, AppTheme.spacingSM)
                }
                .background(.ultraThinMaterial)
            }

            // Active count badge
            if activeDeliveryCount > 0 {
                VStack {
                    Spacer()
                    HStack {
                        ActiveCountBadge(count: activeDeliveryCount)
                            .padding(.leading, AppTheme.spacingLG)
                        Spacer()
                    }
                    .padding(.bottom, selectedOrder != nil ? 200 : AppTheme.spacingLG)
                }
            }

            // Loading overlay
            if isLoading && orders.isEmpty {
                Color.clear
                    .overlay { ProgressView().tint(AppTheme.accent) }
            }

            // Empty state
            if !isLoading && visibleOrders.isEmpty {
                VStack {
                    Spacer()
                    Text(emptyStateMessage)
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                        .padding()
                        .background(.ultraThinMaterial, in: .capsule)
                    Spacer()
                }
            }

            // Selected order info card
            if let order = selectedOrder {
                VStack {
                    Spacer()
                    OrderInfoCard(order: order, onDismiss: { selectedOrder = nil })
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                        .padding(.horizontal, AppTheme.spacingLG)
                        .padding(.bottom, AppTheme.spacingSM)
                }
                .animation(.spring(response: 0.35), value: selectedOrder?.orderId)
            }
        }
        .navigationTitle("Delivery Map")
        .navigationBarTitleDisplayMode(.inline)
        .task { await startPolling() }
        .task { await observeWebSocket() }
        .task(id: refreshCenter.refreshToken) { await refreshTrackingState() }
        .onChange(of: visibleOrders.count) { fitCamera() }
    }

    // MARK: - Data

    private func startPolling() async {
        while !Task.isCancelled {
            await refreshTrackingState()
            try? await Task.sleep(for: .seconds(pollingInterval))
        }
    }

    private func refreshTrackingState() async {
        await fetchTracking()
        await fetchActiveFulfillments()
    }

    private func fetchTracking() async {
        if orders.isEmpty { isLoading = true }
        defer { isLoading = false }
        do {
            let response = try await api.getTracking()
            orders = response.orders
            recentReceipts = response.recentReceipts ?? []
            loadIssue = nil
            let unique = Dictionary(grouping: response.orders, by: \.supplierId)
                .compactMap { (id, group) -> SupplierFilter? in
                    guard let name = group.first?.supplierName else { return nil }
                    return SupplierFilter(id: id, name: name)
                }
                .sorted { $0.name < $1.name }
            suppliers = unique
        } catch {
            // Keep existing data on error
            if orders.isEmpty {
                loadIssue = resolveLoadIssue(error)
            }
        }
    }

    private func fetchActiveFulfillments() async {
        do {
            let response = try await api.getActiveFulfillments()
            activeFulfillmentCount = response.count
            loadIssue = nil
        } catch {
            if orders.isEmpty && activeFulfillmentCount == 0 {
                loadIssue = resolveLoadIssue(error)
            }
        }
    }

    private func observeWebSocket() async {
        for await event in ws.eventStream() {
            switch event {
            case .orderCompleted(let e):
                orders.removeAll { $0.orderId == e.orderId }
                await fetchActiveFulfillments()
            case .driverApproaching(let orderId, _, let lat, let lng, _, _):
                if let idx = orders.firstIndex(where: { $0.orderId == orderId }) {
                    // Mutate the order in-place by creating a new copy
                    let old = orders[idx]
                    let updated = TrackingOrder(
                        orderId: old.orderId, supplierId: old.supplierId, supplierName: old.supplierName,
                        warehouseId: old.warehouseId, warehouseName: old.warehouseName,
                        driverId: old.driverId, state: old.state, totalAmount: old.totalAmount,
                        orderSource: old.orderSource, driverLatitude: lat ?? old.driverLatitude,
                        driverLongitude: lng ?? old.driverLongitude,
                        liveLocationAvailable: old.liveLocationAvailable,
                        isApproaching: true,
                        deliveryToken: old.deliveryToken, createdAt: old.createdAt, items: old.items
                    )
                    orders[idx] = updated
                }
            case .orderStatusChanged, .orderReassigned:
                await refreshTrackingState()
            default:
                break
            }
        }
    }

    private func resolveLoadIssue(_ error: Error) -> TrackingLoadIssue {
        if let apiError = error as? APIError,
           case let .serverError(statusCode, _) = apiError,
           statusCode == 401 || statusCode == 403 {
            return .restricted
        }

        if error is URLError || (error as NSError).domain == NSURLErrorDomain {
            return .offline
        }

        return .error
    }

    private func toggleSupplier(_ id: String) {
        if selectedSupplierIds.contains(id) {
            selectedSupplierIds.remove(id)
        } else {
            selectedSupplierIds.insert(id)
        }
    }

    private func fitCamera() {
        let points = visibleOrders.compactMap { order -> CLLocationCoordinate2D? in
            guard let lat = order.driverLatitude, let lng = order.driverLongitude else { return nil }
            return CLLocationCoordinate2D(latitude: lat, longitude: lng)
        }
        guard !points.isEmpty else { return }
        if points.count == 1 {
            withAnimation {
                cameraPosition = .region(MKCoordinateRegion(
                    center: points[0],
                    span: MKCoordinateSpan(latitudeDelta: 0.02, longitudeDelta: 0.02)
                ))
            }
        } else {
            var minLat = points[0].latitude, maxLat = points[0].latitude
            var minLng = points[0].longitude, maxLng = points[0].longitude
            for p in points {
                minLat = min(minLat, p.latitude); maxLat = max(maxLat, p.latitude)
                minLng = min(minLng, p.longitude); maxLng = max(maxLng, p.longitude)
            }
            let center = CLLocationCoordinate2D(latitude: (minLat + maxLat) / 2, longitude: (minLng + maxLng) / 2)
            let span = MKCoordinateSpan(latitudeDelta: (maxLat - minLat) * 1.4 + 0.01, longitudeDelta: (maxLng - minLng) * 1.4 + 0.01)
            withAnimation {
                cameraPosition = .region(MKCoordinateRegion(center: center, span: span))
            }
        }
    }
}

// MARK: - Supporting Types

private struct SupplierFilter: Identifiable {
    let id: String
    let name: String
}

// MARK: - Driver Marker

private struct DriverMarker: View {
    let isGreen: Bool

    var body: some View {
        ZStack {
            Circle()
                .fill(isGreen ? AppTheme.success : AppTheme.accent)
                .frame(width: 32, height: 32)
            Image(systemName: "box.truck.fill")
                .font(.system(size: 14, weight: .bold))
                .foregroundStyle(.white)
        }
        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
    }
}

// MARK: - Supplier Chip

private struct SupplierChip: View {
    let name: String
    let isSelected: Bool
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            Text(name)
                .font(.system(.caption, design: .rounded, weight: .semibold))
                .foregroundStyle(isSelected ? .white : AppTheme.textPrimary)
                .padding(.horizontal, AppTheme.spacingMD)
                .padding(.vertical, AppTheme.spacingSM)
                .background(isSelected ? AppTheme.accent : AppTheme.surfaceElevated)
                .clipShape(.capsule)
        }
    }
}

// MARK: - Active Count Badge

private struct ActiveCountBadge: View {
    let count: Int

    var body: some View {
        HStack(spacing: 4) {
            Image(systemName: "box.truck.fill")
                .font(.system(size: 12, weight: .semibold))
            Text("\(count) active")
                .font(.system(.caption, design: .rounded, weight: .bold))
        }
        .foregroundStyle(AppTheme.textPrimary)
        .padding(.horizontal, AppTheme.spacingMD)
        .padding(.vertical, AppTheme.spacingSM)
        .background(.ultraThinMaterial, in: .capsule)
    }
}

// MARK: - Order Info Card

private struct OrderInfoCard: View {
    let order: TrackingOrder
    let onDismiss: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Header
            HStack {
                Circle()
                    .fill(order.isGreen ? AppTheme.success : AppTheme.accent)
                    .frame(width: 8, height: 8)
                Text(order.supplierName.isEmpty ? "Unknown Supplier" : order.supplierName)
                    .font(.system(.subheadline, design: .rounded, weight: .bold))
                    .foregroundStyle(AppTheme.textPrimary)
                    .lineLimit(1)
                Spacer()
                HStack(spacing: 4) {
                    Circle().fill(order.isGreen ? AppTheme.success : AppTheme.textTertiary).frame(width: 6, height: 6)
                    Text(order.state.replacingOccurrences(of: "_", with: " "))
                        .font(.system(size: 11, weight: .bold, design: .rounded))
                        .foregroundStyle(order.isGreen ? AppTheme.success : AppTheme.textTertiary)
                }
                .padding(.horizontal, 8).padding(.vertical, 4)
                .background(AppTheme.surfaceElevated)
                .clipShape(.capsule)
            }

            // Items
            Text(order.items.map { "\($0.productName) ×\($0.quantity)" }.joined(separator: ", "))
                .font(.system(.caption, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
                .lineLimit(2)

            // Total
            Text(order.displayTotal)
                .font(.system(.caption, design: .rounded, weight: .bold))
                .foregroundStyle(AppTheme.textPrimary)
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
        .onTapGesture {} // Prevent pass-through
        .gesture(DragGesture(minimumDistance: 20, coordinateSpace: .local)
            .onEnded { value in
                if value.translation.height > 50 { onDismiss() }
            })
    }
}

// TrackingOrder uses its synthesized memberwise init

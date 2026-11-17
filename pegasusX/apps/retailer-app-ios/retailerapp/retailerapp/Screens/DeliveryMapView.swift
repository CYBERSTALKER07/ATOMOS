import MapKit
import SwiftUI

// MARK: - Delivery Map View

struct DeliveryMapView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var network = NetworkLifecycleEngine.shared
    @State private var vm = DeliveryMapViewModel()

    var body: some View {
        @Bindable var vm = vm
        ZStack(alignment: .top) {
            // Map
            Map(position: $vm.cameraPosition, selection: $vm.selectedOrderId) {
                ForEach(vm.visibleOrders) { order in
                    if let geometry = order.routeGeometry, geometry.coordinates.count >= 2 {
                        let coords = geometry.coordinates.map {
                            CLLocationCoordinate2D(latitude: $0.lat, longitude: $0.lng)
                        }
                        MapPolyline(coordinates: coords)
                            .stroke(.blue, lineWidth: 4)
                    }
                    if let lat = order.driverLatitude, let lng = order.driverLongitude {
                        Annotation(order.supplierName, coordinate: CLLocationCoordinate2D(latitude: lat, longitude: lng)) {
                            Button {
                                Haptics.light()
                                vm.selectedOrder = order
                            } label: {
                                DriverMarker(isGreen: order.isGreen)
                                    .pegasusMinimumHitTarget()
                            }
                            .buttonStyle(.pressable)
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

            // Recent receipts (ADR-009: fiscal label + QR when available)
            if !vm.recentReceipts.isEmpty {
                VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                    Text("retailer_desktop.tracking.tracking_status.text.recent_receipts")
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text("mobile_retailer.ui.completed_deliveries_tap_for_fiscal_qr_when_available")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)

                    ScrollView {
                        VStack(spacing: AppTheme.spacingSM) {
                            ForEach(vm.recentReceipts.prefix(6)) { receipt in
                                Button {
                                    if !receipt.fiscalQr.isEmpty {
                                        Haptics.light()
                                        vm.fiscalQRReceipt = receipt
                                    }
                                } label: {
                                    HStack {
                                        VStack(alignment: .leading, spacing: 2) {
                                            Text(receipt.supplierName.isEmpty ? "Supplier" : receipt.supplierName)
                                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                            Text(L10n.format("mobile_retailer.ui.suffix_fiscalreceiptlabel", "\(receipt.orderId.suffix(8))", "\(receipt.fiscalReceiptLabel)"))
                                                .font(.system(.caption2, design: .monospaced))
                                                .foregroundStyle(AppTheme.textTertiary)
                                        }
                                        Spacer()
                                        if !receipt.fiscalQr.isEmpty {
                                            Image(systemName: "qrcode")
                                                .font(.subheadline.weight(.semibold))
                                                .foregroundStyle(AppTheme.accent)
                                        }
                                        Text(receipt.displayTotal)
                                            .font(.system(.subheadline, design: .rounded, weight: .bold))
                                    }
                                    .padding(AppTheme.spacingMD)
                                    .background(.ultraThinMaterial, in: RoundedRectangle(cornerRadius: AppTheme.radiusSM))
                                    .pegasusMinimumHitTarget()
                                }
                                .buttonStyle(.plain)
                                .accessibilityElement(children: .combine)
                                .accessibilityLabel("\(receipt.supplierName.isEmpty ? "Supplier" : receipt.supplierName), receipt \(receipt.fiscalReceiptLabel), \(receipt.displayTotal)")
                            }
                        }
                    }
                    .scrollDismissesKeyboard(.interactively)
                    .frame(maxHeight: 160)
                }
                .padding(.horizontal, AppTheme.spacingLG)
                .padding(.top, AppTheme.spacingSM)
                .background(.ultraThinMaterial)
            }

            // Supplier filter chips
            if vm.suppliers.count > 1 {
                ScrollView(.horizontal, showsIndicators: false) {
                    HStack(spacing: AppTheme.spacingSM) {
                        ForEach(vm.suppliers) { supplier in
                            SupplierChip(
                                name: supplier.name,
                                isSelected: vm.selectedSupplierIds.isEmpty || vm.selectedSupplierIds.contains(supplier.id),
                                onTap: { vm.toggleSupplier(supplier.id) }
                            )
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.vertical, AppTheme.spacingSM)
                }
                .background(.ultraThinMaterial)
            }

            // Active count badge
            if vm.activeDeliveryCount > 0 {
                VStack {
                    Spacer()
                    HStack {
                        ActiveCountBadge(count: vm.activeDeliveryCount)
                            .padding(.leading, AppTheme.spacingLG)
                        Spacer()
                    }
                    .padding(.bottom, vm.selectedOrder != nil ? 200 : AppTheme.spacingLG)
                }
            }

            // Loading overlay
            if vm.isLoading && vm.orders.isEmpty {
                Color.clear
                    .overlay { ProgressView().tint(AppTheme.accent) }
            }

            // Empty state
            if !vm.isLoading && vm.visibleOrders.isEmpty {
                VStack {
                    Spacer()
                    Text(vm.emptyStateMessage)
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                        .padding()
                        .background(.ultraThinMaterial, in: .capsule)
                        .accessibilityElement(children: .combine)
                    Spacer()
                }
            }

            // Selected order info card
            if let order = vm.selectedOrder {
                VStack {
                    Spacer()
                    OrderInfoCard(order: order, onDismiss: { vm.selectedOrder = nil })
                        .transition(.move(edge: .bottom).combined(with: .opacity))
                        .padding(.horizontal, AppTheme.spacingLG)
                        .padding(.bottom, AppTheme.spacingSM)
                }
                .animation(.spring(response: 0.35), value: vm.selectedOrder?.orderId)
            }
        }
        .navigationTitle(network.isOnline ? "mobile_retailer.ui.delivery_map" : "Deliveries (Offline)")
        .navigationBarTitleDisplayMode(.inline)
        .task { await vm.startPolling() }
        .task { await vm.observeWebSocket() }
        .task(id: refreshCenter.refreshToken) { await vm.refreshTrackingState() }
        .onChange(of: network.syncEpoch) { _, _ in
            Task { await vm.refreshTrackingState() }
        }
        .onChange(of: vm.visibleOrders.count) {
            if !vm.hasFittedInitialCamera {
                vm.fitCamera()
            }
        }
        .onChange(of: vm.selectedSupplierIds) { vm.fitCamera() }
        .sheet(item: $vm.fiscalQRReceipt) { receipt in
            NavigationStack {
                VStack(spacing: AppTheme.spacingLG) {
                    Text("mobile_retailer.ui.pegasus_receipt")
                        .font(.system(.title3, design: .rounded, weight: .bold))
                    Text(receipt.fiscalReceiptLabel)
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                    if !receipt.fiscalQr.isEmpty {
                        QRCodeView(data: receipt.fiscalQr, size: 220)
                    }
                    if !receipt.latestFiscalReceiptId.isEmpty {
                        Text(L10n.format("mobile_retailer.ui.id_latestfiscalreceiptid", "\(receipt.latestFiscalReceiptId)"))
                            .font(.system(.caption, design: .monospaced))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    HStack(spacing: AppTheme.spacingMD) {
                        if let html = vm.receiptHTMLURL(for: receipt) {
                            Link("Open receipt", destination: html)
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .pegasusMinimumHitTarget()
                        }
                        if let pdf = vm.receiptPDFURL(for: receipt) {
                            Link("PDF", destination: pdf)
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .pegasusMinimumHitTarget()
                        }
                    }
                    Spacer()
                }
                .padding(AppTheme.spacingLG)
                .navigationBarTitleDisplayMode(.inline)
                .toolbar {
                    ToolbarItem(placement: .cancellationAction) {
                        Button("common.action.close") { vm.fiscalQRReceipt = nil }
                            .pegasusMinimumHitTarget()
                    }
                }
            }
            .presentationDetents([.medium])
        }
    }
}

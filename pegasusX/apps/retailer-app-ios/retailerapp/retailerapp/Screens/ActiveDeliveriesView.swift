import SwiftUI

struct ActiveDeliveriesView: View {
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var orders: [Order] = []
    @State private var isLoading = false
    @State private var loadError = false
    @State private var selectedOrder: Order?
    @State private var qrOverlayOrder: Order?
    @State private var showMap = false
    @State private var approachingOrderIds: Set<String> = []

    private let api = APIClient.shared
    private let ws = RetailerWebSocket.shared

    var body: some View {
        ZStack {
            ScrollView {
                if isLoading {
                    ProgressView()
                        .frame(maxWidth: .infinity, minHeight: 200)
                        .tint(AppTheme.accent)
                } else if orders.isEmpty {
                    emptyState
                } else {
                    LazyVStack(spacing: AppTheme.spacingMD) {
                        ForEach(Array(orders.enumerated()), id: \.element.id) { index, order in
                            DeliveryCard(
                                order: order,
                                isApproaching: approachingOrderIds.contains(order.id),
                                onTap: { selectedOrder = order },
                                onShowQR: { qrOverlayOrder = order }
                            )
                            .staggeredSlideIn(index: index)
                        }
                    }
                    .padding(.horizontal, AppTheme.spacingLG)
                    .padding(.top, AppTheme.spacingSM)
                    .padding(.bottom, AppTheme.spacingXXL)
                }
            }
            .scrollIndicators(.hidden)
            .background(AppTheme.background)
            .task { await loadOrders() }
            .task(id: refreshCenter.refreshToken) { await loadOrders() }
            .refreshable { await loadOrders() }
            .alert("Failed to Load", isPresented: $loadError) {
                Button("common.action.retry") { Task { await loadOrders() } }
                Button("OK", role: .cancel) {}
            } message: {
                Text("mobile_retailer.ui.could_not_load_deliveries_check_your_connection")
            }

            // 3/4 Detail Sheet
            .sheet(item: $selectedOrder) { order in
                OrderDetailSheet(order: order, onCancelled: {
                    orders.removeAll { $0.id == order.id }
                })
                .presentationDetents([.fraction(0.75)])
                .presentationDragIndicator(.visible)
            }

            // Quick QR Overlay — guard: only if token exists and driver nearby
            if let qrOrder = qrOverlayOrder, qrOrder.status.hasDeliveryToken,
               approachingOrderIds.contains(qrOrder.id) || qrOrder.status == .arrived {
                QROverlay(order: qrOrder) {
                    withAnimation(AnimationConstants.fluid) {
                        qrOverlayOrder = nil
                    }
                }
                .transition(.opacity)
                .zIndex(200)
            }
        }
        .animation(AnimationConstants.fluid, value: qrOverlayOrder?.id)
        .task { await listenForApproaching() }
    }

    // MARK: - WebSocket Listener

    private func listenForApproaching() async {
        for await event in ws.eventStream() {
            if case .driverApproaching(let orderId, _, _, _, _, _) = event {
                approachingOrderIds.insert(orderId)
                if let order = orders.first(where: { $0.id == orderId && $0.status.hasDeliveryToken }) {
                    qrOverlayOrder = order
                }
            }
        }
    }

    // MARK: - Empty State

    private var emptyState: some View {
        VStack(spacing: AppTheme.spacingLG) {
            Spacer(minLength: 100)
            ZStack {
                Circle().fill(AppTheme.surfaceElevated).frame(width: 80, height: 80)
                Image(systemName: "shippingbox").font(.system(size: 32)).foregroundStyle(AppTheme.textTertiary)
            }
            Text("mobile_retailer.ui.no_active_orders")
                .font(.system(.headline, design: .rounded))
                .foregroundStyle(AppTheme.textPrimary)
            Text("mobile_retailer.ui.your_en_route_and_confirmed_deliveries_will_appear_here")
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
                .multilineTextAlignment(.center)
            Spacer()
        }
        .padding(AppTheme.spacingXL)
    }

    // MARK: - API

    private func loadOrders() async {
        let rid = AuthManager.shared.currentUser?.id ?? ""
        isLoading = true
        do {
            let result: [Order] = try await api.get(path: "/v1/retailers/\(rid)/orders")
            orders = result.filter { $0.status.isActive }
        } catch {
            orders = []
            loadError = true
        }
        isLoading = false
    }
}

// MARK: - Delivery Card

struct DeliveryCard: View {
    let order: Order
    var isApproaching: Bool = false
    let onTap: () -> Void
    let onShowQR: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            // Header
            HStack(alignment: .top) {
                ZStack {
                    Circle()
                        .fill(statusColor.opacity(0.1))
                        .frame(width: 42, height: 42)
                    Image(systemName: statusIcon)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(statusColor)
                }

                VStack(alignment: .leading, spacing: 3) {
                    if let supplierName = order.supplierName, !supplierName.isEmpty {
                        Text(supplierName)
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.accent)
                    }
                    Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                        .font(.system(.subheadline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Text(L10n.format("mobile_retailer.ui.itemcount_items_displaytotal", "\(order.itemCount)", "\(order.displayTotal)"))
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                }

                Spacer()

                // Status badge
                HStack(spacing: 4) {
                    Circle().fill(statusColor).frame(width: 6, height: 6)
                    Text(order.status.displayName)
                        .font(.system(size: 11, weight: .bold, design: .rounded))
                }
                .foregroundStyle(statusColor)
                .padding(.horizontal, 10).padding(.vertical, 5)
                .background(statusColor.opacity(0.08))
                .clipShape(.capsule)
            }

            // ETA countdown
            if let eta = order.estimatedDelivery {
                HStack(spacing: AppTheme.spacingSM) {
                    Image(systemName: "clock")
                        .font(.system(size: 12, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                    CountdownText(targetISO: eta, font: .system(.caption, design: .monospaced, weight: .bold), color: AppTheme.textPrimary)
                    Text("mobile_retailer.ui.until_arrival")
                        .font(.system(.caption2, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                    Spacer()
                }
                .padding(AppTheme.spacingSM)
                .background(AppTheme.surfaceElevated)
                .clipShape(.rect(cornerRadius: AppTheme.radiusSM))
            }

            // Quick items preview
            HStack(spacing: 6) {
                ForEach(order.items.prefix(3)) { item in
                    Text(item.productName.split(separator: " ").first.map(String.init) ?? "")
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.textSecondary)
                        .padding(.horizontal, 8).padding(.vertical, 4)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.capsule)
                }
                if order.items.count > 3 {
                    Text(L10n.format("mobile_retailer.ui.count_3", "\(order.items.count - 3)"))
                        .font(.system(.caption2, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textTertiary)
                }
            }

            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)

            // Actions row
            HStack(spacing: AppTheme.spacingMD) {
                // View Details
                Button(action: onTap) {
                    HStack(spacing: 4) {
                        Image(systemName: "doc.text")
                            .font(.system(size: 12, weight: .semibold))
                        Text("mobile_retailer.ui.details")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textPrimary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }

                // Quick QR — available when driver is nearby or arrived
                let qrUnlocked = isApproaching || order.status == .arrived
                if order.status.hasDeliveryToken && qrUnlocked {
                    Button(action: onShowQR) {
                        HStack(spacing: 4) {
                            Image(systemName: "qrcode")
                                .font(.system(size: 12, weight: .semibold))
                            Text("mobile_retailer.ui.show_qr")
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                        }
                        .foregroundStyle(.white)
                        .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                        .background(AppTheme.accent)
                        .clipShape(.capsule)
                    }
                } else if order.status.hasDeliveryToken {
                    HStack(spacing: 4) {
                        Image(systemName: "qrcode")
                            .font(.system(size: 12, weight: .semibold))
                        Text("mobile_retailer.ui.awaiting_driver")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                } else {
                    HStack(spacing: 4) {
                        Image(systemName: "qrcode")
                            .font(.system(size: 12, weight: .semibold))
                        Text("mobile_retailer.ui.awaiting_dispatch")
                            .font(.system(.caption, design: .rounded, weight: .semibold))
                    }
                    .foregroundStyle(AppTheme.textTertiary)
                    .padding(.horizontal, AppTheme.spacingMD).padding(.vertical, AppTheme.spacingSM)
                    .background(AppTheme.surfaceElevated)
                    .clipShape(.capsule)
                }

                Spacer()
            }
        }
        .padding(AppTheme.spacingLG)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }

    private var statusColor: Color {
        switch order.status {
        case .pending: AppTheme.warning
        case .loaded: AppTheme.info
        case .dispatched: AppTheme.accent
        case .inTransit: AppTheme.success
        case .arrived: AppTheme.success
        case .completed: AppTheme.success
        case .cancelled: AppTheme.destructive
        case .awaitingPayment: AppTheme.warning
        case .pendingCashCollection: AppTheme.warning
        default: AppTheme.textSecondary
        }
    }

    private var statusIcon: String {
        switch order.status {
        case .pending: "clock"
        case .loaded: "shippingbox"
        case .dispatched: "truck"
        case .inTransit: "shippingbox.fill"
        case .arrived: "checkmark.circle"
        case .completed: "checkmark.circle.fill"
        case .cancelled: "xmark.circle"
        case .awaitingPayment: "creditcard"
        case .pendingCashCollection: "banknote"
        default: "circle"
        }
    }
}

// MARK: - Order Detail Sheet (3/4 height)

struct OrderDetailSheet: View {
    let order: Order
    var onCancelled: (() -> Void)? = nil
    @Environment(\.dismiss) private var dismiss

    @State private var showCancelConfirm = false
    @State private var isCancelling = false
    @State private var cancelError = false
    @State private var showFileClaim = false
    @State private var claimElig: ClaimEligibility?
    private let api = APIClient.shared

    private var claimCandidate: Bool {
        order.status == .completed || order.status == .deliveredOnCredit
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: AppTheme.spacingLG) {
                    // Status header
                    HStack {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                                .font(.system(.title3, design: .rounded, weight: .bold))
                                .foregroundStyle(AppTheme.textPrimary)
                            Text(order.status.displayName)
                                .font(.system(.subheadline, design: .rounded, weight: .medium))
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        Spacer()
                        Text(order.displayTotal)
                            .font(.system(.title2, design: .rounded, weight: .bold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    .slideIn(delay: 0)

                    // ETA
                    if let eta = order.estimatedDelivery {
                        LabCard {
                            VStack(spacing: AppTheme.spacingSM) {
                                Text("mobile_retailer.ui.estimated_arrival")
                                    .font(.system(.caption, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                                CountdownText(targetISO: eta, font: .system(.title2, design: .monospaced, weight: .bold), color: AppTheme.textPrimary)
                            }
                            .frame(maxWidth: .infinity)
                            .padding(AppTheme.spacingLG)
                        }
                        .slideIn(delay: 0.05)
                    }

                    // Line items
                    LabCardWithHeader(title: "Items", subtitle: "\(order.itemCount) items", icon: "list.bullet") {
                        VStack(spacing: 0) {
                            ForEach(Array(order.items.enumerated()), id: \.element.id) { index, item in
                                if index > 0 {
                                    Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)
                                }
                                HStack {
                                    Text(item.productName)
                                        .font(.system(.subheadline, design: .rounded))
                                        .foregroundStyle(AppTheme.textPrimary)
                                    Spacer()
                                    Text(L10n.format("mobile_retailer.ui.quantity", "\(item.quantity)"))
                                        .font(.system(.caption, design: .rounded, weight: .bold))
                                        .foregroundStyle(AppTheme.textTertiary)
                                        .padding(.horizontal, 8).padding(.vertical, 3)
                                        .background(AppTheme.surfaceElevated)
                                        .clipShape(.capsule)
                                }
                                .padding(.vertical, AppTheme.spacingSM)
                            }
                        }
                    }
                    .slideIn(delay: 0.1)

                    // Logistics fee
                    LabCard {
                        VStack(spacing: AppTheme.spacingMD) {
                            summaryRow("Subtotal", value: order.displayTotal)
                            summaryRow("Logistics Fee", value: "$2.50")
                            Rectangle().fill(AppTheme.separator.opacity(0.3)).frame(height: AppTheme.separatorHeight)
                            HStack {
                                Text("retailer_desktop.pos.text.total")
                                    .font(.system(.headline, design: .rounded))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Spacer()
                                Text(order.displayTotal)
                                    .font(.system(.title3, design: .rounded, weight: .bold))
                                    .foregroundStyle(AppTheme.textPrimary)
                            }
                        }
                        .padding(AppTheme.spacingLG)
                    }
                    .slideIn(delay: 0.15)

                    // Show QR — JIT: only after dispatch
                    if order.status.hasDeliveryToken, let qrData = order.deliveryQRCodePayload {
                        VStack(spacing: AppTheme.spacingSM) {
                            QRCodeView(data: qrData, size: 180)
                            Text("mobile_retailer.ui.show_this_qr_code_to_the_driver")
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(AppTheme.spacingLG)
                        .background(AppTheme.cardBackground)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                        .shadow(color: AppTheme.shadowColor, radius: 4, y: 2)
                        .slideIn(delay: 0.2)
                    } else if !order.status.hasDeliveryToken {
                        VStack(spacing: AppTheme.spacingSM) {
                            Image(systemName: "lock.fill")
                                .font(.system(size: 28))
                                .foregroundStyle(AppTheme.textTertiary)
                            Text("mobile_retailer.ui.awaiting_dispatch")
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.textSecondary)
                            Text("mobile_retailer.ui.qr_code_will_appear_when_your_order_is_on_the_way")
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                                .multilineTextAlignment(.center)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(AppTheme.spacingLG)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                        .slideIn(delay: 0.2)
                    }

                    OrderStatusHistorySection(orderId: order.id)
                        .slideIn(delay: 0.18)

                    // Post-delivery claims — gated by claim-eligibility countdown (G2)
                    if claimCandidate {
                        if let claimElig, claimElig.eligible {
                            Text(L10n.format("mobile_retailer.ui.eligible_until_formatclaimendsat_hoursremainingh_left", "\(formatClaimEndsAt(claimElig.endsAt))", "\(claimElig.hoursRemaining)"))
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textTertiary)
                            Button { showFileClaim = true } label: {
                                HStack(spacing: AppTheme.spacingSM) {
                                    Image(systemName: "exclamationmark.bubble")
                                    Text("mobile_retailer.ui.report_damage_or_missing_items")
                                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, AppTheme.spacingMD)
                                .foregroundStyle(AppTheme.textPrimary)
                                .overlay(
                                    RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                                        .stroke(AppTheme.separator.opacity(0.5), lineWidth: 1)
                                )
                            }
                            .slideIn(delay: 0.22)
                        } else if let claimElig, !claimElig.eligible {
                            Text(
                                "Window closed — claim filing unavailable" +
                                    (claimElig.endsAt.map { " (ended \(formatClaimEndsAt($0)))" } ?? "")
                            )
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                            .slideIn(delay: 0.22)
                        } else {
                            Button { showFileClaim = true } label: {
                                HStack(spacing: AppTheme.spacingSM) {
                                    Image(systemName: "exclamationmark.bubble")
                                    Text("mobile_retailer.ui.report_damage_or_missing_items")
                                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                }
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, AppTheme.spacingMD)
                                .foregroundStyle(AppTheme.textPrimary)
                                .overlay(
                                    RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                                        .stroke(AppTheme.separator.opacity(0.5), lineWidth: 1)
                                )
                            }
                            .slideIn(delay: 0.22)
                        }
                    }

                    // Cancel action — permitted for cancellable states
                    if order.status.canCancel {
                        Button { showCancelConfirm = true } label: {
                            HStack(spacing: AppTheme.spacingSM) {
                                if isCancelling {
                                    ProgressView()
                                        .tint(AppTheme.destructive)
                                        .controlSize(.small)
                                } else {
                                    Image(systemName: "xmark.circle")
                                }
                                Text(isCancelling ? "Cancelling\u{2026}" : "Cancel Order")
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            }
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, AppTheme.spacingMD)
                            .foregroundStyle(AppTheme.destructive)
                            .overlay(
                                RoundedRectangle(cornerRadius: AppTheme.radiusCard)
                                    .stroke(AppTheme.destructive.opacity(0.4), lineWidth: 1)
                            )
                        }
                        .disabled(isCancelling)
                        .slideIn(delay: 0.25)
                    } else if order.status != .cancelled && order.status != .completed {
                        HStack(spacing: AppTheme.spacingSM) {
                            Image(systemName: "exclamationmark.triangle")
                                .font(.system(.caption))
                            Text("mobile_retailer.ui.order_in_progress_cannot_be_cancelled")
                                .font(.system(.caption, design: .rounded))
                        }
                        .foregroundStyle(AppTheme.textTertiary)
                        .frame(maxWidth: .infinity, alignment: .center)
                        .padding(.vertical, AppTheme.spacingSM)
                        .slideIn(delay: 0.25)
                    }
                }
                .padding(AppTheme.spacingLG)
                .padding(.bottom, AppTheme.spacingXXL)
            }            .scrollIndicators(.hidden)            .background(AppTheme.background)
            .navigationTitle("mobile_retailer.ui.order_details")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("warehouse_portal.kpi_stat_card.text.done") { dismiss() }
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                }
            }
            .alert("Cancel Order", isPresented: $showCancelConfirm) {
                Button("mobile_retailer.ui.cancel_order", role: .destructive) { Task { await cancelOrder() } }
                Button("mobile_retailer.ui.keep_order", role: .cancel) {}
            } message: {
                Text(L10n.format("mobile_retailer.ui.cancel_order_suffix_this_cannot_be_undone", "\(order.id.suffix(3))"))
            }
            .alert("Cancellation Failed", isPresented: $cancelError) {
                Button("OK", role: .cancel) {}
            } message: {
                Text("mobile_retailer.ui.could_not_cancel_the_order_please_try_again")
            }
            .sheet(isPresented: $showFileClaim) {
                FileClaimView(order: order)
            }
            .task(id: order.id) {
                guard claimCandidate else {
                    claimElig = nil
                    return
                }
                claimElig = try? await api.getClaimEligibility(orderId: order.id)
            }
        }
    }

    private func formatClaimEndsAt(_ raw: String?) -> String {
        guard let raw, let date = ISO8601DateFormatter().date(from: raw) else {
            return raw ?? "window end"
        }
        return date.formatted(date: .abbreviated, time: .shortened)
    }

    private func cancelOrder() async {
        isCancelling = true
        defer { isCancelling = false }
        let retailerId = AuthManager.shared.currentUser?.id ?? ""
        let requestCancelStates: Set<OrderStatus> = [.dispatched, .inTransit, .arrived]
        let useRequestCancel = requestCancelStates.contains(order.status)
        do {
            if useRequestCancel {
                try await api.requestCancelOrder(orderId: order.id, retailerId: retailerId)
            } else {
                let _: [String: String] = try await api.post(
                    path: "/v1/order/cancel",
                    body: [
                        "order_id": order.id,
                        "retailer_id": retailerId,
                    ],
                    headers: ["Idempotency-Key": RetailerIdempotency.cancel(orderId: order.id)]
                )
            }
            onCancelled?()
            dismiss()
        } catch {
            cancelError = true
        }
    }

    private func summaryRow(_ title: String, value: String) -> some View {
        HStack {
            Text(title)
                .font(.system(.subheadline, design: .rounded))
                .foregroundStyle(AppTheme.textTertiary)
            Spacer()
            Text(value)
                .font(.system(.subheadline, design: .rounded, weight: .medium))
                .foregroundStyle(AppTheme.textPrimary)
        }
    }
}

// MARK: - QR Overlay (Full Screen Blurred)

struct QROverlay: View {
    let order: Order
    let onDismiss: () -> Void

    @State private var appeared = false

    var body: some View {
        ZStack {
            // Blurred background
            Color.clear
                .background(.ultraThinMaterial)
                .ignoresSafeArea()
                .onTapGesture { onDismiss() }

            // QR content
            VStack(spacing: AppTheme.spacingXL) {
                Text(L10n.format("mobile_retailer.ui.order_suffix", "\(order.id.suffix(3))"))
                    .font(.system(.headline, design: .rounded))
                    .foregroundStyle(AppTheme.textPrimary)

                if let qrData = order.deliveryQRCodePayload {
                    QRCodeView(data: qrData, size: 240)
                        .padding(AppTheme.spacingXL)
                        .background(AppTheme.cardBackground)
                        .clipShape(.rect(cornerRadius: AppTheme.radiusXL))
                        .shadow(color: AppTheme.shadowColor, radius: 16, y: 8)
                } else {
                    // No token yet — locked state
                    VStack(spacing: AppTheme.spacingSM) {
                        Image(systemName: "lock.fill")
                            .font(.system(size: 40))
                            .foregroundStyle(AppTheme.textTertiary)
                        Text("mobile_retailer.ui.token_not_yet_generated")
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .frame(width: 240, height: 240)
                    .padding(AppTheme.spacingXL)
                    .background(AppTheme.cardBackground)
                    .clipShape(.rect(cornerRadius: AppTheme.radiusXL))
                    .shadow(color: AppTheme.shadowColor, radius: 16, y: 8)
                }

                Text("mobile_retailer.ui.show_to_driver_for_delivery_confirmation")
                    .font(.system(.subheadline, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)

                Button {
                    Haptics.light()
                    onDismiss()
                } label: {
                    Text("factory_portal.toast.text.dismiss")
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textSecondary)
                        .padding(.horizontal, AppTheme.spacingXL)
                        .padding(.vertical, AppTheme.spacingMD)
                        .background(AppTheme.surfaceElevated)
                        .clipShape(.capsule)
                }
            }
            .scaleEffect(appeared ? 1 : 0.85)
            .opacity(appeared ? 1 : 0)
        }
        .onAppear {
            withAnimation(AnimationConstants.fluid) { appeared = true }
        }
    }
}

// MARK: - Order status history (GET /v1/order/{id}/timeline)

struct OrderStatusHistorySection: View {
    let orderId: String

    @State private var items: [OrderTimelineEntry] = []
    @State private var loading = true
    private let api = APIClient.shared

    var body: some View {
        LabCardWithHeader(title: "Status history", subtitle: nil, icon: "clock.arrow.circlepath") {
            Group {
                if loading {
                    Text("warehouse_portal.bins.text.loading")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                } else if items.isEmpty {
                    Text("supplier_portal.order_timeline_panel.text.no_status_history_yet")
                        .font(.system(.caption, design: .rounded))
                        .foregroundStyle(AppTheme.textTertiary)
                } else {
                    VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                        ForEach(items) { entry in
                            VStack(alignment: .leading, spacing: 4) {
                                Text(statusLabel(entry))
                                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                    .foregroundStyle(AppTheme.textPrimary)
                                Text(subtitle(for: entry))
                                    .font(.system(.caption, design: .rounded))
                                    .foregroundStyle(AppTheme.textTertiary)
                            }
                            if entry.id != items.last?.id {
                                Rectangle()
                                    .fill(AppTheme.separator.opacity(0.3))
                                    .frame(height: AppTheme.separatorHeight)
                            }
                        }
                    }
                }
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .task(id: orderId) {
            loading = true
            defer { loading = false }
            do {
                let response = try await api.getOrderTimeline(orderId: orderId)
                items = response.items
            } catch {
                items = []
            }
        }
    }

    private func statusLabel(_ entry: OrderTimelineEntry) -> String {
        if let previous = entry.previousStatus, !previous.isEmpty {
            return "\(previous) → \(entry.newStatus)"
        }
        return entry.newStatus
    }

    private func subtitle(for entry: OrderTimelineEntry) -> String {
        var parts: [String] = [formatWhen(entry.createdAt)]
        if let reason = entry.reason, !reason.isEmpty {
            parts.append(reason)
        }
        if entry.eventKind == "DELAY" {
            parts.append("Delayed")
        }
        return parts.joined(separator: " · ")
    }

    private func formatWhen(_ iso: String) -> String {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: iso) ?? ISO8601DateFormatter().date(from: iso) {
            return date.formatted(date: .abbreviated, time: .shortened)
        }
        return iso
    }
}

#Preview {
    NavigationStack {
        ActiveDeliveriesView()
    }
}

import SwiftUI

struct DashboardView: View {
    @Environment(CartManager.self) private var cart
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var vm = DashboardViewModel()
    @State private var socket = RetailerWebSocket.shared

    private var activeOrders: [Order] { vm.activeOrders }
    private var predictions: [RetailerAIPrediction] { vm.predictions }
    private var reorderProducts: [Product] { vm.reorderProducts }
    private var isLoading: Bool { vm.isLoading }
    private var actingId: String? { vm.actingId }
    private var orderActionPending: Bool {
        get { vm.orderActionPending }
        nonmutating set { vm.orderActionPending = newValue }
    }
    private var loadError: String? {
        get { vm.loadError }
        nonmutating set { vm.loadError = newValue }
    }
    private var pulseEvents: [RetailerPulseEvent] { vm.pulseEvents }
    private var pulseLoading: Bool { vm.pulseLoading }
    private var pulseError: String? { vm.pulseError }
    private var commandPulse: ControlTowerPulseWire? { vm.commandPulse }
    private var commandPulseError: String? { vm.commandPulseError }

    private let api = APIClient.shared

    private var kpiGridMin: CGFloat {
        horizontalSizeClass == .regular ? 160 : 140
    }

    var body: some View {
        @Bindable var vm = vm
        ScrollView {
            VStack(spacing: AppTheme.spacingXL) {
                if isLoading && activeOrders.isEmpty && predictions.isEmpty && reorderProducts.isEmpty {
                    RetailerLoadingView(
                        title: "Loading home",
                        message: "Fetching active orders, AI predictions, and quick reorder picks."
                    )
                } else if let loadError, activeOrders.isEmpty && predictions.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await loadData() }
                    }
                } else {
                    retailerCommandBoard
                        .slideIn(delay: 0)

                    KpiGrid(
                        activeOrdersCount: commandPulse?.openOrders ?? activeOrders.count,
                        predictionsCount: predictions.count,
                        reorderProductsCount: reorderProducts.count,
                        horizontalSizeClass: horizontalSizeClass
                    )
                    .slideIn(delay: 0)

                    PulseStripView(events: pulseEvents, loading: pulseLoading, error: pulseError)
                        .slideIn(delay: 0.02)

                    // Hero Service Grid (Yandex Go style)
                    ServiceGrid(
                        activeOrdersCount: activeOrders.count,
                        predictionsCount: predictions.count
                    )
                        .slideIn(delay: 0.05)

                    // Quick Reorder
                    QuickReorderSection(reorderProducts: reorderProducts)
                        .slideIn(delay: 0.1)

                    // AI Prediction Cards
                    AiPredictionSection(
                        predictions: predictions,
                        actingId: $vm.actingId,
                        onConfirm: { item in
                            await confirmAiOrder(item.orderId)
                        },
                        onReject: { item in
                            await rejectAiOrder(item.orderId)
                        }
                    )
                        .slideIn(delay: 0.15)
                }
            }
            .padding(.horizontal, AppTheme.spacingLG)
            .padding(.bottom, AppTheme.spacingHuge)
            .retailerReadableWidth()
        }
        .scrollIndicators(.hidden)
        .background(AppTheme.background)
        .task {
            await loadData()
        }
        .task(id: refreshCenter.refreshToken) {
            await loadData(silent: hasCachedDashboardData)
        }
        .task {
            for await event in RetailerWebSocket.shared.eventStream() {
                if case .promotionChanged = event {
                    await loadData(silent: hasCachedDashboardData)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .refreshable {
            await loadData()
        }
        .navigationDestination(item: $vm.commandJump) { jump in
            OrdersView(initialCommandStatus: jump.status, initialSupplierId: jump.supplierId)
        }
        .task {
            await vm.pollCommandPulseLoop()
        }
        .onChange(of: NetworkLifecycleEngine.shared.syncEpoch) { _, _ in
            Task { await loadData(silent: hasCachedDashboardData) }
        }
        .onChange(of: socket.reconnectEpoch) { _, _ in
            if orderActionPending {
                orderActionPending = false
                loadError = "Connection restored — verify order status before retrying."
                Task { await loadData(silent: hasCachedDashboardData) }
            }
        }
    }



    private var retailerCommandBoard: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            HStack {
                Text("Retailer command")
                    .font(.system(.headline, design: .rounded))
                Spacer()
                if commandPulseError == nil {
                    SourceChip(source: commandPulse?.source ?? "empty")
                }
            }
            if let commandPulseError {
                Text(commandPulseError)
                    .font(.footnote)
                    .foregroundStyle(AppTheme.destructive)
                    .accessibilityIdentifier("gs-u-retailer-command-error")
            } else if commandPulse?.empty == true {
                VStack(alignment: .leading, spacing: 8) {
                    Text("No live ops signals yet. Empty pulse — not demo tiles.")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.textSecondary)
                        .accessibilityIdentifier("gs-u-retailer-pulse-empty")

                    LazyVGrid(columns: [GridItem(.adaptive(minimum: 140), spacing: 8)], spacing: 8) {
                        NavigationLink {
                            OrdersView()
                        } label: {
                            commandTile("Open orders", "0")
                        }
                        .buttonStyle(.pressable)

                        NavigationLink {
                            DeliveriesHubView()
                        } label: {
                            commandTile("Fulfillment", "0")
                        }
                        .buttonStyle(.pressable)

                        NavigationLink {
                            ArrivalView()
                        } label: {
                            commandTile("Receiving", "0")
                        }
                        .buttonStyle(.pressable)

                        NavigationLink {
                            PosView()
                        } label: {
                            commandTile("POS", "0")
                        }
                        .buttonStyle(.pressable)
                    }
                }
            } else if let pulse = commandPulse {
                LazyVGrid(columns: [GridItem(.adaptive(minimum: 140), spacing: 8)], spacing: 8) {
                    NavigationLink {
                        OrdersView()
                    } label: {
                        commandTile("Open orders", "\(pulse.openOrders)")
                    }
                    .buttonStyle(.pressable)

                    NavigationLink {
                        DeliveriesHubView()
                    } label: {
                        commandTile("Fulfillment", "\(pulse.activeFulfillments)")
                    }
                    .buttonStyle(.pressable)

                    NavigationLink {
                        ArrivalView()
                    } label: {
                        commandTile("Receiving", "\(pulse.dockPending)")
                    }
                    .buttonStyle(.pressable)

                    NavigationLink {
                        PosView()
                    } label: {
                        commandTile("POS", "\(pulse.posOpenSessions)")
                    }
                    .buttonStyle(.pressable)
                }
            }
            if commandPulseError == nil {
                StatusStackView(
                    dictionary: orderStatusFunnel,
                    counts: commandPulse?.ordersByStatus,
                    source: commandPulse?.source,
                    onSelect: { commandJump($0) }
                )
                .accessibilityIdentifier("gs-u-retailer-stack")
                ForEach(Array((commandPulse?.ordersBySupplier ?? []).enumerated()), id: \.offset) { _, facet in
                    VStack(alignment: .leading, spacing: 6) {
                        Text(facet.supplierId.isEmpty ? "missing supplier" : facet.supplierId)
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)
                        StatusStackView(
                            dictionary: orderStatusFunnel,
                            counts: facet.ordersByStatus,
                            source: commandPulse?.source,
                            onSelect: { commandJump($0, supplierId: facet.supplierId) }
                        )
                    }
                    .accessibilityIdentifier("gs-u-retailer-supplier-facet")
                }
                if commandPulse?.loyalty?.enrolled != true {
                    Text("Not enrolled. No fake Bronze — supplier has not configured a program, or you have no points yet.")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.textSecondary)
                        .accessibilityIdentifier("gs-u-retailer-loyalty")
                }
            }
        }
    }

    private func commandJump(_ status: String, supplierId: String? = nil) {
        vm.commandJump = CommandStatusJump(status: status, supplierId: supplierId)
    }

    private func commandTile(_ label: String, _ value: String) -> some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.caption2)
                .foregroundStyle(AppTheme.textTertiary)
            Text(value)
                .font(.system(.headline, design: .rounded))
        }
        .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget, alignment: .leading)
        .padding(10)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .accessibilityElement(children: .combine)
        .accessibilityLabel("\(label): \(value)")
    }

    // MARK: - Active Deliveries

    private var activeDeliveriesSection: some View {
        VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
            RetailerSectionHeader(title: "Active Deliveries", icon: "shippingbox.fill", count: activeOrders.count)

            if activeOrders.isEmpty {
                emptyState(icon: "shippingbox", title: "All clear!", message: "No active deliveries right now")
            } else {
                ForEach(Array(activeOrders.enumerated()), id: \.element.id) { index, order in
                    OrderCardView(
                        order: order,
                        onCancel: {
                            Task { await cancelOrder(order.id) }
                        },
                        onConfirmAi: {
                            Task { await confirmAiOrder(order.id) }
                        },
                        onRejectAi: {
                            Task { await rejectAiOrder(order.id) }
                        },
                        onConfirmPreorder: {
                            Task { await confirmPreorder(order.id) }
                        },
                        onEditPreorder: {
                            Task { await editPreorder(order.id) }
                        }
                    )
                    .staggeredSlideIn(index: index)
                }
            }
        }
    }




    // MARK: - Helpers

    private func emptyState(icon: String, title: String, message: String) -> some View {
        VStack(spacing: AppTheme.spacingMD) {
            ZStack {
                Circle()
                    .fill(AppTheme.surfaceElevated)
                    .frame(width: 64, height: 64)
                Image(systemName: icon)
                    .font(.title2)
                    .foregroundStyle(AppTheme.textTertiary)
            }
            Text(title)
                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                .foregroundStyle(AppTheme.textPrimary)
            Text(message)
                .font(.caption)
                .foregroundStyle(AppTheme.textTertiary)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, AppTheme.spacingXXL)
        .background(AppTheme.cardBackground)
        .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
        .shadow(color: AppTheme.shadowColor, radius: AppTheme.shadowRadius, x: 0, y: AppTheme.shadowOffsetY)
    }

    private func confidenceColor(_ confidence: Double) -> Color {
        if confidence >= 0.8 { return AppTheme.success }
        if confidence >= 0.6 { return AppTheme.warning }
        return AppTheme.destructive
    }

    // MARK: - API

    private var hasCachedDashboardData: Bool {
        !activeOrders.isEmpty || !predictions.isEmpty || !reorderProducts.isEmpty
    }

    private func loadData(silent: Bool = false) async {
        await vm.loadData(silent: silent)
    }

    private func loadCommandPulse() async {
        await vm.loadCommandPulse()
    }

    private func cancelOrder(_ orderId: String) async {
        await vm.cancelOrder(orderId)
    }

    private func confirmAiOrder(_ orderId: String) async {
        await vm.confirmAiOrder(orderId)
    }

    private func rejectAiOrder(_ orderId: String) async {
        await vm.rejectAiOrder(orderId)
    }

    private func confirmPreorder(_ orderId: String) async {
        await vm.confirmPreorder(orderId)
    }

    private func editPreorder(_ orderId: String) async {
        await vm.editPreorder(orderId)
    }

    private func findOrder(_ orderId: String) async -> Order? {
        await vm.findOrder(orderId)
    }
}

private struct ServiceTile {
    enum Size { case small, regular, large }
    let title: String
    let icon: String
    let subtitle: String?
    let size: Size
}

#Preview {
    NavigationStack {
        DashboardView()
            .environment(CartManager())
    }
}

import SwiftUI

struct DashboardView: View {
    @Environment(CartManager.self) private var cart
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var refreshCenter = RetailerRefreshCenter.shared
    @State private var preferencesStore = RetailerPreferencesStore.shared
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
        let _ = preferencesStore.currentLanguage
        ScrollView {
            VStack(spacing: AppTheme.spacingXL) {
                workspaceModeSwitcherBar
                    .slideIn(delay: 0)

                if isLoading && fastHome == nil && activeOrders.isEmpty && predictions.isEmpty && reorderProducts.isEmpty {
                    RetailerLoadingView(
                        title: L10n.string("mobile_retailer.fast_home.loading"),
                        message: "Fetching active orders, AI predictions, and quick reorder picks."
                    )
                } else if let loadError, fastHome == nil && activeOrders.isEmpty && predictions.isEmpty {
                    RetailerErrorView(message: loadError) {
                        Task { await loadData() }
                    }
                } else if vm.workspaceMode == .fast {
                    fastHomeSection
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
            while !Task.isCancelled {
                try? await Task.sleep(for: .seconds(1))
                guard !Task.isCancelled else { break }
                vm.tickCountdown()
            }
        }
        .onChange(of: preferencesStore.workspaceModeOverride) { _, _ in
            vm.syncWorkspaceModeFromPreferences()
        }
        .sheet(isPresented: $vm.showEditProposalSheet) {
            if let proposal = vm.fastHome?.pendingProposal {
                FastProposalEditSheet(
                    proposal: proposal,
                    onCancel: { vm.showEditProposalSheet = false },
                    onSave: { updated in
                        await vm.editFastProposal(updatedItems: updated)
                    }
                )
            }
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

    // MARK: - W5 Fast Mode Home

    private var fastHome: RetailerFastHomeResponse? { vm.fastHome }

    private var workspaceModeSwitcherBar: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingSM) {
                HStack(spacing: AppTheme.spacingSM) {
                    workspaceToggleChip(
                        mode: .fast,
                        title: L10n.string("mobile_retailer.workspace_mode.fast"),
                        icon: "bolt.fill"
                    )
                    workspaceToggleChip(
                        mode: .full,
                        title: L10n.string("mobile_retailer.workspace_mode.full"),
                        icon: "square.grid.2x2.fill"
                    )
                }
                Text(
                    vm.workspaceMode == .fast
                        ? L10n.string("mobile_retailer.workspace_mode.hint_fast")
                        : L10n.string("mobile_retailer.workspace_mode.hint_full")
                )
                .font(.system(.caption2, design: .rounded))
                .foregroundStyle(AppTheme.textSecondary)
            }
            .padding(AppTheme.spacingMD)
        }
        .accessibilityIdentifier("dashboard-workspace-switcher")
    }

    private func workspaceToggleChip(mode: RetailerWorkspaceMode, title: String, icon: String) -> some View {
        let selected = vm.workspaceMode == mode
        return Button {
            Haptics.selection()
            vm.switchWorkspaceMode(mode)
        } label: {
            HStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.system(.caption, weight: .semibold))
                Text(title)
                    .font(.system(.subheadline, design: .rounded, weight: .semibold))
                    .lineLimit(1)
            }
            .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
            .background(selected ? AppTheme.accent : AppTheme.surfaceElevated)
            .foregroundStyle(selected ? .white : AppTheme.textPrimary)
            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
        }
        .buttonStyle(.plain)
        .accessibilityIdentifier("workspace-mode-\(mode.rawValue)")
    }

    private var fastHomeSection: some View {
        VStack(spacing: AppTheme.spacingLG) {
            if let approvals = fastHome?.pendingApprovalsCount, approvals > 0 {
                NavigationLink {
                    OwnerApprovalsView()
                } label: {
                    HStack {
                        Image(systemName: "person.badge.shield.checkmark")
                            .foregroundStyle(AppTheme.warning)
                        Text(L10n.format("mobile_retailer.fast_home.approvals_banner", "\(approvals)"))
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Spacer()
                        Image(systemName: "chevron.right")
                            .font(.caption)
                            .foregroundStyle(AppTheme.textTertiary)
                    }
                    .padding(AppTheme.spacingMD)
                    .background(AppTheme.warning.opacity(0.12))
                    .clipShape(.rect(cornerRadius: AppTheme.radiusCard))
                }
                .buttonStyle(.plain)
            }

            tomorrowDeliveryCard
            whatWeOweCard
            todaysProposalCard
        }
    }

    private var tomorrowDeliveryCard: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack {
                    Label(L10n.string("mobile_retailer.fast_home.tomorrow_delivery"), systemImage: "box.truck.fill")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Spacer()
                    if let delivery = fastHome?.tomorrowDelivery {
                        Text(delivery.status.replacingOccurrences(of: "_", with: " "))
                            .font(.system(.caption2, design: .rounded, weight: .bold))
                            .padding(.horizontal, 8)
                            .padding(.vertical, 4)
                            .background(AppTheme.info.opacity(0.14))
                            .foregroundStyle(AppTheme.info)
                            .clipShape(Capsule())
                    }
                }

                if let delivery = fastHome?.tomorrowDelivery {
                    if let supplier = delivery.supplierName, !supplier.isEmpty {
                        Text(supplier)
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }
                    if let eta = delivery.etaText ?? fastHome?.nextDeliveryDate, !eta.isEmpty {
                        Text(L10n.format("mobile_retailer.fast_home.delivery_arriving", String(eta.prefix(16))))
                            .font(.system(.caption, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                    }
                    HStack {
                        let countLabel = L10n.format("mobile_retailer.fast_home.delivery_items_count", "\(delivery.itemsCount)")
                        let totalFormatted = RetailerMoney.formatWholeCurrency(delivery.totalMinor, currency: delivery.currency)
                        Text(L10n.format("mobile_retailer.fast_home.delivery_meta", countLabel, totalFormatted))
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                        Spacer()
                        NavigationLink {
                            OrdersView()
                        } label: {
                            Text(L10n.string("mobile_retailer.fast_home.view_orders"))
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.accent)
                        }
                    }
                } else {
                    HStack {
                        Text(L10n.string("mobile_retailer.fast_home.delivery_empty"))
                            .font(.system(.subheadline, design: .rounded))
                            .foregroundStyle(AppTheme.textSecondary)
                        Spacer()
                        NavigationLink {
                            OrdersView()
                        } label: {
                            Text(L10n.string("mobile_retailer.fast_home.view_orders"))
                                .font(.system(.caption, design: .rounded, weight: .semibold))
                                .foregroundStyle(AppTheme.accent)
                        }
                    }
                }
            }
            .padding(AppTheme.spacingLG)
        }
        .accessibilityIdentifier("fast-home-tomorrow-delivery")
    }

    private var whatWeOweCard: some View {
        let balance = fastHome?.balanceOwedMinor ?? 0
        let overdue = fastHome?.overdueMinor ?? 0
        let daysOverdue = fastHome?.daysOverdue ?? 0
        let headroom = fastHome?.creditHeadroomMinor ?? 0

        return LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack {
                    Label(L10n.string("mobile_retailer.fast_home.what_we_owe"), systemImage: "creditcard.fill")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Spacer()
                    NavigationLink {
                        CreditPartnersView()
                    } label: {
                        Text(L10n.string("mobile_retailer.fast_home.pay_now"))
                            .font(.system(.caption, design: .rounded, weight: .bold))
                            .padding(.horizontal, 12)
                            .padding(.vertical, 8)
                            .background(AppTheme.accent)
                            .foregroundStyle(.white)
                            .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                    }
                }

                Text(RetailerMoney.formatWholeCurrency(balance, currency: "UZS"))
                    .font(.system(.title, design: .rounded, weight: .bold))
                    .foregroundStyle(overdue > 0 ? AppTheme.destructive : AppTheme.textPrimary)
                    .accessibilityIdentifier("fast-home-balance-owed")

                HStack(spacing: AppTheme.spacingMD) {
                    if overdue > 0 {
                        Text(
                            L10n.format(
                                "mobile_retailer.fast_home.overdue_days",
                                "\(daysOverdue)",
                                RetailerMoney.formatWholeCurrency(overdue, currency: "UZS")
                            )
                        )
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.destructive)
                    }
                    Text(
                        L10n.format(
                            "mobile_retailer.fast_home.credit_headroom",
                            RetailerMoney.formatWholeCurrency(headroom, currency: "UZS")
                        )
                    )
                    .font(.system(.caption, design: .rounded))
                    .foregroundStyle(AppTheme.textSecondary)
                }

                if let promiseUntil = fastHome?.activePromise?.promisedUntil, !promiseUntil.isEmpty {
                    Text(L10n.format("mobile_retailer.fast_home.active_promise", String(promiseUntil.prefix(10))))
                        .font(.system(.caption2, design: .rounded, weight: .medium))
                        .foregroundStyle(AppTheme.info)
                }
            }
            .padding(AppTheme.spacingLG)
        }
        .accessibilityIdentifier("fast-home-what-we-owe")
    }

    private var todaysProposalCard: some View {
        LabCard {
            VStack(alignment: .leading, spacing: AppTheme.spacingMD) {
                HStack {
                    Label(L10n.string("mobile_retailer.fast_home.todays_proposal"), systemImage: "sparkles")
                        .font(.system(.headline, design: .rounded, weight: .bold))
                        .foregroundStyle(AppTheme.textPrimary)
                    Spacer()
                    if let countdown = vm.cutoffCountdown {
                        Text(
                            countdown.isExpired
                                ? L10n.string("mobile_retailer.fast_home.cutoff_closed")
                                : L10n.format("mobile_retailer.fast_home.cutoff_countdown", countdown.formattedHMS)
                        )
                        .font(.system(.caption, design: .monospaced, weight: .bold))
                        .padding(.horizontal, 10)
                        .padding(.vertical, 5)
                        .background((countdown.isExpired ? AppTheme.destructive : AppTheme.warning).opacity(0.14))
                        .foregroundStyle(countdown.isExpired ? AppTheme.destructive : AppTheme.warning)
                        .clipShape(Capsule())
                        .accessibilityIdentifier("fast-home-cutoff-badge")
                    }
                }

                if let msg = vm.fastActionMessage, !msg.isEmpty {
                    Text(msg)
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.success)
                }
                if let err = vm.fastActionError, !err.isEmpty {
                    Text(err)
                        .font(.system(.caption, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.destructive)
                }

                if let proposal = fastHome?.pendingProposal {
                    if let supplier = proposal.supplierName, !supplier.isEmpty {
                        Text(supplier)
                            .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            .foregroundStyle(AppTheme.textPrimary)
                    }

                    let itemSummary = L10n.format("mobile_retailer.fast_home.delivery_items_count", "\(max(proposal.itemsCount, proposal.lineItems.count))")
                    let totalFormatted = RetailerMoney.formatWholeCurrency(proposal.totalMinor, currency: proposal.currency)
                    Text(L10n.format("mobile_retailer.fast_home.proposal_summary", itemSummary, totalFormatted))
                        .font(.system(.subheadline, design: .rounded, weight: .semibold))
                        .foregroundStyle(AppTheme.textPrimary)

                    if !proposal.lineItems.isEmpty {
                        VStack(alignment: .leading, spacing: 6) {
                            ForEach(proposal.lineItems.prefix(3)) { line in
                                HStack {
                                    Text(line.name)
                                        .font(.system(.caption, design: .rounded))
                                        .foregroundStyle(AppTheme.textSecondary)
                                        .lineLimit(1)
                                    Spacer()
                                    Text("×\(line.quantity)")
                                        .font(.system(.caption, design: .rounded, weight: .semibold))
                                        .foregroundStyle(AppTheme.textPrimary)
                                }
                            }
                        }
                        .padding(.vertical, 4)
                    }

                    HStack(spacing: AppTheme.spacingSM) {
                        Button {
                            Task { await vm.confirmFastProposal() }
                        } label: {
                            Text(L10n.string("mobile_retailer.fast_home.confirm_1tap"))
                                .font(.system(.subheadline, design: .rounded, weight: .bold))
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                                .background(AppTheme.success)
                                .foregroundStyle(.white)
                                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                        }
                        .disabled(vm.orderActionPending)
                        .buttonStyle(.plain)
                        .accessibilityIdentifier("fast-home-confirm-button")

                        Button {
                            vm.showEditProposalSheet = true
                        } label: {
                            Text(L10n.string("mobile_retailer.fast_home.edit"))
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                                .background(AppTheme.surfaceElevated)
                                .foregroundStyle(AppTheme.textPrimary)
                                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                        }
                        .disabled(vm.orderActionPending || proposal.lineItems.isEmpty)
                        .buttonStyle(.plain)
                        .accessibilityIdentifier("fast-home-edit-button")

                        Button {
                            Task { await vm.holdFastProposal() }
                        } label: {
                            Text(L10n.string("mobile_retailer.fast_home.hold"))
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                                .frame(maxWidth: .infinity, minHeight: AppTheme.minTouchTarget)
                                .background(AppTheme.destructive.opacity(0.14))
                                .foregroundStyle(AppTheme.destructive)
                                .clipShape(.rect(cornerRadius: AppTheme.radiusButton))
                        }
                        .disabled(vm.orderActionPending)
                        .buttonStyle(.plain)
                        .accessibilityIdentifier("fast-home-hold-button")
                    }
                } else {
                    Text(L10n.string("mobile_retailer.fast_home.proposal_empty"))
                        .font(.system(.subheadline, design: .rounded))
                        .foregroundStyle(AppTheme.textSecondary)
                }
            }
            .padding(AppTheme.spacingLG)
        }
        .accessibilityIdentifier("fast-home-todays-proposal")
    }

    private var hasCachedDashboardData: Bool {
        fastHome != nil || !activeOrders.isEmpty || !predictions.isEmpty || !reorderProducts.isEmpty
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

private struct FastProposalEditSheet: View {
    let proposal: FastHomePendingProposal
    let onCancel: () -> Void
    let onSave: ([FastHomeProposalItem]) async -> Void

    @State private var items: [FastHomeProposalItem]
    @State private var saving = false

    init(
        proposal: FastHomePendingProposal,
        onCancel: @escaping () -> Void,
        onSave: @escaping ([FastHomeProposalItem]) async -> Void
    ) {
        self.proposal = proposal
        self.onCancel = onCancel
        self.onSave = onSave
        _items = State(initialValue: proposal.lineItems)
    }

    var body: some View {
        NavigationStack {
            List {
                ForEach(items.indices, id: \.self) { index in
                    let item = items[index]
                    let step = Int64(max(1, item.packSize ?? 1))
                    HStack {
                        VStack(alignment: .leading, spacing: 2) {
                            Text(item.name)
                                .font(.system(.subheadline, design: .rounded, weight: .semibold))
                            Text(RetailerMoney.formatWholeCurrency(item.unitPriceMinor, currency: proposal.currency))
                                .font(.system(.caption, design: .rounded))
                                .foregroundStyle(AppTheme.textSecondary)
                        }
                        Spacer()
                        HStack(spacing: 12) {
                            Button {
                                items[index].quantity = max(0, items[index].quantity - step)
                            } label: {
                                Image(systemName: "minus.circle.fill")
                                    .foregroundStyle(AppTheme.textSecondary)
                            }
                            .buttonStyle(.plain)

                            Text("\(items[index].quantity)")
                                .font(.system(.subheadline, design: .monospaced, weight: .bold))
                                .frame(minWidth: 32)

                            Button {
                                items[index].quantity += step
                            } label: {
                                Image(systemName: "plus.circle.fill")
                                    .foregroundStyle(AppTheme.accent)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
            .navigationTitle(L10n.string("mobile_retailer.fast_home.edit_proposal_title"))
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button(L10n.string("mobile_retailer.fast_home.cancel")) {
                        onCancel()
                    }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button(L10n.string("mobile_retailer.fast_home.save_changes")) {
                        saving = true
                        Task {
                            await onSave(items)
                            saving = false
                        }
                    }
                    .disabled(saving)
                }
            }
        }
    }
}

#Preview {
    NavigationStack {
        DashboardView()
            .environment(CartManager())
    }
}

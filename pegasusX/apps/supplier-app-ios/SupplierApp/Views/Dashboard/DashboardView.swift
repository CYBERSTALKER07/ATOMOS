import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(SupplierRealtimeHub.self) private var realtimeHub
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    @State private var vm = DashboardViewModel()

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 200 : 150
    }

    var body: some View {
        ScrollView {
            Group {
                if vm.loading && vm.dashboard == nil {
                    SupplierLoadingView(
                        title: "mobile_supplier.loading.dashboard",
                        message: "mobile_supplier.loading.dashboard_message"
                    )
                } else if let error = vm.error, vm.dashboard == nil {
                    SupplierErrorView(message: error) {
                        Task { await vm.load(tokenStore: tokenStore) }
                    }
                } else if let dashboard = vm.dashboard {
                    VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                        if !tokenStore.isConfigured {
                            billingBanner
                        }
                        PackBanner(pack: vm.pack)

                        SupplierSectionHeader(
                            title: "mobile_supplier.chrome.operations_at_a_glance",
                            subtitle: "mobile_supplier.chrome.live_supplier_kpis"
                        )

                        if let meiSummary = vm.meiSummary {
                            VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
                                Text("supplier_portal.residual.text.meio_network")
                                    .font(.headline)
                                Text(L10n.format("mobile_supplier.ui.warehousesscanned_warehouses_transferrecommendations_transfer_recs_insig", "\(meiSummary.warehousesScanned)", "\(meiSummary.transferRecommendations)", "\(meiSummary.insightsGenerated)"))
                                    .font(.caption)
                                    .foregroundStyle(.secondary)
                                if let meiError = vm.meiError {
                                    Text(meiError).font(.caption).foregroundStyle(SupplierTheme.destructive)
                                }
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            .padding()
                            .background(SupplierTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
                        } else if let meiError = vm.meiError {
                            Text(meiError).font(.caption).foregroundStyle(SupplierTheme.destructive)
                        }

                        if let demandConfidence = vm.demandConfidence {
                            ForecastConfidenceView(
                                confidence: demandConfidence,
                                updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: vm.demandGeneratedAt),
                                stale: ForecastConfidenceSupport.isForecastStale(generatedAt: vm.demandGeneratedAt)
                            )
                            if let demandError = vm.demandError {
                                Text(demandError).font(.caption).foregroundStyle(SupplierTheme.destructive)
                            }
                        } else if let demandError = vm.demandError {
                            Text(demandError).font(.caption).foregroundStyle(SupplierTheme.destructive)
                        }

                        NetworkPulseStrip(events: vm.pulseEvents, loading: vm.pulseLoading, error: vm.pulseError)

                        LazyVGrid(
                            columns: [GridItem(.adaptive(minimum: gridMin), spacing: SupplierTheme.spacingMD)],
                            spacing: SupplierTheme.spacingMD
                        ) {
                            NavigationLink {
                                TreasuryHubView()
                            } label: {
                                KpiTile(
                                    title: "mobile_supplier.chrome.revenue_today",
                                    value: dashboard.todayRevenueMinor.map { formatPackMoney($0, pack: vm.pack) } ?? "—",
                                    systemImage: "banknote",
                                    tint: SupplierTheme.accentWarm
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                OrdersHubView()
                            } label: {
                                KpiTile(
                                    title: "mobile_supplier.chrome.completion",
                                    value: "\(MissingNumber.label(dashboard.deliveriesCompletedToday))/\(MissingNumber.label(dashboard.deliveriesAttemptedToday))",
                                    systemImage: "checkmark.circle",
                                    tint: SupplierTheme.success
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                OrdersHubView(initialCommandStatus: "SUBMITTED")
                            } label: {
                                KpiTile(
                                    title: "mobile_supplier.chrome.pending_orders",
                                    value: MissingNumber.label(dashboard.pendingOrders),
                                    systemImage: "shippingbox",
                                    tint: SupplierTheme.warning
                                )
                            }
                            .buttonStyle(.pressable)

                            NavigationLink {
                                InventoryView()
                            } label: {
                                KpiTile(
                                    title: "mobile_supplier.chrome.inventory_skus",
                                    value: MissingNumber.label(dashboard.inventorySKUs),
                                    systemImage: "archivebox",
                                    tint: SupplierTheme.accentWarm
                                )
                            }
                            .buttonStyle(.pressable)
                        }

                        StatusStackView(
                            dictionary: orderStatusFunnel,
                            counts: dashboard.ordersByStatus,
                            source: "live",
                            onSelect: { commandJump($0) }
                        )

                        StatusStackView(
                            dictionary: manifestStates,
                            counts: dashboard.manifestsByState,
                            source: dashboard.manifestsByState.isEmpty ? "empty" : "live"
                        )

                        Text(L10n.format("mobile_supplier.ui.updated_updatedat", "\(dashboard.updatedAt)"))
                            .font(.caption2)
                            .foregroundStyle(.tertiary)
                    }
                    .supplierReadableWidth()
                    .padding()
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.dashboard")
        .toolbar {
            signOutToolbar
            ToolbarItem(placement: .topBarTrailing) {
                NavigationLink {
                    NotificationInboxView()
                } label: {
                    Image(systemName: "bell")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("portal.nav.notifications"))
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    PegasusHaptics.light()
                    Task { await vm.load(tokenStore: tokenStore, silent: true) }
                } label: {
                    Image(systemName: "arrow.clockwise")
                        .font(.body.weight(.semibold))
                        .foregroundStyle(SupplierTheme.label)
                        .frame(minWidth: 44, minHeight: 44)
                }
                .accessibilityLabel(Text("portal.page.orders.action.refresh"))
            }
        }
        .navigationDestination(item: $vm.commandJump) { jump in
            OrdersHubView(initialCommandStatus: jump.status)
        }
        .refreshable {
            PegasusHaptics.light()
            await vm.load(tokenStore: tokenStore, silent: true)
        }
        .task {
            await vm.runDashboardLoop(tokenStore: tokenStore, intervalNs: 60_000_000_000)
        }
        .onChange(of: realtimeHub.refreshEpoch) { _, _ in
            Task { await vm.load(tokenStore: tokenStore, silent: true) }
        }
        .onChange(of: realtimeHub.reconnectEpoch) { _, _ in
            Task { await vm.load(tokenStore: tokenStore, silent: true) }
        }
    }

    private func commandJump(_ status: String) {
        PegasusHaptics.selection()
        vm.commandJump = CommandStatusJump(status: status)
    }

    private var billingBanner: some View {
        HStack(spacing: SupplierTheme.spacingMD) {
            Image(systemName: "creditcard")
            VStack(alignment: .leading, spacing: 4) {
                Text("mobile_supplier.ui.billing_incomplete")
                    .font(.subheadline.bold())
                Text("mobile_supplier.ui.finish_setup_to_enable_treasury_and_payouts")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            Spacer()
            Button("mobile_supplier.ui.setup") {
                PegasusHaptics.light()
                tokenStore.showBillingGate()
            }
            .font(.caption.bold())
            .frame(minWidth: 44, minHeight: 44)
        }
        .supplierCard()
    }

    @ToolbarContentBuilder
    private var signOutToolbar: some ToolbarContent {
        ToolbarItem(placement: .topBarTrailing) {
            Button {
                PegasusHaptics.selection()
                tokenStore.clear()
            } label: {
                Image(systemName: "rectangle.portrait.and.arrow.right")
                    .font(.body.weight(.semibold))
                    .foregroundStyle(SupplierTheme.label)
                    .frame(minWidth: 44, minHeight: 44)
            }
            .accessibilityLabel(Text("common.action.sign_out"))
        }
    }
}

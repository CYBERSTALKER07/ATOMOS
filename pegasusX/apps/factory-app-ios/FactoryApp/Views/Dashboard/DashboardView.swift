import SwiftUI

struct DashboardView: View {
    @Environment(TokenStore.self) private var tokenStore
    @Environment(\.horizontalSizeClass) private var horizontalSizeClass
    let onOpenSupplyRequests: () -> Void
    let onOpenPayloadOverride: () -> Void
    let onOpenManifestExceptions: () -> Void
    let onOpenAnalytics: () -> Void
    let onOpenInsights: () -> Void
    var onOpenTransfers: (String) -> Void = { _ in }
    var onOpenLoadingBay: () -> Void = {}

    @State private var vm = DashboardViewModel()

    private var gridMin: CGFloat {
        horizontalSizeClass == .regular ? 180 : 160
    }

    var body: some View {
        @Bindable var vm = vm

        ScrollView {
            if vm.loading {
                FactoryLoadingState(
                    title: "Loading dashboard",
                    message: "Fetching live factory metrics for loading, fleet, and staffing."
                )
            } else if let error = vm.error {
                FactoryErrorView(message: error) {
                    Task { await vm.load() }
                }
            } else {
                VStack(alignment: .leading, spacing: LabTheme.spacingLG) {
                    ClientPolicyBanner(
                        message: vm.clientPolicyMessage,
                        force: vm.clientPolicyForce,
                        onUpdate: vm.clientPolicyMessage == nil ? nil : {
                            vm.promptInstallUpdate()
                        },
                        onDismiss: vm.clientPolicyForce ? nil : { vm.dismissPolicyBanner() }
                    )
                    DashboardHeroCard(stats: vm.stats)
                    WorkflowLaunchCard(
                        onOpenSupplyRequests: onOpenSupplyRequests,
                        onOpenPayloadOverride: onOpenPayloadOverride,
                        onOpenManifestExceptions: onOpenManifestExceptions,
                        onOpenManifests: { vm.showManifests = true },
                        onOpenAnalytics: onOpenAnalytics,
                        onOpenCreateTransfer: { vm.showCreateTransfer = true },
                        onOpenInsights: onOpenInsights
                    )
                    FactorySectionHeader(
                        title: "Operations at a glance",
                        subtitle: "Live factory KPIs across transfers, fleet, and staffing"
                    )
                    .padding(.horizontal)

                    LazyVGrid(
                        columns: [GridItem(.adaptive(minimum: gridMin), spacing: LabTheme.spacingMD)],
                        spacing: LabTheme.spacingMD
                    ) {
                        ForEach(Array(vm.dashboardMetrics.enumerated()), id: \.element.title) { index, metric in
                            Button {
                                handleMetricTap(title: metric.title)
                            } label: {
                                KpiTile(
                                    title: metric.title,
                                    value: metric.value,
                                    systemImage: metric.icon,
                                    tint: metric.tint,
                                    supporting: metric.supporting,
                                    chip: metric.chip,
                                    staggerIndex: index
                                )
                            }
                            .buttonStyle(.pressable)
                            .accessibilityLabel("\(metric.title): \(metric.value)")
                        }
                    }
                    .padding(.horizontal)

                    HStack {
                        SourceChip(source: vm.stats.source)
                        Text(vm.stats.source == "empty" ? "No factory rows yet" : "Dashboard \(vm.stats.source)")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }
                    .padding(.horizontal)
                    .accessibilityElement(children: .combine)
                    .accessibilityIdentifier("gs-u-factory-source")

                    FactorySectionHeader(
                        title: "Transfers",
                        subtitle: "Factory plane only"
                    )
                    .padding(.horizontal)
                    StatusStackView(
                        dictionary: factoryTransferStates,
                        counts: vm.stats.transfersByState,
                        source: vm.stats.source,
                        onSelect: { onOpenTransfers($0) }
                    )
                    .padding(.horizontal)

                    FactorySectionHeader(
                        title: "Factory trucks",
                        subtitle: "FactoryTruckManifests. Last-mile retailer IN_TRANSIT is not a factory truck."
                    )
                    .padding(.horizontal)
                    StatusStackView(
                        dictionary: manifestStates,
                        counts: vm.stats.manifestsByState,
                        source: vm.stats.source,
                        onSelect: { _ in onOpenLoadingBay() }
                    )
                    .padding(.horizontal)
                    .accessibilityIdentifier("gs-u-factory-trucks")

                    StatusStackView(
                        dictionary: factoryVehicleStates,
                        counts: vm.stats.vehiclesByState,
                        source: vm.stats.source
                    )
                    .padding(.horizontal)
                    StatusStackView(
                        dictionary: factoryDriverDuty,
                        counts: vm.stats.driverDuty,
                        source: vm.stats.source
                    )
                    .padding(.horizontal)
                }
                .labReadableWidth()
                .padding(.vertical)
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(LabTheme.background)
        .navigationTitle("factory_portal.setup.factory.text.factory")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.nav.notifications", systemImage: "bell") {
                    PegasusHaptics.selection()
                    vm.showNotifications = true
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Notifications")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") {
                    PegasusHaptics.selection()
                    Task { await vm.load() }
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Refresh dashboard")
            }
            ToolbarItem(placement: .topBarTrailing) {
                Button("common.action.sign_out", systemImage: "rectangle.portrait.and.arrow.right") {
                    PegasusHaptics.warning()
                    tokenStore.clear()
                }
                .labelStyle(.iconOnly)
                .frame(minWidth: 44, minHeight: 44)
                .accessibilityLabel("Sign out")
            }
        }
        .task {
            await vm.runLifecycleLoop()
        }
        .onDisappear {
            vm.stopRealtime()
        }
        .sheet(isPresented: $vm.showManifests) {
            NavigationStack {
                ManifestsView(showsCloseButton: true)
            }
        }
        .sheet(isPresented: $vm.showNotifications) {
            NavigationStack {
                NotificationInboxView(showsCloseButton: true)
            }
        }
        .sheet(isPresented: $vm.showCreateTransfer) {
            CreateTransferView { _ in
                vm.showCreateTransfer = false
            }
        }
    }

    private func handleMetricTap(title: String) {
        PegasusHaptics.light()
        switch title {
        case "Pending transfers":
            onOpenTransfers("PENDING")
        case "Now loading":
            onOpenLoadingBay()
        case "Active manifests":
            vm.showManifests = true
        case "Dispatched today":
            onOpenTransfers("COMPLETED")
        case "Vehicles total", "Vehicles available":
            onOpenInsights()
        case "Staff on shift":
            onOpenSupplyRequests()
        case "Gate exceptions":
            onOpenManifestExceptions()
        default:
            break
        }
    }
}

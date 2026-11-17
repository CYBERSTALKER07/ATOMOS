import SwiftUI

/// Portal parity hub for secondary warehouse admin routes (pegasus warehouse-portal nav).
struct MoreHubView: View {
    var body: some View {
        ResponsiveGridContentWrapper {
            Section("Fulfillment") {
                NavigationLink { ManifestsView() } label: {
                    Label("portal.nav.manifests", systemImage: "doc.text")
                        .frame(minHeight: 44)
                }
                NavigationLink { DispatchView() } label: {
                    Label("portal.nav.dispatch", systemImage: "paperplane")
                        .frame(minHeight: 44)
                }
                NavigationLink { RescuesView() } label: {
                    Label("mobile_warehouse.ui.fleet_rescues", systemImage: "wrench.and.screwdriver")
                        .frame(minHeight: 44)
                }
                NavigationLink { DispatchSettingsView() } label: {
                    Label("warehouse_portal.dispatch_settings.text.dispatch_settings", systemImage: "slider.horizontal.3")
                        .frame(minHeight: 44)
                }
                NavigationLink { FleetLiveMapView() } label: {
                    Label("supplier_portal.dashboard.text.live_fleet_map", systemImage: "map")
                        .frame(minHeight: 44)
                }
                NavigationLink { TransferActionsView() } label: {
                    Label("warehouse_portal.transfers.text.transfer_actions", systemImage: "arrow.left.arrow.right")
                        .frame(minHeight: 44)
                }
            }
            Section("Inventory") {
                NavigationLink { ProductsView() } label: {
                    Label("portal.nav.products", systemImage: "square.grid.2x2")
                        .frame(minHeight: 44)
                }
                NavigationLink { PreordersView() } label: {
                    Label("portal.nav.preorders", systemImage: "calendar")
                        .frame(minHeight: 44)
                }
                NavigationLink { StockCommitmentsView() } label: {
                    Label("portal.nav.stock_commitments", systemImage: "archivebox")
                        .frame(minHeight: 44)
                }
                NavigationLink { SupplyRequestsHubView() } label: {
                    Label("portal.nav.supply_requests", systemImage: "arrow.triangle.2.circlepath")
                        .frame(minHeight: 44)
                }
                NavigationLink { CoverageView() } label: {
                    Label("Coverage and supply", systemImage: "mappin.and.ellipse")
                        .frame(minHeight: 44)
                }
                NavigationLink { ReplenishmentView() } label: {
                    Label("portal.nav.replenishment", systemImage: "shippingbox")
                        .frame(minHeight: 44)
                }
                NavigationLink { DemandForecastView() } label: {
                    Label("portal.nav.demand_forecast", systemImage: "chart.line.uptrend.xyaxis")
                        .frame(minHeight: 44)
                }
                NavigationLink { OpsSettingsView() } label: {
                    Label("mobile_warehouse.ui.ops_settings", systemImage: "gearshape")
                        .frame(minHeight: 44)
                }
                NavigationLink { ReturnPolicySettingsView() } label: {
                    Label("warehouse_portal.settings.return_policy_settings_section.text.returns_and_reverse_sla", systemImage: "arrow.uturn.backward.circle")
                        .frame(minHeight: 44)
                }
                NavigationLink { LocationSettingsView() } label: {
                    Label("warehouse_portal.settings.text.depot_location", systemImage: "mappin.and.ellipse")
                        .frame(minHeight: 44)
                }
            }
            Section("Operations") {
                NavigationLink { OperationsView() } label: {
                    Label("warehouse_portal.operations.text.depot_operations", systemImage: "paperplane")
                        .frame(minHeight: 44)
                }
                NavigationLink { CashierBlindCountView(onBack: nil) } label: {
                    Label("Cashier Blind Count & Vault", systemImage: "banknote.fill")
                        .frame(minHeight: 44)
                }
                NavigationLink { YardGateView(onBack: nil) } label: {
                    Label("Yard Gate Control", systemImage: "lock.shield.fill")
                        .frame(minHeight: 44)
                }
                NavigationLink { CRMView() } label: {
                    Label("portal.nav.retailers", systemImage: "person.crop.rectangle")
                        .frame(minHeight: 44)
                }
                NavigationLink { ReturnsView() } label: {
                    Label("portal.nav.returns", systemImage: "arrow.uturn.backward")
                        .frame(minHeight: 44)
                }
                NavigationLink { LaborCapacityView() } label: {
                    Label("portal.nav.labor_capacity", systemImage: "person.3")
                        .frame(minHeight: 44)
                }
                NavigationLink { ExceptionsView() } label: {
                    Label("portal.nav.exceptions", systemImage: "exclamationmark.triangle")
                        .frame(minHeight: 44)
                }
                NavigationLink { WarehouseScoredExceptionsView() } label: {
                    Label("Control tower", systemImage: "antenna.radiowaves.left.and.right")
                        .frame(minHeight: 44)
                }
                NavigationLink { ClaimsView() } label: {
                    Label("portal.nav.claims", systemImage: "doc.text")
                        .frame(minHeight: 44)
                }
                NavigationLink { AnalyticsView() } label: {
                    Label("portal.nav.analytics", systemImage: "chart.bar.xaxis")
                        .frame(minHeight: 44)
                }
                NavigationLink { TreasuryView() } label: {
                    Label("portal.nav.treasury", systemImage: "banknote")
                        .frame(minHeight: 44)
                }
                NavigationLink { PaymentConfigView() } label: {
                    Label("warehouse_portal.treasury.text.payment_config", systemImage: "creditcard")
                        .frame(minHeight: 44)
                }
                NavigationLink { NotificationInboxView() } label: {
                    Label("portal.nav.notifications", systemImage: "bell")
                        .frame(minHeight: 44)
                }
            }
            Section("Portal only") {
                NavigationLink { PortalHandoffView(feature: .profile) } label: {
                    Label("portal.nav.profile", systemImage: "person.crop.circle")
                        .frame(minHeight: 44)
                }
                NavigationLink { PortalHandoffView(feature: .search) } label: {
                    Label("mobile_warehouse.ui.global_search", systemImage: "magnifyingglass")
                        .frame(minHeight: 44)
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .navigationTitle("mobile_warehouse.ui.more")
    }
}

import SwiftUI

/// Portal parity hub for secondary warehouse admin routes (pegasus warehouse-portal nav).
struct MoreHubView: View {
    var body: some View {
        ResponsiveGridContentWrapper {
            Section("Fulfillment") {
                NavigationLink { ManifestsView() } label: {
                    Label("portal.nav.manifests", systemImage: "doc.text")
                }
                NavigationLink { DispatchView() } label: {
                    Label("portal.nav.dispatch", systemImage: "paperplane")
                }
                NavigationLink { RescuesView() } label: {
                    Label("mobile_warehouse.ui.fleet_rescues", systemImage: "wrench.and.screwdriver")
                }
                NavigationLink { DispatchSettingsView() } label: {
                    Label("warehouse_portal.dispatch_settings.text.dispatch_settings", systemImage: "slider.horizontal.3")
                }
                NavigationLink { FleetLiveMapView() } label: {
                    Label("supplier_portal.dashboard.text.live_fleet_map", systemImage: "map")
                }
                NavigationLink { TransferActionsView() } label: {
                    Label("warehouse_portal.transfers.text.transfer_actions", systemImage: "arrow.left.arrow.right")
                }
            }
            Section("Inventory") {
                NavigationLink { ProductsView() } label: {
                    Label("portal.nav.products", systemImage: "square.grid.2x2")
                }
                NavigationLink { PreordersView() } label: {
                    Label("portal.nav.preorders", systemImage: "calendar")
                }
                NavigationLink { StockCommitmentsView() } label: {
                    Label("portal.nav.stock_commitments", systemImage: "archivebox")
                }
                NavigationLink { SupplyRequestsHubView() } label: {
                    Label("portal.nav.supply_requests", systemImage: "arrow.triangle.2.circlepath")
                }
                NavigationLink { ReplenishmentView() } label: {
                    Label("portal.nav.replenishment", systemImage: "shippingbox")
                }
                NavigationLink { DemandForecastView() } label: {
                    Label("portal.nav.demand_forecast", systemImage: "chart.line.uptrend.xyaxis")
                }
                NavigationLink { OpsSettingsView() } label: {
                    Label("mobile_warehouse.ui.ops_settings", systemImage: "gearshape")
                }
                NavigationLink { ReturnPolicySettingsView() } label: {
                    Label("warehouse_portal.settings.return_policy_settings_section.text.returns_and_reverse_sla", systemImage: "arrow.uturn.backward.circle")
                }
                NavigationLink { LocationSettingsView() } label: {
                    Label("warehouse_portal.settings.text.depot_location", systemImage: "mappin.and.ellipse")
                }
            }
            Section("Operations") {
                NavigationLink { OperationsView() } label: {
                    Label("warehouse_portal.operations.text.depot_operations", systemImage: "paperplane")
                }
                NavigationLink { CRMView() } label: {
                    Label("portal.nav.retailers", systemImage: "person.crop.rectangle")
                }
                NavigationLink { ReturnsView() } label: {
                    Label("portal.nav.returns", systemImage: "arrow.uturn.backward")
                }
                NavigationLink { ColdChainView() } label: {
                    Label("portal.nav.cold_chain", systemImage: "thermometer")
                }
                NavigationLink { LaborCapacityView() } label: {
                    Label("portal.nav.labor_capacity", systemImage: "person.3")
                }
                NavigationLink { ExceptionsView() } label: {
                    Label("portal.nav.exceptions", systemImage: "exclamationmark.triangle")
                }
                NavigationLink { ClaimsView() } label: {
                    Label("portal.nav.claims", systemImage: "doc.text")
                }
                NavigationLink { AnalyticsView() } label: {
                    Label("portal.nav.analytics", systemImage: "chart.bar.xaxis")
                }
                NavigationLink { TreasuryView() } label: {
                    Label("portal.nav.treasury", systemImage: "banknote")
                }
                NavigationLink { PaymentConfigView() } label: {
                    Label("warehouse_portal.treasury.text.payment_config", systemImage: "creditcard")
                }
                NavigationLink { NotificationInboxView() } label: {
                    Label("portal.nav.notifications", systemImage: "bell")
                }
            }
            Section("Portal only") {
                NavigationLink { PortalHandoffView(feature: .profile) } label: {
                    Label("portal.nav.profile", systemImage: "person.crop.circle")
                }
                NavigationLink { PortalHandoffView(feature: .search) } label: {
                    Label("mobile_warehouse.ui.global_search", systemImage: "magnifyingglass")
                }
            }
        }
        .navigationTitle("mobile_warehouse.ui.more")
    }
}

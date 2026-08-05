import SwiftUI

/// Portal parity hub for secondary warehouse admin routes (pegasus warehouse-portal nav).
struct MoreHubView: View {
    var body: some View {
        ResponsiveGridContentWrapper {
            Section("Fulfillment") {
                NavigationLink { ManifestsView() } label: {
                    Label("Manifests", systemImage: "doc.text")
                }
                NavigationLink { DispatchView() } label: {
                    Label("Dispatch", systemImage: "paperplane")
                }
                NavigationLink { RescuesView() } label: {
                    Label("Fleet rescues", systemImage: "wrench.and.screwdriver")
                }
                NavigationLink { DispatchSettingsView() } label: {
                    Label("Dispatch settings", systemImage: "slider.horizontal.3")
                }
                NavigationLink { FleetLiveMapView() } label: {
                    Label("Live fleet map", systemImage: "map")
                }
                NavigationLink { TransferActionsView() } label: {
                    Label("Transfer actions", systemImage: "arrow.left.arrow.right")
                }
            }
            Section("Inventory") {
                NavigationLink { ProductsView() } label: {
                    Label("Products", systemImage: "square.grid.2x2")
                }
                NavigationLink { PreordersView() } label: {
                    Label("Pre-orders", systemImage: "calendar")
                }
                NavigationLink { StockCommitmentsView() } label: {
                    Label("Stock commitments", systemImage: "archivebox")
                }
                NavigationLink { SupplyRequestsHubView() } label: {
                    Label("Supply Requests", systemImage: "arrow.triangle.2.circlepath")
                }
                NavigationLink { ReplenishmentView() } label: {
                    Label("Replenishment", systemImage: "shippingbox")
                }
                NavigationLink { DemandForecastView() } label: {
                    Label("Demand Forecast", systemImage: "chart.line.uptrend.xyaxis")
                }
                NavigationLink { OpsSettingsView() } label: {
                    Label("Ops settings", systemImage: "gearshape")
                }
                NavigationLink { ReturnPolicySettingsView() } label: {
                    Label("Returns & reverse SLA", systemImage: "arrow.uturn.backward.circle")
                }
                NavigationLink { LocationSettingsView() } label: {
                    Label("Depot location", systemImage: "mappin.and.ellipse")
                }
            }
            Section("Operations") {
                NavigationLink { OperationsView() } label: {
                    Label("Depot operations", systemImage: "paperplane")
                }
                NavigationLink { CRMView() } label: {
                    Label("Retailers", systemImage: "person.crop.rectangle")
                }
                NavigationLink { ReturnsView() } label: {
                    Label("Returns", systemImage: "arrow.uturn.backward")
                }
                NavigationLink { ExceptionsView() } label: {
                    Label("Exceptions", systemImage: "exclamationmark.triangle")
                }
                NavigationLink { ClaimsView() } label: {
                    Label("Claims", systemImage: "doc.text")
                }
                NavigationLink { AnalyticsView() } label: {
                    Label("Analytics", systemImage: "chart.bar.xaxis")
                }
                NavigationLink { TreasuryView() } label: {
                    Label("Treasury", systemImage: "banknote")
                }
                NavigationLink { PaymentConfigView() } label: {
                    Label("Payment config", systemImage: "creditcard")
                }
                NavigationLink { NotificationInboxView() } label: {
                    Label("Notifications", systemImage: "bell")
                }
            }
            Section("Portal only") {
                NavigationLink { PortalHandoffView(feature: .profile) } label: {
                    Label("Profile", systemImage: "person.crop.circle")
                }
                NavigationLink { PortalHandoffView(feature: .search) } label: {
                    Label("Global search", systemImage: "magnifyingglass")
                }
            }
        }
        .navigationTitle("More")
    }
}

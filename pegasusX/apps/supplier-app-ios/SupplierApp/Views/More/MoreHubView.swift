import SwiftUI

struct MoreHubView: View {
    var body: some View {
        List {
            Section("Fulfillment") {
                NavigationLink { ManifestsView() } label: {
                    Label("Manifests", systemImage: "doc.text")
                }
                NavigationLink { DispatchPreviewView() } label: {
                    Label("Dispatch preview", systemImage: "paperplane")
                }
                NavigationLink { FleetOrdersView() } label: {
                    Label("Fleet orders", systemImage: "truck.box")
                }
            }
            Section("Exceptions") {
                NavigationLink { ExceptionsView() } label: {
                    Label("Exceptions", systemImage: "exclamationmark.triangle")
                }
                NavigationLink { ShopClosedView() } label: {
                    Label("Shop closed", systemImage: "storefront")
                }
                NavigationLink { EarlyCompleteView() } label: {
                    Label("Early route complete", systemImage: "checkmark.circle")
                }
                // Quantity negotiation disabled ecosystem-wide.
            }
            Section("Insights") {
                NavigationLink { AnalyticsView() } label: {
                    Label("Analytics", systemImage: "chart.bar")
                }
                NavigationLink { ActivityView() } label: {
                    Label("Activity", systemImage: "clock.arrow.circlepath")
                }
                NavigationLink { AIRecommendationsView() } label: {
                    Label("AI recommendations", systemImage: "sparkles")
                }
                NavigationLink { GeoReportView() } label: {
                    Label("Geo report", systemImage: "map")
                }
            }
            Section("Network") {
                NavigationLink { TopologyView() } label: {
                    Label("Factories & warehouses", systemImage: "building.2.crop.circle")
                }
                NavigationLink { DeliveryZonesView() } label: {
                    Label("Delivery zones", systemImage: "mappin.and.ellipse")
                }
                NavigationLink { SupplyLanesView() } label: {
                    Label("Supply lanes", systemImage: "arrow.triangle.swap")
                }
            }
            Section("Treasury") {
                NavigationLink { LedgerView() } label: {
                    Label("Payment ledger", systemImage: "banknote")
                }
                NavigationLink { PaymentsView() } label: {
                    Label("Payments", systemImage: "creditcard")
                }
                NavigationLink { PortalHandoffView(feature: .chargebacks) } label: {
                    Label("Chargebacks", systemImage: "exclamationmark.bubble")
                }
                NavigationLink { ReconciliationView() } label: {
                    Label("Reconciliation", systemImage: "scalemass")
                }
                NavigationLink { OperationsView() } label: {
                    Label("Operations", systemImage: "wrench.and.screwdriver")
                }
            }
            Section("Account") {
                NavigationLink { NotificationInboxView() } label: {
                    Label("Notifications", systemImage: "bell")
                }
                NavigationLink { CatalogView() } label: {
                    Label("Catalog", systemImage: "square.grid.2x2")
                }
                NavigationLink { InventoryView() } label: {
                    Label("Inventory", systemImage: "archivebox")
                }
                NavigationLink { PricingView() } label: {
                    Label("Pricing", systemImage: "dollarsign.circle")
                }
                NavigationLink { PromotionsView() } label: {
                    Label("Promotions", systemImage: "tag")
                }
                NavigationLink { ReturnsView() } label: {
                    Label("Returns", systemImage: "arrow.uturn.backward")
                }
                NavigationLink { OrgFleetView() } label: {
                    Label("Org & fleet", systemImage: "person.3")
                }
                NavigationLink { EarningsView() } label: {
                    Label("Earnings", systemImage: "chart.line.uptrend.xyaxis")
                }
                NavigationLink { ProfileView() } label: {
                    Label("Profile", systemImage: "building.2")
                }
                NavigationLink { PortalHandoffView(feature: .businessSetup) } label: {
                    Label("Business setup", systemImage: "gearshape.2")
                }
            }
        }
        .navigationTitle("More")
        .background(SupplierTheme.background)
    }
}

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
            Section("Insights") {
                NavigationLink { AnalyticsView() } label: {
                    Label("Analytics", systemImage: "chart.bar")
                }
                NavigationLink { DemandHistoryView() } label: {
                    Label("Demand forecast", systemImage: "chart.xyaxis.line")
                }
                NavigationLink { PlanningBrainView() } label: {
                    Label("Planning sandbox", systemImage: "brain.head.profile")
                }
                NavigationLink { KnowledgeGraphView() } label: {
                    Label("Knowledge graph", systemImage: "point.3.connected.trianglepath.dotted")
                }
                NavigationLink { PlanningSettingsView() } label: {
                    Label("Planning settings", systemImage: "calendar")
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
                    Label("Topology", systemImage: "building.2.crop.circle")
                }
                NavigationLink { FactoriesView() } label: {
                    Label("Factories", systemImage: "building.2")
                }
                NavigationLink { WarehousesView() } label: {
                    Label("Warehouses", systemImage: "shippingbox.fill")
                }
                NavigationLink { DeliveryZonesView() } label: {
                    Label("Delivery zones", systemImage: "mappin.and.ellipse")
                }
                NavigationLink { SupplyLanesView() } label: {
                    Label("Supply lanes", systemImage: "arrow.triangle.swap")
                }
            }
            Section("Treasury") {
                NavigationLink { TreasuryHubView() } label: {
                    Label("Treasury hub", systemImage: "building.columns")
                }
                NavigationLink { LedgerView() } label: {
                    Label("Payment ledger", systemImage: "banknote")
                }
                NavigationLink { PaymentsView() } label: {
                    Label("Payments", systemImage: "creditcard")
                }
                NavigationLink { ChargebacksView() } label: {
                    Label("Chargebacks", systemImage: "exclamationmark.bubble")
                }
                NavigationLink { ReconciliationView() } label: {
                    Label("Reconciliation", systemImage: "scalemass")
                }
                NavigationLink { OperationsView() } label: {
                    Label("Operations", systemImage: "wrench.and.screwdriver")
                }
                NavigationLink { ReplenishmentPoliciesView() } label: {
                    Label("Replenishment policies", systemImage: "doc.text")
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
                NavigationLink { InventoryImportView() } label: {
                    Label("Import inventory", systemImage: "square.and.arrow.down")
                }
                NavigationLink { PricingView() } label: {
                    Label("Pricing", systemImage: "dollarsign.circle")
                }
                NavigationLink { RetailerOverridesView() } label: {
                    Label("Retailer overrides", systemImage: "tag.circle")
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
                NavigationLink { BusinessSetupView() } label: {
                    Label("Business setup", systemImage: "gearshape.2")
                }
            }
        }
        .navigationTitle("More")
        .background(SupplierTheme.background)
    }
}

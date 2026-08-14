import SwiftUI

struct MoreHubView: View {
    var body: some View {
        ResponsiveGridContentWrapper {
            Section("Fulfillment") {
                NavigationLink { ManifestsView() } label: {
                    Label("portal.nav.manifests", systemImage: "doc.text")
                }
                NavigationLink { DispatchPreviewView() } label: {
                    Label("mobile_supplier.ui.dispatch_preview", systemImage: "paperplane")
                }
                NavigationLink { FleetOrdersView() } label: {
                    Label("supplier_portal.fleet.orders.text.fleet_orders", systemImage: "truck.box")
                }
            }
            Section("Insights") {
                NavigationLink { AnalyticsView() } label: {
                    Label("portal.nav.analytics", systemImage: "chart.bar")
                }
                NavigationLink { DemandHistoryView() } label: {
                    Label("supplier_portal.analytics.demand.text.demand_forecast", systemImage: "chart.xyaxis.line")
                }
                NavigationLink { PlanningBrainView() } label: {
                    Label("mobile_supplier.ui.planning_sandbox", systemImage: "brain.head.profile")
                }
                NavigationLink { KnowledgeGraphView() } label: {
                    Label("mobile_supplier.ui.knowledge_graph", systemImage: "point.3.connected.trianglepath.dotted")
                }
                NavigationLink { PlanningSettingsView() } label: {
                    Label("supplier_portal.settings.planning.text.planning_settings", systemImage: "calendar")
                }
                NavigationLink { ReturnPolicySettingsView() } label: {
                    Label("portal.nav.return_policy", systemImage: "arrow.uturn.backward.circle")
                }
                NavigationLink { ActivityView() } label: {
                    Label("portal.nav.activity", systemImage: "clock.arrow.circlepath")
                }
                NavigationLink { AIRecommendationsView() } label: {
                    Label("mobile_supplier.ui.ai_recommendations", systemImage: "sparkles")
                }
                NavigationLink { GeoReportView() } label: {
                    Label("supplier_portal.geo_report.text.geo_report", systemImage: "map")
                }
                NavigationLink { ScoredExceptionsView() } label: {
                    Label("Control tower", systemImage: "antenna.radiowaves.left.and.right")
                }
                NavigationLink { PlaybooksView() } label: {
                    Label("Playbooks", systemImage: "book")
                }
                NavigationLink { JSONFeedView(title: "POS flywheel", path: "v1/supplier/analytics/demand/flywheel", query: ["days": "7"]) } label: {
                    Label("POS flywheel", systemImage: "arrow.triangle.2.circlepath")
                }
                NavigationLink { JSONFeedView(title: "Payday calendar", path: "v1/demand/signals", query: ["type": "PAYDAY"]) } label: {
                    Label("Payday calendar", systemImage: "calendar")
                }
            }
            Section("Network") {
                NavigationLink { TopologyView() } label: {
                    Label("portal.nav.topology", systemImage: "building.2.crop.circle")
                }
                NavigationLink { FactoriesView() } label: {
                    Label("portal.nav.factories", systemImage: "building.2")
                }
                NavigationLink { WarehousesView() } label: {
                    Label("portal.nav.warehouses", systemImage: "shippingbox.fill")
                }
                NavigationLink { CRMView() } label: {
                    Label("portal.nav.crm", systemImage: "person.2")
                }
                NavigationLink { JSONFeedView(title: "Segmentation", path: "v1/supplier/segmentation/retailers") } label: {
                    Label("Segmentation", systemImage: "square.grid.3x3")
                }
                NavigationLink { DeliveryZonesView() } label: {
                    Label("supplier_portal.delivery_zones.text.delivery_zones", systemImage: "mappin.and.ellipse")
                }
                NavigationLink { SupplyLanesView() } label: {
                    Label("supplier_portal.supply_lanes.text.supply_lanes", systemImage: "arrow.triangle.swap")
                }
            }
            Section("Treasury") {
                NavigationLink { TreasuryHubView() } label: {
                    Label("mobile_supplier.ui.treasury_hub", systemImage: "building.columns")
                }
                NavigationLink { PayoutsView() } label: {
                    Label("portal.nav.payouts", systemImage: "banknote")
                }
                NavigationLink { JSONFeedView(title: "Credit policy", path: "v1/supplier/credit-program") } label: {
                    Label("Credit policy", systemImage: "creditcard")
                }
                NavigationLink { CreditAdminDisableView() } label: {
                    Label("Credit admin disable", systemImage: "exclamationmark.octagon")
                }
                NavigationLink { JSONFeedView(title: "Tax regimes", path: "v1/admin/tax-regimes", query: ["country": "UZ"]) } label: {
                    Label("Tax regimes", systemImage: "building.columns")
                }
                NavigationLink { LedgerView() } label: {
                    Label("mobile_supplier.ui.payment_ledger", systemImage: "banknote")
                }
                NavigationLink { PaymentsView() } label: {
                    Label("portal.nav.payments", systemImage: "creditcard")
                }
                NavigationLink { ChargebacksView() } label: {
                    Label("portal.nav.chargebacks", systemImage: "exclamationmark.bubble")
                }
                NavigationLink { ReconciliationView() } label: {
                    Label("portal.nav.reconciliation", systemImage: "scalemass")
                }
                NavigationLink { OperationsView() } label: {
                    Label("portal.nav.operations", systemImage: "wrench.and.screwdriver")
                }
                NavigationLink { ReplenishmentPoliciesView() } label: {
                    Label("supplier_portal.operations.replenishment_policies.text.replenishment_policies", systemImage: "doc.text")
                }
            }
            Section("Account") {
                NavigationLink { NotificationInboxView() } label: {
                    Label("portal.nav.notifications", systemImage: "bell")
                }
                NavigationLink { NotificationPreferencesView() } label: {
                    Label("supplier_portal.settings.notification_preferences.text.notification_preferences", systemImage: "bell.badge")
                }
                NavigationLink { CatalogView() } label: {
                    Label("portal.nav.catalog", systemImage: "square.grid.2x2")
                }
                NavigationLink { InventoryView() } label: {
                    Label("portal.nav.inventory", systemImage: "archivebox")
                }
                NavigationLink { InventoryImportView() } label: {
                    Label("mobile_supplier.ui.import_inventory", systemImage: "square.and.arrow.down")
                }
                NavigationLink { PricingView() } label: {
                    Label("portal.nav.pricing", systemImage: "dollarsign.circle")
                }
                NavigationLink { RetailerOverridesView() } label: {
                    Label("supplier_portal.pricing.retailer_overrides.text.retailer_overrides", systemImage: "tag.circle")
                }
                NavigationLink { PromotionsView() } label: {
                    Label("portal.nav.promotions", systemImage: "tag")
                }
                NavigationLink { ReturnsView() } label: {
                    Label("portal.nav.returns", systemImage: "arrow.uturn.backward")
                }
                NavigationLink { OrgFleetView() } label: {
                    Label("mobile_supplier.ui.org_and_fleet", systemImage: "person.3")
                }
                NavigationLink { EarningsView() } label: {
                    Label("portal.nav.earnings", systemImage: "chart.line.uptrend.xyaxis")
                }
                NavigationLink { ProfileView() } label: {
                    Label("portal.nav.profile", systemImage: "building.2")
                }
                NavigationLink { BusinessSetupView() } label: {
                    Label("mobile_supplier.ui.business_setup", systemImage: "gearshape.2")
                }
            }
        }
        .navigationTitle("mobile_supplier.ui.more")
        .background(SupplierTheme.background)
    }
}

struct JSONFeedView: View {
    let title: String
    let path: String
    var query: [String: String] = [:]
    @State private var bodyText = ""
    @State private var errorText: String?
    @State private var loading = true

    var body: some View {
        Group {
            if loading {
                ProgressView()
            } else if let errorText {
                Text(errorText).padding()
            } else {
                ScrollView {
                    Text(bodyText)
                        .font(.system(.caption, design: .monospaced))
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding()
                }
            }
        }
        .navigationTitle(title)
        .task { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            bodyText = try await APIClient.shared.getJSONString(path, query: query)
            errorText = nil
        } catch {
            errorText = error.localizedDescription
        }
    }
}

struct ScoredExceptionsView: View {
    @State private var rows: [ScoredException] = []
    @State private var errorText: String?
    @State private var loading = true

    var body: some View {
        Group {
            if loading && rows.isEmpty {
                ProgressView()
            } else if let errorText, rows.isEmpty {
                Text(errorText).padding()
            } else if rows.isEmpty {
                Text("No open scored exceptions")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.type.isEmpty ? "—" : row.type)
                            .font(.headline)
                        Text("score \(row.score) · \(row.severity) · \(row.ageMinutes)m")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if !row.orderId.isEmpty {
                            Text(row.orderId)
                                .font(.caption)
                                .foregroundStyle(.tint)
                        }
                        if !row.topPlaybookName.isEmpty {
                            Text(row.topPlaybookName)
                                .font(.caption)
                        }
                    }
                }
            }
        }
        .navigationTitle("Control tower")
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.scoredExceptions()
            errorText = nil
        } catch {
            errorText = error.localizedDescription
        }
    }
}

struct PlaybooksView: View {
    @State private var rows: [ControlTowerPlaybook] = []
    @State private var errorText: String?
    @State private var loading = true

    var body: some View {
        Group {
            if loading && rows.isEmpty {
                ProgressView()
            } else if let errorText, rows.isEmpty {
                Text(errorText).padding()
            } else if rows.isEmpty {
                Text("No playbooks")
                    .foregroundStyle(.secondary)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                List(rows) { row in
                    VStack(alignment: .leading, spacing: 4) {
                        Text(row.name.isEmpty ? row.playbookId : row.name)
                            .font(.headline)
                        Text("\(row.isActive ? "active" : "inactive") · priority \(row.priority)\(row.autoExecute ? " · auto" : "")")
                            .font(.caption)
                            .foregroundStyle(.secondary)
                        if !row.description.isEmpty {
                            Text(row.description).font(.caption)
                        }
                    }
                }
            }
        }
        .navigationTitle("Playbooks")
        .task { await load() }
        .refreshable { await load() }
    }

    private func load() async {
        loading = true
        defer { loading = false }
        do {
            rows = try await SupplierOperationsService.playbooks()
            errorText = nil
        } catch {
            errorText = error.localizedDescription
        }
    }
}

private struct CreditAdminDisableBody: Encodable {
    let ticket_id: String
    let reason: String
}

private struct CreditAdminDisableResponse: Decodable {
    let status: String?
    let error: String?
}

struct CreditAdminDisableView: View {
    @State private var mode = "relationship"
    @State private var supplierId = ""
    @State private var retailerId = ""
    @State private var ticketId = ""
    @State private var reason = ""
    @State private var message: String?
    @State private var busy = false

    var body: some View {
        Form {
            Picker("Mode", selection: $mode) {
                Text("Relationship").tag("relationship")
                Text("Program").tag("program")
            }
            TextField("Supplier ID", text: $supplierId)
            if mode != "program" {
                TextField("Retailer ID", text: $retailerId)
            }
            TextField("Ticket ID", text: $ticketId)
            TextField("Reason", text: $reason, axis: .vertical)
            Button("Disable permanently") {
                Task { await submit() }
            }
            .disabled(busy || supplierId.isEmpty || ticketId.isEmpty || reason.isEmpty || (mode == "relationship" && retailerId.isEmpty))
            if let message {
                Text(message)
            }
        }
        .navigationTitle("Credit admin disable")
    }

    private func submit() async {
        busy = true
        defer { busy = false }
        do {
            let body = CreditAdminDisableBody(ticket_id: ticketId, reason: reason)
            let path = mode == "program"
                ? "v1/admin/credit-program/\(supplierId)/disable"
                : "v1/admin/credit-relationships/\(supplierId)/\(retailerId)/disable"
            let resp: CreditAdminDisableResponse = try await APIClient.shared.post(path, body: body)
            message = resp.status ?? resp.error ?? "ok"
        } catch {
            message = error.localizedDescription
        }
    }
}

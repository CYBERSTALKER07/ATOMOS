import SwiftUI

struct PlanningBrainView: View {
    @State private var tab: PlanBrainTab = .planning
    @State private var sandop: PlanningSAndOPSnapshot?
    @State private var scenario: PlanningScenarioResult?
    @State private var downtimeHours = 8.0
    @State private var demandDeltaPct = 10.0
    @State private var loading = true
    @State private var running = false
    @State private var error: String?
    @State private var demandConfidence: ForecastConfidence?
    @State private var demandGeneratedAt: String?
    @State private var retailerId = ""
    @State private var sparsity: SparsityGateResult?
    @State private var pushStatus: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                Picker("Plan and Brain", selection: $tab) {
                    Text("Planning").tag(PlanBrainTab.planning)
                    Text("Digital Brain").tag(PlanBrainTab.brain)
                }
                .pickerStyle(.segmented)
                .accessibilityIdentifier("gs-u-plan-brain-tabs")

                if tab == .planning {
                    planningTab
                } else {
                    brainTab
                }
            }
            .supplierReadableWidth()
            .padding()
        }
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.planning")
        .task { await load() }
    }

    @ViewBuilder
    private var planningTab: some View {
        SupplierSectionHeader(
            title: "S&OP snapshot",
            subtitle: "Read-only what-if and lightweight S&OP"
        )

        if let error {
            Text(error)
                .font(.caption)
                .foregroundStyle(SupplierTheme.destructive)
        }

        if loading {
            ProgressView("Loading S&OP…")
        } else if let sandop {
            LazyVGrid(
                columns: [GridItem(.adaptive(minimum: 140), spacing: SupplierTheme.spacingMD)],
                spacing: SupplierTheme.spacingMD
            ) {
                KpiTile(title: "Factory cap (7d)", value: "\(sandop.factoryCapacityUnits)", systemImage: "building.2", tint: .accentColor)
                KpiTile(title: "WH inbound", value: "\(sandop.warehouseInboundCapUnits)", systemImage: "arrow.down.to.line", tint: SupplierTheme.secondaryLabel)
                KpiTile(title: "Utilization", value: "\(Int(sandop.utilizationPct))%", systemImage: "gauge", tint: SupplierTheme.warning)
                KpiTile(
                    title: "Capacity",
                    value: sandop.capacityAlert ? "Breach" : "OK",
                    systemImage: sandop.capacityAlert ? "exclamationmark.triangle" : "checkmark.circle",
                    tint: sandop.capacityAlert ? SupplierTheme.destructive : SupplierTheme.success
                )
            }
        }

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("supplier_portal.planning_brain_panel.text.scenario_run")
                .font(.subheadline.bold())
            Stepper("Downtime \(Int(downtimeHours))h", value: $downtimeHours, in: 0...168, step: 1)
            Stepper("Demand \(Int(demandDeltaPct))%", value: $demandDeltaPct, in: -50...200, step: 5)
            Button(running ? "Running…" : "Run scenario") {
                Task { await runScenario() }
            }
            .disabled(running)
            if let scenario {
                Text(L10n.format("mobile_supplier.ui.sla_risk_slariskpct_fleet_fleetvolumeorders_stockouts_count", "\(Int(scenario.slaRiskPct))", "\(scenario.fleetVolumeOrders)", "\(scenario.stockoutSkus.count)"))
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }

    @ViewBuilder
    private var brainTab: some View {
        if let demandConfidence {
            ForecastConfidenceView(
                confidence: demandConfidence,
                updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: demandGeneratedAt),
                stale: ForecastConfidenceSupport.isForecastStale(generatedAt: demandGeneratedAt)
            )
        }
        if brainForecastLine(confidence: demandConfidence, accuracyPoints: []) == nil {
            Text("No forecast line")
                .font(.caption)
                .foregroundStyle(.secondary)
                .accessibilityIdentifier("gs-u-brain-no-forecast-line")
        }

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("Sparsity")
                .font(.subheadline.bold())
            TextField("Retailer id", text: $retailerId)
                .textInputAutocapitalization(.never)
            Button("Check") { Task { await checkSparsity() } }
            if let sparsity {
                Text(sparsity.allowed ? "allowed · \(sparsity.label)" : "blocked · \(sparsity.blockedReason ?? sparsity.label)")
                    .font(.caption)
                    .accessibilityIdentifier("gs-u-sparsity-result")
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("Predictive push")
                .font(.subheadline.bold())
            Button("Preview push") { Task { await runPush() } }
            if let pushStatus {
                Text(pushStatus)
                    .font(.caption)
                    .accessibilityIdentifier("gs-u-planning-push-status")
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        do {
            sandop = try await SupplierOperationsService.planningSAndOP()
        } catch {
            self.error = error.localizedDescription
        }
        if let demand = try? await SupplierOperationsService.demandToday() {
            demandGeneratedAt = demand.generatedAt
            demandConfidence = ForecastConfidenceSupport.fromDemand(demand)
        }
        loading = false
    }

    @MainActor
    private func runScenario() async {
        running = true
        error = nil
        defer { running = false }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            scenario = try await SupplierOperationsService.runPlanningScenario(
                factoryDowntimeHours: Int(downtimeHours),
                demandDeltaPct: demandDeltaPct,
                idempotencyKey: SupplierIdempotencyKeys.planningScenario(
                    scopeId: scope,
                    factoryDowntimeHours: Int(downtimeHours),
                    demandDeltaPct: demandDeltaPct
                )
            )
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func checkSparsity() async {
        let id = retailerId.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !id.isEmpty else { return }
        sparsity = try? await SupplierOperationsService.planningSparsity(retailerId: id)
    }

    @MainActor
    private func runPush() async {
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningPredictivePush(
                idempotencyKey: SupplierIdempotencyKeys.planningPredictivePush(scopeId: scope)
            )
            pushStatus = "preview \(resp.source) \(resp.transfers) transfers"
        } catch {
            let text = error.localizedDescription
            if text.contains("factory_planning_disabled") || text.contains("HTTP 409") {
                pushStatus = "factory_planning_disabled"
            } else {
                pushStatus = text
            }
        }
    }
}

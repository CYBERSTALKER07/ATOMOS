import SwiftUI

struct PlanningBrainView: View {
    @State private var sandop: PlanningSAndOPSnapshot?
    @State private var scenario: PlanningScenarioResult?
    @State private var downtimeHours = 8.0
    @State private var demandDeltaPct = 10.0
    @State private var loading = true
    @State private var running = false
    @State private var error: String?

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
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
                    Text("Scenario run")
                        .font(.subheadline.bold())
                    Stepper("Downtime \(Int(downtimeHours))h", value: $downtimeHours, in: 0...168, step: 1)
                    Stepper("Demand \(Int(demandDeltaPct))%", value: $demandDeltaPct, in: -50...200, step: 5)
                    Button(running ? "Running…" : "Run scenario") {
                        Task { await runScenario() }
                    }
                    .disabled(running)
                    if let scenario {
                        Text("SLA risk \(Int(scenario.slaRiskPct))% · fleet \(scenario.fleetVolumeOrders) · stockouts \(scenario.stockoutSkus.count)")
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.secondaryLabel)
                    }
                }
                .padding()
                .background(SupplierTheme.secondaryBackground)
                .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD))
            }
            .supplierReadableWidth()
            .padding()
        }
        .background(SupplierTheme.background)
        .navigationTitle("Planning sandbox")
        .task { await load() }
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
}

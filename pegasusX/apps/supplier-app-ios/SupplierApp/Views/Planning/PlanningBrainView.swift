import SwiftUI

struct PlanningBrainView: View {
    @State private var viewModel = PlanningBrainViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        ScrollView {
            VStack(alignment: .leading, spacing: SupplierTheme.spacingXL) {
                Picker("mobile_supplier.chrome.plan_and_brain", selection: $vm.tab) {
                    Text("mobile_supplier.chrome.planning").tag(PlanBrainTab.planning)
                    Text("mobile_supplier.chrome.digital_brain").tag(PlanBrainTab.brain)
                }
                .pickerStyle(.segmented)
                .accessibilityIdentifier("gs-u-plan-brain-tabs")

                if viewModel.tab == .planning {
                    planningTab(vm: vm)
                } else {
                    brainTab(vm: vm)
                }
            }
            .supplierReadableWidth()
            .padding()
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("portal.nav.planning")
        .task { await viewModel.load() }
    }

    @ViewBuilder
    private func planningTab(@Bindable vm: PlanningBrainViewModel) -> some View {
        SupplierSectionHeader(
            title: "mobile_supplier.chrome.sandop_snapshot",
            subtitle: "mobile_supplier.chrome.readonly_what_if"
        )

        if let error = viewModel.error {
            Text(error)
                .font(.caption)
                .foregroundStyle(SupplierTheme.destructive)
        }

        if viewModel.loading && viewModel.sandop == nil {
            ProgressView("mobile_supplier.loading.sandop")
        } else if let sandop = viewModel.sandop {
            LazyVGrid(
                columns: [GridItem(.adaptive(minimum: 140), spacing: SupplierTheme.spacingMD)],
                spacing: SupplierTheme.spacingMD
            ) {
                KpiTile(title: "mobile_supplier.chrome.factory_cap_7d", value: "\(sandop.factoryCapacityUnits)", systemImage: "building.2", tint: .accentColor)
                KpiTile(title: "mobile_supplier.chrome.wh_inbound", value: "\(sandop.warehouseInboundCapUnits)", systemImage: "arrow.down.to.line", tint: SupplierTheme.secondaryLabel)
                KpiTile(title: "mobile_supplier.chrome.utilization", value: "\(Int(sandop.utilizationPct))%", systemImage: "gauge", tint: SupplierTheme.warning)
                KpiTile(
                    title: "mobile_supplier.chrome.capacity",
                    value: sandop.capacityAlert ? L10n.format("mobile_supplier.chrome.breach") : L10n.format("mobile_supplier.chrome.ok"),
                    systemImage: sandop.capacityAlert ? "exclamationmark.triangle" : "checkmark.circle",
                    tint: sandop.capacityAlert ? SupplierTheme.destructive : SupplierTheme.success
                )
            }
        }

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("supplier_portal.planning_brain_panel.text.scenario_run")
                .font(.subheadline.bold())
            Stepper(LocalizedStringKey(L10n.format("mobile_supplier.chrome.downtime_hours", "\(Int(viewModel.downtimeHours))")), value: $vm.downtimeHours, in: 0...168, step: 1)
            Stepper(LocalizedStringKey(L10n.format("mobile_supplier.chrome.demand_delta", "\(Int(viewModel.demandDeltaPct))")), value: $vm.demandDeltaPct, in: -50...200, step: 5)
            Button(LocalizedStringKey(viewModel.running ? "mobile_supplier.chrome.running" : "mobile_supplier.chrome.run_scenario")) {
                Task { await viewModel.runScenario() }
            }
            .frame(minHeight: 44)
            .disabled(viewModel.running)
            if let scenario = viewModel.scenario {
                Text(L10n.format("mobile_supplier.ui.sla_risk_slariskpct_fleet_fleetvolumeorders_stockouts_count", "\(Int(scenario.slaRiskPct))", "\(scenario.fleetVolumeOrders)", "\(scenario.stockoutSkus.count)"))
                    .font(.caption)
                    .foregroundStyle(SupplierTheme.secondaryLabel)
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD, style: .continuous))
    }

    @ViewBuilder
    private func brainTab(@Bindable vm: PlanningBrainViewModel) -> some View {
        if let demandConfidence = viewModel.demandConfidence {
            ForecastConfidenceView(
                confidence: demandConfidence,
                updatedAt: ForecastConfidenceSupport.formatForecastUpdatedAt(generatedAt: viewModel.demandGeneratedAt),
                stale: ForecastConfidenceSupport.isForecastStale(generatedAt: viewModel.demandGeneratedAt)
            )
            if let demandError = viewModel.demandError {
                Text(demandError).font(.caption).foregroundStyle(SupplierTheme.destructive)
            }
        } else if let demandError = viewModel.demandError {
            Text(demandError).font(.caption).foregroundStyle(SupplierTheme.destructive)
        }
        if brainForecastLine(confidence: viewModel.demandConfidence, accuracyPoints: []) == nil {
            Text("mobile_supplier.chrome.no_forecast_line")
                .font(.caption)
                .foregroundStyle(.secondary)
                .accessibilityIdentifier("gs-u-brain-no-forecast-line")
        }

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("mobile_supplier.chrome.sparsity")
                .font(.subheadline.bold())
            TextField("mobile_supplier.chrome.retailer_id_field", text: $vm.retailerId)
                .textInputAutocapitalization(.never)
                .frame(minHeight: 44)
            Button("mobile_supplier.chrome.check") { Task { await viewModel.checkSparsity() } }
                .frame(minHeight: 44)
            if let sparsity = viewModel.sparsity {
                Text(sparsity.allowed ? L10n.format("mobile_supplier.chrome.sparsity_allowed", sparsity.label) : L10n.format("mobile_supplier.chrome.sparsity_blocked", sparsity.blockedReason ?? sparsity.label))
                    .font(.caption)
                    .accessibilityIdentifier("gs-u-sparsity-result")
            }
            if let sparsityError = viewModel.sparsityError {
                Text(sparsityError).font(.caption).foregroundStyle(SupplierTheme.destructive)
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD, style: .continuous))

        VStack(alignment: .leading, spacing: SupplierTheme.spacingSM) {
            Text("mobile_supplier.chrome.predictive_push")
                .font(.subheadline.bold())
            Button("mobile_supplier.chrome.preview_push") { Task { await viewModel.runPush() } }
                .frame(minHeight: 44)
            if let pushStatus = viewModel.pushStatus {
                Text(pushStatus)
                    .font(.caption)
                    .accessibilityIdentifier("gs-u-planning-push-status")
            }
        }
        .padding()
        .background(SupplierTheme.secondaryBackground)
        .clipShape(RoundedRectangle(cornerRadius: SupplierTheme.radiusMD, style: .continuous))
    }
}

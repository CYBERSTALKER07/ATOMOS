import SwiftUI

struct PlanningSettingsView: View {
    @State private var viewModel = PlanningSettingsViewModel()

    var body: some View {
        @Bindable var vm = viewModel
        Form {
            if viewModel.loading && viewModel.data == nil {
                Section {
                    ProgressView("mobile_supplier.loading.seasonal")
                }
            } else if let error = viewModel.error, viewModel.data == nil {
                Section {
                    SupplierErrorView(message: error) { Task { await viewModel.load() } }
                }
            } else {
                Section("mobile_supplier.chrome.factory_network_ops") {
                    Text("mobile_supplier.chrome.factory_ops_explainer")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    if let modeError = viewModel.modeError, !viewModel.modeKnown {
                        Text(modeError)
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.destructive)
                    } else if viewModel.modeKnown, !viewModel.planningEnabled {
                        Text("mobile_supplier.chrome.engines_off")
                            .font(.caption)
                    }
                    if let modeError = viewModel.modeError, viewModel.modeKnown {
                        Text(modeError)
                            .font(.caption)
                            .foregroundStyle(SupplierTheme.destructive)
                    }
                    ForEach(["SPEED", "ECONOMY", "BALANCED", "LOW_CARBON", "MANUAL_ONLY"], id: \.self) { mode in
                        Button(viewModel.networkMode == mode ? "● \(mode)" : mode) {
                            Task { await viewModel.putMode(mode) }
                        }
                        .frame(minHeight: 44)
                        .disabled(viewModel.opsBusy)
                    }
                    Button("mobile_supplier.chrome.run_pull_matrix") { Task { await viewModel.runPullMatrix() } }
                        .frame(minHeight: 44)
                        .disabled(viewModel.opsBusy)
                    Button("mobile_supplier.chrome.predictive_push") { Task { await viewModel.runPredictivePush() } }
                        .frame(minHeight: 44)
                        .disabled(viewModel.opsBusy)
                    TextField("mobile_supplier.chrome.kill_switch_reason", text: $vm.killReason)
                    Button("mobile_supplier.chrome.kill_switch") { Task { await viewModel.runKillSwitch() } }
                        .frame(minHeight: 44)
                        .disabled(viewModel.opsBusy)
                    if let opsStatus = viewModel.opsStatus {
                        Text(opsStatus).font(.footnote)
                    }
                }
                Section("mobile_supplier.chrome.create_override") {
                    CreateOverrideForm(
                        templates: viewModel.data?.builtinTemplates ?? [],
                        templateId: $vm.templateId,
                        name: $vm.name,
                        startDate: $vm.startDate,
                        endDate: $vm.endDate,
                        multiplier: $vm.multiplier,
                        formError: viewModel.formError,
                        saving: viewModel.saving,
                        onSubmit: {
                            Task { await viewModel.createOverride() }
                        }
                    )
                }
                Section("mobile_supplier.chrome.active_overrides") {
                    SeasonalOverridesList(overrides: viewModel.data?.overrides ?? [])
                }
            }
        }
        .scrollDismissesKeyboard(.interactively)
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.settings.planning.text.planning_settings")
        .task { await viewModel.load() }
        .refreshable { await viewModel.load(silent: true) }
    }

    static func killSwitchReasonValidation(reason: String) -> String? {
        let trimmed = reason.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.isEmpty {
            return L10n.format("mobile_supplier.chrome.typed_reason_required")
        }
        return nil
    }
}

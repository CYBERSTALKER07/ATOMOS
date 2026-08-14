import SwiftUI

struct PlanningSettingsView: View {
    @State private var data: SeasonalTemplatesResponse?
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var formError: String?
    @State private var templateId = ""
    @State private var name = ""
    @State private var startDate = ""
    @State private var endDate = ""
    @State private var multiplier = ""
    @State private var networkMode = ""
    @State private var planningEnabled = false
    @State private var opsStatus: String? = nil
    @State private var killReason = ""
    @State private var opsBusy = false

    var body: some View {
        Form {
            if loading {
                Section {
                    ProgressView("Loading seasonal overrides…")
                }
            } else if let error {
                Section {
                    SupplierErrorView(message: error) { Task { await load() } }
                }
            } else {
                Section("Factory network ops") {
                    Text("Mode, pull-matrix, kill-switch. Pull-matrix 409 if FACTORY_PLANNING_ENABLED is off.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                    if !planningEnabled {
                        Text("Engines off (env flag).")
                            .font(.caption)
                    }
                    ForEach(["SPEED", "ECONOMY", "BALANCED", "LOW_CARBON", "MANUAL_ONLY"], id: \.self) { mode in
                        Button(networkMode == mode ? "● \(mode)" : mode) {
                            Task { await putMode(mode) }
                        }
                        .disabled(opsBusy)
                    }
                    Button("Run pull-matrix") { Task { await runPullMatrix() } }
                        .disabled(opsBusy)
                    TextField("Kill-switch reason (ADMIN)", text: $killReason)
                    Button("Kill-switch") { Task { await runKillSwitch() } }
                        .disabled(opsBusy)
                    if let opsStatus {
                        Text(opsStatus).font(.footnote)
                    }
                }
                Section("Create override") {
                    CreateOverrideForm(
                        templates: data?.builtinTemplates ?? [],
                        templateId: $templateId,
                        name: $name,
                        startDate: $startDate,
                        endDate: $endDate,
                        multiplier: $multiplier,
                        formError: formError,
                        saving: saving,
                        onSubmit: {
                            Task { await createOverride() }
                        }
                    )
                }
                Section("Active overrides") {
                    SeasonalOverridesList(overrides: data?.overrides ?? [])
                }
            }
        }
        .background(SupplierTheme.background)
        .navigationTitle("supplier_portal.settings.planning.text.planning_settings")
        .task { await load() }
        .refreshable { await load(silent: true) }
    }

    @MainActor
    private func load(silent: Bool = false) async {
        if !silent { loading = true }
        error = nil
        defer { loading = false }
        do {
            data = try await SupplierOperationsService.seasonalOverrides()
            if let nm = try? await SupplierOperationsService.networkMode() {
                networkMode = nm.mode
                planningEnabled = nm.planningEnabled
            }
        } catch {
            if !silent { self.error = error.localizedDescription }
        }
    }

    @MainActor
    private func createOverride() async {
        guard !startDate.isEmpty, !endDate.isEmpty else {
            formError = "Start and end dates are required"
            return
        }
        saving = true
        formError = nil
        defer { saving = false }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            let multVal = Double(multiplier.trimmingCharacters(in: .whitespacesAndNewlines))
            let row = try await SupplierOperationsService.createSeasonalOverride(
                SeasonalOverrideInput(
                    templateId: templateId.isEmpty ? nil : templateId,
                    startDate: startDate,
                    endDate: endDate,
                    name: name.isEmpty ? nil : name,
                    multiplier: multVal
                ),
                idempotencyKey: SupplierIdempotencyKeys.seasonalOverrideCreate(
                    scopeId: scope,
                    startDate: startDate,
                    endDate: endDate
                )
            )
            var overrides = data?.overrides ?? []
            overrides.insert(row, at: 0)
            data = SeasonalTemplatesResponse(
                builtinTemplates: data?.builtinTemplates ?? [],
                overrides: overrides
            )
            name = ""
            startDate = ""
            endDate = ""
            templateId = ""
            multiplier = ""
        } catch {
            formError = error.localizedDescription
        }
    }

    @MainActor
    private func putMode(_ mode: String) async {
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.putNetworkMode(
                NetworkModeUpdateRequest(mode: mode, reason: nil),
                idempotencyKey: SupplierIdempotencyKeys.networkModePut(scopeId: scope, mode: mode)
            )
            networkMode = resp.newMode
            opsStatus = "Mode \(resp.oldMode) → \(resp.newMode)"
        } catch {
            opsStatus = error.localizedDescription
        }
    }

    @MainActor
    private func runPullMatrix() async {
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningPullMatrix(
                idempotencyKey: SupplierIdempotencyKeys.planningPullMatrix(scopeId: scope)
            )
            opsStatus = "Pull-matrix \(resp.status): \(resp.transfers) transfers"
        } catch {
            let text = error.localizedDescription
            if text.contains("factory_planning_disabled") || text.contains("HTTP 409") {
                opsStatus = "factory_planning_disabled — engines off until FACTORY_PLANNING_ENABLED is on"
            } else {
                opsStatus = text
            }
        }
    }

    @MainActor
    private func runKillSwitch() async {
        let reason = killReason.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !reason.isEmpty else {
            opsStatus = "Typed reason required"
            return
        }
        opsBusy = true
        defer { opsBusy = false }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            let resp = try await SupplierOperationsService.planningKillSwitch(
                KillSwitchRequest(reason: reason),
                idempotencyKey: SupplierIdempotencyKeys.planningKillSwitch(scopeId: scope, reason: reason)
            )
            opsStatus = "Kill-switch cancelled \(resp.cancelledTransfers), mode \(resp.mode)"
            killReason = ""
        } catch {
            opsStatus = error.localizedDescription
        }
    }
}

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
}

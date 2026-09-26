import SwiftUI

private let hourPresets: [Int64] = [8, 24, 48, 72]

struct ReturnPolicySettingsView: View {
    @State private var loading = true
    @State private var saving = false
    @State private var error: String?
    @State private var saved = false
    @State private var hours: Int64 = 48
    @State private var hoursText = "48"
    @State private var concealed = ""
    @State private var requirePhoto = true
    @State private var allowExpired = false
    @State private var sourceHint = ""

    var body: some View {
        Group {
            if loading {
                SupplierLoadingView(title: "Loading return policy…")
            } else {
                Form {
                    Section {
                        Text("supplier_portal.residual.text.claim_filing_windows_applied_when_orders_complete_retailers_see_")
                            .font(.footnote)
                            .foregroundStyle(.secondary)
                        if !sourceHint.isEmpty {
                            Text(L10n.format("mobile_supplier.ui.source_hint_sourcehint_2", "\(sourceHint)"))
                                .font(.caption)
                                .foregroundStyle(.secondary)
                        }
                    }

                    if let error {
                        Section {
                            Text(error).foregroundStyle(.red)
                        }
                    }
                    if saved {
                        Section {
                            Text("supplier_portal.settings.return_policy.text.return_policy_saved").foregroundStyle(SupplierTheme.success)
                        }
                    }

                    Section("Default claim window (hours)") {
                        HStack {
                            ForEach(hourPresets, id: \.self) { preset in
                                Button(L10n.format("mobile_supplier.ui.preseth", "\(preset)")) {
                                    hours = preset
                                    hoursText = String(preset)
                                }
                                .buttonStyle(.bordered)
                                .tint(hours == preset ? Color.accentColor : .secondary)
                            }
                        }
                        TextField("mobile_supplier.ui.custom_1_168", text: $hoursText)
                            .keyboardType(.numberPad)
                            .onChange(of: hoursText) { _, newValue in
                                let digits = newValue.filter(\.isNumber)
                                if digits != newValue { hoursText = digits }
                                if let parsed = Int64(digits) { hours = parsed }
                            }
                        Text(L10n.format("mobile_supplier.ui.preview_retailers_may_file_claims_for_isempty_hourstexth_after_delivery", "\(hoursText.isEmpty ? "—" : hoursText)"))
                            .font(.caption)
                            .foregroundStyle(.secondary)
                    }

                    Section("Concealed damage (optional)") {
                        TextField("mobile_supplier.ui.hours_same_as_default_if_empty", text: $concealed)
                            .keyboardType(.numberPad)
                            .onChange(of: concealed) { _, newValue in
                                let digits = newValue.filter(\.isNumber)
                                if digits != newValue { concealed = digits }
                            }
                    }

                    Section("Claims rules") {
                        Toggle("Require photo evidence on claims", isOn: $requirePhoto)
                        Toggle("Allow filing after window expires", isOn: $allowExpired)
                    }

                    Section {
                        Button(saving ? "Saving…" : "Save return policy") { Task { await save() } }
                            .disabled(saving)
                    }
                }
            }
        }
        .navigationTitle("portal.nav.return_policy")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("portal.page.orders.action.refresh", systemImage: "arrow.clockwise") { Task { await load() } }
            }
        }
        .task { await load() }
    }

    @MainActor
    private func load() async {
        loading = true
        error = nil
        defer { loading = false }
        do {
            let p = try await SupplierOperationsService.returnPolicy()
            hours = p.defaultWindowHours > 0 ? p.defaultWindowHours : 48
            hoursText = String(hours)
            if let c = p.concealedDamageWindowHours, c > 0 {
                concealed = String(c)
            } else {
                concealed = ""
            }
            requirePhoto = p.requirePhoto
            allowExpired = p.allowExpiredClaims
            sourceHint = p.policySourceHint ?? ""
        } catch {
            self.error = error.localizedDescription
        }
    }

    @MainActor
    private func save() async {
        let parsed = Int64(hoursText) ?? hours
        guard parsed >= 1, parsed <= 168 else {
            error = "Default window must be between 1 and 168 hours"
            return
        }
        saving = true
        error = nil
        saved = false
        defer { saving = false }
        var body = SupplierReturnPolicy(
            defaultWindowHours: parsed,
            concealedDamageWindowHours: nil,
            requirePhoto: requirePhoto,
            allowExpiredClaims: allowExpired,
            policySourceHint: nil
        )
        if let c = Int64(concealed.trimmingCharacters(in: .whitespaces)), c > 0 {
            body.concealedDamageWindowHours = c
        }
        do {
            let scope = await SupplierIdempotencyKeys.supplierScopeId()
            let key = SupplierIdempotencyKeys.returnPolicyPut(scopeId: scope, hours: parsed)
            let savedPolicy = try await SupplierOperationsService.putReturnPolicy(body, idempotencyKey: key)
            sourceHint = savedPolicy.policySourceHint ?? "SUPPLIER"
            hours = parsed
            hoursText = String(parsed)
            saved = true
        } catch {
            self.error = error.localizedDescription
        }
    }
}
